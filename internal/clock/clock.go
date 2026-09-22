// Package clock provides the same local-time formatting helpers as the
// original src/util.js, so timestamps stored in the database keep the exact
// same shape ("YYYY-MM-DD HH:MM:SS" / "YYYY-MM-DD") across the Node and Go
// backends.
package clock

import "time"

const dateLayout = "2006-01-02"
const dateTimeLayout = "2006-01-02 15:04:05"

// NowLocal mirrors util.js's nowLocal(): the local wall-clock timestamp as
// "YYYY-MM-DD HH:MM:SS".
func NowLocal() string {
	return time.Now().Format(dateTimeLayout)
}

// TodayLocal mirrors util.js's todayLocal(): the local calendar date as
// "YYYY-MM-DD".
func TodayLocal() string {
	return time.Now().Format(dateLayout)
}
