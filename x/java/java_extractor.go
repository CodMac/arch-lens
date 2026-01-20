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
// 1. 主入口流水线 (Core Only)
// =============================================================================

func (e *Extractor) Extract(filePath string, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gCtx.FileContexts[filePath]
	if !ok {
		return nil, fmt.Errorf("file context not found: %s", filePath)
	}

	var allRels []*model.DependencyRelation

	// 阶段 1: 静态结构 (Hierarchy & Structural)
	allRels = append(allRels, e.extractHierarchy(fCtx, gCtx)...)
	allRels = append(allRels, e.extractStructural(fCtx, gCtx)...)

	// 阶段 2: 动作发现 (Discovery)
	actionRels, err := e.discoverActionRelations(fCtx, gCtx)
	if err != nil {
		return nil, err
	}

	// 阶段 3: 核心元数据增强 (Core Enrichment)
	for _, rel := range actionRels {
		e.enrichCoreMetadata(rel, fCtx)
	}

	allRels = append(allRels, actionRels...)
	return allRels, nil
}

// =============================================================================
// 2. 核心关系发现 (Discovery)
// =============================================================================

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
			rel := &model.DependencyRelation{
				Type:     relType,
				Source:   sourceElem,
				Target:   e.quickResolve(e.clean(rawText), kind, gCtx, fCtx),
				Location: e.toLoc(cap.Node, fCtx.FilePath),
				Mores: map[string]interface{}{
					RelRawText: stmtNode.Utf8Text(*fCtx.SourceBytes),
					RelContext: stmtNode.Kind(),
					"tmp_node": &cap.Node, // 暂存用于 Enrich
					"tmp_stmt": stmtNode,
				},
			}
			rels = append(rels, rel)
		}
	}
	return rels, nil
}

// =============================================================================
// 3. 核心元数据增强 (Core Enrichment)
// =============================================================================

func (e *Extractor) enrichCoreMetadata(rel *model.DependencyRelation, fCtx *core.FileContext) {
	node, _ := rel.Mores["tmp_node"].(*sitter.Node)
	stmt, _ := rel.Mores["tmp_stmt"].(*sitter.Node)
	delete(rel.Mores, "tmp_node")
	delete(rel.Mores, "tmp_stmt")

	if node == nil || stmt == nil {
		return
	}

	source := *fCtx.SourceBytes

	// 统一环境检查：如果动作发生在 throw 语句中，提拔为 THROW 关系
	if e.isWithinThrow(stmt) {
		rel.Type = model.Throw
	}

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

func (e *Extractor) enrichCallCore(rel *model.DependencyRelation, capNode *sitter.Node, src []byte, fCtx *core.FileContext) {
	callNode := e.findNearestKind(capNode, "method_invocation", "method_reference", "explicit_constructor_invocation")
	if callNode == nil {
		return
	}
	rel.Mores[RelAstKind] = callNode.Kind()

	if obj := callNode.ChildByFieldName("object"); obj != nil {
		recText := obj.Utf8Text(src)
		rel.Mores[RelCallReceiver] = recText
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

func (e *Extractor) enrichThrowCore(rel *model.DependencyRelation, capNode, stmtNode *sitter.Node, src []byte) {
	// 如果是通过 Create 提拔过来的
	if capNode.Kind() == "type_identifier" || capNode.Parent().Kind() == "object_creation_expression" {
		rel.Mores[RelThrowIsRuntime] = true
		rel.Mores[RelAstKind] = "throw_statement"
	} else if capNode.Kind() == "identifier" {
		// throw e;
		rel.Mores[RelThrowIsRethrow] = true
		rel.Mores[RelAstKind] = "throw_statement"
	}
}

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

func (e *Extractor) enrichUseCore(rel *model.DependencyRelation, capNode *sitter.Node, src []byte) {
	if fieldAccess := e.findNearestKind(capNode, "field_access"); fieldAccess != nil {
		if obj := fieldAccess.ChildByFieldName("object"); obj != nil {
			rel.Mores[RelUseReceiver] = obj.Utf8Text(src)
		}
	}
}

// =============================================================================
// 4. 工具函数 (静态关系与辅助)
// =============================================================================

func (e *Extractor) isWithinThrow(n *sitter.Node) bool {
	for curr := n; curr != nil; curr = curr.Parent() {
		if curr.Kind() == "throw_statement" {
			return true
		}
	}
	return false
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

			// ANNOTATION
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

			// PARAMETER
			if elem.Kind == model.Method {
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
						isVarargs := false
						if strings.HasSuffix(typePart, "...") {
							isVarargs = true
							typePart = strings.TrimSuffix(typePart, "...")
						}
						relMores := map[string]interface{}{
							RelParameterIndex: i,
							RelParameterName:  paramName,
						}
						if isVarargs {
							relMores[RelParameterIsVarargs] = true
						}
						if strings.Contains(rawParam, "final ") {
							relMores[RelParameterIsFinal] = true
						}
						if strings.Contains(rawParam, "@") {
							relMores[RelParameterHasAnnotation] = true
						}
						rels = append(rels, &model.DependencyRelation{
							Type:   model.Parameter,
							Source: elem,
							Target: e.quickResolve(e.clean(typePart), model.Class, gCtx, fCtx),
							Mores:  relMores,
						})
					}
				}
			}

			// Return
			if elem.Kind == model.Method {
				if rawReturnType, ok := elem.Extra.Mores[MethodReturnType].(string); ok && rawReturnType != "" && rawReturnType != "void" {
					dimensions := strings.Count(rawReturnType, "[]")
					isArray := dimensions > 0
					hasTypeArgs := strings.Contains(rawReturnType, "<")
					cleanType := e.clean(rawReturnType)
					cleanType = strings.ReplaceAll(cleanType, "[]", "")
					relMores := map[string]interface{}{
						RelReturnIsPrimitive:      e.isPrimitive(cleanType),
						RelReturnIsArray:          isArray,
						RelReturnDimensions:       dimensions,
						RelReturnHasTypeArguments: hasTypeArgs,
					}
					rels = append(rels, &model.DependencyRelation{
						Type:     model.Return,
						Source:   elem,
						Target:   e.quickResolve(cleanType, model.Class, gCtx, fCtx),
						Location: elem.Location,
						Mores:    relMores,
					})
				}
			}

			// Throw Signature
			if elem.Kind == model.Method {
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
		}
	}
	return rels
}

// --- 基础辅助 ---

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
	case "throw_target":
		return model.Throw, model.Class
	}
	return "", model.Unknown
}

func (e *Extractor) quickResolve(symbol string, kind model.ElementKind, gCtx *core.GlobalContext, fCtx *core.FileContext) *model.CodeElement {
	if entries := gCtx.ResolveSymbol(fCtx, symbol); len(entries) > 0 {
		return entries[0].Element
	}
	return &model.CodeElement{Name: symbol, QualifiedName: symbol, Kind: kind}
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

func (e *Extractor) isPrimitive(typeName string) bool {
	switch typeName {
	case "int", "long", "short", "byte", "char", "boolean", "float", "double":
		return true
	}
	return false
}
