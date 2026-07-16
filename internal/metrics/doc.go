// Package metrics makes profiling a product feature (ARCHITECTURE.md §14):
// ScanStats accumulates files walked and skipped, bytes read versus bytes in
// tree, cache hit rates, per-detector nanoseconds and invocation counts, and
// the selection explanation of which --select expression enabled which
// detector (§6.2) — embedded into the Inventory under --stats, so "what did
// the scanner skip" is always answerable and detector #217 is triaged with
// data, not guesses. No silent caps: every triage decision leaves a trace.
//
// The package also bootstraps the --pprof server and --trace per-phase
// regions wired by the composition root.
package metrics
