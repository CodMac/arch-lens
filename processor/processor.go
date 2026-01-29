package processor

import (
	"path/filepath"
	"sync"

	"github.com/CodMac/go-treesitter-dependency-analyzer/core"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	"github.com/CodMac/go-treesitter-dependency-analyzer/parser"
)

type FileProcessor struct {
	Language    core.Language
	OutputAST   bool
	FormatAST   bool
	Concurrency int
}

func NewFileProcessor(lang core.Language, outputAST, formatAST bool, concurrency int) *FileProcessor {
	if concurrency <= 0 {
		concurrency = 4
	}
	return &FileProcessor{
		Language:    lang,
		OutputAST:   outputAST,
		FormatAST:   formatAST,
		Concurrency: concurrency,
	}
}

func (fp *FileProcessor) ProcessFiles(rootPath string, filePaths []string) ([]*model.DependencyRelation, *core.GlobalContext, error) {
	resolver, err := core.GetSymbolResolver(fp.Language)
	if err != nil {
		return nil, nil, err
	}

	globalContext := core.NewGlobalContext(resolver)
	absRoot, _ := filepath.Abs(rootPath)

	// --- 阶段 1: 并行收集 (Collector) ---
	err = fp.runParallel(filePaths, func(path string, p parser.Parser) error {
		root, source, err := p.ParseFile(path, fp.OutputAST, fp.FormatAST)
		if err != nil {
			return err
		}

		cot, err := core.GetCollector(fp.Language)
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(absRoot, path)
		fCtx, err := cot.CollectDefinitions(root, relPath, source)
		if err != nil {
			return err
		}

		globalContext.RegisterFileContext(fCtx)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	// --- 阶段 2: 拓扑链接 (Linker) ---
	// 这一步由语言插件决定如何构建层级，Processor 保持中立
	linker, err := core.GetLinker(fp.Language)
	if err != nil {
		return nil, nil, err
	}
	hierarchyRels := linker.LinkHierarchy(globalContext)

	// --- 阶段 3: 并行提取依赖 (Extractor) ---
	var allRelations []*model.DependencyRelation
	allRelations = append(allRelations, hierarchyRels...)

	var mu sync.Mutex
	err = fp.runParallel(filePaths, func(path string, p parser.Parser) error {
		ext, err := core.GetExtractor(fp.Language)
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(absRoot, path)
		rels, err := ext.Extract(relPath, globalContext)
		if err != nil {
			return err
		}

		mu.Lock()
		defer mu.Unlock()
		for _, rel := range rels {
			// 归一化位置信息
			if rel.Location != nil && filepath.IsAbs(rel.Location.FilePath) {
				if rPath, err := filepath.Rel(absRoot, rel.Location.FilePath); err == nil {
					rel.Location.FilePath = rPath
				}
			}
			allRelations = append(allRelations, rel)
		}
		return nil
	})

	return allRelations, globalContext, err
}

// runParallel 内部并发调度器
func (fp *FileProcessor) runParallel(paths []string, task func(string, parser.Parser) error) error {
	pathChan := make(chan string, len(paths))
	for _, p := range paths {
		pathChan <- p
	}
	close(pathChan)

	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

	for i := 0; i < fp.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := parser.NewParser(fp.Language)
			if err != nil {
				errOnce.Do(func() { firstErr = err })
				return
			}
			defer p.Close()

			for path := range pathChan {
				if err := task(path, p); err != nil {
					errOnce.Do(func() { firstErr = err })
					return
				}
			}
		}()
	}
	wg.Wait()
	return firstErr
}
