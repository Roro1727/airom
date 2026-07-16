package yamlw_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/Roro1727/airom/internal/writer/writertest"
	"github.com/Roro1727/airom/internal/writer/yamlw"
)

var update = flag.Bool("update", false, "update golden files")

func TestGoldenAndDeterminism(t *testing.T) {
	inv := writertest.BuildFixture()
	var a, b bytes.Buffer
	if err := (yamlw.Writer{}).Write(&a, inv); err != nil {
		t.Fatal(err)
	}
	_ = (yamlw.Writer{}).Write(&b, inv)
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Error("yaml not deterministic (P7)")
	}
	golden := filepath.Join("testdata", "inventory.golden.yaml")
	if *update {
		_ = os.MkdirAll("testdata", 0o750)
		if err := os.WriteFile(golden, a.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run -update): %v", err)
	}
	if !bytes.Equal(want, a.Bytes()) {
		t.Error("yaml differs from golden")
	}
}
