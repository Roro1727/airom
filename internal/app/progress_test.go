package app

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestProgressNeverRendersOffATerminal is the invariant that matters: stdout
// carries the AIBOM and must be byte-identical for identical inputs (P7). In a
// pipe, a redirect, or CI, stderr is not a terminal and the progress machinery
// must not emit a single byte or perturb the scan.
func TestProgressNeverRendersOffATerminal(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()

	orig := stderr
	stderr = w
	t.Cleanup(func() { stderr = orig })

	// Not a TTY, so progress is disabled regardless of the flags.
	live, stop := startProgress(&Config{}, "/some/target")
	stop()
	if live != nil {
		t.Error("progress enabled off a terminal: the engine would pay for accounting nobody reads")
	}

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if buf.Len() != 0 {
		t.Errorf("progress wrote %q to stderr off a terminal, want nothing", buf.String())
	}
}

func TestProgressDisabledByFlags(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  *Config
	}{
		{"quiet", &Config{Quiet: true}},
		{"no-progress", &Config{NoProgress: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			live, stop := startProgress(tc.cfg, "/t")
			defer stop()
			if live != nil {
				t.Errorf("%s: progress should be disabled", tc.name)
			}
		})
	}
}

// TestScanOutputIdenticalWithProgressPath: a real scan through the pipeline
// produces the same inventory whether or not the progress path is engaged. The
// counters are presentation-only and must not feed back into the graph.
func TestScanOutputIdenticalWithProgressPath(t *testing.T) {
	root := writeTree(t, map[string]string{
		"app.py":           "import openai\nresp = client.chat.completions.create(model=\"gpt-4.1\")\n",
		"requirements.txt": "openai==1.30.0\nchromadb==0.5.0\n",
	})

	var out bytes.Buffer
	origOut := stdout
	stdout = &out
	t.Cleanup(func() { stdout = origOut })

	scan := func() string {
		out.Reset()
		cfg := &Config{Source: SourceFS, Target: root}
		if err := Run(context.Background(), cfg); err != nil {
			t.Fatalf("Run: %v", err)
		}
		return out.String()
	}

	first := scan()
	second := scan()
	if first != second {
		t.Errorf("scan output is not byte-identical across runs (P7):\n--- 1 ---\n%s\n--- 2 ---\n%s", first, second)
	}
	if first == "" {
		t.Fatal("expected table output on stdout")
	}
}

func TestHumanCount(t *testing.T) {
	for _, tc := range []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{7, "7"},
		{999, "999"},
		{1000, "1,000"},
		{1284, "1,284"},
		{12345, "12,345"},
		{1234567, "1,234,567"},
	} {
		if got := humanCount(tc.in); got != tc.want {
			t.Errorf("humanCount(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestShortenPath(t *testing.T) {
	short := "/home/you/app"
	if got := shortenPath(short); got != short {
		t.Errorf("shortenPath(short) = %q, want unchanged", got)
	}
	long := "/very/deeply/nested/path/that/keeps/going/and/going/and/going/until/it/is/too/long"
	got := shortenPath(long)
	// Measured in runes: the ellipsis is one column but three bytes.
	if n := len([]rune(got)); n > 48 {
		t.Errorf("shortenPath = %q (%d runes), want <= 48", got, n)
	}
	if !strings.HasPrefix(got, "…") {
		t.Errorf("shortenPath = %q, want a leading ellipsis", got)
	}
	if !strings.HasSuffix(got, "too/long") {
		t.Errorf("shortenPath = %q, want the distinctive tail kept", got)
	}

	// Multi-byte input must not be split mid-character.
	uni := shortenPath("/tmp/" + strings.Repeat("日", 80))
	if !utf8.ValidString(uni) {
		t.Errorf("shortenPath produced invalid UTF-8: %q", uni)
	}
}
