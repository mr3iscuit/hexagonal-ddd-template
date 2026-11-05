package application

import "time"

// ClockPort tells the core the current time.
type ClockPort interface {
	Now() time.Time
}
