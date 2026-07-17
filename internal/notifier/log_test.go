package notifier

import (
	"context"
	"testing"

	"github.com/the6fallenangel/uptime-monitor/internal/models"
)

func TestLogNotifierDoesNotError(t *testing.T) {
	n := NewLogNotifier()

	event := Event{
		Monitor:        models.Monitor{ID: 1, Name: "Test", URL: "https://example.com"},
		PreviousStatus: models.StatusUp,
		NewStatus:      models.StatusDown,
	}

	if err := n.Notify(context.Background(), event); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
