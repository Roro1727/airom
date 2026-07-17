package nativejson_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/airomhq/airom/internal/writer/nativejson"
	"github.com/airomhq/airom/internal/writer/writertest"
	"github.com/airomhq/airom/pkg/airom"
)

var update = flag.Bool("update", false, "update golden files")

// TestRoundTrip is the mapping contract (docs/mapping.md §6.4): the native
// format is the lossless superset — write → read reproduces the inventory
// exactly, including all three tri-state states (Known / Unknown / Absent).
func TestRoundTrip(t *testing.T) {
	inv := writertest.BuildFixture()

	var buf bytes.Buffer
	if err := (nativejson.Writer{}).Write(&buf, inv); err != nil {
		t.Fatal(err)
	}

	var got airom.Inventory
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if !reflect.DeepEqual(*inv, got) {
		t.Error("round-trip changed the inventory (lossy native serialization)")
	}

	// Explicit tri-state survival on the model facet (Task Known, Architecture Unknown).
	m := componentByName(t, &got, "gpt-4.1")
	if v, ok := m.Model.Task.Value(); !ok || v != "text-generation" {
		t.Errorf("Task lost Known state: %+v", m.Model.Task)
	}
	if m.Model.Architecture.P != airom.PresenceUnknown {
		t.Errorf("Architecture lost Unknown state: %+v", m.Model.Architecture)
	}
	// A never-set OptString (Absent) must round-trip Absent, not Unknown.
	if m.Model.Quantization.P != airom.PresenceAbsent {
		t.Errorf("Quantization should be Absent, got %+v", m.Model.Quantization)
	}
}

func TestDeterministic(t *testing.T) {
	inv := writertest.BuildFixture()
	var a, b bytes.Buffer
	_ = (nativejson.Writer{}).Write(&a, inv)
	_ = (nativejson.Writer{}).Write(&b, inv)
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Error("native json not byte-identical across runs (P7)")
	}
}

func TestGolden(t *testing.T) {
	inv := writertest.BuildFixture()
	var buf bytes.Buffer
	if err := (nativejson.Writer{}).Write(&buf, inv); err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "inventory.golden.json")
	if *update || os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll("testdata", 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, buf.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run -update): %v", err)
	}
	if !bytes.Equal(want, buf.Bytes()) {
		t.Error("native json differs from golden (run -update after review)")
	}
}

func componentByName(t *testing.T, inv *airom.Inventory, name string) airom.Component {
	t.Helper()
	for _, c := range inv.Components {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no component %q", name)
	return airom.Component{}
}
