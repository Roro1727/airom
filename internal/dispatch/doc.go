// Package dispatch compiles every registered detector's Selector into one
// index — O(1) basename and extension maps, path-glob and magic-byte tables
// matched against the shared header sample — evaluated once per file
// (ARCHITECTURE.md §6.1). Per-file matching cost is O(matches), never
// O(detectors), which is what keeps hundreds of detectors cheap across
// hundreds of thousands of files.
//
// Together with size triage and the header sample, the compiled index
// implements the decide-before-you-read invariant (P3): path, size, and a
// 32 KB header eliminate more than 95% of files before any full content
// read. The index also reports which detectors were interested and why,
// feeding the selection explanation recorded in Inventory.Stats (§6.2).
package dispatch
