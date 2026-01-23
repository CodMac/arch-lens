package java

import (
	"fmt"
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/core"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Extractor struct{}

func NewJavaExtractor() *Extractor { return &Extractor{} }

// =============================================================================
// 1. 主流水线 (Main Pipeline)
// =============================================================================

func (e *Extractor) Extract(filePath string, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gCtx.FileContexts[filePath]
	if !ok {
		return nil, fmt.Errorf("file context not found: %s", filePath)
	}

	// 1. 静态结构
	hierarchyRels := e.extractHierarchy(fCtx, gCtx)
	structuralRels := e.extractStructural(fCtx, gCtx)

	// 2. 动作发现
	actionRels, err := e.discoverActionRelations(fCtx, gCtx)
	if err != nil {
		return nil, err
	}

	// 3. 统一增强
	enhanceTargets := append(structuralRels, actionRels...)
	for _, rel := range enhanceTargets {
		e.enrichCoreMetadata(rel, fCtx)
	}

	// 4. 合并结果
	var allRels []*model.DependencyRelation
	allRels = append(allRels, hierarchyRels...)
	allRels = append(allRels, structuralRels...)
	allRels = append(allRels, actionRels...)

	return allRels, nil
}

// =============================================================================
// 2. 元数据增强 (Metadata Enrichment)
// =============================================================================

func (e *Extractor) enrichCoreMetadata(rel *model.DependencyRelation, fCtx *core.FileContext) {
	node, _ := rel.Mores["tmp_node"].(*sitter.Node)
	rawText, _ := rel.Mores["tmp_raw"].(string)
	stmt, _ := rel.Mores["tmp_stmt"].(*sitter.Node)

	delete(rel.Mores, "tmp_node")
	delete(rel.Mores, "tmp_raw")
	delete(rel.Mores, "tmp_stmt")

	src := *fCtx.SourceBytes

	switch rel.Type {
	case model.Call:
		e.enrichCallCore(rel, node, stmt, src)
	case model.Create:
		e.enrichCreateCore(rel, node, stmt, src)
	case model.Assign:
		e.enrichAssignCore(rel, node, stmt, src)
	case model.Use:
		e.enrichUseCore(rel, node, src)
	case model.Throw:
		e.enrichThrowCore(rel, node, stmt, rawText, src)
	case model.Parameter:
		e.enrichParameterCore(rel, rawText)
	case model.Return:
		e.enrichReturnCore(rel, rawText)
	case model.Annotation:
		e.enrichAnnotationCore(rel)
	}
}

// =============================================================================
// 3. 核心增强函数 (Enrichment Cores)
// =============================================================================

func (e *Extractor) enrichCallCore(rel *model.DependencyRelation, node *sitter.Node, stmt *sitter.Node, src []byte) {
	rel.Mores[RelCallIsStatic] = false
	rel.Mores[RelCallIsConstructor] = false

	if node == nil {
		return
	}

	// 补全方法名括号，使其符合 collector 规范
	if rel.Target != nil && rel.Target.Kind == model.Method && !strings.HasSuffix(rel.Target.QualifiedName, ")") {
		rel.Target.QualifiedName += "()"
	}

	// 定位调用的真实 AST 容器节点
	callNode := e.findNearestKind(node, "method_invocation", "method_reference", "explicit_constructor_invocation", "object_creation_expression")
	if callNode == nil {
		return
	}
	rel.Mores[RelAstKind] = callNode.Kind()

	switch callNode.Kind() {
	case "method_invocation":
		if objectNode := callNode.ChildByFieldName("object"); objectNode != nil {
			receiverText := objectNode.Utf8Text(src)
			rel.Mores[RelCallReceiver] = receiverText

			// 【核心修复】判定静态调用，必须排除 getList() 这种带括号的 receiver
			isStatic := e.isPotentialClassName(receiverText)
			rel.Mores[RelCallIsStatic] = isStatic
			if isStatic {
				rel.Mores[RelCallReceiverType] = receiverText
			}

			// 识别链式调用
			if objectNode.Kind() == "method_invocation" || objectNode.Kind() == "object_creation_expression" {
				rel.Mores[RelCallIsChained] = true
			}
		} else {
			rel.Mores[RelCallReceiver] = "this"
			rel.Mores[RelCallIsStatic] = false
		}

	case "object_creation_expression":
		rel.Mores[RelCallIsConstructor] = true
		if typeNode := callNode.ChildByFieldName("type"); typeNode != nil {
			rel.Mores[RelCallReceiverType] = typeNode.Utf8Text(src)
		}

	case "method_reference":
		rel.Mores[RelCallIsFunctional] = true
		if objectNode := callNode.ChildByFieldName("object"); objectNode != nil {
			receiverText := objectNode.Utf8Text(src)
			rel.Mores[RelCallReceiver] = receiverText
			if e.isPotentialClassName(receiverText) {
				rel.Mores[RelCallIsStatic] = true
			}
		}

	case "explicit_constructor_invocation":
		rel.Mores[RelCallIsConstructor] = true
		if callNode.ChildCount() > 0 {
			rel.Mores[RelCallReceiver] = callNode.Child(0).Utf8Text(src)
		}
	}

	// EnclosingMethod 溯源 (Lambda/匿名类溯源到所属方法)
	if rel.Source != nil {
		qn := rel.Source.QualifiedName
		stopMarkers := []string{".lambda", ".anonymousClass", "$", ".block"}
		for _, marker := range stopMarkers {
			if idx := strings.Index(qn, marker); idx != -1 {
				rel.Mores[RelCallEnclosingMethod] = qn[:idx]
				break
			}
		}
	}
}

func (e *Extractor) enrichCreateCore(rel *model.DependencyRelation, node, stmt *sitter.Node, src []byte) {
	rel.Mores[RelAstKind] = "object_creation_expression"
	if stmt != nil && strings.Contains(stmt.Utf8Text(src), "[") {
		rel.Mores[RelCreateIsArray] = true
	}
}

func (e *Extractor) enrichAssignCore(rel *model.DependencyRelation, capNode, stmtNode *sitter.Node, src []byte) {
	if stmtNode == nil {
		return
	}
	rel.Mores[RelAstKind] = stmtNode.Kind()
	// 确保 TargetName 只是标识符（如 count），而不是 count += 5
	rel.Mores[RelAssignTargetName] = capNode.Utf8Text(src)

	switch stmtNode.Kind() {
	case "variable_declarator":
		rel.Mores[RelAssignIsInitializer] = true
		rel.Mores[RelAssignOperator] = "="
		for i := 0; i < int(stmtNode.ChildCount()); i++ {
			if stmtNode.FieldNameForChild(uint32(i)) == "value" {
				rel.Mores[RelAssignValueExpression] = stmtNode.Child(uint(i)).Utf8Text(src)
				break
			}
		}
	case "assignment_expression":
		for i := 0; i < int(stmtNode.ChildCount()); i++ {
			child := stmtNode.Child(uint(i))
			fieldName := stmtNode.FieldNameForChild(uint32(i))
			if fieldName == "operator" {
				rel.Mores[RelAssignOperator] = child.Utf8Text(src)
			} else if fieldName == "right" {
				rel.Mores[RelAssignValueExpression] = child.Utf8Text(src)
			}
		}
	case "update_expression":
		raw := stmtNode.Utf8Text(src)
		if strings.Contains(raw, "++") {
			rel.Mores[RelAssignOperator] = "++"
		} else {
			rel.Mores[RelAssignOperator] = "--"
		}
	}
}

func (e *Extractor) enrichThrowCore(rel *model.DependencyRelation, node, stmt *sitter.Node, rawText string, src []byte) {
	if node != nil {
		rel.Mores[RelAstKind] = "throw_statement"
		rel.Target.Name = e.clean(rel.Target.Name)
		rel.Target.QualifiedName = e.clean(rel.Target.QualifiedName)
		if node.Kind() == "type_identifier" || (node.Parent() != nil && node.Parent().Kind() == "object_creation_expression") {
			rel.Mores[RelThrowIsRuntime] = true
		} else if node.Kind() == "identifier" {
			rel.Mores[RelThrowIsRethrow] = true
		}
		return
	}
	if rawText != "" && rel.Source != nil && rel.Source.Extra != nil {
		if ths, ok := rel.Source.Extra.Mores[MethodThrowsTypes].([]string); ok {
			for i, ex := range ths {
				if e.clean(ex) == rel.Target.Name {
					rel.Mores[RelThrowIndex] = i
					rel.Mores[RelThrowIsSignature] = true
					break
				}
			}
		}
	}
}

func (e *Extractor) enrichParameterCore(rel *model.DependencyRelation, rawText string) {
	if params, ok := rel.Source.Extra.Mores[MethodParameters].([]string); ok {
		for i, p := range params {
			if strings.Contains(p, rel.Target.Name) || strings.Contains(p, rawText) {
				rel.Mores[RelParameterIndex] = i
				parts := strings.Fields(p)
				if len(parts) >= 2 {
					rel.Mores[RelParameterName] = parts[len(parts)-1]
				}
				if strings.Contains(p, "...") {
					rel.Mores[RelParameterIsVarargs] = true
				}
			}
		}
	}
}

func (e *Extractor) enrichReturnCore(rel *model.DependencyRelation, rawText string) {
	rel.Mores[RelReturnIsPrimitive] = e.isPrimitive(e.clean(rawText))
	rel.Mores[RelReturnIsArray] = strings.Contains(rawText, "[]")
}

func (e *Extractor) enrichAnnotationCore(rel *model.DependencyRelation) {
	target := e.mapElementKindToAnnotationTarget(rel.Source)
	rel.Mores[RelAnnotationTarget] = target
	rel.Target.Name = strings.Split(rel.Target.Name, "(")[0]
	rel.Target.QualifiedName = strings.Split(rel.Target.QualifiedName, "(")[0]
}

func (e *Extractor) enrichUseCore(rel *model.DependencyRelation, node *sitter.Node, src []byte) {
	if node != nil {
		rel.Mores[RelAstKind] = "identifier"
	}
}

// =============================================================================
// 4. 发现逻辑 (Discovery Logic)
// =============================================================================

func (e *Extractor) extractStructural(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			elem := entry.Element
			if elem.Extra == nil {
				continue
			}

			if sc, ok := elem.Extra.Mores[ClassSuperClass].(string); ok && sc != "" {
				rels = append(rels, &model.DependencyRelation{
					Type: model.Extend, Source: elem, Target: e.quickResolve(e.clean(sc), model.Class, gCtx, fCtx),
				})
			}
			for _, anno := range elem.Extra.Annotations {
				cleanName := e.clean(anno)
				rels = append(rels, &model.DependencyRelation{
					Type: model.Annotation, Source: elem, Target: e.quickResolve(cleanName, model.KAnnotation, gCtx, fCtx),
					Mores: map[string]interface{}{RelRawText: anno},
				})
			}
			if elem.Kind == model.Method {
				if pts, ok := elem.Extra.Mores[MethodParameters].([]string); ok {
					for _, p := range pts {
						typePart := e.extractTypeFromParam(p)
						rels = append(rels, &model.DependencyRelation{
							Type: model.Parameter, Source: elem, Target: e.quickResolve(e.clean(typePart), model.Class, gCtx, fCtx),
							Mores: map[string]interface{}{"tmp_raw": p},
						})
					}
				}
				if rt, ok := elem.Extra.Mores[MethodReturnType].(string); ok && rt != "void" && rt != "" {
					rels = append(rels, &model.DependencyRelation{
						Type: model.Return, Source: elem, Target: e.quickResolve(e.clean(rt), model.Class, gCtx, fCtx),
						Mores: map[string]interface{}{"tmp_raw": rt},
					})
				}
				if ths, ok := elem.Extra.Mores[MethodThrowsTypes].([]string); ok {
					for _, ex := range ths {
						rels = append(rels, &model.DependencyRelation{
							Type: model.Throw, Source: elem, Target: e.quickResolve(e.clean(ex), model.Class, gCtx, fCtx),
							Mores: map[string]interface{}{"tmp_raw": ex},
						})
					}
				}
			}
			for _, rt := range e.getRawTypesForTypeArgs(elem) {
				rels = append(rels, e.collectAllTypeArgs(rt, elem, gCtx, fCtx)...)
			}
		}
	}
	return rels
}

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
		capturedNode := &match.Captures[0].Node
		sourceElem := e.determinePreciseSource(capturedNode, fCtx, gCtx)
		if sourceElem == nil {
			continue
		}

		for _, cap := range match.Captures {
			capName := q.CaptureNames()[cap.Index]
			if !strings.HasSuffix(capName, "_target") && capName != "update_stmt" && capName != "explicit_constructor_stmt" {
				continue
			}

			// 治理点：mapAction 返回 ActionTarget 数组，区分 TargetNode 和 ContextNode
			actionTargets := e.mapAction(capName, &cap.Node)
			for _, at := range actionTargets {
				if at.RelType == "" {
					continue
				}

				// TargetText 用于 resolve (保证纯净)
				targetText := at.TargetNode.Utf8Text(*fCtx.SourceBytes)
				// ContextNode 用于提取 RawText (保证完整)
				ctxNode := at.ContextNode
				if ctxNode == nil {
					ctxNode = at.TargetNode
				}

				rels = append(rels, &model.DependencyRelation{
					Type:     at.RelType,
					Source:   sourceElem,
					Target:   e.quickResolve(e.clean(targetText), at.Kind, gCtx, fCtx),
					Location: e.toLoc(*at.TargetNode, fCtx.FilePath),
					Mores: map[string]interface{}{
						RelRawText: ctxNode.Utf8Text(*fCtx.SourceBytes),
						"tmp_node": at.TargetNode,
						"tmp_stmt": ctxNode,
					},
				})
			}
		}
	}
	return rels, nil
}

