// Package source defines the Source abstraction over scan targets
// (ARCHITECTURE.md §7): a Source couples a Walker (push-style, ignore-aware
// enumeration feeding phase 1), a Resolver (pull-style access for phase-2
// project detectors), content identity (image digest, git HEAD, dir
// realpath), layer IDs for blob-cache granularity, and SourceInfo provenance.
//
// The interface is shaped by the WORST source — a squashed OCI tar stream:
// sequential, non-seekable, consume-during-walk; the directory case is the
// easy specialization. Concrete implementations live in the dirsource,
// gitsource, imagesource, and k8ssource subpackages. One resolver
// abstraction keeps every detector — including third-party — automatically
// source-agnostic.
package source
