package conformance

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/airomhq/airom/internal/writer"
	// Blank imports register the format writers via their init() side effects,
	// so writer.New("cyclonedx"|"sarif"|"json") resolves.
	_ "github.com/airomhq/airom/internal/writer/cdx"
	_ "github.com/airomhq/airom/internal/writer/nativejson"
	_ "github.com/airomhq/airom/internal/writer/sarifw"
	"github.com/airomhq/airom/internal/writer/writertest"
	"github.com/airomhq/airom/pkg/airom"
)

// render writes the shared fixture through the named writer and returns the
// bytes. The fixture (writertest.BuildFixture) is deterministic, so every
// render is byte-stable.
func render(t *testing.T, format string, opts writer.Options) []byte {
	t.Helper()
	return renderInv(t, format, opts, writertest.BuildFixture())
}

// renderInv renders an arbitrary inventory (used by the confidence test, which
// needs a component whose identity claim carries the 0.8738 value).
func renderInv(t *testing.T, format string, opts writer.Options, inv *airom.Inventory) []byte {
	t.Helper()
	w, err := writer.New(format, opts)
	if err != nil {
		t.Fatalf("writer.New(%q): %v", format, err)
	}
	var buf bytes.Buffer
	if err := w.Write(&buf, inv); err != nil {
		t.Fatalf("Write(%q): %v", format, err)
	}
	return buf.Bytes()
}

// ── Self-contained JSON Schema validator ────────────────────────────────────
//
// gojsonschema is not on this module's require list (it is only an indirect,
// test-scoped dependency of cyclonedx-go) and cannot be imported without
// editing go.mod, which is out of scope for this package. It also does not
// support the native schema's draft 2020-12 declaration. So the native schema
// is enforced by this small validator, which implements exactly the keyword
// subset schemas/airom-v1.schema.json uses — type (incl. ["string","null"]
// unions), required, properties, additionalProperties (bool or schema), items,
// enum, const, pattern, minimum, maximum, format:date-time, and local
// #/$defs/* $refs. Every construct in that schema is covered; it uses no
// allOf/oneOf/anyOf/if/then, so this is a faithful check, not an approximation.

type schemaValidator struct {
	root  map[string]any
	cache map[string]*regexp.Regexp
}

func newSchemaValidator(t *testing.T, schema []byte) schemaValidator {
	t.Helper()
	return schemaValidator{root: mustJSONMap(t, schema), cache: map[string]*regexp.Regexp{}}
}

// validate returns every violation as "path: message", sorted for determinism.
func (v schemaValidator) validate(doc any) []string {
	var errs []string
	v.check("", doc, v.root, &errs)
	sort.Strings(errs)
	return errs
}

func (v schemaValidator) check(path string, doc any, sch map[string]any, errs *[]string) {
	if ref, ok := sch["$ref"].(string); ok {
		target := v.resolve(ref)
		if target == nil {
			*errs = append(*errs, at(path)+": unresolved $ref "+ref)
			return
		}
		v.check(path, doc, target, errs)
		return
	}
	if enum, ok := sch["enum"].([]any); ok && !containsJSON(enum, doc) {
		*errs = append(*errs, fmt.Sprintf("%s: %s not in enum", at(path), show(doc)))
	}
	if c, ok := sch["const"]; ok && !reflect.DeepEqual(c, doc) {
		*errs = append(*errs, fmt.Sprintf("%s: %s != const %s", at(path), show(doc), show(c)))
	}
	if tp, ok := sch["type"]; ok && !typeMatches(tp, doc) {
		*errs = append(*errs, fmt.Sprintf("%s: %s is not type %v", at(path), show(doc), tp))
		return // further keyword checks assume the type held
	}
	switch node := doc.(type) {
	case map[string]any:
		v.checkObject(path, node, sch, errs)
	case []any:
		v.checkArray(path, node, sch, errs)
	case string:
		v.checkString(path, node, sch, errs)
	case float64:
		checkNumber(path, node, sch, errs)
	}
}

