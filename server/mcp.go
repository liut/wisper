package server

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// CreateMcpServer creates the MCP server with tools
func (s *WebServer) CreateMcpServer() *server.MCPServer {
	srv := server.NewMCPServer(
		"webpawm",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Add web_search tool with dynamic schema based on available engines
	engines := s.getAvailableEngines()
	webSearchSchema := buildWebSearchSchema(engines)
	webSearchTool := mcp.NewToolWithRawSchema("web_search",
		"Search the web using various search engines. Supports single engine, multi-engine parallel search, and intelligent query expansion with deduplication by default.",
		webSearchSchema,
	)
	srv.AddTool(webSearchTool, mcp.NewStructuredToolHandler(s.handleWebSearchHandler))

	// Add web_fetch tool
	webFetchTool := mcp.NewTool("web_fetch",
		mcp.WithDescription("Fetch a website and return its content. Supports HTML to Markdown conversion for readability."),
		mcp.WithInputSchema[WebFetchParams](),
	)
	srv.AddTool(webFetchTool, mcp.NewStructuredToolHandler(s.handleWebFetchHandler))

	return srv
}

// handleWebSearchHandler is the handler for web_search tool
func (s *WebServer) handleWebSearchHandler(ctx context.Context, request mcp.CallToolRequest, args WebSearchParams) (webSearchOutput, error) {
	result, err := s.handleWebSearch(ctx, args)
	return *result, err
}

// buildWebSearchSchema creates the JSON schema for web_search tool with dynamic engine enums
func buildWebSearchSchema(engines []string) json.RawMessage {
	// Sort engines for deterministic enum order
	engines = slices.Sorted(slices.Values(engines))

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query",
			},
			"engine": map[string]any{
				"type":        "string",
				"description": "Single search engine to use (mutually exclusive with engines)",
				"enum":        engines,
			},
			"engines": map[string]any{
				"type":        "array",
				"description": "List of search engines to use (mutually exclusive with engine)",
				"items": map[string]any{
					"type": "string",
					"enum": engines,
				},
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 10)",
				"minimum":     1,
				"maximum":    50,
			},
			"language": map[string]any{
				"type":        "string",
				"description": "Language code for search results (e.g., 'en', 'zh')",
			},
			"arxiv_category": map[string]any{
				"type":        "string",
				"description": "Arxiv category for academic paper search (e.g., 'cs.AI', 'math.CO')",
			},
			"search_depth": map[string]any{
				"type":        "string",
				"description": "Search depth: 'quick' (1 query), 'normal' (2 queries), 'deep' (3 queries). Default: 'normal'",
				"enum":        []string{"quick", "normal", "deep"},
			},
			"include_academic": map[string]any{
				"type":        "boolean",
				"description": "Include academic papers from Arxiv (default: false)",
			},
			"auto_query_expand": map[string]any{
				"type":        "boolean",
				"description": "Automatically expand query with variations (news, academic) based on search_depth (default: true)",
			},
			"auto_deduplicate": map[string]any{
				"type":        "boolean",
				"description": "Automatically deduplicate results by URL (default: true)",
			},
		},
		"required":               []string{"query"},
		"additionalProperties":   false,
	}

	data, _ := json.Marshal(schema)
	return data
}

// handleWebFetchHandler is the handler for web_fetch tool
func (s *WebServer) handleWebFetchHandler(ctx context.Context, request mcp.CallToolRequest, args WebFetchParams) (webFetchOutput, error) {
	result, err := s.handleWebFetch(ctx, args)
	return *result, err
}
