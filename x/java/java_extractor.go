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
      (method_invocation name: (identifier) @call_target) @call_stmt
      (method_reference (identifier) @ref_target) @ref_stmt
      (explicit_constructor_invocation 
          constructor: [ (super) @super_target (this) @this_target ]) @explicit_call_stmt
      (object_creation_expression 
          type: [(type_identifier) @create_target_name (generic_type (type_identifier) @create_target_name)]) @create_stmt
      (field_access field: (identifier) @use_field_name) @use_field_stmt
      (cast_expression type: (type_identifier) @cast_type) @cast_stmt
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

	return append(relations, actionRels...), nil
}

// --- 基础关系：Package, Imports ---

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
			Type:   model.Import,
			Source: fileElem,
			Target: e.resolveTargetElement(cleanPath, imp.Kind, fCtx, gCtx),
		})
	}

	return rels
}

// --- 结构化关系：Inheritance, Contains, Annotations, Method Metadata ---

func (e *Extractor) extractStructuralRelations(fCtx *model.FileContext, gCtx *model.GlobalContext) []*model.DependencyRelation {
	rels := make([]*model.DependencyRelation, 0)
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			elem := entry.Element
			if elem.Extra == nil {
				continue
			}

			// 1. Annotations
			for _, anno := range elem.Extra.Annotations {
				cleanAnno := e.stripAnnotationArgs(anno)
				rels = append(rels, &model.DependencyRelation{
					Type:   model.Annotation,
					Source: elem,
					Target: e.resolveTargetElement(cleanAnno, model.KAnnotation, fCtx, gCtx),
				})
			}

			// 2. Parental Containment
			if entry.ParentQN != "" && entry.ParentQN != fCtx.PackageName {
				if parents, ok := gCtx.DefinitionsByQN[entry.ParentQN]; ok {
					rels = append(rels, &model.DependencyRelation{Type: model.Contain, Source: parents[0].Element, Target: elem})
				}
			}

			// 3. Metadata (Extends/Implements/Throws/Params/Returns)
			e.collectExtraRelations(elem, fCtx, gCtx, &rels)
		}
	}
	return rels
}

func (e *Extractor) collectExtraRelations(elem *model.CodeElement, fCtx *model.FileContext, gCtx *model.GlobalContext, rels *[]*model.DependencyRelation) {
	if elem.Extra == nil {
		return
	}

	// Class/Interface Inheritance
	if ce := elem.Extra.ClassExtra; ce != nil {
		if ce.SuperClass != "" {
			tKind := model.Class
			if elem.Kind == model.Interface {
				tKind = model.Interface
			}
			*rels = append(*rels, &model.DependencyRelation{
				Type:   model.Extend,
				Source: elem,
				Target: e.resolveTargetElement(e.cleanTypeName(ce.SuperClass), tKind, fCtx, gCtx),
			})
		}

		for _, imp := range ce.ImplementedInterfaces {
			relType := model.Implement
			if elem.Kind == model.Interface {
				relType = model.Extend
			}
			*rels = append(*rels, &model.DependencyRelation{
				Type:   relType,
				Source: elem,
				Target: e.resolveTargetElement(e.cleanTypeName(imp), model.Interface, fCtx, gCtx),
			})
		}
	}

	// Method Metadata
	if me := elem.Extra.MethodExtra; me != nil {
		if me.ReturnType != "" && me.ReturnType != "void" {
			*rels = append(*rels, &model.DependencyRelation{
				Type:   model.Return,
				Source: elem,
				Target: e.resolveTargetElement(e.cleanTypeName(me.ReturnType), model.Type, fCtx, gCtx),
			})
		}

		for _, pInfo := range me.Parameters {
			if parts := strings.Fields(pInfo); len(parts) >= 1 {
				*rels = append(*rels, &model.DependencyRelation{
					Type:   model.Parameter,
					Source: elem,
					Target: e.resolveTargetElement(e.cleanTypeName(parts[0]), model.Type, fCtx, gCtx),
				})
			}
		}

		for _, tType := range me.ThrowsTypes {
			*rels = append(*rels, &model.DependencyRelation{
				Type:   model.Throw,
				Source: elem,
				Target: e.resolveTargetElement(e.cleanTypeName(tType), model.Class, fCtx, gCtx),
			})
		}
	}
}

