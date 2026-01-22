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

	var allRels []*model.DependencyRelation

	// 阶段 1: 静态结构与层级
	allRels = append(allRels, e.extractHierarchy(fCtx, gCtx)...)
	structuralRels := e.extractStructural(fCtx, gCtx)

	// 阶段 2: 动作发现
	actionRels, err := e.discoverActionRelations(fCtx, gCtx)
	if err != nil {
		return nil, err
	}

	// 阶段 3: 统一元数据增强
	enhanceTargets := append(structuralRels, actionRels...)
	for _, rel := range enhanceTargets {
		e.enrichCoreMetadata(rel, fCtx)
	}

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
		e.enrichCallCore(rel, node, src, fCtx)
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

func (e *Extractor) enrichCallCore(rel *model.DependencyRelation, node *sitter.Node, src []byte, fCtx *core.FileContext) {
	if node == nil {
		return
	}

	// 只有当目标是 Method 且名称还不带括号时才补全
	if rel.Target != nil && rel.Target.Kind == model.Method && !strings.Contains(rel.Target.Name, "(") {
		// 目前统一补全为 ()，因为精确参数推断需要后续的 Lambda/Type 推断支持
		rel.Target.QualifiedName += "()"
	}

	// 1. 定位真实的调用节点
	callNode := e.findNearestKind(node, "method_invocation", "method_reference", "explicit_constructor_invocation", "object_creation_expression")
	if callNode == nil {
		return
	}

	rel.Mores[RelAstKind] = callNode.Kind()

	// 2. 核心逻辑：区分调用类型并提取 Receiver/Static/Constructor 属性
	switch callNode.Kind() {
	case "method_invocation":
		// 查找执行对象 (e.g., "System" in "System.out.println")
		if objectNode := callNode.ChildByFieldName("object"); objectNode != nil {
			receiverText := objectNode.Utf8Text(src)
			rel.Mores[RelCallReceiver] = receiverText

			// 静态调用判断逻辑 (非精准，但在 collector 补全前作为关键参考)
			// 如果 receiver 是首字母大写的（如 System），则标记为 ReceiverType 并设为 Static
			if e.isPotentialClassName(receiverText) {
				rel.Mores[RelCallIsStatic] = true
				rel.Mores[RelCallReceiverType] = receiverText
			} else {
				rel.Mores[RelCallIsStatic] = false
			}

			// 链式调用判断：如果 object 节点本身又是另一种调用
			if objectNode.Kind() == "method_invocation" || objectNode.Kind() == "object_creation_expression" {
				rel.Mores[RelCallIsChained] = true
			}
		} else {
			// 无 object，隐式 this 调用
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
			rel.Mores[RelCallReceiver] = objectNode.Utf8Text(src)
		}

	case "explicit_constructor_invocation":
		// 处理 super() 或 this()
		rel.Mores[RelCallIsConstructor] = true
		if callNode.ChildCount() > 0 {
			rel.Mores[RelCallReceiver] = callNode.Child(0).Utf8Text(src) // "super" 或 "this"
		}
	}

	// 3. 改进 EnclosingMethod (溯源) 截断逻辑
	if rel.Source != nil {
		qn := rel.Source.QualifiedName
		if rel.Source.Kind == model.Lambda || strings.Contains(qn, "$") || strings.Contains(qn, ".anonymousClass") {
			// 向上寻找最近的一个真正的 METHOD (即不含 lambda 或 anonymousClass 的部分)
			// 简单处理：截断到第一个括号或特殊标记前
			stopMarkers := []string{".lambda", ".anonymousClass", "$", ".block"}
			baseQN := qn
			for _, marker := range stopMarkers {
				if idx := strings.Index(baseQN, marker); idx != -1 {
					baseQN = baseQN[:idx]
				}
			}
			// 如果截断后末尾带括号，保留它以对齐 collector 的方法命名规范
			rel.Mores[RelCallEnclosingMethod] = baseQN
		}
	}
}

func (e *Extractor) enrichCreateCore(rel *model.DependencyRelation, node, stmt *sitter.Node, src []byte) {
	rel.Mores[RelAstKind] = "object_creation_expression"
	if stmt != nil && strings.Contains(stmt.Utf8Text(src), "[") {
		rel.Mores[RelCreateIsArray] = true
	}
}

