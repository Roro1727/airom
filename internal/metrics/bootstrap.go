package metrics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime/trace"
	"time"

	// Blank import registers the pprof handlers on http.DefaultServeMux,
	// which the --pprof server below serves. The server only exists when
	// the user asks for it; nothing is exposed by default.
	_ "net/http/pprof" // #nosec G108 -- served only on explicit --pprof; default bind is localhost
)

// Options selects the profiling facilities for one run (docs/cli.md:
// --pprof, --trace are product features, not test-only).
type Options struct {
	PProfAddr string // empty = disabled; CLI defaults the bare flag to localhost:6060
	TraceFile string // empty = disabled
}

// Bootstrap starts the requested profiling facilities and returns a stop
// function that shuts them down and flushes the trace. It fails fast (rather
// than in the background) when the pprof listener cannot bind or the trace
// file cannot be created, so a mistyped flag never silently no-ops.
func Bootstrap(opts Options) (stop func(), err error) {
	var stops []func()
	stop = func() {
		for i := len(stops) - 1; i >= 0; i-- {
			stops[i]()
		}
	}

	if opts.TraceFile != "" {
		f, err := os.Create(opts.TraceFile)
		if err != nil {
			return stop, fmt.Errorf("create trace file: %w", err)
		}
		if err := trace.Start(f); err != nil {
			_ = f.Close() // best-effort: the Start failure is the error that matters
			return stop, fmt.Errorf("start execution trace: %w", err)
		}
		stops = append(stops, func() {
			trace.Stop()
			if err := f.Close(); err != nil {
				slog.Warn("closing trace file", "path", opts.TraceFile, "err", err)
			}
		})
	}

	if opts.PProfAddr != "" {
		ln, err := net.Listen("tcp", opts.PProfAddr)
		if err != nil {
			stop()
			return func() {}, fmt.Errorf("bind pprof listener on %s: %w", opts.PProfAddr, err)
		}
		srv := &http.Server{Handler: http.DefaultServeMux, ReadHeaderTimeout: 5 * time.Second}
		go func() {
			if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Warn("pprof server exited", "err", err)
			}
		}()
		slog.Info("pprof server listening", "addr", ln.Addr().String())
		stops = append(stops, func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
		})
	}

	return stop, nil
}
