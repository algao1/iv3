package store

import "time"

type GlucosePoint struct {
	WT    string  `json:"WT"` // Not exactly sure what this stands for.
	Value float64 `json:"Value"`
	Trend string  `json:"Trend"`
	Time  time.Time
}

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
