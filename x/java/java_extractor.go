package java

import (
	"fmt"
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	"github.com/CodMac/go-treesitter-dependency-analyzer/parser"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Extractor 实现了 extractor.Extractor 接口
type Extractor struct{}

func NewJavaExtractor() *Extractor {
	return &Extractor{}
}

// --- Tree-sitter Queries ---

const (
	// JavaDefinitionQuery 收集定义和结构关系 (CONTAIN, EXTEND, IMPLEMENT, USE, ANNOTATION)
	JavaDefinitionQuery = `
       (program
          [
             (package_declaration (scoped_identifier) @package_name) @package_def
             (import_declaration (scoped_identifier) @import_target) @import_def
             (class_declaration
                name: (identifier) @class_name
                (type_parameters)?
                (superclass (identifier) @extends_class)?
                (super_interfaces (type_list (identifier) @implements_interface+))?
                body: (class_body
                   (field_declaration 
                      type: (_) @field_type
                      (variable_declarator (identifier) @field_name)
                   ) @field_def
                   (method_declaration
                      type: (_) @return_type
                      name: (identifier) @method_name
                      parameters: (formal_parameters (formal_parameter type: (_) @param_type) @param_node+)?
                      (throws (scoped_type_identifier) @throws_type)?
                   ) @method_def
                   (constructor_declaration
                      name: (identifier) @constructor_name
                      parameters: (formal_parameters (formal_parameter type: (_) @param_type) @param_node+)?
                      (throws (scoped_type_identifier) @throws_type)?
                   ) @constructor_def
                )
                (modifiers (annotation name: (identifier) @annotation_name)) @annotation_stmt
             ) @class_def
             (interface_declaration
                name: (identifier) @interface_name
                (super_interfaces (type_list (identifier) @extends_interface+))?
                (modifiers (annotation name: (identifier) @annotation_name)) @annotation_stmt
             ) @interface_def
          ]
       )
    `
	// JavaRelationQuery 收集操作关系 (CALL, CREATE, USE, CAST, ANNOTATION)
	JavaRelationQuery = `
       [
          (method_invocation name: (identifier) @call_target) @call_stmt
          (object_creation_expression type: (unqualified_class_instance_expression type: (identifier) @create_target_name)) @create_stmt
          (field_access field: (identifier) @use_field_name) @use_field_stmt
          (cast_expression type: (_) @cast_type) @cast_stmt
          (identifier) @use_identifier
          (local_variable_declaration
             (modifiers (annotation name: (identifier) @annotation_name)) @annotation_stmt_local
          )
       ]
    `
)

// Extract 实现了 extractor.ContextExtractor 接口
func (e *Extractor) Extract(rootNode *sitter.Node, filePath string, gCtx *model.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gCtx.FileContexts[filePath]
	if !ok {
		return nil, fmt.Errorf("failed to get FileContext: %s", filePath)
	}

	relations := make([]*model.DependencyRelation, 0)
	tsLang, err := parser.GetLanguage(model.LangJava)
	if err != nil {
		return nil, err
	}
	sourceBytes := fCtx.SourceBytes

	// 1. 结构和定义关系
	if err := e.processQuery(rootNode, sourceBytes, tsLang, JavaDefinitionQuery, filePath, gCtx, &relations, e.handleDefinitionAndStructureRelations); err != nil {
		return nil, fmt.Errorf("failed to process definition query: %w", err)
	}

	// 2. 操作关系
	if err := e.processQuery(rootNode, sourceBytes, tsLang, JavaRelationQuery, filePath, gCtx, &relations, e.handleActionRelations); err != nil {
		return nil, fmt.Errorf("failed to process relation query: %w", err)
	}

	return relations, nil
}

type RelationHandler func(q *sitter.Query, match *sitter.QueryMatch, sourceBytes *[]byte, filePath string, gc *model.GlobalContext) ([]*model.DependencyRelation, error)

func (e *Extractor) handleDefinitionAndStructureRelations(q *sitter.Query, match *sitter.QueryMatch, sourceBytes *[]byte, filePath string, gc *model.GlobalContext) ([]*model.DependencyRelation, error) {
	relations := make([]*model.DependencyRelation, 0)

	sourceNode := match.NodesForCaptureIndex(0)[0]
	sourceElement := e.determineSourceElement(&sourceNode, sourceBytes, filePath, gc)
	if sourceElement == nil {
		sourceElement = &model.CodeElement{Kind: model.File, QualifiedName: filePath, Path: filePath}
	}

	// 1. IMPORT
	if importTargetNode := e.findCapturedNode(q, match, sourceBytes, "import_target"); importTargetNode != nil {
		importName := e.getNodeContent(importTargetNode, *sourceBytes)
		relations = append(relations, &model.DependencyRelation{
			Type:     model.Import,
			Source:   sourceElement,
			Target:   &model.CodeElement{Kind: model.Package, Name: importName, QualifiedName: importName},
			Location: e.nodeToLocation(importTargetNode, filePath),
		})
	}

	// 2. EXTEND (Class/Interface)
	if extendsNode := e.findCapturedNode(q, match, sourceBytes, "extends_class"); extendsNode != nil {
		relations = append(relations, &model.DependencyRelation{
			Type:     model.Extend,
			Source:   sourceElement,
			Target:   &model.CodeElement{Kind: model.Class, Name: e.getNodeContent(extendsNode, *sourceBytes), QualifiedName: e.resolveQualifiedName(extendsNode, sourceBytes, filePath, gc)},
			Location: e.nodeToLocation(extendsNode, filePath),
		})
	}
	if extendsIntfNode := e.findCapturedNode(q, match, sourceBytes, "extends_interface"); extendsIntfNode != nil {
		relations = append(relations, &model.DependencyRelation{
			Type:     model.Extend,
			Source:   sourceElement,
			Target:   &model.CodeElement{Kind: model.Interface, Name: e.getNodeContent(extendsIntfNode, *sourceBytes), QualifiedName: e.resolveQualifiedName(extendsIntfNode, sourceBytes, filePath, gc)},
			Location: e.nodeToLocation(extendsIntfNode, filePath),
		})
	}

	// 3. IMPLEMENT
	if implementsNode := e.findCapturedNode(q, match, sourceBytes, "implements_interface"); implementsNode != nil {
		relations = append(relations, &model.DependencyRelation{
			Type:     model.Implement,
			Source:   sourceElement,
			Target:   &model.CodeElement{Kind: model.Interface, Name: e.getNodeContent(implementsNode, *sourceBytes), QualifiedName: e.resolveQualifiedName(implementsNode, sourceBytes, filePath, gc)},
			Location: e.nodeToLocation(implementsNode, filePath),
		})
	}

	// 4. ANNOTATION
	if annotationNameNode := e.findCapturedNode(q, match, sourceBytes, "annotation_name"); annotationNameNode != nil {
		relations = append(relations, &model.DependencyRelation{
			Type:   model.Annotation,
			Source: sourceElement,
			// 注解类型使用 Interface
			Target:   &model.CodeElement{Kind: model.Interface, Name: e.getNodeContent(annotationNameNode, *sourceBytes), QualifiedName: e.resolveQualifiedName(annotationNameNode, sourceBytes, filePath, gc)},
			Location: e.nodeToLocation(annotationNameNode, filePath),
		})
	}

	// 5. TYPE USAGE
	if returnTypeNode := e.findCapturedNode(q, match, sourceBytes, "return_type"); returnTypeNode != nil {
		relations = append(relations, e.createTypeUsageRelation(sourceElement, returnTypeNode, sourceBytes, filePath, gc, "Return Type"))
	}
	if paramTypeNode := e.findCapturedNode(q, match, sourceBytes, "param_type"); paramTypeNode != nil {
		relations = append(relations, e.createTypeUsageRelation(sourceElement, paramTypeNode, sourceBytes, filePath, gc, "Parameter Type"))
	}
	if throwsTypeNode := e.findCapturedNode(q, match, sourceBytes, "throws_type"); throwsTypeNode != nil {
		relations = append(relations, e.createTypeUsageRelation(sourceElement, throwsTypeNode, sourceBytes, filePath, gc, "Throws Type"))
	}
	if fieldTypeNode := e.findCapturedNode(q, match, sourceBytes, "field_type"); fieldTypeNode != nil {
		relations = append(relations, e.createTypeUsageRelation(sourceElement, fieldTypeNode, sourceBytes, filePath, gc, "Field Type"))
	}

	return relations, nil
}

func (e *Extractor) createTypeUsageRelation(source *model.CodeElement, targetNode *sitter.Node, sourceBytes *[]byte, filePath string, gc *model.GlobalContext, detail string) *model.DependencyRelation {
	typeName := e.getNodeContent(targetNode, *sourceBytes)
	return &model.DependencyRelation{
		Type:     model.Use,
		Source:   source,
		Target:   &model.CodeElement{Kind: model.Type, Name: typeName, QualifiedName: e.resolveQualifiedName(targetNode, sourceBytes, filePath, gc)},
		Location: e.nodeToLocation(targetNode, filePath),
		Details:  detail,
	}
}

func (e *Extractor) handleActionRelations(q *sitter.Query, match *sitter.QueryMatch, sourceBytes *[]byte, filePath string, gc *model.GlobalContext) ([]*model.DependencyRelation, error) {
	relations := make([]*model.DependencyRelation, 0)

	sourceNode := match.NodesForCaptureIndex(0)[0]
	sourceElement := e.determineSourceElement(&sourceNode, sourceBytes, filePath, gc)
	if sourceElement == nil {
		sourceElement = &model.CodeElement{Kind: model.File, QualifiedName: filePath, Path: filePath}
	}

	if callTarget := e.findCapturedNode(q, match, sourceBytes, "call_target"); callTarget != nil {
		callStmt := callTarget.Parent()
		relations = append(relations, &model.DependencyRelation{
			Type:     model.Call,
			Source:   sourceElement,
			Target:   &model.CodeElement{Kind: model.Method, Name: e.getNodeContent(callTarget, *sourceBytes), QualifiedName: e.resolveQualifiedName(callTarget, sourceBytes, filePath, gc)},
			Location: e.nodeToLocation(callStmt, filePath),
			Details:  "Method Call",
		})
	}

	if createTarget := e.findCapturedNode(q, match, sourceBytes, "create_target_name"); createTarget != nil {
		createStmt := createTarget.Parent()
		relations = append(relations, &model.DependencyRelation{
			Type:     model.Create,
			Source:   sourceElement,
			Target:   &model.CodeElement{Kind: model.Class, Name: e.getNodeContent(createTarget, *sourceBytes), QualifiedName: e.resolveQualifiedName(createTarget, sourceBytes, filePath, gc)},
			Location: e.nodeToLocation(createStmt, filePath),
			Details:  "Object Creation",
		})
	}

	if useFieldName := e.findCapturedNode(q, match, sourceBytes, "use_field_name"); useFieldName != nil {
		useStmt := useFieldName.Parent()
		relations = append(relations, &model.DependencyRelation{
			Type:     model.Use,
			Source:   sourceElement,
			Target:   &model.CodeElement{Kind: model.Field, Name: e.getNodeContent(useFieldName, *sourceBytes), QualifiedName: e.resolveQualifiedName(useFieldName, sourceBytes, filePath, gc)},
			Location: e.nodeToLocation(useStmt, filePath),
			Details:  "Field Access",
		})
	}

	if castType := e.findCapturedNode(q, match, sourceBytes, "cast_type"); castType != nil {
		castStmt := castType.Parent()
		relations = append(relations, &model.DependencyRelation{
			Type:     model.Cast,
			Source:   sourceElement,
			Target:   &model.CodeElement{Kind: model.Type, Name: e.getNodeContent(castType, *sourceBytes), QualifiedName: e.resolveQualifiedName(castType, sourceBytes, filePath, gc)},
			Location: e.nodeToLocation(castStmt, filePath),
			Details:  "Explicit Type Cast",
		})
	}

	if genericID := e.findCapturedNode(q, match, sourceBytes, "use_identifier"); genericID != nil {
		parentType := genericID.Parent().Kind()
		if parentType != "method_invocation" && parentType != "field_access" && genericID.Kind() == "identifier" {
			relations = append(relations, &model.DependencyRelation{
				Type:     model.Use,
				Source:   sourceElement,
				Target:   &model.CodeElement{Kind: model.Unknown, Name: e.getNodeContent(genericID, *sourceBytes), QualifiedName: e.resolveQualifiedName(genericID, sourceBytes, filePath, gc)},
				Location: e.nodeToLocation(genericID, filePath),
				Details:  "Generic Identifier Use",
			})
		}
	}

	return relations, nil
}

func (e *Extractor) processQuery(rootNode *sitter.Node, sourceBytes *[]byte, tsLang *sitter.Language, queryStr string, filePath string, gc *model.GlobalContext, relations *[]*model.DependencyRelation, handler RelationHandler) error {
	formatQueryStr := strings.ReplaceAll(queryStr, "\t", " ")
	formatQueryStr = strings.ReplaceAll(formatQueryStr, "\n", " ")

	q, err := sitter.NewQuery(tsLang, formatQueryStr)
	if err != nil {
		return fmt.Errorf("failed to create query: %w", err)
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, rootNode, *sourceBytes)

	for {
		match := matches.Next()
		if match == nil {
			break
		}
		newRelations, err := handler(q, match, sourceBytes, filePath, gc)
		if err != nil {
			return err
		}
		*relations = append(*relations, newRelations...)
	}
	return nil
}

// --- 私有辅助函数 (带 e 接收者以避免同包名冲突) ---

func (e *Extractor) findCapturedNode(q *sitter.Query, match *sitter.QueryMatch, sourceBytes *[]byte, name string) *sitter.Node {
	index, ok := q.CaptureIndexForName(name)
	if !ok {
		return nil
	}
	nodes := match.NodesForCaptureIndex(index)
	if len(nodes) > 0 {
		return &nodes[0]
	}
	return nil
}

func (e *Extractor) getNodeContent(n *sitter.Node, sourceBytes []byte) string {
	start := n.StartByte()
	end := n.EndByte()
	if int(end) > len(sourceBytes) || start >= end {
		return ""
	}
	return string(sourceBytes[start:end])
}

func (e *Extractor) getDefinitionElement(n *sitter.Node, sourceBytes *[]byte, filePath string) (*model.CodeElement, model.ElementKind) {
	kind := model.Unknown
	nameNode := n.ChildByFieldName("name")
	if nameNode == nil {
		return nil, kind
	}
	name := e.getNodeContent(nameNode, *sourceBytes)
	nodeType := n.Kind()
	switch nodeType {
	case "class_declaration":
		kind = model.Class
	case "interface_declaration":
		kind = model.Interface
	case "method_declaration", "constructor_declaration":
		kind = model.Method
	default:
		return nil, model.Unknown
	}
	return &model.CodeElement{Kind: kind, Name: name, Path: filePath}, kind
}

func (e *Extractor) determineSourceElement(n *sitter.Node, sourceBytes *[]byte, filePath string, gc *model.GlobalContext) *model.CodeElement {
	cursor := n.Walk()
	defer cursor.Close()
	if cursor.GotoParent() {
		for {
			node := cursor.Node()
			nodeType := node.Kind()
			if nodeType == "method_declaration" || nodeType == "constructor_declaration" || nodeType == "class_declaration" || nodeType == "interface_declaration" {
				if elem, _ := e.getDefinitionElement(node, sourceBytes, filePath); elem != nil {
					elem.QualifiedName = e.resolveQualifiedName(node, sourceBytes, filePath, gc)
					return elem
				}
				if nodeType == "class_declaration" || nodeType == "interface_declaration" {
					break
				}
			}
			if !cursor.GotoParent() {
				break
			}
		}
	}
	return &model.CodeElement{Kind: model.File, QualifiedName: filePath, Path: filePath}
}

func (e *Extractor) resolveQualifiedName(n *sitter.Node, sourceBytes *[]byte, filePath string, gc *model.GlobalContext) string {
	name := e.getNodeContent(n, *sourceBytes)
	fc, ok := gc.FileContexts[filePath]
	if !ok {
		return name
	}

	if entries, ok := fc.DefinitionsBySN[name]; ok && len(entries) > 0 {
		return entries[0].Element.QualifiedName
	}
	if fullImport, ok := fc.Imports[name]; ok {
		return fullImport
	}
	if fc.PackageName != "" {
		possibleQN := model.BuildQualifiedName(fc.PackageName, name)
		if definitions := gc.ResolveSymbol(fc, possibleQN); len(definitions) > 0 {
			return possibleQN
		}
	}
	if definitions := gc.ResolveSymbol(fc, name); len(definitions) > 0 {
		return definitions[0].Element.QualifiedName
	}
	return name
}

func (e *Extractor) nodeToLocation(n *sitter.Node, filePath string) *model.Location {
	return &model.Location{
		FilePath:    filePath,
		StartLine:   int(n.StartPosition().Row) + 1,
		EndLine:     int(n.EndPosition().Row) + 1,
		StartColumn: int(n.StartPosition().Column),
		EndColumn:   int(n.EndPosition().Column),
	}
}
