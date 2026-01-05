package extractor

import (
	"fmt"

	"github.com/CodMac/go-treesitter-dependency-analyzer/context"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
)

// Extractor 用于提取关系，需要全局上下文。
type Extractor interface {
	// Extract 基于全局上下文，返回文件中的依赖关系。
	Extract(filePath string, gCtx *context.GlobalContext) ([]*model.DependencyRelation, error)
}

// LanguageExtractorFactory 是一个工厂函数类型，用于创建特定语言的 Extractor 实例。
type LanguageExtractorFactory func() Extractor

var extractorFactories = make(map[model.Language]LanguageExtractorFactory)

// RegisterExtractor 注册一个语言与其对应的 Extractor 工厂函数。
func RegisterExtractor(lang model.Language, factory LanguageExtractorFactory) {
	extractorFactories[lang] = factory
}

// GetExtractor 根据语言类型获取对应的 Extractor 实例。
func GetExtractor(lang model.Language) (Extractor, error) {
	factory, ok := extractorFactories[lang]
	if !ok {
		return nil, fmt.Errorf("no extractor registered for language: %s", lang)
	}
	return factory(), nil
}
