package infra

import (
	"sort"
	"strings"

	"github.com/airomhq/airom/pkg/airom"
	"github.com/airomhq/airom/pkg/airom/detect"
)

// imageSignatures maps a base-image substring to the serving tool it names.
// Checked in order; the first substring the reference contains wins.
var imageSignatures = []struct {
	substr string
	tool   string
}{
	{"ollama/ollama", "ollama"},
	{"vllm/vllm-openai", "vllm"},
	{"vllm/vllm", "vllm"},
	{"huggingface/text-generation-inference", "tgi"},
	{"nvidia/tritonserver", "triton"},
	{"rayproject/ray", "ray"},
}

// aiEnvKeys are environment variables that signal AI serving infrastructure.
var aiEnvKeys = []string{"OLLAMA_HOST", "MODEL_ID", "HUGGING_FACE_HUB_TOKEN"}

// modelIDEnvKeys name environment variables whose VALUE is a model identifier.
// MODEL_ID is the curated cross-source AI signal — the k8s source already
// treats it as one — so a value assigned here names the model a container is
// configured to serve, and we surface it as a component: the config-file
// analog of a model="gpt-4o" literal in code. Kept deliberately narrow;
// broader names like MODEL or MODEL_NAME are common non-model config and would
// trade this signal for noise.
var modelIDEnvKeys = map[string]bool{"MODEL_ID": true}

// envModelID returns the model identifier assigned to a model-id env key on a
// line, if any. It accepts the shapes the infra detectors encounter — the
// caller strips any Dockerfile `ENV` keyword and compose `- ` list dash first:
//
//	MODEL_ID: gpt-4o       (compose mapping)
//	MODEL_ID=gpt-4o        (compose list item / Dockerfile ENV KEY=VAL)
//	"MODEL_ID=gpt-4o"      (quoted compose list item — the common idiom)
//	MODEL_ID gpt-4o        (Dockerfile ENV KEY VAL)
//	MODEL_ID: gpt-4o # ... (trailing YAML comment)
//
// Weights-file paths and unresolved ${interpolations} yield ("", false): a
// hosted-model claim would misname those.
func envModelID(assign string) (string, bool) {
	// Trim quotes off the WHOLE entry first: compose writes list items as
	// `- "MODEL_ID=gpt-4o"`, and an unstripped leading quote would leave the
	// key as `"MODEL_ID` and miss the exact-match lookup.
	key, val, ok := cutEnvAssign(strings.Trim(strings.TrimSpace(assign), `"'`))
	if !ok || !modelIDEnvKeys[key] {
		return "", false
	}
	val = strings.Trim(strings.TrimSpace(val), `"'`)
	// A model id has no internal whitespace, so the first token is the value:
	// this drops a trailing `# comment` and any second VAR=v on an ENV line.
	if i := strings.IndexAny(val, " \t"); i >= 0 {
		val = val[:i]
	}
	if !plausibleModelID(val) {
		return "", false
	}
	return val, true
}

// cutEnvAssign splits "KEY=VAL", "KEY: VAL", or "KEY VAL" on the earliest of
// '=', ':', or whitespace, so a value's own ':' (e.g. gpt-4o:latest) survives.
func cutEnvAssign(s string) (key, val string, ok bool) {
	sep := strings.IndexAny(s, "=: \t")
	if sep <= 0 {
		return "", "", false
	}
	return s[:sep], s[sep+1:], true
}

// plausibleModelID rejects values a hosted-model claim would misname: empty,
// internally-spaced, unresolved interpolations, and filesystem paths or
// weights files (which are local-model-file territory, not hosted).
func plausibleModelID(v string) bool {
	if v == "" || strings.ContainsAny(v, " \t") {
		return false
	}
	if strings.Contains(v, "${") || strings.Contains(v, "$(") {
		return false
	}
	if strings.HasPrefix(v, "/") || strings.HasPrefix(v, ".") || strings.HasPrefix(v, "~") {
		return false
	}
	switch {
	case strings.HasSuffix(v, ".gguf"), strings.HasSuffix(v, ".bin"),
		strings.HasSuffix(v, ".safetensors"), strings.HasSuffix(v, ".pt"),
		strings.HasSuffix(v, ".onnx"):
		return false
	}
	return true
}

// modelEnvFinding renders a MODEL_ID env assignment as a hosted-model claim.
// Provider is left unknown: a bare config string does not name a vendor, and
// the assembler will fold this into any code/manifest model of the same name.
func modelEnvFinding(value string, line int) detect.Finding {
	return detect.Finding{
		Claim: detect.ComponentClaim{Kind: airom.KindHostedLLM, Name: value},
		Occurrence: airom.Occurrence{
			Location:   airom.Location{Line: line},
			Method:     airom.MethodConfig,
			Confidence: 0.6,
		},
	}
}

// matchImage returns the serving tool named by a base-image reference.
func matchImage(ref string) (string, bool) {
	for _, sig := range imageSignatures {
		if strings.Contains(ref, sig.substr) {
			return sig.tool, true
		}
	}
	return "", false
}

// hit is one recognized serving tool with the endpoint and env keys scoped
// to it (attributed by proximity during the line scan).
type hit struct {
	tool     string
	line     int
	conf     airom.Confidence
	endpoint string
	envKeys  map[string]bool
}

// newHit starts a hit for a tool at a 1-based line.
func newHit(tool string, line int, conf airom.Confidence) *hit {
	return &hit{tool: tool, line: line, conf: conf, envKeys: map[string]bool{}}
}

// addEnv records every AI env key mentioned on a line.
func (h *hit) addEnv(line string) {
	for _, k := range aiEnvKeys {
		if strings.Contains(line, k) {
			h.envKeys[k] = true
		}
	}
}

// finding renders the hit as a detector finding.
func (h *hit) finding() detect.Finding {
	claim := detect.ComponentClaim{Kind: airom.KindInfra, Name: h.tool}
	if h.endpoint != "" {
		claim.Infra = &detect.InfraClaim{Endpoint: h.endpoint}
	}
	var fields map[string]string
	if len(h.envKeys) > 0 {
		keys := make([]string, 0, len(h.envKeys))
		for k := range h.envKeys {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fields = map[string]string{"env": strings.Join(keys, ",")}
	}
	return detect.Finding{
		Claim: claim,
		Occurrence: airom.Occurrence{
			Location:   airom.Location{Line: h.line},
			Method:     airom.MethodConfig,
			Confidence: h.conf,
			Fields:     fields,
		},
	}
}

// findings renders a list of hits in scan order.
func findings(hits []*hit) []detect.Finding {
	if len(hits) == 0 {
		return nil
	}
	out := make([]detect.Finding, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.finding())
	}
	return out
}

// firstPort extracts the leading port number from a whitespace-separated
// EXPOSE argument list (e.g. "8000/tcp 9000" → "8000").
func firstPort(arg string) string {
	for _, tok := range strings.Fields(arg) {
		if p := digitsBefore(tok, '/'); p != "" {
			return p
		}
	}
	return ""
}

// digitsBefore returns tok up to sep if that prefix is all digits.
func digitsBefore(tok string, sep byte) string {
	if i := strings.IndexByte(tok, sep); i >= 0 {
		tok = tok[:i]
	}
	if !allDigits(tok) {
		return ""
	}
	return tok
}

// allDigits reports whether s is a non-empty run of ASCII digits.
func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}
