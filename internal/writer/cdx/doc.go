// Package cdx projects the inventory to CycloneDX ML-BOM via
// CycloneDX/cyclonedx-go (ARCHITECTURE.md §11, decision D16) — 1.6 by
// default, 1.7 via --cdx-version (the modelCard shape is identical in both).
// Model kinds map to machine-learning-model with a modelCard (params,
// hyperparams, considerations, energy); datasets and prompts to data;
// frameworks and libraries to native types. IdentityClaims become
// evidence.identity[] (confidence + technique) and Occurrences become
// evidence.occurrences[] with file/line/snippet — the differentiator no
// other shipping AIBOM tool populates. depends-on maps to dependencies[],
// trained-on to modelCard.modelParameters.datasets[].ref, and remaining edge
// types to documented airom:rel.* properties until CycloneDX grows typed
// relationships; overflow lands in airom:* properties.
//
// Goldens are validated against the official bom-1.6 schema in CI, and
// docs/mapping.md is enforced by a round-trip test (§14) — spec compliance
// is a test, not a hope.
package cdx
