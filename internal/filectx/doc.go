// Package filectx implements the read-once file access contract
// (ARCHITECTURE.md §8, invariant P1): each file's bytes are read from the
// source at most once, and every interested detector shares that one buffer
// — detectors for a file run sequentially in one worker, so no buffer
// synchronization exists or is needed.
//
// A file context exposes the shared 32 KB header sample taken at walk time,
// a lazy single bounded Content read that is tee-hashed as it happens (xxh3
// for cache keys, SHA-256 for weights identity — both free), and a ReaderAt
// that works on directory sources but returns ErrNotSeekable on stream
// sources; the asymmetry is explicit, not papered over. This package is the
// internal realization of the pkg/airom/detect.File contract detectors
// program against.
package filectx
