package java

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/core"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// 预编译正则用于解析泛型 TypeArg
var genericRegex = regexp.MustCompile(`<([^>]+)>`)

type Extractor struct{}

func NewJavaExtractor() *Extractor {
	return &Extractor{}
}

// JavaActionQuery 涵盖动作捕获
const JavaActionQuery = `
[
  (method_invocation name: (identifier) @call_target) @call_stmt
  (method_reference (identifier) @ref_target) @ref_stmt
  (object_creation_expression
    type: [
        (type_identifier) @create_target 
        (generic_type (type_identifier) @create_target)
    ]) @create_stmt
  (field_access field: (identifier) @use_field_target) @use_field_stmt
  (cast_expression type: (type_identifier) @cast_target) @cast_stmt
  (assignment_expression left: (identifier) @assign_target) @assign_stmt
]
`

func (e *Extractor) Extract(filePath string, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gCtx.FileContexts[filePath]
	if !ok {
		return nil, fmt.Errorf("file context not found: %s", filePath)
	}

	var relations []*model.DependencyRelation

	// 1. 组织与层级 (IMPORT, CONTAIN)
	relations = append(relations, e.extractHierarchyRelations(fCtx, gCtx)...)

	// 2. 静态结构 (EXTEND, IMPLEMENT, ANNOTATION, PARAMETER, RETURN, THROW, TYPE_ARG)
	relations = append(relations, e.extractStructuralRelations(fCtx, gCtx)...)

	// 3. 动态动作 (CALL, CREATE, USE, CAST, ASSIGN, CAPTURE)
	actionRels, err := e.extractActionRelations(fCtx, gCtx)
	if err != nil {
		return nil, err
	}
	relations = append(relations, actionRels...)

	return relations, nil
}

// --- 1. 组织层级优化 ---
func (e *Extractor) extractHierarchyRelations(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	fileElems, ok := gCtx.DefinitionsByQN[fCtx.FilePath]
	if !ok {
		return nil
	}
	fileSource := fileElems[0].Element

	// IMPORT
	for _, imports := range fCtx.Imports {
		for _, imp := range imports {
			target := e.quickResolve(imp.RawImportPath, imp.Kind, gCtx, fCtx)
			rels = append(rels, &model.DependencyRelation{
				Type:     model.Import,
				Source:   fileSource,
				Target:   target,
				Location: imp.Location,
			})
		}
	}

	// CONTAIN
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			if entry.ParentQN == "" {
				continue
			}

			if parents, ok := gCtx.DefinitionsByQN[entry.ParentQN]; ok && len(parents) > 0 {
				for _, parent := range parents {
					rels = append(rels, &model.DependencyRelation{
						Type:   model.Contain,
						Source: parent.Element,
						Target: entry.Element,
					})
				}
			}
		}
	}

	return rels
}

