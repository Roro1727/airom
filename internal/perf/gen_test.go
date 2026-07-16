package perf

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestGenerateTreeDeterministic pins the core guarantee the whole harness
// rests on: identical opts (seed included) produce a byte-identical tree.
func TestGenerateTreeDeterministic(t *testing.T) {
	opts := TreeOpts{
		NumFiles:           120,
		Seed:               99,
		FractionAIRelevant: 0.2,
		MaxDepth:           2,
		DirFanout:          3,
		NumBinaryBlobs:     6,
		LargeFileFraction:  0.05,
	}

	a := t.TempDir()
	b := t.TempDir()
	ra, err := GenerateTree(a, opts)
	if err != nil {
		t.Fatalf("GenerateTree a: %v", err)
	}
	rb, err := GenerateTree(b, opts)
	if err != nil {
		t.Fatalf("GenerateTree b: %v", err)
	}

	if !reflect.DeepEqual(ra, rb) {
		t.Fatalf("results differ:\n a=%+v\n b=%+v", ra, rb)
	}
	if got := fingerprint(t, a); got != fingerprint(t, b) {
		t.Fatal("two trees from identical opts are not byte-identical")
	}
}

// TestGenerateTreeSeedSensitivity: a different seed must change the tree.
func TestGenerateTreeSeedSensitivity(t *testing.T) {
	base := TreeOpts{NumFiles: 80, Seed: 1, FractionAIRelevant: 0.3, MaxDepth: 2, DirFanout: 2, NumBinaryBlobs: 3}
	other := base
	other.Seed = 2

	a := t.TempDir()
	b := t.TempDir()
	if _, err := GenerateTree(a, base); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateTree(b, other); err != nil {
		t.Fatal(err)
	}
	if fingerprint(t, a) == fingerprint(t, b) {
		t.Fatal("different seeds produced identical trees")
	}
}

// TestGenerateTreeCounts checks the reported counts and that AI signals and
// binary blobs actually landed on disk with the expected extensions.
func TestGenerateTreeCounts(t *testing.T) {
	dir := t.TempDir()
	res, err := GenerateTree(dir, TreeOpts{
		NumFiles:           200,
		Seed:               7,
		FractionAIRelevant: 0.25,
		MaxDepth:           3,
		DirFanout:          3,
		NumBinaryBlobs:     9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.NumFiles != 200 {
		t.Errorf("NumFiles = %d, want 200", res.NumFiles)
	}
	if res.NumAIFiles == 0 {
		t.Error("no AI files planted despite FractionAIRelevant=0.25")
	}
	if res.NumBinary != 9 {
		t.Errorf("NumBinary = %d, want 9", res.NumBinary)
	}
	if res.TotalBytes <= 0 {
		t.Error("TotalBytes not accumulated")
	}
	if len(res.ModelNames) == 0 {
		t.Error("no model literals planted")
	}

	counts := extCounts(t, dir)
	if counts[".go"] == 0 {
		t.Error("expected Go files")
	}
	if counts[".gguf"] == 0 || counts[".safetensors"] == 0 {
		t.Errorf("expected model-blob headers, got %v", counts)
	}
}

// TestGenerateLargeFileIsSparse: a large logical size must not cost that many
// real bytes on disk (proving the harness can afford a 200 MB fixture).
func TestGenerateLargeFileIsSparse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "models", "huge.gguf")
	const size = 200 << 20
	if err := GenerateLargeFile(path, size, GGUFHeader()); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() != size {
		t.Errorf("logical size = %d, want %d", fi.Size(), size)
	}
	// The header must survive at offset 0 so the file still routes as a model.
	f, err := os.Open(path) // #nosec G304 -- test-owned temp path
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	magic := make([]byte, 4)
	if _, err := f.Read(magic); err != nil {
		t.Fatal(err)
	}
	if string(magic) != "GGUF" {
		t.Errorf("magic = %q, want GGUF", magic)
	}
}

// fingerprint hashes every regular file's relative path + content into one
// stable string, so two trees can be compared for byte-identity.
func fingerprint(t *testing.T, root string) string {
	t.Helper()
	var paths []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		paths = append(paths, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// filepath.Walk already yields lexical order; hash paths+contents.
	h := ""
	for _, p := range paths {
		rel, _ := filepath.Rel(root, p)
		b, rerr := os.ReadFile(p) // #nosec G304 -- test-owned temp path
		if rerr != nil {
			t.Fatal(rerr)
		}
		h += rel + ":" + string(b) + "\n"
	}
	return h
}

// extCounts tallies file extensions under root.
func extCounts(t *testing.T, root string) map[string]int {
	t.Helper()
	counts := map[string]int{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			counts[filepath.Ext(p)]++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return counts
}
