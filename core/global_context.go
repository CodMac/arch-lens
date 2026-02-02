package core

import (
	"path/filepath"
	"sync"

	"github.com/CodMac/arch-lens/model"
)

type GlobalContext struct {
	FileContexts     map[string]*FileContext
	Definitions      []*DefinitionEntry
	qualifiedNameMap map[string]*DefinitionEntry
	resolver         SymbolResolver // 持有具体语言的解析器
	mutex            sync.RWMutex
}

func NewGlobalContext(resolver SymbolResolver) *GlobalContext {
	return &GlobalContext{
		FileContexts:     make(map[string]*FileContext),
		Definitions:      make([]*DefinitionEntry, 0),
		qualifiedNameMap: make(map[string]*DefinitionEntry),
		resolver:         resolver,
	}
}

// RegisterFileContext 逻辑现在调用 resolver 处理包名
func (gc *GlobalContext) RegisterFileContext(fc *FileContext) {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	gc.FileContexts[fc.FilePath] = fc

	// 1. 注册文件节点
	fileElem := &model.CodeElement{
		Kind:          model.File,
		Name:          filepath.Base(fc.FilePath),
		QualifiedName: fc.FilePath,
		Path:          fc.FilePath,
		IsFormSource:  true,
	}
	gc.Definitions = append(gc.Definitions, &DefinitionEntry{Element: fileElem})
	gc.qualifiedNameMap[fc.FilePath] = &DefinitionEntry{Element: fileElem}

	// 2. 委托 Resolver 处理包/命名空间注册 (Java 拆分, Go 不拆)
	gc.resolver.RegisterPackage(gc, fc.PackageName)

	// 3. 注册文件内定义
	for _, entry := range fc.Definitions {
		gc.Definitions = append(gc.Definitions, entry)
		gc.qualifiedNameMap[entry.Element.QualifiedName] = entry
	}
}

// ResolveSymbol 彻底由 Resolver 驱动
func (gc *GlobalContext) ResolveSymbol(fc *FileContext, symbol string) []*DefinitionEntry {
	return gc.resolver.Resolve(gc, fc, symbol)
}

func (gc *GlobalContext) AddDefinition(def *DefinitionEntry) {
	defQN := def.Element.QualifiedName

	_, ok := gc.qualifiedNameMap[defQN]
	if !ok {
		gc.Definitions = append(gc.Definitions, def)
		gc.qualifiedNameMap[defQN] = def
	}
}

func (gc *GlobalContext) FindByQualifiedName(qn string) (*DefinitionEntry, bool) {
	entry, ok := gc.qualifiedNameMap[qn]
	return entry, ok
}

func (gc *GlobalContext) BuildQualifiedName(parentQN, name string) string {
	return gc.resolver.BuildQualifiedName(parentQN, name)
}

func (gc *GlobalContext) RLock() { gc.mutex.RLock() }

func (gc *GlobalContext) RUnlock() { gc.mutex.RUnlock() }
