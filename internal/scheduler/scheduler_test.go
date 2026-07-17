package scheduler

import (
	"context"
	"sync"
	"testing"

	"github.com/the6fallenangel/uptime-monitor/internal/models"
	"github.com/the6fallenangel/uptime-monitor/internal/notifier"
	"github.com/the6fallenangel/uptime-monitor/internal/storage"
)

type mockNotifier struct {
	mu     sync.Mutex
	events []notifier.Event
}

func (m *mockNotifier) Notify(ctx context.Context, event notifier.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *mockNotifier) eventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.events)
}

type stubStorage struct {
	storage.Storage
	userEmail string
}

func (s *stubStorage) GetUserByID(ctx context.Context, id int64) (models.User, error) {
	return models.User{ID: id, Email: s.userEmail}, nil
}

func TestDetectAndNotifyTransition(t *testing.T) {
	notif := &mockNotifier{}
	store := &stubStorage{userEmail: "owner@example.com"}

	s := &Scheduler{
		store:      store,
		notifier:   notif,
		lastStatus: make(map[int64]models.CheckStatus),
	}

	monitor := models.Monitor{ID: 1, UserID: 99, Name: "Test"}
	ctx := context.Background()

	s.detectAndNotifyTransition(ctx, monitor, models.Check{Status: models.StatusUp})
	if notif.eventCount() != 0 {
		t.Errorf("expected no notification on first check, got %d", notif.eventCount())
	}

	s.detectAndNotifyTransition(ctx, monitor, models.Check{Status: models.StatusUp})
	if notif.eventCount() != 0 {
		t.Errorf("expected no notification when status unchanged, got %d", notif.eventCount())
	}

	s.detectAndNotifyTransition(ctx, monitor, models.Check{Status: models.StatusDown})
	if notif.eventCount() != 1 {
		t.Fatalf("expected 1 notification on transition, got %d", notif.eventCount())
	}

	event := notif.events[0]
	if event.PreviousStatus != models.StatusUp || event.NewStatus != models.StatusDown {
		t.Errorf("expected up->down transition, got %s->%s", event.PreviousStatus, event.NewStatus)
	}
	if event.OwnerEmail != "owner@example.com" {
		t.Errorf("expected owner email to be looked up correctly, got %q", event.OwnerEmail)
	}

	s.detectAndNotifyTransition(ctx, monitor, models.Check{Status: models.StatusUp})
	if notif.eventCount() != 2 {
		t.Errorf("expected 2 notifications total after recovery, got %d", notif.eventCount())
	}
}
