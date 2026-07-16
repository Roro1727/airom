# AIROM rule packs

This directory holds the **embedded declarative rule packs** — the contributor hot zone of
the project ([ARCHITECTURE.md §6.3](../docs/ARCHITECTURE.md)). The directory layout and the
rules below are binding now; the pack contents land in **Phase 6**.

## The bright line

> If the detection is expressible as **keywords + regex over classified text regions + a
> templated claim**, it is YAML and belongs here. The moment you need a loop, a parser, or
> two files, it is Go and belongs in `internal/detectors/`.

Declarative rules cover roughly 80 % of the detection surface and 100 % of the fast-moving
surface: hosted-model IDs, SDK call-site patterns, framework and vector-DB usage,
embedding-model names, prompt-template usage, generation parameters, and infra client usage.
The point of the split: **a new model ID is a rules PR, never a release** (decision D2).

Code stays code: binary header parsers, pickle opcode walking, manifest parsers, HF
directory assembly, Go AST analysis, and all phase-2 cross-file detectors.

## Layout: one file per provider

Packs are organized **one file per provider, never per-category monoliths** (decision D3).
With hundreds of rules, monolithic packs become merge-conflict hotspots and defeat
CODEOWNERS review routing.

Planned category directories (populated in Phase 6):

| Directory     | Contents |
|---------------|----------|
| `models/`     | Hosted-LLM providers: `openai.yaml`, `anthropic.yaml`, `gemini.yaml`, `bedrock.yaml`, `azure-openai.yaml`, `cohere.yaml`, `mistral.yaml`, `groq.yaml`, `ollama.yaml`, `huggingface.yaml` |
| `embeddings/` | Embedding models: `openai.yaml`, `sentence-transformers.yaml`, `bge-e5-minilm.yaml`, `voyage.yaml` |
| `frameworks/` | Framework usage: `langchain.yaml`, `llamaindex.yaml`, `haystack.yaml`, `dspy.yaml`, `crewai.yaml`, `autogen.yaml`, `semantic-kernel.yaml`, `transformers.yaml`, `vllm.yaml`, `mlflow.yaml` |
| `vectordb/`   | Vector stores: `chroma.yaml`, `milvus.yaml`, `qdrant.yaml`, `pinecone.yaml`, `weaviate.yaml`, `faiss.yaml`, `pgvector.yaml`, `redis.yaml`, `elastic.yaml`, `mongodb-atlas.yaml` |
| `infra/`      | Serving-infra client usage: `ollama.yaml`, `vllm.yaml`, `tgi.yaml`, `rayserve.yaml`, `sagemaker.yaml`, `vertex.yaml`, `azureml.yaml` |
| `params/`     | Generation-parameter capture rules (temperature, top_p, max_tokens, …) |
| `prompts/`    | `PromptTemplate` / `ChatPromptTemplate` / `system_prompt` patterns |
| `datasets/`   | `load_dataset()` calls, Kaggle refs, HF dataset ids |

## The three rule layers

1. **Embedded defaults** — this directory, compiled into the binary via `go:embed`.
   Offline by construction, versioned with the release.
2. **User overlay** — `--rules extra.yaml` (repeatable), merged by rule ID:
   add new rules, override existing ones, or disable them.
3. **Remote registry** — v2 (ARCHITECTURE.md §16): OCI-distributed packs, paired with the
   signing and trust-policy work. Not part of v1.

The SHA-256 of the *effective compiled ruleset* (defaults + overlays) participates in every
cache key (§10), so rule changes self-invalidate the cache — no manual version bumps.

## Hard requirements for every rule

Enforced by `airom rules lint` in CI; a pack that violates any of these does not merge:

- **Non-empty `keywords:`.** Every rule feeds the single Aho–Corasick prefilter trie; a
  keyword-less rule would run its regex un-gated on every file and is rejected outright
  (invariant P3).
- **Globally unique `id:`** across all packs (`<provider>/<rule-name>`, e.g.
  `openai/model-literal` — this becomes the SARIF ruleId).
- **Regexes compile**, and every named capture group referenced by the `claim:` template
  exists in the pattern.
- **`regions:`** restricts matching to classified `code`/`string` regions — rules never
  match inside comments.
- **Fixtures: at least one positive and one negative per rule**, checked in alongside the
  pack. The negative fixture is what keeps false-positive regressions visible in review.

## Contributing a pack (the one-hour story, §6.5)

```
airom dev new-rulepack <provider>     # scaffolds rules/models/<provider>.yaml + fixtures
# write ~30 lines of YAML: keywords, patterns, claim template
airom rules lint
go test ./rules/... -run <Provider> -update   # writes the golden
```

A pack PR is one YAML file, two fixtures, and one golden — zero Go. See
`docs/plugin-guide.md` for the walkthrough and `docs/rule-schema.md` for the full YAML
schema (which stays internal in v0 and graduates to `pkg/` when stable, §4).
