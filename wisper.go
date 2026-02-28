package wisper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/liut/wisper/engine"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config for the web search server.
// Can be loaded from environment variables using ProcessConfig.
// Supports configuration from file (~/.wisper/config.json) and environment variables.
type Config struct {
	SearchXNGURL  string `mapstructure:"searchxng_url" envconfig:"SEARCHXNG_URL"`    // SearXNG base URL (e.g., https://searchx.ng)
	GoogleAPIKey  string `mapstructure:"google_api_key" envconfig:"GOOGLE_API_KEY"`  // Google Custom Search API key
	GoogleCX      string `mapstructure:"google_cx" envconfig:"GOOGLE_CX"`            // Google Search Engine ID
	BingAPIKey    string `mapstructure:"bing_api_key" envconfig:"BING_API_KEY"`      // Bing Search API key
	MaxResults    int    `mapstructure:"max_results" envconfig:"MAX_RESULTS"`         // Default max results (default: 10)
	DefaultEngine string `mapstructure:"default_engine" envconfig:"DEFAULT_ENGINE"`  // Default search engine
	ListenAddr    string `mapstructure:"listen_addr" envconfig:"LISTEN_ADDR"`         // HTTP listen address
	URIPrefix     string `mapstructure:"uri_prefix" envconfig:"URI_PREFIX"`          // URI prefix for HTTP endpoints
}

// WebSearchServer represents the MCP web search server
type WebSearchServer struct {
	engines       map[string]engine.Engine
	defaultEngine string
	maxResults    int
}

// NewWebSearchServer creates a new web search server
func NewWebSearchServer(config Config) *WebSearchServer {
	engines := make(map[string]engine.Engine)

	// Add Google engine if configured
	if config.GoogleAPIKey != "" && config.GoogleCX != "" {
		engines["google"] = engine.NewGoogleEngine(config.GoogleAPIKey, config.GoogleCX)
	}

	// Add Bing engine if configured
	if config.BingAPIKey != "" {
		engines["bing"] = engine.NewBingEngine(config.BingAPIKey)
	}

	// Add Bing CN engine (free, always available)
	engines["bingcn"] = engine.NewBingCNEngine()

	// Add SearXNG engine if configured
	if config.SearchXNGURL != "" {
		engines["searchxng"] = engine.NewSearchXNGEngine(config.SearchXNGURL)
	}

	// Add Arxiv engine (always available)
	engines["arxiv"] = engine.NewArxivEngine()

	defaultEngine := config.DefaultEngine
	if defaultEngine == "" && len(engines) > 0 {
		for name := range engines {
			defaultEngine = name
			break
		}
	}

	maxResults := config.MaxResults
	if maxResults <= 0 {
		maxResults = 10
	}

	return &WebSearchServer{
		engines:       engines,
		defaultEngine: defaultEngine,
		maxResults:    maxResults,
	}
}

// WebSearchParams represents the parameters for web_search tool
type WebSearchParams struct {
	Query         string `json:"query"`
	Engine        string `json:"engine"`
	MaxResults    int    `json:"max_results"`
	Language      string `json:"language"`
	ArxivCategory string `json:"arxiv_category"`
}

// MultiSearchParams represents the parameters for multi_search tool
type MultiSearchParams struct {
	Query               string   `json:"query"`
	Engines             []string `json:"engines"`
	MaxResultsPerEngine int      `json:"max_results_per_engine"`
}

// SmartSearchParams represents the parameters for smart_search tool
type SmartSearchParams struct {
	Question        string `json:"question"`
	SearchDepth     string `json:"search_depth"`
	IncludeAcademic bool   `json:"include_academic"`
}

// SearchResultResponse represents the response for search results
type SearchResultResponse struct {
	Engine  string                   `json:"engine"`
	Query   string                   `json:"query"`
	Count   int                      `json:"count"`
	Results []map[string]interface{} `json:"results"`
}

