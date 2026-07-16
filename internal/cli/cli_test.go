package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Roro1727/airom/internal/app"
)

// captureScan substitutes the app.Run seam and returns a pointer that will
// hold the Config the command resolved.
func captureScan(t *testing.T) **app.Config {
	t.Helper()
	var captured *app.Config
	orig := runScan
	runScan = func(_ context.Context, cfg *app.Config) error {
		captured = cfg
		return nil
	}
	t.Cleanup(func() { runScan = orig })
	return &captured
}

// execute runs the root command with args and returns stdout and the error.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd(BuildInfo{Version: "test", Commit: "abc", Date: "today"})
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return out.String(), err
}

func TestHelpRuns(t *testing.T) {
	out, err := execute(t, "--help")
	if err != nil {
		t.Fatalf("--help error: %v", err)
	}
	for _, cmd := range []string{"scan", "fs", "repo", "image", "k8s", "clean", "version"} {
		if !strings.Contains(out, cmd) {
			t.Errorf("--help missing command %q", cmd)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	out, err := execute(t, "version")
	if err != nil {
		t.Fatalf("version error: %v", err)
	}
	for _, want := range []string{"airom test", "commit: abc", "built:  today", "go:"} {
		if !strings.Contains(out, want) {
			t.Errorf("version output missing %q in:\n%s", want, out)
		}
	}
}

func TestFSResolvesConfig(t *testing.T) {
	got := captureScan(t)
	t.Chdir(t.TempDir())

	_, err := execute(t, "fs", ".",
		"--parallel", "3",
		"-o", "table", "-o", "cyclonedx=bom.json",
		"--io-budget", "64m",
		"--fail-on", "hosted-llm&confidence>=0.9",
		"--ignore", "**/fixtures/**",
	)
	if err != nil {
		t.Fatalf("fs error: %v", err)
	}
	cfg := *got
	if cfg == nil {
		t.Fatal("runScan not invoked")
	}
	if cfg.Source != app.SourceFS || cfg.Target != "." {
		t.Errorf("source/target = %v/%q", cfg.Source, cfg.Target)
	}
	if cfg.Parallel != 3 {
		t.Errorf("Parallel = %d, want 3", cfg.Parallel)
	}
	if cfg.IOBudget != 64<<20 {
		t.Errorf("IOBudget = %d, want 64MiB", cfg.IOBudget)
	}
	if len(cfg.Outputs) != 2 || cfg.Outputs[1].Path != "bom.json" {
		t.Errorf("Outputs = %v", cfg.Outputs)
	}
	if cfg.Policy == nil {
		t.Error("Policy = nil, want parsed --fail-on policy")
	}
	if len(cfg.IgnoreGlobs) != 1 || cfg.IgnoreGlobs[0] != "**/fixtures/**" {
		t.Errorf("IgnoreGlobs = %v", cfg.IgnoreGlobs)
	}
}

func TestConfigPrecedence(t *testing.T) {
	got := captureScan(t)
	dir := t.TempDir()
	yaml := "parallel: 4\nio-budget: 32m\noutput:\n  - yaml\nmin-confidence: 0.5\n"
	if err := os.WriteFile(filepath.Join(dir, ".airom.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	// file only: file beats defaults
	if _, err := execute(t, "fs", "."); err != nil {
		t.Fatalf("fs error: %v", err)
	}
	cfg := *got
	if cfg.Parallel != 4 || cfg.IOBudget != 32<<20 || cfg.MinConfidence != 0.5 {
		t.Errorf("file layer not applied: parallel=%d io=%d conf=%v", cfg.Parallel, cfg.IOBudget, cfg.MinConfidence)
	}
	if len(cfg.Outputs) != 1 || cfg.Outputs[0].Format != app.FormatYAML {
		t.Errorf("Outputs = %v, want yaml from file", cfg.Outputs)
	}

	// env beats file
	t.Setenv("AIROM_PARALLEL", "8")
	t.Setenv("AIROM_OUTPUT", "table,sarif=airom.sarif")
	if _, err := execute(t, "fs", "."); err != nil {
		t.Fatalf("fs error: %v", err)
	}
	cfg = *got
	if cfg.Parallel != 8 {
		t.Errorf("Parallel = %d, want env 8 over file 4", cfg.Parallel)
	}
	if len(cfg.Outputs) != 2 || cfg.Outputs[1].Path != "airom.sarif" {
		t.Errorf("Outputs = %v, want env list", cfg.Outputs)
	}

	// flag beats env
	if _, err := execute(t, "fs", ".", "--parallel", "2"); err != nil {
		t.Fatalf("fs error: %v", err)
	}
	cfg = *got
	if cfg.Parallel != 2 {
		t.Errorf("Parallel = %d, want flag 2 over env 8", cfg.Parallel)
	}
	// unrelated file keys still apply
	if cfg.IOBudget != 32<<20 {
		t.Errorf("IOBudget = %d, want file 32MiB", cfg.IOBudget)
	}
}

func TestFormatAlias(t *testing.T) {
	got := captureScan(t)
	t.Chdir(t.TempDir())

	if _, err := execute(t, "fs", ".", "--format", "json"); err != nil {
		t.Fatalf("fs error: %v", err)
	}
	cfg := *got
	if len(cfg.Outputs) != 1 || cfg.Outputs[0].Format != app.FormatJSON {
		t.Errorf("Outputs = %v, want single json", cfg.Outputs)
	}

	if _, err := execute(t, "fs", ".", "--format", "json", "-o", "table"); err == nil {
		t.Error("want error for --format with -o, got nil")
	}
}

func TestExitCodeImpliesMatchAny(t *testing.T) {
	got := captureScan(t)
	t.Chdir(t.TempDir())

	if _, err := execute(t, "fs", ".", "--exit-code", "5"); err != nil {
		t.Fatalf("fs error: %v", err)
	}
	cfg := *got
	if cfg.Policy == nil {
		t.Fatal("Policy = nil, want match-any when --exit-code set without --fail-on")
	}
	if cfg.ExitCode != 5 {
		t.Errorf("ExitCode = %d, want 5", cfg.ExitCode)
	}
}

func TestBadFailOnIsUsageError(t *testing.T) {
	t.Chdir(t.TempDir())
	_, err := execute(t, "fs", ".", "--fail-on", "confidence>>1")
	if err == nil {
		t.Fatal("want parse error, got nil")
	}
}

func TestImageArgValidation(t *testing.T) {
	captureScan(t)
	if _, err := execute(t, "image"); err == nil || !strings.Contains(err.Error(), "reference or --input") {
		t.Errorf("image with no args: err = %v", err)
	}
	if _, err := execute(t, "image", "ubuntu:24.04", "--input", "x.tar"); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("image ref+input: err = %v", err)
	}
}

func TestScanAutoDetect(t *testing.T) {
	got := captureScan(t)
	dir := t.TempDir()
	t.Chdir(dir)

	if _, err := execute(t, "scan", "."); err != nil {
		t.Fatalf("scan .: %v", err)
	}
	if (*got).Source != app.SourceFS {
		t.Errorf("scan . source = %v, want fs", (*got).Source)
	}

	if _, err := execute(t, "scan", "https://github.com/acme/x.git"); err != nil {
		t.Fatalf("scan git url: %v", err)
	}
	if (*got).Source != app.SourceRepo {
		t.Errorf("scan git url source = %v, want repo", (*got).Source)
	}

	if _, err := execute(t, "scan", "image:ubuntu:24.04"); err != nil {
		t.Fatalf("scan image: %v", err)
	}
	if (*got).Source != app.SourceImage || (*got).Target != "ubuntu:24.04" {
		t.Errorf("scan forced image = %v %q", (*got).Source, (*got).Target)
	}
}

func TestCleanCommand(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "airom-cache")
	if err := os.MkdirAll(filepath.Join(cacheDir, "ns1"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())

	out, err := execute(t, "clean", "--cache-dir", cacheDir)
	if err != nil {
		t.Fatalf("clean error: %v", err)
	}
	if !strings.Contains(out, "removed cache") {
		t.Errorf("clean output = %q", out)
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Error("cache dir still exists after clean")
	}

	// idempotent: cleaning again reports no cache
	out, err = execute(t, "clean", "--cache-dir", cacheDir)
	if err != nil {
		t.Fatalf("second clean error: %v", err)
	}
	if !strings.Contains(out, "no cache") {
		t.Errorf("second clean output = %q", out)
	}
}

func TestQuietVerboseConflict(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := execute(t, "fs", ".", "-q", "-v"); err == nil {
		t.Error("want -q/-v conflict error, got nil")
	}
}
