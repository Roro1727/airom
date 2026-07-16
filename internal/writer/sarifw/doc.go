// Package sarifw projects the inventory to SARIF 2.1.0 via
// owenrumney/go-sarif/v3 for GitHub Code Scanning (ARCHITECTURE.md §11). The
// projection is a pure function of Evidence: one SARIF rule per DetectorID
// (a stable vocabulary), one result per Occurrence, default level "note" for
// GitHub compatibility with --sarif-strict-kinds opting into spec-pure
// kind "informational".
//
// partialFingerprints["airomComponentIdentity/v1"] carries
// sha256(detectorID|componentID|path) — deliberately line-free, so
// fingerprints survive code motion between scans. Goldens are validated
// against the OASIS SARIF schema in CI (§14).
package sarifw
