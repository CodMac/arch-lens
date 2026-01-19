package java

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/core"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// 预编译正则，提高泛型解析效率
var genericRegex = regexp.MustCompile(`<([^>]+)>`)

// Extractor 负责 Java 源代码的依赖关系提取，支持层级、结构、动作及数据流特征提取。
type Extractor struct{}

// NewJavaExtractor 创建一个新的 Java 提取器实例。
func NewJavaExtractor() *Extractor {
	return &Extractor{}
}

// JavaActionQuery 定义了 Tree-sitter 查询语句，用于捕获动态动作（方法调用、赋值、对象创建等）。
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

// =============================================================================
// 主入口逻辑
// =============================================================================

// Extract 是提取器的核心入口，协调并整合三类依赖关系的提取。
func (e *Extractor) Extract(filePath string, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gCtx.FileContexts[filePath]
	if !ok {
		return nil, fmt.Errorf("file context not found: %s", filePath)
	}

	var allRelations []*model.DependencyRelation

	// 1. 提取层级关系 (IMPORT, CONTAIN)
	allRelations = append(allRelations, e.extractHierarchy(fCtx, gCtx)...)

	// 2. 提取静态结构关系 (EXTEND, IMPLEMENT, ANNOTATION, PARAMETER, RETURN, THROW, TYPE_ARG)
	allRelations = append(allRelations, e.extractStructural(fCtx, gCtx)...)

	// 3. 提取动态动作关系 (CALL, CREATE, USE, CAST, ASSIGN, CAPTURE)
	actionRels, err := e.extractActions(fCtx, gCtx)
	if err != nil {
		return nil, err
	}
	allRelations = append(allRelations, actionRels...)

	return allRelations, nil
}

// =============================================================================
// 1. 层级处理器 (Hierarchy)
// =============================================================================

func (e *Extractor) extractHierarchy(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	fileElems, ok := gCtx.DefinitionsByQN[fCtx.FilePath]
	if !ok {
		return nil
	}
	fileSource := fileElems[0].Element

	// 处理导入关系
	for _, imports := range fCtx.Imports {
		for _, imp := range imports {
			target := e.quickResolve(imp.RawImportPath, imp.Kind, gCtx, fCtx)
			rels = append(rels, &model.DependencyRelation{
				Type: model.Import, Source: fileSource, Target: target, Location: imp.Location,
			})
		}
	}

	// 处理包含关系：构建代码元素的树形层级
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			if entry.ParentQN == "" {
				continue
			}
			if parents, ok := gCtx.DefinitionsByQN[entry.ParentQN]; ok && len(parents) > 0 {
				for _, parent := range parents {
					rels = append(rels, &model.DependencyRelation{
						Type: model.Contain, Source: parent.Element, Target: entry.Element,
					})
				}
			}
		}
	}
	return rels
}

// =============================================================================
// 2. 结构处理器 (Structural)
// =============================================================================

func (e *Extractor) extractStructural(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			elem := entry.Element
			if elem.Extra == nil || elem.Extra.Mores == nil {
				continue
			}
			mores := elem.Extra.Mores

			// 2.1 注解关系
			for _, annoStr := range elem.Extra.Annotations {
				rels = append(rels, e.createAnnotationRelation(elem, annoStr, fCtx, gCtx))
			}

			// 2.2 继承与实现
			if sc, ok := mores[ClassSuperClass].(string); ok && sc != "" {
				rels = append(rels, &model.DependencyRelation{
					Type: model.Extend, Source: elem, Target: e.quickResolve(e.clean(sc), model.Class, gCtx, fCtx),
				})
			}
			if ifaces, ok := mores[ClassImplementedInterfaces].([]string); ok {
				for _, iface := range ifaces {
					relType := model.Implement
					if elem.Kind == model.Interface {
						relType = model.Extend
					}
					rels = append(rels, &model.DependencyRelation{
						Type: relType, Source: elem, Target: e.quickResolve(e.clean(iface), model.Interface, gCtx, fCtx),
					})
				}
			}

			// 2.3 方法特征 (返回、参数、异常)
			if ret, ok := mores[MethodReturnType].(string); ok && ret != "void" {
				rels = append(rels, &model.DependencyRelation{
					Type: model.Return, Source: elem, Target: e.quickResolve(e.clean(ret), model.Type, gCtx, fCtx),
				})
			}
			if params, ok := mores[MethodParameters].([]string); ok {
				for _, p := range params {
					if pureType := e.extractPureType(p); pureType != "" {
						rels = append(rels, &model.DependencyRelation{
							Type: model.Parameter, Source: elem, Target: e.quickResolve(e.clean(pureType), model.Type, gCtx, fCtx),
						})
					}
				}
			}
			if throws, ok := mores[MethodThrowsTypes].([]string); ok {
				for _, t := range throws {
					rels = append(rels, &model.DependencyRelation{
						Type: model.Throw, Source: elem, Target: e.quickResolve(e.clean(t), model.Class, gCtx, fCtx),
					})
				}
			}

			// 2.4 泛型参数 (TYPE_ARG)
			rels = append(rels, e.extractGenericTypeArgs(elem, mores, fCtx, gCtx)...)
		}
	}
	return rels
}

