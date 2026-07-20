package airom

import "testing"

// TestRiskByIDKnown: catalog entries resolve to their fixed metadata.
func TestRiskByIDKnown(t *testing.T) {
	m := RiskByID(RiskPickleImport)
	if m.Severity != RiskHigh || m.Slug != "pickle-import" {
		t.Errorf("pickle-import = %+v, want high/pickle-import", m)
	}
}

// TestRiskByIDUnknownDistinctSlugs: two distinct out-of-catalog ids must NOT
// collapse to the same slug — otherwise the SARIF writer emits duplicate rule
// ids. The fallback slug is derived from the id.
func TestRiskByIDUnknownDistinctSlugs(t *testing.T) {
	a := RiskByID("AIROM-RISK-FUTURE-ONE")
	b := RiskByID("AIROM-RISK-FUTURE-TWO")
	if a.Slug == b.Slug {
		t.Fatalf("distinct unknown ids share slug %q", a.Slug)
	}
	if a.Severity != RiskLow || b.Severity != RiskLow {
		t.Errorf("unknown severities = %s/%s, want low/low", a.Severity, b.Severity)
	}
	// Slugs are slug-safe (lowercase, dash-separated, no leading/trailing dash).
	for _, s := range []string{a.Slug, b.Slug} {
		if s == "" || s[0] == '-' || s[len(s)-1] == '-' {
			t.Errorf("slug %q is not slug-safe", s)
		}
	}
}

// TestRiskSlugRoundTrip: catalog slugs invert back to their id.
func TestRiskSlugRoundTrip(t *testing.T) {
	id, ok := RiskSlugToID("pickle-import")
	if !ok || id != RiskPickleImport {
		t.Errorf("slug lookup = %s,%v, want pickle-import id", id, ok)
	}
	if _, ok := RiskSlugToID("nope"); ok {
		t.Error("unknown slug resolved")
	}
}
