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
	rootCtx     context.Context
	wg          sync.WaitGroup
	mu          sync.Mutex
	cancels     map[int64]context.CancelFunc
}

func New(store storage.Storage, chk *checker.Checker, workerCount int) *Scheduler {
	return &Scheduler{
		store:       store,
		checker:     chk,
		workerCount: workerCount,
		jobs:        make(chan models.Monitor),
		cancels:     make(map[int64]context.CancelFunc),
	}
}

func (s *Scheduler) Run(ctx context.Context, monitors []models.Monitor) {
	s.rootCtx = ctx

	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(ctx)
	}

	for _, m := range monitors {
		s.Add(m)
	}

	<-ctx.Done()
	slog.Info("scheduler shutting down, waiting for in-flight checks")
	s.wg.Wait()
	slog.Info("scheduler stopped cleanly")
}

func (s *Scheduler) Add(monitor models.Monitor) {
	monitorCtx, cancel := context.WithCancel(s.rootCtx)

	s.mu.Lock()
	s.cancels[monitor.ID] = cancel
	s.mu.Unlock()

	s.wg.Add(1)
	go s.scheduleMonitor(monitorCtx, monitor)
}

func (s *Scheduler) Remove(monitorID int64) {
	s.mu.Lock()
	cancel, ok := s.cancels[monitorID]
	if ok {
		delete(s.cancels, monitorID)
	}
	s.mu.Unlock()

	if ok {
		cancel()
	}
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
