// Package nativejson writes the canonical AIROM JSON (docs/mapping.md): a
// direct, versioned, lossless serialization of the Inventory — the
// superset every other format projects from, and the round-trip reference
// (Phase 8). Tri-states serialize as value / null / omitted so the
// Known/Unknown/Absent distinction survives (§6.4).
package nativejson

import (
	"encoding/json"
	"io"

	"github.com/airomhq/airom/internal/writer"
	"github.com/airomhq/airom/pkg/airom"
)

func init() { writer.Register("json", func(writer.Options) writer.Writer { return Writer{} }) }

// Writer renders native airom-json.
type Writer struct{}

// Format implements writer.Writer.
func (Writer) Format() string { return "json" }

// Write emits the Inventory as indented JSON with a trailing newline.
// Deterministic: struct fields serialize in declaration order and the sole
// map (Occurrence.Fields) is key-sorted by encoding/json (P7).
func (Writer) Write(w io.Writer, inv *airom.Inventory) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(inv)
}
