package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		requestHeader  string
		requestValue   string
		wantStatus     int
		wantAuthHeader bool
	}{
		{
			name:           "valid X-API-Key header",
			apiKey:         "test-secret-key",
			requestHeader:  "X-API-Key",
			requestValue:   "test-secret-key",
			wantStatus:     http.StatusOK,
			wantAuthHeader: false,
		},
		{
			name:           "valid Bearer token",
			apiKey:         "test-secret-key",
			requestHeader:  "Authorization",
			requestValue:   "Bearer test-secret-key",
			wantStatus:     http.StatusOK,
			wantAuthHeader: false,
		},
		{
			name:           "invalid X-API-Key",
			apiKey:         "test-secret-key",
			requestHeader:  "X-API-Key",
			requestValue:   "wrong-key",
			wantStatus:     http.StatusUnauthorized,
			wantAuthHeader: true,
		},
		{
			name:           "invalid Bearer token",
			apiKey:         "test-secret-key",
			requestHeader:  "Authorization",
			requestValue:   "Bearer wrong-key",
			wantStatus:     http.StatusUnauthorized,
			wantAuthHeader: true,
		},
		{
			name:           "missing X-API-Key",
			apiKey:         "test-secret-key",
			requestHeader:  "",
			requestValue:   "",
			wantStatus:     http.StatusUnauthorized,
			wantAuthHeader: true,
		},
		{
			name:           "empty Authorization header",
			apiKey:         "test-secret-key",
			requestHeader:  "Authorization",
			requestValue:   "",
			wantStatus:     http.StatusUnauthorized,
			wantAuthHeader: true,
		},
		{
			name:           "Authorization without Bearer prefix",
			apiKey:         "test-secret-key",
			requestHeader:  "Authorization",
			requestValue:   "test-secret-key",
			wantStatus:     http.StatusUnauthorized,
			wantAuthHeader: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := apiKeyAuthMiddleware(tt.apiKey, nextHandler, nil)

			req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
			if tt.requestHeader != "" {
				req.Header.Set(tt.requestHeader, tt.requestValue)
			}

			rec := httptest.NewRecorder()
			middleware.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}

			wwwAuth := rec.Header().Get("WWW-Authenticate")
			if tt.wantAuthHeader && wwwAuth == "" {
				t.Errorf("expected WWW-Authenticate header, got empty")
			}
			if !tt.wantAuthHeader && wwwAuth != "" {
				t.Errorf("did not expect WWW-Authenticate header, got %q", wwwAuth)
			}
		})
	}
}

func TestAPIKeyAuthMiddleware_Passthrough(t *testing.T) {
	apiKey := "valid-key"

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusOK)
	})

	middleware := apiKeyAuthMiddleware(apiKey, nextHandler, nil)

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("X-API-Key", apiKey)

	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	customHeader := rec.Header().Get("X-Custom-Header")
	if customHeader != "custom-value" {
		t.Errorf("X-Custom-Header not passed through, got %q", customHeader)
	}
}
