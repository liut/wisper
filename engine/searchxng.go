package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SearchXNGEngine implements search using SearXNG instance
type SearchXNGEngine struct {
	baseURL string
	client  *http.Client
}

// NewSearchXNGEngine creates a new SearchXNG engine instance
func NewSearchXNGEngine(baseURL string) *SearchXNGEngine {
	baseURL = strings.TrimRight(baseURL, "/")
	return &SearchXNGEngine{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Name returns the engine name
func (s *SearchXNGEngine) Name() string {
	return "searchxng"
}

type searchXNGResponse struct {
	Results []searchXNGResult `json:"results"`
}

type searchXNGResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Engine  string `json:"engine"`
}

// Search performs a search using the SearXNG instance
func (s *SearchXNGEngine) Search(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("q", strings.Join(query.Queries, " "))
	params.Set("format", "json")

	if query.Language != "" {
		params.Set("language", query.Language)
	}

	searchURL := fmt.Sprintf("%s/search?%s", s.baseURL, params.Encode())
	slog.Debug("searchxng search request", "url", searchURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Wisper/1.0)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("searchxng search failed: %d - %s", resp.StatusCode, string(body))
	}

	var response searchXNGResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]SearchResult, 0, len(response.Results))
	for i, item := range response.Results {
		if query.MaxResults > 0 && i >= query.MaxResults {
			break
		}

		content := item.Content
		if content == "" {
			content = "Result from " + item.Engine
		}

		results = append(results, SearchResult{
			Title:   item.Title,
			Link:    item.URL,
			Content: content,
		})
	}

	return results, nil
}
