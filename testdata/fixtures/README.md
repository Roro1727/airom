# Test fixtures

Miniature polyglot fixture repositories and their per-writer golden outputs — the
end-to-end layer of the testing strategy in
[ARCHITECTURE.md §14](../../docs/ARCHITECTURE.md).

## Planned fixture repos

Each fixture is a small, self-contained "repository" exercising one detection story
end-to-end. Fixtures are checked in as plain trees; they are populated alongside the
detectors and rule packs that consume them (Phases 4–6).

| Fixture | Exercises |
|---------|-----------|
| `python-langchain-rag/` | LangChain + OpenAI SDK call sites, embedding model, vector DB, prompt files, `capture_params` → the full RAG-pipeline stitch (`raglink`) |
| `go-openai-service/` | `go/parser` AST detection (`gosrc`), go.mod manifest evidence, hosted-model literals in Go |
| `node-openai-app/` | JS/TS region lexers, package.json manifest join, SDK usage rules |
| `local-llama-gguf/` | **Handcrafted valid GGUF headers** (small files, real magic + metadata: architecture, param count, quantization) → `modelfile` header parsing and content-hash identity |
| `mixed-monorepo/` | Cross-language dedup: the same model referenced from Python and TS collapses to one component; version-unknown folding (§9.1) |
| `k8s-manifests/` | `k8ssource --manifests` offline mode: workload enumeration and image extraction without a cluster |

An **OCI image layout is built in CI** from fixture content (never checked in as a blob) to
exercise `imagesource` streaming, spooling, and tee-hashing against the same expectations.

Additionally, every YAML rule ships **at least one positive and one negative fixture** next
to its pack (see `rules/README.md`); those are rule-level, not repo-level, fixtures.

## Golden-file policy

- Every fixture repo is scanned in CI and **all five writer outputs**
  (native JSON, CycloneDX, SARIF, YAML, table) are golden-filed alongside it.
- Determinism is engineered, not hoped for: goldens are produced with an **injected clock
  and serial number**, and the assembler's total ordering (P7) guarantees byte-stable
  output — CI also diffs `--parallel 1` vs `--parallel 16` runs.
- Golden diffs are the review surface: a detector or rule change must show its full
  observable effect as a golden diff in the PR ("do the goldens look right" is the review
  question, §6.5).
- Native goldens are validated against `schemas/airom-v1.schema.json`; CycloneDX and SARIF
  goldens against their official upstream schemas.

## Regenerating goldens (UPDATE_GOLDEN workflow)

Goldens are regenerated, never hand-edited:

```
UPDATE_GOLDEN=1 go test ./...
```

The harness (`pkg/airom/detectortest` and the E2E golden tests) rewrites golden files when
`UPDATE_GOLDEN=1` is set (the rule-pack tests also accept the conventional `-update` flag,
§6.5). Then:

1. `git diff` the goldens and read every hunk — the diff **is** the behavior change.
2. Commit regenerated goldens in the same PR as the change that caused them.
3. CI runs without `UPDATE_GOLDEN`, so stale goldens fail the build rather than silently
   drifting.