func (v schemaValidator) checkObject(path string, node map[string]any, sch map[string]any, errs *[]string) {
	if req, ok := sch["required"].([]any); ok {
		for _, r := range req {
			if _, present := node[r.(string)]; !present {
				*errs = append(*errs, fmt.Sprintf("%s: missing required property %q", at(path), r))
			}
		}
	}
	props, _ := sch["properties"].(map[string]any)
	for key, val := range node {
		if ps, ok := props[key]; ok {
			v.check(join(path, key), val, ps.(map[string]any), errs)
			continue
		}
		switch ap := sch["additionalProperties"].(type) {
		case bool:
			if !ap {
				*errs = append(*errs, fmt.Sprintf("%s: property %q not allowed (additionalProperties:false)", at(path), key))
			}
		case map[string]any:
			v.check(join(path, key), val, ap, errs)
		}
	}
}

func (v schemaValidator) checkArray(path string, node []any, sch map[string]any, errs *[]string) {
	items, ok := sch["items"].(map[string]any)
	if !ok {
		return
	}
	for i, el := range node {
		v.check(fmt.Sprintf("%s[%d]", path, i), el, items, errs)
	}
}

func (v schemaValidator) checkString(path, s string, sch map[string]any, errs *[]string) {
	if pat, ok := sch["pattern"].(string); ok && !v.re(pat).MatchString(s) {
		*errs = append(*errs, fmt.Sprintf("%s: %q does not match pattern %s", at(path), s, pat))
	}
	if f, ok := sch["format"].(string); ok && f == "date-time" {
		if _, err := time.Parse(time.RFC3339, s); err != nil {
			*errs = append(*errs, fmt.Sprintf("%s: %q is not an RFC3339 date-time", at(path), s))
		}
	}
}

func checkNumber(path string, n float64, sch map[string]any, errs *[]string) {
	if m, ok := sch["minimum"].(float64); ok && n < m {
		*errs = append(*errs, fmt.Sprintf("%s: %v < minimum %v", at(path), n, m))
	}
	if m, ok := sch["maximum"].(float64); ok && n > m {
		*errs = append(*errs, fmt.Sprintf("%s: %v > maximum %v", at(path), n, m))
	}
}

func (v schemaValidator) resolve(ref string) map[string]any {
	const prefix = "#/$defs/"
	if !strings.HasPrefix(ref, prefix) {
		return nil
	}
	defs, ok := v.root["$defs"].(map[string]any)
	if !ok {
		return nil
	}
	target, ok := defs[strings.TrimPrefix(ref, prefix)].(map[string]any)
	if !ok {
		return nil
	}
	return target
}

func (v schemaValidator) re(pat string) *regexp.Regexp {
	if r, ok := v.cache[pat]; ok {
		return r
	}
	r := regexp.MustCompile(pat)
	v.cache[pat] = r
	return r
}

func typeMatches(tp, doc any) bool {
	switch t := tp.(type) {
	case string:
		return oneType(t, doc)
	case []any:
		for _, name := range t {
			if s, ok := name.(string); ok && oneType(s, doc) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func oneType(name string, doc any) bool {
	switch name {
	case "object":
		_, ok := doc.(map[string]any)
		return ok
	case "array":
		_, ok := doc.([]any)
		return ok
	case "string":
		_, ok := doc.(string)
		return ok
	case "boolean":
		_, ok := doc.(bool)
		return ok
	case "number":
		_, ok := doc.(float64)
		return ok
	case "integer":
		f, ok := doc.(float64)
		return ok && math.Trunc(f) == f
	case "null":
		return doc == nil
	default:
		return false
	}
}

// ── small shared helpers ────────────────────────────────────────────────────

func at(path string) string {
	if path == "" {
		return "(root)"
	}
	return path
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func containsJSON(set []any, v any) bool {
	for _, e := range set {
		if reflect.DeepEqual(e, v) {
			return true
		}
	}
	return false
}

func show(v any) string {
	if v == nil {
		return "null"
	}
	return fmt.Sprintf("%q", fmt.Sprintf("%v", v))
}