func (e *Extractor) enrichThrowCore(rel *model.DependencyRelation, node, stmt *sitter.Node, rawText string, src []byte) {
	if node != nil {
		rel.Mores[RelAstKind] = "throw_statement"

		// 关键点：Action 发现的 Target Name 可能是 "new RuntimeException"
		// 必须确保它被 clean 过，变成 "RuntimeException"
		rel.Target.Name = e.clean(rel.Target.Name)
		rel.Target.QualifiedName = e.clean(rel.Target.QualifiedName)

		// 识别是否是 new 出来的运行时异常
		if node.Kind() == "type_identifier" || (node.Parent() != nil && node.Parent().Kind() == "object_creation_expression") {
			rel.Mores[RelThrowIsRuntime] = true
		} else if node.Kind() == "identifier" {
			rel.Mores[RelThrowIsRethrow] = true // throw e;
		}
		return
	}

	// 情况 B: 静态扫描捕获的方法签名 throws
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
	// 关键修复：细化 VARIABLE 到 LOCAL_VARIABLE 或 PARAMETER
	target := e.mapElementKindToAnnotationTarget(rel.Source)
	rel.Mores[RelAnnotationTarget] = target
	// 关键修复：确保目标名称不含括号参数
	rel.Target.Name = strings.Split(rel.Target.Name, "(")[0]
	rel.Target.QualifiedName = strings.Split(rel.Target.QualifiedName, "(")[0]
}

func (e *Extractor) enrichAssignCore(rel *model.DependencyRelation, capNode, stmtNode *sitter.Node, src []byte) {
	rel.Mores[RelAstKind] = stmtNode.Kind()
	rel.Mores[RelAssignTargetName] = capNode.Utf8Text(src)

	switch stmtNode.Kind() {
	case "variable_declarator":
		rel.Mores[RelAssignIsInitializer] = true
		rel.Mores[RelAssignOperator] = "="
		// 显式查找 value 节点
		for i := 0; i < int(stmtNode.ChildCount()); i++ {
			child := stmtNode.Child(uint(i))
			if stmtNode.FieldNameForChild(uint32(i)) == "value" {
				rel.Mores[RelAssignValueExpression] = child.Utf8Text(src)
				break
			}
		}
	case "assignment_expression":
		// 提取操作符 (如 =, +=, -=)
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

			// Extend
			if sc, ok := elem.Extra.Mores[ClassSuperClass].(string); ok && sc != "" {
				rels = append(rels, &model.DependencyRelation{
					Type: model.Extend, Source: elem, Target: e.quickResolve(e.clean(sc), model.Class, gCtx, fCtx),
				})
			}

			// Annotation
			for _, anno := range elem.Extra.Annotations {
				cleanName := e.clean(anno)
				rels = append(rels, &model.DependencyRelation{
					Type: model.Annotation, Source: elem, Target: e.quickResolve(cleanName, model.KAnnotation, gCtx, fCtx),
					Mores: map[string]interface{}{RelRawText: anno},
				})
			}

			// Method Signatures
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

			// TypeArgs
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
			if !strings.HasSuffix(capName, "_target") {
				continue
			}
			relType, kind, stmtNode := e.mapAction(capName, &cap.Node)
			if relType == "" {
				continue
			}

			// --- 核心修改：处理 Create 和 Call 双重关系 ---
			if relType == model.Create {
				// 1. 生成 Create 关系
				rels = append(rels, &model.DependencyRelation{
					Type:     model.Create,
					Source:   sourceElem,
					Target:   e.quickResolve(e.clean(cap.Node.Utf8Text(*fCtx.SourceBytes)), model.Class, gCtx, fCtx),
					Location: e.toLoc(cap.Node, fCtx.FilePath),
					Mores: map[string]interface{}{
						RelRawText: stmtNode.Utf8Text(*fCtx.SourceBytes),
						"tmp_node": &cap.Node,
						"tmp_stmt": stmtNode,
					},
				})

				// 2. 派生生成 Call (构造函数) 关系
				// 这里将 Kind 设为 Method，以便在 enrichCallCore 中处理括号
				rels = append(rels, &model.DependencyRelation{
					Type:     model.Call,
					Source:   sourceElem,
					Target:   e.quickResolve(e.clean(cap.Node.Utf8Text(*fCtx.SourceBytes)), model.Method, gCtx, fCtx),
					Location: e.toLoc(cap.Node, fCtx.FilePath),
					Mores: map[string]interface{}{
						RelRawText: stmtNode.Utf8Text(*fCtx.SourceBytes),
						"tmp_node": &cap.Node,
						"tmp_stmt": stmtNode,
					},
				})
				continue
			}

			// 原有的通用处理逻辑 (Call, Assign, Use, Throw 等)
			rels = append(rels, &model.DependencyRelation{
				Type:     relType,
				Source:   sourceElem,
				Target:   e.quickResolve(e.clean(cap.Node.Utf8Text(*fCtx.SourceBytes)), kind, gCtx, fCtx),
				Location: e.toLoc(cap.Node, fCtx.FilePath),
				Mores: map[string]interface{}{
					RelRawText: stmtNode.Utf8Text(*fCtx.SourceBytes),
					"tmp_node": &cap.Node,
					"tmp_stmt": stmtNode,
				},
			})
		}
	}
	return rels, nil
}

