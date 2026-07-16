// Command airom is the AIROM CLI entrypoint. Per ARCHITECTURE.md §4, this
// package contains main.go only: build-metadata stamping and a handoff to
// internal/cli. Nothing else may live here.
package main

import (
	"fmt"
	"os"
)

// Build metadata, stamped at release time by goreleaser:
//
//	-ldflags "-X main.version=v0.1.0 -X main.commit=<sha> -X main.date=<rfc3339>"
//
// The defaults identify a locally built, unstamped development binary.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// main is Phase-1 scaffolding: it keeps `go build ./...` green and proves the
// ldflags stamping path end-to-end before the command tree exists.
//
// TODO(phase-3): replace body with internal/cli.Execute(ctx).
func main() {
	fmt.Printf("airom %s (%s, built %s)\n", version, commit, date)
	fmt.Println("The airom command tree (scan, fs, repo, image, k8s, ...) is wired in Phase 3; this binary is a build scaffold until then.")
	os.Exit(0)
}