// =============================================================================
// 3. 动作处理器 (Action & Metadata Enrichment)
// =============================================================================

func (e *Extractor) extractActions(fCtx *core.FileContext, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
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

		stmtNode := &match.Captures[0].Node
		sourceElem := e.determinePreciseSource(stmtNode, fCtx, gCtx)
		if sourceElem == nil {
			continue
		}

		for _, cap := range match.Captures {
			capName := q.CaptureNames()[cap.Index]
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
				},
			}

			// 处理 Lambda Capture
			if rel.Type == model.Use && strings.Contains(sourceElem.QualifiedName, "lambda$") {
				if e.isCaptured(target, sourceElem) {
					rel.Type = model.Capture
				}
			}

			// 元数据增强分发
			switch rel.Type {
			case model.Assign:
				e.enrichAssignMetadata(rel, &cap.Node, fCtx)
			case model.Call, model.Create:
				e.enrichCallMetadata(rel, &cap.Node, fCtx, gCtx)
			}

			rels = append(rels, rel)
		}
	}
	return rels, nil
}

// enrichAssignMetadata 增强赋值关系的元数据（操作符、去壳后的右值表达式等）。
func (e *Extractor) enrichAssignMetadata(rel *model.DependencyRelation, capNode *sitter.Node, fCtx *core.FileContext) {
	rel.Mores[RelAssignTargetName] = capNode.Utf8Text(*fCtx.SourceBytes)

	for curr := capNode.Parent(); curr != nil; curr = curr.Parent() {
		kind := curr.Kind()

		if strings.Contains(rel.Source.QualifiedName, "$static$") {
			rel.Mores[RelAssignIsStaticContext] = true
		}

		switch kind {
		case "assignment_expression":
			rel.Mores[RelAstKind] = kind
			if opNode := curr.ChildByFieldName("operator"); opNode != nil {
				op := opNode.Utf8Text(*fCtx.SourceBytes)
				rel.Mores[RelAssignOperator] = op
				rel.Mores[RelAssignIsCompound] = (op != "=")
			}

			if right := curr.ChildByFieldName("right"); right != nil {
				// 【赋值顺序关键】：先赋默认值，再通过数据流分析进行“去壳”覆盖
				rel.Mores[RelAssignValueExpression] = right.Utf8Text(*fCtx.SourceBytes)
				e.analyzeDataFlow(rel, right, fCtx)

				// 处理链式赋值
				for finalVal := right; finalVal.Kind() == "assignment_expression"; {
					rel.Mores[RelAssignIsChained] = true
					finalVal = finalVal.ChildByFieldName("right")
					e.analyzeDataFlow(rel, finalVal, fCtx)
					rel.Mores[RelAssignValueExpression] = finalVal.Utf8Text(*fCtx.SourceBytes)
				}
			}

			// 处理 Receiver 和 索引
			if left := curr.ChildByFieldName("left"); left != nil {
				if left.Kind() == "field_access" {
					if obj := left.ChildByFieldName("object"); obj != nil {
						rel.Mores[RelAssignReceiver] = obj.Utf8Text(*fCtx.SourceBytes)
					}
				} else if left.Kind() == "array_access" {
					if idx := left.ChildByFieldName("index"); idx != nil {
						rel.Mores[RelAssignIndexExpression] = idx.Utf8Text(*fCtx.SourceBytes)
					}
				}
			}
			return

		case "variable_declarator":
			rel.Mores[RelAstKind] = kind
			rel.Mores[RelAssignIsInitializer] = true
			rel.Mores[RelAssignOperator] = "="
			if val := curr.ChildByFieldName("value"); val != nil {
				rel.Mores[RelAssignValueExpression] = val.Utf8Text(*fCtx.SourceBytes)
				e.analyzeDataFlow(rel, val, fCtx)
			}
			return

		case "update_expression":
			e.enrichUpdateMetadata(rel, curr, fCtx)
			return
		}

		if kind == "expression_statement" || kind == "local_variable_declaration" {
			break
		}
	}
}

