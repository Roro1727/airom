package tui

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestIsTTYRejectsNonTerminals(t *testing.T) {
	// A pipe is not a terminal — this is the check that keeps CI and redirected
	// output free of spinner bytes.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close(); _ = w.Close() }()
	if IsTTY(w) {
		t.Error("IsTTY(pipe) = true, want false")
	}
	if IsTTY(nil) {
		t.Error("IsTTY(nil) = true, want false")
	}

	f, err := os.CreateTemp(t.TempDir(), "f")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if IsTTY(f) {
		t.Error("IsTTY(regular file) = true, want false")
	}
}

func TestColorEnabledHonorsEnvironment(t *testing.T) {
	r, w, _ := os.Pipe()
	defer func() { _ = r.Close(); _ = w.Close() }()

	t.Setenv("NO_COLOR", "1")
	t.Setenv("CLICOLOR_FORCE", "")
	if ColorEnabled(w) {
		t.Error("NO_COLOR set: want color disabled")
	}

	// CLICOLOR_FORCE wins over everything, including a non-TTY.
	t.Setenv("CLICOLOR_FORCE", "1")
	if !ColorEnabled(w) {
		t.Error("CLICOLOR_FORCE=1: want color enabled")
	}

	t.Setenv("CLICOLOR_FORCE", "")
	t.Setenv("NO_COLOR", "")
	os.Unsetenv("NO_COLOR") //nolint:errcheck // best-effort cleanup; t.Setenv restores it
	t.Setenv("TERM", "dumb")
	if ColorEnabled(w) {
		t.Error("TERM=dumb: want color disabled")
	}
}

func TestStyleIsNoOpWhenDisabled(t *testing.T) {
	off := Style{codes: sgrBold, enabled: false}
	if got := off.S("x"); got != "x" {
		t.Errorf("disabled style = %q, want plain %q", got, "x")
	}
	on := Style{codes: sgrBold, enabled: true}
	if got := on.S("x"); !strings.Contains(got, "\x1b[") {
		t.Errorf("enabled style = %q, want ANSI codes", got)
	}
	if got := on.S(""); got != "" {
		t.Errorf("styling empty string = %q, want empty", got)
	}
}

// TestDisabledSpinnerWritesNothing is the load-bearing one: off a terminal the
// spinner must emit ZERO bytes of its own. Anything else would corrupt piped
// output and break the byte-identical contract.
func TestDisabledSpinnerWritesNothing(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner(&buf, false, Palette{})
	s.Start(func() string { return "scanning" })
	s.Stop()
	if buf.Len() != 0 {
		t.Errorf("disabled spinner wrote %q, want nothing", buf.String())
	}
}

func TestDisabledSpinnerPassesWritesThrough(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner(&buf, false, Palette{})
	n, err := s.Write([]byte("log line\n"))
	if err != nil || n != 9 {
		t.Fatalf("Write = (%d, %v)", n, err)
	}
	if buf.String() != "log line\n" {
		t.Errorf("passthrough = %q, want the log line verbatim", buf.String())
	}
}

// TestSpinnerWriteClearsItsLine: a log record written through an ACTIVE spinner
// must be preceded by a line clear, so the two do not shred each other.
func TestSpinnerWriteClearsItsLine(t *testing.T) {
	var buf lockedBuf
	s := NewSpinner(&buf, true, Palette{})
	s.mu.Lock()
	s.drawn = true // pretend a frame is on screen, without racing the ticker
	s.mu.Unlock()

	if _, err := s.Write([]byte("warn: something\n")); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.HasPrefix(got, clearLine) {
		t.Errorf("write = %q, want it to start with a line clear", got)
	}
	if !strings.HasSuffix(got, "warn: something\n") {
		t.Errorf("write = %q, want the log record intact at the end", got)
	}
}

func TestSpinnerStopIsIdempotent(_ *testing.T) {
	var buf lockedBuf
	s := NewSpinner(&buf, true, Palette{})
	s.Stop() // never started
	s.Start(func() string { return "x" })
	s.Stop()
	s.Stop() // must not panic or deadlock
}

// lockedBuf is a bytes.Buffer safe for the ticker goroutine to write to
// concurrently with the test reading it.
type lockedBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuf) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuf) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
