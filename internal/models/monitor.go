package models

import "time"

type Monitor struct {
	ID        int64         `json:"id"`
	UserID    int64         `json:"userId"`
	Name      string        `json:"name"`
	URL       string        `json:"url"`
	Interval  time.Duration `json:"interval"`
	CreatedAt time.Time     `json:"createdAt"`
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
