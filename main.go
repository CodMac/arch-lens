package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	context2 "github.com/CodMac/go-treesitter-dependency-analyzer/context"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	"github.com/CodMac/go-treesitter-dependency-analyzer/noisefilter"
	"github.com/CodMac/go-treesitter-dependency-analyzer/output"
	"github.com/CodMac/go-treesitter-dependency-analyzer/processor"
)

const (
	MaxMermaidNodes = 150
	MaxMermaidEdges = 250
)

func main() {
	lang := flag.String("lang", "java", "分析语言")
	path := flag.String("path", ".", "源代码项目根路径")
	filter := flag.String("filter", "", "文件过滤正则")
	jobs := flag.Int("jobs", 4, "并发数")
	outDir := flag.String("out-dir", "./output", "输出目录")
	format := flag.String("format", "jsonl", "输出格式 (jsonl, mermaid)")
	skipExternal := flag.Bool("skip-external", true, "是否隐藏外部库及噪音依赖")

	flag.Parse()

	startTime := time.Now()

	// 1. 根据语言获取对应的 NoiseFilter
	noiseFilter, err := noisefilter.GetNoiseFilter(model.Language(*lang))
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ 无法获取过滤器: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "[1/4] 🚀 正在扫描目录: %s\n", *path)
	actualFilter := *filter
	if actualFilter == "" {
		actualFilter = fmt.Sprintf(".*\\.%s$", *lang)
	}

	files, err := scanFiles(*path, actualFilter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 扫描文件失败: %v\n", err)
		os.Exit(1)
	}

	proc := processor.NewFileProcessor(model.Language(*lang), false, true, *jobs)
	rels, gCtx, err := proc.ProcessFiles(*path, files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 分析失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "[3/4] 💾 正在准备导出...\n")
	os.MkdirAll(*outDir, 0755)

	targetFormat := *format
	if targetFormat == "mermaid" {
		nodeCount := 0
		for _, defs := range gCtx.DefinitionsByQN {
			nodeCount += len(defs)
		}
		if nodeCount > MaxMermaidNodes || len(rels) > MaxMermaidEdges {
			fmt.Fprintf(os.Stderr, "    ⚠️ 数据过大，降级为 jsonl\n")
			targetFormat = "jsonl"
		}
	}

	switch targetFormat {
	case "jsonl":
		exportAsJSONL(*outDir, gCtx, rels, *skipExternal, noiseFilter)
	case "mermaid":
		mermaidPath := filepath.Join(*outDir, "visualization.html")
		output.ExportMermaidHTML(mermaidPath, gCtx, rels, *skipExternal, noiseFilter)
	default:
		exportAsJSONL(*outDir, gCtx, rels, *skipExternal, noiseFilter)
	}

	fmt.Fprintf(os.Stderr, "\n[4/4] ✨ 完成! 耗时: %v\n", time.Since(startTime).Round(time.Millisecond))
}

func exportAsJSONL(outDir string, gCtx *context2.GlobalContext, rels []*model.DependencyRelation, skip bool, nf noisefilter.NoiseFilter) {
	output.ExportElements(filepath.Join(outDir, "element.jsonl"), gCtx)
	output.ExportRelations(filepath.Join(outDir, "relation.jsonl"), rels, gCtx, skip, nf)
}

func scanFiles(root, filter string) ([]string, error) {
	re, _ := regexp.Compile(filter)
	var files []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && re.MatchString(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, nil
}
