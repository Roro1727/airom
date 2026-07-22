package osv

import (
	"math"
	"strings"
)

// cvssV3Score computes the CVSS v3.0/v3.1 base score from a vector string per
// the FIRST specification. Returns (score, true) for a well-formed CVSS:3.x
// vector, or (0, false) otherwise (v2/v4 vectors, or garbage) — the caller
// then falls back to the advisory's textual severity.
func cvssV3Score(vector string) (float64, bool) {
	if !strings.HasPrefix(vector, "CVSS:3.0/") && !strings.HasPrefix(vector, "CVSS:3.1/") {
		return 0, false
	}
	m := map[string]string{}
	for _, part := range strings.Split(vector, "/")[1:] {
		if k, v, ok := strings.Cut(part, ":"); ok {
			m[k] = v
		}
	}

	// A well-formed CVSS:3.x base vector carries all eight mandatory metrics.
	// If any is absent the vector is truncated/garbled — bail out so the caller
	// falls back to the advisory's textual severity rather than trusting a score
	// computed from defaulted metrics. (C/I/A need an explicit presence check:
	// their "N" weight is a legitimate 0.0, indistinguishable from missing.)
	for _, k := range []string{"AV", "AC", "PR", "UI", "S", "C", "I", "A"} {
		if _, ok := m[k]; !ok {
			return 0, false
		}
	}

	av := pick(map[string]float64{"N": 0.85, "A": 0.62, "L": 0.55, "P": 0.20}, m["AV"])
	ac := pick(map[string]float64{"L": 0.77, "H": 0.44}, m["AC"])
	ui := pick(map[string]float64{"N": 0.85, "R": 0.62}, m["UI"])
	c := pick(map[string]float64{"H": 0.56, "L": 0.22, "N": 0.0}, m["C"])
	i := pick(map[string]float64{"H": 0.56, "L": 0.22, "N": 0.0}, m["I"])
	a := pick(map[string]float64{"H": 0.56, "L": 0.22, "N": 0.0}, m["A"])
	scopeChanged := m["S"] == "C"
	var pr float64
	if scopeChanged {
		pr = pick(map[string]float64{"N": 0.85, "L": 0.68, "H": 0.50}, m["PR"])
	} else {
		pr = pick(map[string]float64{"N": 0.85, "L": 0.62, "H": 0.27}, m["PR"])
	}
	if av == 0 || ac == 0 || ui == 0 || pr == 0 { // a required base metric was missing
		return 0, false
	}

	iss := 1 - (1-c)*(1-i)*(1-a)
	var impact float64
	if scopeChanged {
		impact = 7.52*(iss-0.029) - 3.25*math.Pow(iss-0.02, 15)
	} else {
		impact = 6.42 * iss
	}
	if impact <= 0 {
		return 0, true
	}
	expl := 8.22 * av * ac * pr * ui
	base := impact + expl
	if scopeChanged {
		base = 1.08 * base
	}
	return roundup(math.Min(base, 10)), true
}

func pick(t map[string]float64, k string) float64 { return t[k] }

// roundup is the CVSS v3.1 rounding: the smallest value to one decimal place
// that is >= input, computed in integer space to avoid float drift.
func roundup(input float64) float64 {
	n := int(math.Round(input * 100000))
	if n%10000 == 0 {
		return float64(n) / 100000
	}
	return float64(n/10000+1) / 10
}
