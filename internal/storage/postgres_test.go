package storage

import (
	"context"
	"testing"
	"time"

	"github.com/the6fallenangel/uptime-monitor/internal/models"
)

func TestCreateAndGetMonitor(t *testing.T) {
	store := newTestStorage(t)
	ctx := context.Background()

	monitor := models.NewMonitor("Example", "https://example.com", 30*time.Second)

	saved, err := store.CreateMonitor(ctx, monitor)
	if err != nil {
		t.Fatalf("unexpected error creating monitor: %v", err)
	}
	if saved.ID == 0 {
		t.Errorf("expected non-zero id")
	}

	fetched, err := store.GetMonitor(ctx, saved.ID)
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

	store.CreateMonitor(ctx, models.NewMonitor("A", "https://a.example.com", time.Minute))
	store.CreateMonitor(ctx, models.NewMonitor("B", "https://b.example.com", time.Minute))

	monitors, err := store.ListMonitors(ctx)
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

	saved, _ := store.CreateMonitor(ctx, models.NewMonitor("Temp", "https://example.com", time.Minute))

	if err := store.DeleteMonitor(ctx, saved.ID); err != nil {
		t.Fatalf("unexpected error deleting monitor: %v", err)
	}

	if _, err := store.GetMonitor(ctx, saved.ID); err == nil {
		t.Errorf("expected error fetching deleted monitor, got nil")
	}
}

func TestDeleteMonitorNotFound(t *testing.T) {
	store := newTestStorage(t)
	ctx := context.Background()

	if err := store.DeleteMonitor(ctx, 999999); err == nil {
		t.Errorf("expected error deleting nonexistent monitor, got nil")
	}
}

func TestSaveAndListChecks(t *testing.T) {
	store := newTestStorage(t)
	ctx := context.Background()

	monitor, _ := store.CreateMonitor(ctx, models.NewMonitor("Example", "https://example.com", time.Minute))

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

	monitor, _ := store.CreateMonitor(ctx, models.NewMonitor("Example", "https://example.com", time.Minute))
	store.SaveCheck(ctx, models.Check{
		MonitorID: monitor.ID,
		Status:    models.StatusUp,
		CheckedAt: time.Now(),
	})

	store.DeleteMonitor(ctx, monitor.ID)

	checks, err := store.ListChecks(ctx, monitor.ID, 10)
	if err != nil {
		t.Fatalf("unexpected error listing checks: %v", err)
	}
	if len(checks) != 0 {
		t.Errorf("expected checks to be cascade-deleted, got %d", len(checks))
	}
}
