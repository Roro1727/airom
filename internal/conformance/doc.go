// Package conformance is AIROM's output-format conformance suite: a permanent,
// CI-enforced check that every writer's bytes satisfy the external contract the
// format claims to speak (docs/mapping.md). It renders the shared
// writertest.BuildFixture inventory through each writer and asserts:
//
//   - CycloneDX 1.6/1.7 is well-formed CDX the official library re-decodes, and
//     the fields the official schema constrains by regex (serialNumber, hash
//     content) match those official patterns — with a documented allow-list of
//     the violations the current fixture provokes (§ findings in the package
//     tests);
//   - the CDX→internal mapping round-trips: every airom:kind survives the
//     coarser CDX type enum, edges route to dependencies[] / modelCard datasets
//     / airom:rel.* per docs/mapping.md §3.10, and evidence/technique/confidence
//     encode as specified (§3.8–§3.10, §4, §5);
//   - the native JSON validates against schemas/airom-v1.schema.json (checked by
//     a self-contained validator covering the exact JSON-Schema subset that
//     schema uses — gojsonschema cannot parse its draft 2020-12 declaration);
//   - the SARIF 2.1.0 envelope, rules/results wiring, level/kind toggle, and
//     partialFingerprints recipe hold structurally (docs/mapping.md §7);
//   - confidences serialize in the single §6.2 form across every format.
//
// The package body is intentionally empty: the suite lives entirely in the
// _test.go files so it ships no production code.
package conformance
