package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/the6fallenangel/uptime-monitor/internal/models"
	"github.com/the6fallenangel/uptime-monitor/internal/scheduler"
	"github.com/the6fallenangel/uptime-monitor/internal/storage"
)

type createMonitorRequest struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Interval string `json:"interval"`
}

func handleCreateMonitor(store storage.Storage, sched *scheduler.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createMonitorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		if req.Name == "" || req.URL == "" {
			writeError(w, http.StatusBadRequest, errString("name and url are required"))
			return
		}

		interval, err := time.ParseDuration(req.Interval)
		if err != nil || interval <= 0 {
			writeError(w, http.StatusBadRequest, errString("interval must be a valid duration, e.g. \"30s\", \"5m\""))
			return
		}

		monitor := models.NewMonitor(req.Name, req.URL, interval)

		saved, err := store.CreateMonitor(r.Context(), monitor)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		sched.Add(saved)

		writeJSON(w, http.StatusCreated, saved)
	}
}

func handleListMonitors(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		monitors, err := store.ListMonitors(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, monitors)
	}
}

func handleGetMonitor(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, errString("invalid monitor id"))
			return
		}

		monitor, err := store.GetMonitor(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, monitor)
	}
}

func handleDeleteMonitor(store storage.Storage, sched *scheduler.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, errString("invalid monitor id"))
			return
		}

		if err := store.DeleteMonitor(r.Context(), id); err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}

		sched.Remove(id)

		w.WriteHeader(http.StatusNoContent)
	}
}

func handleListChecks(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, errString("invalid monitor id"))
			return
		}

		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			parsed, err := strconv.Atoi(l)
			if err == nil && parsed > 0 {
				limit = parsed
			}
		}

		checks, err := store.ListChecks(r.Context(), id, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, checks)
	}
}
