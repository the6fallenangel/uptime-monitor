package models

import "time"

type CheckStatus string

const (
	StatusUp   CheckStatus = "up"
	StatusDown CheckStatus = "down"
)

type Check struct {
	ID           int
	MonitorID    int
	Status       CheckStatus
	StatusCode   int
	ResponseTime time.Duration
	Error        string
	CheckedAt    time.Time
}
