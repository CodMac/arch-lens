package java

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	"github.com/CodMac/go-treesitter-dependency-analyzer/parser"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Extractor struct{}

func NewJavaExtractor() *Extractor {
	return &Extractor{}
}

const JavaActionQuery = `
   [
      (method_invocation 
          name: (identifier) @call_target) @call_stmt
      
      (method_reference 
          (identifier) @ref_target) @ref_stmt
      
      (object_creation_expression 
          type: [
              (type_identifier) @create_target_name
              (generic_type (type_identifier) @create_target_name)
          ]) @create_stmt
      
      (field_access 
          field: (identifier) @use_field_name) @use_field_stmt
      
      (cast_expression 
          type: (type_identifier) @cast_type) @cast_stmt
   ]
`

func (e *Extractor) Extract(filePath string, gCtx *model.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gCtx.FileContexts[filePath]
	if !ok {
		return nil, fmt.Errorf("failed to get FileContext: %s", filePath)
	}

	tsLang, err := parser.GetLanguage(model.LangJava)
	if err != nil {
		return nil, err
	}

	relations := make([]*model.DependencyRelation, 0)
	relations = append(relations, e.extractFileBaseRelations(fCtx, gCtx)...)
	relations = append(relations, e.extractStructuralRelations(fCtx, gCtx)...)

	actionRels, err := e.processActionQuery(fCtx, gCtx, tsLang)
	if err != nil {
		return nil, err
	}
	relations = append(relations, actionRels...)

	return relations, nil
}

func (e *Extractor) extractFileBaseRelations(fCtx *model.FileContext, gCtx *model.GlobalContext) []*model.DependencyRelation {
	rels := make([]*model.DependencyRelation, 0)
	fileElem := &model.CodeElement{
		Kind: model.File, Name: filepath.Base(fCtx.FilePath), QualifiedName: fCtx.FilePath, Path: fCtx.FilePath,
	}
	if fCtx.PackageName != "" {
		pkgElem := &model.CodeElement{Kind: model.Package, Name: fCtx.PackageName, QualifiedName: fCtx.PackageName}
		rels = append(rels, &model.DependencyRelation{Type: model.Contain, Source: pkgElem, Target: fileElem})
	}
	for _, imp := range fCtx.Imports {
		cleanPath := strings.TrimSuffix(imp.RawImportPath, ".*")
		rels = append(rels, &model.DependencyRelation{
			Type: model.Import, Source: fileElem, Target: e.resolveTargetElement(cleanPath, imp.Kind, fCtx, gCtx),
		})
	}
	return rels
}

func (e *Extractor) extractStructuralRelations(fCtx *model.FileContext, gCtx *model.GlobalContext) []*model.DependencyRelation {
	rels := make([]*model.DependencyRelation, 0)
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			elem := entry.Element
			if elem.Extra == nil {
				continue
			}
			for _, anno := range elem.Extra.Annotations {
				rels = append(rels, &model.DependencyRelation{
					Type: model.Annotation, Source: elem, Target: e.resolveTargetElement(e.stripGenericsAndAt(anno), model.KAnnotation, fCtx, gCtx),
				})
			}
			if entry.ParentQN != "" && entry.ParentQN != fCtx.PackageName {
				if parents, ok := gCtx.DefinitionsByQN[entry.ParentQN]; ok {
					rels = append(rels, &model.DependencyRelation{Type: model.Contain, Source: parents[0].Element, Target: elem})
				}
			}
			e.collectExtraRelations(elem, fCtx, gCtx, &rels)
		}
	}
	return rels
}

func (e *Extractor) processActionQuery(fCtx *model.FileContext, gCtx *model.GlobalContext, tsLang *sitter.Language) ([]*model.DependencyRelation, error) {
	rels := make([]*model.DependencyRelation, 0)
	q, err := sitter.NewQuery(tsLang, JavaActionQuery)
	if err != nil {
		return nil, err
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, fCtx.RootNode, *fCtx.SourceBytes)

	for {
		match := matches.Next()
		if match == nil {
			break
		}
		sourceNode := &match.Captures[0].Node
		sourceElem := e.determineSourceElement(sourceNode, fCtx, gCtx)
		if sourceElem == nil {
			continue
		}

		for _, cap := range match.Captures {
			capName := q.CaptureNames()[cap.Index]
			node := cap.Node
			rawName := node.Utf8Text(*fCtx.SourceBytes)
			var targetElem *model.CodeElement
			var relType model.DependencyType = model.Use

			switch capName {
			case "call_target":
				relType = model.Call
				prefix := e.getObjectPrefix(&node, "method_invocation", fCtx)
				targetElem = e.resolveWithPrefix(rawName, prefix, model.Method, sourceElem, fCtx, gCtx)
			case "ref_target":
				relType = model.Call
				prefix := e.getObjectPrefix(&node, "method_reference", fCtx)
				targetElem = e.resolveWithPrefix(rawName, prefix, model.Method, sourceElem, fCtx, gCtx)
			case "create_target_name":
				relType = model.Create
				targetElem = e.resolveTargetElement(e.stripGenericsAndAt(rawName), model.Class, fCtx, gCtx)
			case "use_field_name":
				relType = model.Use
				targetElem = e.resolveTargetElement(e.stripGenericsAndAt(rawName), model.Field, fCtx, gCtx)
			case "cast_type":
				relType = model.Cast
				targetElem = e.resolveTargetElement(e.stripGenericsAndAt(rawName), model.Type, fCtx, gCtx)
			}
			if targetElem != nil {
				rels = append(rels, &model.DependencyRelation{
					Type: relType, Source: sourceElem, Target: targetElem, Location: e.nodeToLocation(&node, fCtx.FilePath),
				})
			}
		}
	}
	return rels, nil
}

