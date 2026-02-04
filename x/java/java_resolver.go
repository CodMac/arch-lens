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
// 1. 基础接口实现
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

// Resolve 是对外统一入口
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
// 2. 核心查找流程 (Variable, Method, Structure)
// =============================================================================

// resolveVariable 实现作用域回溯查找
func (j *SymbolResolver) resolveVariable(gc *core.GlobalContext, fc *core.FileContext, node *sitter.Node, name string) *model.CodeElement {
	cleanName := strings.TrimSpace(name)
	// 定义可以作为变量容器的类型
	containerTypes := []model.ElementKind{model.Class, model.AnonymousClass, model.Lambda, model.Method, model.ScopeBlock}
	container := j.determinePreciseContainer(fc, node, containerTypes)

	if container == nil {
		return j.resolveStructure(gc, fc, cleanName, model.Variable)
	}

	// 初始判定：是否处于静态上下文 (静态方法/静态块)
	isStatic := slices.Contains(container.Extra.Modifiers, "static")

	return j.resolveInScopeHierarchy(gc, fc, container.QualifiedName, cleanName, isStatic, container)
}

// resolveMethod 处理重载识别
func (j *SymbolResolver) resolveMethod(gc *core.GlobalContext, fc *core.FileContext, node *sitter.Node, symbol string) *model.CodeElement {
	cleanName := strings.TrimSpace(symbol)

	// 1. 识别调用处的实参个数 (Argument Count)
	argCount := -1
	if node != nil {
		if callNode := j.findInvocationNode(node); callNode != nil {
			if args := callNode.ChildByFieldName("arguments"); args != nil {
				argCount = int(args.NamedChildCount())
			}
		}
	}

	// 2. 获取当前上下文的容器，用于开启回溯
	container := j.determinePreciseContainer(fc, node, []model.ElementKind{model.Class, model.Method})
	var startQN string
	if container != nil {
		startQN = container.QualifiedName
	}

	// 3. 开启方法查找：当前类 -> 父类继承链 -> Import/静态导入
	method := j.resolveInScopeHierarchy(gc, fc, startQN, cleanName, false, container)

	// 如果回溯没找到（可能是本类/父类方法），退而求其次使用导入解析
	if method == nil || method.IsFormExternal {
		entries := j.preciseResolve(gc, fc, cleanName)
		if len(entries) > 0 {
			// 在匹配到的多个重载中，寻找参数数量最匹配的
			return j.pickBestOverload(entries, argCount)
		}
	} else {
		return method
	}

	// 4. 彻底没找到，返回外部占位符
	qualifiedName := cleanName
	if imps, ok := fc.Imports[cleanName]; ok && len(imps) > 0 {
		qualifiedName = imps[0].RawImportPath
	}
	return &model.CodeElement{Name: cleanName, QualifiedName: qualifiedName, Kind: model.Method, IsFormExternal: true}
}

// resolveStructure 处理类、接口等结构体符号
func (j *SymbolResolver) resolveStructure(gc *core.GlobalContext, fc *core.FileContext, symbol string, kind model.ElementKind) *model.CodeElement {
	if entries := j.preciseResolve(gc, fc, symbol); len(entries) > 0 {
		return entries[0].Element
	}

	// 外部符号占位：尝试利用 Import 还原 QualifiedName
	qualifiedName := symbol
	if imps, ok := fc.Imports[symbol]; ok && len(imps) > 0 {
		qualifiedName = imps[0].RawImportPath
	}
	return &model.CodeElement{Name: symbol, QualifiedName: qualifiedName, Kind: kind, IsFormExternal: true}
}

// =============================================================================
// 3. 递归回溯与继承查找 (The Core Logic)
// =============================================================================

