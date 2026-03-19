package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/liut/webpawm/engine"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// handleWebSearch handles the web_search tool
func (s *WebServer) handleWebSearch(ctx context.Context, params WebSearchParams) (*mcp.CallToolResult, error) {
	if params.Query == "" {
		return nil, errors.New("query is required")
	}

	engineName := params.Engine
	if engineName == "" {
		engineName = s.defaultEngine
	}

	searchEngine, exists := s.engines[engineName]
	if !exists {
		return nil, fmt.Errorf("search engine '%s' is not available", engineName)
	}

	maxRes := params.MaxResults
	if maxRes <= 0 {
		maxRes = s.maxResults
	}

	results, err := searchEngine.Search(ctx, engine.SearchQuery{
		Queries:       []string{params.Query},
		MaxResults:    maxRes,
		Language:      params.Language,
		ArxivCategory: params.ArxivCategory,
	})
	if err != nil {
		slog.Warn("search failed", "engine", engineName, "query", params.Query, "error", err)
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var formattedResults []map[string]any
	for i, result := range results {
		formattedResults = append(formattedResults, map[string]any{
			"index":   i + 1,
			"title":   result.Title,
			"link":    result.Link,
			"snippet": result.Content,
		})
	}

	response := SearchResultResponse{
		Engine:  engineName,
		Query:   params.Query,
		Count:   len(results),
		Results: formattedResults,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(responseJSON)},
		},
	}, nil
}

// handleMultiSearch handles the multi_search tool
func (s *WebServer) handleMultiSearch(ctx context.Context, params MultiSearchParams) (*mcp.CallToolResult, error) {
	if params.Query == "" {
		return nil, errors.New("query is required")
	}

	engineNames := params.Engines
	if len(engineNames) == 0 {
		for name := range s.engines {
			engineNames = append(engineNames, name)
		}
	}

	maxResults := params.MaxResultsPerEngine
	if maxResults <= 0 {
		maxResults = 5
	}

	allResults := make(map[string][]map[string]any)

	for _, engineName := range engineNames {
		searchEngine, exists := s.engines[engineName]
		if !exists {
			continue
		}

		results, err := searchEngine.Search(ctx, engine.SearchQuery{
			Queries:    []string{params.Query},
			MaxResults: maxResults,
		})
		if err != nil {
			allResults[engineName] = []map[string]any{
				{"error": err.Error()},
			}
			continue
		}

		var engineResults []map[string]any
		for i, result := range results {
			engineResults = append(engineResults, map[string]any{
				"index":   i + 1,
				"title":   result.Title,
				"link":    result.Link,
				"snippet": result.Content,
			})
		}

		allResults[engineName] = engineResults
	}

	response := map[string]any{
		"query":   params.Query,
		"engines": engineNames,
		"results": allResults,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(responseJSON)},
		},
	}, nil
}

// handleSmartSearch handles the smart_search tool
func (s *WebServer) handleSmartSearch(ctx context.Context, params SmartSearchParams) (*mcp.CallToolResult, error) {
	if params.Question == "" {
		return nil, errors.New("question is required")
	}

	searchDepth := params.SearchDepth
	if searchDepth == "" {
		searchDepth = "normal"
	}

	queries := generateSearchQueries(params.Question, searchDepth)

	var allResults []engine.SearchResult
	searchSummary := map[string]any{
		"original_question": params.Question,
		"search_queries":    queries,
		"engines_used":      []string{},
	}

	for _, q := range queries {
		engineName := s.determineEngine(q.Type, params.IncludeAcademic)
		if engineName == "" {
			continue
		}

		searchEngine := s.engines[engineName]

		results, err := searchEngine.Search(ctx, engine.SearchQuery{
			Queries:    []string{q.Query},
			MaxResults: q.MaxResults,
		})
		if err == nil {
			allResults = append(allResults, results...)

			enginesUsed, _ := searchSummary["engines_used"].([]string)
			if !slices.Contains(enginesUsed, engineName) {
				searchSummary["engines_used"] = append(enginesUsed, engineName)
			}
		}
	}

	uniqueResults := s.removeDuplicates(allResults)

	var formattedResults []map[string]any
	for i, result := range uniqueResults {
		formattedResults = append(formattedResults, map[string]any{
			"index":   i + 1,
			"title":   result.Title,
			"link":    result.Link,
			"snippet": result.Content,
		})
	}

	response := map[string]any{
		"summary":        searchSummary,
		"total_results":  len(uniqueResults),
		"results":        formattedResults,
		"search_time":    time.Now().Format(time.RFC3339),
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(responseJSON)},
		},
	}, nil
}
