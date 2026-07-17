package models

import "time"

type CheckStatus string

const (
	StatusUp   CheckStatus = "up"
	StatusDown CheckStatus = "down"
)

type Check struct {
	ID           int64         `json:"id"`
	MonitorID    int64         `json:"monitorId"`
	Status       CheckStatus   `json:"status"`
	StatusCode   *int          `json:"statusCode"`
	ResponseTime time.Duration `json:"responseTime"`
	Error        string        `json:"error"`
	CheckedAt    time.Time     `json:"checkedAt"`
}
