package fetcher

import "time"

// Anticipating these would be used in the future.

type InsulinPoint struct {
	Value int
	Type  string
	Time  time.Time
}

type CarbPoint struct {
	Value int
	Time  time.Time
}
