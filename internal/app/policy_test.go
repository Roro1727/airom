package app

import (
	"strings"
	"testing"

	"github.com/airomhq/airom/pkg/airom"
)

func TestParsePolicyValid(t *testing.T) {
	cases := []struct {
		expr     string
		wantAnys int   // number of OR clauses
		wantLens []int // terms per clause
	}{
		{"hosted-llm", 1, []int{1}},
		{"pickle-risk", 1, []int{1}},
		{"hosted-llm&confidence>=0.9", 1, []int{2}},
		{"  hosted-llm & confidence >= 0.9 ", 1, []int{2}},
		{"local-model-file|hosted-llm&confidence>=0.8", 2, []int{1, 2}},
		{"prompt|dataset|framework", 3, []int{1, 1, 1}},
		{"confidence<0.5", 1, []int{1}},
		{"confidence=1", 1, []int{1}},
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			p, err := ParsePolicy(tc.expr)
			if err != nil {
				t.Fatalf("ParsePolicy(%q) error: %v", tc.expr, err)
			}
			if got := len(p.anyOf); got != tc.wantAnys {
				t.Fatalf("clauses = %d, want %d", got, tc.wantAnys)
			}
			for i, want := range tc.wantLens {
				if got := len(p.anyOf[i].terms); got != want {
					t.Errorf("clause %d terms = %d, want %d", i, got, want)
				}
			}
			if p.String() != strings.TrimSpace(tc.expr) {
				t.Errorf("String() = %q, want trimmed input", p.String())
			}
		})
	}
}

func TestParsePolicyInvalid(t *testing.T) {
	for _, expr := range []string{
		"",
		"   ",
		"&",
		"hosted-llm&",
		"|hosted-llm",
		"a||b",
		"confidence>>1",
		"confidence>=1.5",
		"confidence>=-0.1",
		"confidence>=",
		"confidence", // bare reserved word: almost certainly a typo'd comparison
		"Hosted-LLM", // uppercase not allowed
		"has space",
		"confidence0.9",
	} {
		t.Run(expr, func(t *testing.T) {
			if _, err := ParsePolicy(expr); err == nil {
				t.Fatalf("ParsePolicy(%q): want error, got nil", expr)
			}
		})
	}
}

func TestMatchAny(t *testing.T) {
	p := MatchAny()
	if p == nil || len(p.anyOf) != 1 {
		t.Fatalf("MatchAny misshapen: %+v", p)
	}
}

// TestParsePolicyRejectsUnknownIdentifiers pins the worst defect this gate ever
// had: an unknown term parsed happily and then never matched, so a one-character
// typo turned a CI gate into a permanent, silent pass.
func TestParsePolicyRejectsUnknownIdentifiers(t *testing.T) {
	for _, expr := range []string{
		"hosted-llmm",           // the typo that started it
		"totalnonsense",         //
		"rules",                 // a detector tag: never matchable here
		"application",           // real kind, but Matches skips the scan root
		"hosted-llm|bogus",      // unknown in the second clause
		"hosted-llm&bogus",      // unknown in a conjunction
		"bogus&confidence>=0.9", //
	} {
		if _, err := ParsePolicy(expr); err == nil {
			t.Errorf("ParsePolicy(%q) = nil error; an unmatched term must fail loudly, not pass forever", expr)
		}
	}
}

// TestConfidenceIsAReservedWordNotAReservedPrefix keeps the distinction the
// grammar draws: "confidencex" is a malformed identifier, not a malformed
// comparison, and its error must say so. (Both are rejected now — the point is
// WHICH diagnostic the user gets.)
func TestConfidenceIsAReservedWordNotAReservedPrefix(t *testing.T) {
	cases := []struct{ expr, wantErrSubstr string }{
		{"confidencex", "unknown term"},
		{"confidence-risk", "unknown term"},
		{"confidence", "bad confidence comparison"},
		{"confidence>=abc", "bad confidence comparison"},
	}
	for _, c := range cases {
		_, err := ParsePolicy(c.expr)
		if err == nil {
			t.Errorf("ParsePolicy(%q) = nil error, want %q", c.expr, c.wantErrSubstr)
			continue
		}
		if !strings.Contains(err.Error(), c.wantErrSubstr) {
			t.Errorf("ParsePolicy(%q) error = %q, want it to mention %q", c.expr, err, c.wantErrSubstr)
		}
	}
}

// TestParsePolicyAcceptsEveryMatchableIdentifier: the validation must not
// over-reach. Every kind termMatches can match has to remain expressible.
func TestParsePolicyAcceptsEveryMatchableIdentifier(t *testing.T) {
	for _, k := range airom.Kinds() {
		if k == airom.KindApplication {
			continue // by design: Matches skips the scan root
		}
		if _, err := ParsePolicy(string(k)); err != nil {
			t.Errorf("ParsePolicy(%q): %v; every matchable kind must be gate-able", k, err)
		}
	}
	for _, expr := range []string{
		"pickle-risk",
		"confidence>=0.9",
		"hosted-llm&confidence>=0.9",
		"local-model-file|hosted-llm&confidence>=0.8",
	} {
		if _, err := ParsePolicy(expr); err != nil {
			t.Errorf("ParsePolicy(%q): %v", expr, err)
		}
	}
}
