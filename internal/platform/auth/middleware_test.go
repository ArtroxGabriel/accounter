package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtroxGabriel/accounter/internal/platform/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBearerMiddleware(t *testing.T) {
	validToken := "supersecret"
	middleware := auth.BearerMiddleware(validToken)

	// Create a dummy next handler that always returns 200 OK
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	handlerToTest := middleware(nextHandler)

	tests := []struct {
		name           string
		authHeader     string
		queryToken     string
		cookieToken    string
		expectedStatus int
		expectedBody   string
		expectCookie   bool
	}{
		{
			name:           "valid token calls next handler",
			authHeader:     "Bearer supersecret",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"unauthorized"}`,
		},
		{
			name:           "wrong token",
			authHeader:     "Bearer wrongtoken",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"unauthorized"}`,
		},
		{
			name:           "malformed header no prefix",
			authHeader:     "supersecret",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"unauthorized"}`,
		},
		{
			name:           "empty token value",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"unauthorized"}`,
		},
		{
			name:           "different prefix",
			authHeader:     "Basic supersecret",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"unauthorized"}`,
		},
		{
			name:           "valid token in query param",
			queryToken:     "supersecret",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
			expectCookie:   true,
		},
		{
			name:           "valid token in cookie",
			cookieToken:    "supersecret",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "invalid token in query falls back to auth header",
			queryToken:     "wrong",
			authHeader:     "Bearer supersecret",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"unauthorized"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/", nil)
			require.NoError(t, err)

			if tt.queryToken != "" {
				q := req.URL.Query()
				q.Add("token", tt.queryToken)
				req.URL.RawQuery = q.Encode()
			}

			if tt.cookieToken != "" {
				req.AddCookie(&http.Cookie{Name: "accounter_token", Value: tt.cookieToken})
			}

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			handlerToTest.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectCookie {
				cookies := rr.Result().Cookies()
				require.Len(t, cookies, 1)
				assert.Equal(t, "accounter_token", cookies[0].Name)
				assert.Equal(t, tt.queryToken, cookies[0].Value)
			}
			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, tt.expectedBody, rr.Body.String())
			} else {
				assert.JSONEq(t, tt.expectedBody, rr.Body.String())
			}
		})
	}
}
