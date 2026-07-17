package auth

import (
	"testing"
	"time"
)

func TestIssueAndVerify(t *testing.T) {
	issuer := NewTokenIssuer("test-secret", time.Hour)

	token, err := issuer.Issue(42)
	if err != nil {
		t.Fatalf("unexpected error issuing token: %v", err)
	}

	userID, err := issuer.Verify(token)
	if err != nil {
		t.Fatalf("unexpected error verifying token: %v", err)
	}
	if userID != 42 {
		t.Errorf("expected user id 42, got %d", userID)
	}
}

func TestVerifyRejectsTamperedToken(t *testing.T) {
	issuer := NewTokenIssuer("test-secret", time.Hour)

	token, _ := issuer.Issue(42)
	tampered := token[:len(token)-1] + "x" // flip the last character

	if _, err := issuer.Verify(tampered); err == nil {
		t.Errorf("expected error verifying tampered token, got nil")
	}
}

func TestVerifyRejectsTokenFromDifferentSecret(t *testing.T) {
	issuerA := NewTokenIssuer("secret-a", time.Hour)
	issuerB := NewTokenIssuer("secret-b", time.Hour)

	token, _ := issuerA.Issue(42)

	if _, err := issuerB.Verify(token); err == nil {
		t.Errorf("expected error verifying token signed with a different secret, got nil")
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	issuer := NewTokenIssuer("test-secret", -time.Hour) // already expired

	token, _ := issuer.Issue(42)

	if _, err := issuer.Verify(token); err == nil {
		t.Errorf("expected error verifying expired token, got nil")
	}
}
