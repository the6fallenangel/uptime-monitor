package api

import (
	"net/http"

	"github.com/the6fallenangel/uptime-monitor/internal/auth"
	"github.com/the6fallenangel/uptime-monitor/internal/scheduler"
	"github.com/the6fallenangel/uptime-monitor/internal/storage"
)

func RegisterRoutes(mux *http.ServeMux, store storage.Storage, sched *scheduler.Scheduler, issuer *auth.TokenIssuer) {
	mux.HandleFunc("POST /signup", handleSignup(store, issuer))
	mux.HandleFunc("POST /login", handleLogin(store, issuer))
	mux.HandleFunc("POST /logout", handleLogout)

	mux.Handle("POST /monitors", requireAuth(issuer, handleCreateMonitor(store, sched)))
	mux.Handle("GET /monitors", requireAuth(issuer, handleListMonitorsForUser(store)))
	mux.Handle("GET /monitors/{id}", requireAuth(issuer, handleGetMonitorForUser(store)))
	mux.Handle("DELETE /monitors/{id}", requireAuth(issuer, handleDeleteMonitorForUser(store, sched)))
	mux.Handle("GET /monitors/{id}/checks", requireAuth(issuer, handleListChecks(store)))
}
