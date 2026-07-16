// Package engine drives the two-phase scan pipeline (ARCHITECTURE.md §3, §8):
// phase 1 streams files from exactly one walker/producer through a bounded
// task channel into a worker pool where all matched detectors run
// SEQUENTIALLY on one shared buffer, with exactly one collector goroutine
// owning all mutable aggregation state — no locks. A hard barrier (walker
// drained, workers joined, cache writes flushed) then admits the flat
// phase-2 ProjectDetector set, which pulls files via the source Resolver
// against an immutable view of phase-1 findings.
//
// The engine also hosts the detector catalog and enforces the pipeline
// invariants: bounded memory by construction (P2), per-file panics and
// errors degrading to Unknowns rather than killing the scan (P6), and
// byte-identical output at any --parallel value (P7).
package engine
