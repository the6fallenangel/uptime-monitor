package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/the6fallenangel/uptime-monitor/internal/api"
	"github.com/the6fallenangel/uptime-monitor/internal/checker"
	"github.com/the6fallenangel/uptime-monitor/internal/config"
	"github.com/the6fallenangel/uptime-monitor/internal/scheduler"
	"github.com/the6fallenangel/uptime-monitor/internal/storage"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := storage.NewPostgresStorage(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("error initializing storage:", err)
	}
	defer store.Close()

	monitors, err := store.ListMonitors(ctx)
	if err != nil {
		log.Fatal("error loading monitors:", err)
	}

	chk := checker.New(10 * time.Second)
	sched := scheduler.New(store, chk, 10)
	go sched.Run(ctx, monitors)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux, store, sched)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	go func() {
		fmt.Println("listening on", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error:", err)
		}
	}()

	<-ctx.Done()
	fmt.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal("graceful shutdown failed:", err)
	}
	fmt.Println("server stopped cleanly")
}
