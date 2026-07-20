package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"github.com/airomhq/airom/internal/app"
)

// addGlobalFlags registers the global scan flags from docs/cli.md ("Global
// flags") on the root's persistent flag set. Flag defaults double as the
// bottom layer of the configuration precedence (flags > env > file >
// defaults): the posflag provider only overrides file/env values when a
// flag was explicitly set.
func addGlobalFlags(fs *pflag.FlagSet) {
	fs.StringArrayP("output", "o", nil,
		fmt.Sprintf("output as fmt[=path]; repeatable; formats: %s (default table to stdout)",
			strings.Join(app.Formats(), ", ")))
	fs.String("format", "", "single-format alias for -o (mutually exclusive with -o)")
	fs.String("select", "", `detector selection expression, e.g. "rules,+modelfile/gguf,-dataset/file"`)
	fs.StringArray("rules", nil, "overlay rule pack file; repeatable; merged by rule ID")
	fs.StringArray("compliance", nil,
		fmt.Sprintf("map the AIBOM onto a governance framework; repeatable; frameworks: %s",
			strings.Join(app.ComplianceFrameworks(), ", ")))
	fs.Int("parallel", 0, "worker count (default: GOMAXPROCS)")
	fs.String("io-budget", formatSize(app.DefaultIOBudget), "byte-weighted I/O semaphore budget (k/m/g suffixes)")
	fs.String("max-file-size", formatSize(app.DefaultMaxFileSize), "full-content read cap for text detectors (k/m/g suffixes)")
	fs.Float64("min-confidence", 0, "presentation-layer confidence filter, 0-1")
	fs.StringArray("ignore", nil, "additional ignore glob; repeatable; applied on top of .gitignore/.airomignore")
	// The two-tier cache (internal/cache) is not implemented yet, so these
	// configure nothing. Say so rather than describing behavior the binary does
	// not have — `airom clean` does still use --cache-dir.
	// No backquotes in usage strings: pflag reads backquoted text as the flag's
	// placeholder name, so "`airom clean`" renders as "--cache-dir airom clean".
	fs.String("cache-dir", "", "scan cache location (default: <user cache dir>/airom); used by 'airom clean' — caching itself is not implemented yet")
	fs.Bool("no-cache", false, "disable cache reads and writes (no-op: caching is not implemented yet, every scan is cold)")
	fs.String("cdx-version", app.DefaultCDXVersion, "CycloneDX spec version: 1.6 or 1.7")
	fs.Bool("sarif-strict-kinds", false, `emit spec-pure kind:"informational" instead of level:"note"`)
	fs.Int("exit-code", exitCodeUnset, "exit status when --fail-on matches (default 1 when a policy is active; 0 reports matches without failing)")
	// exitCodeUnset is a sentinel, not a default. pflag renders any non-zero
	// default as "(default -1)", which is not a valid exit status and flatly
	// contradicts the sentence above it. DefValue is display-only — parsing and
	// Changed() read the Value — so clearing it to the type's zero suppresses the
	// line and leaves the real defaults where they are already stated: in the
	// help text.
	fs.Lookup("exit-code").DefValue = "0"
	fs.String("fail-on", "", `CI policy expression, e.g. "hosted-llm&confidence>=0.9" (see docs/cli.md)`)
	fs.Bool("offline", false, "assert no network access for the entire run")
	fs.String("pprof", "", "serve net/http/pprof (bare flag: localhost:6060; custom addr must be attached: --pprof=host:port)")
	fs.Lookup("pprof").NoOptDefVal = "localhost:6060"
	fs.String("trace", "", "write a Go execution trace to file")
	fs.Bool("no-progress", false, "disable the scan progress indicator (auto-off when stderr is not a terminal)")
	fs.Bool("stats", false, "emit the full ScanStats block in the output")
	fs.CountP("verbose", "v", "increase log verbosity (repeatable; -vv adds source locations)")
	fs.BoolP("quiet", "q", false, "errors only")
}