// =============================================================================
// 5. 辅助工具 (Helper Utilities)
// =============================================================================

// clean 清理 Java 类型/注解名称，移除泛型、变长参数、注解参数、通配符等。
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
	s = strings.Split(s, "(")[0]     // 移除注解参数，如 SuppressWarnings("all") -> SuppressWarnings
	s = strings.TrimSuffix(s, "...") // 移除变长参数标识
	return strings.TrimSpace(strings.TrimRight(s, "> ,[]"))
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

func (e *Extractor) mapAction(capName string, node *sitter.Node) (model.DependencyType, model.ElementKind, *sitter.Node) {
	switch capName {
	case "call_target", "ref_target":
		return model.Call, model.Method, node
	case "create_target":
		return model.Create, model.Class, node
	case "assign_target":
		// 关键：必须能向上找到这三种表达式之一，才能算作有效的赋值动作
		stmtNode := e.findNearestKind(node, "assignment_expression", "variable_declarator", "update_expression")
		if stmtNode != nil {
			return model.Assign, model.Variable, stmtNode
		}
		// 如果找不到（这通常不应该发生），降级返回自身
		return model.Assign, model.Variable, node

	case "update_stmt": // 自增自减
		return model.Assign, model.Variable, node

	case "use_field_target":
		return model.Use, model.Field, node
	case "throw_target":
		return model.Throw, model.Class, e.findThrowStatement(node)
	case "explicit_constructor_stmt":
		return model.Call, model.Method, node
	default:
		return "", model.Unknown, nil
	}
}

func (e *Extractor) determinePreciseSource(n *sitter.Node, fCtx *core.FileContext, gCtx *core.GlobalContext) *model.CodeElement {
	for curr := n.Parent(); curr != nil; curr = curr.Parent() {
		line := int(curr.StartPosition().Row) + 1
		var k model.ElementKind

		switch curr.Kind() {
		case "method_declaration", "constructor_declaration":
			k = model.Method
		case "static_initializer":
			k = model.ScopeBlock // 对应 $static$1
		case "lambda_expression":
			k = model.Lambda // 对应 lambda$1
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

		// 关键点：使用 fCtx.DefinitionsBySN 或 DefinitionsByQN(如果按位置存了)
		// 恢复旧代码的查找逻辑，确保行号对齐
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

func (e *Extractor) findParentKind(n *sitter.Node, kind string) *sitter.Node {
	for curr := n.Parent(); curr != nil; curr = curr.Parent() {
		if curr.Kind() == kind {
			return curr
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

func (e *Extractor) isPotentialClassName(s string) bool {
	if len(s) == 0 || strings.Contains(s, "(") {
		return false
	}
	firstChar := s[0]
	return firstChar >= 'A' && firstChar <= 'Z'
}
