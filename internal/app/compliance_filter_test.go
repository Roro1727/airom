package app

import (
	"testing"

	"github.com/airomhq/airom/pkg/airom"
)

// TestPresentationFilterRemapsCompliance is the regression guard for the
// --min-confidence / compliance dangling-evidence defect: after the filter
// drops a low-confidence component, no compliance control may reference a
// component absent from the emitted inventory, and no scored "met" may survive
// with all its evidence filtered away.
func TestPresentationFilterRemapsCompliance(t *testing.T) {
	score := 1.0
	lowConf := airom.ID("airom:1111111111111111") // dropped at min 0.9
	inv := &airom.Inventory{
		SchemaVersion: "1",
		Root:          "airom:0000000000000000",
		Components: []airom.Component{
			{ID: "airom:0000000000000000", Kind: airom.KindApplication, Name: "app", Confidence: 1},
			{ID: lowConf, Kind: airom.KindHostedLLM, Name: "gpt-4o", Confidence: 0.5},
			{ID: "airom:2222222222222222", Kind: airom.KindFramework, Name: "transformers", Confidence: 0.95},
		},
		// Stale overlay referencing the soon-to-be-dropped low-conf component.
		Compliance: []airom.ComplianceResult{{
			Framework: "nist-ai-rmf", Name: "NIST AI Risk Management Framework", Version: "1.0",
			Controls: []airom.ControlOutcome{
				{
					ID: "MAP-2.1", Title: "t", State: airom.ControlMet, Score: &score,
					Rationale: "2 component(s) provide supporting evidence",
					Evidence:  []airom.ID{lowConf, "airom:2222222222222222"},
				},
			},
		}},
	}

	out := presentationFilter(inv, &Config{MinConfidence: 0.9, Compliance: []string{"nist-ai-rmf"}})

	alive := map[airom.ID]bool{}
	for _, c := range out.Components {
		alive[c.ID] = true
	}
	if alive[lowConf] {
		t.Fatal("precondition: low-confidence component should have been filtered")
	}
	if len(out.Compliance) == 0 {
		t.Fatal("compliance overlay was dropped entirely")
	}
	for _, cr := range out.Compliance {
		for _, ctl := range cr.Controls {
			for _, id := range append(append([]airom.ID{}, ctl.Evidence...), ctl.Counter...) {
				if !alive[id] {
					t.Errorf("control %s references %s, absent from emitted components", ctl.ID, id)
				}
			}
			// A met must still be backed: no scored evidence_of-met with empty evidence.
			if ctl.State == airom.ControlMet && ctl.Score != nil && *ctl.Score == 1.0 &&
				len(ctl.Evidence) == 0 && len(ctl.Counter) == 0 {
				// Allowed only for gap_if-derived met (absence of a gap). MAP-2.1
				// is evidence_of, so here it must retain the surviving framework.
				if ctl.ID == "MAP-2.1" {
					t.Errorf("MAP-2.1 is met/scored but has no surviving evidence")
				}
			}
		}
	}
}
