package model

import (
	"fmt"
	"sync"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// DefinitionEntry 存储了一个符号定义的完整信息，以及它所属的父元素QN。
type DefinitionEntry struct {
	Element  *CodeElement // 符号本身
	ParentQN string       // 父元素的QualifiedName
}

// FileContext 存储了单个文件的所有符号定义、包名和源代码内容。
type FileContext struct {
	FilePath        string                        // 文件路径
	PackageName     string                        // 文件所属的包/模块名 (例如 Java 的 package)
	RootNode        *sitter.Node                  // AST根节点
	SourceBytes     *[]byte                       // 源码内容 (指针)
	DefinitionsBySN map[string][]*DefinitionEntry // 局部定义查找 (短名称 -> 定义列表), 使用切片支持重载（如多个构造函数）或内部类与方法同名的情况
	Imports         map[string]string             // 导入表 (短名称/别名 -> 全限定名), 例如 Java: "List" -> "java.util.List"
	mutex           sync.RWMutex
}

// NewFileContext 创建一个新的 FileContext 实例。
func NewFileContext(filePath string, rootNode *sitter.Node, sourceBytes *[]byte) *FileContext {
	return &FileContext{
		FilePath:        filePath,
		RootNode:        rootNode,
		SourceBytes:     sourceBytes,
		DefinitionsBySN: make(map[string][]*DefinitionEntry),
		Imports:         make(map[string]string),
	}
}

// AddDefinition 将一个符号定义添加到 FileContext 中。
func (fc *FileContext) AddDefinition(elem *CodeElement, parentQN string) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	entry := &DefinitionEntry{
		Element:  elem,
		ParentQN: parentQN,
	}

	fc.DefinitionsBySN[elem.Name] = append(fc.DefinitionsBySN[elem.Name], entry)
}

// GlobalContext 存储了整个项目范围内的符号信息。
type GlobalContext struct {
	FileContexts    map[string]*FileContext       // 文件路径 -> FileContext
	DefinitionsByQN map[string][]*DefinitionEntry // 全局定义索引 (QN -> 定义列表), 支持同名 QN（处理多版本库或增量扫描时的冲突）
	mutex           sync.RWMutex
}

// NewGlobalContext 创建一个新的 GlobalContext 实例。
func NewGlobalContext() *GlobalContext {
	return &GlobalContext{
		FileContexts:    make(map[string]*FileContext),
		DefinitionsByQN: make(map[string][]*DefinitionEntry),
	}
}

// RegisterFileContext 将 FileContext 的信息同步到全局索引。
func (gc *GlobalContext) RegisterFileContext(fc *FileContext) {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	gc.FileContexts[fc.FilePath] = fc

	for _, entries := range fc.DefinitionsBySN {
		for _, entry := range entries {
			qn := entry.Element.QualifiedName
			gc.DefinitionsByQN[qn] = append(gc.DefinitionsByQN[qn], entry)
		}
	}
}

// ResolveSymbol 尝试解析一个标识符的具体定义。
// 返回所有可能的定义，由调用者根据参数签名或 Context 进一步过滤。
func (gc *GlobalContext) ResolveSymbol(fc *FileContext, symbol string) []*DefinitionEntry {
	gc.mutex.RLock()
	defer gc.mutex.RUnlock()

	var results []*DefinitionEntry

	// 1. 局部定义优先
	if defs, ok := fc.DefinitionsBySN[symbol]; ok {
		results = append(results, defs...)
		return results
	}

	// 2. 检查 Import (例如 "List" -> "java.util.List")
	if fullQN, ok := fc.Imports[symbol]; ok {
		if defs, found := gc.DefinitionsByQN[fullQN]; found {
			return defs
		}
	}

	// 3. 尝试当前包前缀 (隐式引用同包下的其他类)
	if fc.PackageName != "" {
		pkgQN := BuildQualifiedName(fc.PackageName, symbol)
		if defs, ok := gc.DefinitionsByQN[pkgQN]; ok {
			return defs
		}
	}

	// 4. 兜底：直接按 QN 查找
	if defs, ok := gc.DefinitionsByQN[symbol]; ok {
		return defs
	}

	return nil
}

// BuildQualifiedName 构建限定名称 (Qualified Name, QN)
func BuildQualifiedName(parentQN, name string) string {
	if parentQN == "" || parentQN == "." {
		return name
	}
	return fmt.Sprintf("%s.%s", parentQN, name)
}
