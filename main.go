package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/CodMac/go-treesitter-dependency-analyzer/context"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	"github.com/CodMac/go-treesitter-dependency-analyzer/noisefilter"
	"github.com/CodMac/go-treesitter-dependency-analyzer/output"
	"github.com/CodMac/go-treesitter-dependency-analyzer/processor"
	_ "github.com/CodMac/go-treesitter-dependency-analyzer/x/java" // 插件注册
)

const (
	MaxMermaidNodes = 150
	MaxMermaidEdges = 250
)

type Config struct {
	Lang         string
	SourcePath   string
	Filter       string
	Jobs         int
	OutDir       string
	Format       string
	SkipExternal bool
}

func main() {
	cfg := parseFlags()
	startTime := time.Now()

	// 1. 扫描文件
	fmt.Fprintf(os.Stderr, "[1/4] 🔍 正在扫描目录: %s\n", cfg.SourcePath)
	files, err := scanFiles(cfg.SourcePath, cfg.Filter, cfg.Lang)
	if err != nil {
		exitWithError("扫描文件失败", err)
	}
	fmt.Fprintf(os.Stderr, "    找到 %d 个候选文件\n", len(files))

	// 2. 执行核心分析过程
	fmt.Fprintf(os.Stderr, "[2/4] ⚙️  正在并发分析代码符号与关系 (CGO_ENABLED=1)...\n")
	proc := processor.NewFileProcessor(model.Language(cfg.Lang), false, false, cfg.Jobs)
	rels, gCtx, err := proc.ProcessFiles(cfg.SourcePath, files)
	if err != nil {
		exitWithError("分析执行失败", err)
	}

	// 3. 执行导出逻辑
	fmt.Fprintf(os.Stderr, "[3/4] 💾 正在准备数据导出...\n")
	nf := noisefilter.GetNoiseFilter(model.Language(cfg.Lang))
	if err := runExport(cfg, gCtx, rels, nf); err != nil {
		exitWithError("导出失败", err)
	}

	fmt.Fprintf(os.Stderr, "\n[4/4] ✨ 任务完成! 总耗时: %v\n", time.Since(startTime).Round(time.Millisecond))
}

// --- 辅助函数 ---

func parseFlags() Config {
	c := Config{}
	flag.StringVar(&c.Lang, "lang", "java", "分析语言 (e.g. java)")
	flag.StringVar(&c.SourcePath, "path", ".", "源代码根路径")
	flag.StringVar(&c.Filter, "filter", "", "文件过滤正则 (可选)")
	flag.IntVar(&c.Jobs, "jobs", 4, "并发线程数")
	flag.StringVar(&c.OutDir, "out-dir", "./output", "输出结果目录")
	flag.StringVar(&c.Format, "format", "jsonl", "导出格式: jsonl, mermaid")
	flag.BoolVar(&c.SkipExternal, "skip-external", true, "是否隐藏外部噪音依赖")
	flag.Parse()
	return c
}

func runExport(cfg Config, gCtx *context.GlobalContext, rels []*model.DependencyRelation, nf noisefilter.NoiseFilter) error {
	_ = os.MkdirAll(cfg.OutDir, 0755)

	format := cfg.Format
	// 自动降级逻辑
	if format == "mermaid" {
		nodeCount := len(gCtx.DefinitionsByQN)
		if nodeCount > MaxMermaidNodes || len(rels) > MaxMermaidEdges {
			fmt.Fprintf(os.Stderr, "    ⚠️  节点数(%d)或关系数(%d)过大，Mermaid 渲染可能卡顿，降级为 jsonl\n", nodeCount, len(rels))
			format = "jsonl"
		}
	}

	switch format {
	case "mermaid":
		p := filepath.Join(cfg.OutDir, "visualization.html")
		return output.ExportMermaidHTML(p, gCtx, rels, cfg.SkipExternal, nf)
	default:
		return exportJSONLSet(cfg.OutDir, gCtx, rels, cfg.SkipExternal, nf)
	}
}

func exportJSONLSet(dir string, gCtx *context.GlobalContext, rels []*model.DependencyRelation, skip bool, nf noisefilter.NoiseFilter) error {
	elemPath := filepath.Join(dir, "element.jsonl")
	relPath := filepath.Join(dir, "relation.jsonl")

	ec, _ := output.ExportElements(elemPath, gCtx)
	rc, _ := output.ExportRelations(relPath, rels, gCtx, skip, nf)

	fmt.Fprintf(os.Stderr, "    ✅ 导出完成: 元素=%d, 关系=%d\n", ec, rc)
	return nil
}

func scanFiles(root, filter, lang string) ([]string, error) {
	if filter == "" {
		filter = fmt.Sprintf(`.*\.%s$`, lang)
	}
	re, err := regexp.Compile(filter)
	if err != nil {
		return nil, err
	}

	var files []string
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && re.MatchString(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func exitWithError(msg string, err error) {
	fmt.Fprintf(os.Stderr, "❌ %s: %v\n", msg, err)
	os.Exit(1)
}