// --- 2. 结构关系增强 (ANNOTATION & EXTEND & IMPLEMENT & OVERRIDE & RETURN & PARAMETER & THROW & TypeArg) ---
func (e *Extractor) extractStructuralRelations(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			elem := entry.Element
			if elem.Extra == nil || elem.Extra.Mores == nil {
				continue
			}
			mores := elem.Extra.Mores

			// 1. ANNOTATION (独立重构函数，处理类、方法、字段、参数上的所有注解)
			for _, annoStr := range elem.Extra.Annotations {
				rels = append(rels, e.createAnnotationRelation(elem, annoStr, fCtx, gCtx))
			}

			// 2. EXTEND & IMPLEMENT & OVERRIDE
			if sc, ok := mores[ClassSuperClass].(string); ok && sc != "" {
				rels = append(rels, &model.DependencyRelation{
					Type:   model.Extend,
					Source: elem,
					Target: e.quickResolve(e.clean(sc), model.Class, gCtx, fCtx),
				})
			}
			if ifaces, ok := mores[ClassImplementedInterfaces].([]string); ok {
				for _, iface := range ifaces {
					relType := model.Implement
					if elem.Kind == model.Interface {
						relType = model.Extend
					}
					rels = append(rels, &model.DependencyRelation{
						Type:   relType,
						Source: elem,
						Target: e.quickResolve(e.clean(iface), model.Interface, gCtx, fCtx),
					})
				}
			}

			// 3. 方法签名 (RETURN, PARAMETER, THROW)
			if ret, ok := mores[MethodReturnType].(string); ok && ret != "void" {
				rels = append(rels, &model.DependencyRelation{
					Type:   model.Return,
					Source: elem,
					Target: e.quickResolve(e.clean(ret), model.Type, gCtx, fCtx),
				})
			}

			// 修改：针对 PARAMETER 的处理，剥离注解干扰
			if params, ok := mores[MethodParameters].([]string); ok {
				for _, p := range params {
					// 方案：从 Collector 存入的 "@NotNull String data" 中提取纯净类型名
					// 逻辑：过滤掉所有 @ 开头的单词，取第一个非 @ 的单词作为 Type
					parts := strings.Fields(p)
					var pureType string
					for _, part := range parts {
						if !strings.HasPrefix(part, "@") {
							pureType = part
							break
						}
					}

					if pureType != "" {
						rels = append(rels, &model.DependencyRelation{
							Type:   model.Parameter,
							Source: elem,
							Target: e.quickResolve(e.clean(pureType), model.Type, gCtx, fCtx),
						})
					}
				}
			}

			if throws, ok := mores[MethodThrowsTypes].([]string); ok {
				for _, t := range throws {
					rels = append(rels, &model.DependencyRelation{
						Type:   model.Throw,
						Source: elem,
						Target: e.quickResolve(e.clean(t), model.Class, gCtx, fCtx),
					})
				}
			}

			// 4. 泛型 TypeArg 提取
			rawTypes := []string{}
			if rt, ok := mores[MethodReturnType].(string); ok {
				rawTypes = append(rawTypes, rt)
			}
			if pt, ok := mores[MethodParameters].([]string); ok {
				rawTypes = append(rawTypes, pt...)
			}
			if vt, ok := mores[VariableType].(string); ok {
				rawTypes = append(rawTypes, vt)
			}

			for _, rt := range rawTypes {
				for _, arg := range e.extractTypeArgs(rt) {
					rels = append(rels, &model.DependencyRelation{
						Type:   model.TypeArg,
						Source: elem,
						Target: e.quickResolve(arg, model.Class, gCtx, fCtx),
					})
				}
			}
		}
	}
	return rels
}

// --- 3. 动作关系与 Capture ---
func (e *Extractor) extractActionRelations(fCtx *core.FileContext, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
	tsLang, _ := core.GetLanguage(core.LangJava)
	q, err := sitter.NewQuery(tsLang, JavaActionQuery)
	if err != nil {
		return nil, err
	}
	defer q.Close()

	var rels []*model.DependencyRelation
	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, fCtx.RootNode, *fCtx.SourceBytes)

	for {
		match := matches.Next()
		if match == nil {
			break
		}

		stmtNode := &match.Captures[0].Node
		sourceElem := e.determinePreciseSource(stmtNode, fCtx, gCtx)
		if sourceElem == nil {
			continue
		}

		for _, cap := range match.Captures {
			capName := q.CaptureNames()[cap.Index]
			rawText := cap.Node.Utf8Text(*fCtx.SourceBytes)

			// 逻辑分发
			relType, kind := e.mapAction(capName)
			if relType == "" {
				continue
			}

			target := e.quickResolve(e.clean(rawText), kind, gCtx, fCtx)

			// CAPTURE 判定逻辑：
			// 如果在一个 Lambda 作用域内 USE 了一个非 Lambda 参数、非 Field 的变量
			if relType == model.Use && strings.Contains(sourceElem.QualifiedName, "lambda$") {
				if e.isCaptured(target, sourceElem) {
					relType = model.Capture
				}
			}

			rels = append(rels, &model.DependencyRelation{
				Type:     relType,
				Source:   sourceElem,
				Target:   target,
				Location: e.toLoc(cap.Node, fCtx.FilePath),
				Mores: map[string]interface{}{
					RelRawText: rawText,
					RelContext: stmtNode.Kind(),
				},
			})
		}
	}
	return rels, nil
}

// --- 辅助逻辑 ---

func (e *Extractor) createAnnotationRelation(source *model.CodeElement, rawAnno string, fCtx *core.FileContext, gCtx *core.GlobalContext) *model.DependencyRelation {
	// 1. 解析字符串提取 Name 和初始 Mores
	targetName, mores := e.parseAnnotationString(rawAnno)

	// 2. 填充 Target 类型 (基于我们分析的 Collector 标识)
	mores[RelAnnotationTarget] = e.determineAnnotationTarget(source)

	return &model.DependencyRelation{
		Type:   model.Annotation,
		Source: source,
		Target: e.quickResolve(targetName, model.KAnnotation, gCtx, fCtx),
		Mores:  mores,
	}
}

