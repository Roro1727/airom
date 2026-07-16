// Package schemas embeds AIROM's published JSON Schemas (docs/mapping.md):
// the native AIBOM format is a versioned API, and its schema ships with the
// binary and the repo. Additive-only within a major version.
package schemas

import _ "embed"

// NativeV1 is the JSON Schema for the native airom-json format (schemaVersion 1).
//
//go:embed airom-v1.schema.json
var NativeV1 []byte
