package perf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"testing"
	"time"

	"github.com/airomhq/airom/internal/app"
	"github.com/airomhq/airom/internal/assemble"
	"github.com/airomhq/airom/pkg/airom"
	"github.com/airomhq/airom/pkg/airom/detect"
)

// heavyEnv opts into the heavy trees (10x files + a 200 MB model file). Kept
// off by default so `go test ./internal/perf/` stays fast in ordinary CI.
const heavyEnv = "AIROM_PERF_HEAVY"

// swapEmbeddedRules drops the built-in rule packs for the duration of a test
// or benchmark and returns a restore func. We isolate from the embedded packs
// so timing and memory are a pure function of the built-in Go/model/manifest
// detectors and the synthetic tree — the rule engine's regex compilation and
// per-file matching are real work, but their cost varies with the shipped rule
// corpus, which is exactly the ambient variability a perf gate must exclude.
// The built-in detectors alone already produce a rich component graph
// (library, framework, vector-db, hosted-llm, local-model-file), so the
// harness loses no coverage by turning the packs off.
func swapEmbeddedRules() (restore func()) {
	orig := app.EmbeddedRules
	app.EmbeddedRules = nil
	return func() { app.EmbeddedRules = orig }
}

// scanConfig builds a fixed-configuration fs scan. Parallel is pinned (not
// GOMAXPROCS) so the config-bound memory ceiling is identical machine to
// machine and scan to scan — the whole point of the ratio assertion.
func scanConfig(dir string) *app.Config {
	return &app.Config{Source: app.SourceFS, Target: dir, Parallel: 4}
}

// TestDeterminismUnderLoad is the coarse P7 check at scale: the same tree,
// scanned single-threaded and with eight workers, must yield the identical set
// of component IDs. It complements the byte-identical e2e golden another
// package owns; here we only compare sorted ID lists, but at a few-hundred-file
// scale with real concurrency. This test also doubles as the -race workload —
// it drives the generator and two full scans, which is where the race detector
// earns its keep, so it deliberately does NOT skip under -race.
func TestDeterminismUnderLoad(t *testing.T) {
	defer swapEmbeddedRules()()

	dir := t.TempDir()
	res, err := GenerateTree(dir, TreeOpts{
		NumFiles:           300,
		Seed:               7,
		FractionAIRelevant: 0.2,
		MaxDepth:           3,
		DirFanout:          3,
		NumBinaryBlobs:     9,
		LargeFileFraction:  0.03,
	})
	if err != nil {
		t.Fatal(err)
	}

	ids1 := scanComponentIDs(t, dir, 1)
	ids8 := scanComponentIDs(t, dir, 8)

	if len(ids1) == 0 {
		t.Fatalf("no components produced (AI files=%d) — determinism check is vacuous", res.NumAIFiles)
	}
	if !equalStrings(ids1, ids8) {
		t.Errorf("component sets diverge by parallelism (P7 violation)\n P=1: %v\n P=8: %v", ids1, ids8)
	}
}

// scanComponentIDs scans dir at the given parallelism and returns the sorted
// component IDs (the normalization: drop everything volatile, keep identity).
func scanComponentIDs(t *testing.T, dir string, parallel int) []string {
	t.Helper()
	cfg := &app.Config{Source: app.SourceFS, Target: dir, Parallel: parallel}
	inv, err := app.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("scan (parallel=%d): %v", parallel, err)
	}
	ids := make([]string, 0, len(inv.Components))
	for _, c := range inv.Components {
		ids = append(ids, string(c.ID))
	}
	sort.Strings(ids)
	return ids
}

// TestPeakMemoryConfigBound is the core P2 assertion in its fast form: a 5x
// larger tree (same number of AI/discovery files, five times the inert files,
// plus a 32 MB model file exceeding MaxFileSize) must NOT use 5x the peak
// heap. Peak is a function of configuration — worker count times the bounded
// per-file buffers, plus a graph that grows with discoveries, not with raw
// input size. The heavy 10x + 200 MB variant lives in TestHeavyPeakMemory.
func TestPeakMemoryConfigBound(t *testing.T) {
	if raceEnabled {
		t.Skip("RSS measurement is meaningless under -race (inflated, reshuffled heap accounting)")
	}
	defer swapEmbeddedRules()()

	// Equal AI-file counts (≈32 each): only the inert-file count differs, so
	// any peak growth would have to come from raw file count — precisely what
	// P2 forbids.
	small := TreeOpts{NumFiles: 400, Seed: 11, FractionAIRelevant: 0.08, MaxDepth: 3, DirFanout: 3, NumBinaryBlobs: 8, LargeFileFraction: 0.02}
	large := TreeOpts{NumFiles: 2000, Seed: 12, FractionAIRelevant: 0.016, MaxDepth: 4, DirFanout: 4, NumBinaryBlobs: 12, LargeFileFraction: 0.004}

	assertPeakBounded(t, small, large, 5, 4<<20)
}

