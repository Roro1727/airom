// Package tablew writes the human-readable table (ARCHITECTURE.md §11): a
// scannable summary sorted by kind then name. The scan-root application
// component is omitted (it is metadata, not a finding). Verbose mode expands
// the file:line occurrences under each row.
package tablew

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/airomhq/airom/internal/writer"
	"github.com/airomhq/airom/pkg/airom"
)

func init() {
	writer.Register("table", func(o writer.Options) writer.Writer { return Writer{wide: o.TableWide} })
}

// Writer renders the table format.
type Writer struct{ wide bool }

// Format implements writer.Writer.
func (Writer) Format() string { return "table" }

// Write emits the component table.
func (t Writer) Write(w io.Writer, inv *airom.Inventory) error {
	comps := make([]airom.Component, 0, len(inv.Components))
	for _, c := range inv.Components {
		if c.Kind == airom.KindApplication {
			continue
		}
		comps = append(comps, c)
	}
	sort.SliceStable(comps, func(i, j int) bool {
		if comps[i].Kind != comps[j].Kind {
			return comps[i].Kind < comps[j].Kind
		}
		return comps[i].Name < comps[j].Name
	})

	if len(comps) == 0 {
		fmt.Fprintf(w, "No AI components found in %s.\n", inv.Source.Target)
		return nil
	}

	fmt.Fprintf(w, "AI Bill of Materials — %s\n", inv.Source.Target)
	fmt.Fprintf(w, "%d component(s), %d relationship(s)\n\n", len(comps), len(inv.Relationships))

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "KIND\tNAME\tVERSION\tPROVIDER\tCONF\tEVIDENCE")
	for _, c := range comps {
		version := ""
		if v, ok := c.Version.Value(); ok {
			version = v
		}
		provider := ""
		if p, ok := c.Provider.Value(); ok {
			provider = p
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%d occ\n",
			c.Kind, name(c), dash(version), dash(provider),
			writer.FormatConfidence(c.Confidence), len(c.Evidence.Occurrences))
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if t.wide {
		fmt.Fprintln(w)
		for _, c := range comps {
			fmt.Fprintf(w, "%s %s:\n", c.Kind, name(c))
			for _, o := range c.Evidence.Occurrences {
				loc := o.Location.Path
				if o.Location.Line > 0 {
					loc = fmt.Sprintf("%s:%d", o.Location.Path, o.Location.Line)
				}
				fmt.Fprintf(w, "    %s  [%s]\n", loc, o.DetectorID)
			}
		}
	}

	if n := len(inv.Unknowns); n > 0 {
		fmt.Fprintf(w, "\n%d file(s) could not be fully processed (see --stats or the json output).\n", n)
	}
	return nil
}

func name(c airom.Component) string {
	if c.Group != "" {
		return c.Group + "/" + c.Name
	}
	return c.Name
}

func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
