package java

import (
	"fmt"
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/core"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Extractor 负责 Java 源代码的依赖关系提取。
type Extractor struct{}

// NewJavaExtractor 创建一个新的 Java 提取器实例。
func NewJavaExtractor() *Extractor {
	return &Extractor{}
}

// =============================================================================
// 1. 主入口流水线
// =============================================================================

// Extract 是提取器的核心入口，协调并整合三类依赖关系的提取。
func (e *Extractor) Extract(filePath string, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gCtx.FileContexts[filePath]
	if !ok {
		return nil, fmt.Errorf("file context not found: %s", filePath)
	}

	var allRelations []*model.DependencyRelation

	// 阶段 1: 提取静态层级与结构关系 (IMPORT, CONTAIN, EXTEND, IMPLEMENT等)
	allRelations = append(allRelations, e.extractHierarchy(fCtx, gCtx)...)
	allRelations = append(allRelations, e.extractStructural(fCtx, gCtx)...)

	// 阶段 2: 动作关系发现 (Discovery) - 识别 CALL, CREATE, ASSIGN, USE 等
	actionRels, err := e.discoverActionRelations(fCtx, gCtx)
	if err != nil {
		return nil, err
	}

	// 阶段 3: 元数据增强 (Enrichment) - 遍历发现的关系，填充详细的 Mores 常量
	for _, rel := range actionRels {
		e.enrichRelation(rel, fCtx, gCtx)
	}

	allRelations = append(allRelations, actionRels...)

	return allRelations, nil
}

// =============================================================================
// 2. 动作关系发现 (Discovery Phase)
// =============================================================================

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
  
  ; --- 显式构造函数调用 (super/this) ---
  (explicit_constructor_invocation [
      (this) @call_target
      (super) @call_target
  ]) @explicit_constructor_stmt
  
  ; --- 赋值与更新 (ASSIGN) ---
  (assignment_expression 
      left: [
          (identifier) @assign_target
          (field_access field: (identifier) @assign_target)
          (array_access array: (identifier) @assign_target)
      ]
  ) @assign_stmt
  
  (variable_declarator 
      name: (identifier) @assign_target 
      value: (_) @assign_value
  ) @assign_init_stmt
  
  (update_expression
      [
          (identifier) @assign_target
          (field_access field: (identifier) @assign_target)
      ]
  ) @assign_update_stmt
]
`

func (e *Extractor) discoverActionRelations(fCtx *core.FileContext, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
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

		// 约定：JavaActionQuery 中每个模式的第一个捕获点是语句/表达式容器 (@xxx_stmt)
		stmtNode := &match.Captures[0].Node
		sourceElem := e.determinePreciseSource(stmtNode, fCtx, gCtx)
		if sourceElem == nil {
			continue
		}

		for _, cap := range match.Captures {
			capName := q.CaptureNames()[cap.Index]
			// 我们只处理以 _target 结尾的捕获名
			if !strings.HasSuffix(capName, "_target") {
				continue
			}

			relType, kind := e.mapAction(capName)
			if relType == "" {
				continue
			}

			rawText := cap.Node.Utf8Text(*fCtx.SourceBytes)
			target := e.quickResolve(e.clean(rawText), kind, gCtx, fCtx)

			rel := &model.DependencyRelation{
				Type:     relType,
				Source:   sourceElem,
				Target:   target,
				Location: e.toLoc(cap.Node, fCtx.FilePath),
				Mores: map[string]interface{}{
					RelRawText: stmtNode.Utf8Text(*fCtx.SourceBytes),
					RelContext: stmtNode.Kind(),
					// 临时存储 AST 节点信息，仅供 Enrichment 阶段使用，后续会清理
					"tmp_ast_node":  &cap.Node,
					"tmp_stmt_node": stmtNode,
				},
			}
			rels = append(rels, rel)
		}
	}
	return rels, nil
}

// =============================================================================
// 3. 元数据增强 (Enrichment Phase)
// =============================================================================

func (e *Extractor) enrichRelation(rel *model.DependencyRelation, fCtx *core.FileContext, gCtx *core.GlobalContext) {
	// 提取暂存的 AST 节点
	node, _ := rel.Mores["tmp_ast_node"].(*sitter.Node)
	stmt, _ := rel.Mores["tmp_stmt_node"].(*sitter.Node)
	// 清理临时节点，避免污染最终的 model
	delete(rel.Mores, "tmp_ast_node")
	delete(rel.Mores, "tmp_stmt_node")

	if node == nil || stmt == nil {
		return
	}

	switch rel.Type {
	case model.Call:
		e.enrichCallMetadata(rel, node, fCtx)
	case model.Create:
		e.enrichCreateMetadata(rel, node, stmt, fCtx)
	case model.Assign:
		e.enrichAssignMetadata(rel, node, stmt, fCtx)
	case model.Use:
		e.enrichUseMetadata(rel, node, fCtx)
	}
}

// --- CALL 增强 ---
func (e *Extractor) enrichCallMetadata(rel *model.DependencyRelation, capNode *sitter.Node, fCtx *core.FileContext) {
	source := *fCtx.SourceBytes
	callNode := e.findNearestCallContainer(capNode)
	if callNode == nil {
		return
	}

	kind := callNode.Kind()
	rel.Mores[RelAstKind] = kind

	switch kind {
	case "method_invocation":
		if objNode := callNode.ChildByFieldName("object"); objNode != nil {
			recText := objNode.Utf8Text(source)
			rel.Mores[RelCallReceiver] = recText

			// 识别静态调用与类型
			if len(recText) > 0 && recText[0] >= 'A' && recText[0] <= 'Z' && !strings.Contains(recText, "(") {
				rel.Mores[RelCallIsStatic] = true
				rel.Mores[RelCallReceiverType] = recText
			}

			// 识别链式调用 (Case 4)
			if objNode.Kind() == "method_invocation" {
				rel.Mores[RelCallIsChained] = true
				rel.Mores[RelCallReceiverExpression] = recText
			}
		} else {
			rel.Mores[RelCallReceiver] = "this"
		}

		// 识别继承调用 (Case 5 & 6)
		if rel.Mores[RelCallReceiver] == "super" {
			rel.Mores[RelCallIsInherited] = true
		}

	case "method_reference":
		rel.Mores[RelCallIsFunctional] = true
		if objNode := callNode.ChildByFieldName("object"); objNode != nil {
			rel.Mores[RelCallReceiver] = objNode.Utf8Text(source)
		}

	case "explicit_constructor_invocation":
		rel.Mores[RelCallIsConstructor] = true
		rel.Mores[RelCallReceiver] = capNode.Utf8Text(source) // super 或 this
	}

	// 环境识别 (Enclosing Method)
	if enclosing := e.findParentKind(callNode, "method_declaration"); enclosing != nil {
		if nameNode := enclosing.ChildByFieldName("name"); nameNode != nil {
			rel.Mores[RelCallEnclosingMethod] = nameNode.Utf8Text(source) + "()"
		}
	}
}

// --- CREATE 增强 ---
func (e *Extractor) enrichCreateMetadata(rel *model.DependencyRelation, capNode, stmtNode *sitter.Node, fCtx *core.FileContext) {
	rel.Mores[RelAstKind] = "object_creation_expression"
	rel.Mores[RelCallIsConstructor] = true
	source := *fCtx.SourceBytes

	// 向上寻找真实的 object_creation_expression 节点
	createNode := e.findNearestKind(capNode, "object_creation_expression")
	if createNode == nil {
		return
	}

	// 处理泛型
	if typeNode := createNode.ChildByFieldName("type"); typeNode != nil {
		if typeNode.Kind() == "generic_type" {
			argNode := typeNode.ChildByFieldName("arguments")
			if argNode != nil && argNode.NamedChildCount() > 0 {
				rel.Mores[RelCallTypeArguments] = strings.Trim(argNode.Utf8Text(source), "<>")
			} else {
				// 菱形语法 <> 情况：向上寻找变量声明类型
				rel.Mores[RelCallTypeArguments] = e.inferDiamondType(stmtNode, source)
			}
		}
	}
}

// --- ASSIGN 增强 ---
func (e *Extractor) enrichAssignMetadata(rel *model.DependencyRelation, capNode, stmtNode *sitter.Node, fCtx *core.FileContext) {
	source := *fCtx.SourceBytes
	rel.Mores[RelAssignTargetName] = capNode.Utf8Text(source)

	// 查找具体的赋值容器
	assignNode := e.findNearestKind(capNode, "assignment_expression", "variable_declarator", "update_expression")
	if assignNode == nil {
		return
	}

	rel.Mores[RelAstKind] = assignNode.Kind()

	switch assignNode.Kind() {
	case "assignment_expression":
		if opNode := assignNode.ChildByFieldName("operator"); opNode != nil {
			op := opNode.Utf8Text(source)
			rel.Mores[RelAssignOperator] = op
			rel.Mores[RelAssignIsCompound] = (op != "=")
		}
		if right := assignNode.ChildByFieldName("right"); right != nil {
			rel.Mores[RelAssignValueExpression] = right.Utf8Text(source)
			e.analyzeDataFlow(rel, right, fCtx)
		}
	case "variable_declarator":
		rel.Mores[RelAssignIsInitializer] = true
		rel.Mores[RelAssignOperator] = "="
		if val := assignNode.ChildByFieldName("value"); val != nil {
			rel.Mores[RelAssignValueExpression] = val.Utf8Text(source)
			e.analyzeDataFlow(rel, val, fCtx)
		}
	case "update_expression":
		e.enrichUpdateMetadata(rel, assignNode, fCtx)
	}
}

// --- USE 增强 ---
func (e *Extractor) enrichUseMetadata(rel *model.DependencyRelation, capNode *sitter.Node, fCtx *core.FileContext) {
	// TODO: 后续路标实现具体的 Usage Role (Argument, Operand 等)
}

// =============================================================================
// 4. 辅助解析工具
// =============================================================================

func (e *Extractor) inferDiamondType(stmtNode *sitter.Node, source []byte) string {
	// 针对 List<String> list = new ArrayList<>() 结构
	if stmtNode.Kind() == "local_variable_declaration" || stmtNode.Kind() == "field_declaration" {
		for i := 0; i < int(stmtNode.NamedChildCount()); i++ {
			child := stmtNode.NamedChild(uint(i))
			if child.Kind() == "generic_type" {
				return e.extractTypeArgsFromNode(child, source)
			}
		}
	}
	return ""
}

func (e *Extractor) extractTypeArgsFromNode(n *sitter.Node, source []byte) string {
	if n != nil && n.Kind() == "generic_type" {
		argNode := n.ChildByFieldName("arguments")
		if argNode != nil {
			return strings.Trim(argNode.Utf8Text(source), "<>")
		}
	}
	return ""
}

func (e *Extractor) findNearestCallContainer(n *sitter.Node) *sitter.Node {
	return e.findNearestKind(n, "method_invocation", "method_reference", "object_creation_expression", "explicit_constructor_invocation")
}

func (e *Extractor) findNearestKind(n *sitter.Node, kinds ...string) *sitter.Node {
	for curr := n; curr != nil; curr = curr.Parent() {
		for _, k := range kinds {
			if curr.Kind() == k {
				return curr
			}
		}
		if strings.HasSuffix(curr.Kind(), "_statement") || curr.Kind() == "class_body" {
			break
		}
	}
	return nil
}

func (e *Extractor) findParentKind(n *sitter.Node, kind string) *sitter.Node {
	for curr := n.Parent(); curr != nil; curr = curr.Parent() {
		if curr.Kind() == kind {
			return curr
		}
	}
	return nil
}

func (e *Extractor) analyzeDataFlow(rel *model.DependencyRelation, node *sitter.Node, fCtx *core.FileContext) {
	kind := node.Kind()
	if strings.Contains(kind, "literal") || kind == "null_literal" {
		rel.Mores[RelAssignIsConstant] = true
	}
	if kind == "method_invocation" {
		rel.Mores[RelAssignIsReturnValue] = true
	}
	if kind == "cast_expression" {
		rel.Mores[RelAssignIsCastCheck] = true
		if val := node.ChildByFieldName("value"); val != nil {
			rel.Mores[RelAssignValueExpression] = val.Utf8Text(*fCtx.SourceBytes)
		}
	}
}

func (e *Extractor) determinePreciseSource(n *sitter.Node, fCtx *core.FileContext, gCtx *core.GlobalContext) *model.CodeElement {
	for curr := n.Parent(); curr != nil; curr = curr.Parent() {
		k := curr.Kind()
		line := int(curr.StartPosition().Row) + 1
		var targetKind model.ElementKind

		switch k {
		case "method_declaration", "constructor_declaration":
			targetKind = model.Method
		case "lambda_expression":
			targetKind = model.Lambda
		case "static_initializer":
			targetKind = model.ScopeBlock
		case "field_declaration":
			targetKind = model.Field
		case "variable_declarator":
			if curr.Parent() != nil && curr.Parent().Kind() == "field_declaration" {
				targetKind = model.Field
			} else {
				continue
			}
		default:
			continue
		}

		for _, entries := range fCtx.DefinitionsBySN {
			for _, entry := range entries {
				if entry.Element.Kind == targetKind && entry.Element.Location.StartLine == line {
					return entry.Element
				}
			}
		}
	}
	return nil
}

// =============================================================================
// 5. 静态关系提取 (Hierarchy & Structural)
// =============================================================================

func (e *Extractor) extractHierarchy(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	fileElems, ok := gCtx.DefinitionsByQN[fCtx.FilePath]
	if !ok || len(fileElems) == 0 {
		return nil
	}
	fileSource := fileElems[0].Element

	for _, imports := range fCtx.Imports {
		for _, imp := range imports {
			target := e.quickResolve(imp.RawImportPath, imp.Kind, gCtx, fCtx)
			rels = append(rels, &model.DependencyRelation{
				Type: model.Import, Source: fileSource, Target: target, Location: imp.Location,
			})
		}
	}

	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			if entry.ParentQN != "" {
				if parents, ok := gCtx.DefinitionsByQN[entry.ParentQN]; ok && len(parents) > 0 {
					for _, parent := range parents {
						rels = append(rels, &model.DependencyRelation{
							Type: model.Contain, Source: parent.Element, Target: entry.Element,
						})
					}
				}
			}
		}
	}
	return rels
}

func (e *Extractor) extractStructural(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			elem := entry.Element
			if elem.Extra == nil || elem.Extra.Mores == nil {
				continue
			}
			mores := elem.Extra.Mores

			// 注解
			for _, annoStr := range elem.Extra.Annotations {
				name := strings.TrimPrefix(strings.Split(annoStr, "(")[0], "@")
				rels = append(rels, &model.DependencyRelation{
					Type: model.Annotation, Source: elem, Target: e.quickResolve(name, model.KAnnotation, gCtx, fCtx),
					Mores: map[string]interface{}{RelRawText: annoStr},
				})
			}

			// 继承/实现
			if sc, ok := mores[ClassSuperClass].(string); ok && sc != "" {
				rels = append(rels, &model.DependencyRelation{
					Type: model.Extend, Source: elem, Target: e.quickResolve(e.clean(sc), model.Class, gCtx, fCtx),
				})
			}
		}
	}
	return rels
}

// --- 基础工具函数 ---

func (e *Extractor) mapAction(capName string) (model.DependencyType, model.ElementKind) {
	switch capName {
	case "call_target", "ref_target", "explicit_constructor_stmt":
		return model.Call, model.Method
	case "create_target":
		return model.Create, model.Class
	case "assign_target":
		return model.Assign, model.Variable
	case "use_field_target":
		return model.Use, model.Field
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
	if entries := gCtx.ResolveSymbol(fCtx, symbol); len(entries) > 0 {
		return entries[0].Element
	}
	return &model.CodeElement{Name: symbol, QualifiedName: symbol, Kind: kind}
}

func (e *Extractor) enrichUpdateMetadata(rel *model.DependencyRelation, node *sitter.Node, fCtx *core.FileContext) {
	raw := node.Utf8Text(*fCtx.SourceBytes)
	if strings.Contains(raw, "++") {
		rel.Mores[RelAssignOperator] = "++"
	} else {
		rel.Mores[RelAssignOperator] = "--"
	}
	rel.Mores[RelAssignIsPostfix] = strings.HasSuffix(raw, "++") || strings.HasSuffix(raw, "--")
}
