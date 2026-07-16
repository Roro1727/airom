// Command airom is the AIROM CLI entrypoint. Per ARCHITECTURE.md §4, this
// package contains main.go only: build-metadata stamping and a handoff to
// internal/cli. Nothing else may live here.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Roro1727/airom/internal/cli"
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

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := cli.Execute(ctx, cli.BuildInfo{Version: version, Commit: commit, Date: date})
	stop()
	os.Exit(code)
}
