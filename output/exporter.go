package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CodMac/arch-lens/core"
	"github.com/CodMac/arch-lens/model"
)

type OutType string

const (
	JsonL   OutType = "jsonl"
	Mermaid OutType = "mermaid"
)

type Exporter struct {
	outputDir  string
	outputType OutType
}

func NewExporter(outputDir string, outputType OutType) *Exporter {
	return &Exporter{outputDir: outputDir, outputType: outputType}
}

func (p *Exporter) ExportJsonL(gCtx *core.GlobalContext, rels []*model.DependencyRelation) (int, int, error) {
	elemPath := filepath.Join(p.outputDir, "element.jsonl")
	relPath := filepath.Join(p.outputDir, "relation.jsonl")

	elemFile, err := os.Create(elemPath)
	if err != nil {
		return 0, 0, err
	}
	defer elemFile.Close()

	relFile, err := os.Create(relPath)
	if err != nil {
		return 0, 0, err
	}
	defer relFile.Close()

	elemWriter := NewJSONLWriter(elemFile)
	elemCount := 0
	// 导出 GlobalContext 中记录的所有定义
	for _, entries := range gCtx.DefinitionsByQN {
		for _, entry := range entries {
			elemWriter.Write(entry.Element)
			elemCount++
		}
	}

	relWriter := NewJSONLWriter(relFile)
	relCount := 0
	for _, rel := range rels {
		relWriter.Write(rel)
		relCount++
	}

	return elemCount, relCount, nil
}

func (p *Exporter) ExportMermaidHTML(gCtx *core.GlobalContext, rels []*model.DependencyRelation) (int, int, error) {
	htmlPath := filepath.Join(p.outputDir, "visualization.html")

	f, err := os.Create(htmlPath)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	fmt.Fprintln(f, `<!DOCTYPE html><html><head><meta charset="UTF-8"><script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script></head>
<body><div class="mermaid">graph LR`)

	elemCount := 0
	// 1. 绘制子图结构 (File -> Elements)
	for _, fCtx := range gCtx.FileContexts {
		fmt.Fprintf(f, "  subgraph %s [📄 %s]\n", safeID(fCtx.FilePath), fCtx.FilePath)
		for _, entries := range fCtx.DefinitionsBySN {
			for _, entry := range entries {
				nodeID := safeID(entry.Element.QualifiedName)
				fmt.Fprintf(f, "    %s%s\n", nodeID, getNodeShape(entry.Element))
				elemCount++
			}
		}
		fmt.Fprintln(f, "  end")
	}

	// 2. 绘制依赖线条
	relCount := 0
	for _, rel := range rels {
		// 跳过包含关系，因为 subgraph 已经体现了
		if rel.Type == model.Contain {
			continue
		}

		srcID, tgtID := safeID(rel.Source.QualifiedName), safeID(rel.Target.QualifiedName)
		if srcID == tgtID {
			continue
		}

		// 如果目标是外部符号且通过了过滤，给它一个特殊样式
		edgeStyle := ""
		if rel.Target.IsFormExternal {
			edgeStyle = "---" // 外部依赖用虚线或不同颜色区分
		}

		fmt.Fprintf(f, "  %s -- %s --> %s%s\n", srcID, rel.Type, tgtID, edgeStyle)
		relCount++
	}

	fmt.Fprintln(f, `</div><script>mermaid.initialize({startOnLoad:true, maxTextSize:1000000});</script></body></html>`)

	return elemCount, relCount, nil
}

func safeID(id string) string {
	r := strings.NewReplacer(".", "_", "(", "_", ")", "_", "[", "_", "]", "_", " ", "_", "@", "at", "$", "_")
	return "n_" + r.Replace(id)
}

func getNodeShape(el *model.CodeElement) string {
	name := el.Name
	if el.IsFormExternal {
		name = name + " (ext)"
	}
	switch el.Kind {
	case model.Interface:
		return fmt.Sprintf("([\"%s <small>(%s)</small>\"])", name, el.Kind)
	case model.Class:
		return fmt.Sprintf("[\"%s <small>(%s)</small>\"]", name, el.Kind)
	case model.Method:
		return fmt.Sprintf("[/\"%s <small>(%s)</small>\"/]", name, el.Kind)
	default:
		return fmt.Sprintf("[\"%s <small>(%s)</small>\"]", name, el.Kind)
	}
}
