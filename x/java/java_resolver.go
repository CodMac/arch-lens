package java

import (
	"slices"
	"strings"

	"github.com/CodMac/arch-lens/core"
	"github.com/CodMac/arch-lens/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type SymbolResolver struct{}

func NewJavaSymbolResolver() *SymbolResolver {
	return &SymbolResolver{}
}

// =============================================================================
// 1. 基础接口实现 (Basic Interface)
// =============================================================================

func (j *SymbolResolver) BuildQualifiedName(parentQN, name string) string {
	if parentQN == "" || parentQN == "." {
		return name
	}
	return parentQN + "." + name
}

func (j *SymbolResolver) RegisterPackage(gc *core.GlobalContext, packageName string) {
	parts := strings.Split(packageName, ".")
	var current []string
	for _, part := range parts {
		current = append(current, part)
		pkgQN := strings.Join(current, ".")
		if _, ok := gc.FindByQualifiedName(pkgQN); !ok {
			entry := core.DefinitionEntry{
				Element: &model.CodeElement{Kind: model.Package, Name: part, QualifiedName: pkgQN, IsFormSource: true},
			}
			gc.AddDefinition(&entry)
		}
	}
}

// Resolve 为外部统一入口
func (j *SymbolResolver) Resolve(gc *core.GlobalContext, fc *core.FileContext, node *sitter.Node, symbol string, kind model.ElementKind) *model.CodeElement {
	switch kind {
	case model.Variable:
		return j.resolveVariable(gc, fc, node, symbol)
	case model.Method:
		return j.resolveMethod(gc, fc, node, symbol)
	default:
		return j.resolveStructure(gc, fc, symbol, kind)
	}
}

// =============================================================================
// 2. 核心查找流程 (Core Resolution Flow)
// =============================================================================

// resolveVariable 处理变量查找，支持本地作用域回溯和类成员继承查找
func (j *SymbolResolver) resolveVariable(gc *core.GlobalContext, fc *core.FileContext, node *sitter.Node, symbol string) *model.CodeElement {
	cleanName := strings.TrimSpace(symbol)
	containerTypes := []model.ElementKind{model.Method, model.ScopeBlock, model.Lambda, model.Class, model.AnonymousClass}
	container := j.determinePreciseContainer(fc, node, containerTypes)
	if container == nil {
		return nil
	}

	isStatic := slices.Contains(container.Extra.Modifiers, "static")
	previousQN := container.QualifiedName
	return j.resolveInScopeHierarchy(gc, fc, previousQN, cleanName, isStatic, container)
}

// resolveMethod 处理方法查找，利用 Collector 的元数据进行重载过滤
func (j *SymbolResolver) resolveMethod(gc *core.GlobalContext, fc *core.FileContext, node *sitter.Node, symbol string) *model.CodeElement {
	cleanName := strings.TrimSpace(symbol)

	// 1. 启发式分析：识别调用处的实参个数与大致类型
	argCount := -1
	var inferredArgTypes []string
	if node != nil {
		if callNode := j.findInvocationNode(node); callNode != nil {
			if argsNode := callNode.ChildByFieldName("arguments"); argsNode != nil {
				argCount = int(argsNode.NamedChildCount())
				inferredArgTypes = j.inferArgumentTypes(argsNode, fc)
			}
		}
	}

	// 2. 确定起点：获取当前节点所在的容器类或方法
	container := j.determinePreciseContainer(fc, node, []model.ElementKind{model.Class, model.Method})
	var startQN string
	if container != nil {
		startQN = container.QualifiedName
	}

	// 3. 递归回溯：搜索本类 -> 继承链 -> 父级作用域
	// 这里直接复用层级查找逻辑
	methodElem := j.resolveInScopeHierarchy(gc, fc, startQN, cleanName, false, container)

	// 4. 重载匹配：如果在全局找到多个同名 entry，进行过滤
	if methodElem == nil || methodElem.IsFormExternal {
		entries := j.preciseResolve(gc, fc, cleanName)
		if len(entries) > 0 {
			return j.pickBestOverloadEnhanced(entries, argCount, inferredArgTypes)
		}
	} else {
		return methodElem
	}

	// 5. 兜底处理：尝试利用 Import 还原 QualifiedName
	qualifiedName := cleanName
	if imps, ok := fc.Imports[cleanName]; ok && len(imps) > 0 {
		qualifiedName = imps[0].RawImportPath
	}
	return &model.CodeElement{Name: cleanName, QualifiedName: qualifiedName, Kind: model.Method, IsFormExternal: true}
}

// resolveStructure 处理类、接口、包等结构性符号
func (j *SymbolResolver) resolveStructure(gc *core.GlobalContext, fc *core.FileContext, symbol string, kind model.ElementKind) *model.CodeElement {
	if entries := j.preciseResolve(gc, fc, symbol); len(entries) > 0 {
		return entries[0].Element
	}

	qualifiedName := symbol
	if imps, ok := fc.Imports[symbol]; ok && len(imps) > 0 {
		qualifiedName = imps[0].RawImportPath
	}
	return &model.CodeElement{Name: symbol, QualifiedName: qualifiedName, Kind: kind, IsFormExternal: true}
}

// =============================================================================
// 3. 递归查找逻辑 (Hierarchical Search)
// =============================================================================

// resolveInScopeHierarchy 递归向上查找容器及继承链
func (j *SymbolResolver) resolveInScopeHierarchy(gc *core.GlobalContext, fc *core.FileContext, previousQN, symbol string, isStatic bool, container *model.CodeElement) *model.CodeElement {
	if previousQN == "" {
		return nil
	}

	// 1. 尝试在当前层级直接匹配
	targetQN := j.BuildQualifiedName(previousQN, symbol)
	if entry, ok := gc.FindByQualifiedName(targetQN); ok {
		if j.checkVisibility(gc, fc, container, entry) {
			isIllegalStatic := isStatic && entry.Element.Kind == model.Field && !slices.Contains(entry.Element.Extra.Modifiers, "static")
			if !isIllegalStatic {
				return entry.Element
			}
		}
	}

	previousEntry, ok := gc.FindByQualifiedName(previousQN)
	if !ok {
		return nil
	}

	// 2. 如果是类/接口，递归查找其继承链 (extends/implements)
	previousEleKind := previousEntry.Element.Kind
	if previousEleKind == model.Class || previousEleKind == model.Interface || previousEleKind == model.AnonymousClass {
		if inherited := j.resolveFromInheritance(gc, fc, previousEntry.Element, symbol, isStatic, container); inherited != nil {
			return inherited
		}
	}

	// 3. 递归到上一级 Lexical Scope
	return j.resolveInScopeHierarchy(gc, fc, previousEntry.ParentQN, symbol, isStatic, container)
}

// resolveFromInheritance 处理继承树查找
func (j *SymbolResolver) resolveFromInheritance(gc *core.GlobalContext, fc *core.FileContext, elem *model.CodeElement, symbol string, isStatic bool, sourceElem *model.CodeElement) *model.CodeElement {
	if elem.Extra == nil {
		return nil
	}

	var superTargets []string
	if sc, ok := elem.Extra.Mores[ClassSuperClass].(string); ok && sc != "" {
		superTargets = append(superTargets, sc)
	}
	if itfs, ok := elem.Extra.Mores[ClassImplementedInterfaces].([]string); ok {
		superTargets = append(superTargets, itfs...)
	}

	for _, rawSuperName := range superTargets {
		cleanSuperName := strings.Split(rawSuperName, "<")[0]
		parentEntries := j.preciseResolve(gc, fc, cleanSuperName)

		if len(parentEntries) > 0 {
			parentElem := parentEntries[0].Element
			targetQN := j.BuildQualifiedName(parentElem.QualifiedName, symbol)

			if fieldEntry, ok := gc.FindByQualifiedName(targetQN); ok {
				if j.checkVisibility(gc, fc, sourceElem, fieldEntry) {
					if !isStatic || slices.Contains(fieldEntry.Element.Extra.Modifiers, "static") {
						return fieldEntry.Element
					}
				}
			}
			// 深度优先递归父类的父类
			if found := j.resolveFromInheritance(gc, fc, parentElem, symbol, isStatic, sourceElem); found != nil {
				return found
			}
		}
	}
	return nil
}

// =============================================================================
// 4. 重载与类型匹配辅助 (Overload & Type Inference)
// =============================================================================

// pickBestOverloadEnhanced 结合参数数量和启发式类型匹配选择最优重载
func (j *SymbolResolver) pickBestOverloadEnhanced(entries []*core.DefinitionEntry, argCount int, inferredTypes []string) *model.CodeElement {
	var bestMatch *model.CodeElement
	maxScore := -1

	for _, entry := range entries {
		if entry.Element.Kind != model.Method {
			continue
		}

		params, ok := entry.Element.Extra.Mores[MethodParameters].([]string)
		if !ok {
			if bestMatch == nil {
				bestMatch = entry.Element
			}
			continue
		}

		currentScore := 0
		definedParamCount := len(params) / 2

		// 1. 参数数量匹配 (主权重)
		if definedParamCount == argCount {
			currentScore += 100

			// 2. 启发式类型匹配 (次权重)
			if argCount > 0 && len(inferredTypes) == argCount {
				for i := 0; i < argCount; i++ {
					definedRawType := params[i*2]
					erasedDefinedType := strings.Split(definedRawType, "<")[0] // 泛型擦除

					inferred := inferredTypes[i]
					if inferred == "unknown" {
						continue
					}

					if inferred == "null" {
						if !j.isPrimitive(erasedDefinedType) {
							currentScore += 5
						}
						continue
					}

					if j.typeMatches(erasedDefinedType, inferred) {
						currentScore += 20
					}
				}
			}
		}

		if currentScore > maxScore {
			maxScore = currentScore
			bestMatch = entry.Element
		}
	}

	if bestMatch != nil {
		return bestMatch
	}
	return entries[0].Element
}

// inferArgumentTypes 尝试从实参 AST 节点推断大致类型
func (j *SymbolResolver) inferArgumentTypes(argsNode *sitter.Node, fc *core.FileContext) []string {
	var types []string
	src := *fc.SourceBytes

	for i := 0; i < int(argsNode.NamedChildCount()); i++ {
		arg := argsNode.NamedChild(uint(i))
		kind := arg.Kind()

		switch kind {
		case "string_literal":
			types = append(types, "String")
		case "decimal_integer_literal", "hex_integer_literal":
			types = append(types, "int")
		case "decimal_floating_point_literal":
			types = append(types, "double")
		case "true", "false", "boolean_type":
			types = append(types, "boolean")
		case "null_literal":
			types = append(types, "null")
		case "object_creation_expression", "cast_expression":
			if typeNode := arg.ChildByFieldName("type"); typeNode != nil {
				types = append(types, j.getNodeContent(typeNode, src))
			} else {
				types = append(types, "unknown")
			}
		case "array_creation_expression":
			if typeNode := arg.ChildByFieldName("type"); typeNode != nil {
				types = append(types, j.getNodeContent(typeNode, src)+"[]")
			} else {
				types = append(types, "unknown")
			}
		default:
			types = append(types, "unknown")
		}
	}
	return types
}

// =============================================================================
// 5. 校验与底层工具 (Utilities)
// =============================================================================

func (j *SymbolResolver) checkVisibility(gc *core.GlobalContext, fc *core.FileContext, container *model.CodeElement, target *core.DefinitionEntry) bool {
	// 1. 局部变量/形参/Lambda参数无限制
	if target.Element.Kind == model.Variable {
		return true
	}

	// 2. 检查是否属于同一个顶层类 (处理内部类、匿名类)
	containerOutermost := j.getOutermostClassQN(container.QualifiedName)
	targetOutermost := j.getOutermostClassQN(target.Element.QualifiedName)
	if containerOutermost != "" && containerOutermost == targetOutermost {
		return true
	}

	// 3. 显式修饰符判断
	mods := target.Element.Extra.Modifiers
	if slices.Contains(mods, "public") {
		return true
	}

	// 4. 包级私有 (Default/Package-Private) 判定
	// 注意：getPackageFromQN 应该确保拿到真正的 Java Package 名
	targetPkg := j.getRealJavaPackage(target.Element.QualifiedName, gc)
	if targetPkg == fc.PackageName {
		return true
	}

	// 5. Protected: 检查子类关系
	if slices.Contains(mods, "protected") {
		sourceClass := j.getOwnerClassQN(gc, container)
		return j.isSubClassOf(gc, fc, sourceClass, target.ParentQN)
	}

	return false
}

func (j *SymbolResolver) typeMatches(defined, inferred string) bool {
	if defined == inferred {
		return true
	}
	// 处理基础类型包装类
	if (defined == "Integer" && inferred == "int") || (defined == "int" && inferred == "Integer") {
		return true
	}
	// 简单的全限定名后缀匹配
	if strings.HasSuffix(defined, inferred) {
		return true
	}
	return false
}

func (j *SymbolResolver) isPrimitive(t string) bool {
	switch t {
	case "int", "long", "short", "byte", "char", "boolean", "float", "double":
		return true
	}
	return false
}

func (j *SymbolResolver) preciseResolve(gc *core.GlobalContext, fc *core.FileContext, symbol string) []*core.DefinitionEntry {
	gc.RLock()
	defer gc.RUnlock()

	if defs, ok := fc.FindByShortName(symbol); ok {
		return defs
	}
	if imps, ok := fc.Imports[symbol]; ok {
		for _, imp := range imps {
			if def, found := gc.FindByQualifiedName(imp.RawImportPath); found {
				return []*core.DefinitionEntry{def}
			}
		}
	}
	pkgQN := j.BuildQualifiedName(fc.PackageName, symbol)
	if def, ok := gc.FindByQualifiedName(pkgQN); ok {
		return []*core.DefinitionEntry{def}
	}

	for _, imps := range fc.Imports {
		for _, imp := range imps {
			if imp.IsWildcard {
				basePath := strings.TrimSuffix(imp.RawImportPath, "*")
				if def, ok := gc.FindByQualifiedName(basePath + symbol); ok {
					return []*core.DefinitionEntry{def}
				}
			}
		}
	}
	if def, ok := gc.FindByQualifiedName(symbol); ok {
		return []*core.DefinitionEntry{def}
	}
	return nil
}

func (j *SymbolResolver) determinePreciseContainer(fc *core.FileContext, n *sitter.Node, kinds []model.ElementKind) *model.CodeElement {
	if n == nil {
		return nil
	}
	var best *model.CodeElement
	var minSize uint32 = 0xFFFFFFFF
	row := int(n.StartPosition().Row + 1)
	for _, entry := range fc.Definitions {
		if slices.Contains(kinds, entry.Element.Kind) {
			if row >= entry.Element.Location.StartLine && row <= entry.Element.Location.EndLine {
				size := uint32(entry.Element.Location.EndLine - entry.Element.Location.StartLine)
				if size < minSize {
					minSize, best = size, entry.Element
				}
			}
		}
	}
	return best
}

func (j *SymbolResolver) getOwnerClassQN(gc *core.GlobalContext, elem *model.CodeElement) string {
	curr := elem
	for curr != nil {
		if curr.Kind == model.Class || curr.Kind == model.Interface {
			return curr.QualifiedName
		}
		if entry, ok := gc.FindByQualifiedName(curr.QualifiedName); ok && entry.ParentQN != "" {
			if next, ok := gc.FindByQualifiedName(entry.ParentQN); ok {
				curr = next.Element
				continue
			}
		}
		break
	}
	return ""
}

// 获取最外层的类名 (例如把 A.B.C$1 还原为 A)
func (j *SymbolResolver) getOutermostClassQN(qn string) string {
	// 逻辑：在 Java 中，类名通常是大写开头
	parts := strings.Split(qn, ".")
	for i, part := range parts {
		// 简单判定：首字母大写通常是类名 (Java 规范)
		if len(part) > 0 && part[0] >= 'A' && part[0] <= 'Z' {
			return strings.Join(parts[:i+1], ".")
		}
	}
	return ""
}

// 从 QN 中剥离出真实的 Package
func (j *SymbolResolver) getRealJavaPackage(qn string, gc *core.GlobalContext) string {
	curr := qn
	for {
		idx := strings.LastIndex(curr, ".")
		if idx == -1 {
			return ""
		}
		curr = curr[:idx]

		if entry, ok := gc.FindByQualifiedName(curr); ok {
			if entry.Element.Kind == model.Package {
				return curr
			}
		} else {
			// 如果全局上下文没找到，继续向上找，直到匹配已知的 Package 模式
			continue
		}
	}
}

func (j *SymbolResolver) isSubClassOf(gc *core.GlobalContext, fc *core.FileContext, sub, super string) bool {
	if sub == "" || super == "" || sub == super {
		return sub == super
	}
	entry, ok := gc.FindByQualifiedName(sub)
	if !ok || entry.Element.Extra == nil {
		return false
	}
	if sc, ok := entry.Element.Extra.Mores[ClassSuperClass].(string); ok && sc != "" {
		parents := j.preciseResolve(gc, fc, strings.Split(sc, "<")[0])
		for _, p := range parents {
			if p.Element.QualifiedName == super || j.isSubClassOf(gc, fc, p.Element.QualifiedName, super) {
				return true
			}
		}
	}
	return false
}

func (j *SymbolResolver) findInvocationNode(n *sitter.Node) *sitter.Node {
	for curr := n; curr != nil; curr = curr.Parent() {
		k := curr.Kind()
		if k == "method_invocation" || k == "object_creation_expression" || k == "explicit_constructor_invocation" {
			return curr
		}
		if strings.HasSuffix(k, "_statement") {
			break
		}
	}
	return nil
}

func (j *SymbolResolver) getNodeContent(n *sitter.Node, src []byte) string {
	return strings.TrimSpace(string(src[n.StartByte():n.EndByte()]))
}
