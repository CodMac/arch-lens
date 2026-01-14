package java

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/core"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

var genericRegex = regexp.MustCompile(`<([^>]+)>`)

type Extractor struct{}

func NewJavaExtractor() *Extractor {
	return &Extractor{}
}

// JavaActionQuery 增强说明：
// 1. 增加 (identifier) @maybe_use_target 以捕获 return id; 中的 id
// 2. 确保 assign_target 能覆盖标识符和成员变量访问
const JavaActionQuery = `
[
  (method_invocation name: (identifier) @call_target) @call_stmt
  (object_creation_expression
    type: [
        (type_identifier) @create_target 
        (generic_type (type_identifier) @create_target)
    ]) @create_stmt
  (field_access field: (identifier) @use_target) @use_stmt
  (assignment_expression left: [
        (identifier) @assign_target
        (field_access field: (identifier) @assign_target)
    ]) @assign_stmt
  (cast_expression type: (type_identifier) @cast_target) @cast_stmt
  (identifier) @maybe_use_target
]
`

func (e *Extractor) Extract(filePath string, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gCtx.FileContexts[filePath]
	if !ok {
		return nil, fmt.Errorf("file context not found: %s", filePath)
	}

	var relations []*model.DependencyRelation

	// 1. 组织层级
	relations = append(relations, e.extractHierarchyRelations(fCtx, gCtx)...)

	// 2. 静态结构 (含 THROW, PARAM, RETURN, TYPE_ARG)
	relations = append(relations, e.extractStructuralRelations(fCtx, gCtx)...)

	// 3. 动态动作 (含 CALL, CREATE, USE, ASSIGN, CAPTURE)
	actionRels, err := e.extractActionRelations(fCtx, gCtx)
	if err != nil {
		return nil, err
	}
	relations = append(relations, actionRels...)

	return relations, nil
}

// --- 1. 组织层级 ---
func (e *Extractor) extractHierarchyRelations(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			if entry.ParentQN != "" {
				if parents, ok := gCtx.DefinitionsByQN[entry.ParentQN]; ok && len(parents) > 0 {
					rels = append(rels, &model.DependencyRelation{
						Type:   model.Contain,
						Source: parents[0].Element,
						Target: entry.Element,
					})
				}
			} else {
				// 修复点：测试用例期待 Source 为 PACKAGE 类型的 CodeElement
				rels = append(rels, &model.DependencyRelation{
					Type: model.Contain,
					Source: &model.CodeElement{
						Name:          fCtx.PackageName,
						QualifiedName: fCtx.PackageName,
						Kind:          model.Package,
					},
					Target: entry.Element,
				})
			}
		}
	}
	return rels
}

// --- 2. 静态结构 ---
func (e *Extractor) extractStructuralRelations(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			elem := entry.Element
			if elem.Extra == nil || elem.Extra.Mores == nil {
				continue
			}
			m := elem.Extra.Mores

			// IMPLEMENT
			if ifaces, ok := m[ClassImplementedInterfaces].([]string); ok {
				for _, iface := range ifaces {
					rels = append(rels, &model.DependencyRelation{
						Type:   model.Implement,
						Source: elem,
						Target: e.quickResolve(e.clean(iface), model.Interface, gCtx, fCtx),
					})
				}
			}

			// THROW
			if throws, ok := m[MethodThrowsTypes].([]string); ok {
				for _, t := range throws {
					rels = append(rels, &model.DependencyRelation{
						Type:   model.Throw,
						Source: elem,
						Target: e.quickResolve(e.clean(t), model.Class, gCtx, fCtx),
					})
				}
			}

			// RETURN & PARAMETER
			if ret, ok := m[MethodReturnType].(string); ok && ret != "void" {
				rels = append(rels, &model.DependencyRelation{
					Type:   model.Return,
					Source: elem,
					Target: e.quickResolve(e.clean(ret), model.Type, gCtx, fCtx),
				})
			}

			// PARAMETER (通过元数据解析)
			if params, ok := m[MethodParameters].([]string); ok {
				for _, p := range params {
					typeName := strings.Fields(p)[0]
					rels = append(rels, &model.DependencyRelation{
						Type:   model.Parameter,
						Source: elem,
						Target: e.quickResolve(e.clean(typeName), model.Type, gCtx, fCtx),
					})
				}
			}
		}
	}
	return rels
}