// =============================================================================
// 5. 辅助工具 (Helper Utilities)
// =============================================================================

type ActionTarget struct {
	RelType     model.DependencyType
	Kind        model.ElementKind
	TargetNode  *sitter.Node // 真正被引用的标识符
	ContextNode *sitter.Node // 所在的完整语句/表达式
}

func (e *Extractor) mapAction(capName string, node *sitter.Node) []ActionTarget {
	switch capName {
	case "call_target", "ref_target":
		ctx := e.findNearestKind(node, "method_invocation", "method_reference", "explicit_constructor_invocation", "object_creation_expression")
		return []ActionTarget{{model.Call, model.Method, node, ctx}}
	case "create_target":
		ctx := e.findNearestKind(node, "object_creation_expression")
		return []ActionTarget{
			{model.Create, model.Class, node, ctx},
			{model.Call, model.Method, node, ctx},
		}
	case "assign_target", "update_stmt":
		ctx := e.findNearestKind(node, "assignment_expression", "variable_declarator", "update_expression")
		return []ActionTarget{{model.Assign, model.Variable, node, ctx}}
	case "use_field_target":
		return []ActionTarget{{model.Use, model.Field, node, node}}
	case "throw_target":
		return []ActionTarget{{model.Throw, model.Class, node, e.findThrowStatement(node)}}
	case "explicit_constructor_stmt":
		return []ActionTarget{{model.Call, model.Method, node, node}}
	default:
		return nil
	}
}

