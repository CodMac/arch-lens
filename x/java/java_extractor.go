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

// Extract 是 Java 提取器的核心入口。它按照：静态结构提取 -> 动态动作发现 -> 元数据增强 的顺序执行。
func (e *Extractor) Extract(filePath string, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gCtx.FileContexts[filePath]
	if !ok {
		return nil, fmt.Errorf("file context not found: %s", filePath)
	}

	var allRels []*model.DependencyRelation

	// --- 阶段 1: 静态结构 (Hierarchy & Structural) ---
	allRels = append(allRels, e.extractHierarchy(fCtx, gCtx)...)
	allRels = append(allRels, e.extractStructural(fCtx, gCtx)...)

	// --- 阶段 2: 动作发现 (Discovery) ---
	actionRels, err := e.discoverActionRelations(fCtx, gCtx)
	if err != nil {
		return nil, err
	}

	// --- 阶段 3: 核心元数据增强 (Core Enrichment) ---
	for _, rel := range actionRels {
		e.enrichCoreMetadata(rel, fCtx)
	}

	allRels = append(allRels, actionRels...)
	return allRels, nil
}

// =============================================================================
// 2. 核心关系发现 (Action Discovery)
// =============================================================================

// discoverActionRelations 通过 Tree-sitter Query 捕获方法体内的 Call/Create/Assign 等动态行为。
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

		// capturedNode 通常是具体的表达式节点（如 method_invocation, object_creation_expression）
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

			// --- 映射动作类型并确定语句范围 ---
			// mapAction 会自动识别是否在 throw 语句中，并调整 stmtNode 范围
			relType, kind, stmtNode := e.mapAction(capName, capturedNode)
			if relType == "" {
				continue
			}

			rawText := cap.Node.Utf8Text(*fCtx.SourceBytes)
			rel := &model.DependencyRelation{
				Type:     relType,
				Source:   sourceElem,
				Target:   e.quickResolve(e.clean(rawText), kind, gCtx, fCtx),
				Location: e.toLoc(cap.Node, fCtx.FilePath),
				Mores: map[string]interface{}{
					RelRawText: stmtNode.Utf8Text(*fCtx.SourceBytes),
					RelContext: stmtNode.Kind(),
					"tmp_node": &cap.Node, // 暂存用于后续 Enrichment 阶段
					"tmp_stmt": stmtNode,
				},
			}
			rels = append(rels, rel)
		}
	}
	return rels, nil
}

// =============================================================================
// 3. 元数据增强 (Metadata Enrichment)
// =============================================================================

// enrichCoreMetadata 根据关系类型进一步补充详细字段（如 Receiver, IsStatic, ThrowKind 等）。
func (e *Extractor) enrichCoreMetadata(rel *model.DependencyRelation, fCtx *core.FileContext) {
	node, _ := rel.Mores["tmp_node"].(*sitter.Node)
	stmt, _ := rel.Mores["tmp_stmt"].(*sitter.Node)
	delete(rel.Mores, "tmp_node")
	delete(rel.Mores, "tmp_stmt")

	if node == nil || stmt == nil {
		return
	}

	source := *fCtx.SourceBytes

	switch rel.Type {
	case model.Call:
		e.enrichCallCore(rel, node, source, fCtx)
	case model.Create:
		e.enrichCreateCore(rel, node, stmt, source)
	case model.Assign:
		e.enrichAssignCore(rel, node, stmt, source)
	case model.Use:
		e.enrichUseCore(rel, node, source)
	case model.Throw:
		e.enrichThrowCore(rel, node, stmt, source)
	}
}

// enrichCallCore 提取方法调用的接收者、静态状态及闭包上下文。
func (e *Extractor) enrichCallCore(rel *model.DependencyRelation, capNode *sitter.Node, src []byte, fCtx *core.FileContext) {
	callNode := e.findNearestKind(capNode, "method_invocation", "method_reference", "explicit_constructor_invocation")
	if callNode == nil {
		return
	}
	rel.Mores[RelAstKind] = callNode.Kind()

	if obj := callNode.ChildByFieldName("object"); obj != nil {
		recText := obj.Utf8Text(src)
		rel.Mores[RelCallReceiver] = recText
		// 简单启发式判定静态调用 (大写字母开头且非方法调用)
		if len(recText) > 0 && recText[0] >= 'A' && recText[0] <= 'Z' && !strings.Contains(recText, "(") {
			rel.Mores[RelCallIsStatic] = true
			rel.Mores[RelCallReceiverType] = recText
		}
		if obj.Kind() == "method_invocation" {
			rel.Mores[RelCallIsChained] = true
		}
	} else {
		rel.Mores[RelCallReceiver] = "this"
	}

	if callNode.Kind() == "method_reference" {
		rel.Mores[RelCallIsFunctional] = true
	}
	if callNode.Kind() == "explicit_constructor_invocation" {
		rel.Mores[RelCallIsConstructor] = true
	}

	if enc := e.findParentKind(callNode, "method_declaration"); enc != nil {
		if name := enc.ChildByFieldName("name"); name != nil {
			rel.Mores[RelCallEnclosingMethod] = name.Utf8Text(src) + "()"
		}
	}
}

