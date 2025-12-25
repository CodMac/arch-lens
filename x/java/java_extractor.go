package java

import (
	"fmt"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	"github.com/CodMac/go-treesitter-dependency-analyzer/parser"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Extractor 增强版：基于 AST 倒推逻辑，支持内部类、枚举、泛型及注解提取
type Extractor struct{}

func NewJavaExtractor() *Extractor {
	return &Extractor{}
}

// --- Tree-sitter Queries ---

const (
	JavaDefinitionQuery = `
      (program
         [
            (package_declaration (scoped_identifier) @package_name) @package_def
            (import_declaration (scoped_identifier) @import_target) @import_def
            
            (class_declaration
               (modifiers
                  [
                     (marker_annotation name: (identifier) @annotation_name)
                     (annotation name: (identifier) @annotation_name)
                  ] @annotation_stmt
               )?
               name: (identifier) @class_name
               superclass: (superclass (type_identifier) @extends_class)?
               interfaces: (super_interfaces 
                  (type_list 
                     [
                        (type_identifier) @implements_interface
                        (generic_type (type_identifier) @implements_interface)
                     ]+
                  )
               )?
               body: (class_body [
                  (field_declaration 
                     type: [
                        (type_identifier) @field_type
                        (integral_type) @field_type
                        (generic_type) @field_type
                     ]
                     declarator: (variable_declarator name: (identifier) @field_name)
                  ) @field_def
                  (method_declaration
                     type: [
                        (type_identifier) @return_type
                        (integral_type) @return_type
                        (void_type) @return_type
                        (generic_type) @return_type
                     ]
                     name: (identifier) @method_name
                     (throws (type_identifier) @throws_type)?
                  ) @method_def
                  (constructor_declaration name: (identifier) @method_name) @method_def
                  (class_declaration) @inner_class_def
               ])
            ) @class_def

            (enum_declaration
               name: (identifier) @enum_name
               body: (enum_body 
                  (enum_constant name: (identifier) @enum_constant_name) @enum_constant_def
                  (enum_body_declarations [
                     (field_declaration) @field_def
                     (method_declaration) @method_def
                     (constructor_declaration) @method_def
                  ]*)
               )
            ) @enum_def

            (interface_declaration
               name: (identifier) @interface_name
               interfaces: (super_interfaces (type_list (type_identifier) @extends_interface+))?
            ) @interface_def
         ]
      )
    `

	JavaRelationQuery = `
       [
          (method_invocation name: (identifier) @call_target) @call_stmt
          (object_creation_expression type: (type_identifier) @create_target_name) @create_stmt
          (field_access field: (identifier) @use_field_name) @use_field_stmt
          (cast_expression type: (type_identifier) @cast_type) @cast_stmt
       ]
    `
)

type RelationHandler func(q *sitter.Query, match *sitter.QueryMatch, sourceBytes *[]byte, filePath string, gc *model.GlobalContext) ([]*model.DependencyRelation, error)

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
	sourceBytes := fCtx.SourceBytes
	rootNode := fCtx.RootNode

	if err := e.processQuery(rootNode, sourceBytes, tsLang, JavaDefinitionQuery, filePath, gCtx, &relations, e.handleDefinitionAndStructureRelations); err != nil {
		return nil, err
	}

	if err := e.processQuery(rootNode, sourceBytes, tsLang, JavaRelationQuery, filePath, gCtx, &relations, e.handleActionRelations); err != nil {
		return nil, err
	}

	return relations, nil
}

// handleDefinitionAndStructureRelations 核心逻辑优化
func (e *Extractor) handleDefinitionAndStructureRelations(q *sitter.Query, match *sitter.QueryMatch, sourceBytes *[]byte, filePath string, gc *model.GlobalContext) ([]*model.DependencyRelation, error) {
	relations := make([]*model.DependencyRelation, 0)

	if len(match.Captures) == 0 {
		return relations, nil
	}
	sourceNode := &match.Captures[0].Node

	sourceElement := e.determineSourceElement(sourceNode, sourceBytes, filePath, gc)
	if sourceElement == nil {
		sourceElement = &model.CodeElement{Kind: model.File, QualifiedName: filePath, Path: filePath}
	}

	// 1. 内部类 (CONTAIN)
	if innerDef := e.findCapturedNode(q, match, "inner_class_def"); innerDef != nil {
		nameNode := innerDef.ChildByFieldName("name")
		if nameNode != nil {
			relations = append(relations, &model.DependencyRelation{
				Type:   model.Contain,
				Source: sourceElement,
				Target: e.resolveTargetElement(nameNode, model.Class, sourceBytes, filePath, gc),
			})
		}
	}

	// 2. 枚举常量 (CONTAIN)
	if constantDef := e.findCapturedNode(q, match, "enum_constant_def"); constantDef != nil {
		nameNode := e.findCapturedNode(q, match, "enum_constant_name")
		if nameNode != nil {
			relations = append(relations, &model.DependencyRelation{
				Type:     model.Contain,
				Source:   sourceElement,
				Target:   e.resolveTargetElement(nameNode, model.EnumConstant, sourceBytes, filePath, gc),
				Location: e.nodeToLocation(constantDef, filePath),
			})
		}
	}

	// 3. 基础结构关系 (优先使用解析后的全名和 Kind)
	if node := e.findCapturedNode(q, match, "import_target"); node != nil {
		relations = append(relations, &model.DependencyRelation{
			Type:   model.Import,
			Source: sourceElement,
			Target: e.resolveTargetElement(node, model.Package, sourceBytes, filePath, gc),
		})
	}
	if node := e.findCapturedNode(q, match, "extends_class"); node != nil {
		relations = append(relations, &model.DependencyRelation{
			Type:     model.Extend,
			Source:   sourceElement,
			Target:   e.resolveTargetElement(node, model.Class, sourceBytes, filePath, gc),
			Location: e.nodeToLocation(node, filePath),
		})
	}
	if node := e.findCapturedNode(q, match, "implements_interface"); node != nil {
		relations = append(relations, &model.DependencyRelation{
			Type:     model.Implement,
			Source:   sourceElement,
			Target:   e.resolveTargetElement(node, model.Interface, sourceBytes, filePath, gc),
			Location: e.nodeToLocation(node, filePath),
		})
	}
	if node := e.findCapturedNode(q, match, "annotation_name"); node != nil {
		relations = append(relations, &model.DependencyRelation{
			Type:     model.Annotation,
			Source:   sourceElement,
			Target:   e.resolveTargetElement(node, model.KAnnotation, sourceBytes, filePath, gc),
			Location: e.nodeToLocation(node, filePath),
		})
	}

	return relations, nil
}

func (e *Extractor) handleActionRelations(q *sitter.Query, match *sitter.QueryMatch, sourceBytes *[]byte, filePath string, gc *model.GlobalContext) ([]*model.DependencyRelation, error) {
	relations := make([]*model.DependencyRelation, 0)
	if len(match.Captures) == 0 {
		return relations, nil
	}
	sourceNode := &match.Captures[0].Node
	sourceElement := e.determineSourceElement(sourceNode, sourceBytes, filePath, gc)
	if sourceElement == nil {
		sourceElement = &model.CodeElement{Kind: model.File, QualifiedName: filePath, Path: filePath}
	}

	// 行为关系映射
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

	for _, m := range mapping {
		if node := e.findCapturedNode(q, match, m.capName); node != nil {
			relations = append(relations, &model.DependencyRelation{
				Type:     m.relType,
				Source:   sourceElement,
				Target:   e.resolveTargetElement(node, m.defKind, sourceBytes, filePath, gc),
				Location: e.nodeToLocation(node, filePath),
			})
		}
	}

	return relations, nil
}

// resolveTargetElement 核心优化：优先从全局上下文查找真实符号信息
func (e *Extractor) resolveTargetElement(n *sitter.Node, defaultKind model.ElementKind, sb *[]byte, fp string, gc *model.GlobalContext) *model.CodeElement {
	rawName := e.getNodeContent(n, *sb)

	// 1. 尝试从 GlobalContext 解析
	fc, ok := gc.FileContexts[fp]
	if ok {
		entries := gc.ResolveSymbol(fc, rawName)
		if len(entries) > 0 {
			// 优先使用查找到的第一个确切定义
			found := entries[0].Element
			return &model.CodeElement{
				Kind:          found.Kind,          // 使用真实查找到的类型（如 Enum 还是 Class）
				Name:          found.Name,          // 使用真实名称
				QualifiedName: found.QualifiedName, // 使用全限定名
				Path:          found.Path,
			}
		}
	}

	// 2. 兜底逻辑：如果解析不到，使用默认 Kind 和 rawName
	return &model.CodeElement{
		Kind:          defaultKind,
		Name:          rawName,
		QualifiedName: rawName, // 此时由于没解析到，QualifiedName 等同于 rawName
	}
}

// determineSourceElement 向上遍历语法树寻找最近的定义容器
func (e *Extractor) determineSourceElement(n *sitter.Node, sb *[]byte, fp string, gc *model.GlobalContext) *model.CodeElement {
	curr := n.Parent()
	for curr != nil {
		k := curr.Kind()
		if k == "method_declaration" || k == "class_declaration" || k == "enum_declaration" || k == "interface_declaration" || k == "constructor_declaration" {
			nameNode := curr.ChildByFieldName("name")
			if nameNode != nil {
				var kind model.ElementKind
				switch k {
				case "enum_declaration":
					kind = model.Enum
				case "method_declaration", "constructor_declaration":
					kind = model.Method
				case "interface_declaration":
					kind = model.Interface
				default:
					kind = model.Class
				}

				// 这里也调用 resolveTargetElement 确保 Source 也是 Qualified 的
				return e.resolveTargetElement(nameNode, kind, sb, fp, gc)
			}
		}
		curr = curr.Parent()
	}
	return nil
}

// --- 辅助方法保持不变 ---

func (e *Extractor) processQuery(rootNode *sitter.Node, sourceBytes *[]byte, tsLang *sitter.Language, queryStr string, filePath string, gc *model.GlobalContext, relations *[]*model.DependencyRelation, handler RelationHandler) error {
	q, err := sitter.NewQuery(tsLang, queryStr)
	if err != nil {
		return fmt.Errorf("Query init error: %w", err)
	}
	defer q.Close()
	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, rootNode, *sourceBytes)
	for {
		match := matches.Next()
		if match == nil {
			break
		}
		newRels, err := handler(q, match, sourceBytes, filePath, gc)
		if err != nil {
			return err
		}
		*relations = append(*relations, newRels...)
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

func (e *Extractor) getNodeContent(n *sitter.Node, sb []byte) string {
	if n == nil {
		return ""
	}
	return n.Utf8Text(sb)
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
