package schemas

import (
	"encoding/json"
	"testing"
)

func TestNativeV1IsValidJSON(t *testing.T) {
	var m map[string]any
	if err := json.Unmarshal(NativeV1, &m); err != nil {
		t.Fatalf("airom-v1.schema.json is not valid JSON: %v", err)
	}
	if m["$schema"] == nil || m["$id"] == nil {
		t.Error("schema missing $schema/$id")
	}
}