// TestHeavyPeakMemory is the full-strength P2 assertion: 10x the files plus a
// genuine 200 MB model file that must never be fully read. Opt in with
// AIROM_PERF_HEAVY=1 (and run without -race); documented in the package report.
func TestHeavyPeakMemory(t *testing.T) {
	if raceEnabled {
		t.Skip("RSS measurement is meaningless under -race")
	}
	if os.Getenv(heavyEnv) == "" {
		t.Skipf("set %s=1 to run the heavy 10x + 200 MB memory assertion", heavyEnv)
	}
	defer swapEmbeddedRules()()

	small := TreeOpts{NumFiles: 500, Seed: 21, FractionAIRelevant: 0.08, MaxDepth: 3, DirFanout: 4, NumBinaryBlobs: 8, LargeFileFraction: 0.02}
	large := TreeOpts{NumFiles: 5000, Seed: 22, FractionAIRelevant: 0.008, MaxDepth: 4, DirFanout: 5, NumBinaryBlobs: 12, LargeFileFraction: 0.004}

	assertPeakBounded(t, small, large, 10, 4<<20)
}

// assertPeakBounded generates the two trees, plants a large model file in the
// larger one (bigModelMB), scans both under identical configuration, and
// asserts the peak-heap ratio stays far below the file-count ratio.
//
// The threshold K=3 is deliberately generous: the *expected* ratio is ≈1
// (identical configuration ⇒ identical buffer ceiling; equal discovery counts
// ⇒ equal graph), so K=3 leaves wide margin for GC scheduling jitter while
// still catching any true linear-in-file-count regression (which would show up
// as a ratio tracking the 5x/10x file ratio, not sitting near 1). The absolute
// floor guards the ratio against a tiny, noise-dominated denominator: when the
// small scan's peak delta is itself below a few MiB, "3x of noise" is not a
// regression, so we additionally require the large delta to clear the floor
// before failing.
func assertPeakBounded(t *testing.T, small, large TreeOpts, fileRatio int, floor uint64) {
	t.Helper()

	smallDir := t.TempDir()
	sres, err := GenerateTree(smallDir, small)
	if err != nil {
		t.Fatal(err)
	}

	largeDir := t.TempDir()
	lres, err := GenerateTree(largeDir, large)
	if err != nil {
		t.Fatal(err)
	}
	// A model file far larger than MaxFileSize, dropped into the large tree.
	// It routes to the GGUF detector (real magic) yet its content read stops
	// at MaxFileSize — the "huge file costs a bounded read" invariant.
	bigModel := int64(32 << 20)
	if os.Getenv(heavyEnv) != "" {
		bigModel = 200 << 20
	}
	if err := GenerateLargeFile(filepath.Join(largeDir, "models", "huge.gguf"), bigModel, GGUFHeader()); err != nil {
		t.Fatal(err)
	}

	const K = 3

	smallPeak := peakHeapInuse(t, func() {
		if _, err := app.Scan(context.Background(), scanConfig(smallDir)); err != nil {
			t.Fatal(err)
		}
	})
	largePeak := peakHeapInuse(t, func() {
		if _, err := app.Scan(context.Background(), scanConfig(largeDir)); err != nil {
			t.Fatal(err)
		}
	})

	t.Logf("small tree: files=%d aifiles=%d peakHeapInuseDelta=%s",
		sres.NumFiles, sres.NumAIFiles, humanBytes(smallPeak))
	t.Logf("large tree: files=%d aifiles=%d bigModel=%s peakHeapInuseDelta=%s",
		lres.NumFiles, lres.NumAIFiles, humanBytes(uint64(bigModel)), humanBytes(largePeak))
	t.Logf("file-count ratio=%dx  peak-memory ratio=%.2fx  (want peak ratio << file ratio, K=%d)",
		fileRatio, ratio(largePeak, smallPeak), K)

	if largePeak > K*smallPeak && largePeak > floor {
		t.Errorf("peak heap scaled with input size (P2 regression): small=%s large=%s ratio=%.2fx > %dx, and above %s floor",
			humanBytes(smallPeak), humanBytes(largePeak), ratio(largePeak, smallPeak), K, humanBytes(floor))
	}
}

