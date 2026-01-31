package output

import (
	"encoding/json"
	"io"
)

type JSONLWriter struct {
	encoder *json.Encoder
}

func NewJSONLWriter(w io.Writer) *JSONLWriter {
	return &JSONLWriter{encoder: json.NewEncoder(w)}
}

func (w *JSONLWriter) Write(v interface{}) error { return w.encoder.Encode(v) }