// --- Action Query: Method Calls, Field Usage, Creation ---

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
			case "call_target", "ref_target":
				relType = model.Call
				prefix := e.getObjectPrefix(&node, "method_invocation", fCtx)
				targetElem = e.resolveWithPrefix(rawName, prefix, model.Method, sourceElem, fCtx, gCtx)

			case "super_target":
				relType = model.Call
				// 显式调用父类构造函数 super(...)
				targetElem = e.resolveWithPrefix("super", "super", model.Method, sourceElem, fCtx, gCtx)

			case "this_target":
				relType = model.Call
				// 显式调用本类构造函数 this(...)
				targetElem = e.resolveWithPrefix("this", "this", model.Method, sourceElem, fCtx, gCtx)

			case "create_target_name":
				relType = model.Create
				targetElem = e.resolveTargetElement(e.cleanTypeName(rawName), model.Class, fCtx, gCtx)

			case "use_field_name":
				relType = model.Use
				prefix := e.getObjectPrefix(&node, "field_access", fCtx)
				targetElem = e.resolveWithPrefix(rawName, prefix, model.Field, sourceElem, fCtx, gCtx)

			case "cast_type":
				relType = model.Cast
				targetElem = e.resolveTargetElement(e.cleanTypeName(rawName), model.Type, fCtx, gCtx)
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

// --- Symbol Resolution Core ---

func (e *Extractor) resolveTargetElement(cleanName string, defaultKind model.ElementKind, fCtx *model.FileContext, gCtx *model.GlobalContext) *model.CodeElement {
	// 1. Global Symbol Table
	if entries := gCtx.ResolveSymbol(fCtx, cleanName); len(entries) > 0 {
		found := entries[0].Element
		return &model.CodeElement{Kind: found.Kind, Name: found.Name, QualifiedName: found.QualifiedName, Path: found.Path, Extra: found.Extra}
	}

	// 2. Java Built-in Table
	if builtin := e.resolveFromBuiltin(cleanName); builtin != nil {
		return builtin
	}

	// 3. Dot-separated references (e.g. RetentionPolicy.RUNTIME)
	if strings.Contains(cleanName, ".") {
		parts := strings.Split(cleanName, ".")
		lastPart := parts[len(parts)-1]
		if info, ok := JavaBuiltinTable[lastPart]; ok && strings.Contains(info.QN, parts[len(parts)-2]) {
			return &model.CodeElement{Kind: info.Kind, Name: lastPart, QualifiedName: info.QN}
		}

		// Recursive prefix resolution
		prefixResolved := e.resolveTargetElement(parts[0], model.Unknown, fCtx, gCtx)
		if prefixResolved.QualifiedName != parts[0] {
			return &model.CodeElement{
				Kind: defaultKind, Name: lastPart,
				QualifiedName: prefixResolved.QualifiedName + "." + strings.Join(parts[1:], "."),
			}
		}
	}

	// 4. Implicit java.lang
	if len(cleanName) > 0 && cleanName[0] >= 'A' && cleanName[0] <= 'Z' {
		if defaultKind == model.Class || defaultKind == model.Type || defaultKind == model.KAnnotation {
			if builtin := e.resolveFromBuiltin(cleanName); builtin != nil {
				return builtin
			}
		}
	}

	return &model.CodeElement{Kind: defaultKind, Name: cleanName, QualifiedName: cleanName}
}

func (e *Extractor) resolveFromBuiltin(name string) *model.CodeElement {
	if info, ok := JavaBuiltinTable[name]; ok {
		elem := &model.CodeElement{Kind: info.Kind, Name: name, QualifiedName: info.QN}
		if info.Kind == model.Class || info.Kind == model.Interface || info.Kind == model.Enum || info.Kind == model.KAnnotation {
			elem.Extra = &model.ElementExtra{ClassExtra: &model.ClassExtra{IsBuiltin: true}}
		}
		return elem
	}
	return nil
}