// enrichCallMetadata 增强调用关系的元数据（Receiver、泛型参数、链式调用等）。
func (e *Extractor) enrichCallMetadata(rel *model.DependencyRelation, capNode *sitter.Node, fCtx *core.FileContext, gCtx *core.GlobalContext) {
	callNode := e.findNearestCallContainer(capNode)
	if callNode == nil {
		return
	}

	kind := callNode.Kind()
	rel.Mores[RelAstKind] = kind

	switch kind {
	case "method_invocation":
		receiver := callNode.ChildByFieldName("object")
		if receiver != nil {
			text := receiver.Utf8Text(*fCtx.SourceBytes)
			rel.Mores[RelCallReceiver] = text
			rel.Mores[RelCallIsChained] = (receiver.Kind() == "method_invocation")
			if len(text) > 0 && text[0] >= 'A' && text[0] <= 'Z' {
				rel.Mores[RelCallIsStatic] = true
				rel.Mores[RelCallReceiverType] = text
			}
		} else {
			rel.Mores[RelCallReceiver] = "this"
			rel.Mores[RelCallIsStatic] = false
		}
		if rel.Mores[RelCallReceiver] == "super" {
			rel.Mores[RelCallIsInherited] = true
		}
		if typeArgs := callNode.ChildByFieldName("type_arguments"); typeArgs != nil {
			rel.Mores[RelCallTypeArguments] = strings.Trim(typeArgs.Utf8Text(*fCtx.SourceBytes), "<>")
		}

	case "object_creation_expression":
		rel.Mores[RelCallIsConstructor] = true
		if typeNode := callNode.ChildByFieldName("type"); typeNode != nil && typeNode.Kind() == "generic_type" {
			if typeArgs := typeNode.ChildByFieldName("type_arguments"); typeArgs != nil {
				rel.Mores[RelCallTypeArguments] = strings.Trim(typeArgs.Utf8Text(*fCtx.SourceBytes), "<>")
			}
		}

	case "method_reference":
		rel.Mores[RelCallIsFunctional] = true
		if obj := callNode.Child(0); obj != nil {
			rel.Mores[RelCallReceiver] = obj.Utf8Text(*fCtx.SourceBytes)
		}

	case "explicit_constructor_invocation":
		rel.Mores[RelCallReceiver] = capNode.Utf8Text(*fCtx.SourceBytes) // super 或 this
	}

	// 补充 Lambda 容器信息
	if strings.Contains(rel.Source.QualifiedName, "lambda$") {
		if parts := strings.Split(rel.Source.QualifiedName, "."); len(parts) >= 2 {
			rel.Mores[RelCallEnclosingMethod] = parts[len(parts)-2]
		}
	}
}

// =============================================================================
// 辅助解析工具 (Internal Helpers)
// =============================================================================

// analyzeDataFlow 分析表达式节点的数据流特征（常量、返回值、Cast 去壳）。
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
		if typeNode := node.ChildByFieldName("type"); typeNode != nil {
			rel.Mores[RelAssignCastType] = typeNode.Utf8Text(*fCtx.SourceBytes)
		}
		// 【去壳关键点】：识别到 Cast 后，覆盖外层的 ValueExpression 指向真实值
		if valueNode := node.ChildByFieldName("value"); valueNode != nil {
			rel.Mores[RelAssignValueExpression] = valueNode.Utf8Text(*fCtx.SourceBytes)
		}
	}
}

// determinePreciseSource 根据 AST 节点位置向上溯源，寻找精确的动作发起者（方法、Lambda、静态块等）。
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

// 内部逻辑拆分辅助函数

func (e *Extractor) extractPureType(paramStr string) string {
	parts := strings.Fields(paramStr)
	for _, part := range parts {
		if !strings.HasPrefix(part, "@") {
			return part
		}
	}
	return ""
}

