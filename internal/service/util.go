package service

import "math"

// round2 mirrors JS's Math.round(x * 100) / 100 — Math.round rounds half
// towards +Infinity (floor(x + 0.5)), unlike Go's usual round-half-away-
// from-zero, so it's reproduced explicitly to match the Node backend's
// cash-register and change-due math exactly.
func round2(v float64) float64 {
	return math.Floor(v*100+0.5) / 100
}

// centsOf rounds to the nearest cent and returns an integer so money
// comparisons (e.g. "valor recebido >= total") aren't tripped up by binary
// floating point representation error (0.1 + 0.2 != 0.3).
func centsOf(v float64) int64 {
	return int64(math.Floor(v*100 + 0.5))
}
