package app

import (
	"runtime"
	"strings"
	"testing"
)

func validConfig() *Config {
	return &Config{
		Source:  SourceFS,
		Target:  ".",
		Outputs: []OutputSpec{{Format: FormatTable}},
	}
}

func TestApplyDefaults(t *testing.T) {
	c := &Config{Source: SourceFS, Target: "."}
	c.ApplyDefaults()

	if c.Parallel != runtime.GOMAXPROCS(0) {
		t.Errorf("Parallel = %d, want GOMAXPROCS", c.Parallel)
	}
	if c.IOBudget != 256<<20 {
		t.Errorf("IOBudget = %d, want 256MiB", c.IOBudget)
	}
	if c.MaxFileSize != 1<<20 {
		t.Errorf("MaxFileSize = %d, want 1MiB", c.MaxFileSize)
	}
	if c.CDXVersion != "1.6" {
		t.Errorf("CDXVersion = %q, want 1.6", c.CDXVersion)
	}
	if c.CacheDir == "" {
		t.Error("CacheDir empty after defaults")
	}
	if len(c.Outputs) != 1 || c.Outputs[0].Format != FormatTable {
		t.Errorf("Outputs = %v, want default table", c.Outputs)
	}
	if c.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0 with no policy", c.ExitCode)
	}
}

func TestApplyDefaultsPolicyExitCode(t *testing.T) {
	c := validConfig()
	c.Policy = MatchAny()
	c.ApplyDefaults()
	if c.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want documented default 1 when policy active", c.ExitCode)
	}

	c2 := validConfig()
	c2.Policy = MatchAny()
	c2.ExitCode = 3
	c2.ApplyDefaults()
	if c2.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want explicit 3 preserved", c2.ExitCode)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr string // substring; "" = valid
	}{
		{"valid", func(*Config) {}, ""},
		{"bad source", func(c *Config) { c.Source = "ftp" }, "unknown source"},
		{"no target", func(c *Config) { c.Target = "" }, "no scan target"},
		{"k8s no target ok", func(c *Config) { c.Source = SourceK8s; c.Target = "" }, ""},
		{"image input no target ok", func(c *Config) { c.Source = SourceImage; c.Target = ""; c.ImageInput = "x.tar" }, ""},
		{"neg confidence", func(c *Config) { c.MinConfidence = -0.1 }, "min-confidence"},
		{"confidence >1", func(c *Config) { c.MinConfidence = 1.1 }, "min-confidence"},
		{"bad cdx", func(c *Config) { c.CDXVersion = "2.0" }, "cdx-version"},
		{"bad format", func(c *Config) { c.Outputs = []OutputSpec{{Format: "xml"}} }, "unknown output format"},
		{"two stdout", func(c *Config) {
			c.Outputs = []OutputSpec{{Format: FormatTable}, {Format: FormatJSON}}
		}, "stdout"},
		{"two with paths ok", func(c *Config) {
			c.Outputs = []OutputSpec{{Format: FormatTable}, {Format: FormatJSON, Path: "a.json"}}
		}, ""},
		{"bad exit code", func(c *Config) { c.ExitCode = 300 }, "exit-code"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validConfig()
			tc.mutate(c)
			c.ApplyDefaults()
			err := c.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestParseFormat(t *testing.T) {
	for _, ok := range []string{"table", "JSON", " cyclonedx ", "sarif", "yaml"} {
		if _, err := ParseFormat(ok); err != nil {
			t.Errorf("ParseFormat(%q) unexpected error: %v", ok, err)
		}
	}
	if _, err := ParseFormat("spdx"); err == nil {
		t.Error("ParseFormat(spdx): want error (v2 format), got nil")
	}
}
