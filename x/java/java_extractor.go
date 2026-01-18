package java

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/core"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

var genericRegex = regexp.MustCompile(`<([^>]+)>`)

type Extractor struct{}

func NewJavaExtractor() *Extractor {
	return &Extractor{}
}

// JavaActionQuery 涵盖动作捕获。增强逻辑：
// 1. 捕获赋值表达式 (assignment_expression) 的左值。
// 2. 捕获变量声明初始化 (variable_declarator)。
// 3. 捕获一元更新表达式 (update_expression)。
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
  
  ; --- ASSIGN 增强部分 ---
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

// Extract 是提取器的入口，协调组织、结构和动作三类关系的提取。
func (e *Extractor) Extract(filePath string, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gCtx.FileContexts[filePath]
	if !ok {
		return nil, fmt.Errorf("file context not found: %s", filePath)
	}

	var relations []*model.DependencyRelation

	// 1. 组织与层级 (IMPORT, CONTAIN)
	relations = append(relations, e.extractHierarchyRelations(fCtx, gCtx)...)

	// 2. 静态结构 (EXTEND, IMPLEMENT, ANNOTATION, PARAMETER, RETURN, THROW, TYPE_ARG)
	relations = append(relations, e.extractStructuralRelations(fCtx, gCtx)...)

	// 3. 动态动作 (CALL, CREATE, USE, CAST, ASSIGN, CAPTURE)
	actionRels, err := e.extractActionRelations(fCtx, gCtx)
	if err != nil {
		return nil, err
	}
	relations = append(relations, actionRels...)

	return relations, nil
}

// =============================================================================
// 1. 层级关系提取 (Hierarchy)
// =============================================================================

func (e *Extractor) extractHierarchyRelations(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	fileElems, ok := gCtx.DefinitionsByQN[fCtx.FilePath]
	if !ok {
		return nil
	}
	fileSource := fileElems[0].Element

	// 处理 Import 关系
	for _, imports := range fCtx.Imports {
		for _, imp := range imports {
			target := e.quickResolve(imp.RawImportPath, imp.Kind, gCtx, fCtx)
			rels = append(rels, &model.DependencyRelation{
				Type: model.Import, Source: fileSource, Target: target, Location: imp.Location,
			})
		}
	}

	// 处理 Contain 关系 (建立父子树形结构)
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
// 2. 结构关系提取 (Structural)
// =============================================================================

func (e *Extractor) extractStructuralRelations(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	for _, entries := range fCtx.DefinitionsBySN {
		for _, entry := range entries {
			elem := entry.Element
			if elem.Extra == nil || elem.Extra.Mores == nil {
				continue
			}
			mores := elem.Extra.Mores

			// ANNOTATION
			for _, annoStr := range elem.Extra.Annotations {
				rels = append(rels, e.createAnnotationRelation(elem, annoStr, fCtx, gCtx))
			}

			// EXTEND
			if sc, ok := mores[ClassSuperClass].(string); ok && sc != "" {
				rels = append(rels, &model.DependencyRelation{
					Type: model.Extend, Source: elem, Target: e.quickResolve(e.clean(sc), model.Class, gCtx, fCtx),
				})
			}

			// IMPLEMENT / EXTEND (Interface)
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

			// RETURN
			if ret, ok := mores[MethodReturnType].(string); ok && ret != "void" {
				rels = append(rels, &model.DependencyRelation{
					Type: model.Return, Source: elem, Target: e.quickResolve(e.clean(ret), model.Type, gCtx, fCtx),
				})
			}

			// PARAMETER
			if params, ok := mores[MethodParameters].([]string); ok {
				for _, p := range params {
					parts := strings.Fields(p)
					var pureType string
					for _, part := range parts {
						if !strings.HasPrefix(part, "@") {
							pureType = part
							break
						}
					}
					if pureType != "" {
						rels = append(rels, &model.DependencyRelation{
							Type: model.Parameter, Source: elem, Target: e.quickResolve(e.clean(pureType), model.Type, gCtx, fCtx),
						})
					}
				}
			}

			// THROW
			if throws, ok := mores[MethodThrowsTypes].([]string); ok {
				for _, t := range throws {
					rels = append(rels, &model.DependencyRelation{
						Type: model.Throw, Source: elem, Target: e.quickResolve(e.clean(t), model.Class, gCtx, fCtx),
					})
				}
			}

			// TYPE_ARG (泛型参数)
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
		}
	}
	return rels
}

// =============================================================================
// 3. 动作关系提取 (Action & Capture)
// =============================================================================

func (e *Extractor) extractActionRelations(fCtx *core.FileContext, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
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

		// 第一个 capture 为整个语句块 (stmt)
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

			rawText := cap.Node.Utf8Text(*fCtx.SourceBytes)
			relType, kind := e.mapAction(capName)
			if relType == "" {
				continue
			}

			target := e.quickResolve(e.clean(rawText), kind, gCtx, fCtx)

			// 每一条关系必须拥有独立的 Mores 防止并发或指针引用问题
			mores := make(map[string]interface{})
			mores[RelRawText] = stmtNode.Utf8Text(*fCtx.SourceBytes)
			mores[RelContext] = stmtNode.Kind()

			rel := &model.DependencyRelation{
				Type:     relType,
				Source:   sourceElem,
				Target:   target,
				Location: e.toLoc(cap.Node, fCtx.FilePath),
				Mores:    mores,
			}

			// CAPTURE 判定：在 Lambda 内部引用了外部变量
			if relType == model.Use && strings.Contains(sourceElem.QualifiedName, "lambda$") {
				if e.isCaptured(target, sourceElem) {
					rel.Type = model.Capture
				}
			}

			// 补充 ASSIGN 关系的元数据
			if relType == model.Assign {
				e.enrichAssignMetadata(rel, &cap.Node, fCtx)
			}

			rels = append(rels, rel)
		}
	}
	return rels, nil
}

