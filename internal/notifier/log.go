package notifier

import (
	"context"
	"log/slog"
)

type LogNotifier struct{}

func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

func (n *LogNotifier) Notify(ctx context.Context, event Event) error {
	slog.Warn("monitor status changed",
		"monitor_id", event.Monitor.ID,
		"monitor_name", event.Monitor.Name,
		"url", event.Monitor.URL,
		"previous_status", event.PreviousStatus,
		"new_status", event.NewStatus,
		"error", event.Check.Error,
	)
	return nil
}
