// Package perf is AIROM's performance-regression harness (ARCHITECTURE.md
// invariant P2: peak memory is a function of CONFIGURATION, never of input
// size). It provides a deterministic synthetic-tree generator plus the tests
// and benchmarks that pin the scanner's memory ceiling, throughput floor, and
// determinism-under-parallelism to that invariant.
//
// The generator is the reusable piece; the assertions live in the test files.
// Nothing here is imported by the product — this package is test tooling that
// happens to expose a small, documented API so benchmarks and future
// regression suites can share one tree builder.
package perf

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
)

// TreeOpts controls one synthetic tree. Identical opts (same Seed included)
// produce a byte-identical tree, so every test built on it is reproducible.
type TreeOpts struct {
	// NumFiles is the number of regular (non-blob) files to create. Binary
	// blobs (NumBinaryBlobs) and any file added via GenerateLargeFile are
	// counted separately.
	NumFiles int

	// Seed seeds the PRNG. The generator uses rand.New(rand.NewSource(Seed))
	// exclusively — never the global/default source — so a run is a pure
	// function of (Seed, opts) and can never be perturbed by ambient state.
	Seed int64

	// FractionAIRelevant is the share of regular files (in [0,1]) that carry
	// real AI signals (a recognized AI SDK import + model literal, or an
	// `import openai` / `model="gpt-4.1"` block). The remainder is inert code
	// and prose the scanner walks but never turns into a component.
	FractionAIRelevant float64

	// MaxDepth and DirFanout shape the directory tree: DirFanout subdirs per
	// level, MaxDepth levels deep. Files are scattered across every directory.
	MaxDepth  int
	DirFanout int

	// NumBinaryBlobs scatters small binary files. A third carry a real GGUF
	// header, a third a safetensors-ish header, and the rest are inert random
	// bytes with no recognized magic (§P2: a blob costs a bounded header read).
	NumBinaryBlobs int

	// LargeFileFraction is the share of regular text files (in [0,1]) padded
	// to LargeFileBytes — the "a few large" tail of the size distribution.
	LargeFileFraction float64

	// LargeFileBytes is the size of a "large" text file. Zero defaults to 64 KiB.
	LargeFileBytes int
}

// GenResult reports what GenerateTree wrote, for assertions and for a
// benchmark's b.SetBytes (throughput is reported against TotalBytes).
type GenResult struct {
	NumFiles   int      // regular files written
	NumAIFiles int      // regular files carrying AI signals
	NumBinary  int      // binary blobs written
	NumDirs    int      // directories created (including the root)
	TotalBytes int64    // sum of logical bytes written (sparse tails excluded)
	ModelNames []string // distinct model literals planted, sorted
}

const defaultLargeFileBytes = 64 << 10

// maxDirs caps directory explosion so a careless (fanout, depth) pair cannot
// create millions of directories before the file loop runs.
const maxDirs = 4096

// GenerateTree writes a deterministic synthetic source tree under dir and
// returns what it produced. dir must already exist (callers pass t.TempDir()).
func GenerateTree(dir string, opts TreeOpts) (GenResult, error) {
	if opts.NumFiles < 0 {
		return GenResult{}, fmt.Errorf("perf: NumFiles must be >= 0, got %d", opts.NumFiles)
	}
	large := opts.LargeFileBytes
	if large <= 0 {
		large = defaultLargeFileBytes
	}

	// #nosec G404 -- deterministic synthetic test data; a seeded, reproducible
	// stream is the point. This RNG guards no secret and gates no security
	// decision, so a cryptographic source would only cost reproducibility.
	rng := rand.New(rand.NewSource(opts.Seed))

	dirs, err := buildDirs(dir, opts.MaxDepth, opts.DirFanout)
	if err != nil {
		return GenResult{}, err
	}

	g := &genState{
		rng:    rng,
		dirs:   dirs,
		models: map[string]struct{}{},
	}

	for i := 0; i < opts.NumFiles; i++ {
		isAI := rng.Float64() < opts.FractionAIRelevant
		isLarge := rng.Float64() < opts.LargeFileFraction
		if err := g.writeRegular(isAI, isLarge, large); err != nil {
			return GenResult{}, err
		}
	}
	for i := 0; i < opts.NumBinaryBlobs; i++ {
		if err := g.writeBlob(i); err != nil {
			return GenResult{}, err
		}
	}

	models := make([]string, 0, len(g.models))
	for m := range g.models {
		models = append(models, m)
	}
	sort.Strings(models)

	return GenResult{
		NumFiles:   opts.NumFiles,
		NumAIFiles: g.aiFiles,
		NumBinary:  opts.NumBinaryBlobs,
		NumDirs:    len(dirs),
		TotalBytes: g.totalBytes,
		ModelNames: models,
	}, nil
}

