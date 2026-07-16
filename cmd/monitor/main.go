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
	"github.com/the6fallenangel/uptime-monitor/internal/auth"
	"github.com/the6fallenangel/uptime-monitor/internal/checker"
	"github.com/the6fallenangel/uptime-monitor/internal/config"
	"github.com/the6fallenangel/uptime-monitor/internal/notifier"
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

	monitors, err := store.ListAllMonitors(ctx)
	if err != nil {
		log.Fatal("error loading monitors:", err)
	}

	var notif notifier.Notifier
	if cfg.SMTPHost != "" {
		notif = notifier.NewEmailNotifier(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.AlertFrom)
	} else {
		notif = notifier.NewLogNotifier()
	}

	chk := checker.New(10 * time.Second)
	sched := scheduler.New(store, chk, notif, 10)
	go sched.Run(ctx, monitors)

	issuer := auth.NewTokenIssuer(cfg.JWTSecret, 7*24*time.Hour)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux, store, sched, issuer)

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
