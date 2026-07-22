package airom

import "testing"

// TestSeverityFromScore pins the CVSS v3.1 qualitative bands at their exact
// boundaries — the cut points (9.0, 7.0, 4.0, 0.0) are where an off-by-a-tenth
// bug would silently misbucket an advisory.
func TestSeverityFromScore(t *testing.T) {
	cases := []struct {
		score float64
		want  VulnSeverity
	}{
		{10.0, VulnCritical},
		{9.0, VulnCritical},
		{8.9, VulnHigh},
		{7.0, VulnHigh},
		{6.9, VulnMedium},
		{4.0, VulnMedium},
		{3.9, VulnLow},
		{0.1, VulnLow},
		{0.0, VulnUnknown},
	}
	for _, tc := range cases {
		if got := SeverityFromScore(tc.score); got != tc.want {
			t.Errorf("SeverityFromScore(%.1f) = %q, want %q", tc.score, got, tc.want)
		}
	}
}

// TestVulnSeveritiesDescending keeps the list in descending order — gate
// validation and threshold ranking both rely on it.
func TestVulnSeveritiesDescending(t *testing.T) {
	got := VulnSeverities()
	want := []VulnSeverity{VulnCritical, VulnHigh, VulnMedium, VulnLow, VulnUnknown}
	if len(got) != len(want) {
		t.Fatalf("VulnSeverities() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("VulnSeverities()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
