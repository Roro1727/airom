package tui

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// spinnerFrames is the conventional braille spinner. Rendered only on a real
// terminal, so the UTF-8 requirement is safe.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const (
	tickInterval = 100 * time.Millisecond
	clearLine    = "\r\x1b[2K" // carriage return + erase the whole line
	hideCursor   = "\x1b[?25l"
	showCursor   = "\x1b[?25h"
)

// Spinner is a single-line progress indicator for a long-running scan.
//
// It doubles as the stderr multiplexer: log records are written *through* it
// (see Write), so a warning emitted mid-scan erases the spinner line, prints
// cleanly, and lets the next tick redraw. Without that cooperation the two
// writers interleave and shred each other's output.
//
// A disabled Spinner (not a terminal, --quiet, --no-progress) renders nothing
// and passes writes straight through, so non-interactive output is byte-for-byte
// what it was before progress existed.
type Spinner struct {
	w       io.Writer
	enabled bool
	pal     Palette

	mu    sync.Mutex
	drawn bool // a spinner line is currently on screen
	frame int
	label func() string

	stop chan struct{}
	done chan struct{}
}

// NewSpinner returns a Spinner writing to w. When enabled is false every method
// is a no-op beyond passing writes through.
func NewSpinner(w io.Writer, enabled bool, pal Palette) *Spinner {
	return &Spinner{w: w, enabled: enabled, pal: pal}
}

// Start begins animating. label is called on every tick for the status text, so
// it must be cheap and safe to call from another goroutine (read atomics, not
// locks the scan holds). Calling Start twice without Stop is a no-op.
func (s *Spinner) Start(label func() string) {
	if !s.enabled {
		return
	}
	s.mu.Lock()
	if s.stop != nil { // already running
		s.mu.Unlock()
		return
	}
	s.label = label
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	stop, done := s.stop, s.done
	fmt.Fprint(s.w, hideCursor)
	s.mu.Unlock()

	go func() {
		defer close(done)
		t := time.NewTicker(tickInterval)
		defer t.Stop()
		s.draw() // paint immediately; a fast scan should still show something
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				s.draw()
			}
		}
	}()
}

// Stop halts the animation and erases the spinner line. Safe to call more than
// once, and safe to call when never started.
func (s *Spinner) Stop() {
	if !s.enabled {
		return
	}
	s.mu.Lock()
	stop, done := s.stop, s.done
	s.stop, s.done = nil, nil
	s.mu.Unlock()
	if stop == nil {
		return
	}
	close(stop)
	<-done // the draw loop cannot be mid-write when we clear below

	s.mu.Lock()
	s.clearLocked()
	fmt.Fprint(s.w, showCursor)
	s.mu.Unlock()
}

// Write implements io.Writer so log output can share stderr with the spinner:
// the spinner line is erased, p is written at column zero, and the next tick
// redraws. Route slog through this and warnings during a scan stay readable.
func (s *Spinner) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearLocked()
	return s.w.Write(p)
}

// draw renders one frame.
func (s *Spinner) draw() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.label == nil {
		return
	}
	text := s.label()
	frame := spinnerFrames[s.frame%len(spinnerFrames)]
	s.frame++
	fmt.Fprint(s.w, clearLine+s.pal.Accent.S(frame)+" "+text)
	s.drawn = true
}

// clearLocked erases the spinner line if one is on screen. Caller holds mu.
func (s *Spinner) clearLocked() {
	if !s.enabled || !s.drawn {
		return
	}
	fmt.Fprint(s.w, clearLine)
	s.drawn = false
}
