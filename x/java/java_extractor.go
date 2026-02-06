package java

import (
	"fmt"
	"strings"

	"github.com/CodMac/arch-lens/core"
	"github.com/CodMac/arch-lens/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Extractor struct {
	resolver core.SymbolResolver
}

func NewJavaExtractor() *Extractor {
	resolver, err := core.GetSymbolResolver(core.LangJava)
	if err != nil {
		panic(err)
	}

	return &Extractor{
		resolver: resolver,
	}
}

// =============================================================================
// 主流水线 (Main Pipeline)
// =============================================================================

func (e *Extractor) Extract(filePath string, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gCtx.FileContexts[filePath]
	if !ok {
		return nil, fmt.Errorf("file context not found: %s", filePath)
	}

	// 1. 静态结构 + 动作发现
	hierarchyRels := e.extractHierarchy(fCtx, gCtx)
	structuralRels := e.extractStructural(fCtx, gCtx)
	actionRels, err := e.discoverActionRelations(fCtx, gCtx)
	if err != nil {
		return nil, err
	}

	// 2. 元数据增强
	enhanceTargets := append(structuralRels, actionRels...)
	for _, rel := range enhanceTargets {
		e.enrichCoreMetadata(rel, fCtx)
	}

	// 3. Capture关系发现（基于已提取到的ASSIGN和USE关系）
	captureRels := e.genCaptureRelations(enhanceTargets)

	// 4. 合并结果
	var allRels []*model.DependencyRelation
	allRels = append(allRels, hierarchyRels...)
	allRels = append(allRels, structuralRels...)
	allRels = append(allRels, actionRels...)
	allRels = append(allRels, captureRels...)

	return allRels, nil
}

// =============================================================================
// 1. 静态结构 + 发现逻辑 (Discovery Logic)
// =============================================================================

func (e *Extractor) extractHierarchy(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation

	fileDef, ok := gCtx.FindByQualifiedName(fCtx.FilePath)
	if ok {
		fileSource := fileDef.Element
		for _, imports := range fCtx.Imports {
			for _, imp := range imports {
				target := e.resolver.Resolve(gCtx, fCtx, nil, imp.RawImportPath, imp.Kind)
				rels = append(rels, &model.DependencyRelation{
					Type: model.Import, Source: fileSource, Target: target, Location: imp.Location,
				})
			}
		}
	}

	for _, entry := range fCtx.Definitions {
		if entry.ParentQN != "" {
			if parent, ok := gCtx.FindByQualifiedName(entry.ParentQN); ok {
				rels = append(rels, &model.DependencyRelation{Type: model.Contain, Source: parent.Element, Target: entry.Element})
			}
		}
	}
	return rels
}

