package cli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/knadh/koanf/v2"
)

// sizeRe accepts "1048576", "512k", "256m", "2g", with an optional trailing
// "b"/"B" (docs/cli.md: <size> values take k/m/g suffixes).
var sizeRe = regexp.MustCompile(`^([0-9]+)\s*([kKmMgG]?)[bB]?$`)

// parseSize converts a human size string to bytes (binary multiples).
func parseSize(s string) (int64, error) {
	m := sizeRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, fmt.Errorf("invalid size %q (want e.g. 512k, 256m, 2g)", s)
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q: %w", s, err)
	}
	var shift uint
	switch strings.ToLower(m[2]) {
	case "k":
		shift = 10
	case "m":
		shift = 20
	case "g":
		shift = 30
	}
	if shift > 0 && n > (1<<(63-shift))-1 {
		return 0, fmt.Errorf("size %q overflows", s)
	}
	return n << shift, nil
}

// formatSize is the inverse of parseSize for exact binary multiples; it
// renders the app-level default constants as flag-default strings so the
// two never drift.
func formatSize(n int64) string {
	switch {
	case n >= 1<<30 && n%(1<<30) == 0:
		return strconv.FormatInt(n>>30, 10) + "g"
	case n >= 1<<20 && n%(1<<20) == 0:
		return strconv.FormatInt(n>>20, 10) + "m"
	case n >= 1<<10 && n%(1<<10) == 0:
		return strconv.FormatInt(n>>10, 10) + "k"
	default:
		return strconv.FormatInt(n, 10)
	}
}

// parseSizeKey reads a size-typed configuration key, attributing errors to
// the flag name.
func parseSizeKey(k *koanf.Koanf, key string) (int64, error) {
	v := k.String(key)
	if v == "" {
		return 0, nil // ApplyDefaults fills the documented default
	}
	n, err := parseSize(v)
	if err != nil {
		return 0, fmt.Errorf("--%s: %w", key, err)
	}
	return n, nil
}
