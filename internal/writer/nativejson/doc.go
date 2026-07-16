// Package nativejson emits AIROM's native JSON format (ARCHITECTURE.md §11):
// the lossless, round-trip reference serialization of the Inventory graph,
// versioned from release one (schemaVersion "1") with its JSON Schema
// published per release under schemas/ and enforced by conformance and
// fuzz round-trip tests in CI (§14).
//
// This format is the stable substrate for every deferred v2 consumer —
// server mode, SBOM ingestion/merge, Dependency-Track push, VEX (§16) — so
// it evolves only additively within a schema version.
package nativejson
