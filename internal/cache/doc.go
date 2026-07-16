// Package cache implements the bbolt-backed scan cache (ARCHITECTURE.md §10,
// decision D10). Every entry lives under a namespace keyed by
// sha256(detectorVersions ‖ effectiveRulesetSHA256 ‖ sizeCaps ‖ ignoreConfig),
// so the entire cache self-invalidates on any behavior change — structurally
// eliminating the forgotten-version-bump stale-cache bug for the fast-moving
// rule surface. Within a namespace: a stat-key tier (path, size, mtimeNs,
// dev, inode — a hit means the file is never opened), a content-key tier on
// the xxh3 hash that the read tee produces for free (a hit skips detector
// CPU), and a per-layer blob cache with a MissingBlobs-shaped API so
// unchanged image base layers are never re-streamed — and a remote/shared
// backend stays possible (v2, §16).
//
// Rules with teeth: pre-assembly findings only, never assembled inventories;
// lock acquisition is try-acquire, degrading to no-cache with a warning — a
// second concurrent run must never hang CI; async batched writes are flushed
// and joined before the phase barrier (§8).
package cache
