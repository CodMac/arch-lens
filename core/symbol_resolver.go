package core

import (
	"fmt"

	"github.com/CodMac/arch-lens/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// --- 语言特有的符号解析接口 ---

type SymbolResolver interface {
	// BuildQualifiedName 根据父节点和当前名构建 QN
	// (Java 用 ".", C++ 用 "::")
	BuildQualifiedName(parentQN, name string) string

	// Resolve 具体的解析逻辑：处理局部、导入、通配符等逻辑
	Resolve(gc *GlobalContext, fc *FileContext, node *sitter.Node, receiver, symbol string, kind model.ElementKind) *model.CodeElement

	// RegisterPackage 注册包/命名空间逻辑
	// (Java 需要拆分点号，Go 只需要单层)
	RegisterPackage(gc *GlobalContext, packageName string)
}

var symbolResolverMap = make(map[Language]SymbolResolver)

// RegisterSymbolResolver 注册一个语言与其对应的 SymbolResolver 工厂函数。
func RegisterSymbolResolver(lang Language, resolver SymbolResolver) {
	symbolResolverMap[lang] = resolver
}

// GetSymbolResolver 根据语言类型获取对应的 SymbolResolver 实例。
func GetSymbolResolver(lang Language) (SymbolResolver, error) {
	resolver, ok := symbolResolverMap[lang]
	if !ok {
		return nil, fmt.Errorf("no SymbolResolver for language: %s", lang)
	}

	return resolver, nil
}
