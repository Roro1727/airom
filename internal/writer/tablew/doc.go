// Package tablew renders the human-facing terminal summary (ARCHITECTURE.md
// §11): KIND | NAME | VERSION | PROVIDER | CONF | EVIDENCE (n) | FIRST SEEN,
// TTY-aware (width, color, non-TTY fallback), with -v expanding per-component
// file:line evidence lists. It is the default sink for interactive runs and,
// like every writer, a pure projection of the assembled Inventory (invariant
// P5) — it renders nothing the graph does not already carry.
package tablew
