// Command gen materializes the tiny, handcrafted BINARY model fixtures used
// by the internal/e2e golden suite. It is parked under testdata/ so the Go
// toolchain and golangci-lint ignore it; regenerate the binaries by explicit
// path:
//
//	go run ./internal/e2e/testdata/gen.go
//
// Every file here is header-only or deliberately corrupt: NONE contain real
// model weights. The valid GGUF is a bare, well-formed header (magic +
// version 3 + zero counts) that the modelfile/gguf detector accepts; the
// broken.pt / corrupt.safetensors / garbage.onnx files exercise per-file
// degradation (invariant P6) — a truncated PyTorch zip surfaces as an
// attributed Unknown, while the malformed safetensors/onnx degrade silently
// to no finding. Regenerated output is byte-stable, so goldens stay portable.
package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("cannot locate gen.go")
	}
	root := filepath.Dir(self) // .../internal/e2e/testdata
	fixtures := filepath.Join(root, "fixtures")

	// Valid GGUF header shared by the langchain-rag and chaos fixtures.
	write(filepath.Join(fixtures, "python-langchain-rag", "models", "tiny.gguf"), validGGUF())
	write(filepath.Join(fixtures, "malformed-models", "models", "tiny.gguf"), validGGUF())

	// Inert risky checkpoint: a bare os.system GLOBAL reference (no reduce, no
	// call) — the static pickle walk flags AIROM-RISK-PICKLE-IMPORT, proving the
	// end-to-end risk overlay through the assembler and every writer.
	write(filepath.Join(fixtures, "python-langchain-rag", "models", "poisoned.pt"), riskyTorchPickle())

	// Deliberately corrupt weight files (chaos fixture).
	write(filepath.Join(fixtures, "malformed-models", "models", "broken.pt"), brokenTorchZip())
	write(filepath.Join(fixtures, "malformed-models", "models", "corrupt.safetensors"), corruptSafetensors())
	write(filepath.Join(fixtures, "malformed-models", "models", "garbage.onnx"), garbageONNX())
}

// validGGUF is a well-formed, header-only GGUF: the four-byte magic, version
// 3, then zero tensors and zero metadata pairs. parseGGUF accepts it and the
// detector emits a local-model-file finding (format "gguf").
func validGGUF() []byte {
	var b bytes.Buffer
	b.WriteString("GGUF")
	putU32(&b, 3) // version 3
	putU64(&b, 0) // tensor_count
	putU64(&b, 0) // metadata_kv_count
	return b.Bytes()
}

// brokenTorchZip opens with the PyTorch zip magic ("PK\x03\x04") so the torch
// detector's selector matches, then trails garbage. archive/zip fails to find
// a central directory, so DetectFile returns an error -> the pipeline records
// an attributed Unknown for modelfilex/torch (P6), never a crash.
func brokenTorchZip() []byte {
	var b bytes.Buffer
	b.Write([]byte{'P', 'K', 0x03, 0x04})
	b.WriteString("this is not a real zip central directory, just noise\x00\x01\x02")
	return b.Bytes()
}

// riskyTorchPickle is a minimal, INERT pickle stream: proto 2, a GLOBAL
// opcode naming os.system, then STOP. It only *references* the callable (no
// REDUCE, no arguments), so even a real unpickle would merely bind the name —
// nothing executes. The static walk still flags it, which is the point: this
// exercises the AIROM-RISK-PICKLE-IMPORT path end-to-end without shipping a
// working exploit.
func riskyTorchPickle() []byte {
	var b bytes.Buffer
	b.Write([]byte{0x80, 0x02}) // PROTO 2
	b.WriteString("cos\nsystem\n") // GLOBAL "os" "system"
	b.WriteByte('.')               // STOP
	return b.Bytes()
}

// corruptSafetensors declares an 8-byte little-endian header length far larger
// than the file, so the parser's bounds check fails and the detector degrades
// to no finding and no error (silent, honest degradation).
func corruptSafetensors() []byte {
	var b bytes.Buffer
	putU64(&b, 1<<40) // claimed header length (1 TiB) >> actual bytes
	b.WriteString(`{"weight":{"dtype":"F16"`)
	return b.Bytes()
}

// garbageONNX is random-looking bytes that do not decode to a confirmed ONNX
// ModelProto (no producer_name, no ir_version), so the sniff declines it: no
// finding, no error.
func garbageONNX() []byte {
	return []byte{
		0xde, 0xad, 0xbe, 0xef, 0xff, 0x00, 0x11, 0x22,
		0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa,
		0xbb, 0xcc, 0xdd, 0xee, 0xf0, 0x0d, 0xca, 0xfe,
	}
}

func putU32(b *bytes.Buffer, v uint32) {
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], v)
	b.Write(tmp[:])
}

func putU64(b *bytes.Buffer, v uint64) {
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], v)
	b.Write(tmp[:])
}

func write(path string, data []byte) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		log.Fatalf("write %s: %v", path, err)
	}
	log.Printf("wrote %d bytes to %s", len(data), path)
}
