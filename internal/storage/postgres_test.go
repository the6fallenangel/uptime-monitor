package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/the6fallenangel/uptime-monitor/internal/models"
)

func createTestUser(t *testing.T, store *PostgresStorage) models.User {
	t.Helper()

	user, err := models.NewUser("Test User", fmt.Sprintf("test-%d@example.com", time.Now().UnixNano()), "password123")
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}

	saved, err := store.CreateUser(context.Background(), user)
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}
	return saved
}

func TestCreateAndGetMonitor(t *testing.T) {
	store := newTestStorage(t)
	ctx := context.Background()

	user := createTestUser(t, store)
	monitor := models.NewMonitor(user.ID, "Example", "https://example.com", 30*time.Second)

	saved, err := store.CreateMonitor(ctx, monitor)
	if err != nil {
		t.Fatalf("unexpected error creating monitor: %v", err)
	}
	if saved.ID == 0 {
		t.Errorf("expected non-zero id")
	}

	fetched, err := store.GetMonitorForUser(ctx, saved.ID, user.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching monitor: %v", err)
	}
	if fetched.Name != "Example" {
		t.Errorf("expected name %q, got %q", "Example", fetched.Name)
	}
	if fetched.Interval != 30*time.Second {
		t.Errorf("expected interval 30s, got %v", fetched.Interval)
	}
}

func TestListMonitors(t *testing.T) {
	store := newTestStorage(t)
	ctx := context.Background()

	user := createTestUser(t, store)
	store.CreateMonitor(ctx, models.NewMonitor(user.ID, "A", "https://a.example.com", time.Minute))
	store.CreateMonitor(ctx, models.NewMonitor(user.ID, "B", "https://b.example.com", time.Minute))

	monitors, err := store.ListMonitorsForUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("unexpected error listing monitors: %v", err)
	}
	if len(monitors) != 2 {
		t.Fatalf("expected 2 monitors, got %d", len(monitors))
	}
}

func TestDeleteMonitor(t *testing.T) {
	store := newTestStorage(t)
	ctx := context.Background()

	user := createTestUser(t, store)
	saved, _ := store.CreateMonitor(ctx, models.NewMonitor(user.ID, "Temp", "https://example.com", time.Minute))
	if err := store.DeleteMonitorForUser(ctx, saved.ID, user.ID); err != nil {
		t.Fatalf("unexpected error deleting monitor: %v", err)
	}

	if _, err := store.GetMonitorForUser(ctx, saved.ID, user.ID); err == nil {
		t.Errorf("expected error fetching deleted monitor, got nil")
	}
}

func TestDeleteMonitorNotFound(t *testing.T) {
	store := newTestStorage(t)
	ctx := context.Background()

	user := createTestUser(t, store)
	if err := store.DeleteMonitorForUser(ctx, 999999, user.ID); err == nil {
		t.Errorf("expected error deleting nonexistent monitor, got nil")
	}
}

func TestSaveAndListChecks(t *testing.T) {
	store := newTestStorage(t)
	ctx := context.Background()

	user := createTestUser(t, store)
	monitor, _ := store.CreateMonitor(ctx, models.NewMonitor(user.ID, "Example", "https://example.com", time.Minute))

	statusCode := 200
	check := models.Check{
		MonitorID:    monitor.ID,
		Status:       models.StatusUp,
		StatusCode:   &statusCode,
		ResponseTime: 250 * time.Millisecond,
		CheckedAt:    time.Now(),
	}

	saved, err := store.SaveCheck(ctx, check)
	if err != nil {
		t.Fatalf("unexpected error saving check: %v", err)
	}
	if saved.ID == 0 {
		t.Errorf("expected non-zero check id")
	}

	checks, err := store.ListChecks(ctx, monitor.ID, 10)
	if err != nil {
		t.Fatalf("unexpected error listing checks: %v", err)
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(checks))
	}
	if checks[0].Status != models.StatusUp {
		t.Errorf("expected status %q, got %q", models.StatusUp, checks[0].Status)
	}
}

func TestDeleteMonitorCascadesChecks(t *testing.T) {
	store := newTestStorage(t)
	ctx := context.Background()

	user := createTestUser(t, store)
	monitor, _ := store.CreateMonitor(ctx, models.NewMonitor(user.ID, "Example", "https://example.com", time.Minute))
	store.SaveCheck(ctx, models.Check{
		MonitorID: monitor.ID,
		Status:    models.StatusUp,
		CheckedAt: time.Now(),
	})

	store.DeleteMonitorForUser(ctx, monitor.ID, user.ID)

	checks, err := store.ListChecks(ctx, monitor.ID, 10)
	if err != nil {
		t.Fatalf("unexpected error listing checks: %v", err)
	}
	if len(checks) != 0 {
		t.Errorf("expected checks to be cascade-deleted, got %d", len(checks))
	}
}

func TestGetMonitorForUserRejectsOtherUsersMonitor(t *testing.T) {
	store := newTestStorage(t)
	ctx := context.Background()

	owner := createTestUser(t, store)
	otherUser := createTestUser(t, store)

	monitor, _ := store.CreateMonitor(ctx, models.NewMonitor(owner.ID, "Private", "https://example.com", time.Minute))

	if _, err := store.GetMonitorForUser(ctx, monitor.ID, otherUser.ID); err == nil {
		t.Errorf("expected error fetching another user's monitor, got nil")
	}
}

func TestDeleteMonitorForUserRejectsOtherUsersMonitor(t *testing.T) {
	store := newTestStorage(t)
	ctx := context.Background()

	owner := createTestUser(t, store)
	otherUser := createTestUser(t, store)

	monitor, _ := store.CreateMonitor(ctx, models.NewMonitor(owner.ID, "Private", "https://example.com", time.Minute))

	if err := store.DeleteMonitorForUser(ctx, monitor.ID, otherUser.ID); err == nil {
		t.Errorf("expected error deleting another user's monitor, got nil")
	}

	if _, err := store.GetMonitorForUser(ctx, monitor.ID, owner.ID); err != nil {
		t.Errorf("expected monitor to still exist for its real owner, got error: %v", err)
	}
}

func TestListMonitorsForUserExcludesOtherUsersMonitors(t *testing.T) {
	store := newTestStorage(t)
	ctx := context.Background()

	userA := createTestUser(t, store)
	userB := createTestUser(t, store)

	store.CreateMonitor(ctx, models.NewMonitor(userA.ID, "A's monitor", "https://a.example.com", time.Minute))
	store.CreateMonitor(ctx, models.NewMonitor(userB.ID, "B's monitor", "https://b.example.com", time.Minute))

	monitorsForA, err := store.ListMonitorsForUser(ctx, userA.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(monitorsForA) != 1 {
		t.Fatalf("expected 1 monitor for user A, got %d", len(monitorsForA))
	}
	if monitorsForA[0].Name != "A's monitor" {
		t.Errorf("expected user A's monitor list to only contain their own monitor, got %q", monitorsForA[0].Name)
	}
}
