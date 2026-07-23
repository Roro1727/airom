package app

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/airomhq/airom/internal/ruleengine"
	"github.com/airomhq/airom/internal/ruleengine/ruletest"
	"github.com/airomhq/airom/internal/rulesync"
)

// EmbeddedRules is the built-in rule-pack filesystem (rules/**). It is set
// once, from the rules embed package, by the composition root — kept as a
// var so the SDK stays free of a go:embed dependency and tests can inject an
// empty set. nil means "no embedded packs".
var EmbeddedRules fs.FS

// resolveRuleBase picks the base rule layer for a scan: a verified, cached
// bundle when one is installed (and --no-cached-rules is off), else the
// embedded packs — the offline floor. It returns the filesystem plus a
// provenance label ("builtin" or the bundle version). --rules overlays layer
// on top of whichever wins, downstream in ruleengine.Load.
func resolveRuleBase(cfg *Config) (fs.FS, string) {
	if cfg.NoCachedRules {
		return EmbeddedRules, "builtin"
	}
	dir := cfg.CacheDir
	if dir == "" {
		dir = DefaultCacheDir()
	}
	if bundle, version, ok := rulesync.Active(dir); ok {
		return bundle, version
	}
	return EmbeddedRules, "builtin"
}

// loadRuleset assembles the effective ruleset (base layer + --rules overlays)
// and reports its provenance label. A cached bundle that fails to load must
// never brick a scan: it falls back to the embedded packs (with the same
// overlays) and a warning, so a bad fetch degrades instead of turning every
// scan fatal.
func loadRuleset(cfg *Config) (*ruleengine.Ruleset, string, error) {
	base, version := resolveRuleBase(cfg)
	rs, err := ruleengine.Load(base, cfg.RulePaths, os.ReadFile)
	if err != nil && version != "builtin" {
		slog.Warn("cached rule bundle failed to load; using the built-in packs", "version", version, "error", err)
		version = "builtin"
		rs, err = ruleengine.Load(EmbeddedRules, cfg.RulePaths, os.ReadFile)
	}
	if err != nil {
		return nil, "", &UsageError{Err: err}
	}
	return rs, version, nil
}

// RulesList returns the effective compiled ruleset (each rule with its
// originating layer) for `airom rules list`.
func RulesList(cfg *Config) ([]ruleengine.EffectiveRule, error) {
	rs, _, err := loadRuleset(cfg)
	if err != nil {
		return nil, err
	}
	return rs.Rules, nil
}

// RulesUpdate fetches, verifies, and caches a signed rule bundle from the
// airom-rules release channel for `airom rules update` (Model B). version is
// the positional argument ("" or "latest" → the newest release). It touches
// the network; a scan never does.
func RulesUpdate(ctx context.Context, cfg *Config, version string) (*rulesync.Result, error) {
	dir := cfg.CacheDir
	if dir == "" {
		dir = DefaultCacheDir()
	}
	return rulesync.Update(ctx, rulesync.Options{
		CacheDir:              dir,
		Version:               version,
		Source:                cfg.RulesSource,
		Offline:               cfg.Offline,
		InsecureSkipSignature: cfg.InsecureSkipSignature,
	})
}

// RulesLint validates a single user rule-pack file against the full lint
// contract and reports its fixture coverage (docs/rule-schema.md). fixtures
// are expected under <pack-dir>/testdata/<pack>/ when present.
func RulesLint(path string) (*ruletest.Report, error) {
	rs, err := ruleengine.Load(nil, []string{path}, os.ReadFile)
	if err != nil {
		return nil, &UsageError{Err: err}
	}
	m, err := ruleengine.Compile(rs)
	if err != nil {
		return nil, err
	}
	dir := fixturesDirFor(path)
	if _, statErr := os.Stat(dir); statErr != nil {
		// Compilation already validated the lint contract items 1-9; only
		// fixture coverage (item 10) needs a testdata dir.
		return &ruletest.Report{}, fmt.Errorf("no fixtures at %s: every rule needs ≥1 positive and ≥1 negative fixture (rule-schema.md item 10)", dir)
	}
	return ruletest.Run(m, dir)
}

// RulesTest runs a user rule pack against its fixtures for `airom rules test`.
func RulesTest(path string) (*ruletest.Report, error) {
	return ruletest.RunPackFile(path, fixturesDirFor(path))
}

// fixturesDirFor maps a pack file to its fixture directory:
// rules/models/openai.yaml -> rules/models/testdata/openai.
func fixturesDirFor(packPath string) string {
	dir := filepath.Dir(packPath)
	stem := stemOf(filepath.Base(packPath))
	return filepath.Join(dir, "testdata", stem)
}

func stemOf(name string) string {
	if ext := filepath.Ext(name); ext != "" {
		return name[:len(name)-len(ext)]
	}
	return name
}
