package compliancew_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/airomhq/airom/internal/writer/compliancew"
	"github.com/airomhq/airom/internal/writer/writertest"
	"github.com/airomhq/airom/pkg/airom"
)

var update = flag.Bool("update", false, "update golden files")

// TestGoldenAndDeterminism pins the report shape and its byte-stability (P7).
func TestGoldenAndDeterminism(t *testing.T) {
	inv := writertest.BuildFixture() // carries a nist-ai-rmf compliance result
	var a, b bytes.Buffer
	if err := (compliancew.Writer{}).Write(&a, inv); err != nil {
		t.Fatal(err)
	}
	_ = (compliancew.Writer{}).Write(&b, inv)
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Error("compliance report not deterministic (P7)")
	}

	golden := filepath.Join("testdata", "report.golden.md")
	if *update || os.Getenv("UPDATE_GOLDEN") != "" {
		_ = os.MkdirAll("testdata", 0o750)
		if err := os.WriteFile(golden, a.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run -update): %v", err)
	}
	if !bytes.Equal(want, a.Bytes()) {
		t.Error("compliance report differs from golden; run: go test ./internal/writer/compliancew/... -update")
	}
}

// TestEmptyComplianceHint: with no evaluated frameworks the report is a hint,
// not a blank document.
func TestEmptyComplianceHint(t *testing.T) {
	inv := &airom.Inventory{Source: airom.SourceInfo{Target: "/x"}}
	var buf bytes.Buffer
	if err := (compliancew.Writer{}).Write(&buf, inv); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "--compliance") {
		t.Errorf("empty report should hint at --compliance; got:\n%s", buf.String())
	}
}

// TestPipeInValueDoesNotBreakTable: a pipe in a control title is escaped so it
// cannot break the Markdown table layout.
func TestPipeInValueDoesNotBreakTable(t *testing.T) {
	inv := &airom.Inventory{
		Source: airom.SourceInfo{Target: "/x"},
		Compliance: []airom.ComplianceResult{{
			Framework: "f", Name: "F", Version: "1",
			Controls: []airom.ControlOutcome{
				{ID: "C1", Title: "a | b", State: airom.ControlManual, Rationale: "x | y"},
			},
		}},
	}
	var buf bytes.Buffer
	_ = (compliancew.Writer{}).Write(&buf, inv)
	// The raw unescaped "a | b" must not appear; the escaped form must.
	if strings.Contains(buf.String(), "a | b") {
		t.Error("pipe in title was not escaped")
	}
	if !strings.Contains(buf.String(), `a \| b`) {
		t.Error("escaped pipe not found")
	}
}

// TestEvidenceCountMatchesListed: a dangling evidence id (no component in the
// inventory) must not desync the "N component(s)" count from what is listed or
// leave a doubled separator.
func TestEvidenceCountMatchesListed(t *testing.T) {
	inv := &airom.Inventory{
		Source: airom.SourceInfo{Target: "/x"},
		Components: []airom.Component{
			{ID: "airom:1111111111111111", Kind: airom.KindHostedLLM, Name: "alpha"},
			{ID: "airom:3333333333333333", Kind: airom.KindFramework, Name: "gamma"},
		},
		Compliance: []airom.ComplianceResult{{
			Framework: "f", Name: "F", Version: "1",
			Controls: []airom.ControlOutcome{{
				ID: "C1", Title: "t", State: airom.ControlMet,
				// middle id is absent from Components.
				Evidence: []airom.ID{"airom:1111111111111111", "airom:2222222222222222", "airom:3333333333333333"},
			}},
		}},
	}
	var buf bytes.Buffer
	_ = (compliancew.Writer{}).Write(&buf, inv)
	s := buf.String()
	if strings.Contains(s, "; ;") {
		t.Errorf("doubled separator from a missing evidence id:\n%s", s)
	}
	if !strings.Contains(s, "2 component(s): alpha; gamma") {
		t.Errorf("count should match the 2 resolved components; got:\n%s", s)
	}
}
