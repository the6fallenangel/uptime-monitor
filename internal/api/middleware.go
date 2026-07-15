package api

import (
	"context"
	"net/http"

	"github.com/the6fallenangel/uptime-monitor/internal/auth"
)

type contextKey string

const userIDKey contextKey = "userID"

func requireAuth(issuer *auth.TokenIssuer, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, errString("not authenticated"))
			return
		}

		userID, err := issuer.Verify(cookie.Value)
		if err != nil {
			writeError(w, http.StatusUnauthorized, errString("not authenticated"))
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}
