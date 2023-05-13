package store

import "time"

type InsulinPoint struct {
	Value int
	Type  string
	Time  time.Time
}

type CarbPoint struct {
	Value int
	Time  time.Time
}

type EventPoint struct {
	Event   string
	Message string
	Time    time.Time
}