func (e *Extractor) extractGenericTypeArgs(elem *model.CodeElement, mores map[string]interface{}, fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	rawTypes := []string{}
	if rt, ok := mores[MethodReturnType].(string); ok {
		rawTypes = append(rawTypes, rt)
	}
	if pt, ok := mores[MethodParameters].([]string); ok {
		rawTypes = append(rawTypes, pt...)
	}
	if vt, ok := mores[VariableType].(string); ok {
		rawTypes = append(rawTypes, vt)
	}

	for _, rt := range rawTypes {
		for _, arg := range e.extractTypeArgs(rt) {
			rels = append(rels, &model.DependencyRelation{
				Type: model.TypeArg, Source: elem, Target: e.quickResolve(arg, model.Class, gCtx, fCtx),
			})
		}
	}
	return rels
}

func (e *Extractor) enrichUpdateMetadata(rel *model.DependencyRelation, node *sitter.Node, fCtx *core.FileContext) {
	rel.Mores[RelAstKind] = "update_expression"
	raw := node.Utf8Text(*fCtx.SourceBytes)
	if strings.Contains(raw, "++") {
		rel.Mores[RelAssignOperator] = "++"
	} else {
		rel.Mores[RelAssignOperator] = "--"
	}
	rel.Mores[RelAssignIsPostfix] = strings.HasSuffix(raw, "++") || strings.HasSuffix(raw, "--")
	rel.Mores[RelAssignIsPrefix] = !rel.Mores[RelAssignIsPostfix].(bool)
}

func (e *Extractor) findNearestCallContainer(n *sitter.Node) *sitter.Node {
	for curr := n.Parent(); curr != nil; curr = curr.Parent() {
		k := curr.Kind()
		if k == "method_invocation" || k == "method_reference" ||
			k == "object_creation_expression" || k == "explicit_constructor_invocation" {
			return curr
		}
	}
	return nil
}

func (e *Extractor) mapAction(capName string) (model.DependencyType, model.ElementKind) {
	switch capName {
	case "call_target", "ref_target", "explicit_constructor_stmt":
		return model.Call, model.Method
	case "create_target":
		return model.Create, model.Class
	case "use_field_target":
		return model.Use, model.Field
	case "assign_target":
		return model.Assign, model.Variable
	case "cast_target":
		return model.Cast, model.Class
	}
	return "", model.Unknown
}

func (e *Extractor) createAnnotationRelation(source *model.CodeElement, rawAnno string, fCtx *core.FileContext, gCtx *core.GlobalContext) *model.DependencyRelation {
	targetName, mores := e.parseAnnotationString(rawAnno)
	mores[RelAnnotationTarget] = e.determineAnnotationTarget(source)
	return &model.DependencyRelation{
		Type: model.Annotation, Source: source,
		Target: e.quickResolve(targetName, model.KAnnotation, gCtx, fCtx),
		Mores:  mores,
	}
}

func (e *Extractor) parseAnnotationString(raw string) (string, map[string]interface{}) {
	mores := map[string]interface{}{RelRawText: raw}
	content := strings.TrimPrefix(raw, "@")
	if idx := strings.Index(content, "("); idx != -1 {
		name := content[:idx]
		params := content[idx+1 : strings.LastIndex(content, ")")]
		if strings.Contains(params, "=") {
			mores[RelAnnotationParams] = params
		} else {
			mores[RelAnnotationValue] = params
		}
		return name, mores
	}
	return content, mores
}

func (e *Extractor) determineAnnotationTarget(elem *model.CodeElement) string {
	switch elem.Kind {
	case model.Class, model.Interface, model.Enum:
		return "TYPE"
	case model.Field:
		return "FIELD"
	case model.Method:
		return "METHOD"
	case model.Variable:
		if isParam, _ := elem.Extra.Mores[VariableIsParam].(bool); isParam {
			return "PARAMETER"
		}
		return "LOCAL_VARIABLE"
	}
	return "UNKNOWN"
}

func (e *Extractor) extractTypeArgs(signature string) []string {
	match := genericRegex.FindStringSubmatch(signature)
	if len(match) < 2 {
		return nil
	}
	args := strings.Split(match[1], ",")
	for i := range args {
		args[i] = e.clean(args[i])
	}
	return args
}

func (e *Extractor) isCaptured(target *model.CodeElement, source *model.CodeElement) bool {
	if target.Kind != model.Variable {
		return false
	}
	return !strings.HasPrefix(target.QualifiedName, source.QualifiedName)
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
