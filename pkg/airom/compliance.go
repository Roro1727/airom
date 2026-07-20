package airom

// ComplianceResult is one framework's evaluation against the assembled
// inventory: the mapping of its controls onto discovered components, with the
// evidence behind each verdict.
//
// It is a **mapping, never a certification**. A control a static scan cannot
// verify is reported as ControlManual — never silently passed — and a manual
// control carries no score, so the output never asserts a conformance figure
// AIROM cannot back. An `evidence_of` "met" always points at the concrete
// components that satisfy it; a `gap_if` "met" instead asserts the *absence*
// of a gap, so it legitimately carries a score with no evidence to list.
type ComplianceResult struct {
	Framework string           `json:"framework"` // stable id, e.g. "nist-ai-rmf"
	Name      string           `json:"name"`
	Version   string           `json:"version"`
	URL       string           `json:"url,omitempty"`
	Controls  []ControlOutcome `json:"controls"` // sorted by ID (deterministic)
}

// ControlState is a control's verdict.
type ControlState string

const (
	// ControlMet means an evidence_of expression matched (or a gap_if did not).
	ControlMet ControlState = "met"
	// ControlGap means a gap_if expression matched (or an evidence_of did not).
	ControlGap ControlState = "gap"
	// ControlManual means the control is not automatable and requires human
	// attestation. Carries no score.
	ControlManual ControlState = "manual"
)

// ControlOutcome is one control's verdict, its rationale, and the component
// evidence behind it. Evidence and Counter hold component IDs (never fabricated
// — only IDs already in the inventory), so a reader can trace every verdict
// back to the file:line that produced the component.
type ControlOutcome struct {
	ID        string       `json:"id"`             // "MAP-1.1"
	Title     string       `json:"title"`          //
	Text      string       `json:"text,omitempty"` // the control's descriptive text
	Ref       string       `json:"ref,omitempty"`  // the framework's control URL
	State     ControlState `json:"state"`
	Score     *float64     `json:"score,omitempty"`           // 1.0 met / 0.0 gap; nil for manual
	Rationale string       `json:"rationale"`                 // human-readable "why this verdict"
	Evidence  []ID         `json:"evidence,omitempty"`        // component IDs supporting a "met"
	Counter   []ID         `json:"counterEvidence,omitempty"` // component IDs proving a "gap"
}
