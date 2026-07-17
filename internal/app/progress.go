package app

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/airomhq/airom/internal/engine"
	"github.com/airomhq/airom/internal/tui"
)

// stderr is the diagnostics/progress destination, injectable for tests. It is
// deliberately NOT stdout: stdout carries the AIBOM, and invariant P7 requires
// it to be byte-identical for identical inputs.
var stderr *os.File = os.Stderr

// startProgress begins a progress display for a scan of target, unless it would
// be inappropriate or harmful:
//
//   - not a terminal (piped, redirected, CI) — the bytes must not change
//   - --quiet — the user asked for errors only
//   - --no-progress — the user said so explicitly
//
// It returns a live counter to hand to the engine (nil when disabled, which
// costs the pipeline nothing) and a stop func that is always safe to call.
//
// While active, slog is routed through the spinner so a warning mid-scan erases
// the spinner line, prints cleanly, and lets the next tick redraw. The previous
// logger is restored on stop.
func startProgress(cfg *Config, target string) (*engine.Live, func()) {
	enabled := !cfg.Quiet && !cfg.NoProgress && tui.IsTTY(stderr)
	if !enabled {
		return nil, func() {}
	}

	pal := tui.NewPalette(stderr)
	spin := tui.NewSpinner(stderr, true, pal)
	live := &engine.Live{}

	label := func() string {
		walked, done := live.Walked.Load(), live.Processed.Load()
		var b strings.Builder
		b.WriteString(pal.Bold.S("scanning"))
		b.WriteByte(' ')
		b.WriteString(pal.Dim.S(shortenPath(target)))
		b.WriteString(pal.Dim.S("  ·  "))
		b.WriteString(humanCount(walked))
		b.WriteString(pal.Dim.S(" walked"))
		if done > 0 {
			b.WriteString(pal.Dim.S(", "))
			b.WriteString(humanCount(done))
			b.WriteString(pal.Dim.S(" read"))
		}
		return b.String()
	}

	// Share stderr with the log stream while the spinner owns the line.
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(spin, &slog.HandlerOptions{
		Level: currentLogLevel(),
	})))

	spin.Start(label)
	return live, func() {
		spin.Stop()
		slog.SetDefault(prev)
	}
}

// currentLogLevel reports the level the default logger is emitting at, so
// rerouting through the spinner does not silently change verbosity.
func currentLogLevel() slog.Level {
	for _, l := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		if slog.Default().Enabled(nil, l) { //nolint:staticcheck // a nil ctx is accepted here
			return l
		}
	}
	return slog.LevelError
}

// humanCount renders a count with thousands separators: 1284 -> "1,284".
func humanCount(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	lead := len(s) % 3
	if lead > 0 {
		b.WriteString(s[:lead])
	}
	for i := lead; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// shortenPath keeps a long target readable on one line, keeping the tail (the
// distinctive end of a path) and eliding the head.
//
// Counted in runes, not bytes: the ellipsis is one column but three bytes, and
// a byte slice could also split a multi-byte character in half.
func shortenPath(p string) string {
	const maxRunes = 48
	r := []rune(p)
	if len(r) <= maxRunes {
		return p
	}
	return "…" + string(r[len(r)-maxRunes+1:])
}
