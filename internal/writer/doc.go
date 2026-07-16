// Package writer defines the output stage (ARCHITECTURE.md §11): a Writer is
// a pure function from *airom.Inventory to bytes (invariant P5) that never
// invents, drops, or re-derives data — every format is a projection of the
// same assembled graph. The Inventory is small (components, not files), so
// streaming discipline applies to scanning, not rendering. Multi-output is
// first-class: the repeatable -o fmt[=path] flag emits, say, a table to the
// TTY plus CycloneDX and SARIF to files from one scan.
//
// Concrete writers live in the nativejson, cdx, sarifw, yamlw, and tablew
// subpackages. The SPDX 3.0.1 AI-profile writer is a reserved v2 slot (§16)
// that must land as one new package with zero core changes — that asymmetry
// is the acceptance test for this architecture.
package writer
