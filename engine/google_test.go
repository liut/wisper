package engine

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGoogleEngine_Name(t *testing.T) {
	engine := NewGoogleEngine("test-key", "test-cx")
	if engine.Name() != "google" {
		t.Errorf("expected name 'google', got '%s'", engine.Name())
	}
}

func TestGoogleEngine_Search_MissingCredentials(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		cx     string
	}{
		{
			name:   "missing api key",
			apiKey: "",
			cx:     "test-cx",
		},
		{
			name:   "missing cx",
			apiKey: "test-key",
			cx:     "",
		},
		{
			name:   "missing both",
			apiKey: "",
			cx:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewGoogleEngine(tt.apiKey, tt.cx)
			_, err := engine.Search(context.Background(), SearchQuery{
				Queries:    []string{"test"},
				MaxResults: 10,
			})
			if err == nil {
				t.Error("expected error for missing credentials, got nil")
			}
		})
	}
}

func TestGoogleEngine_Search_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    400,
				"message": "Request contains an invalid argument.",
				"errors": []map[string]interface{}{
					{
						"message": "Request contains an invalid argument.",
						"domain":  "global",
						"reason":  "badRequest",
					},
				},
				"status": "INVALID_ARGUMENT",
			},
		})
	}))
	defer server.Close()

	engine := &GoogleEngine{
		apiKey: "test-key",
		cx:     "test-cx",
		client: server.Client(),
	}

	_, err := engine.Search(context.Background(), SearchQuery{
		Queries:    []string{"Agent 翻译为智能体 由来 背景"},
		MaxResults: 10,
	})
	if err == nil {
		t.Fatal("expected error for server error response, got nil")
	}
}

type mockTransport struct {
	handler func(w http.ResponseWriter, r *http.Request)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	m.handler(recorder, req)
	return recorder.Result(), nil
}

