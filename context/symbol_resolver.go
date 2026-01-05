package context

import (
	"fmt"

	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
)

// --- 语言特有的符号解析接口 ---

type SymbolResolver interface {
	// BuildQualifiedName 根据父节点和当前名构建 QN
	// (Java 用 ".", C++ 用 "::")
	BuildQualifiedName(parentQN, name string) string

	// Resolve 具体的解析逻辑：处理局部、导入、通配符等逻辑
	Resolve(gc *GlobalContext, fc *FileContext, symbol string) []*DefinitionEntry

	// RegisterPackage 注册包/命名空间逻辑
	// (Java 需要拆分点号，Go 只需要单层)
	RegisterPackage(gc *GlobalContext, packageName string)
}

// LanguageSymbolResolverFactory 是一个工厂函数类型，用于创建特定语言的 SymbolResolver 实例。
type LanguageSymbolResolverFactory func() SymbolResolver

var symbolResolverFactories = make(map[model.Language]LanguageSymbolResolverFactory)

// RegisterSymbolResolver 注册一个语言与其对应的 SymbolResolver 工厂函数。
func RegisterSymbolResolver(lang model.Language, factory LanguageSymbolResolverFactory) {
	symbolResolverFactories[lang] = factory
}

// GetSymbolResolver 根据语言类型获取对应的 SymbolResolver 实例。
func GetSymbolResolver(lang model.Language) (SymbolResolver, error) {
	factory, ok := symbolResolverFactories[lang]
	if !ok {
		return nil, fmt.Errorf("no SymbolResolver for language: %s", lang)
	}

	return factory(), nil
}
