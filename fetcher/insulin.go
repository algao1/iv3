package fetcher

import "time"

// Anticipating this would be used in the future.
type InsulinPoint struct {
	Value int
	Type  string
	Time  time.Time
}
