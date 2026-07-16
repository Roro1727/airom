// Package cli implements the AIROM command tree (ARCHITECTURE.md §12): scan,
// fs, repo, image, k8s, detectors {list|explain}, rules {list|lint|test},
// dev {new-rulepack|new-detector}, clean, and version — built on cobra with
// koanf configuration binding (flags > env > file > defaults; viper's weight
// and global state rejected in decision D15) and stdlib slog.
//
// The package owns the exit-code contract (decision D17): exit 0 means the
// scan succeeded — findings are NOT failures; --exit-code/--fail-on is
// opt-in CI policy. Commands parse and validate input, then hand a resolved
// configuration to internal/app; no construction or wiring happens here.
//
// The cobra tree and the Execute entrypoint consumed by cmd/airom land in
// Phase 3.
package cli
