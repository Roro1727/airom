// Package compliancew writes the human-readable compliance report
// (docs/compliance.md): each framework's controls as met / gap / manual, with
// the component evidence behind every verdict and a per-framework summary. It
// is a pure projection of inventory.compliance[] (invariant P5) — the mapping
// is computed upstream; this writer only renders it.
package compliancew

import (
	"fmt"
	"io"
	"strings"

	"github.com/airomhq/airom/internal/writer"
	"github.com/airomhq/airom/pkg/airom"
)

func init() {
	writer.Register("compliance", func(writer.Options) writer.Writer { return Writer{} })
}

// Writer renders the Markdown compliance report.
type Writer struct{}

// Format implements writer.Writer.
func (Writer) Format() string { return "compliance" }

// Write emits the report. With no evaluated frameworks it prints a one-line
// hint rather than an empty document, so `-o compliance` without --compliance
// is self-explaining instead of blank.
func (Writer) Write(w io.Writer, inv *airom.Inventory) error {
	fmt.Fprintf(w, "# AI Compliance Report — %s\n\n", inv.Source.Target)

	if len(inv.Compliance) == 0 {
		fmt.Fprintln(w, "_No compliance frameworks were evaluated. Pass `--compliance <framework>` to map this AIBOM._")
		return nil
	}

	fmt.Fprintln(w, "> A mapping, never a certification. `manual` controls are not automatable by a")
	fmt.Fprintln(w, "> static scan and require human attestation; a `met` from evidence points at the")
	fmt.Fprintln(w, "> components that satisfy it.")
	fmt.Fprintln(w)

	byID := componentIndex(inv)
	for _, fw := range inv.Compliance {
		writeFramework(w, fw, byID)
	}
	return nil
}

func writeFramework(w io.Writer, fw airom.ComplianceResult, byID map[airom.ID]*airom.Component) {
	met, gap, manual := 0, 0, 0
	for _, c := range fw.Controls {
		switch c.State {
		case airom.ControlMet:
			met++
		case airom.ControlGap:
			gap++
		case airom.ControlManual:
			manual++
		}
	}

	fmt.Fprintf(w, "## %s %s\n\n", fw.Name, fw.Version)
	fmt.Fprintf(w, "**%d controls — %d met, %d gap, %d manual.**", len(fw.Controls), met, gap, manual)
	if fw.URL != "" {
		fmt.Fprintf(w, " · <%s>", fw.URL)
	}
	fmt.Fprint(w, "\n\n")

	fmt.Fprintln(w, "| Control | State | Evidence |")
	fmt.Fprintln(w, "|---|---|---|")
	for _, c := range fw.Controls {
		fmt.Fprintf(w, "| %s — %s | %s | %s |\n",
			c.ID, mdEscape(c.Title), stateLabel(c.State), evidenceCell(c, byID))
	}
	fmt.Fprintln(w)
}

// stateLabel renders a control state as a stable, scannable marker.
func stateLabel(s airom.ControlState) string {
	switch s {
	case airom.ControlMet:
		return "**MET**"
	case airom.ControlGap:
		return "**GAP**"
	default:
		return "manual"
	}
}

// evidenceCell renders the components behind a verdict, or the rationale when
// there are none (manual, or a gap_if-met asserting absence). Components are
// RESOLVED first, so the "N component(s)" count and the separators always match
// what is actually listed even if an id has no component (defensive — the
// evaluator only ever references present components).
func evidenceCell(c airom.ControlOutcome, byID map[airom.ID]*airom.Component) string {
	ids := c.Evidence
	if len(ids) == 0 {
		ids = c.Counter
	}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		cm, ok := byID[id]
		if !ok {
			continue
		}
		if loc := minLocation(cm); loc != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", mdEscape(cm.Name), loc))
		} else {
			parts = append(parts, mdEscape(cm.Name))
		}
	}
	if len(parts) == 0 {
		return mdEscape(c.Rationale)
	}
	total := len(parts)
	const maxListed = 8
	suffix := ""
	if total > maxListed {
		suffix = fmt.Sprintf("; …(+%d more)", total-maxListed)
		parts = parts[:maxListed]
	}
	return fmt.Sprintf("%d component(s): %s%s", total, strings.Join(parts, "; "), suffix)
}

// componentIndex maps component ID → component for evidence rendering.
func componentIndex(inv *airom.Inventory) map[airom.ID]*airom.Component {
	m := make(map[airom.ID]*airom.Component, len(inv.Components))
	for i := range inv.Components {
		m[inv.Components[i].ID] = &inv.Components[i]
	}
	return m
}

// minLocation returns the smallest (path:line) occurrence of a component, or ""
// if it has none — deterministic across runs.
func minLocation(c *airom.Component) string {
	best := ""
	for _, o := range c.Evidence.Occurrences {
		if o.Location.Path == "" {
			continue
		}
		loc := o.Location.Path
		if o.Location.Line > 0 {
			loc = fmt.Sprintf("%s:%d", o.Location.Path, o.Location.Line)
		}
		if best == "" || loc < best {
			best = loc
		}
	}
	return best
}

// mdEscape neutralizes the Markdown table delimiter so a value containing a
// pipe cannot break the table layout.
func mdEscape(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '|' {
			out = append(out, '\\')
		}
		out = append(out, r)
	}
	return string(out)
}
