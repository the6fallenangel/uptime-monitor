package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/the6fallenangel/uptime-monitor/internal/checker"
	"github.com/the6fallenangel/uptime-monitor/internal/models"
	"github.com/the6fallenangel/uptime-monitor/internal/storage"
)

type Scheduler struct {
	store       storage.Storage
	checker     *checker.Checker
	workerCount int
	jobs        chan models.Monitor
	wg          sync.WaitGroup
}

func New(store storage.Storage, chk *checker.Checker, workerCount int) *Scheduler {
	return &Scheduler{
		store:       store,
		checker:     chk,
		workerCount: workerCount,
		jobs:        make(chan models.Monitor),
	}
}

func (s *Scheduler) Run(ctx context.Context, monitors []models.Monitor) {
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(ctx)
	}

	for _, m := range monitors {
		s.wg.Add(1)
		go s.scheduleMonitor(ctx, m)
	}

	<-ctx.Done()
	slog.Info("scheduler shutting down, waiting for in-flight checks")
	s.wg.Wait()
	slog.Info("scheduler stopped cleanly")
}

func (s *Scheduler) worker(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case monitor, ok := <-s.jobs:
			if !ok {
				return
			}
			s.runCheck(ctx, monitor)
		}
	}
}

func (s *Scheduler) scheduleMonitor(ctx context.Context, monitor models.Monitor) {
	defer s.wg.Done()

	ticker := time.NewTicker(monitor.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			select {
			case s.jobs <- monitor:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (s *Scheduler) runCheck(ctx context.Context, monitor models.Monitor) {
	check := s.checker.Check(ctx, monitor)

	if _, err := s.store.SaveCheck(ctx, check); err != nil {
		slog.Error("failed to save check result",
			"monitor_id", monitor.ID,
			"url", monitor.URL,
			"error", err,
		)
		return
	}

	slog.Info("check completed",
		"monitor_id", monitor.ID,
		"url", monitor.URL,
		"status", check.Status,
		"response_time", check.ResponseTime,
	)
}
