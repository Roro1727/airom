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
			// No vcs.* stamps: the proxy serves a zip, not a checkout. The
			// commit and time come back out of the pseudo-version itself —
			// see TestParsePseudoVersion.
			name: "go install @latest: a pseudo-version carries the commit and time",
			v:    unsetVersion, c: unsetCommit, d: unsetDate,
			info:  bi("v0.0.0-20260717111035-2f59d1d37c25", nil),
			ok:    true,
			wantV: "v0.0.0-20260717111035-2f59d1d37c25", wantC: "2f59d1d", wD: "2026-07-17T11:10:35Z",
		},
		{
			// A tag names a release and says nothing about a commit. Leave the
			// sentinel rather than invent one.
			name: "go install module@v0.1.1: a real tag yields no commit",
			v:    unsetVersion, c: unsetCommit, d: unsetDate,
			info:  bi("v0.1.1", nil),
			ok:    true,
			wantV: "v0.1.1", wantC: unsetCommit, wD: unsetDate,
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

// TestParsePseudoVersion: `go install` fetches a zip from the module proxy, so
// there are no vcs.* stamps — but the pseudo-version encodes the revision and
// the commit time. Reading them back is the difference between
// "commit none, built unknown" and the truth, right beside a version string that
// spells out both.
func TestParsePseudoVersion(t *testing.T) {
	cases := []struct {
		name, in, wantRev, wantWhen string
	}{
		{
			"no base version (vX.0.0-<ts>-<rev>)",
			"v0.0.0-20260717115955-5ab297f08f99",
			"5ab297f08f99", "2026-07-17T11:59:55Z",
		},
		{
			"after a pre-release (vX.Y.Z-pre.0.<ts>-<rev>)",
			"v0.2.0-beta.0.20260717115955-5ab297f08f99",
			"5ab297f08f99", "2026-07-17T11:59:55Z",
		},
		{
			"after a release (vX.Y.Z-0.<ts>-<rev>)",
			"v0.1.1-0.20260101000000-abcdefabcdef",
			"abcdefabcdef", "2026-01-01T00:00:00Z",
		},
		// A real tag names a release. It says nothing about a commit, and
		// inventing one would be worse than admitting we do not know.
		{"a real tag is not a pseudo-version", "v0.1.0", "", ""},
		{"the dev sentinel", unsetVersion, "", ""},
		{"empty", "", "", ""},
		{"timestamp too short", "v0.0.0-2026071711595-5ab297f08f99", "", ""},
		{"revision too short", "v0.0.0-20260717115955-5ab297f08f9", "", ""},
		{"revision not hex", "v0.0.0-20260717115955-5ab297f08fzz", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rev, when := parsePseudoVersion(c.in)
			if rev != c.wantRev || when != c.wantWhen {
				t.Errorf("parsePseudoVersion(%q) = (%q, %q), want (%q, %q)", c.in, rev, when, c.wantRev, c.wantWhen)
			}
		})
	}
}

// TestResolveRecoversCommitFromAPseudoVersion is the end-to-end of the above:
// the exact shape `go install ...@latest` produces.
func TestResolveRecoversCommitFromAPseudoVersion(t *testing.T) {
	got := resolve(unsetVersion, unsetCommit, unsetDate,
		bi("v0.0.0-20260717115955-5ab297f08f99", nil), true)
	if got.Version != "v0.0.0-20260717115955-5ab297f08f99" {
		t.Errorf("Version = %q", got.Version)
	}
	if got.Commit != "5ab297f" {
		t.Errorf("Commit = %q, want %q — the pseudo-version spells it out", got.Commit, "5ab297f")
	}
	if got.Date != "2026-07-17T11:59:55Z" {
		t.Errorf("Date = %q, want %q", got.Date, "2026-07-17T11:59:55Z")
	}
}

// TestVCSStampsBeatThePseudoVersion: in a real checkout the toolchain records
// the revision directly, including whether the tree is dirty — something a
// pseudo-version cannot express. The stamps must win.
func TestVCSStampsBeatThePseudoVersion(t *testing.T) {
	got := resolve(unsetVersion, unsetCommit, unsetDate,
		bi("v0.0.0-20260717115955-5ab297f08f99", map[string]string{
			"vcs.revision": "deadbeefcafe1234",
			"vcs.time":     "2026-01-01T00:00:00Z",
			"vcs.modified": "true",
		}), true)
	if got.Commit != "deadbee-dirty" {
		t.Errorf("Commit = %q, want %q (the checkout knows more than the proxy)", got.Commit, "deadbee-dirty")
	}
	if got.Date != "2026-01-01T00:00:00Z" {
		t.Errorf("Date = %q, want the vcs.time stamp", got.Date)
	}
}
