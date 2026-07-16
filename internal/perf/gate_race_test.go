//go:build race

package perf

// raceEnabled reports whether the test binary was built with -race. The race
// detector inflates and reshuffles heap accounting, so the RSS-ceiling and
// ratio assertions are meaningless under it — they gate on this constant and
// skip, leaving the -race run to exercise only the generator and a real scan
// (the determinism smoke), which is where -race earns its keep.
const raceEnabled = true
