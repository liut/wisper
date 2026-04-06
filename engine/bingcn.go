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

// BingCNEngine implements search using Bing CN (free, no API key required)
type BingCNEngine struct {
	client *http.Client
}

// NewBingCNEngine creates a new Bing CN engine instance
func NewBingCNEngine() *BingCNEngine {
	return &BingCNEngine{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the engine name
func (b *BingCNEngine) Name() string {
	return "bingcn"
}

// Search performs a search using Bing CN
func (b *BingCNEngine) Search(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("q", strings.Join(query.Queries, " "))
	params.Set("setlang", "zh-CN")

	searchURL := fmt.Sprintf("https://api.bing.com/qsonhs.aspx?%s", params.Encode())
	slog.Debug("bingcn search request", "url", searchURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; BingCN/1.0)")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bing CN search failed: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		AS struct {
			Results []struct {
				Title string `json:"Title"`
				Url   string `json:"Url"`
				Desc  string `json:"Desc"`
			} `json:"Results"`
		} `json:"AS"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]SearchResult, 0, len(result.AS.Results))
	for _, item := range result.AS.Results {
		if query.MaxResults > 0 && len(results) >= query.MaxResults {
			break
		}
		results = append(results, SearchResult{
			Title:   item.Title,
			Link:    item.Url,
			Content: item.Desc,
		})
	}

	return results, nil
}
