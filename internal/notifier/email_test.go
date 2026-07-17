package notifier

import (
	"context"
	"testing"

	"github.com/the6fallenangel/uptime-monitor/internal/models"
)

func TestEmailNotifierRejectsMissingOwnerEmail(t *testing.T) {
	n := NewEmailNotifier("smtp.example.com", 587, "user", "pass", "alerts@example.com")

	event := Event{
		Monitor: models.Monitor{ID: 1, Name: "Test", URL: "https://example.com"},
	}

	err := n.Notify(context.Background(), event)
	if err == nil {
		t.Errorf("expected error when OwnerEmail is empty, got nil")
	}
}
