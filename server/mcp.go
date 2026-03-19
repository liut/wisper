package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateMcpServer creates the MCP server with tools
func (s *WebServer) CreateMcpServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "webpawm",
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

	// Add web_fetch tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "web_fetch",
		Description: "Fetch a website and return its content. Supports HTML to Markdown conversion for readability.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "URL of the website to fetch",
				},
				"max_length": map[string]any{
					"type":        "integer",
					"description": "Maximum number of characters to return (default: 5000)",
					"minimum":     1,
					"maximum":     999999,
				},
				"start_index": map[string]any{
					"type":        "integer",
					"description": "Start content from this character index (default: 0)",
					"minimum":     0,
				},
				"raw": map[string]any{
					"type":        "boolean",
					"description": "If true, returns the raw HTML including <script> and <style> blocks (default: false)",
				},
			},
			"required": []string{"url"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, params WebFetchParams) (*mcp.CallToolResult, any, error) {
		result, err := s.handleWebFetch(ctx, params)
		return result, nil, err
	})

	return server
}

// Run starts the MCP server over SSE
func (s *WebServer) Run(addr string) error {
	mcpServer := s.CreateMcpServer()

	handler := mcp.NewSSEHandler(func(request *http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	fmt.Printf("Webpawm MCP server starting on %s\n", addr)
	return http.ListenAndServe(addr, handler)
}
