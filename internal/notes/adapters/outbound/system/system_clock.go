// Package system adapts the operating system's clock and randomness.
package system

import "time"

// SystemClock never calls UTC, because that removes Go's monotonic reading.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }
