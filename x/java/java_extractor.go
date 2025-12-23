package java

import (
	"fmt"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	"github.com/CodMac/go-treesitter-dependency-analyzer/parser"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Extractor 增强版：支持内部类、枚举类及其成员的关系提取
type Extractor struct{}

func NewJavaExtractor() *Extractor {
	return &Extractor{}
}

// --- Tree-sitter Queries ---

const (
	// JavaDefinitionQuery 补充了 enum 和内部类的支持
	JavaDefinitionQuery = `
       (program
          [
             (package_declaration (scoped_identifier) @package_name) @package_def
             (import_declaration (scoped_identifier) @import_target) @import_def
             
             ;; 类与内部类定义
             (class_declaration
                name: (identifier) @class_name
                (superclass (identifier) @extends_class)?
                (super_interfaces (type_list (identifier) @implements_interface+))?
                body: (class_body [
                   ;; 成员变量
                   (field_declaration 
                      type: (_) @field_type
                      (variable_declarator (identifier) @field_name)
                   ) @field_def
                   ;; 方法
                   (method_declaration
                      type: (_) @return_type
                      name: (identifier) @method_name
                      parameters: (formal_parameters (formal_parameter type: (_) @param_type) @param_node+)?
                      (throws (scoped_type_identifier) @throws_type)?
                   ) @method_def
                   ;; 内部类 (Nested Class)
                   (class_declaration) @inner_class_def
                   ;; 内部枚举 (Nested Enum)
                   (enum_declaration) @inner_enum_def
                ])
                (modifiers (annotation name: (identifier) @annotation_name)) @annotation_stmt
             ) @class_def

             ;; 枚举类定义
             (enum_declaration
                name: (identifier) @enum_name
                (super_interfaces (type_list (identifier) @implements_interface+))?
                body: (enum_body 
                   (enum_constant name: (identifier) @enum_constant_name) @enum_constant_def
                )
             ) @enum_def
             
             (interface_declaration
                name: (identifier) @interface_name
                (super_interfaces (type_list (identifier) @extends_interface+))?
             ) @interface_def
          ]
       )
    `
	// 操作关系：涵盖方法调用、对象创建、字段访问等
	JavaRelationQuery = `
       [
          (method_invocation name: (identifier) @call_target) @call_stmt
          (object_creation_expression type: (unqualified_class_instance_expression type: (identifier) @create_target_name)) @create_stmt
          (field_access field: (identifier) @use_field_name) @use_field_stmt
          (cast_expression type: (_) @cast_type) @cast_stmt
          (identifier) @use_identifier
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

	// 1. 结构和定义（内部类、枚举、包含关系）
	if err := e.processQuery(rootNode, sourceBytes, tsLang, JavaDefinitionQuery, filePath, gCtx, &relations, e.handleDefinitionAndStructureRelations); err != nil {
		return nil, err
	}

	// 2. 操作关系
	if err := e.processQuery(rootNode, sourceBytes, tsLang, JavaRelationQuery, filePath, gCtx, &relations, e.handleActionRelations); err != nil {
		return nil, err
	}

	return relations, nil
}

func (e *Extractor) handleDefinitionAndStructureRelations(q *sitter.Query, match *sitter.QueryMatch, sourceBytes *[]byte, filePath string, gc *model.GlobalContext) ([]*model.DependencyRelation, error) {
	relations := make([]*model.DependencyRelation, 0)

	sourceNode := match.NodesForCaptureIndex(0)[0]
	sourceElement := e.determineSourceElement(&sourceNode, sourceBytes, filePath, gc)
	if sourceElement == nil {
		sourceElement = &model.CodeElement{Kind: model.File, QualifiedName: filePath, Path: filePath}
	}

	// 处理内部类/内部枚举的 CONTAIN 关系
	if innerDef := e.findCapturedNode(q, match, "inner_class_def"); innerDef != nil {
		nameNode := innerDef.ChildByFieldName("name")
		if nameNode != nil {
			relations = append(relations, &model.DependencyRelation{
				Type:   model.Contain,
				Source: sourceElement,
				Target: &model.CodeElement{Kind: model.Class, Name: e.getNodeContent(nameNode, *sourceBytes), QualifiedName: e.resolveQualifiedName(nameNode, sourceBytes, filePath, gc)},
			})
		}
	}

	// 处理枚举常量 (Enum -> Enum Constant)
	if constantDef := e.findCapturedNode(q, match, "enum_constant_def"); constantDef != nil {
		nameNode := e.findCapturedNode(q, match, "enum_constant_name")
		if nameNode != nil {
			relations = append(relations, &model.DependencyRelation{
				Type:     model.Contain,
				Source:   sourceElement,
				Target:   &model.CodeElement{Kind: model.EnumConstant, Name: e.getNodeContent(nameNode, *sourceBytes)},
				Location: e.nodeToLocation(constantDef, filePath),
			})
		}
	}

	// 基础引用关系
	if node := e.findCapturedNode(q, match, "import_target"); node != nil {
		name := e.getNodeContent(node, *sourceBytes)
		relations = append(relations, &model.DependencyRelation{
			Type: model.Import, Source: sourceElement,
			Target:   &model.CodeElement{Kind: model.Package, Name: name, QualifiedName: name},
			Location: e.nodeToLocation(node, filePath),
		})
	}
	if node := e.findCapturedNode(q, match, "extends_class"); node != nil {
		relations = append(relations, e.newRel(model.Extend, sourceElement, node, model.Class, sourceBytes, filePath, gc))
	}
	if node := e.findCapturedNode(q, match, "implements_interface"); node != nil {
		relations = append(relations, e.newRel(model.Implement, sourceElement, node, model.Interface, sourceBytes, filePath, gc))
	}
	// 修正：使用 model.Annotation 作为关系类型，model.KAnnotation 作为目标 Kind
	if node := e.findCapturedNode(q, match, "annotation_name"); node != nil {
		relations = append(relations, e.newRel(model.Annotation, sourceElement, node, model.KAnnotation, sourceBytes, filePath, gc))
	}

	return relations, nil
}

func (e *Extractor) handleActionRelations(q *sitter.Query, match *sitter.QueryMatch, sourceBytes *[]byte, filePath string, gc *model.GlobalContext) ([]*model.DependencyRelation, error) {
	relations := make([]*model.DependencyRelation, 0)
	sourceNode := match.NodesForCaptureIndex(0)[0]
	sourceElement := e.determineSourceElement(&sourceNode, sourceBytes, filePath, gc)
	if sourceElement == nil {
		sourceElement = &model.CodeElement{Kind: model.File, QualifiedName: filePath, Path: filePath}
	}

	if node := e.findCapturedNode(q, match, "call_target"); node != nil {
		relations = append(relations, e.newRel(model.Call, sourceElement, node, model.Method, sourceBytes, filePath, gc))
	}
	if node := e.findCapturedNode(q, match, "create_target_name"); node != nil {
		relations = append(relations, e.newRel(model.Create, sourceElement, node, model.Class, sourceBytes, filePath, gc))
	}
	if node := e.findCapturedNode(q, match, "use_field_name"); node != nil {
		relations = append(relations, e.newRel(model.Use, sourceElement, node, model.Field, sourceBytes, filePath, gc))
	}
	if node := e.findCapturedNode(q, match, "cast_type"); node != nil {
		relations = append(relations, e.newRel(model.Cast, sourceElement, node, model.Type, sourceBytes, filePath, gc))
	}

	return relations, nil
}

// resolveQualifiedName 利用 GlobalContext 进行符号解析
func (e *Extractor) resolveQualifiedName(n *sitter.Node, sb *[]byte, fp string, gc *model.GlobalContext) string {
	name := e.getNodeContent(n, *sb)
	fc, ok := gc.FileContexts[fp]
	if !ok {
		return name
	}

	entries := gc.ResolveSymbol(fc, name)
	if len(entries) > 0 {
		return entries[0].Element.QualifiedName
	}

	return name
}

// determineSourceElement 向上溯源确定当前节点所属的代码实体
func (e *Extractor) determineSourceElement(n *sitter.Node, sb *[]byte, fp string, gc *model.GlobalContext) *model.CodeElement {
	curr := n.Parent()
	for curr != nil {
		k := curr.Kind()
		if k == "method_declaration" || k == "class_declaration" || k == "enum_declaration" || k == "interface_declaration" {
			nameNode := curr.ChildByFieldName("name")
			if nameNode != nil {
				var kind model.ElementKind
				switch k {
				case "enum_declaration":
					kind = model.Enum
				case "method_declaration":
					kind = model.Method
				case "interface_declaration":
					kind = model.Interface
				default:
					kind = model.Class
				}

				return &model.CodeElement{
					Kind:          kind,
					Name:          e.getNodeContent(nameNode, *sb),
					QualifiedName: e.resolveQualifiedName(nameNode, sb, fp, gc),
					Path:          fp,
				}
			}
		}
		curr = curr.Parent()
	}
	return nil
}

func (e *Extractor) processQuery(rootNode *sitter.Node, sourceBytes *[]byte, tsLang *sitter.Language, queryStr string, filePath string, gc *model.GlobalContext, relations *[]*model.DependencyRelation, handler RelationHandler) error {
	q, err := sitter.NewQuery(tsLang, queryStr)
	if err != nil {
		return err
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

func (e *Extractor) newRel(t model.DependencyType, src *model.CodeElement, targetNode *sitter.Node, targetKind model.ElementKind, sb *[]byte, fp string, gc *model.GlobalContext) *model.DependencyRelation {
	name := e.getNodeContent(targetNode, *sb)
	return &model.DependencyRelation{
		Type: t, Source: src,
		Target:   &model.CodeElement{Kind: targetKind, Name: name, QualifiedName: e.resolveQualifiedName(targetNode, sb, fp, gc)},
		Location: e.nodeToLocation(targetNode, fp),
	}
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
		FilePath:  fp,
		StartLine: int(n.StartPosition().Row) + 1, EndLine: int(n.EndPosition().Row) + 1,
		StartColumn: int(n.StartPosition().Column), EndColumn: int(n.EndPosition().Column),
	}
}
