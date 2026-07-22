package osv

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/airomhq/airom/pkg/airom"
)

// TestCVSSv3Score checks the base-score formula against published CVSS v3.1
// examples (FIRST calculator reference values).
func TestCVSSv3Score(t *testing.T) {
	cases := []struct {
		vector string
		want   float64
	}{
		{"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", 9.8},  // critical
		{"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H", 10.0}, // scope changed
		{"CVSS:3.1/AV:L/AC:H/PR:H/UI:R/S:U/C:L/I:N/A:N", 1.8},  // low
		{"CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", 8.8},  // high, v3.0
	}
	for _, c := range cases {
		got, ok := cvssV3Score(c.vector)
		if !ok || math.Abs(got-c.want) > 0.05 {
			t.Errorf("cvssV3Score(%q) = %v,%v want %v", c.vector, got, ok, c.want)
		}
	}
	if _, ok := cvssV3Score("CVSS:2.0/AV:N/AC:L/Au:N/C:P/I:P/A:P"); ok {
		t.Error("v2 vector should not parse as v3")
	}
	if _, ok := cvssV3Score("not-a-vector"); ok {
		t.Error("garbage should not parse")
	}
	// A truncated v3 vector missing its impact metrics must NOT silently score 0
	// (which would mask a text severity of, say, CRITICAL). It must fall back.
	if _, ok := cvssV3Score("CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U"); ok {
		t.Error("a vector missing C/I/A must not parse as a valid v3 score")
	}
}

// TestFirstFixedIsVersionAware checks the multi-range fix: the reported fix is
// the next release on the installed line, not whichever range comes first.
func TestFirstFixedIsVersionAware(t *testing.T) {
	// A backport line (< 1.5.0) and the current line (>= 2.0.0, < 2.3.0).
	affected := []osvAffected{{Ranges: []osvRange{
		{Type: "ECOSYSTEM", Events: []map[string]string{{"introduced": "0"}, {"fixed": "1.5.0"}}},
		{Type: "ECOSYSTEM", Events: []map[string]string{{"introduced": "2.0.0"}, {"fixed": "2.3.0"}}},
	}}}
	cases := []struct {
		version, want string
	}{
		{"2.1.0", "2.3.0"}, // on the 2.x line → the 2.x fix, not 1.5.0
		{"1.2.0", "1.5.0"}, // on the 1.x line → the 1.x fix
		{"", "1.5.0"},      // unknown version → first real fixed (fallback)
	}
	for _, c := range cases {
		if got := firstFixed(affected, c.version); got != c.want {
			t.Errorf("firstFixed(version=%q) = %q, want %q", c.version, got, c.want)
		}
	}
	// A GIT-only range yields the commit hash only as a last resort.
	git := []osvAffected{{Ranges: []osvRange{
		{Type: "GIT", Events: []map[string]string{{"introduced": "0"}, {"fixed": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"}}},
	}}}
	if got := firstFixed(git, "1.0.0"); got != "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef" {
		t.Errorf("GIT-only fixed = %q, want the commit hash fallback", got)
	}
}

func TestParseAndCompareVersions(t *testing.T) {
	if _, ok := parseVersion("deadbeef"); ok {
		t.Error("a commit hash must not parse as a version")
	}
	if v, ok := parseVersion("v1.0.0-rc1"); !ok || len(v) != 3 || v[0] != 1 || v[2] != 0 {
		t.Errorf("parseVersion(v1.0.0-rc1) = %v,%v; want [1 0 0] (pre-release dropped)", v, ok)
	}
	cmp := []struct {
		a, b string
		want int
	}{
		{"2.3.0", "2.1.0", 1},
		{"1.5", "1.5.0", 0}, // trailing zero equals
		{"1.0.0", "2.0.0", -1},
		{"4.36.0", "4.9.0", 1}, // numeric, not lexical (36 > 9)
	}
	for _, c := range cmp {
		av, _ := parseVersion(c.a)
		bv, _ := parseVersion(c.b)
		if got := compareVersions(av, bv); got != c.want {
			t.Errorf("compareVersions(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// TestQueryAndParse drives Enrich against an httptest OSV stand-in — no live
// network — and checks the CVE mapping (CVE-alias preferred, CVSS→severity,
// fixed version, url).
func TestQueryAndParse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Package struct{ Purl string } `json:"package"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Package.Purl != "pkg:pypi/langchain@0.2.1" {
			_, _ = w.Write([]byte(`{}`)) // unknown purl -> no vulns
			return
		}
		_, _ = w.Write([]byte(`{"vulns":[
          {"id":"GHSA-aaaa-bbbb-cccc","summary":"SSRF in langchain",
           "aliases":["CVE-2024-9999"],
           "severity":[{"type":"CVSS_V3","score":"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"}],
           "affected":[{"ranges":[{"events":[{"introduced":"0"},{"fixed":"0.2.5"}]}]}]}
        ]}`))
	}))
	defer srv.Close()

	inv := &airom.Inventory{Components: []airom.Component{
		{ID: "airom:1", Kind: airom.KindFramework, Name: "langchain", PURL: "pkg:pypi/langchain@0.2.1"},
		{ID: "airom:2", Kind: airom.KindHostedLLM, Name: "gpt-4o"},                                               // no purl -> skipped
		{ID: "airom:3", Kind: airom.KindLocalModelFile, Name: "m.gguf", PURL: "pkg:generic/m?checksum=sha256:x"}, // generic -> skipped
	}}
	Enrich(context.Background(), inv, Options{Endpoint: srv.URL, HTTP: srv.Client()})

	v := inv.Components[0].Vulnerabilities
	if len(v) != 1 {
		t.Fatalf("langchain vulns = %d, want 1", len(v))
	}
	got := v[0]
	if got.ID != "CVE-2024-9999" {
		t.Errorf("id = %q, want the CVE alias", got.ID)
	}
	if got.Severity != airom.VulnCritical || math.Abs(got.Score-9.8) > 0.05 {
		t.Errorf("severity/score = %s/%v, want critical/9.8", got.Severity, got.Score)
	}
	if got.Fixed != "0.2.5" || got.Source != "osv.dev" || got.URL == "" {
		t.Errorf("fixed/source/url = %q/%q/%q", got.Fixed, got.Source, got.URL)
	}
	if len(inv.Components[1].Vulnerabilities) != 0 || len(inv.Components[2].Vulnerabilities) != 0 {
		t.Error("non-package components should not be queried")
	}
}

// TestDegradesOnFailure: a failing OSV endpoint yields no CVEs and a warning,
// not a crash or a failed scan.
func TestDegradesOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	inv := &airom.Inventory{Components: []airom.Component{
		{ID: "airom:1", Kind: airom.KindFramework, Name: "x", PURL: "pkg:pypi/x@1.0"},
	}}
	failed := Enrich(context.Background(), inv, Options{Endpoint: srv.URL, HTTP: srv.Client()})
	if failed != 1 {
		t.Errorf("Enrich returned failed = %d, want 1 (so a CVE gate can fail closed)", failed)
	}
	if len(inv.Components[0].Vulnerabilities) != 0 {
		t.Error("expected no vulns on failure")
	}
	if len(inv.Stats.Warnings) == 0 {
		t.Error("expected a degradation warning")
	}
}
