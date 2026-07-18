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
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, errString("not authenticated"))
			return
		}

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

		monitor := models.NewMonitor(userID, req.Name, req.URL, interval)

		saved, err := store.CreateMonitor(r.Context(), monitor)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		sched.Add(saved)

		writeJSON(w, http.StatusCreated, saved)
	}
}

type updateMonitorRequest struct {
	Name     string `json:"name"`
	Interval string `json:"interval"`
}

func handleUpdateMonitor(store storage.Storage, sched *scheduler.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, errString("not authenticated"))
			return
		}

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, errString("invalid monitor id"))
			return
		}

		var req updateMonitorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		if req.Name == "" {
			writeError(w, http.StatusBadRequest, errString("name is required"))
			return
		}

		interval, err := time.ParseDuration(req.Interval)
		if err != nil || interval <= 0 {
			writeError(w, http.StatusBadRequest, errString("interval must be a valid duration, e.g. \"30s\", \"5m\""))
			return
		}

		updated, err := store.UpdateMonitor(r.Context(), id, userID, req.Name, interval)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}

		sched.Update(updated)

		writeJSON(w, http.StatusOK, updated)
	}
}

func handleListMonitorsForUser(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, errString("not authenticated"))
			return
		}

		monitors, err := store.ListMonitorsForUser(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, monitors)
	}
}

func handleGetMonitorForUser(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, errString("not authenticated"))
			return
		}

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, errString("invalid monitor id"))
			return
		}

		monitor, err := store.GetMonitorForUser(r.Context(), id, userID)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, monitor)
	}
}

func handleDeleteMonitorForUser(store storage.Storage, sched *scheduler.Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, errString("not authenticated"))
			return
		}

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, errString("invalid monitor id"))
			return
		}

		if err := store.DeleteMonitorForUser(r.Context(), id, userID); err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}

		sched.Remove(id)

		w.WriteHeader(http.StatusNoContent)
	}
}

func handleListChecks(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, errString("not authenticated"))
			return
		}

		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, errString("invalid monitor id"))
			return
		}

		if _, err := store.GetMonitorForUser(r.Context(), id, userID); err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}

		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			parsed, err := strconv.Atoi(p)
			if err == nil && parsed > 0 {
				page = parsed
			}
		}

		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			parsed, err := strconv.Atoi(l)
			if err == nil && parsed > 0 {
				limit = parsed
			}
		}

		offset := (page - 1) * limit

		checks, err := store.ListChecks(r.Context(), id, limit, offset)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		total, err := store.CountChecks(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"checks":     checks,
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + limit - 1) / limit,
		})
	}
}
