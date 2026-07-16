package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/Roro1727/airom/internal/metrics"
)

// UsageError marks configuration and flag errors: the CLI maps it (like any
// fatal error) to exit code 2 per the docs/cli.md contract, but prefixes the
// message differently from runtime failures.
type UsageError struct{ Err error }

func (e *UsageError) Error() string { return e.Err.Error() }
func (e *UsageError) Unwrap() error { return e.Err }

// ErrEngineNotWired reports that the scan pipeline behind the CLI surface is
// not yet assembled. The CLI (Phase 3), filesystem scanner (Phase 4), plugin
// framework (Phase 5), and writers (Phase 7) land in that order per
// docs/ROADMAP.md; until Phase 4 connects them, every scan command fails
// fast with this error instead of pretending to scan.
var ErrEngineNotWired = errors.New("scan engine not wired yet")

// Run executes one scan described by cfg: it is the single entry point the
// CLI (and pkg/airom.Scan, later) calls. It owns defaulting, validation, and
// run-environment bootstrap (pprof/trace), then hands off to the engine.
//
// TODO(phase-4): replace the ErrEngineNotWired return with the real
// pipeline: source.Detect -> engine.Scan -> assemble.Build -> writer fan-out.
// Suggested tracking issue: "Phase 4: implement the filesystem scanner and
// wire it into app.Run".
func Run(ctx context.Context, cfg *Config) error {
	if err := ctx.Err(); err != nil {
		return err // honor a context canceled before we started
	}
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return &UsageError{Err: err}
	}

	stop, err := metrics.Bootstrap(metrics.Options{
		PProfAddr: cfg.PProfAddr,
		TraceFile: cfg.TraceFile,
	})
	if err != nil {
		return fmt.Errorf("bootstrap profiling: %w", err)
	}
	defer stop()

	return fmt.Errorf("cannot run %s scan of %q: %w (the pipeline arrives with Phases 4-7; see docs/ROADMAP.md)",
		cfg.Source, cfg.Target, ErrEngineNotWired)
}
