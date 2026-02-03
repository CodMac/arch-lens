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
// Resolve (符号解析)
// =============================================================================

// preciseResolve 精准符号解析，不存在则返回nil
func (j *SymbolResolver) preciseResolve(gc *core.GlobalContext, fc *core.FileContext, symbol string) []*core.DefinitionEntry {
	gc.RLock()
	defer gc.RUnlock()

	// 1. 局部定义
	if defs, ok := fc.FindByShortName(symbol); ok {
		return defs
	}

	// 2. 精确导入
	if imps, ok := fc.Imports[symbol]; ok {
		for _, imp := range imps {
			// 仅import的源码符号才有可能在上下文中被发现，外部导入直接返回nil
			if def, found := gc.FindByQualifiedName(imp.RawImportPath); found {
				return []*core.DefinitionEntry{def}
			} else {
				return nil
			}
		}
	}

	// 3. 同包前缀
	pkgQN := j.BuildQualifiedName(fc.PackageName, symbol)
	if def, ok := gc.FindByQualifiedName(pkgQN); ok {
		return []*core.DefinitionEntry{def}
	}

	// 4. Java 特有的通配符导入
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

	// 5. 直接按 QN 查找 (处理代码中使用全限定名调用的情况)
	if def, ok := gc.FindByQualifiedName(symbol); ok {
		return []*core.DefinitionEntry{def}
	}

	return nil
}

// resolveStructure 结构体符号解析 (Class、Interface、Annotation...)
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

// resolveVariable 变量符号解析 (Variable), 支持递归往上寻找Field
func (j *SymbolResolver) resolveVariable(gc *core.GlobalContext, fc *core.FileContext, node *sitter.Node, name string) *model.CodeElement {
	cleanName := strings.TrimSpace(name)
	varContainerTypes := []model.ElementKind{model.Class, model.AnonymousClass, model.Lambda, model.Method, model.ScopeBlock}
	container := j.determinePreciseContainer(fc, node, varContainerTypes)

	if container == nil {
		return j.resolveStructure(gc, fc, cleanName, model.Variable)
	}

	// 初始判定：如果起点在静态方法或静态块中，标记为静态上下文
	isStaticContext := false
	if container.Kind == model.Method || container.Kind == model.ScopeBlock {
		if slices.Contains(container.Extra.Modifiers, "static") {
			isStaticContext = true
		}
	}

	return j.resolveInScopeHierarchy(gc, fc, container.QualifiedName, cleanName, isStaticContext)
}

// resolveMethod 方法符号解析 (method), 支持重载识别逻辑
func (j *SymbolResolver) resolveMethod(gc *core.GlobalContext, fc *core.FileContext, node *sitter.Node, symbol string) *model.CodeElement {
	entries := j.preciseResolve(gc, fc, symbol)
	if len(entries) == 0 {
		return &model.CodeElement{Name: symbol, Kind: model.Method, IsFormExternal: true}
	}

	// 从 AST node 尝试还原参数个数
	argCount := -1
	if node != nil {
		// 寻找最近的调用节点以获取参数列表
		callNode := j.findInvocationNode(node)
		if callNode != nil {
			if args := callNode.ChildByFieldName("arguments"); args != nil {
				argCount = int(args.NamedChildCount())
			}
		}
	}

	// 匹配重载
	for _, entry := range entries {
		if entry.Element.Kind != model.Method {
			continue
		}
		if params, ok := entry.Element.Extra.Mores[MethodParameters].([]string); ok {
			if argCount == -1 || len(params)/2 == argCount {
				return entry.Element
			}
		}
	}

	return entries[0].Element
}

// =============================================================================
// 辅助函数
// =============================================================================

// 递归向上查找容器及继承链
func (j *SymbolResolver) resolveInScopeHierarchy(gc *core.GlobalContext, fc *core.FileContext, containerQN, symbol string, isStatic bool) *model.CodeElement {
	if containerQN == "" {
		return nil
	}

	targetQN := j.BuildQualifiedName(containerQN, symbol)
	if entry, ok := gc.FindByQualifiedName(targetQN); ok {
		// 优化点：如果是静态上下文，过滤掉非静态 Field
		if isStatic && entry.Element.Kind == model.Field {
			if !slices.Contains(entry.Element.Extra.Modifiers, "static") {
				goto CONTINUE_UP // 静态上下文引用了非静态成员，继续向上找（可能外层还有同名变量）
			}
		}
		return entry.Element
	}

CONTINUE_UP:
	entry, ok := gc.FindByQualifiedName(containerQN)
	if ok {
		// 类级别继承查找
		if entry.Element.Kind == model.Class || entry.Element.Kind == model.Interface {
			if found := j.resolveFromInheritance(gc, fc, entry.Element, symbol); found != nil {
				return found
			}
		}
		// 递归向上时，如果穿过类边界，通常意味着进入了外部类作用域
		return j.resolveInScopeHierarchy(gc, fc, entry.ParentQN, symbol, isStatic)
	}
	return nil
}

// 递归处理父类与接口的 Field
func (j *SymbolResolver) resolveFromInheritance(gc *core.GlobalContext, fc *core.FileContext, elem *model.CodeElement, symbol string) *model.CodeElement {
	if elem.Extra == nil {
		return nil
	}

	// 1. 搜集所有父类和接口
	var superTargets []string
	// 获取父类 (extends)
	if sc, ok := elem.Extra.Mores[ClassSuperClass].(string); ok && sc != "" {
		superTargets = append(superTargets, sc)
	}
	// 获取接口 (implements)
	if itfs, ok := elem.Extra.Mores[ClassImplementedInterfaces].([]string); ok {
		superTargets = append(superTargets, itfs...)
	}

	for _, rawSuperName := range superTargets {
		// 清理泛型 (ArrayList<String> -> ArrayList)
		cleanSuperName := strings.Split(rawSuperName, "<")[0]

		// 2. 解析父类的定义条目 (可能在当前文件，也可能在其他文件)
		parentEntries := j.preciseResolve(gc, fc, cleanSuperName)
		if len(parentEntries) > 0 {
			parentElem := parentEntries[0].Element

			// 3. 拼接 QN 尝试在父类中寻找符号 (ParentQN.symbol)
			targetFieldQN := j.BuildQualifiedName(parentElem.QualifiedName, symbol)
			if fieldEntry, ok := gc.FindByQualifiedName(targetFieldQN); ok {
				// 找到了字段（此处暂不处理可见性，默认为可访问）
				return fieldEntry.Element
			}

			// 4. 深度优先递归：在父类的父类中继续找
			if found := j.resolveFromInheritance(gc, fc, parentElem, symbol); found != nil {
				return found
			}
		}
	}

	return nil
}

// 根据行号范围寻找最深层容器
func (j *SymbolResolver) determinePreciseContainer(fc *core.FileContext, n *sitter.Node, containerTypes []model.ElementKind) *model.CodeElement {
	var best *model.CodeElement
	var minSize uint32 = 0xFFFFFFFF
	row := n.StartPosition().Row + 1

	for _, entry := range fc.Definitions {
		elem := entry.Element
		if slices.Contains(containerTypes, elem.Kind) {
			if int(row) >= elem.Location.StartLine && int(row) <= elem.Location.EndLine {
				size := uint32(elem.Location.EndLine - elem.Location.StartLine)
				if size < minSize {
					minSize = size
					best = elem
				}
			}
		}
	}

	return best
}

// 根据引用点寻找调用表达式
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