func (e *Extractor) parseAnnotationString(raw string) (string, map[string]interface{}) {
	mores := make(map[string]interface{})
	mores[RelRawText] = raw

	content := strings.TrimPrefix(raw, "@")
	if idx := strings.Index(content, "("); idx != -1 {
		name := content[:idx]
		params := content[idx+1 : strings.LastIndex(content, ")")]

		if strings.Contains(params, "=") {
			mores[RelAnnotationParams] = params
		} else {
			mores[RelAnnotationValue] = params
		}
		return name, mores
	}
	return content, mores
}

func (e *Extractor) determineAnnotationTarget(elem *model.CodeElement) string {
	switch elem.Kind {
	case model.Class, model.Interface, model.Enum:
		return "TYPE"
	case model.Field:
		return "FIELD"
	case model.Method:
		return "METHOD"
	case model.Variable:
		// 关键：利用 Collector 填充的 VariableIsParam 判定
		if isParam, _ := elem.Extra.Mores[VariableIsParam].(bool); isParam {
			return "PARAMETER"
		}
		return "LOCAL_VARIABLE"
	}
	return "UNKNOWN"
}

func (e *Extractor) extractTypeArgs(signature string) []string {
	match := genericRegex.FindStringSubmatch(signature)
	if len(match) < 2 {
		return nil
	}
	// 处理 List<A, B> 情况
	args := strings.Split(match[1], ",")
	for i := range args {
		args[i] = e.clean(args[i])
	}
	return args
}

func (e *Extractor) isCaptured(target *model.CodeElement, source *model.CodeElement) bool {
	// 简单的 CAPTURE 判定准则：
	// Target 是变量，且 Target 的 QN 不包含 Source 的 QN 前缀（即不在当前 Lambda 内部定义）
	if target.Kind != model.Variable {
		return false
	}
	// 且 Target 不属于类字段 (这里需要你的 Resolver 能区分出 LocalVar 和 Field)
	return !strings.HasPrefix(target.QualifiedName, source.QualifiedName)
}

func (e *Extractor) mapAction(capName string) (model.DependencyType, model.ElementKind) {
	switch capName {
	case "call_target", "ref_target":
		return model.Call, model.Method
	case "create_target":
		return model.Create, model.Class
	case "use_field_target":
		return model.Use, model.Field
	case "assign_target":
		return model.Assign, model.Variable
	case "cast_target":
		return model.Cast, model.Class
	}
	return "", model.Unknown
}

func (e *Extractor) clean(s string) string {
	s = strings.TrimPrefix(s, "@")
	s = strings.Split(s, "<")[0]
	return strings.TrimSpace(s)
}

func (e *Extractor) toLoc(n sitter.Node, path string) *model.Location {
	return &model.Location{
		FilePath: path, StartLine: int(n.StartPosition().Row) + 1, EndLine: int(n.EndPosition().Row) + 1,
		StartColumn: int(n.StartPosition().Column), EndColumn: int(n.EndPosition().Column),
	}
}

func (e *Extractor) quickResolve(symbol string, kind model.ElementKind, gCtx *core.GlobalContext, fCtx *core.FileContext) *model.CodeElement {
	entries := gCtx.ResolveSymbol(fCtx, symbol)
	if len(entries) > 0 {
		return entries[0].Element
	}

	return &model.CodeElement{Name: symbol, QualifiedName: symbol, Kind: kind}
}

func (e *Extractor) determinePreciseSource(n *sitter.Node, fCtx *core.FileContext, gCtx *core.GlobalContext) *model.CodeElement {
	for curr := n.Parent(); curr != nil; curr = curr.Parent() {
		k := curr.Kind()
		// 1. 如果是在变量声明/字段初始化中创建
		if k == "variable_declarator" || k == "field_declaration" {
			nameNode := curr.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Utf8Text(*fCtx.SourceBytes)
				if entries, ok := fCtx.DefinitionsBySN[name]; ok {
					// 校验行号以防同名变量
					for _, entry := range entries {
						if entry.Element.Location.StartLine == int(curr.StartPosition().Row)+1 {
							return entry.Element
						}
					}
				}
			}
		}
		// 2. 否则归属于所在的 Method
		if k == "method_declaration" || k == "constructor_declaration" {
			for _, entries := range fCtx.DefinitionsBySN {
				for _, entry := range entries {
					if entry.Element.Kind == model.Method &&
						int(curr.StartPosition().Row)+1 == entry.Element.Location.StartLine {
						return entry.Element
					}
				}
			}
		}
	}
	return nil
}
