package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/the6fallenangel/uptime-monitor/internal/auth"
	"github.com/the6fallenangel/uptime-monitor/internal/checker"
	"github.com/the6fallenangel/uptime-monitor/internal/notifier"
	"github.com/the6fallenangel/uptime-monitor/internal/scheduler"
	"github.com/the6fallenangel/uptime-monitor/internal/storage"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()

	store := storage.NewTestStorage(t)

	issuer := auth.NewTokenIssuer("test-secret", time.Hour)
	chk := checker.New(2 * time.Second)
	sched := scheduler.New(store, chk, notifier.NewLogNotifier(), 2)
	sched.SetRootContext(context.Background())

	mux := http.NewServeMux()
	RegisterRoutes(mux, store, sched, issuer, false)
	return mux
}

func signupAndGetCookie(t *testing.T, handler http.Handler, email string) *http.Cookie {
	t.Helper()

	body := bytes.NewBufferString(`{"name":"Test","email":"` + email + `","password":"supersecret123"}`)
	req := httptest.NewRequest(http.MethodPost, "/signup", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("signup failed: status %d, body %s", rec.Code, rec.Body.String())
	}

	for _, c := range rec.Result().Cookies() {
		if c.Name == "session_token" {
			return c
		}
	}
	t.Fatal("no session cookie returned from signup")
	return nil
}

func TestSignupAndLogin(t *testing.T) {
	handler := newTestHandler(t)

	body := bytes.NewBufferString(`{"name":"Ali","email":"ali@example.com","password":"supersecret123"}`)
	req := httptest.NewRequest(http.MethodPost, "/signup", body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	loginBody := bytes.NewBufferString(`{"email":"ali@example.com","password":"supersecret123"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/login", loginBody)
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, loginRec.Code)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	handler := newTestHandler(t)

	signupBody := bytes.NewBufferString(`{"name":"Ali","email":"ali2@example.com","password":"supersecret123"}`)
	signupReq := httptest.NewRequest(http.MethodPost, "/signup", signupBody)
	handler.ServeHTTP(httptest.NewRecorder(), signupReq)

	loginBody := bytes.NewBufferString(`{"email":"ali2@example.com","password":"wrongpassword"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/login", loginBody)
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, loginRec.Code)
	}
}

func TestSignupDuplicateEmail(t *testing.T) {
	handler := newTestHandler(t)

	body := `{"name":"Ali","email":"dup@example.com","password":"supersecret123"}`

	req1 := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(body))
	handler.ServeHTTP(httptest.NewRecorder(), req1)

	req2 := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(body))
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, rec2.Code)
	}
}

func TestMonitorsRequireAuth(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/monitors", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestUserCannotAccessAnotherUsersMonitor(t *testing.T) {
	handler := newTestHandler(t)

	cookieA := signupAndGetCookie(t, handler, "usera@example.com")
	cookieB := signupAndGetCookie(t, handler, "userb@example.com")

	createBody := bytes.NewBufferString(`{"name":"A's Monitor","url":"https://example.com","interval":"30s"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/monitors", createBody)
	createReq.AddCookie(cookieA)
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body %s", http.StatusCreated, createRec.Code, createRec.Body.String())
	}

	var created map[string]any
	json.NewDecoder(createRec.Body).Decode(&created)
	monitorID := int64(created["id"].(float64))

	getReq := httptest.NewRequest(http.MethodGet, "/monitors/"+jsonInt(monitorID), nil)
	getReq.AddCookie(cookieB)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusNotFound {
		t.Errorf("expected status %d when accessing another user's monitor, got %d", http.StatusNotFound, getRec.Code)
	}
}

func jsonInt(id int64) string {
	b, _ := json.Marshal(id)
	return string(b)
}
