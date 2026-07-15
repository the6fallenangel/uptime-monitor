package checker

import (
	"context"
	"net/http"
	"time"

	"github.com/the6fallenangel/uptime-monitor/internal/models"
)

type Checker struct {
	client *http.Client
}

func New(timeout time.Duration) *Checker {
	return &Checker{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Checker) Check(ctx context.Context, monitor models.Monitor) models.Check {
	check := models.Check{
		MonitorID: monitor.ID,
		CheckedAt: time.Now(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, monitor.URL, nil)
	if err != nil {
		check.Status = models.StatusDown
		check.Error = err.Error()
		check.CheckedAt = time.Now()
		return check
	}

	start := time.Now()
	resp, err := c.client.Do(req)
	check.ResponseTime = time.Since(start)

	if err != nil {
		check.Status = models.StatusDown
		check.Error = err.Error()
		check.CheckedAt = time.Now()
		return check
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	check.StatusCode = &statusCode

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		check.Status = models.StatusUp
	} else {
		check.Status = models.StatusDown
	}

	check.CheckedAt = time.Now()
	return check
}
