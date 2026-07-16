// Package sarifw projects the inventory to SARIF 2.1.0 for GitHub Code
// Scanning (ARCHITECTURE.md §11, docs/mapping.md §3/§7). The projection is a
// pure function of Evidence: one SARIF rule per DetectorID (a stable
// vocabulary), one result per Occurrence, default level "note" for GitHub
// compatibility, with --sarif-strict-kinds opting into the spec-pure
// kind "informational".
//
// partialFingerprints["airomComponentIdentity/v1"] carries
// hex(sha256(detectorID|componentID|path)) — deliberately line-free, so
// fingerprints survive code motion between scans (§7.2). Unknowns surface as
// invocation toolExecutionNotifications, never results (§3.11).
//
// The report is emitted from a small hand-rolled SARIF struct set (model.go)
// rather than a third-party library, buying byte-exact control over field
// order, the level/kind toggle, and property-bag key-sorting — the
// determinism the P7 byte-identical contract requires. Goldens are validated
// against the OASIS SARIF schema in CI (§14).
package sarifw
