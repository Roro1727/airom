package app

import (
	"strings"
	"testing"
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
		{"a|b|c", 3, []int{1, 1, 1}},
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
