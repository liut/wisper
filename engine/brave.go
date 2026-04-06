package engine

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// BraveEngine implements search using Brave Search API
type BraveEngine struct {
	apiKey string
	client *http.Client
}

// NewBraveEngine creates a new Brave engine instance
func NewBraveEngine(apiKey string) *BraveEngine {
	return &BraveEngine{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the engine name
func (b *BraveEngine) Name() string {
	return "brave"
}

// Search performs a search using Brave Search API
func (b *BraveEngine) Search(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	if b.apiKey == "" {
		return nil, fmt.Errorf("Brave API key is required")
	}

	searchTerm := query.Queries[0]
	if len(query.Queries) > 1 {
		searchTerm = query.Queries[0]
	}

	params := url.Values{}
	params.Set("q", searchTerm)
	params.Set("count", fmt.Sprintf("%d", query.MaxResults))

	searchURL := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?%s", params.Encode())
	slog.Debug("brave search request", "url", searchURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Subscription-Token", b.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("User-Agent", "webpawm/1.0")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Brave search failed: %d - %s", resp.StatusCode, string(body))
	}

	body := resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		body, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer body.Close()
	}

	var result braveSearchResponse
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Web == nil || len(result.Web.Results) == 0 {
		return nil, nil
	}

	results := make([]SearchResult, 0, len(result.Web.Results))
	for _, item := range result.Web.Results {
		results = append(results, SearchResult{
			Title:   item.Title,
			Link:    item.URL,
			Content: item.Description,
		})
	}

	return results, nil
}

// braveSearchResponse represents the Brave Search API response
type braveSearchResponse struct {
	Web *braveWebResults `json:"web"`
}

type braveWebResults struct {
	Results []braveSearchResult `json:"results"`
}

type braveSearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}
