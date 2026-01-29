package main

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
	// 新增：过滤等级配置
	FilterLevel core.FilterLevel
}

func NewFileProcessor(lang core.Language, outputAST, formatAST bool, concurrency int, filterLevel core.FilterLevel) *FileProcessor {
	if concurrency <= 0 {
		concurrency = 4
	}
	return &FileProcessor{
		Language:    lang,
		OutputAST:   outputAST,
		FormatAST:   formatAST,
		Concurrency: concurrency,
		FilterLevel: filterLevel,
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
	linker, err := core.GetLinker(fp.Language)
	if err != nil {
		return nil, nil, err
	}
	hierarchyRels := linker.LinkHierarchy(globalContext)

	// --- 阶段 3: 并行提取依赖 (Extractor) ---
	var allRelations []*model.DependencyRelation
	// 初始放入层级关系（注意：根据讨论，Contain 关系通常不作为噪音过滤）
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
	if err != nil {
		return nil, nil, err
	}

	// --- 阶段 4: 噪音过滤 (Noise Filtering) ---
	// 在所有关系收集完毕后，根据设定的 Level 进行统一清洗
	filteredRelations := fp.filterNoise(allRelations)

	return filteredRelations, globalContext, nil
}

// filterNoise 调用语言特定的过滤器进行数据清洗
func (fp *FileProcessor) filterNoise(rels []*model.DependencyRelation) []*model.DependencyRelation {
	filter := core.GetNoiseFilter(fp.Language)
	filter.SetLevel(fp.FilterLevel)

	// 如果是 Raw 级别，直接返回
	if fp.FilterLevel == core.LevelRaw {
		return rels
	}

	result := make([]*model.DependencyRelation, 0, len(rels))
	for _, rel := range rels {
		// 如果不是噪音，则保留
		if !filter.IsNoise(*rel) {
			result = append(result, rel)
		}
	}
	return result
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
