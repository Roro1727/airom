// Package imagesource implements the container-image Source (ARCHITECTURE.md
// §7, decision D11): go-containerregistry resolves a v1.Image through the
// remote → daemon → tarball → OCI-layout fallback chain, and the squashed
// tar from mutate.Extract is streamed exactly once. Header reads and spool
// decisions execute in the walker goroutine because tar entry content is
// only valid during traversal; workers consume spooled buffers (≤4 MiB in
// memory, ≤64 MiB tmpfile, else header-only) — never the live stream.
//
// The union of phase-2 ProjectDetector selectors is folded into the spool
// policy so cross-file detectors see their files in image scans, and large
// model files are tee-hashed during the mandatory discard copy, making
// content-hash identity for in-image weights free (§9.1). Net effect: a
// 40 GB GGUF inside an image costs a 32 KB header parse plus a hashing
// discard-copy — zero memory growth, zero disk.
package imagesource