func (e *Extractor) clean(s string) string {
	s = strings.TrimPrefix(s, "@")
	s = strings.TrimPrefix(s, "new ")
	if strings.Contains(s, "extends ") {
		s = strings.Split(s, "extends ")[1]
	}
	if strings.Contains(s, "super ") {
		s = strings.Split(s, "super ")[1]
	}
	s = strings.Split(s, "<")[0]
	s = strings.Split(s, "(")[0]
	s = strings.TrimSuffix(s, "...")
	return strings.TrimSpace(strings.TrimRight(s, "> ,[]"))
}

func (e *Extractor) isPotentialClassName(s string) bool {
	if s == "" || s == "this" || s == "super" {
		return false
	}
	// 如果包含括号，通常是方法返回的对象，不是类名
	if strings.Contains(s, "(") {
		return false
	}
	// 处理 com.example.Config 这种情况，取最后一部分
	parts := strings.Split(s, ".")
	last := parts[len(parts)-1]
	if len(last) > 0 && last[0] >= 'A' && last[0] <= 'Z' {
		return true
	}
	return false
}

func (e *Extractor) extractTypeFromParam(p string) string {
	parts := strings.Fields(p)
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return p
}

func (e *Extractor) getRawTypesForTypeArgs(elem *model.CodeElement) (res []string) {
	keys := []string{FieldType, VariableType, MethodReturnType}
	for _, k := range keys {
		if v, ok := elem.Extra.Mores[k].(string); ok {
			res = append(res, v)
		}
	}
	if pts, ok := elem.Extra.Mores[MethodParameters].([]string); ok {
		for _, p := range pts {
			res = append(res, e.extractTypeFromParam(p))
		}
	}
	return
}

