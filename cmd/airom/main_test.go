package main

import (
	"runtime/debug"
	"testing"
)

// bi builds a synthetic *debug.BuildInfo: the module version plus the VCS
// settings the toolchain stamps into a build.
func bi(mainVersion string, settings map[string]string) *debug.BuildInfo {
	out := &debug.BuildInfo{}
	out.Main.Version = mainVersion
	for k, v := range settings {
		out.Settings = append(out.Settings, debug.BuildSetting{Key: k, Value: v})
	}
	return out
}

// TestResolveRecoversVersionWithoutLdflags is the point of the fallback:
// `go install …@latest` cannot pass ldflags, so without this every AIBOM a
// pip/go-installed airom emits would claim tool.version "dev".
func TestResolveRecoversVersionWithoutLdflags(t *testing.T) {
	cases := []struct {
		name             string
		v, c, d          string
		info             *debug.BuildInfo
		ok               bool
		wantV, wantC, wD string
	}{
		{
			name: "ldflags win over build info",
			v:    "v0.1.0", c: "abc1234", d: "2026-07-17T00:00:00Z",
			info:  bi("v9.9.9", map[string]string{"vcs.revision": "deadbeefcafe"}),
			ok:    true,
			wantV: "v0.1.0", wantC: "abc1234", wD: "2026-07-17T00:00:00Z",
		},
		{
			name: "go install module@version: recover the module version",
			v:    unsetVersion, c: unsetCommit, d: unsetDate,
			info:  bi("v0.1.1", nil),
			ok:    true,
			wantV: "v0.1.1", wantC: unsetCommit, wD: unsetDate,
		},
		{
			name: "go install @latest: a pseudo-version is still the truth",
			v:    unsetVersion, c: unsetCommit, d: unsetDate,
			info:  bi("v0.0.0-20260717111035-2f59d1d37c25", nil),
			ok:    true,
			wantV: "v0.0.0-20260717111035-2f59d1d37c25", wantC: unsetCommit, wD: unsetDate,
		},
		{
			name: "go build in a checkout: VCS stamps fill commit, date, version",
			v:    unsetVersion, c: unsetCommit, d: unsetDate,
			info: bi("(devel)", map[string]string{
				"vcs.revision": "2f59d1d37c25aaaabbbb",
				"vcs.time":     "2026-07-17T11:10:35Z",
				"vcs.modified": "false",
			}),
			ok:    true,
			wantV: "devel-2f59d1d", wantC: "2f59d1d", wD: "2026-07-17T11:10:35Z",
		},
		{
			name: "a dirty worktree says so",
			v:    unsetVersion, c: unsetCommit, d: unsetDate,
			info: bi("(devel)", map[string]string{
				"vcs.revision": "2f59d1d37c25aaaabbbb",
				"vcs.modified": "true",
			}),
			ok:    true,
			wantV: "devel-2f59d1d-dirty", wantC: "2f59d1d-dirty", wD: unsetDate,
		},
		{
			name: "(devel) is not a version and must never be reported as one",
			v:    unsetVersion, c: unsetCommit, d: unsetDate,
			info:  bi("(devel)", nil),
			ok:    true,
			wantV: unsetVersion, wantC: unsetCommit, wD: unsetDate,
		},
		{
			name: "no build info: fall back to the sentinels rather than crash",
			v:    unsetVersion, c: unsetCommit, d: unsetDate,
			info:  nil,
			ok:    false,
			wantV: unsetVersion, wantC: unsetCommit, wD: unsetDate,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolve(tc.v, tc.c, tc.d, tc.info, tc.ok)
			if got.Version != tc.wantV {
				t.Errorf("Version = %q, want %q", got.Version, tc.wantV)
			}
			if got.Commit != tc.wantC {
				t.Errorf("Commit = %q, want %q", got.Commit, tc.wantC)
			}
			if got.Date != tc.wD {
				t.Errorf("Date = %q, want %q", got.Date, tc.wD)
			}
		})
	}
}

func TestShortRev(t *testing.T) {
	cases := []struct{ in, want string }{
		{"2f59d1d37c25aaaabbbb", "2f59d1d"},
		{"abc", "abc"},
		// Trim before truncating: padding must never survive into the output,
		// which it would if the length test ran against the untrimmed string.
		{"  2f59d1d37c25aaaabbbb  ", "2f59d1d"},
		{"  abc  ", "abc"},
		{"", ""},
	}
	for _, c := range cases {
		if got := shortRev(c.in); got != c.want {
			t.Errorf("shortRev(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
