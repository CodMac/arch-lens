package output

import (
	"encoding/json"
	"io"
	"os"

	"github.com/CodMac/go-treesitter-dependency-analyzer/core"
	"github.com/CodMac/go-treesitter-dependency-analyzer/model"
)

type JSONLWriter struct {
	encoder *json.Encoder
}

func NewJSONLWriter(w io.Writer) *JSONLWriter {
	return &JSONLWriter{encoder: json.NewEncoder(w)}
}

func (w *JSONLWriter) Write(v interface{}) error { return w.encoder.Encode(v) }

func ExportElements(path string, gCtx *core.GlobalContext) (int, error) {
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	writer := NewJSONLWriter(f)
	count := 0
	for _, entries := range gCtx.DefinitionsByQN {
		for _, entry := range entries {
			writer.Write(entry.Element)
			count++
		}
	}
	return count, nil
}

func ExportRelations(path string, rels []*model.DependencyRelation, gCtx *core.GlobalContext, skipExternal bool, filter core.NoiseFilter) (int, error) {
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	writer := NewJSONLWriter(f)
	count := 0
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
		}

		writer.Write(rel)
		count++
	}
	return count, nil
}
