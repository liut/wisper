package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// disableFetchValidation disables SSRF URL validation for tests using local HTTP servers.
func disableFetchValidation() { validateFetchURL = func(_ string) error { return nil } }

func testWebServer() *WebServer {
	return &WebServer{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func TestHandleWebFetch_URLValidation(t *testing.T) {
	disableFetchValidation()
	s := testWebServer()

	tests := []struct {
		name    string
		params  WebFetchParams
		wantErr error
	}{
		{
			name:    "empty URL returns error",
			params:  WebFetchParams{URL: ""},
			wantErr: errors.New("url is required"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.HandleWebFetch(context.Background(), tt.params)
			if err == nil && tt.wantErr != nil {
				t.Errorf("expected error %v, got nil", tt.wantErr)
			}
			if err != nil && tt.wantErr != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestHandleWebFetch_MaxLengthValidation(t *testing.T) {
	disableFetchValidation()
	s := testWebServer()

	tests := []struct {
		name    string
		params  WebFetchParams
		wantErr error
	}{
		{
			name:    "zero max_length uses default",
			params:  WebFetchParams{URL: "http://example.com", MaxLength: 0},
			wantErr: nil,
		},
		{
			name:    "negative max_length returns error",
			params:  WebFetchParams{URL: "http://example.com", MaxLength: -1},
			wantErr: errors.New("max_length must be between 1 and 999999"),
		},
		{
			name:    "max_length too large returns error",
			params:  WebFetchParams{URL: "http://example.com", MaxLength: 1000000},
			wantErr: errors.New("max_length must be between 1 and 999999"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.HandleWebFetch(context.Background(), tt.params)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr.Error() {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestHandleWebFetch_StartIndexValidation(t *testing.T) {
	disableFetchValidation()
	s := testWebServer()

	tests := []struct {
		name    string
		params  WebFetchParams
		wantErr error
	}{
		{
			name:    "negative start_index returns error",
			params:  WebFetchParams{URL: "http://example.com", StartIndex: -1},
			wantErr: errors.New("start_index must be >= 0"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.HandleWebFetch(context.Background(), tt.params)
			if err == nil {
				t.Errorf("expected error %v, got nil", tt.wantErr)
			} else if err.Error() != tt.wantErr.Error() {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestHandleWebFetch_SuccessfulFetch(t *testing.T) {
	disableFetchValidation()
	s := testWebServer()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	params := WebFetchParams{
		URL:       server.URL,
		MaxLength: 5000,
	}

	result, err := s.HandleWebFetch(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if !strings.Contains(result.Content, "Hello, World!") {
		t.Errorf("expected content to contain 'Hello, World!', got: %s", result.Content)
	}
}

func TestHandleWebFetch_Truncation(t *testing.T) {
	disableFetchValidation()
	s := testWebServer()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("This is a very long content that should be truncated."))
	}))
	defer server.Close()

	params := WebFetchParams{
		URL:       server.URL,
		MaxLength: 10,
	}

	result, err := s.HandleWebFetch(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Truncated {
		t.Errorf("expected truncated to be true, got: %v", result.Truncated)
	}
}

func TestHandleWebFetch_StartIndexBeyondContent(t *testing.T) {
	disableFetchValidation()
	s := testWebServer()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Short content"))
	}))
	defer server.Close()

	params := WebFetchParams{
		URL:        server.URL,
		MaxLength:  5000,
		StartIndex: 100,
	}

	result, err := s.HandleWebFetch(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "No more content available") {
		t.Errorf("expected 'No more content available' message, got: %s", result.Content)
	}
}

func TestHandleWebFetch_HTTPError(t *testing.T) {
	disableFetchValidation()
	s := testWebServer()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	params := WebFetchParams{
		URL: server.URL,
	}

	result, err := s.HandleWebFetch(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Error == "" {
		t.Errorf("expected error message in response, got empty string")
	}
}

func TestHandleWebFetch_RawHTML(t *testing.T) {
	disableFetchValidation()
	s := testWebServer()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body><p>Hello, World!</p></body></html>"))
	}))
	defer server.Close()

	params := WebFetchParams{
		URL: server.URL,
		Raw: true,
	}

	result, err := s.HandleWebFetch(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ContentType != "raw" {
		t.Errorf("expected content type 'raw', got: %s", result.ContentType)
	}
}

func TestHandleWebFetch_ConvertedHTML(t *testing.T) {
	disableFetchValidation()
	s := testWebServer()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body><h1>Test Page</h1><p>This is a test paragraph with some content.</p></body></html>"))
	}))
	defer server.Close()

	params := WebFetchParams{
		URL: server.URL,
		Raw: false,
	}

	result, err := s.HandleWebFetch(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ContentType != "markdown" {
		t.Errorf("expected content type 'markdown', got: %s", result.ContentType)
	}
}

func TestWebFetchParams_JSON(t *testing.T) {
	tests := []struct {
		name   string
		params WebFetchParams
	}{
		{
			name: "full params",
			params: WebFetchParams{
				URL:        "https://example.com",
				MaxLength:  10000,
				StartIndex: 0,
				Raw:        false,
			},
		},
		{
			name: "minimal params",
			params: WebFetchParams{
				URL: "https://example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.params)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded WebFetchParams
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.URL != tt.params.URL {
				t.Errorf("URL mismatch: got %v, want %v", decoded.URL, tt.params.URL)
			}
		})
	}
}
