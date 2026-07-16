// Package airom defines the canonical AIBOM domain model (ARCHITECTURE.md §5):
// Inventory, Component, Evidence, Occurrence, IdentityClaim, Relationship, the
// ComponentKind and DetectionMethod vocabularies, Confidence, and the
// tri-state optionals (Presence, OptString, OptInt64, OptTime, TriState) that
// carry the SPDX NOASSERTION discipline — distinguishing "unknown" from
// "not applicable" deterministically.
//
// The graph defined here is the product (invariant P5): it is designed as a
// superset of what CycloneDX 1.6 ML-BOM, SPDX 3.0.1 AI profile, SARIF 2.1.0,
// and the native format each need, so every writer is a pure projection that
// never invents, drops, or re-derives data.
//
// This package is the root of the public plugin SDK (§4): it imports nothing
// outside the standard library and is semver-guarded by apidiff in CI,
// shipping as v0.x until the interfaces survive real third-party use.
package airom