// TestBigFileBoundedRead is the standalone big-file bound (P2): a 150 MB model
// file scanned under the default 1 MiB MaxFileSize must (a) complete quickly
// and (b) contribute almost none of its bytes to Stats.ContentBytes — proof
// the file was never fully read into memory. Behind -short so the default
// -short run stays trivial; the sparse fixture keeps it fast otherwise.
func TestBigFileBoundedRead(t *testing.T) {
	if testing.Short() {
		t.Skip("big-file bound test skipped in -short")
	}
	defer swapEmbeddedRules()()

	dir := t.TempDir()
	// A handful of small real files so the scan does normal work too.
	if _, err := GenerateTree(dir, TreeOpts{NumFiles: 30, Seed: 5, FractionAIRelevant: 0.2, MaxDepth: 2, DirFanout: 2, NumBinaryBlobs: 4}); err != nil {
		t.Fatal(err)
	}
	const bigSize = int64(150 << 20)
	if err := GenerateLargeFile(filepath.Join(dir, "weights", "model.gguf"), bigSize, GGUFHeader()); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	inv, err := app.Scan(context.Background(), scanConfig(dir))
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("scanned a %s tree in %s; ContentBytes=%s HeaderBytes=%s",
		humanBytes(uint64(bigSize)), elapsed, humanBytes(uint64(inv.Stats.ContentBytes)), humanBytes(uint64(inv.Stats.HeaderBytes)))

	// (a) A 150 MB file must not dominate the runtime. The bound is generous
	// (any full read would be far slower and, worse, allocate 150 MB).
	if elapsed > 10*time.Second {
		t.Errorf("scan took %s for a mostly-sparse tree — the big file was likely read in full", elapsed)
	}
	// (b) The size cap held: total content read is far below the file size and
	// within a small multiple of MaxFileSize (one capped read for the model,
	// plus the tiny real files).
	if inv.Stats.ContentBytes >= bigSize/16 {
		t.Errorf("ContentBytes=%d is not << file size %d — the size cap did not hold",
			inv.Stats.ContentBytes, bigSize)
	}
	if inv.Stats.ContentBytes > 8*app.DefaultMaxFileSize {
		t.Errorf("ContentBytes=%d exceeds 8x MaxFileSize (%d) — content read is not bounded by configuration",
			inv.Stats.ContentBytes, app.DefaultMaxFileSize)
	}
}

// ── Benchmarks (tracked, not asserted) ──────────────────────────────────────

