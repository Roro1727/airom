package source

import (
	"strings"
	"testing"
)

func TestDetectTarget(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name     string
		target   string
		wantKind TargetKind
		wantTgt  string
		wantErr  string // substring; "" = success
	}{
		{"existing dir", dir, TargetDir, dir, ""},
		{"dot", ".", TargetDir, ".", ""},
		{"forced dir", "dir:" + dir, TargetDir, dir, ""},
		{"forced dir nonexistent", "dir:/no/such/thing", TargetDir, "/no/such/thing", ""},
		{"forced repo", "repo:ubuntu", TargetRepo, "ubuntu", ""},
		{"forced image", "image:ubuntu:24.04", TargetImage, "ubuntu:24.04", ""},
		{"https git", "https://github.com/acme/rag-service.git", TargetRepo, "https://github.com/acme/rag-service.git", ""},
		{"https no .git", "https://github.com/acme/rag-service", TargetRepo, "https://github.com/acme/rag-service", ""},
		{"scp-like", "git@github.com:acme/x.git", TargetRepo, "git@github.com:acme/x.git", ""},
		{"ssh", "ssh://git@host/x.git", TargetRepo, "ssh://git@host/x.git", ""},
		{"image tag", "ubuntu:24.04", TargetImage, "ubuntu:24.04", ""},
		{"image registry", "ghcr.io/acme/inference:v3", TargetImage, "ghcr.io/acme/inference:v3", ""},
		{"image digest", "alpine@sha256:abcd", TargetImage, "alpine@sha256:abcd", ""},
		{"missing rel path", "./does-not-exist", "", "", "does not exist"},
		{"missing abs path", "/does/not/exist", "", "", "does not exist"},
		{"empty", "", "", "", "empty"},
		{"empty prefix", "image:", "", "", "needs a reference"},
		{"garbage", "not a target!!", "", "", "cannot interpret"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, tgt, err := DetectTarget(tc.target)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("DetectTarget(%q) err = %v, want containing %q", tc.target, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("DetectTarget(%q) unexpected error: %v", tc.target, err)
			}
			if kind != tc.wantKind || tgt != tc.wantTgt {
				t.Fatalf("DetectTarget(%q) = (%q, %q), want (%q, %q)", tc.target, kind, tgt, tc.wantKind, tc.wantTgt)
			}
		})
	}
}
