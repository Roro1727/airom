package conformance

import (
	"testing"
	"time"

	"github.com/airomhq/airom/internal/writer"
	"github.com/airomhq/airom/pkg/airom"
)

// TestConfidenceCanonicalForm pins the §6.2 serialization of the sample value:
// 0.8738 is already 4 fractional digits, so it renders verbatim as text
// "0.8738" and number 0.8738 — no rounding, no trailing-zero trim needed.
func TestConfidenceCanonicalForm(t *testing.T) {
	if got := writer.FormatConfidence(airom.Confidence(0.8738)); got != "0.8738" {
		t.Errorf("FormatConfidence(0.8738) = %q, want 0.8738", got)
	}
	if got := writer.ConfidenceNumber(airom.Confidence(0.8738)); got != 0.8738 {
		t.Errorf("ConfidenceNumber(0.8738) = %v, want 0.8738", got)
	}
}

// TestConfidenceFormatAcrossFormats asserts the identical §6.2 form for a
// confidence of 0.8738 in every sink that carries it: native JSON (number), the
// CDX airom:confidence property (string), and CDX evidence.identity[].confidence
// (number). The shared fixture's model already has component confidence 0.8738;
// its identity claim carries a different value (0.85), so a dedicated inventory
// drives the identity[].confidence sink with 0.8738.
func TestConfidenceFormatAcrossFormats(t *testing.T) {
	// (1) Shared fixture: component-level 0.8738 in native + CDX property.
	nativeModel := findGenComp(t, render(t, "json", writer.Options{}), refModel)
	if got := nativeModel["confidence"].(float64); got != 0.8738 {
		t.Errorf("native component confidence = %v, want 0.8738", got)
	}
	cdxModel := findGenComp(t, render(t, "cyclonedx", writer.Options{}), refModel)
	if got := propStr(t, cdxModel, "airom:confidence"); got != "0.8738" {
		t.Errorf("CDX airom:confidence property = %q, want 0.8738", got)
	}

	// (2) Dedicated inventory: identity claim confidence 0.8738 as well, so the
	// evidence.identity[].confidence sink is exercised with the same value.
	const cid = "airom:8738873887388738"
	inv := &airom.Inventory{
		SchemaVersion: "1",
		Tool:          airom.ToolInfo{Name: "airom", Version: "1.0.0"},
		Serial:        "00000000-0000-4000-8000-000000000002",
		Timestamp:     time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC),
		Source:        airom.SourceInfo{Kind: "dir", Target: "/x"},
		Root:          airom.ID(refRoot),
		Components: []airom.Component{
			{ID: airom.ID(refRoot), Kind: airom.KindApplication, Name: "app", Confidence: 1},
			{
				ID: airom.ID(cid), Kind: airom.KindHostedLLM, Name: "m", Confidence: 0.8738,
				Evidence: airom.Evidence{Identity: []airom.IdentityClaim{
					{Field: "name", Value: "m", Confidence: 0.8738, Methods: []airom.DetectionMethod{airom.MethodSourceCode}},
				}},
			},
		},
	}

	nc := findGenComp(t, renderInv(t, "json", writer.Options{}, inv), cid)
	if got := nc["confidence"].(float64); got != 0.8738 {
		t.Errorf("dedicated native component confidence = %v, want 0.8738", got)
	}
	if got := identityConfidence(t, nc); got != 0.8738 {
		t.Errorf("native evidence.identity[0].confidence = %v, want 0.8738", got)
	}

	cc := findGenComp(t, renderInv(t, "cyclonedx", writer.Options{}, inv), cid)
	if got := propStr(t, cc, "airom:confidence"); got != "0.8738" {
		t.Errorf("dedicated CDX airom:confidence = %q, want 0.8738", got)
	}
	if got := identityConfidence(t, cc); got != 0.8738 {
		t.Errorf("CDX evidence.identity[0].confidence = %v, want 0.8738", got)
	}
}

// ── generic-JSON navigation helpers ─────────────────────────────────────────

// findGenComp decodes a rendered document and returns the component object with
// the given id, from the top-level "components" array.
func findGenComp(t *testing.T, raw []byte, id string) map[string]any {
	t.Helper()
	doc := mustJSONMap(t, raw)
	for _, c := range arr(t, doc["components"]) {
		cm := obj(t, c)
		if idOf(cm) == id {
			return cm
		}
	}
	t.Fatalf("component %q not found", id)
	return nil
}

// idOf returns a component's identity across formats: native uses "id", CDX uses
// "bom-ref".
func idOf(comp map[string]any) string {
	if v, ok := comp["id"].(string); ok {
		return v
	}
	if v, ok := comp["bom-ref"].(string); ok {
		return v
	}
	return ""
}

// propStr returns the value of a CDX airom:* property by name.
func propStr(t *testing.T, comp map[string]any, name string) string {
	t.Helper()
	props, ok := comp["properties"].([]any)
	if !ok {
		t.Fatalf("component %q has no properties", idOf(comp))
	}
	for _, p := range props {
		pm := obj(t, p)
		if str(t, pm["name"]) == name {
			return str(t, pm["value"])
		}
	}
	t.Fatalf("property %q not found on %q", name, idOf(comp))
	return ""
}

// identityConfidence reads evidence.identity[0].confidence, which is a JSON
// array in both native and CDX 1.6+.
func identityConfidence(t *testing.T, comp map[string]any) float64 {
	t.Helper()
	ev := obj(t, comp["evidence"])
	ids := arr(t, ev["identity"])
	return obj(t, ids[0])["confidence"].(float64)
}
