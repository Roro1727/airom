// Package assemble is the single-threaded, deterministic stage that turns
// raw detector findings into the final *airom.Inventory (ARCHITECTURE.md
// §9). It holds the assembler monopolies of invariant P4 — detectors emit
// claims; only this package mints identity: CanonicalKey with Class ≠ Kind
// and a content-hash discriminator for weights files (§9.1, decision D7);
// keep-and-relate merging where losing identity claims are retained as
// competing CycloneDX evidence.identity[] entries, never discarded (§9.2);
// the grouped noisy-OR confidence calculus with capped repetition and the
// 0.99 clamp — only a known-weights hash match or a verified attestation may
// assert 1.0 (§9.3, decision D8); purl derivation as an output of identity
// (§9.4); generation-parameter binding under the refusal-first policy
// (§9.5); and relation resolution, where dangling TargetHints become Stats
// warnings — never phantom nodes, never guessed edges.
//
// Everything is sorted on the way out, making assembly the guarantor of
// deterministic output (P7).
package assemble
