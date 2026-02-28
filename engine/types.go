package engine

import (
	"context"
)

// SearchResult represents a single search result
type SearchResult struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Content string `json:"content"`
}

// SearchQuery represents search parameters
type SearchQuery struct {
	Queries       []string
	MaxResults    int
	Language      string
	ArxivCategory string
}

// Engine is the interface for search engines
type Engine interface {
	// Search performs a search and returns results
	Search(ctx context.Context, query SearchQuery) ([]SearchResult, error)
	// Name returns the engine name
	Name() string
}