// --- 3. 动作提取 ---
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

		// 捕获的最外层节点作为上下文
		stmtNode := &match.Captures[0].Node
		sourceElem := e.determinePreciseSource(stmtNode, fCtx, gCtx)
		if sourceElem == nil {
			continue
		}

		for _, cap := range match.Captures {
			name := q.CaptureNames()[cap.Index]

			// 排除定义处的标识符，防止 self-loop
			if name == "maybe_use_target" && e.isDefinitionNode(&cap.Node) {
				continue
			}

			relType, kind := e.mapAction(name, &cap.Node, fCtx.SourceBytes)
			if relType == "" {
				continue
			}

			rawText := cap.Node.Utf8Text(*fCtx.SourceBytes)
			target := e.quickResolve(e.clean(rawText), kind, gCtx, fCtx)

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

// 修复点：增加对赋值左侧的判断逻辑
func (e *Extractor) mapAction(capName string, node *sitter.Node, src *[]byte) (model.DependencyType, model.ElementKind) {
	switch capName {
	case "assign_target":
		return model.Assign, model.Field
	case "call_target":
		return model.Call, model.Method
	case "create_target":
		return model.Create, model.Class
	case "use_target", "maybe_use_target":
		// 关键点：如果标识符在赋值表达式左边，应识别为 ASSIGN
		if e.isWithinAssignmentLeft(node, src) {
			return model.Assign, model.Field
		}
		return model.Use, model.Field
	case "cast_target":
		return model.Cast, model.Class
	}
	return "", model.Unknown
}

func (e *Extractor) isWithinAssignmentLeft(n *sitter.Node, src *[]byte) bool {
	for curr := n.Parent(); curr != nil; curr = curr.Parent() {
		if curr.Kind() == "assignment_expression" {
			left := curr.ChildByFieldName("left")
			if left != nil && left.Utf8Text(*src) == n.Utf8Text(*src) {
				return true
			}
			break
		}
		if strings.HasSuffix(curr.Kind(), "statement") {
			break
		}
	}
	return false
}

func (e *Extractor) isDefinitionNode(n *sitter.Node) bool {
	parent := n.Parent()
	if parent == nil {
		return false
	}
	parentKind := parent.Kind()
	return strings.Contains(parentKind, "declaration") || parentKind == "variable_declarator"
}

func (e *Extractor) determinePreciseSource(n *sitter.Node, fCtx *core.FileContext, gCtx *core.GlobalContext) *model.CodeElement {
	for curr := n.Parent(); curr != nil; curr = curr.Parent() {
		k := curr.Kind()
		// 1. 变量/字段定义中发生动作 (如 Field Initializer)
		if k == "variable_declarator" || k == "field_declaration" {
			nameNode := curr.ChildByFieldName("name")
			if nameNode != nil {
				txt := nameNode.Utf8Text(*fCtx.SourceBytes)
				if defs, ok := fCtx.DefinitionsBySN[txt]; ok {
					return defs[0].Element
				}
			}
		}
		// 2. 方法体内发生动作
		if k == "method_declaration" || k == "constructor_declaration" {
			line := int(curr.StartPosition().Row) + 1
			for _, defs := range fCtx.DefinitionsBySN {
				for _, d := range defs {
					if d.Element.Kind == model.Method && d.Element.Location.StartLine == line {
						return d.Element
					}
				}
			}
		}
	}
	return nil
}

func (e *Extractor) quickResolve(symbol string, kind model.ElementKind, gCtx *core.GlobalContext, fCtx *core.FileContext) *model.CodeElement {
	entries := gCtx.ResolveSymbol(fCtx, symbol)
	if len(entries) > 0 {
		return entries[0].Element
	}

	return &model.CodeElement{Name: symbol, QualifiedName: symbol, Kind: kind}
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
