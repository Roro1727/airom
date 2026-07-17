package app

import "github.com/airomhq/airom/rules"

// Wire the embedded rule packs into the composition root. Kept in its own
// file so the SDK-facing app types stay free of the go:embed dependency and
// tests can override EmbeddedRules with an empty set.
func init() { EmbeddedRules = rules.FS() }
