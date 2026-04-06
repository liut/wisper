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

// GoogleEngine implements search using Google Custom Search API
type GoogleEngine struct {
	apiKey string
	cx     string
	client *http.Client
}

// NewGoogleEngine creates a new Google engine instance
func NewGoogleEngine(apiKey, cx string) *GoogleEngine {
	return &GoogleEngine{
		apiKey: apiKey,
		cx:     cx,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the engine name
func (g *GoogleEngine) Name() string {
	return "google"
}

// Search performs a search using Google Custom Search API
func (g *GoogleEngine) Search(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	if g.apiKey == "" || g.cx == "" {
		return nil, fmt.Errorf("Google API key and Search Engine ID are required")
	}

	params := url.Values{}
	params.Set("key", g.apiKey)
	params.Set("cx", g.cx)
	params.Set("q", strings.Join(query.Queries, " "))
	params.Set("num", fmt.Sprintf("%d", query.MaxResults))

	if query.Language != "" {
		params.Set("lr", "lang_"+query.Language)
	}

	searchURL := fmt.Sprintf("https://www.googleapis.com/customsearch/v1?%s", params.Encode())
	slog.Debug("google search request", "url", searchURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google search failed: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]SearchResult, 0, len(result.Items))
	for _, item := range result.Items {
		results = append(results, SearchResult{
			Title:   item.Title,
			Link:    item.Link,
			Content: item.Snippet,
		})
	}

	return results, nil
}