func (e *Extractor) determinePreciseSource(n *sitter.Node, fCtx *core.FileContext, gCtx *core.GlobalContext) *model.CodeElement {
	for curr := n.Parent(); curr != nil; curr = curr.Parent() {
		line := int(curr.StartPosition().Row) + 1
		var k model.ElementKind
		switch curr.Kind() {
		case "method_declaration", "constructor_declaration":
			k = model.Method
		case "static_initializer":
			k = model.ScopeBlock
		case "lambda_expression":
			k = model.Lambda
		case "field_declaration":
			k = model.Field
		case "variable_declarator":
			if p := curr.Parent(); p != nil && p.Kind() == "field_declaration" {
				k = model.Field
			} else {
				continue
			}
		case "class_body", "interface_body", "program":
			return nil
		default:
			continue
		}
		for _, entries := range fCtx.DefinitionsBySN {
			for _, entry := range entries {
				if entry.Element.Kind == k && entry.Element.Location.StartLine == line {
					return entry.Element
				}
			}
		}
	}
	return nil
}

func (e *Extractor) findThrowStatement(n *sitter.Node) *sitter.Node {
	for curr := n; curr != nil; curr = curr.Parent() {
		if curr.Kind() == "throw_statement" {
			return curr
		}
		if curr.Kind() == "method_declaration" || curr.Kind() == "class_body" {
			break
		}
	}
	return nil
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

func (e *Extractor) quickResolve(symbol string, kind model.ElementKind, gCtx *core.GlobalContext, fCtx *core.FileContext) *model.CodeElement {
	if entries := gCtx.ResolveSymbol(fCtx, symbol); len(entries) > 0 {
		return entries[0].Element
	}
	return &model.CodeElement{Name: symbol, QualifiedName: symbol, Kind: kind}
}

func (e *Extractor) toLoc(n sitter.Node, path string) *model.Location {
	return &model.Location{
		FilePath: path, StartLine: int(n.StartPosition().Row) + 1, EndLine: int(n.EndPosition().Row) + 1,
		StartColumn: int(n.StartPosition().Column), EndColumn: int(n.EndPosition().Column),
	}
}

func (e *Extractor) isPrimitive(typeName string) bool {
	switch typeName {
	case "int", "long", "short", "byte", "char", "boolean", "float", "double":
		return true
	}
	return false
}

func (e *Extractor) mapElementKindToAnnotationTarget(elem *model.CodeElement) string {
	switch elem.Kind {
	case model.Class, model.Interface, model.Enum:
		return "TYPE"
	case model.Field:
		return "FIELD"
	case model.Method:
		return "METHOD"
	case model.Variable:
		if isParam, _ := elem.Extra.Mores["java.variable.is_param"].(bool); isParam {
			return "PARAMETER"
		}
		return "LOCAL_VARIABLE"
	}
	return "UNKNOWN"
}

func (e *Extractor) extractHierarchy(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	fileSource := gCtx.DefinitionsByQN[fCtx.FilePath][0].Element
	for _, imports := range fCtx.Imports {
		for _, imp := range imports {
			rels = append(rels, &model.DependencyRelation{
				Type: model.Import, Source: fileSource, Target: e.quickResolve(imp.RawImportPath, imp.Kind, gCtx, fCtx), Location: imp.Location,
			})
		}
	}
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			if entry.ParentQN != "" {
				if parents := gCtx.DefinitionsByQN[entry.ParentQN]; len(parents) > 0 {
					rels = append(rels, &model.DependencyRelation{Type: model.Contain, Source: parents[0].Element, Target: entry.Element})
				}
			}
		}
	}
	return rels
}

