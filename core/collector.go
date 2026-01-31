package core

import (
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Collector 用于收集文件中的符号定义。
type Collector interface {
	// CollectDefinitions 负责遍历 AST，建立并返回该文件的 FileContext。
	CollectDefinitions(rootNode *sitter.Node, filePath string, sourceBytes *[]byte) (*FileContext, error)
}

var collectorMap = make(map[Language]Collector)

// RegisterCollector 注册一个语言与其对应的 Collector
func RegisterCollector(lang Language, collector Collector) {
	collectorMap[lang] = collector
}

// GetCollector 根据语言类型获取对应的 Collector 实例。
func GetCollector(lang Language) (Collector, error) {
	collector, ok := collectorMap[lang]
	if !ok {
		return nil, fmt.Errorf("no collector registered for language: %s", lang)
	}

	return collector, nil
}
