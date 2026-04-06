package server

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/liut/webpawm/engine"
)

// handleWebSearch handles the unified web_search tool
func (s *WebServer) handleWebSearch(ctx context.Context, params WebSearchParams) (*WebSearchResponse, error) {
	if params.Query == "" {
		return nil, errors.New("query is required")
	}

	// Set defaults for optional boolean params
	if params.SearchDepth == "" {
		params.SearchDepth = "normal"
	}

	// Resolve engines to use
	enginesToUse := s.resolveEngines(params)

	// Generate queries based on auto_query_expand setting
	queries := s.generateQueries(params)

	// Execute searches
	allResults, enginesUsed := s.executeSearches(ctx, enginesToUse, queries, params)

	// Deduplicate if enabled
	rawCount := len(allResults)
	if params.AutoDeduplicate {
		allResults = s.removeDuplicates(allResults)
	}
	slog.Info("search done", "count", rawCount, "engines", enginesToUse)

	// Format response
	return s.formatResponse(params, queries, enginesUsed, allResults, rawCount), nil
}

// resolveEngines determines which engines to use based on params
func (s *WebServer) resolveEngines(params WebSearchParams) []string {
	// If specific engines are requested, use those
	if len(params.Engines) > 0 {
		return params.Engines
	}

	// If a single engine is specified, use that
	if params.Engine != "" {
		return []string{params.Engine}
	}

	// If auto_query_expand is enabled, use all available engines
	if params.AutoQueryExpand {
		engines := make([]string, 0, len(s.engines))
		for name := range s.engines {
			engines = append(engines, name)
		}
		return engines
	}

	// Default to the server's default engine
	return []string{s.defaultEngine}
}

// generateQueries generates search queries based on auto_query_expand setting
func (s *WebServer) generateQueries(params WebSearchParams) []searchQuery {
	if !params.AutoQueryExpand {
		return []searchQuery{{
			Query:      params.Query,
			MaxResults: s.resolveMaxResults(params),
			Type:       "general",
		}}
	}

	return generateSearchQueries(params.Query, params.SearchDepth)
}

// resolveMaxResults resolves the max results to use
func (s *WebServer) resolveMaxResults(params WebSearchParams) int {
	if params.MaxResults > 0 {
		return params.MaxResults
	}
	return s.maxResults
}

// executeSearches runs searches across engines and queries
func (s *WebServer) executeSearches(ctx context.Context, engines []string, queries []searchQuery, params WebSearchParams) ([]engine.SearchResult, []string) {
	var allResults []engine.SearchResult
	enginesUsed := make([]string, 0)

	for _, engineName := range engines {
		searchEngine, exists := s.engines[engineName]
		if !exists {
			continue
		}

		for _, q := range queries {
			// Determine if this query type should use this engine
			if !s.shouldUseEngine(engineName, q.Type, params.IncludeAcademic) {
				continue
			}

			maxResults := q.MaxResults
			if maxResults <= 0 {
				maxResults = s.resolveMaxResults(params)
			}

			results, err := searchEngine.Search(ctx, engine.SearchQuery{
				Queries:       []string{q.Query},
				MaxResults:    maxResults,
				Language:      params.Language,
				ArxivCategory: params.ArxivCategory,
			})
			if err != nil {
				slog.Warn("search failed", "engine", engineName, "query", q.Query, "error", err)
				continue
			}

			allResults = append(allResults, results...)
			if !slices.Contains(enginesUsed, engineName) {
				enginesUsed = append(enginesUsed, engineName)
			}
		}
	}

	return allResults, enginesUsed
}

// shouldUseEngine determines if an engine should be used for a query type
func (s *WebServer) shouldUseEngine(engineName, queryType string, includeAcademic bool) bool {
	if queryType == "academic" {
		return includeAcademic && engineName == "arxiv"
	}
	return true
}

// formatResponse creates the unified response structure
func (s *WebServer) formatResponse(params WebSearchParams, queries []searchQuery, enginesUsed []string, results []engine.SearchResult, rawCount int) *WebSearchResponse {
	searchQueries := make([]string, len(queries))
	for i, q := range queries {
		searchQueries[i] = q.Query
	}

	searchSummary := SearchSummary{
		OriginalQuery:      params.Query,
		SearchQueries:      searchQueries,
		EnginesUsed:        enginesUsed,
		SearchDepth:        params.SearchDepth,
		TotalRawResults:    rawCount,
		TotalUniqueResults: len(results),
	}

	formattedResults := make([]SearchResult, len(results))
	for i, result := range results {
		formattedResults[i] = SearchResult{
			Index:   i + 1,
			Title:   result.Title,
			Link:    result.Link,
			Snippet: result.Content,
		}
	}

	return &WebSearchResponse{
		Summary:      searchSummary,
		TotalResults: len(results),
		Results:      formattedResults,
		SearchTime:   time.Now().Format(time.RFC3339),
	}
}
