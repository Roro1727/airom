// Package detectortest is the public contract-test and golden-file harness
// for detector authors (ARCHITECTURE.md §14) — the same harness that
// exercises the built-ins. Run asserts that golden findings match, that
// Selector() actually gates, that locations are 1-based, that two runs are
// byte-identical (determinism), and that the detector never panics on
// truncated or empty input; every detector is run against BOTH dir-backed
// and tar-stream-backed inputs, catching seekability bugs before merge.
//
// Publishing the harness is part of the plugin-SDK bet (decision D5): a
// third-party detector is held to exactly the standard the built-ins are,
// with no private test infrastructure required.
package detectortest
