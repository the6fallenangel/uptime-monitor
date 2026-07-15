package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/the6fallenangel/uptime-monitor/internal/models"
)

func TestCheckUp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(2 * time.Second)
	monitor := models.Monitor{ID: 1, URL: server.URL}

	check := c.Check(context.Background(), monitor)

	if check.Status != models.StatusUp {
		t.Errorf("expected status %q, got %q", models.StatusUp, check.Status)
	}
	if check.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, check.StatusCode)
	}
	if check.Error != "" {
		t.Errorf("expected no error, got %q", check.Error)
	}
	if check.ResponseTime <= 0 {
		t.Errorf("expected positive response time, got %v", check.ResponseTime)
	}
}

func TestCheckDownOnServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := New(2 * time.Second)
	monitor := models.Monitor{ID: 1, URL: server.URL}

	check := c.Check(context.Background(), monitor)

	if check.Status != models.StatusDown {
		t.Errorf("expected status %q, got %q", models.StatusDown, check.Status)
	}
	if check.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, check.StatusCode)
	}
}

func TestCheckDownOnTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(10 * time.Millisecond)
	monitor := models.Monitor{ID: 1, URL: server.URL}

	check := c.Check(context.Background(), monitor)

	if check.Status != models.StatusDown {
		t.Errorf("expected status %q, got %q", models.StatusDown, check.Status)
	}
	if check.Error == "" {
		t.Errorf("expected a timeout error message, got empty string")
	}
}

func TestCheckDownOnInvalidURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	unreachableURL := server.URL
	server.Close()

	c := New(2 * time.Second)
	monitor := models.Monitor{ID: 1, URL: unreachableURL}

	check := c.Check(context.Background(), monitor)

	if check.Status != models.StatusDown {
		t.Errorf("expected status %q, got %q", models.StatusDown, check.Status)
	}
	if check.Error == "" {
		t.Errorf("expected an error message for unreachable host, got empty string")
	}
}
