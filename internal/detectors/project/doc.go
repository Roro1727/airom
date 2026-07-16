// Package project holds the built-in phase-2 ProjectDetectors
// (ARCHITECTURE.md §3, §17) — cross-file logic the streaming phase cannot
// express: hfdir assembles a HuggingFace model directory (config.json +
// weights) into ONE component; adapterlink turns adapter_config.json into
// DERIVED_FROM base-model lineage; configbind attaches separated generation
// configs to the model they name via CONFIGURES edges under the
// refusal-first ambiguity policy (§9.5) — never a guessed edge; raglink
// stitches retriever, store, and embedder findings into a rag-pipeline
// composite with CONTAINS/QUERIES/EMBEDS_WITH edges; lockjoin joins
// manifests with their lockfiles.
//
// The set is deliberately flat: one barrier, every detector sees the same
// immutable phase-1 findings view, no inter-detector ordering — anything
// needing multi-stage reasoning belongs in the assembler.
package project
