package airom

// Vulnerability is a known CVE affecting a component, matched by its purl
// against a vulnerability database (OSV.dev). Unlike an ArtifactRisk — which is
// AIROM's own statically-detected code-execution finding — a Vulnerability is a
// third-party advisory with a real id and CVSS score, so it projects into
// CycloneDX vulnerabilities[] with a genuine CVSSv3 rating (not method "other").
//
// The CVE overlay is opt-in (`--cve`): it queries a live database, so unlike the
// rest of AIROM it is neither offline nor deterministic across time (the same
// scan yields more CVEs as the database grows). It is scoped to the AI packages
// AIROM already inventories — it is not a general-purpose SCA scanner.
type Vulnerability struct {
	ID       string       `json:"id"`                     // CVE-YYYY-NNNN preferred; else the OSV/GHSA id
	Aliases  []string     `json:"aliases,omitempty"`      // the other ids for the same advisory
	Severity VulnSeverity `json:"severity"`               // from the CVSS base score, or "unknown"
	Score    float64      `json:"score,omitempty"`        // CVSS base score in [0,10], 0 when unknown
	Vector   string       `json:"vector,omitempty"`       // the CVSS vector string
	Summary  string       `json:"summary,omitempty"`      // one-line advisory summary
	Fixed    string       `json:"fixedVersion,omitempty"` // the first fixed version, when the advisory names one
	Source   string       `json:"source"`                 // "osv.dev"
	URL      string       `json:"url,omitempty"`          // advisory URL
}

// VulnSeverity is the CVSS-derived severity bucket. Includes "critical" (which
// artifact risks do not) and "unknown" (an advisory with no parseable score).
type VulnSeverity string

// The CVSS v3.1 qualitative severity buckets, plus "unknown" for an advisory
// with no parseable score.
const (
	VulnCritical VulnSeverity = "critical"
	VulnHigh     VulnSeverity = "high"
	VulnMedium   VulnSeverity = "medium"
	VulnLow      VulnSeverity = "low"
	VulnUnknown  VulnSeverity = "unknown"
)

// VulnSeverities lists the buckets in descending order, for gate validation.
func VulnSeverities() []VulnSeverity {
	return []VulnSeverity{VulnCritical, VulnHigh, VulnMedium, VulnLow, VulnUnknown}
}

// SeverityFromScore buckets a CVSS base score (CVSS v3.1 qualitative bands).
func SeverityFromScore(score float64) VulnSeverity {
	switch {
	case score >= 9.0:
		return VulnCritical
	case score >= 7.0:
		return VulnHigh
	case score >= 4.0:
		return VulnMedium
	case score > 0.0:
		return VulnLow
	default:
		return VulnUnknown
	}
}
