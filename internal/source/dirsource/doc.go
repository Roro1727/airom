// Package dirsource implements the local-directory Source (ARCHITECTURE.md
// §7): fastwalk enumeration with a per-directory nested .gitignore stack
// (gocodewalker semantics), .airomignore, and default skips (.git,
// node_modules, vendor, virtualenvs); an 8 KB NUL-sniff for binary
// detection; symlink cycles guarded by (dev, inode); and permission errors
// degraded to Unknowns, never fatal (invariant P6).
//
// Directory trees are the seekable easy case of the Source contract, so
// ReaderAt is fully supported here. dirsource is also the delegation target
// for gitsource, which scans its shallow clone as a plain filesystem.
package dirsource
