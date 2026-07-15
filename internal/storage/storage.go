package storage

import (
	"context"

	"github.com/the6fallenangel/uptime-monitor/internal/models"
)

type Storage interface {
	CreateMonitor(ctx context.Context, monitor models.Monitor) (models.Monitor, error)
	ListMonitors(ctx context.Context) ([]models.Monitor, error)
	GetMonitor(ctx context.Context, id int64) (models.Monitor, error)
	DeleteMonitor(ctx context.Context, id int64) error

	SaveCheck(ctx context.Context, check models.Check) (models.Check, error)
	ListChecks(ctx context.Context, monitorID int64, limit int) ([]models.Check, error)
}
