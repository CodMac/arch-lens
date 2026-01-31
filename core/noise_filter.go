package core

import "github.com/CodMac/arch-lens/model"

// FilterLevel 定义过滤的严苛程度
type FilterLevel int

const (
	LevelRaw      FilterLevel = iota // 不进行任何过滤，保留所有原始关系
	LevelBalanced                    // 过滤掉外部符号中的基础背景噪音（如 Object, String）
	LevelPure                        // 只保留源码产生的实体之间的关系 (Source -> Source)
)

// NoiseFilter 接口定义
type NoiseFilter interface {
	IsNoise(rel model.DependencyRelation) bool
	SetLevel(level FilterLevel)
}

var noiseFilterMap = make(map[Language]NoiseFilter)

// RegisterNoiseFilter 注册一个语言与其对应的 NoiseFilter
func RegisterNoiseFilter(lang Language, noiseFilter NoiseFilter) {
	noiseFilterMap[lang] = noiseFilter
}

// GetNoiseFilter 根据语言类型获取对应的 NoiseFilter 实例。
func GetNoiseFilter(lang Language) NoiseFilter {
	noiseFilter, ok := noiseFilterMap[lang]
	if !ok {
		// 如果没注册，返回一个默认不进行过滤的过滤器，防止程序奔溃
		return &DefaultNoiseFilter{}
	}

	return noiseFilter
}

// DefaultNoiseFilter 提供基础的等级管理，供各语言 Filter 嵌入
type DefaultNoiseFilter struct {
	Level FilterLevel
}

func (d *DefaultNoiseFilter) SetLevel(level FilterLevel) {
	d.Level = level
}

func (d *DefaultNoiseFilter) IsNoise(rel model.DependencyRelation) bool { return false }
