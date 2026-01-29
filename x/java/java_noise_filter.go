package java

import (
	"github.com/CodMac/go-treesitter-dependency-analyzer/core"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
)

type NoiseFilter struct {
	core.DefaultNoiseFilter
}

func NewJavaNoiseFilter(level core.FilterLevel) *NoiseFilter {
	return &NoiseFilter{
		DefaultNoiseFilter: core.DefaultNoiseFilter{Level: level},
	}
}

func (f *NoiseFilter) IsNoise(rel model.DependencyRelation) bool {
	if rel.Target == nil || f.Level == core.LevelRaw {
		return false
	}

	// 规则 1 & 2: 源码、语法糖、包含关系 始终保留
	if rel.Target.IsFormSource || rel.Target.IsFormSugar || rel.Type == model.Contain {
		return false
	}

	// 进入 LevelBalanced 的判定
	if f.Level == core.LevelBalanced {
		// 目标是外部符号
		if rel.Target.IsFormExternal {
			switch rel.Type {
			// 规则 3: 外部继承/实现 -> 噪音
			case model.Extend, model.Implement:
				return true
			// 规则 4: 外部类型引用 -> 噪音
			case model.Annotation, model.Parameter, model.Return, model.TypeArg:
				return true
			// 规则 5: 外部行为与执行流 -> 噪音
			case model.Call, model.Create, model.Cast, model.Use, model.Assign:
				return true
			}
		}
	}

	// LevelPure 逻辑 (如果你在 core 实现了通用逻辑，这里可以简化)
	if f.Level == core.LevelPure {
		return rel.Target.IsFormExternal
	}

	return false
}

func (f *NoiseFilter) SetLevel(level core.FilterLevel) {
	f.Level = level
}