func (e *Extractor) getObjectPrefix(node *sitter.Node, parentKind string, fCtx *model.FileContext) string {
	parent := node.Parent()
	for parent != nil && parent.Kind() != parentKind {
		parent = parent.Parent()
	}
	if parent == nil {
		return ""
	}

	if parentKind == "method_invocation" {
		if obj := parent.ChildByFieldName("object"); obj != nil {
			return obj.Utf8Text(*fCtx.SourceBytes)
		}
	}
	if parentKind == "method_reference" {
		var prefixParts []string
		for i := 0; i < int(parent.ChildCount()); i++ {
			child := parent.Child(uint(i))
			if child.Kind() == "::" {
				break
			}
			prefixParts = append(prefixParts, child.Utf8Text(*fCtx.SourceBytes))
		}
		return strings.Join(prefixParts, "")
	}
	return ""
}

// resolveWithPrefix 增强：引入 sourceElem 处理 this 关键字
func (e *Extractor) resolveWithPrefix(name, prefix string, kind model.ElementKind, sourceElem *model.CodeElement, fCtx *model.FileContext, gCtx *model.GlobalContext) *model.CodeElement {
	if prefix == "" {
		return e.resolveTargetElement(name, kind, fCtx, gCtx)
	}

	// 处理 this 关键字
	if prefix == "this" && sourceElem != nil {
		// 寻找当前所属的 Class (QualifiedName 中剥离方法名部分)
		parentQN := sourceElem.QualifiedName
		if idx := strings.LastIndex(parentQN, "."); idx != -1 {
			// 如果 sourceElem 是方法，它的父级应该是类
			parentQN = parentQN[:idx]
		}
		return e.resolveTargetElement(parentQN+"."+name, kind, fCtx, gCtx)
	}

	resolvedPrefix := e.resolveTargetElement(e.stripGenericsAndAt(prefix), model.Variable, fCtx, gCtx)
	return e.resolveTargetElement(resolvedPrefix.QualifiedName+"."+name, kind, fCtx, gCtx)
}

func (e *Extractor) resolveTargetElement(cleanName string, defaultKind model.ElementKind, fCtx *model.FileContext, gCtx *model.GlobalContext) *model.CodeElement {
	entries := gCtx.ResolveSymbol(fCtx, cleanName)
	if len(entries) > 0 {
		found := entries[0].Element
		return &model.CodeElement{Kind: found.Kind, Name: found.Name, QualifiedName: found.QualifiedName, Path: found.Path}
	}
	if info, ok := JavaBuiltinTable[cleanName]; ok {
		return &model.CodeElement{Kind: info.Kind, Name: cleanName, QualifiedName: info.QN}
	}
	if strings.Contains(cleanName, ".") {
		parts := strings.Split(cleanName, ".")
		prefixResolved := e.resolveTargetElement(parts[0], model.Unknown, fCtx, gCtx)
		// 如果前缀被成功解析（比如 System 解析成 java.lang.System）
		if prefixResolved.QualifiedName != parts[0] {
			return &model.CodeElement{Kind: defaultKind, Name: parts[len(parts)-1], QualifiedName: prefixResolved.QualifiedName + "." + strings.Join(parts[1:], ".")}
		}
	}
	if len(cleanName) > 0 && cleanName[0] >= 'A' && cleanName[0] <= 'Z' {
		if defaultKind == model.Class || defaultKind == model.Type || defaultKind == model.KAnnotation {
			return &model.CodeElement{Kind: defaultKind, Name: cleanName, QualifiedName: "java.lang." + cleanName}
		}
	}
	return &model.CodeElement{Kind: defaultKind, Name: cleanName, QualifiedName: cleanName}
}

func (e *Extractor) stripGenericsAndAt(name string) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), "@")
	if idx := strings.Index(name, "<"); idx != -1 {
		return strings.TrimSpace(name[:idx])
	}
	return name
}