func (e *Extractor) parseTypeArgs(rawType string) []string {
	start, end := strings.Index(rawType, "<"), strings.LastIndex(rawType, ">")
	if start == -1 || end == -1 || start >= end {
		return nil
	}
	content := rawType[start+1 : end]
	var args []string
	bracketLevel := 0
	current := strings.Builder{}
	for _, r := range content {
		switch r {
		case '<':
			bracketLevel++
			current.WriteRune(r)
		case '>':
			bracketLevel--
			current.WriteRune(r)
		case ',':
			if bracketLevel == 0 {
				args = append(args, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, strings.TrimSpace(current.String()))
	}
	return args
}

func (e *Extractor) collectAllTypeArgs(rt string, source *model.CodeElement, gCtx *core.GlobalContext, fCtx *core.FileContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	if !strings.Contains(rt, "<") {
		return nil
	}
	args := e.parseTypeArgs(rt)
	for i, arg := range args {
		rels = append(rels, &model.DependencyRelation{
			Type: model.TypeArg, Source: source, Target: e.quickResolve(e.clean(arg), model.Class, gCtx, fCtx),
			Mores: map[string]interface{}{RelTypeArgIndex: i, RelRawText: arg, RelAstKind: "type_arguments"},
		})
		if strings.Contains(arg, "<") {
			rels = append(rels, e.collectAllTypeArgs(arg, source, gCtx, fCtx)...)
		}
	}
	return rels
}
