package cli

import (
	"strings"
	"testing"

	"github.com/Roro1727/airom/internal/app"
)

func TestParseOutputSpecs(t *testing.T) {
	specs, err := parseOutputSpecs([]string{"table", "cyclonedx=bom.json", "sarif=out.sarif"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []app.OutputSpec{
		{Format: app.FormatTable},
		{Format: app.FormatCycloneDX, Path: "bom.json"},
		{Format: app.FormatSARIF, Path: "out.sarif"},
	}
	if len(specs) != len(want) {
		t.Fatalf("got %d specs, want %d", len(specs), len(want))
	}
	for i := range want {
		if specs[i] != want[i] {
			t.Errorf("spec[%d] = %+v, want %+v", i, specs[i], want[i])
		}
	}
}

func TestParseOutputSpecsErrors(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want string
	}{
		{"unknown format", []string{"xml"}, "unknown output format"},
		{"two stdout", []string{"table", "json"}, "stdout"},
		{"empty spec", []string{""}, "empty output spec"},
		{"empty path", []string{"json="}, "empty path"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseOutputSpecs(tc.in); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("parseOutputSpecs(%v) err = %v, want containing %q", tc.in, err, tc.want)
			}
		})
	}
}
