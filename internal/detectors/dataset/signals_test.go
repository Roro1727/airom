package dataset

import (
	"context"
	"testing"

	"github.com/airomhq/airom/pkg/airom/detect"
)

// bom is spelled as an escape on purpose: a literal U+FEFF is illegal in a Go
// source file past byte 0, and invisible in review besides.
const bom = "\ufeff"

// TestBOMDoesNotDeleteRealDatasets pins a total-drop bug: bytes.TrimSpace does
// not strip U+FEFF (it is White_Space=No), so a BOM used to corrupt the first
// column name and — worse — hide a JSON line's leading brace, taking the file
// from a finding to silence. Excel's "Save as CSV UTF-8", pandas'
// encoding="utf-8-sig" and PowerShell's Export-Csv all emit one.
func TestBOMDoesNotDeleteRealDatasets(t *testing.T) {
	cases := []struct{ name, path, body string }{
		{"classification csv", "reviews.csv", "text,label\nnice,pos\n"},
		{"SFT jsonl under data/", "data/train.jsonl", `{"prompt":"hi","completion":"yo"}` + "\n"},
		{"SQuAD shape", "datasets/squad.jsonl", `{"question":"q","answer":"a","context":"c"}` + "\n"},
		{"DPO preference data", "prefs.jsonl", `{"chosen":"a","rejected":"b"}` + "\n"},
	}
	for _, c := range cases {
		plain, err := NewDataset().DetectFile(context.Background(), file(t, c.path, c.body))
		if err != nil {
			t.Fatal(err)
		}
		withBOM, err := NewDataset().DetectFile(context.Background(), file(t, c.path, bom+c.body))
		if err != nil {
			t.Fatal(err)
		}
		if len(plain) != 1 {
			t.Fatalf("%s (%s): fixture is wrong — got %d findings without a BOM, want 1", c.name, c.path, len(plain))
		}
		if len(withBOM) != 1 {
			t.Errorf("%s (%s): a BOM dropped the file entirely (%d findings, want 1)", c.name, c.path, len(withBOM))
			continue
		}
		if withBOM[0].Occurrence.Confidence != plain[0].Occurrence.Confidence {
			t.Errorf("%s (%s): BOM changed confidence %.2f -> %.2f; an encoding artifact is not evidence",
				c.name, c.path, plain[0].Occurrence.Confidence, withBOM[0].Occurrence.Confidence)
		}
	}
}

// TestNameSignalSurvivesAParseFailure: a failed structural parse is a missing
// signal, not a verdict. The Parquet branch always fell back to the name; the
// CSV/JSONL branches used to return before ever consulting it, so a real
// dataset whose header simply did not parse was deleted rather than downgraded.
func TestNameSignalSurvivesAParseFailure(t *testing.T) {
	cases := []struct{ name, path, body string }{
		{"semicolon CSV (EU Excel locale)", "data/train.csv", "text;label\nnice;pos\n"},
		{"single-column corpus", "data/train.csv", "text\nhello world\n"},
		{"JSONL of bare arrays", "data/train.jsonl", "[1,2,3]\n[4,5,6]\n"},
		{"leading blank line", "data/train.jsonl", "\n{\"a\":1}\n"},
	}
	for _, c := range cases {
		got, err := NewDataset().DetectFile(context.Background(), file(t, c.path, c.body))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("%s (%s): got %d findings, want 1 at name grade — the path still says dataset",
				c.name, c.path, len(got))
			continue
		}
		if got := got[0].Occurrence.Confidence; float64(got) != confName {
			t.Errorf("%s (%s): confidence %.2f, want %.2f (name-grade: the fields did not corroborate)",
				c.name, c.path, got, confName)
		}
	}
}

// TestUncorroboratedFilesStaySilent: the precision pass must still hold. These
// parse cleanly and are named nothing in particular, so they are some other
// program's data.
func TestUncorroboratedFilesStaySilent(t *testing.T) {
	cases := []struct{ path, body string }{
		{"vendor/oui.csv", "oui,organization\n00-00-0C,Cisco\n"},
		{"ui/colors.csv", "hex,name\n#fff,white\n"},
		{"metrics/export.jsonl", `{"metric":"cpu","value":1}` + "\n"},
		{"test-linux.csv", "arch,result\namd64,pass\n"},
	}
	for _, c := range cases {
		got, err := NewDataset().DetectFile(context.Background(), file(t, c.path, c.body))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("%s: reported at %.2f; parsing as CSV/JSONL is not evidence of a dataset",
				c.path, got[0].Occurrence.Confidence)
		}
	}
}

// file builds a detect.File whose header is the content, mirroring NeedHeader.
func file(t *testing.T, p, content string) *detect.File {
	t.Helper()
	b := []byte(content)
	return detect.NewFile(
		detect.FileRef{Path: p, Size: int64(len(b))},
		b,
		detect.FileProviders{Content: func() ([]byte, bool, error) { return b, false, nil }},
	)
}
