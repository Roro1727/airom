// Package ruleengine turns declarative YAML rule packs into running
// detectors (ARCHITECTURE.md §6.3, decision D2). Compile runs once at
// startup: it validates every pack — IDs globally unique across packs,
// regexes compile, named groups referenced by claim templates exist, and
// keywords non-empty (the linter rejects keyword-less rules, so no
// un-prefiltered regex ever ships) — then builds ONE Aho–Corasick trie over
// all packs' keywords. Per file, the region lexer classifies code, comment,
// and string regions; the trie runs over code+string regions; and only
// regexes whose keywords hit ever execute — the literal-gated shape gitleaks
// and semgrep both proved, keeping hundreds of rules × 100k files cheap.
//
// The SHA-256 of the effective compiled ruleset (embedded defaults merged
// with --rules overlays) participates in every cache key (§10), so
// rules-as-data is self-invalidating. The rule-pack YAML schema stays
// internal in v0 and graduates to pkg/ when stable (§4).
package ruleengine
