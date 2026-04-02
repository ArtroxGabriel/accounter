package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

// BearerMiddleware validates that requests contain a valid Bearer token.
// If the token matches the configured static token, the request proceeds.
// Otherwise, it returns a 401 Unauthorized JSON response.
func BearerMiddleware(validToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				unauthorized(w)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != validToken || token == "" {
				unauthorized(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "unauthorized",
	})
}
