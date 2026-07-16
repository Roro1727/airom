// Package dataset detects dataset files by format signature (ARCHITECTURE.md
// §4, §17): CSV and JSONL by structural sniffing of the shared header
// sample, Parquet and Arrow by magic bytes — emitting KindDataset claims
// that phase-2 stitching can attach to models via TRAINED_ON edges (the
// SPDX trainedOn mapping).
//
// In-code dataset references — load_dataset() calls, Kaggle refs, HF dataset
// ids — are declarative surface owned by rules/datasets/*.yaml under the
// §6.3 bright line, not this package.
package dataset
