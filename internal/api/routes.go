package api

import (
	"net/http"

	"github.com/the6fallenangel/uptime-monitor/internal/storage"
)

func RegisterRoutes(mux *http.ServeMux, store storage.Storage) {
	mux.HandleFunc("POST /monitors", handleCreateMonitor(store))
	mux.HandleFunc("GET /monitors", handleListMonitors(store))
	mux.HandleFunc("GET /monitors/{id}", handleGetMonitor(store))
	mux.HandleFunc("DELETE /monitors/{id}", handleDeleteMonitor(store))
	mux.HandleFunc("GET /monitors/{id}/checks", handleListChecks(store))
}
