package noisefilter

import "github.com/CodMac/go-treesitter-dependency-analyzer/model"

// NoiseFilter 定义了如何识别特定语言中的背景噪音
type NoiseFilter interface {
	IsNoise(qualifiedName string) bool
}

// LanguageNoiseFilterFactory 是一个工厂函数类型，用于创建特定语言的 NoiseFilter 实例。
type LanguageNoiseFilterFactory func() NoiseFilter

var noiseFilterFactories = make(map[model.Language]LanguageNoiseFilterFactory)

// RegisterNoiseFilter 注册一个语言与其对应的 NoiseFilter 工厂函数。
func RegisterNoiseFilter(lang model.Language, factory LanguageNoiseFilterFactory) {
	noiseFilterFactories[lang] = factory
}

// GetNoiseFilter 根据语言类型获取对应的 NoiseFilter 实例。
func GetNoiseFilter(lang model.Language) (NoiseFilter, error) {
	factory, ok := noiseFilterFactories[lang]
	if !ok {
		// 如果没注册，返回一个默认不进行过滤的过滤器，防止程序奔溃
		return &DefaultNoiseFilter{}, nil
	}

	return factory(), nil
}

// DefaultNoiseFilter 默认过滤器：不对任何 QN 进行噪音判定
type DefaultNoiseFilter struct{}

func (d *DefaultNoiseFilter) IsNoise(qn string) bool { return false }
