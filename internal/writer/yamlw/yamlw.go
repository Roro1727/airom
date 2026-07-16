// Package yamlw writes the native model as YAML (docs/mapping.md): the same
// projection as native JSON, rendered through the JSON representation so the
// key names and tri-state null handling match byte-for-byte in structure.
package yamlw

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/Roro1727/airom/internal/writer"
	"github.com/Roro1727/airom/pkg/airom"
)

func init() { writer.Register("yaml", func(writer.Options) writer.Writer { return Writer{} }) }

// Writer renders native YAML.
type Writer struct{}

// Format implements writer.Writer.
func (Writer) Format() string { return "yaml" }

// Write emits the Inventory as YAML. It routes through JSON so the tri-state
// custom marshalers and the exact native key names apply; yaml.v3 then
// key-sorts maps for determinism (P7).
func (Writer) Write(w io.Writer, inv *airom.Inventory) error {
	data, err := json.Marshal(inv)
	if err != nil {
		return fmt.Errorf("marshal inventory: %w", err)
	}
	var generic any
	if err := json.Unmarshal(data, &generic); err != nil {
		return fmt.Errorf("reparse inventory: %w", err)
	}
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(generic); err != nil {
		return err
	}
	return enc.Close()
}
