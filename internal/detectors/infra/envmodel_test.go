package infra

import "testing"

// TestEnvModelID covers the MODEL_ID extraction across the compose-mapping,
// list-item, and Dockerfile-ENV shapes, plus the values a hosted-model claim
// must refuse to name. The caller strips the compose `- ` dash and Dockerfile
// `ENV` keyword, so inputs here are the post-strip assignment text.
func TestEnvModelID(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantOK  bool
		whatFor string
	}{
		{"MODEL_ID: gpt-4o", "gpt-4o", true, "compose mapping"},
		{"MODEL_ID:gpt-4o", "gpt-4o", true, "compose mapping, no space"},
		{"MODEL_ID=gpt-4o", "gpt-4o", true, "list item / ENV KEY=VAL"},
		{"MODEL_ID gpt-4o", "gpt-4o", true, "ENV KEY VAL"},
		{`MODEL_ID: "gpt-4o"`, "gpt-4o", true, "quoted value"},
		{`"MODEL_ID=gpt-4o"`, "gpt-4o", true, "quoted whole entry (common compose idiom)"},
		{`'MODEL_ID=gpt-4o'`, "gpt-4o", true, "single-quoted whole entry"},
		{"MODEL_ID: gpt-4o  # the served model", "gpt-4o", true, "trailing YAML comment"},
		{"MODEL_ID=gpt-4o OTHER=1", "gpt-4o", true, "first token wins on multi-var ENV"},
		{"MODEL_ID: gpt-4o:latest", "gpt-4o:latest", true, "value keeps its own colon"},
		{"MODEL_ID=meta-llama/Llama-3-8B", "meta-llama/Llama-3-8B", true, "HF repo id"},

		{"OLLAMA_HOST=0.0.0.0", "", false, "not a model-id key"},
		{"MODEL_NAME=gpt-4o", "", false, "adjacent key is not curated"},
		{"MODEL_ID=", "", false, "empty value"},
		{"MODEL_ID=${MODEL}", "", false, "unresolved interpolation"},
		{"MODEL_ID=$(cat model)", "", false, "command substitution"},
		{"MODEL_ID=/models/llama.gguf", "", false, "weights path"},
		{"MODEL_ID=./local", "", false, "relative path"},
		{"MODEL_ID=model.safetensors", "", false, "weights file"},
	}
	for _, c := range cases {
		got, ok := envModelID(c.in)
		if ok != c.wantOK || got != c.want {
			t.Errorf("envModelID(%q) = (%q, %v), want (%q, %v) — %s",
				c.in, got, ok, c.want, c.wantOK, c.whatFor)
		}
	}
}
