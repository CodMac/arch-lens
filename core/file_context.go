package core

import (
	"sync"

	"github.com/CodMac/arch-lens/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type DefinitionEntry struct {
	Element  *model.CodeElement
	ParentQN string
	Node     *sitter.Node // 保留 AST 节点引用用于后期元数据填充
}

type ImportEntry struct {
	RawImportPath string            `json:"RawImportPath"`
	Alias         string            `json:"Alias"`
	Kind          model.ElementKind `json:"Kind"`
	IsWildcard    bool              `json:"IsWildcard"`
	IsStatic      bool              `json:"IsStatic"`
	Location      *model.Location   `json:"Location,omitempty"`
}

type FileContext struct {
	FilePath     string
	PackageName  string
	RootNode     *sitter.Node
	SourceBytes  *[]byte
	Definitions  []*DefinitionEntry
	Imports      map[string][]*ImportEntry
	mutex        sync.RWMutex
	shortNameMap map[string][]*DefinitionEntry
	kindMap      map[model.ElementKind][]*DefinitionEntry
}

func NewFileContext(filePath string, rootNode *sitter.Node, sourceBytes *[]byte) *FileContext {
	return &FileContext{
		FilePath:     filePath,
		RootNode:     rootNode,
		SourceBytes:  sourceBytes,
		Definitions:  []*DefinitionEntry{},
		Imports:      make(map[string][]*ImportEntry),
		shortNameMap: make(map[string][]*DefinitionEntry),
		kindMap:      make(map[model.ElementKind][]*DefinitionEntry),
	}
}

func (fc *FileContext) AddDefinition(elem *model.CodeElement, parentQN string, node *sitter.Node) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	entry := DefinitionEntry{Element: elem, ParentQN: parentQN, Node: node}

	fc.Definitions = append(fc.Definitions, &entry)
	fc.shortNameMap[elem.Name] = append(fc.shortNameMap[elem.Name], &entry)
	fc.kindMap[elem.Kind] = append(fc.kindMap[elem.Kind], &entry)
}

func (fc *FileContext) AddImport(alias string, imp *ImportEntry) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.Imports[alias] = append(fc.Imports[alias], imp)
}

func (fc *FileContext) FindByShortName(sn string) ([]*DefinitionEntry, bool) {
	entries, ok := fc.shortNameMap[sn]
	return entries, ok
}

func (fc *FileContext) FindByElementKind(kind model.ElementKind) ([]*DefinitionEntry, bool) {
	entries, ok := fc.kindMap[kind]
	return entries, ok
}
