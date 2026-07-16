//go:build !race

package perf

// raceEnabled is false in an ordinary (non-race) test binary, where memory
// measurement is trustworthy and the RSS assertions run.
const raceEnabled = false
