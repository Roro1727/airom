// Package gosrc detects AI usage in Go source with the stdlib go/parser
// (ARCHITECTURE.md §6.4, decision D1): exact AST analysis of import paths,
// SDK call sites, and model-name literals — Go is the one language where a
// real parser is free, so it gets one instead of the region-lexer + regex
// path used elsewhere.
//
// Findings carry the enclosing function or method as the Occurrence symbol
// and use MethodAST, which the confidence calculus treats as a distinct
// corroborating channel from source-code-analysis evidence (§9.3).
package gosrc
