// Package app is the composition root — the ONLY wiring site in the codebase
// (ARCHITECTURE.md §12, decision D4). One hand-built function (~60 lines, no
// DI framework) compiles the rule packs, builds the detector catalog through
// explicit constructors (the compiled matcher injected as an argument, never
// a global; duplicate detector IDs panic at startup), opens the cache under
// its self-invalidating namespace hash, detects the source type, runs the
// engine, assembles the inventory, and fans out the writers.
//
// pkg/airom.Scan(ctx, target, opts) wraps this same path for library
// embedders, so the CLI, embedders, and tests all exercise identical wiring.
// A constructor called anywhere else is a layering bug.
package app