// enrichCreateCore 提取对象/数组创建的详细元数据。
func (e *Extractor) enrichCreateCore(rel *model.DependencyRelation, capNode, stmtNode *sitter.Node, src []byte) {
	rel.Mores[RelAstKind] = "object_creation_expression"
	rel.Mores[RelCallIsConstructor] = true
	rel.Mores[RelCreateIsArray] = strings.Contains(stmtNode.Utf8Text(src), "[")

	if stmtNode.Kind() == "variable_declarator" {
		if name := stmtNode.ChildByFieldName("name"); name != nil {
			rel.Mores[RelCreateVariableName] = name.Utf8Text(src)
		}
	}
}

// enrichThrowCore 判定抛出异常的行为类型（主动抛出 vs 重新抛出）。
func (e *Extractor) enrichThrowCore(rel *model.DependencyRelation, capNode, stmtNode *sitter.Node, src []byte) {
	rel.Mores[RelAstKind] = "throw_statement"
	// 如果捕获的是类型标识符或位于对象创建中，则视为 new 出来的 RuntimeException
	if capNode.Kind() == "type_identifier" || capNode.Parent().Kind() == "object_creation_expression" {
		rel.Mores[RelThrowIsRuntime] = true
	} else if capNode.Kind() == "identifier" {
		rel.Mores[RelThrowIsRethrow] = true // e.g., throw e;
	}
}

// enrichAssignCore 提取赋值操作符和右值表达式。
func (e *Extractor) enrichAssignCore(rel *model.DependencyRelation, capNode, stmtNode *sitter.Node, src []byte) {
	rel.Mores[RelAssignTargetName] = capNode.Utf8Text(src)
	assignNode := e.findNearestKind(capNode, "assignment_expression", "variable_declarator", "update_expression")
	if assignNode == nil {
		return
	}
	rel.Mores[RelAstKind] = assignNode.Kind()

	switch assignNode.Kind() {
	case "assignment_expression":
		if op := assignNode.ChildByFieldName("operator"); op != nil {
			rel.Mores[RelAssignOperator] = op.Utf8Text(src)
		}
		if right := assignNode.ChildByFieldName("right"); right != nil {
			rel.Mores[RelAssignValueExpression] = right.Utf8Text(src)
		}
	case "variable_declarator":
		rel.Mores[RelAssignIsInitializer] = true
		rel.Mores[RelAssignOperator] = "="
		if val := assignNode.ChildByFieldName("value"); val != nil {
			rel.Mores[RelAssignValueExpression] = val.Utf8Text(src)
		}
	case "update_expression":
		raw := assignNode.Utf8Text(src)
		if strings.Contains(raw, "++") {
			rel.Mores[RelAssignOperator] = "++"
		} else {
			rel.Mores[RelAssignOperator] = "--"
		}
	}
}

// enrichUseCore 处理字段访问的上下文（Receiver）。
func (e *Extractor) enrichUseCore(rel *model.DependencyRelation, capNode *sitter.Node, src []byte) {
	if fieldAccess := e.findNearestKind(capNode, "field_access"); fieldAccess != nil {
		if obj := fieldAccess.ChildByFieldName("object"); obj != nil {
			rel.Mores[RelUseReceiver] = obj.Utf8Text(src)
		}
	}
}

// =============================================================================
// 4. 辅助工具 (Helper Utilities)
// =============================================================================

// mapAction 将 Query Capture 名称映射为模型中的关系类型，并自动提升 throw 语句的语句范围。
func (e *Extractor) mapAction(capName string, node *sitter.Node) (model.DependencyType, model.ElementKind, *sitter.Node) {
	var relType model.DependencyType
	var kind model.ElementKind

	switch capName {
	case "call_target", "ref_target", "explicit_constructor_stmt":
		relType, kind = model.Call, model.Method
	case "create_target":
		relType, kind = model.Create, model.Class
	case "assign_target":
		relType, kind = model.Assign, model.Variable
	case "use_field_target":
		relType, kind = model.Use, model.Field
	case "throw_target":
		relType, kind = model.Throw, model.Class
	default:
		return "", model.Unknown, nil
	}

	// 向上检查当前节点是否包裹在 throw_statement 中
	if throwStmt := e.findThrowStatement(node); throwStmt != nil {
		return model.Throw, model.Class, throwStmt
	}

	return relType, kind, node
}