// handleWebSearch handles the web_search tool
func (s *WebSearchServer) handleWebSearch(ctx context.Context, params WebSearchParams) (*mcp.CallToolResult, error) {
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
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var formattedResults []map[string]interface{}
	for i, result := range results {
		formattedResults = append(formattedResults, map[string]interface{}{
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
func (s *WebSearchServer) handleMultiSearch(ctx context.Context, params MultiSearchParams) (*mcp.CallToolResult, error) {
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

	allResults := make(map[string][]map[string]interface{})

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
			allResults[engineName] = []map[string]interface{}{
				{"error": err.Error()},
			}
			continue
		}

		var engineResults []map[string]interface{}
		for i, result := range results {
			engineResults = append(engineResults, map[string]interface{}{
				"index":   i + 1,
				"title":   result.Title,
				"link":    result.Link,
				"snippet": result.Content,
			})
		}

		allResults[engineName] = engineResults
	}

	response := map[string]interface{}{
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
func (s *WebSearchServer) handleSmartSearch(ctx context.Context, params SmartSearchParams) (*mcp.CallToolResult, error) {
	if params.Question == "" {
		return nil, errors.New("question is required")
	}

	searchDepth := params.SearchDepth
	if searchDepth == "" {
		searchDepth = "normal"
	}

	queries := generateSearchQueries(params.Question, searchDepth)

	var allResults []engine.SearchResult
	searchSummary := map[string]interface{}{
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

	var formattedResults []map[string]interface{}
	for i, result := range uniqueResults {
		formattedResults = append(formattedResults, map[string]interface{}{
			"index":   i + 1,
			"title":   result.Title,
			"link":    result.Link,
			"snippet": result.Content,
		})
	}

	response := map[string]interface{}{
		"summary":       searchSummary,
		"total_results": len(uniqueResults),
		"results":       formattedResults,
		"search_time":   time.Now().Format(time.RFC3339),
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

// searchQuery represents a search query with parameters
type searchQuery struct {
	Query      string
	MaxResults int
	Type       string // "general", "academic", "news"
}

// generateSearchQueries creates search queries based on the user's question and depth
func generateSearchQueries(question, depth string) []searchQuery {
	queries := []searchQuery{}

	baseQueries := 1
	switch depth {
	case "quick":
		baseQueries = 1
	case "normal":
		baseQueries = 2
	case "deep":
		baseQueries = 3
	}

	queries = append(queries, searchQuery{
		Query:      question,
		MaxResults: 10,
		Type:       "general",
	})

	if baseQueries >= 2 {
		queries = append(queries, searchQuery{
			Query:      question + " latest news",
			MaxResults: 5,
			Type:       "news",
		})
	}

	if baseQueries >= 3 {
		queries = append(queries, searchQuery{
			Query:      question + " research papers",
			MaxResults: 5,
			Type:       "academic",
		})
	}

	return queries
}

// determineEngine selects the appropriate search engine for a query
func (s *WebSearchServer) determineEngine(queryType string, includeAcademic bool) string {
	if queryType == "academic" && includeAcademic {
		if _, ok := s.engines["arxiv"]; ok {
			return "arxiv"
		}
	}

	if _, ok := s.engines["searchxng"]; ok {
		return "searchxng"
	}

	for name := range s.engines {
		return name
	}

	return ""
}

// removeDuplicates removes duplicate search results based on URL
func (s *WebSearchServer) removeDuplicates(results []engine.SearchResult) []engine.SearchResult {
	seen := make(map[string]bool, len(results))
	var unique []engine.SearchResult

	for _, result := range results {
		if !seen[result.Link] {
			seen[result.Link] = true
			unique = append(unique, result)
		}
	}

	return unique
}

// getAvailableEngines returns a list of available search engine names
func (s *WebSearchServer) getAvailableEngines() []string {
	names := make([]string, 0, len(s.engines))
	for name := range s.engines {
		names = append(names, name)
	}
	return names
}

// CreateMcpServer creates the MCP server with tools
func (s *WebSearchServer) CreateMcpServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "wisper",
		Version: "1.0.0",
	}, nil)

	// Add web_search tool
		mcp.AddTool(server, &mcp.Tool{
		Name:        "web_search",
		Description: "Search the web using various search engines (SearXNG, Arxiv, Google, Bing)",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The search query",
				},
				"engine": map[string]any{
					"type":        "string",
					"description": "Search engine to use",
					"enum":        s.getAvailableEngines(),
				},
				"max_results": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results to return",
					"minimum":     1,
					"maximum":     50,
				},
				"language": map[string]any{
					"type":        "string",
					"description": "Language code for search results (e.g., 'en', 'zh')",
				},
				"arxiv_category": map[string]any{
					"type":        "string",
					"description": "Arxiv category for academic paper search (e.g., 'cs.AI', 'math.CO')",
				},
			},
			"required": []string{"query"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, params WebSearchParams) (*mcp.CallToolResult, any, error) {
		result, err := s.handleWebSearch(ctx, params)
		return result, nil, err
	})

	// Add multi_search tool
		mcp.AddTool(server, &mcp.Tool{
		Name:        "multi_search",
		Description: "Search across multiple search engines simultaneously",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The search query",
				},
				"engines": map[string]any{
					"type":        "array",
					"description": "List of search engines to use",
					"items": map[string]any{
						"type": "string",
						"enum": s.getAvailableEngines(),
					},
				},
				"max_results_per_engine": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results per engine",
					"minimum":     1,
					"maximum":     20,
				},
			},
			"required": []string{"query"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, params MultiSearchParams) (*mcp.CallToolResult, any, error) {
		result, err := s.handleMultiSearch(ctx, params)
		return result, nil, err
	})

	// Add smart_search tool
		mcp.AddTool(server, &mcp.Tool{
		Name:        "smart_search",
		Description: "Intelligently search the web with query optimization and result aggregation",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"question": map[string]any{
					"type":        "string",
					"description": "The user's question or search intent",
				},
				"search_depth": map[string]any{
					"type":        "string",
					"description": "Search depth: 'quick' (1-2 queries), 'normal' (3-5 queries), 'deep' (5-10 queries)",
					"enum":        []string{"quick", "normal", "deep"},
				},
				"include_academic": map[string]any{
					"type":        "boolean",
					"description": "Whether to include academic papers from Arxiv",
				},
			},
			"required": []string{"question"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, params SmartSearchParams) (*mcp.CallToolResult, any, error) {
		result, err := s.handleSmartSearch(ctx, params)
		return result, nil, err
	})

	return server
}

// Run starts the MCP server over SSE
func (s *WebSearchServer) Run(addr string) error {
	mcpServer := s.CreateMcpServer()

	handler := mcp.NewSSEHandler(func(request *http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	fmt.Printf("Wisper MCP server starting on %s\n", addr)
	return http.ListenAndServe(addr, handler)
}

// MustNewWebSearchServer creates a new web search server with error handling
func MustNewWebSearchServer(searchXNGURL string, maxResults int) *WebSearchServer {
	config := Config{
		SearchXNGURL: searchXNGURL,
		MaxResults:   maxResults,
	}
	return NewWebSearchServer(config)
}