// enrichAssignMetadata 向上回溯 AST 以填充赋值细节 (操作符、右值、Receiver、链式赋值等)。
func (e *Extractor) enrichAssignMetadata(rel *model.DependencyRelation, capNode *sitter.Node, fCtx *core.FileContext) {
	rel.Mores[RelAssignTargetName] = capNode.Utf8Text(*fCtx.SourceBytes)

	for curr := capNode.Parent(); curr != nil; curr = curr.Parent() {
		kind := curr.Kind()

		// 判定是否处于静态初始化块
		if strings.Contains(rel.Source.QualifiedName, "$static$") {
			rel.Mores[RelAssignIsStaticContext] = true
		}

		switch kind {
		case "assignment_expression":
			rel.Mores[RelAstKind] = kind
			// 操作符与复合赋值
			if opNode := curr.ChildByFieldName("operator"); opNode != nil {
				op := opNode.Utf8Text(*fCtx.SourceBytes)
				rel.Mores[RelAssignOperator] = op
				if op != "=" {
					rel.Mores[RelAssignIsCompound] = true
				}
			}
			// 右值与链式赋值
			right := curr.ChildByFieldName("right")
			if right != nil {
				rel.Mores[RelAssignValueExpression] = right.Utf8Text(*fCtx.SourceBytes)
				if right.Kind() == "assignment_expression" {
					rel.Mores[RelAssignIsChained] = true
				}
			}
			// 左值细节：处理 this.field 或 arr[i]
			left := curr.ChildByFieldName("left")
			if left != nil {
				if left.Kind() == "field_access" {
					if obj := left.ChildByFieldName("object"); obj != nil {
						rel.Mores[RelCallReceiver] = obj.Utf8Text(*fCtx.SourceBytes)
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
			}
			return

		case "update_expression":
			rel.Mores[RelAstKind] = kind
			raw := curr.Utf8Text(*fCtx.SourceBytes)
			if strings.Contains(raw, "++") {
				rel.Mores[RelAssignOperator] = "++"
			} else {
				rel.Mores[RelAssignOperator] = "--"
			}
			// 判定前置还是后置 (i++ vs ++i)
			if strings.HasSuffix(raw, "++") || strings.HasSuffix(raw, "--") {
				rel.Mores[RelAssignIsPostfix] = true
			}
			return
		}

		// 边界：超出赋值语句的作用域
		if kind == "expression_statement" || kind == "local_variable_declaration" {
			break
		}
	}
}

// =============================================================================
// 4. 辅助解析工具
// =============================================================================

// determinePreciseSource 确定触发动作的精确 Source (Method, Lambda, Field 等)
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

func (e *Extractor) createAnnotationRelation(source *model.CodeElement, rawAnno string, fCtx *core.FileContext, gCtx *core.GlobalContext) *model.DependencyRelation {
	targetName, mores := e.parseAnnotationString(rawAnno)
	mores[RelAnnotationTarget] = e.determineAnnotationTarget(source)

	return &model.DependencyRelation{
		Type:   model.Annotation,
		Source: source,
		Target: e.quickResolve(targetName, model.KAnnotation, gCtx, fCtx),
		Mores:  mores,
	}
}

func (e *Extractor) parseAnnotationString(raw string) (string, map[string]interface{}) {
	mores := make(map[string]interface{})
	mores[RelRawText] = raw
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

func (e *Extractor) mapAction(capName string) (model.DependencyType, model.ElementKind) {
	switch capName {
	case "call_target", "ref_target":
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
	entries := gCtx.ResolveSymbol(fCtx, symbol)
	if len(entries) > 0 {
		return entries[0].Element
	}
	return &model.CodeElement{Name: symbol, QualifiedName: symbol, Kind: kind}
}