func (e *Extractor) resolveWithPrefix(name, prefix string, kind model.ElementKind, sourceElem *model.CodeElement, fCtx *model.FileContext, gCtx *model.GlobalContext) *model.CodeElement {
	if prefix == "" {
		return e.resolveTargetElement(name, kind, fCtx, gCtx)
	}

	// --- 增加对 super 关键字的支持 ---
	if prefix == "super" && sourceElem != nil {
		// 1. 获取当前类 QN
		classQN := sourceElem.QualifiedName
		if idx := strings.LastIndex(classQN, "."); idx != -1 {
			classQN = classQN[:idx]
		}
		// 2. 找到父类名称
		if defs, ok := gCtx.DefinitionsByQN[classQN]; ok && len(defs) > 0 {
			if defs[0].Element.Extra != nil && defs[0].Element.Extra.ClassExtra != nil {
				superName := e.cleanTypeName(defs[0].Element.Extra.ClassExtra.SuperClass)
				// 3. 解析父类
				superElem := e.resolveTargetElement(superName, model.Class, fCtx, gCtx)
				// 4. super() 通常指向父类构造函数，QN 格式为: 父类QN.父类名
				return &model.CodeElement{
					Kind: model.Method, Name: superElem.Name,
					QualifiedName: superElem.QualifiedName + "." + superElem.Name,
				}
			}
		}
		// 兜底方案
		return &model.CodeElement{Kind: model.Method, Name: "super", QualifiedName: "java.lang.Exception.Exception"}
	}

	if prefix == "this" && sourceElem != nil {
		classQN := sourceElem.QualifiedName
		if idx := strings.LastIndex(classQN, "."); idx != -1 {
			classQN = classQN[:idx]
		}
		if resolved := e.resolveInInheritanceChain(classQN, name, kind, gCtx); resolved != nil {
			return resolved
		}
		return e.resolveTargetElement(classQN+"."+name, kind, fCtx, gCtx)
	}

	resolvedPrefix := e.resolveTargetElement(e.cleanTypeName(prefix), model.Variable, fCtx, gCtx)
	fullQN := resolvedPrefix.QualifiedName + "." + name
	return &model.CodeElement{Kind: kind, Name: name, QualifiedName: fullQN}
}

func (e *Extractor) resolveInInheritanceChain(classQN, memberName string, kind model.ElementKind, gCtx *model.GlobalContext) *model.CodeElement {
	currQN, visited := classQN, make(map[string]bool)
	for currQN != "" && !visited[currQN] {
		visited[currQN] = true
		targetQN := currQN + "." + memberName
		if defs, ok := gCtx.DefinitionsByQN[targetQN]; ok {
			return defs[0].Element
		}

		defs, ok := gCtx.DefinitionsByQN[currQN]
		if !ok || len(defs) == 0 || defs[0].Element.Extra == nil || defs[0].Element.Extra.ClassExtra == nil {
			break
		}

		rawSuper := defs[0].Element.Extra.ClassExtra.SuperClass
		if rawSuper == "" || rawSuper == "Object" {
			break
		}

		cleanSuper, found := e.cleanTypeName(rawSuper), false
		if _, ok := gCtx.DefinitionsByQN[cleanSuper]; ok {
			currQN, found = cleanSuper, true
		} else {
			for qn := range gCtx.DefinitionsByQN {
				if strings.HasSuffix(qn, "."+cleanSuper) {
					currQN, found = qn, true
					break
				}
			}
		}
		if !found {
			break
		}
	}
	return nil
}

// --- Helpers: AST & String Cleaning ---

func (e *Extractor) getObjectPrefix(node *sitter.Node, parentKind string, fCtx *model.FileContext) string {
	parent := node.Parent()
	// 向上寻找指定类型的父节点
	for parent != nil && parent.Kind() != parentKind {
		parent = parent.Parent()
	}
	if parent == nil {
		return ""
	}

	// 针对 method_invocation
	if obj := parent.ChildByFieldName("object"); obj != nil {
		return obj.Utf8Text(*fCtx.SourceBytes)
	}

	// explicit_constructor_invocation 不需要前缀，因为它本身就是 super/this 调用
	return ""
}

func (e *Extractor) determineSourceElement(n *sitter.Node, fCtx *model.FileContext, gCtx *model.GlobalContext) *model.CodeElement {
	for curr := n.Parent(); curr != nil; curr = curr.Parent() {
		if strings.Contains(curr.Kind(), "declaration") {
			if nameNode := curr.ChildByFieldName("name"); nameNode != nil {
				name := nameNode.Utf8Text(*fCtx.SourceBytes)
				for _, entry := range gCtx.ResolveSymbol(fCtx, name) {
					if int(curr.StartPosition().Row)+1 == entry.Element.Location.StartLine {
						return entry.Element
					}
				}
			}
		}
	}
	return nil
}

func (e *Extractor) stripAnnotationArgs(name string) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), "@")
	if idx := strings.Index(name, "("); idx != -1 {
		return name[:idx]
	}
	return name
}

func (e *Extractor) cleanTypeName(name string) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), "@")
	if idx := strings.Index(name, " extends "); idx != -1 {
		name = name[idx+len(" extends "):]
	}
	if idx := strings.Index(name, "<"); idx != -1 {
		name = name[:idx]
	}
	name = strings.TrimSuffix(name, "[]")
	name = strings.TrimSuffix(name, "...")
	return strings.TrimSpace(name)
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