func (e *Extractor) determineSourceElement(n *sitter.Node, fCtx *model.FileContext, gCtx *model.GlobalContext) *model.CodeElement {
	curr := n.Parent()
	for curr != nil {
		if strings.Contains(curr.Kind(), "declaration") {
			nameNode := curr.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Utf8Text(*fCtx.SourceBytes)
				entries := gCtx.ResolveSymbol(fCtx, name)
				for _, entry := range entries {
					// 增加行号匹配，确保在内部类/重载场景下的准确性
					if int(curr.StartPosition().Row)+1 == entry.Element.Location.StartLine {
						return entry.Element
					}
				}
			}
		}
		curr = curr.Parent()
	}
	return nil
}

func (e *Extractor) nodeToLocation(n *sitter.Node, fp string) *model.Location {
	if n == nil {
		return nil
	}
	return &model.Location{
		FilePath: fp, StartLine: int(n.StartPosition().Row) + 1, EndLine: int(n.EndPosition().Row) + 1,
		StartColumn: int(n.StartPosition().Column), EndColumn: int(n.EndPosition().Column),
	}
}

func (e *Extractor) collectExtraRelations(elem *model.CodeElement, fCtx *model.FileContext, gCtx *model.GlobalContext, rels *[]*model.DependencyRelation) {
	if elem.Extra == nil {
		return
	}

	// Class 关系
	if elem.Extra.ClassExtra != nil {
		ce := elem.Extra.ClassExtra
		if ce.SuperClass != "" {
			*rels = append(*rels, &model.DependencyRelation{Type: model.Extend, Source: elem, Target: e.resolveTargetElement(e.stripGenericsAndAt(ce.SuperClass), model.Class, fCtx, gCtx)})
		}
		for _, imp := range ce.ImplementedInterfaces {
			*rels = append(*rels, &model.DependencyRelation{Type: model.Implement, Source: elem, Target: e.resolveTargetElement(e.stripGenericsAndAt(imp), model.Interface, fCtx, gCtx)})
		}
	}

	// Method 关系
	if elem.Extra.MethodExtra != nil {
		me := elem.Extra.MethodExtra
		// 1. RETURN
		if me.ReturnType != "" && me.ReturnType != "void" {
			*rels = append(*rels, &model.DependencyRelation{Type: model.Return, Source: elem, Target: e.resolveTargetElement(e.stripGenericsAndAt(me.ReturnType), model.Type, fCtx, gCtx)})
		}
		// 2. PARAMETER
		for _, pInfo := range me.Parameters {
			parts := strings.Fields(pInfo)
			if len(parts) >= 1 {
				*rels = append(*rels, &model.DependencyRelation{Type: model.Parameter, Source: elem, Target: e.resolveTargetElement(e.stripGenericsAndAt(parts[0]), model.Type, fCtx, gCtx)})
			}
		}
		// 3. THROW (修复补全)
		for _, tType := range me.ThrowsTypes {
			*rels = append(*rels, &model.DependencyRelation{
				Type: model.Throw, Source: elem,
				Target: e.resolveTargetElement(e.stripGenericsAndAt(tType), model.Class, fCtx, gCtx),
			})
		}
	}
}

// --- Java 内置符号表 ---

var JavaBuiltinTable = map[string]struct {
	QN   string
	Kind model.ElementKind
}{
	"String":                   {"java.lang.String", model.Class},
	"Object":                   {"java.lang.Object", model.Class},
	"System":                   {"java.lang.System", model.Class},
	"Integer":                  {"java.lang.Integer", model.Class},
	"Long":                     {"java.lang.Long", model.Class},
	"Double":                   {"java.lang.Double", model.Class},
	"Boolean":                  {"java.lang.Boolean", model.Class},
	"Exception":                {"java.lang.Exception", model.Class},
	"RuntimeException":         {"java.lang.RuntimeException", model.Class},
	"IllegalArgumentException": {"java.lang.IllegalArgumentException", model.Class},
	"Thread":                   {"java.lang.Thread", model.Class},
	"Runnable":                 {"java.lang.Runnable", model.Interface},
	"Class":                    {"java.lang.Class", model.Class},
	"Iterable":                 {"java.lang.Iterable", model.Interface},
	"Override":                 {"java.lang.Override", model.KAnnotation},
	"Deprecated":               {"java.lang.Deprecated", model.KAnnotation},
	"SuppressWarnings":         {"java.lang.SuppressWarnings", model.KAnnotation},
	"List":                     {"java.util.List", model.Interface},
	"ArrayList":                {"java.util.ArrayList", model.Class},
	"Map":                      {"java.util.Map", model.Interface},
	"HashMap":                  {"java.util.HashMap", model.Class},
	"Optional":                 {"java.util.Optional", model.Class},
	"UUID":                     {"java.util.UUID", model.Class},
	"Set":                      {"java.util.Set", model.Interface},
	"HashSet":                  {"java.util.HashSet", model.Class},
	"out":                      {"java.lang.System.out", model.Field},
	"err":                      {"java.lang.System.err", model.Field},
	"in":                       {"java.lang.System.in", model.Field},
}
