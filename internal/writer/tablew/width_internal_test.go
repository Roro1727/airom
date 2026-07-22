package tablew

import (
	"bytes"
	"strings"
	"testing"

	"github.com/airomhq/airom/pkg/airom"
)

func TestDispWidth(t *testing.T) {
	cases := []struct {
		s    string
		want int
	}{
		{"", 0}, {"abc", 3}, {"日本語", 6}, {"a日b", 4}, {"🚀", 2},
	}
	for _, c := range cases {
		if got := dispWidth(c.s); got != c.want {
			t.Errorf("dispWidth(%q) = %d, want %d", c.s, got, c.want)
		}
	}
}

// TestVulnTableWideGlyphRectangular guards the fix for wide (CJK/emoji) advisory
// titles: every border/row line of the per-CVE detail table must have identical
// display width, or the box stops being rectangular in a terminal.
func TestVulnTableWideGlyphRectangular(t *testing.T) {
	inv := &airom.Inventory{
		Source: airom.SourceInfo{Target: "/x"},
		Components: []airom.Component{{
			ID: "a", Kind: airom.KindFramework, Name: "torch", Version: airom.KnownString("2.1.0"), Confidence: 0.9,
			Vulnerabilities: []airom.Vulnerability{{
				ID: "CVE-2024-1", Severity: airom.VulnCritical, Score: 9.8, Fixed: "2.2.0",
				Summary: "远程代码执行漏洞在模型加载器中触发任意命令执行", Source: "osv.dev",
				URL: "https://osv.dev/vulnerability/CVE-2024-1",
			}},
			Evidence: airom.Evidence{Occurrences: []airom.Occurrence{{Location: airom.Location{Path: "requirements.txt", Line: 1}}}},
		}},
	}
	var buf bytes.Buffer
	if err := (Writer{}).Write(&buf, inv); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(buf.String(), "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "Vulnerabilities (") {
			start = i + 1
			break
		}
	}
	if start < 0 {
		t.Fatalf("no vuln detail table in:\n%s", buf.String())
	}
	want := 0
	for _, l := range lines[start:] {
		if l == "" {
			break
		}
		r := []rune(l)
		if len(r) == 0 || (r[0] != '┌' && r[0] != '├' && r[0] != '│' && r[0] != '└') {
			continue
		}
		if w := dispWidth(l); want == 0 {
			want = w
		} else if w != want {
			t.Errorf("vuln box line display-width = %d, want %d (box not rectangular):\n%q", w, want, l)
		}
	}
}
