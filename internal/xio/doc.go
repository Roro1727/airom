// Package xio holds the bounded-I/O primitives behind the bounded-everything
// invariant (ARCHITECTURE.md §8, P2): sync.Pool buffer pools per size class
// (findings copy out ≤200-byte snippets and never retain buffers), the spool
// that grows from memory to a temp file under hard caps (≤4 MiB memory,
// ≤64 MiB tmpfile — §7), and the byte-weighted I/O semaphore (default budget
// 256 MiB, a separate knob from CPU parallelism) acquired at
// min(size, budget) around any read over 1 MiB.
//
// The clamp is not a nicety: the unclamped variant is a latent deadlock on a
// 40 GB file, and it is contract-tested (decision D6). These primitives are
// what make peak RSS a function of configuration, never input size.
package xio