// BenchmarkScanTree reports ns/op and MB/s over a fixed generated tree.
// Maintainers run it to watch the throughput floor:
//
//	go test -run '^$' -bench BenchmarkScanTree ./internal/perf/
func BenchmarkScanTree(b *testing.B) {
	defer swapEmbeddedRules()()

	dir := b.TempDir()
	res, err := GenerateTree(dir, TreeOpts{
		NumFiles:           1500,
		Seed:               42,
		FractionAIRelevant: 0.05,
		MaxDepth:           3,
		DirFanout:          4,
		NumBinaryBlobs:     20,
		LargeFileFraction:  0.02,
	})
	if err != nil {
		b.Fatal(err)
	}
	cfg := &app.Config{Source: app.SourceFS, Target: dir, Parallel: runtime.GOMAXPROCS(0)}

	b.SetBytes(res.TotalBytes)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := app.Scan(context.Background(), cfg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAssemble isolates the assembler: N findings across a handful of
// repeated component identities, exercising the dedup/merge/ID-minting path
// without any I/O.
func BenchmarkAssemble(b *testing.B) {
	findings := syntheticFindings(2000)
	opts := assemble.Options{
		Tool:      airom.ToolInfo{Name: "airom", Version: "bench"},
		Source:    airom.SourceInfo{Kind: "dir", Target: "/bench"},
		Lifecycle: "pre-build",
		Serial:    "urn:uuid:00000000-0000-4000-8000-000000000000",
		Timestamp: time.Unix(0, 0).UTC(),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if inv := assemble.Build(findings, nil, airom.ScanStats{}, opts); inv == nil {
			b.Fatal("nil inventory")
		}
	}
}

// syntheticFindings builds n findings whose identities repeat across a small
// set, so the assembler must merge many occurrences per component.
func syntheticFindings(n int) []detect.Finding {
	out := make([]detect.Finding, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, detect.Finding{
			Claim: detect.ComponentClaim{
				Kind:     airom.KindHostedLLM,
				Name:     modelNames[i%len(modelNames)],
				Provider: "openai",
			},
			Occurrence: airom.Occurrence{
				Location:   airom.Location{Path: fmt.Sprintf("src/pkg%d/f%d.go", i%16, i), Line: i%200 + 1},
				DetectorID: "bench/synthetic",
				Method:     airom.MethodSourceCode,
				Confidence: 0.8,
			},
		})
	}
	return out
}

// ── Measurement + small helpers ─────────────────────────────────────────────

// peakReps is how many times each measurement is repeated. Every error in this
// measurement is one-sided: both an undersampled peak and an unsettled baseline
// can only make the observed delta too SMALL, never too large. So the maximum
// across repetitions converges upward on the true peak, and three cheap reps
// (a scan is tens of milliseconds) buy far more stability than loosening the
// assertion would. Raising K or the floor instead would hide the very
// regression this gate exists to catch.
const peakReps = 3

// peakHeapInuse returns the largest peak-above-baseline observed across
// peakReps runs of fn. HeapInuse (bytes in in-use spans) is the closest cheap
// proxy for the live working set; a ratio of two such deltas is far more
// portable than any absolute RSS number, which is why the P2 assertion is
// expressed as a ratio.
//
// GC is tightened to 25% for the duration so the peak tracks the *live*
// working set rather than floating garbage. Floating garbage (allocated but
// not-yet-collected) scales with total allocation, i.e. with file count — the
// very thing we want to exclude from a "peak is config-bound" measurement.
// Tightening applies identically to both scans, so it never biases the ratio;
// it only stops garbage-scheduling luck from injecting a file-count signal.
func peakHeapInuse(t *testing.T, fn func()) uint64 {
	t.Helper()

	defer debug.SetGCPercent(debug.SetGCPercent(25))

	var best uint64
	for i := 0; i < peakReps; i++ {
		if d := peakHeapInuseOnce(fn); d > best {
			best = d
		}
	}
	return best
}

// peakHeapInuseOnce is a single measurement: settle the heap, sample HeapInuse
// across one run of fn, and report the peak above the settled floor.
func peakHeapInuseOnce(fn func()) uint64 {
	// Two collections, not one. runtime.GC returns once the mark phase is
	// done, but in-use spans are handed back by the *sweeper*, which runs on
	// concurrently afterward. Reading HeapInuse straight after a single GC can
	// therefore catch the heap mid-sweep, still holding the previous
	// workload's garbage — here, the generated trees and the 32 MB model file.
	// That inflates the baseline, which deflates `peak - base`: if the scan
	// fits inside the not-yet-released spans, the measurement reads 0 and the
	// ratio explodes. A second GC cannot begin until the first cycle's sweep
	// has finished, so it pins HeapInuse to a genuine idle floor.
	runtime.GC()
	runtime.GC()

	var base runtime.MemStats
	runtime.ReadMemStats(&base)
	peak := base.HeapInuse

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		tk := time.NewTicker(time.Millisecond)
		defer tk.Stop()
		var m runtime.MemStats
		for {
			select {
			case <-stop:
				return
			case <-tk.C:
				runtime.ReadMemStats(&m)
				if m.HeapInuse > peak {
					peak = m.HeapInuse
				}
			}
		}
	}()

	fn()

	// Capture a final reading into a local (no shared access), then fold it in
	// only after the sampler has stopped — keeps the helper data-race-free.
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	close(stop)
	<-done

	if after.HeapInuse > peak {
		peak = after.HeapInuse
	}
	if peak < base.HeapInuse {
		return 0
	}
	return peak - base.HeapInuse
}

func ratio(a, b uint64) float64 {
	if b == 0 {
		return float64(a)
	}
	return float64(a) / float64(b)
}

func humanBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