// determinePreciseSource 向上查找确定代码动作的具体所有者（Method/Field）。
func (e *Extractor) determinePreciseSource(n *sitter.Node, fCtx *core.FileContext, gCtx *core.GlobalContext) *model.CodeElement {
	for curr := n.Parent(); curr != nil; curr = curr.Parent() {
		line := int(curr.StartPosition().Row) + 1
		var k model.ElementKind
		switch curr.Kind() {
		case "method_declaration", "constructor_declaration":
			k = model.Method
		case "field_declaration":
			k = model.Field
		case "variable_declarator":
			if curr.Parent() != nil && curr.Parent().Kind() == "field_declaration" {
				k = model.Field
			} else {
				continue
			}
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

// findThrowStatement 查找节点所属的 throw 语句块。
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

// findNearestKind 向上搜索指定类型的节点，遇到语句/类边界时停止。
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

// quickResolve 尝试在全局上下文中解析符号，若解析失败则返回存根元素。
func (e *Extractor) quickResolve(symbol string, kind model.ElementKind, gCtx *core.GlobalContext, fCtx *core.FileContext) *model.CodeElement {
	if entries := gCtx.ResolveSymbol(fCtx, symbol); len(entries) > 0 {
		return entries[0].Element
	}
	return &model.CodeElement{Name: symbol, QualifiedName: symbol, Kind: kind}
}

// clean 去除类型中的泛型符号和注解前缀。
func (e *Extractor) clean(s string) string {
	s = strings.TrimPrefix(s, "@")

	// 1. 处理通配符边界
	if strings.Contains(s, "extends ") {
		parts := strings.Split(s, "extends ")
		s = parts[len(parts)-1]
	} else if strings.Contains(s, "super ") {
		parts := strings.Split(s, "super ")
		s = parts[len(parts)-1]
	}

	// 2. 移除泛型起始及其后的所有内容
	s = strings.Split(s, "<")[0]

	// 3. 核心修复：移除末尾可能残留的泛型闭合符、逗号或数组括号残留
	// 使用 TrimRight 移除所有在类型解析中可能残留的“垃圾”字符
	s = strings.TrimRight(s, "> ,[]")

	return strings.TrimSpace(s)
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

// =============================================================================
// 5. 静态结构处理 (Hierarchy & Structural)
// =============================================================================

// extractHierarchy 处理 Import 导入及包/类/方法的 Contain 包含关系。
func (e *Extractor) extractHierarchy(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	// 文件通常是第一个定义
	fileSource := gCtx.DefinitionsByQN[fCtx.FilePath][0].Element
	for _, imports := range fCtx.Imports {
		for _, imp := range imports {
			rels = append(rels, &model.DependencyRelation{
				Type:     model.Import,
				Source:   fileSource,
				Target:   e.quickResolve(imp.RawImportPath, imp.Kind, gCtx, fCtx),
				Location: imp.Location,
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

// extractStructural 处理继承、注解、方法参数、返回值以及抛出异常签名。
func (e *Extractor) extractStructural(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			elem := entry.Element
			if elem.Extra == nil {
				continue
			}

			// 继承关系 (Extend)
			if sc, ok := elem.Extra.Mores[ClassSuperClass].(string); ok && sc != "" {
				rels = append(rels, &model.DependencyRelation{
					Type:   model.Extend,
					Source: elem,
					Target: e.quickResolve(e.clean(sc), model.Class, gCtx, fCtx),
				})
			}

			// 注解 (Annotation)
			for _, anno := range elem.Extra.Annotations {
				namePart := strings.TrimPrefix(anno, "@")
				baseName := strings.Split(namePart, "(")[0]
				mores := map[string]interface{}{
					RelRawText:          anno,
					RelAnnotationTarget: e.mapElementKindToAnnotationTarget(elem),
				}
				if strings.Contains(namePart, "(") {
					val := strings.TrimSuffix(strings.SplitN(namePart, "(", 2)[1], ")")
					if !strings.Contains(val, "=") {
						mores[RelAnnotationValue] = val
					}
				}
				rels = append(rels, &model.DependencyRelation{
					Type:   model.Annotation,
					Source: elem,
					Target: e.quickResolve(e.clean(baseName), model.KAnnotation, gCtx, fCtx),
					Mores:  mores,
				})
			}

			// 方法签名相关 (Parameter / Return / Throw Signature)
			if elem.Kind == model.Method {
				// 参数处理
				if params, ok := elem.Extra.Mores[MethodParameters].([]string); ok {
					for i, rawParam := range params {
						parts := strings.Fields(rawParam)
						if len(parts) == 0 {
							continue
						}
						var typePart string
						var paramName string
						if len(parts) >= 2 {
							paramName = parts[len(parts)-1]
							typePart = parts[len(parts)-2]
						} else {
							typePart = parts[0]
						}
						isVarargs := strings.HasSuffix(typePart, "...")
						typePart = strings.TrimSuffix(typePart, "...")

						relMores := map[string]interface{}{
							RelParameterIndex: i,
							RelParameterName:  paramName,
						}
						if isVarargs {
							relMores[RelParameterIsVarargs] = true
						}
						rels = append(rels, &model.DependencyRelation{
							Type:   model.Parameter,
							Source: elem,
							Target: e.quickResolve(e.clean(typePart), model.Class, gCtx, fCtx),
							Mores:  relMores,
						})
					}
				}

				// 返回值处理
				if rawReturnType, ok := elem.Extra.Mores[MethodReturnType].(string); ok && rawReturnType != "" && rawReturnType != "void" {
					dims := strings.Count(rawReturnType, "[]")
					cleanType := e.clean(rawReturnType)
					cleanType = strings.ReplaceAll(cleanType, "[]", "")
					rels = append(rels, &model.DependencyRelation{
						Type:     model.Return,
						Source:   elem,
						Target:   e.quickResolve(cleanType, model.Class, gCtx, fCtx),
						Location: elem.Location,
						Mores: map[string]interface{}{
							RelReturnIsPrimitive: e.isPrimitive(cleanType),
							RelReturnIsArray:     dims > 0,
							RelReturnDimensions:  dims,
						},
					})
				}

				// 抛出异常声明处理 (Throws Clause)
				if throws, ok := elem.Extra.Mores[MethodThrowsTypes].([]string); ok {
					for i, exStr := range throws {
						rels = append(rels, &model.DependencyRelation{
							Type:   model.Throw,
							Source: elem,
							Target: e.quickResolve(e.clean(exStr), model.Class, gCtx, fCtx),
							Mores: map[string]interface{}{
								RelThrowIsSignature: true,
								RelThrowIndex:       i,
							},
						})
					}
				}
			}

			// --- 泛型实参提取 (TypeArg) ---
			var rawTypeStrings []string

			// 1. 字段类型
			if vt, ok := elem.Extra.Mores[FieldType].(string); ok {
				rawTypeStrings = append(rawTypeStrings, vt)
			}
			// 2. 变量类型
			if vt, ok := elem.Extra.Mores[VariableType].(string); ok {
				rawTypeStrings = append(rawTypeStrings, vt)
			}
			// 3. 方法返回类型
			if rt, ok := elem.Extra.Mores[MethodReturnType].(string); ok {
				rawTypeStrings = append(rawTypeStrings, rt)
			}
			// 4. 方法参数类型 (这里针对结构化定义的参数)
			if pts, ok := elem.Extra.Mores[MethodParameters].([]string); ok {
				for _, p := range pts {
					parts := strings.Fields(p)
					if len(parts) >= 2 {
						// 取倒数第二个作为类型名
						rawTypeStrings = append(rawTypeStrings, parts[len(parts)-2])
					}
				}
			}

			// 处理 TypeArg
			for _, rt := range rawTypeStrings {
				// 调用递归提取函数
				rels = append(rels, e.collectAllTypeArgs(rt, elem, gCtx, fCtx)...)
			}
		}
	}
	return rels
}

// mapElementKindToAnnotationTarget 辅助判定注解目标类型。
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

// extractTypeArgs 解析类型字符串中的泛型实参, (eg,: "Map<String, Integer>" 返回 ["String", "Integer"])
func (e *Extractor) parseTypeArgs(rawType string) []string {
	start := strings.Index(rawType, "<")
	end := strings.LastIndex(rawType, ">")
	if start == -1 || end == -1 || start >= end {
		return nil
	}

	content := rawType[start+1 : end]
	var args []string
	bracketLevel := 0
	current := strings.Builder{}

	// 处理带逗号的嵌套，如 Map<String, List<Integer>>
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

// collectAllTypeArgs 递归提取所有层级的泛型实参
func (e *Extractor) collectAllTypeArgs(rt string, source *model.CodeElement, gCtx *core.GlobalContext, fCtx *core.FileContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	if !strings.Contains(rt, "<") {
		return nil
	}

	args := e.parseTypeArgs(rt)
	for i, arg := range args {
		// 1. 生成当前层的关系
		rels = append(rels, &model.DependencyRelation{
			Type:   model.TypeArg,
			Source: source,
			Target: e.quickResolve(e.clean(arg), model.Class, gCtx, fCtx),
			Mores: map[string]interface{}{
				RelTypeArgIndex: i,
				RelRawText:      arg,
				RelAstKind:      "type_arguments",
			},
		})

		// 2. 递归处理下一层 (例如 Map<String, Object> 里的 String 和 Object)
		if strings.Contains(arg, "<") {
			rels = append(rels, e.collectAllTypeArgs(arg, source, gCtx, fCtx)...)
		}
	}
	return rels
}
