// Command airom is the AIROM CLI entrypoint. Per ARCHITECTURE.md §4, this
// package contains main.go only: build-metadata stamping and a handoff to
// internal/cli. Nothing else may live here.
package main

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/airomhq/airom/internal/cli"
)

// Build metadata, stamped at release time by goreleaser and by `make build`:
//
//	-ldflags "-X main.version=v0.1.0 -X main.commit=<sha> -X main.date=<rfc3339>"
//
// These defaults mean "nobody stamped this binary" — see resolveBuildInfo,
// which recovers the real values from the Go build info instead of reporting
// them.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Sentinels for "not stamped".
const (
	unsetVersion = "dev"
	unsetCommit  = "none"
	unsetDate    = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		// First signal: cancel ctx (graceful). Releasing the registration
		// here restores default delivery, so a second Ctrl-C kills the
		// process even if shutdown hangs.
		<-ctx.Done()
		stop()
	}()
	code := cli.Execute(ctx, resolveBuildInfo())
	stop() // idempotent; covers the signal-free path
	os.Exit(code)
}

// resolveBuildInfo reports the binary's identity, preferring ldflags and
// falling back to the build info the Go toolchain embeds automatically.
//
// This matters beyond cosmetics: ToolInfo is embedded in every AIBOM airom
// emits, so an unstamped binary makes every document claim it was produced by
// "dev". `go install github.com/airomhq/airom/cmd/airom@latest` cannot pass
// ldflags — but the toolchain records the module version anyway, and a plain
// `go build` inside a checkout records the VCS revision. Reading those turns
// "dev (commit none, built unknown)" into the truth.
//
// Sources, in order of preference:
//
//	ldflags        goreleaser and `make build`  -> "v0.1.0", "1a2b3c4"
//	Main.Version   go install module@version    -> "v0.1.0" or a pseudo-version
//	vcs.*          go build inside a checkout   -> revision, time, dirty flag
func resolveBuildInfo() cli.BuildInfo {
	bi, ok := debug.ReadBuildInfo()
	return resolve(version, commit, date, bi, ok)
}

// resolve is the pure core of resolveBuildInfo, taking the build info as an
// argument so the fallback logic is testable.
func resolve(v, c, d string, bi *debug.BuildInfo, ok bool) cli.BuildInfo {
	if !ok || bi == nil { // built without module support; nothing to recover
		return cli.BuildInfo{Version: v, Commit: c, Date: d}
	}

	// `go install module@version` stamps the resolved module version. "(devel)"
	// means a local build, which carries no version — the VCS stamps below are
	// the better answer there.
	if v == unsetVersion && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		v = bi.Main.Version
	}

	var rev, when string
	var dirty bool
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.time":
			when = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}

	if c == unsetCommit && rev != "" {
		c = shortRev(rev)
		if dirty {
			c += "-dirty"
		}
	}
	if d == unsetDate && when != "" {
		d = when
	}
	// A local build has no version but does have a revision: say so rather than
	// claiming the bare word "dev".
	if v == unsetVersion && rev != "" {
		v = "devel-" + shortRev(rev)
		if dirty {
			v += "-dirty"
		}
	}
	return cli.BuildInfo{Version: v, Commit: c, Date: d}
}

// shortRev abbreviates a git revision the way git itself does.
func shortRev(rev string) string {
	const short = 7
	rev = strings.TrimSpace(rev)
	if len(rev) > short {
		return rev[:short]
	}
	return rev
}