// GenerateLargeFile writes a file of logical size sizeBytes cheaply: it writes
// header (if any) then truncates to sizeBytes, so the tail is a sparse run of
// zeros that costs no disk and no memory. It exists to prove invariant P2 — a
// multi-hundred-MiB file must cost only a bounded, header/cap-limited read, so
// the harness needs to materialize one without actually spending that many
// bytes. When header carries a real model magic (e.g. gguf), the file routes
// to a content detector yet the read still stops at MaxFileSize; when header is
// nil the file is inert and only its 32 KiB header sample is ever touched.
func GenerateLargeFile(path string, sizeBytes int64, header []byte) error {
	if sizeBytes < int64(len(header)) {
		return fmt.Errorf("perf: sizeBytes %d smaller than header %d", sizeBytes, len(header))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600) // #nosec G304 -- caller-controlled test path under a temp dir
	if err != nil {
		return err
	}
	if len(header) > 0 {
		if _, werr := f.Write(header); werr != nil {
			_ = f.Close()
			return werr
		}
	}
	if terr := f.Truncate(sizeBytes); terr != nil {
		_ = f.Close()
		return terr
	}
	return f.Close()
}

// GGUFHeader returns a minimal, valid GGUF header (magic + version 3, zero
// tensors, zero metadata KV pairs). The modelfile detector parses it into a
// local-model-file component without error, so a large GGUF-headed file is a
// genuine recognized model whose content read is still capped at MaxFileSize.
func GGUFHeader() []byte {
	b := make([]byte, 0, 20)
	b = append(b, 'G', 'G', 'U', 'F')
	b = binary.LittleEndian.AppendUint32(b, 3) // version
	b = binary.LittleEndian.AppendUint64(b, 0) // tensor count
	b = binary.LittleEndian.AppendUint64(b, 0) // metadata kv count
	return b
}

// safetensorsHeader returns a minimal safetensors header: an 8-byte
// little-endian JSON length followed by an empty-metadata object.
func safetensorsHeader() []byte {
	body := []byte(`{"__metadata__":{}}`)
	b := binary.LittleEndian.AppendUint64(nil, uint64(len(body)))
	return append(b, body...)
}

// genState carries the single RNG and running counters through one tree build.
type genState struct {
	rng        *rand.Rand
	dirs       []string
	n          int // monotonic file index → unique basenames tree-wide
	aiFiles    int
	totalBytes int64
	models     map[string]struct{}
}

// buildDirs creates the nested directory skeleton and returns every directory
// path (root first). Expansion stops at maxDirs to bound pathological fan-out.
func buildDirs(root string, maxDepth, fanout int) ([]string, error) {
	dirs := []string{root}
	if maxDepth <= 0 || fanout <= 0 {
		return dirs, nil
	}
	frontier := []string{root}
	for d := 0; d < maxDepth && len(dirs) < maxDirs; d++ {
		var next []string
		for _, parent := range frontier {
			for i := 0; i < fanout && len(dirs) < maxDirs; i++ {
				child := filepath.Join(parent, fmt.Sprintf("pkg%d", i))
				if err := os.MkdirAll(child, 0o750); err != nil {
					return nil, err
				}
				dirs = append(dirs, child)
				next = append(next, child)
			}
		}
		frontier = next
	}
	return dirs, nil
}

// pickDir returns a pseudo-random directory for the next file.
func (g *genState) pickDir() string { return g.dirs[g.rng.Intn(len(g.dirs))] }

// modelNames are the hosted-model literals planted in go-openai templates.
// Each distinct value becomes its own component, so cycling them yields a
// stable, multi-node graph for the determinism assertion.
var modelNames = []string{
	"gpt-4o", "gpt-4o-mini", "gpt-4.1", "claude-3-5-sonnet", "o3-mini",
}

// writeRegular writes one regular file: AI-relevant or inert, small or large.
func (g *genState) writeRegular(isAI, isLarge bool, largeBytes int) error {
	if isAI {
		g.aiFiles++
		return g.writeAIFile()
	}
	return g.writeInertFile(isLarge, largeBytes)
}

