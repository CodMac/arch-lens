package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
	"github.com/CodMac/go-treesitter-dependency-analyzer/noisefilter"
)

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

func ExportMermaidHTML(outputPath string, gCtx *model.GlobalContext, rels []*model.DependencyRelation, skipExternal bool, filter noisefilter.NoiseFilter) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, `<!DOCTYPE html><html><head><meta charset="UTF-8"><script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script></head>
<body><div class="mermaid">graph LR`)

	gCtx.RLock()
	for _, fCtx := range gCtx.FileContexts {
		fmt.Fprintf(f, "  subgraph %s [📄 %s]\n", safeID(fCtx.FilePath), fCtx.FilePath)
		for _, entries := range fCtx.DefinitionsBySN {
			for _, entry := range entries {
				nodeID := safeID(entry.Element.QualifiedName)
				fmt.Fprintf(f, "    %s%s\n", nodeID, getNodeShape(entry.Element.Kind, entry.Element.Name))
			}
		}
		fmt.Fprintln(f, "  end")
	}
	gCtx.RUnlock()

	for _, rel := range rels {
		if skipExternal && filter != nil {
			if filter.IsNoise(rel.Target.QualifiedName) {
				continue
			}
			gCtx.RLock()
			_, exists := gCtx.DefinitionsByQN[rel.Target.QualifiedName]
			gCtx.RUnlock()
			if !exists {
				continue
			}
			if rel.Type == model.Parameter || rel.Type == model.Return {
				continue
			}
		}

		srcID, tgtID := safeID(rel.Source.QualifiedName), safeID(rel.Target.QualifiedName)
		if srcID == tgtID {
			continue
		}
		fmt.Fprintf(f, "  %s --> %s\n", srcID, tgtID)
	}

	fmt.Fprintln(f, `</div><script>mermaid.initialize({startOnLoad:true, maxTextSize:1000000});</script></body></html>`)
	return nil
}
