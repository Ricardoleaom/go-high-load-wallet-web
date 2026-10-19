//go:build race

package walletapi

// raceEnabled is true when the test binary was built with -race. The race
// detector's instrumentation overhead (tens of times slower per memory
// access) makes throughput-sensitive tests unreliable: 2,000 real HTTP
// round trips a second start hitting the client's own request timeout
// under that overhead, not because the code is wrong. Correctness is
// still verified under -race elsewhere; this flag only loosens the one
// test that asserts a sustained rate.
const raceEnabled = true