func (e *Extractor) extractStructural(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	for _, entry := range fCtx.Definitions {
		elem := entry.Element
		if elem.Extra == nil {
			continue
		}

		// --- 1. 处理 Class (Extend/Implement) ---
		if elem.Kind == model.Class {
			if sc, ok := elem.Extra.Mores[ClassSuperClass].(string); ok && sc != "" {
				target := e.resolver.Resolve(gCtx, fCtx, nil, e.clean(sc), model.Class)
				rels = append(rels, &model.DependencyRelation{Type: model.Extend, Source: elem, Target: target})
			}
			if impls, ok := elem.Extra.Mores[ClassImplementedInterfaces].([]string); ok {
				for _, implName := range impls {
					target := e.resolver.Resolve(gCtx, fCtx, nil, e.clean(implName), model.Interface)
					rels = append(rels, &model.DependencyRelation{Type: model.Implement, Source: elem, Target: target})
				}
			}
		}
		if elem.Kind == model.AnonymousClass {
			if ac, ok := elem.Extra.Mores[AnonymousClassType].(string); ok && ac != "" {
				target := e.resolver.Resolve(gCtx, fCtx, nil, e.clean(ac), model.Class)
				rels = append(rels, &model.DependencyRelation{Type: model.Extend, Source: elem, Target: target})
			}
		}

		// --- 2. 处理 Interface (Extend) ---
		if elem.Kind == model.Interface {
			if impls, ok := elem.Extra.Mores[InterfaceImplementedInterfaces].([]string); ok {
				for _, implName := range impls {
					target := e.resolver.Resolve(gCtx, fCtx, nil, e.clean(implName), model.Interface)
					rels = append(rels, &model.DependencyRelation{Type: model.Extend, Source: elem, Target: target})
				}
			}
		}

		// --- 3. 处理注解 (Annotation) ---
		for _, anno := range elem.Extra.Annotations {
			target := e.resolver.Resolve(gCtx, fCtx, nil, e.clean(anno), model.KAnnotation)
			rels = append(rels, &model.DependencyRelation{
				Type: model.Annotation, Source: elem, Target: target,
				Mores: map[string]interface{}{RelRawText: anno},
			})
		}

		// --- 4. 处理方法签名 (Parameter/Return/Throw) ---
		if elem.Kind == model.Method {
			if pts, ok := elem.Extra.Mores[MethodParameters].([]string); ok {
				for _, p := range pts {
					typePart := e.extractTypeFromParam(p)
					target := e.resolver.Resolve(gCtx, fCtx, nil, e.clean(typePart), model.Class)
					rels = append(rels, &model.DependencyRelation{
						Type: model.Parameter, Source: elem, Target: target,
						Mores: map[string]interface{}{"tmp_raw": p},
					})
				}
			}
			if rt, ok := elem.Extra.Mores[MethodReturnType].(string); ok && rt != "void" && rt != "" {
				target := e.resolver.Resolve(gCtx, fCtx, nil, e.clean(rt), model.Class)
				rels = append(rels, &model.DependencyRelation{
					Type: model.Return, Source: elem, Target: target,
					Mores: map[string]interface{}{"tmp_raw": rt},
				})
			}
			if ths, ok := elem.Extra.Mores[MethodThrowsTypes].([]string); ok {
				for _, ex := range ths {
					target := e.resolver.Resolve(gCtx, fCtx, nil, e.clean(ex), model.Class)
					rels = append(rels, &model.DependencyRelation{
						Type: model.Throw, Source: elem, Target: target,
						Mores: map[string]interface{}{"tmp_raw": ex},
					})
				}
			}
		}

		// --- 5. 处理变量泛型 (TypeArg) ---
		for _, rt := range e.getRawTypesForTypeArgs(elem) {
			rels = append(rels, e.collectAllTypeArgs(rt, elem, gCtx, fCtx)...)
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
			if !strings.HasSuffix(capName, "_target") && capName != "explicit_constructor_stmt" && capName != "id_atom" {
				continue
			}

			// 1. 调用 mapAction 获取动作定义
			actionTargets := e.mapAction(capName, &cap.Node, fCtx, gCtx)
			for _, at := range actionTargets {
				if at.RelType == "" || at.Target == nil {
					continue
				}

				// 2. 这里的 at.Target 已经在 mapAction 中经过了过滤和 resolve
				ctxNode := at.ContextNode
				if ctxNode == nil {
					ctxNode = at.TargetNode.Parent()
				}

				rels = append(rels, &model.DependencyRelation{
					Type:     at.RelType,
					Source:   sourceElem,
					Target:   at.Target, // 使用 mapAction resolve 好的对象
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
		e.enrichUseCore(rel, node, stmt, src)
	case model.Cast:
		e.enrichCastCore(rel, node, stmt, src)
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
	if stmt == nil {
		return
	}

	// 1. 通用属性 (无需前缀)
	rel.Mores[RelAstKind] = stmt.Kind()
	rel.Mores[RelRawText] = stmt.Utf8Text(src)

	// 2. 专用属性提取：变量名 (RelCreateVariableName)
	contextNode := stmt
	if stmt.Kind() == "object_creation_expression" || stmt.Kind() == "array_creation_expression" {
		if p := stmt.Parent(); p != nil && p.Kind() == "variable_declarator" {
			contextNode = p
		}
	}
	if contextNode.Kind() == "variable_declarator" {
		if nameNode := contextNode.ChildByFieldName("name"); nameNode != nil {
			rel.Mores[RelCreateVariableName] = nameNode.Utf8Text(src)
		}
	}

	// 3. 专用属性提取：数组 (RelCreateIsArray)
	if stmt.Kind() == "array_creation_expression" {
		rel.Mores[RelCreateIsArray] = true
	}

	// 4. 特殊处理 super() -> Object 的情况
	if stmt.Kind() == "explicit_constructor_invocation" && strings.Contains(stmt.Utf8Text(src), "super") {
		rel.Target.Name = "Object"
		rel.Target.QualifiedName = "Object"
	}
}

func (e *Extractor) enrichAssignCore(rel *model.DependencyRelation, node, stmt *sitter.Node, src []byte) {
	if stmt == nil {
		return
	}

	rel.Mores[RelAssignTargetName] = node.Utf8Text(src)
	rel.Mores[RelRawText] = stmt.Utf8Text(src)
	rel.Mores[RelAstKind] = node.Kind()

	switch stmt.Kind() {
	case "variable_declarator":
		rel.Mores[RelAssignIsInitializer] = true
		rel.Mores[RelAssignOperator] = "="
		if val := stmt.ChildByFieldName("value"); val != nil {
			rel.Mores[RelAssignValueExpression] = val.Utf8Text(src)
		}

	case "assignment_expression":
		rel.Mores[RelAssignIsInitializer] = false
		if op := stmt.ChildByFieldName("operator"); op != nil {
			rel.Mores[RelAssignOperator] = op.Utf8Text(src)
		} else {
			rel.Mores[RelAssignOperator] = "="
		}
		if right := stmt.ChildByFieldName("right"); right != nil {
			rel.Mores[RelAssignValueExpression] = right.Utf8Text(src)
		}

	case "update_expression":
		rel.Mores[RelAssignIsInitializer] = false
		txt := stmt.Utf8Text(src)
		if strings.Contains(txt, "++") {
			rel.Mores[RelAssignOperator] = "++"
		}
		if strings.Contains(txt, "--") {
			rel.Mores[RelAssignOperator] = "--"
		}
		rel.Mores[RelAssignValueExpression] = ""
	}

	// 3. 处理 EnclosingMethod 和 IsCapture
	if rel.Source != nil {
		qn := rel.Source.QualifiedName
		stopMarkers := []string{".lambda", ".anonymousClass", "$", ".block"}
		for _, marker := range stopMarkers {
			if idx := strings.Index(qn, marker); idx != -1 {
				rel.Mores[RelAssignEnclosingMethod] = qn[:idx]
				break
			}
		}

		isSubScope := strings.Contains(qn, "lambda$") || strings.Contains(qn, ".anonymousClass")
		isTargetField := rel.Target != nil && rel.Target.Kind == model.Field

		if isSubScope && isTargetField {
			rel.Mores[RelAssignIsCapture] = true
		}
	}
}

func (e *Extractor) enrichCastCore(rel *model.DependencyRelation, node, stmt *sitter.Node, src []byte) {
	if stmt == nil {
		return
	}
	rel.Mores[RelAstKind] = stmt.Kind()
	rel.Mores[RelRawText] = stmt.Utf8Text(src)
	rel.Mores[RelCastIsInstanceof] = stmt.Kind() == "instanceof_expression"
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

func (e *Extractor) enrichUseCore(rel *model.DependencyRelation, node, stmt *sitter.Node, src []byte) {
	if node == nil || stmt == nil {
		return
	}

	rel.Mores[RelAstKind] = node.Kind()
	rel.Mores[RelContext] = stmt.Kind()
	rel.Mores[RelRawText] = stmt.Utf8Text(src)

	// --- 核心优化：多维度提取 Receiver ---
	parent := node.Parent()
	if parent != nil {
		switch parent.Kind() {
		case "field_access":
			// 处理 user.name 或 this.field
			if obj := parent.ChildByFieldName("object"); obj != nil {
				rel.Mores[RelUseReceiver] = obj.Utf8Text(src)
				rel.Mores[RelUseTargetName] = node.Utf8Text(src)
			}
		case "scoped_identifier":
			// 处理 ClassName.CONSTANT
			if scope := parent.ChildByFieldName("scope"); scope != nil {
				rel.Mores[RelUseReceiver] = scope.Utf8Text(src)
				rel.Mores[RelUseTargetName] = node.Utf8Text(src)
			}
		case "method_invocation":
			// 如果标识符本身是方法名（由 call_target 处理时），这里可以提取 object
			if obj := parent.ChildByFieldName("object"); obj != nil {
				rel.Mores[RelUseReceiver] = obj.Utf8Text(src)
			}
		}
	}

	// --- 兜底逻辑：隐式 this 判定 ---
	if _, exists := rel.Mores[RelUseReceiver]; !exists {
		// 如果 Target 是字段，且没有显式的接收者，则默认为 this
		if rel.Target != nil && rel.Target.Kind == model.Field {
			rel.Mores[RelUseReceiver] = "this"
			rel.Mores[RelUseTargetName] = node.Utf8Text(src)
		}
	}

	// --- 提取接收者类型 QN ---
	// 这里利用你之前从 Target.Extra 中收集到的 RawType
	if rel.Target != nil && rel.Target.Extra != nil {
		keys := []string{FieldRawType, VariableRawType}
		for _, k := range keys {
			if rt, ok := rel.Target.Extra.Mores[k].(string); ok {
				// e.clean 会去掉泛型和修饰符，保留纯粹的类型名
				rel.Mores[RelUseReceiverType] = e.clean(rt)
				break
			}
		}
	}

	// 处理 EnclosingMethod 和 IsCapture
	if rel.Source != nil {
		qn := rel.Source.QualifiedName

		// 1. 溯源 EnclosingMethod
		stopMarkers := []string{".lambda", ".anonymousClass", "$", ".block"}
		for _, marker := range stopMarkers {
			if idx := strings.Index(qn, marker); idx != -1 {
				rel.Mores[RelUseEnclosingMethod] = qn[:idx]
				break
			}
		}

		// 2. 识别跨作用域捕获 (IsCapture)
		isSubScope := strings.Contains(qn, "lambda$") || strings.Contains(qn, ".anonymousClass")
		if isSubScope {
			if rel.Target.Kind == model.Field {
				rel.Mores[RelUseIsCapture] = true
			}
			if rel.Target.Kind == model.Variable && rel.Source.Location != nil && rel.Target.Location != nil {
				if rel.Source.Location.FilePath == rel.Target.Location.FilePath {
					srcStart := rel.Source.Location.StartLine
					srcEnd := rel.Source.Location.EndLine
					defLine := rel.Target.Location.StartLine
					if defLine < srcStart || defLine > srcEnd {
						rel.Mores[RelUseIsCapture] = true
					}
				}
			}
		}
	}
}

// =============================================================================
// 3.  Capture关系发现（基于已提取到的ASSIGN和USE关系）
// =============================================================================

func (e *Extractor) genCaptureRelations(deps []*model.DependencyRelation) []*model.DependencyRelation {
	var captures []*model.DependencyRelation
	seen := make(map[string]bool)

	for _, rel := range deps {
		if rel.Source == nil || rel.Target == nil {
			continue
		}

		isCapture := false

		if rel.Type == model.Use {
			if val, ok := rel.Mores[RelUseIsCapture]; ok {
				if b, isBool := val.(bool); isBool && b {
					isCapture = true
				}
			}
		}

		if rel.Type == model.Assign {
			if val, ok := rel.Mores[RelAssignIsCapture]; ok {
				if b, isBool := val.(bool); isBool && b {
					isCapture = true
				}
			}
		}

		if isCapture {
			key := rel.Source.QualifiedName + "->" + rel.Target.QualifiedName

			if !seen[key] {
				seen[key] = true
				captureRel := &model.DependencyRelation{
					Source:   rel.Source,
					Target:   rel.Target,
					Type:     model.Capture,
					Location: rel.Location,
					Mores:    make(map[string]interface{}),
				}
				captures = append(captures, captureRel)
			}
		}
	}
	return captures
}

// =============================================================================
// 4. 辅助工具 (Helper Utilities)
// =============================================================================

type ActionTarget struct {
	RelType     model.DependencyType
	TargetNode  *sitter.Node
	ContextNode *sitter.Node
	Target      *model.CodeElement
}

func (e *Extractor) mapAction(capName string, node *sitter.Node, fCtx *core.FileContext, gCtx *core.GlobalContext) []ActionTarget {
	src := *fCtx.SourceBytes
	text := node.Utf8Text(src)

	// element 匹配
	resolve := func(symbol string, kind model.ElementKind) *model.CodeElement {
		return e.resolver.Resolve(gCtx, fCtx, node, e.clean(symbol), kind)
	}

	switch capName {
	case "call_target", "ref_target":
		ctx := e.findNearestKind(node, "method_invocation", "method_reference", "explicit_constructor_invocation", "object_creation_expression")
		return []ActionTarget{{model.Call, node, ctx, resolve(text, model.Method)}}

	case "create_target":
		ctx := e.findNearestKind(node, "object_creation_expression", "array_creation_expression")
		return []ActionTarget{
			{model.Create, node, ctx, resolve(text, model.Class)},
			{model.Call, node, ctx, resolve(text, model.Method)},
		}

	case "cast_target":
		ctx := e.findNearestKind(node, "cast_expression", "instanceof_expression")
		return []ActionTarget{{model.Cast, node, ctx, resolve(text, model.Class)}}

	case "assign_target":
		// 1. 向上寻找赋值的上下文容器
		ctx := e.findNearestKind(node, "assignment_expression", "variable_declarator", "update_expression")
		if ctx == nil {
			return nil
		}

		return []ActionTarget{{model.Assign, node, ctx, resolve(text, model.Variable)}}

	case "id_atom":
		target := resolve(text, model.Variable)
		if !e.isUseRel(node, target) {
			return nil
		}

		ctx := e.findNearestKind(node, "expression_statement", "local_variable_declaration", "enhanced_for_statement", "binary_expression", "cast_expression", "array_access", "parenthesized_expression", "field_access", "lambda_expression", "assignment_expression")
		return []ActionTarget{{model.Use, node, ctx, resolve(text, model.Variable)}}

	case "throw_target":
		ctx := e.findNearestKind(node, "throw_statement")
		return []ActionTarget{{model.Throw, node, ctx, resolve(text, model.Class)}}

	case "explicit_constructor_stmt":
		return []ActionTarget{
			{model.Call, node, node, resolve(text, model.Method)},
			{model.Create, node, node, resolve(text, model.Class)},
		}

	default:
		return nil
	}
}

func (e *Extractor) isUseRel(node *sitter.Node, target *model.CodeElement) bool {
	// 1. 快速定位符号：如果不是变量或字段，直接 pass
	if target == nil || (target.Kind != model.Variable && target.Kind != model.Field) {
		return false
	}

	// 2. 排除定义点：通过父节点的 FieldName 判断
	// 如果当前 identifier 是其父节点的 "name" 字段，说明它是声明，不是使用
	parent := node.Parent()
	if parent.ChildByFieldName("name") != nil && parent.ChildByFieldName("name").Id() == node.Id() {
		return false
	}

	// 3. 排除特定语法噪声
	switch parent.Kind() {
	case "method_invocation", "method_reference":
		// 如果是方法调用/引用的名称部分，由 call_target 处理
		if nameNode := parent.ChildByFieldName("name"); nameNode != nil && nameNode.Id() == node.Id() {
			return false
		}
	case "scoped_identifier", "package_declaration", "import_declaration":
		return false
	}

	return true
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
	if strings.Contains(s, "(") {
		return false
	}
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
	keys := []string{FieldRawType, VariableRawType, MethodReturnType}

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
		target := e.resolver.Resolve(gCtx, fCtx, nil, e.clean(arg), model.Class)
		rels = append(rels, &model.DependencyRelation{
			Type: model.TypeArg, Source: source, Target: target,
			Mores: map[string]interface{}{RelTypeArgIndex: i, RelRawText: arg, RelAstKind: "type_arguments"},
		})

		if strings.Contains(arg, "<") {
			rels = append(rels, e.collectAllTypeArgs(arg, source, gCtx, fCtx)...)
		}
	}
	return rels
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
		for _, entry := range fCtx.Definitions {
			if entry.Element.Kind == k && entry.Element.Location.StartLine == line {
				return entry.Element
			}
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
