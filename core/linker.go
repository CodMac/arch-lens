package core

import (
	"fmt"

	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
)

// Linker 用于提取文件间的关系，需要全局上下文。
type Linker interface {
	// LinkHierarchy 根据全局上下文，推导出层级关系（Package->Package, Package->File 等）
	LinkHierarchy(gc *GlobalContext) []*model.DependencyRelation
}

var linkerMap = make(map[Language]Linker)

// RegisterLinker 注册一个语言与其对应的 Linker
func RegisterLinker(lang Language, linker Linker) {
	linkerMap[lang] = linker
}

// GetLinker 根据语言类型获取对应的 Linker 实例。
func GetLinker(lang Language) (Linker, error) {
	linker, ok := linkerMap[lang]
	if !ok {
		return nil, fmt.Errorf("no linker registered for language: %s", lang)
	}

	return linker, nil
}
