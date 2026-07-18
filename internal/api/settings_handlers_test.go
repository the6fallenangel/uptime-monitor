package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateName(t *testing.T) {
	handler := newTestHandler(t)
	cookie := signupAndGetCookie(t, handler, "nameupdate@example.com")

	body := bytes.NewBufferString(`{"name":"New Name"}`)
	req := httptest.NewRequest(http.MethodPatch, "/me/name", body)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestChangePasswordRejectsWrongCurrentPassword(t *testing.T) {
	handler := newTestHandler(t)
	cookie := signupAndGetCookie(t, handler, "pwchange@example.com")

	body := bytes.NewBufferString(`{"currentPassword":"wrongpass","newPassword":"newpassword123"}`)
	req := httptest.NewRequest(http.MethodPost, "/me/password", body)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestChangePasswordSucceedsAndAllowsLoginWithNewPassword(t *testing.T) {
	handler := newTestHandler(t)
	cookie := signupAndGetCookie(t, handler, "pwchange2@example.com")

	body := bytes.NewBufferString(`{"currentPassword":"supersecret123","newPassword":"newpassword123"}`)
	req := httptest.NewRequest(http.MethodPost, "/me/password", body)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d, body %s", http.StatusNoContent, rec.Code, rec.Body.String())
	}

	loginBody := bytes.NewBufferString(`{"email":"pwchange2@example.com","password":"newpassword123"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/login", loginBody)
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Errorf("expected login with new password to succeed, got status %d", loginRec.Code)
	}
}
