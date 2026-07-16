// Package yamlw renders the native inventory model as YAML through yaml.v3
// with stable key order (ARCHITECTURE.md §11) — the same lossless content as
// the native JSON writer, in a form suited to human review. Combined with
// the determinism invariant (P7: the assembler sorts everything, serial and
// timestamp are injectable), consecutive scans of an unchanged tree are
// byte-identical, so the output diffs cleanly in code review and goldens.
package yamlw
