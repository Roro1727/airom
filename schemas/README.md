# AIROM native output schemas

JSON Schema documents for AIROM's native output format (`-o json`, the `nativejson` writer),
per [ARCHITECTURE.md §11 and §14](../docs/ARCHITECTURE.md). The first schema,
`airom-v1.schema.json`, lands in **Phase 7** together with the schema-conformance CI job.

## The schema is a versioned API

The native JSON output is AIROM's lossless, round-trip reference format and is treated as a
**versioned API from release one** — every deferred consumer (server mode, SBOM
ingestion/merge, Dependency-Track push, VEX; §16) builds on it, so its stability is a
product commitment, not an implementation detail.

Policy:

- Every document declares its major version in-band: `"schemaVersion": "1"`.
- **`airom-v1.schema.json` is published with each release** and pinned to that release's
  tag; consumers validate against the schema shipped with the producing version.
- Within a major version, changes are **additive only**: new optional fields and new enum
  values may appear; fields are never removed, renamed, or retyped. Consumers must ignore
  unknown fields.
- A breaking change requires a new document (`airom-v2.schema.json`) and a `schemaVersion`
  bump — and is expected to be rare and loudly announced.
- Pre-release (`v0.1.0-dev`, current state): the schema is still forming and carries no
  compatibility promise until the first tagged release publishes it.

## Enforcement (CI, §14)

- **Schema conformance:** every native golden output in `testdata/fixtures/` validates
  against `airom-v1.schema.json` on every CI run.
- **Round-trip:** a fuzz-populated `Inventory` serialized to native JSON and re-read must be
  identical — the schema describes a lossless format, and the test proves it.
- CycloneDX and SARIF outputs are validated against their **official upstream schemas**
  (`bom-1.6.schema.json`, the OASIS SARIF 2.1.0 schema); those schemas are not maintained
  here.
