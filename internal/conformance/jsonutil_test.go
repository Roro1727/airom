package conformance

import (
	"encoding/json"
	"os"
	"testing"
)

// mustReadFile reads a file, failing the test on error.
func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path) // #nosec G304 -- test-controlled path into the module cache
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}

// mustJSONMap decodes bytes into a generic JSON object, failing the test on any
// error. Numbers land as float64, objects as map[string]any, arrays as []any —
// the shapes the schema validator and mapping asserts walk.
func mustJSONMap(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("decode JSON object: %v", err)
	}
	return m
}

// mustJSONAny decodes bytes into an arbitrary JSON value.
func mustJSONAny(t *testing.T, b []byte) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	return v
}

// obj asserts v is a JSON object.
func obj(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected JSON object, got %T", v)
	}
	return m
}

// arr asserts v is a JSON array.
func arr(t *testing.T, v any) []any {
	t.Helper()
	a, ok := v.([]any)
	if !ok {
		t.Fatalf("expected JSON array, got %T", v)
	}
	return a
}

// str asserts v is a JSON string.
func str(t *testing.T, v any) string {
	t.Helper()
	s, ok := v.(string)
	if !ok {
		t.Fatalf("expected JSON string, got %T (%v)", v, v)
	}
	return s
}
