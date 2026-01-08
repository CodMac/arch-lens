package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/core"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
)

type OutType string

const (
	JsonL   OutType = "jsonl"
	Mermaid OutType = "mermaid"
)

type Exporter struct {
	outputDir    string
	outputType   OutType
	skipExternal bool
	filter       core.NoiseFilter
}

func NewExporter(outputDir string, outputType OutType, skipExternal bool, filter core.NoiseFilter) *Exporter {
	return &Exporter{outputDir: outputDir, outputType: outputType, skipExternal: skipExternal, filter: filter}
}

func (p *Exporter) ExportJsonL(gCtx *core.GlobalContext, rels []*model.DependencyRelation) (int, int, error) {
	elemPath := filepath.Join(p.outputDir, "element.jsonl")
	relPath := filepath.Join(p.outputDir, "relation.jsonl")

	// 导出
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
	for _, entries := range gCtx.DefinitionsByQN {
		for _, entry := range entries {
			elemWriter.Write(entry.Element)
			elemCount++
		}
	}

	relWriter := NewJSONLWriter(relFile)
	relCount := 0
	for _, rel := range rels {
		if p.skipExternal && p.filter != nil {
			// 过滤逻辑待补充
			if p.filter.IsNoise(rel.Target.QualifiedName) {
				continue
			}

			_, exists := gCtx.DefinitionsByQN[rel.Target.QualifiedName]
			if !exists {
				continue
			}
		}

		relWriter.Write(rel)
		relCount++
	}

	return elemCount, relCount, nil
}

func (p *Exporter) ExportMermaidHTML(gCtx *core.GlobalContext, rels []*model.DependencyRelation) (int, int, error) {
	htmlPath := filepath.Join(p.outputDir, "visualization.html")

	// 导出
	f, err := os.Create(htmlPath)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	fmt.Fprintln(f, `<!DOCTYPE html><html><head><meta charset="UTF-8"><script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script></head>
<body><div class="mermaid">graph LR`)

	elemCount := 0
	for _, fCtx := range gCtx.FileContexts {
		fmt.Fprintf(f, "  subgraph %s [📄 %s]\n", safeID(fCtx.FilePath), fCtx.FilePath)
		for _, entries := range fCtx.DefinitionsBySN {
			for _, entry := range entries {
				nodeID := safeID(entry.Element.QualifiedName)
				fmt.Fprintf(f, "    %s%s\n", nodeID, getNodeShape(entry.Element.Kind, entry.Element.Name))

				elemCount++
			}
		}
		fmt.Fprintln(f, "  end")
	}

	relCount := 0
	for _, rel := range rels {
		if p.skipExternal && p.filter != nil {
			if p.filter.IsNoise(rel.Target.QualifiedName) {
				continue
			}

			_, exists := gCtx.DefinitionsByQN[rel.Target.QualifiedName]
			if !exists {
				continue
			}
		}

		srcID, tgtID := safeID(rel.Source.QualifiedName), safeID(rel.Target.QualifiedName)
		if srcID == tgtID {
			continue
		}

		fmt.Fprintf(f, "  %s -- %s --> %s\n", srcID, rel.Type, tgtID)

		relCount++
	}

	fmt.Fprintln(f, `</div><script>mermaid.initialize({startOnLoad:true, maxTextSize:1000000});</script></body></html>`)

	return elemCount, relCount, nil
}

// 辅助函数

func safeID(id string) string {
	r := strings.NewReplacer(".", "_", "(", "_", ")", "_", "[", "_", "]", "_", " ", "_", "@", "at")
	return "n_" + r.Replace(id)
}

func getNodeShape(kind model.ElementKind, name string) string {
	switch kind {
	case model.Interface:
		return fmt.Sprintf("([\"%s <small>(%s)</small>\"])", name, kind)
	case model.Class:
		return fmt.Sprintf("[\"%s <small>(%s)</small>\"]", name, kind)
	case model.Method:
		return fmt.Sprintf("[/\"%s <small>(%s)</small>\"/]", name, kind)
	default:
		return fmt.Sprintf("[\"%s <small>(%s)</small>\"]", name, kind)
	}
}