// resolveInScopeHierarchy 递归向上查找容器及继承链
func (j *SymbolResolver) resolveInScopeHierarchy(gc *core.GlobalContext, fc *core.FileContext, containerQN, symbol string, isStatic bool, sourceElem *model.CodeElement) *model.CodeElement {
	if containerQN == "" {
		return nil
	}

	// 1. 尝试在当前层级直接匹配 QN
	targetQN := j.BuildQualifiedName(containerQN, symbol)
	if entry, ok := gc.FindByQualifiedName(targetQN); ok {
		// 检查静态约束与可见性
		if j.checkVisibility(gc, fc, sourceElem, entry) {
			isIllegalStatic := isStatic && entry.Element.Kind == model.Field && !slices.Contains(entry.Element.Extra.Modifiers, "static")
			if !isIllegalStatic {
				return entry.Element
			}
		}
	}

	// 2. 当前层级未命中，获取容器实体以继续向上
	containerEntry, ok := gc.FindByQualifiedName(containerQN)
	if !ok {
		return nil
	}

	// A. 如果是类，递归查找其继承链 (extends/implements)
	if containerEntry.Element.Kind == model.Class || containerEntry.Element.Kind == model.Interface {
		if inherited := j.resolveFromInheritance(gc, fc, containerEntry.Element, symbol, isStatic, sourceElem); inherited != nil {
			return inherited
		}
	}

	// B. 递归到父级容器 (Lexical Scope Up)
	return j.resolveInScopeHierarchy(gc, fc, containerEntry.ParentQN, symbol, isStatic, sourceElem)
}

// resolveFromInheritance 递归处理父类与接口的 Field，支持静态过滤与可见性
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
			targetFieldQN := j.BuildQualifiedName(parentElem.QualifiedName, symbol)

			if fieldEntry, ok := gc.FindByQualifiedName(targetFieldQN); ok {
				// 校验可见性与静态约束
				if j.checkVisibility(gc, fc, sourceElem, fieldEntry) {
					if !isStatic || slices.Contains(fieldEntry.Element.Extra.Modifiers, "static") {
						return fieldEntry.Element
					}
				}
			}
			// 深度优先：递归查找父类的父类
			if found := j.resolveFromInheritance(gc, fc, parentElem, symbol, isStatic, sourceElem); found != nil {
				return found
			}
		}
	}
	return nil
}

// =============================================================================
// 4. 辅助校验与解析工具
// =============================================================================

// preciseResolve 基础符号查找逻辑 (本地->精确导入->同包->通配符)
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

// checkVisibility 检查访问权限
func (j *SymbolResolver) checkVisibility(gc *core.GlobalContext, fc *core.FileContext, source *model.CodeElement, target *core.DefinitionEntry) bool {
	if target.Element.Kind != model.Field {
		return true
	} // 局部变量始终可见

	mods := target.Element.Extra.Modifiers
	if slices.Contains(mods, "public") {
		return true
	}

	targetPkg := j.getPackageFromQN(target.ParentQN)
	if targetPkg == fc.PackageName {
		return true
	}

	// Protected 跨包检查：是否为子类关系
	if slices.Contains(mods, "protected") {
		sourceClass := j.getOwnerClassQN(gc, source)
		return j.isSubClassOf(gc, fc, sourceClass, target.ParentQN)
	}

	return false // Private 或 Default 跨包均不可见
}

func (j *SymbolResolver) determinePreciseContainer(fc *core.FileContext, n *sitter.Node, kinds []model.ElementKind) *model.CodeElement {
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

// pickBestOverload 在多个方法定义中根据参数量选择最优解
func (j *SymbolResolver) pickBestOverload(entries []*core.DefinitionEntry, argCount int) *model.CodeElement {
	if len(entries) == 1 {
		return entries[0].Element
	}

	for _, entry := range entries {
		if entry.Element.Kind != model.Method {
			continue
		}
		// 对应 Collector 中的存储格式：[type1, name1, type2, name2]
		if params, ok := entry.Element.Extra.Mores[MethodParameters].([]string); ok {
			if len(params)/2 == argCount {
				return entry.Element
			}
		}
	}
	return entries[0].Element // 找不到精确匹配时返回第一个
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

func (j *SymbolResolver) getPackageFromQN(qn string) string {
	if idx := strings.LastIndex(qn, "."); idx != -1 {
		return qn[:idx]
	}
	return ""
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
