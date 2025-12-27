package java

import (
	"fmt"
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	"github.com/CodMac/go-treesitter-dependency-analyzer/parser"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Extractor struct{}

func NewJavaExtractor() *Extractor {
	return &Extractor{}
}

const (
	// JavaActionQuery 仅捕获方法体内的动态行为
	JavaActionQuery = `
       [
          (method_invocation name: (identifier) @call_target) @call_stmt
          (object_creation_expression type: [
              (type_identifier) @create_target_name
              (generic_type (type_identifier) @create_target_name)
          ]) @create_stmt
          (field_access field: (identifier) @use_field_name) @use_field_stmt
          (cast_expression type: (type_identifier) @cast_type) @cast_stmt
       ]
    `
)

// --- 核心提取逻辑 ---

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

	// 1. 结构化关系提取 (CONTAIN, EXTEND, IMPLEMENT, ANNOTATION, PARAMETER, RETURN, THROW)
	// 利用 Collector 已经生成的 Definition 及其 Extra 信息
	structRels := e.extractStructuralRelations(fCtx, gCtx)
	relations = append(relations, structRels...)

	// 2. 行为关系提取 (CALL, CREATE, USE, CAST)
	// 利用 Tree-sitter Query 扫描方法体内部
	actionRels, err := e.processActionQuery(fCtx, gCtx, tsLang)
	if err != nil {
		return nil, err
	}
	relations = append(relations, actionRels...)

	return relations, nil
}

// extractStructuralRelations 基于 Collector 的结果和 model 层级提取完整关系
func (e *Extractor) extractStructuralRelations(fCtx *model.FileContext, gCtx *model.GlobalContext) []*model.DependencyRelation {
	rels := make([]*model.DependencyRelation, 0)

	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			elem := entry.Element
			if elem.Extra == nil {
				continue
			}

			// 1. ANNOTATION (通用：类、方法、字段均可拥有注解)
			for _, anno := range elem.Extra.Annotations {
				cleanName := e.stripGenericsAndAt(anno)
				rels = append(rels, &model.DependencyRelation{
					Type:   model.Annotation,
					Source: elem,
					Target: e.resolveTargetElement(cleanName, model.KAnnotation, fCtx, gCtx),
				})
			}

			// 2. CONTAIN: 处理父子层级关系 (排除包名作为父级的情况)
			if entry.ParentQN != "" && entry.ParentQN != fCtx.PackageName {
				if parents, ok := gCtx.DefinitionsByQN[entry.ParentQN]; ok {
					rels = append(rels, &model.DependencyRelation{
						Type:   model.Contain,
						Source: parents[0].Element,
						Target: elem,
					})
				}
			}

			// 3. ClassExtra 处理 (EXTEND, IMPLEMENT)
			if elem.Extra.ClassExtra != nil {
				ce := elem.Extra.ClassExtra
				if ce.SuperClass != "" {
					rels = append(rels, &model.DependencyRelation{
						Type:   model.Extend,
						Source: elem,
						Target: e.resolveTargetElement(e.stripGenericsAndAt(ce.SuperClass), model.Class, fCtx, gCtx),
					})
				}
				for _, imp := range ce.ImplementedInterfaces {
					rels = append(rels, &model.DependencyRelation{
						Type:   model.Implement,
						Source: elem,
						Target: e.resolveTargetElement(e.stripGenericsAndAt(imp), model.Interface, fCtx, gCtx),
					})
				}
			}

			// 4. MethodExtra 处理 (PARAMETER, RETURN, THROW)
			if elem.Extra.MethodExtra != nil {
				me := elem.Extra.MethodExtra
				// RETURN
				if me.ReturnType != "" && me.ReturnType != "void" {
					rels = append(rels, &model.DependencyRelation{
						Type:   model.Return,
						Source: elem,
						Target: e.resolveTargetElement(e.stripGenericsAndAt(me.ReturnType), model.Type, fCtx, gCtx),
					})
				}
				// THROW
				for _, tType := range me.ThrowsTypes {
					rels = append(rels, &model.DependencyRelation{
						Type:   model.Throw,
						Source: elem,
						Target: e.resolveTargetElement(e.stripGenericsAndAt(tType), model.Class, fCtx, gCtx),
					})
				}
				// PARAMETER (解析格式如 "String name" 或 "@NotNull List<User> list")
				for _, pInfo := range me.Parameters {
					parts := strings.Fields(pInfo)
					if len(parts) >= 1 {
						// 变量名通常是最后一个单词，倒数第二个通常是类型
						typeIdx := len(parts) - 2
						if typeIdx < 0 {
							typeIdx = 0
						}
						pType := parts[typeIdx]
						rels = append(rels, &model.DependencyRelation{
							Type:   model.Parameter,
							Source: elem,
							Target: e.resolveTargetElement(e.stripGenericsAndAt(pType), model.Type, fCtx, gCtx),
						})
					}
				}
			}
		}
	}
	return rels
}

