package notifier

import (
	"context"

	"github.com/the6fallenangel/uptime-monitor/internal/models"
)

type Event struct {
	Monitor        models.Monitor
	PreviousStatus models.CheckStatus
	NewStatus      models.CheckStatus
	Check          models.Check
}

type Notifier interface {
	Notify(ctx context.Context, event Event) error
}
