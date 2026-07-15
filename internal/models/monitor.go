package models

import "time"

type Monitor struct {
	ID        int64
	Name      string
	URL       string
	Interval  time.Duration
	CreatedAt time.Time
}

func NewMonitor(name, url string, interval time.Duration) Monitor {
	return Monitor{
		Name:      name,
		URL:       url,
		Interval:  interval,
		CreatedAt: time.Now(),
	}
}
