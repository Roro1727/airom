package compliance

import (
	"reflect"
	"testing"

	"github.com/airomhq/airom/pkg/airom"
)

// inv builds a small inventory: a root, a hosted model, a framework, and a
// weights file carrying a risk.
func inv() *airom.Inventory {
	return &airom.Inventory{
		Root: "airom:0000000000000000",
		Components: []airom.Component{
			{ID: "airom:0000000000000000", Kind: airom.KindApplication, Name: "app"},
			{ID: "airom:1111111111111111", Kind: airom.KindHostedLLM, Name: "gpt-4o"},
			{ID: "airom:2222222222222222", Kind: airom.KindFramework, Name: "transformers"},
			{
				ID: "airom:3333333333333333", Kind: airom.KindLocalModelFile, Name: "m.pt",
				Risks: []airom.ArtifactRisk{{ID: airom.RiskUnsafeLoad, Severity: airom.RiskMedium}},
			},
		},
	}
}

// TestEmbeddedSpecsLoad: every shipped framework spec parses, validates, and
// compiles — a broken spec would break the binary.
func TestEmbeddedSpecsLoad(t *testing.T) {
	fws, err := loadFrameworks()
	if err != nil {
		t.Fatalf("embedded specs do not load: %v", err)
	}
	if len(fws) == 0 {
		t.Fatal("no frameworks embedded")
	}
	if _, ok := fws["nist-ai-rmf"]; !ok {
		t.Errorf("nist-ai-rmf not embedded; got %v", IDs())
	}
}

// TestEvaluateStates: met (evidence found), gap (a risk present), and manual
// each resolve correctly, and manual never carries a score.
func TestEvaluateStates(t *testing.T) {
	results, err := Evaluate(inv(), []string{"nist-ai-rmf"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	byID := map[string]airom.ControlOutcome{}
	for _, c := range results[0].Controls {
		byID[c.ID] = c
	}

	// MAP-2.1 is evidence_of models/frameworks → met. The hosted model, the
	// framework, AND the local weights file all match (local-model-file is in
	// the expression), so 3 components supply evidence.
	if c := byID["MAP-2.1"]; c.State != airom.ControlMet || len(c.Evidence) != 3 || c.Score == nil || *c.Score != 1.0 {
		t.Errorf("MAP-2.1 = %+v, want met/score1/3 evidence", c)
	}
	// MEASURE-2.7 is gap_if risk → the weights risk trips it.
	if c := byID["MEASURE-2.7"]; c.State != airom.ControlGap || len(c.Counter) != 1 || c.Score == nil || *c.Score != 0.0 {
		t.Errorf("MEASURE-2.7 = %+v, want gap/score0/1 counter", c)
	}
	// A governance control is manual with NO score — the honesty invariant.
	if c := byID["GOVERN-1.1"]; c.State != airom.ControlManual || c.Score != nil {
		t.Errorf("GOVERN-1.1 = %+v, want manual with nil score", c)
	}
}

// TestGapClears: with no risks in the inventory, a gap_if control flips to met.
func TestGapClears(t *testing.T) {
	in := inv()
	in.Components[3].Risks = nil // drop the risk
	results, _ := Evaluate(in, []string{"nist-ai-rmf"})
	for _, c := range results[0].Controls {
		if c.ID == "MEASURE-2.7" {
			if c.State != airom.ControlMet {
				t.Errorf("MEASURE-2.7 = %s, want met (no risk present)", c.State)
			}
			return
		}
	}
	t.Fatal("MEASURE-2.7 not found")
}

// TestManualNeverScores: NO manual control in any spec ever carries a score.
func TestManualNeverScores(t *testing.T) {
	fws, _ := loadFrameworks()
	for id := range fws {
		res, _ := Evaluate(inv(), []string{id})
		for _, c := range res[0].Controls {
			if c.State == airom.ControlManual && c.Score != nil {
				t.Errorf("%s/%s is manual but has score %v", id, c.ID, *c.Score)
			}
		}
	}
}

// TestUnknownFramework errors, naming the valid set.
func TestUnknownFramework(t *testing.T) {
	if _, err := Evaluate(inv(), []string{"nope"}); err == nil {
		t.Error("unknown framework did not error")
	}
}

// TestDeterministic: evaluation is stable across runs (P7).
func TestDeterministic(t *testing.T) {
	a, _ := Evaluate(inv(), []string{"nist-ai-rmf"})
	b, _ := Evaluate(inv(), []string{"nist-ai-rmf"})
	if !reflect.DeepEqual(a, b) {
		t.Error("evaluation is not deterministic")
	}
}

// TestValidateRejectsBadSpec: the load-time contract rejects malformed specs.
func TestValidateRejectsBadSpec(t *testing.T) {
	cases := []framework{
		{ID: "", Name: "n", Version: "1"},                                                                                                  // no id
		{ID: "x", Name: "n", Version: "1"},                                                                                                 // no controls
		{ID: "x", Name: "n", Version: "1", Controls: []control{{ID: "A", Title: "t"}}},                                                     // no directive
		{ID: "x", Name: "n", Version: "1", Controls: []control{{ID: "A", Title: "t", Manual: true, EvidenceOf: "*"}}},                      // two directives
		{ID: "x", Name: "n", Version: "1", Controls: []control{{ID: "A", Title: "t", EvidenceOf: "not-a-kind"}}},                           // bad expr
		{ID: "x", Name: "n", Version: "1", Controls: []control{{ID: "A", Title: "t", Manual: true}, {ID: "A", Title: "t2", Manual: true}}}, // dup id
	}
	for i := range cases {
		if err := validate(&cases[i]); err == nil {
			t.Errorf("case %d validated but should have failed", i)
		}
	}
}
