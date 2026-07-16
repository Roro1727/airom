// Package lexer provides per-language region classifiers (~250 LOC each)
// that split source text into code, comment, and string regions for the rule
// engine (ARCHITECTURE.md §6.4, decision D1). These are not parsers and
// build no ASTs: region classification is exactly enough context for
// keyword-gated regex rules to never match inside comments, at a cost
// compatible with the pure-Go, CGO_ENABLED=0 distribution story (P8).
//
// Planned languages: Python, JavaScript, TypeScript, Java, Rust, C#, Kotlin.
// Go needs no lexer here — Go source gets an exact AST via the stdlib
// go/parser in internal/detectors/gosrc. Lexer precision is tracked against
// real ASTs by the //go:build oracle CI job (§14), which gates any future
// wazero-WASM tree-sitter layer on measured — not assumed — failures.
package lexer
