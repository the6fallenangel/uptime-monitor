package models

import "time"

type Monitor struct {
	ID        int64
	UserID    int64
	Name      string
	URL       string
	Interval  time.Duration
	CreatedAt time.Time
}

func NewMonitor(userID int64, name, url string, interval time.Duration) Monitor {
	return Monitor{
		UserID:    userID,
		Name:      name,
		URL:       url,
		Interval:  interval,
		CreatedAt: time.Now(),
	}
}
