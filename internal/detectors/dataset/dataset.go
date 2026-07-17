package dataset

import (
	"bytes"
	"context"
	"path"
	"strings"

	"github.com/airomhq/airom/pkg/airom"
	"github.com/airomhq/airom/pkg/airom/detect"
)

// Magic prefixes for the self-describing binary dataset formats.
var (
	parquetMagic = []byte("PAR1")
	arrowMagic   = []byte("ARROW1")
)

// Dataset detects dataset files by extension plus a header-only format
// signature: CSV and JSONL by structural sniffing, Parquet and Arrow by
// magic bytes.
type Dataset struct{}

// NewDataset constructs the dataset-file detector.
func NewDataset() *Dataset { return &Dataset{} }

// ID is the stable SARIF ruleId.
func (*Dataset) ID() string { return "dataset/file" }

// Version participates in the cache key; bump on any behavior change.
func (*Dataset) Version() int { return 1 }

// Selector routes the four dataset extensions; only the header is needed.
func (*Dataset) Selector() detect.Selector {
	return detect.Selector{
		Extensions: []string{".csv", ".jsonl", ".parquet", ".arrow"},
		Need:       detect.NeedHeader,
	}
}

// DetectFile classifies the routed file from its header sample, name, and size.
//
// A structurally-valid CSV or JSONL is NOT enough: see signals.go. The file must
// corroborate the dataset claim through its fields, its name/path, or a
// self-describing columnar format — otherwise no finding is emitted at all.
func (d *Dataset) DetectFile(_ context.Context, f *detect.File) ([]detect.Finding, error) {
	format, method, conf, ok := sniff(f.Path(), f.Header())
	if !ok {
		return nil, nil
	}
	return []detect.Finding{{
		Claim: detect.ComponentClaim{
			Kind: airom.KindDataset,
			Name: f.Base(),
			Data: &detect.DataClaim{Format: format, SizeBytes: f.Ref().Size},
		},
		Occurrence: airom.Occurrence{
			Method:     method,
			Confidence: conf,
		},
	}}, nil
}

// Confidence by corroboration. Two independent signals beat one, and the
// content-derived field signal outranks the name, which a file can carry by
// coincidence.
const (
	confFields      = 0.75 // ML-shaped columns/keys: evidence about the content
	confName        = 0.65 // named/filed as a dataset
	confFormat      = 0.7  // magic-verified Parquet/Arrow
	confCorroborate = 0.85 // two or more of the above
)

// sniff decides the format, method, and confidence for a routed file, or
// reports ok=false when nothing corroborates the extension.
func sniff(p string, header []byte) (format string, method airom.DetectionMethod, conf airom.Confidence, ok bool) {
	ext := strings.ToLower(path.Ext(p))
	named := nameSignal(p)

	switch ext {
	case ".parquet", ".arrow":
		format = strings.TrimPrefix(ext, ".")
		magic := parquetMagic
		if ext == ".arrow" {
			magic = arrowMagic
		}
		if !bytes.HasPrefix(header, magic) {
			// The extension lies, or the file is truncated. Only a dataset-ish
			// name keeps it, and then only as a filename-grade claim.
			if named {
				return format, airom.MethodFilename, confName, true
			}
			return "", "", 0, false
		}
		if named {
			return format, airom.MethodBinary, confCorroborate, true
		}
		return format, airom.MethodBinary, confFormat, true

	case ".jsonl":
		// A failed parse is not a verdict, only a missing signal: JSONL of bare
		// arrays, a leading blank line, or an unfamiliar encoding all land here
		// while still being `data/train.jsonl`. Fall through to the name the way
		// the Parquet branch above does, rather than deleting the file outright.
		return textual("jsonl", fieldSignal(jsonlFields(header)), named)

	case ".csv":
		// Likewise: a semicolon-delimited export (the EU Excel locale) or a
		// single-column corpus yields no fields but is still corroborated by a
		// dataset name.
		return textual("csv", fieldSignal(csvFields(header)), named)
	}
	return "", "", 0, false
}

// textual grades a structurally-valid CSV/JSONL by its corroborating signals.
// With neither, the file is some other program's data and we say nothing.
func textual(format string, fields, named bool) (string, airom.DetectionMethod, airom.Confidence, bool) {
	switch {
	case fields && named:
		return format, airom.MethodSourceCode, confCorroborate, true
	case fields:
		// Derived from the header row, not the path: a content claim.
		return format, airom.MethodSourceCode, confFields, true
	case named:
		return format, airom.MethodFilename, confName, true
	default:
		return "", "", 0, false
	}
}

// utf8BOM is the UTF-8 byte order mark. Excel's "Save as CSV UTF-8", pandas'
// encoding="utf-8-sig" and PowerShell's Export-Csv all emit it, so it arrives on
// mainstream real-world datasets. It is not whitespace (U+FEFF is White_Space=No),
// so bytes.TrimSpace leaves it in place — where it corrupts the first column
// name into a "<BOM>prompt" no lookup matches and, worse, hides a JSON line's
// leading brace.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// firstLine returns the first line of the sample with any byte order mark and
// surrounding whitespace trimmed, without its terminator.
func firstLine(b []byte) []byte {
	b = bytes.TrimPrefix(b, utf8BOM)
	if i := bytes.IndexByte(b, '\n'); i >= 0 {
		b = b[:i]
	}
	return bytes.TrimSpace(b)
}
