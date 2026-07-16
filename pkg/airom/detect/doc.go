// Package detect defines the public detector framework (ARCHITECTURE.md §6.1):
// the Detector, FileDetector (phase 1, one file at a time, streaming), and
// ProjectDetector (phase 2, cross-file, pull-style over a Resolver)
// interfaces; the declarative Selector the dispatcher compiles into a single
// per-file index; the File read-once access contract (shared 32 KB Header,
// one lazy tee-hashed Content read, ReaderAt that is explicitly unavailable
// on stream sources); and the Finding, ComponentClaim, and RelationClaim
// types that detectors emit.
//
// Detectors emit claims, never components (invariant P4): Finding carries no
// ID field, so identity, dedup, merging, and confidence remain assembler
// monopolies that a contributor physically cannot break. Like pkg/airom,
// this package is stdlib-only and part of the semver-guarded plugin SDK (§4).
package detect