// processActionQuery 基于 Query 提取行为关系
func (e *Extractor) processActionQuery(fCtx *model.FileContext, gCtx *model.GlobalContext, tsLang *sitter.Language) ([]*model.DependencyRelation, error) {
	rels := make([]*model.DependencyRelation, 0)
	q, err := sitter.NewQuery(tsLang, JavaActionQuery)
	if err != nil {
		return nil, err
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, fCtx.RootNode, *fCtx.SourceBytes)

	mapping := []struct {
		capName string
		relType model.DependencyType
		defKind model.ElementKind
	}{
		{"call_target", model.Call, model.Method},
		{"create_target_name", model.Create, model.Class},
		{"use_field_name", model.Use, model.Field},
		{"cast_type", model.Cast, model.Type},
	}

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

		for _, m := range mapping {
			if node := e.findCapturedNode(q, match, m.capName); node != nil {
				rawName := node.Utf8Text(*fCtx.SourceBytes)
				rels = append(rels, &model.DependencyRelation{
					Type:     m.relType,
					Source:   sourceElem,
					Target:   e.resolveTargetElement(e.stripGenericsAndAt(rawName), m.defKind, fCtx, gCtx),
					Location: e.nodeToLocation(node, fCtx.FilePath),
				})
			}
		}
	}
	return rels, nil
}

// --- 辅助方法 ---

// stripGenericsAndAt 清理泛型和注解符号 (e.g., "@Loggable" -> "Loggable", "List<String>" -> "List")
func (e *Extractor) stripGenericsAndAt(name string) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), "@")
	if idx := strings.Index(name, "<"); idx != -1 {
		return strings.TrimSpace(name[:idx])
	}
	return name
}

// resolveTargetElement 利用全局上下文将短名称还原为全限定名
func (e *Extractor) resolveTargetElement(cleanName string, defaultKind model.ElementKind, fCtx *model.FileContext, gCtx *model.GlobalContext) *model.CodeElement {
	entries := gCtx.ResolveSymbol(fCtx, cleanName)
	if len(entries) > 0 {
		found := entries[0].Element
		return &model.CodeElement{
			Kind:          found.Kind,
			Name:          found.Name,
			QualifiedName: found.QualifiedName,
			Path:          found.Path,
		}
	}

	return &model.CodeElement{
		Kind:          defaultKind,
		Name:          cleanName,
		QualifiedName: cleanName,
	}
}

// determineSourceElement 向上寻找最近的命名定义节点作为 Source
func (e *Extractor) determineSourceElement(n *sitter.Node, fCtx *model.FileContext, gCtx *model.GlobalContext) *model.CodeElement {
	curr := n.Parent()
	for curr != nil {
		kind := curr.Kind()
		if strings.Contains(kind, "declaration") || kind == "constructor_declaration" {
			nameNode := curr.ChildByFieldName("name")
			if nameNode != nil {
				name := nameNode.Utf8Text(*fCtx.SourceBytes)
				entries := gCtx.ResolveSymbol(fCtx, name)
				for _, entry := range entries {
					// 增加行号校验，防止重载方法冲突
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

func (e *Extractor) findCapturedNode(q *sitter.Query, match *sitter.QueryMatch, name string) *sitter.Node {
	idx, ok := q.CaptureIndexForName(name)
	if !ok {
		return nil
	}
	nodes := match.NodesForCaptureIndex(idx)
	if len(nodes) > 0 {
		return &nodes[0]
	}
	return nil
}

func (e *Extractor) nodeToLocation(n *sitter.Node, fp string) *model.Location {
	if n == nil {
		return nil
	}
	return &model.Location{
		FilePath:    fp,
		StartLine:   int(n.StartPosition().Row) + 1,
		EndLine:     int(n.EndPosition().Row) + 1,
		StartColumn: int(n.StartPosition().Column),
		EndColumn:   int(n.EndPosition().Column),
	}
}