// writeAIFile rotates through the AI templates. Three of four produce a
// component via the built-in Go detector (library / framework / vector-db,
// plus a hosted-llm per model literal); the fourth is a Python signal file
// that exercises a real AI import without a matching built-in detector.
func (g *genState) writeAIFile() error {
	idx := g.aiFiles - 1
	switch idx % 4 {
	case 0:
		model := modelNames[idx%len(modelNames)]
		g.models[model] = struct{}{}
		return g.write(".go", fmt.Sprintf(tmplGoOpenAI, g.n, model))
	case 1:
		return g.write(".go", fmt.Sprintf(tmplGoLangchain, g.n))
	case 2:
		return g.write(".go", fmt.Sprintf(tmplGoPinecone, g.n))
	default:
		return g.write(".py", fmt.Sprintf(tmplPyOpenAI, g.n, "gpt-4.1"))
	}
}

// writeInertFile writes code or prose with no AI signal. Extensions are chosen
// to avoid every built-in detector (no .csv/.jsonl datasets, no manifest
// basenames, no Dockerfiles) so inert files never become components.
func (g *genState) writeInertFile(isLarge bool, largeBytes int) error {
	switch g.n % 3 {
	case 0:
		body := fmt.Sprintf(tmplGoInert, g.n, g.n)
		if isLarge {
			body = padTo(body, largeBytes, "// filler line to grow the file\n")
		}
		return g.write(".go", body)
	case 1:
		body := fmt.Sprintf("# note %d\n\nPlain prose with no AI content whatsoever.\n", g.n)
		if isLarge {
			body = padTo(body, largeBytes, "More inert prose to grow the file.\n")
		}
		return g.write(".md", body)
	default:
		body := fmt.Sprintf("{\"id\": %d, \"name\": \"item-%d\", \"tags\": []}\n", g.n, g.n)
		if isLarge {
			body = padTo(body, largeBytes, "\n")
		}
		return g.write(".txt", body)
	}
}

// writeBlob writes one small binary blob. i selects the header flavor so the
// mix (GGUF / safetensors / inert) is deterministic.
func (g *genState) writeBlob(i int) error {
	switch i % 3 {
	case 0:
		return g.writeBytes(".gguf", GGUFHeader())
	case 1:
		return g.writeBytes(".safetensors", safetensorsHeader())
	default:
		buf := make([]byte, 512)
		for j := range buf {
			buf[j] = byte(g.rng.Intn(256)) // #nosec G115 -- Intn(256) is always in [0,255] and fits a byte exactly
		}
		buf[0], buf[1] = 0xDE, 0xAD // deliberately not a recognized magic
		return g.writeBytes(".dat", buf)
	}
}

// write writes a text file, advancing the basename counter and byte total.
func (g *genState) write(ext, content string) error {
	return g.writeBytes(ext, []byte(content))
}

// writeBytes writes raw bytes to the next uniquely-named file in a random dir.
func (g *genState) writeBytes(ext string, content []byte) error {
	name := fmt.Sprintf("f%06d%s", g.n, ext)
	g.n++
	path := filepath.Join(g.pickDir(), name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return err
	}
	g.totalBytes += int64(len(content))
	return nil
}

// padTo grows s to at least n bytes by repeating filler.
func padTo(s string, n int, filler string) string {
	if len(s) >= n {
		return s
	}
	buf := make([]byte, 0, n+len(filler))
	buf = append(buf, s...)
	for len(buf) < n {
		buf = append(buf, filler...)
	}
	return string(buf)
}

// ── Content templates (all Go templates must parse; go/parser tolerates
// unused imports, which is why the bodies stay minimal). ────────────────────

const tmplGoOpenAI = `package aigen

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
)

// run%d issues one chat completion against a hosted model.
func run%[1]d(ctx context.Context, c *openai.Client) error {
	req := openai.ChatCompletionRequest{Model: %q}
	_, err := c.CreateChatCompletion(ctx, req)
	return err
}
`

const tmplGoLangchain = `package aigen

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

// chain%d is a trivial langchaingo entry point.
func chain%[1]d(ctx context.Context) (llms.MessageContent, error) {
	_ = ctx
	return llms.MessageContent{}, nil
}
`

const tmplGoPinecone = `package aigen

import (
	"context"

	"github.com/pinecone-io/go-pinecone/pinecone"
)

// index%d references the pinecone vector-db client.
func index%[1]d(ctx context.Context) (*pinecone.Client, error) {
	_ = ctx
	return nil, nil
}
`

const tmplPyOpenAI = `import openai


def call_%d():
    resp = openai.chat.completions.create(model=%q, temperature=0.2)
    return resp
`

const tmplGoInert = `package inert

// helper%d doubles its input. No AI content.
func helper%d(x int) int {
	return x * 2
}
`