func TestGoogleEngine_Search_Success(t *testing.T) {
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("q")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"title":   "Test Result",
					"link":    "https://example.com/1",
					"snippet": "This is a test snippet",
				},
				{
					"title":   "Test Result 2",
					"link":    "https://example.com/2",
					"snippet": "Another test snippet",
				},
			},
		})
	}))
	defer server.Close()

	engine := &GoogleEngine{
		apiKey: "test-key",
		cx:     "test-cx",
		client: &http.Client{
			Transport: &mockTransport{
				handler: func(w http.ResponseWriter, r *http.Request) {
					r.URL.Scheme = "http"
					r.URL.Host = server.Listener.Addr().String()
					server.Config.Handler.ServeHTTP(w, r)
				},
			},
		},
	}

	results, err := engine.Search(context.Background(), SearchQuery{
		Queries:    []string{"test query"},
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedQuery != "test query" {
		t.Errorf("expected query 'test query', got '%s'", receivedQuery)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Title != "Test Result" {
		t.Errorf("expected title 'Test Result', got '%s'", results[0].Title)
	}
	if results[0].Link != "https://example.com/1" {
		t.Errorf("expected link 'https://example.com/1', got '%s'", results[0].Link)
	}
}

func TestGoogleEngine_Search_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	engine := &GoogleEngine{
		apiKey: "test-key",
		cx:     "test-cx",
		client: &http.Client{
			Transport: &mockTransport{
				handler: func(w http.ResponseWriter, r *http.Request) {
					r.URL.Scheme = "http"
					r.URL.Host = server.Listener.Addr().String()
					server.Config.Handler.ServeHTTP(w, r)
				},
			},
		},
	}

	results, err := engine.Search(context.Background(), SearchQuery{
		Queries:    []string{"nonexistent query"},
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestGoogleEngine_Search_ChineseQuery(t *testing.T) {
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query().Get("q")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"title":   "Agent智能体翻译由来",
					"link":    "https://example.com/agent",
					"snippet": "关于Agent翻译为智能体的背景介绍",
				},
			},
		})
	}))
	defer server.Close()

	engine := &GoogleEngine{
		apiKey: "test-key",
		cx:     "test-cx",
		client: &http.Client{
			Transport: &mockTransport{
				handler: func(w http.ResponseWriter, r *http.Request) {
					r.URL.Scheme = "http"
					r.URL.Host = server.Listener.Addr().String()
					server.Config.Handler.ServeHTTP(w, r)
				},
			},
		},
	}

	results, err := engine.Search(context.Background(), SearchQuery{
		Queries:    []string{"Agent 翻译为智能体 由来 背景"},
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedQuery != "Agent 翻译为智能体 由来 背景" {
		t.Errorf("expected Chinese query, got '%s'", receivedQuery)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestGoogleEngine_Search_WithLanguage(t *testing.T) {
	var receivedLang string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedLang = r.URL.Query().Get("lr")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	engine := &GoogleEngine{
		apiKey: "test-key",
		cx:     "test-cx",
		client: &http.Client{
			Transport: &mockTransport{
				handler: func(w http.ResponseWriter, r *http.Request) {
					r.URL.Scheme = "http"
					r.URL.Host = server.Listener.Addr().String()
					server.Config.Handler.ServeHTTP(w, r)
				},
			},
		},
	}

	_, err := engine.Search(context.Background(), SearchQuery{
		Queries:    []string{"test"},
		MaxResults: 10,
		Language:   "zh-CN",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedLang != "lang_zh-CN" {
		t.Errorf("expected lr 'lang_zh-CN', got '%s'", receivedLang)
	}
}

func TestGoogleEngine_Search_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid json{"))
	}))
	defer server.Close()

	engine := &GoogleEngine{
		apiKey: "test-key",
		cx:     "test-cx",
		client: &http.Client{
			Transport: &mockTransport{
				handler: func(w http.ResponseWriter, r *http.Request) {
					r.URL.Scheme = "http"
					r.URL.Host = server.Listener.Addr().String()
					server.Config.Handler.ServeHTTP(w, r)
				},
			},
		},
	}

	_, err := engine.Search(context.Background(), SearchQuery{
		Queries:    []string{"test"},
		MaxResults: 10,
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON response, got nil")
	}
}

func TestGoogleEngine_Search_NetworkError(t *testing.T) {
	engine := &GoogleEngine{
		apiKey: "test-key",
		cx:     "test-cx",
		client: &http.Client{},
	}

	_, err := engine.Search(context.Background(), SearchQuery{
		Queries:    []string{"test"},
		MaxResults: 10,
	})
	if err == nil {
		t.Fatal("expected error for network failure, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		if err.Error() == "Google API key and Search Engine ID are required" {
			t.Skip("skipping network error test due to missing credentials")
		}
	}
}

func TestGoogleEngine_Search_HTTPStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	engine := &GoogleEngine{
		apiKey: "test-key",
		cx:     "test-cx",
		client: &http.Client{
			Transport: &mockTransport{
				handler: func(w http.ResponseWriter, r *http.Request) {
					r.URL.Scheme = "http"
					r.URL.Host = server.Listener.Addr().String()
					server.Config.Handler.ServeHTTP(w, r)
				},
			},
		},
	}

	_, err := engine.Search(context.Background(), SearchQuery{
		Queries:    []string{"test"},
		MaxResults: 10,
	})
	if err == nil {
		t.Fatal("expected error for HTTP error status, got nil")
	}
}

func TestNewGoogleEngine(t *testing.T) {
	engine := NewGoogleEngine("my-api-key", "my-cx")
	if engine.apiKey != "my-api-key" {
		t.Errorf("expected apiKey 'my-api-key', got '%s'", engine.apiKey)
	}
	if engine.cx != "my-cx" {
		t.Errorf("expected cx 'my-cx', got '%s'", engine.cx)
	}
	if engine.client == nil {
		t.Error("expected non-nil HTTP client")
	}
}

func TestGoogleEngine_Integration(t *testing.T) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	cx := os.Getenv("GOOGLE_CX")
	if apiKey == "" || cx == "" {
		t.Skip("skipping integration test: GOOGLE_API_KEY or GOOGLE_CX not set")
	}

	engine := NewGoogleEngine(apiKey, cx)

	results, err := engine.Search(t.Context(), SearchQuery{
		Queries:    []string{"Agent 翻译为智能体 由来 背景"},
		MaxResults: 5,
	})
	if err != nil {
		t.Fatalf("Google search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected at least one search result")
	}

	for i, r := range results {
		t.Logf("result %d: title=%q link=%q", i, r.Title, r.Link)
	}
}
