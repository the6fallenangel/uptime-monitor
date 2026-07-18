package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/the6fallenangel/uptime-monitor/internal/auth"
	"github.com/the6fallenangel/uptime-monitor/internal/models"
	"github.com/the6fallenangel/uptime-monitor/internal/storage"
)

const cookieName = "session_token"

type signupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func isAuthenticated(r *http.Request, issuer *auth.TokenIssuer) bool {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	_, err = issuer.Verify(cookie.Value)
	return err == nil
}

func handleSignup(store storage.Storage, issuer *auth.TokenIssuer, isProduction bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAuthenticated(r, issuer) {
			writeError(w, http.StatusConflict, errString("already logged in"))
			return
		}
		var req signupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		if req.Name == "" || req.Email == "" || len(req.Password) < 8 {
			writeError(w, http.StatusBadRequest, errString("name, email, and a password of at least 8 characters are required"))
			return
		}

		user, err := models.NewUser(req.Name, req.Email, req.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		saved, err := store.CreateUser(r.Context(), user)
		if err != nil {
			writeError(w, http.StatusConflict, errString("email already registered"))
			return
		}

		issueSessionCookie(w, issuer, saved.ID, isProduction)
		writeJSON(w, http.StatusCreated, map[string]any{"id": saved.ID, "name": saved.Name, "email": saved.Email})
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func handleLogin(store storage.Storage, issuer *auth.TokenIssuer, isProduction bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAuthenticated(r, issuer) {
			writeError(w, http.StatusConflict, errString("already logged in"))
			return
		}
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		user, err := store.GetUserByEmail(r.Context(), req.Email)
		if err != nil || !user.CheckPassword(req.Password) {
			writeError(w, http.StatusUnauthorized, errString("invalid email or password"))
			return
		}
		issueSessionCookie(w, issuer, user.ID, isProduction)
		writeJSON(w, http.StatusOK, map[string]any{"id": user.ID, "name": user.Name, "email": user.Email})
	}
}

func handleMe(store storage.Storage, issuer *auth.TokenIssuer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, errString("not authenticated"))
			return
		}

		user, err := store.GetUserByID(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"id": user.ID, "name": user.Name, "email": user.Email})
	}
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusNoContent)
}

type updateNameRequest struct {
	Name string `json:"name"`
}

func handleUpdateName(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, errString("not authenticated"))
			return
		}

		var req updateNameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, errString("name is required"))
			return
		}

		updated, err := store.UpdateUserName(r.Context(), userID, req.Name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"id": updated.ID, "name": updated.Name, "email": updated.Email})
	}
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func handleChangePassword(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, errString("not authenticated"))
			return
		}

		var req changePasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if len(req.NewPassword) < 8 {
			writeError(w, http.StatusBadRequest, errString("new password must be at least 8 characters"))
			return
		}

		user, err := store.GetUserByID(r.Context(), userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if !user.CheckPassword(req.CurrentPassword) {
			writeError(w, http.StatusUnauthorized, errString("current password is incorrect"))
			return
		}

		hashed, err := models.NewUser(user.Name, user.Email, req.NewPassword)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		if err := store.UpdateUserPassword(r.Context(), userID, hashed.PasswordHash); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func issueSessionCookie(w http.ResponseWriter, issuer *auth.TokenIssuer, userID int64, isProduction bool) {
	token, err := issuer.Issue(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isProduction,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
	})
}
