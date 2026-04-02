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
			token, fromQuery := extractToken(r)

			if token != validToken || token == "" {
				unauthorized(w)
				return
			}

			// If token came from query, set cookie for future requests
			if fromQuery {
				const cookieMaxAge = 30 * 24 * 60 * 60 // 30 days
				http.SetCookie(w, &http.Cookie{
					Name:     "accounter_token",
					Value:    token,
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
					MaxAge:   cookieMaxAge,
				})
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractToken(r *http.Request) (string, bool) {
	if token := r.URL.Query().Get("token"); token != "" {
		return token, true
	}

	if cookie, err := r.Cookie("accounter_token"); err == nil && cookie.Value != "" {
		return cookie.Value, false
	}

	if token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return strings.TrimSpace(token), false
	}

	return "", false
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "unauthorized",
	})
}
