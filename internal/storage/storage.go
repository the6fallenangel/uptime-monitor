package storage

import (
	"context"
	"errors"
	"time"

	"github.com/the6fallenangel/uptime-monitor/internal/models"
)

var ErrMonitorNotFound = errors.New("monitor not found")

type Storage interface {
	CreateUser(ctx context.Context, user models.User) (models.User, error)
	GetUserByEmail(ctx context.Context, email string) (models.User, error)
	GetUserByID(ctx context.Context, id int64) (models.User, error)

	CreateMonitor(ctx context.Context, monitor models.Monitor) (models.Monitor, error)
	UpdateMonitor(ctx context.Context, id, userID int64, name string, interval time.Duration) (models.Monitor, error)
	ListMonitorsForUser(ctx context.Context, userID int64) ([]models.Monitor, error)
	GetMonitorForUser(ctx context.Context, id int64, userID int64) (models.Monitor, error)
	DeleteMonitorForUser(ctx context.Context, id int64, userID int64) error

	SaveCheck(ctx context.Context, check models.Check) (models.Check, error)
	ListChecks(ctx context.Context, monitorID int64, limit int) ([]models.Check, error)
}
