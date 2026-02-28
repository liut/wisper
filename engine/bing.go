package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BingEngine implements search using Bing Search API
type BingEngine struct {
	apiKey string
	client *http.Client
}

// NewBingEngine creates a new Bing engine instance
func NewBingEngine(apiKey string) *BingEngine {
	return &BingEngine{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the engine name
func (b *BingEngine) Name() string {
	return "bing"
}

// Search performs a search using Bing Search API
func (b *BingEngine) Search(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	if b.apiKey == "" {
		return nil, fmt.Errorf("Bing API key is required")
	}

	params := url.Values{}
	params.Set("q", strings.Join(query.Queries, " "))
	params.Set("count", fmt.Sprintf("%d", query.MaxResults))

	searchURL := fmt.Sprintf("https://api.bing.microsoft.com/v7.0/search?%s", params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Ocp-Apim-Subscription-Key", b.apiKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bing search failed: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		WebPages struct {
			Value []struct {
				Name         string `json:"name"`
				URL          string `json:"url"`
				Snippet      string `json:"snippet"`
			} `json:"value"`
		} `json:"webPages"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]SearchResult, 0, len(result.WebPages.Value))
	for _, item := range result.WebPages.Value {
		results = append(results, SearchResult{
			Title:   item.Name,
			Link:    item.URL,
			Content: item.Snippet,
		})
	}

	return results, nil
}
