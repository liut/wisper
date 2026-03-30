package server

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
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
	sortedEngines := slices.Sorted(slices.Values(engines))
	enumOverrides := map[string][]string{
		"engine":  sortedEngines,
		"engines": sortedEngines,
	}
	webSearchTool := mcp.NewTool("web_search",
		mcp.WithDescription("Search the web using various search engines. Supports single engine, multi-engine parallel search, and intelligent query expansion with deduplication by default."),
		withInputSchemaWithEnums[WebSearchParams](enumOverrides),
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

// withInputSchemaWithEnums creates input schema with enum constraints for specific fields.
// It directly operates on *jsonschema.Schema object without unmarshaling to map.
func withInputSchemaWithEnums[T any](enumOverrides map[string][]string) mcp.ToolOption {
	return func(t *mcp.Tool) {
		schema, err := jsonschema.For[T](&jsonschema.ForOptions{IgnoreInvalidTypes: true})
		if err != nil {
			return
		}

		for fieldName, enumValues := range enumOverrides {
			if prop, ok := schema.Properties[fieldName]; ok {
				anyValues := make([]any, len(enumValues))
				for i, v := range enumValues {
					anyValues[i] = v
				}
				prop.Enum = anyValues
			}
		}

		mcpSchema, err := json.Marshal(schema)
		if err != nil {
			return
		}

		t.InputSchema.Type = ""
		t.RawInputSchema = json.RawMessage(mcpSchema)
	}
}

// handleWebFetchHandler is the handler for web_fetch tool
func (s *WebServer) handleWebFetchHandler(ctx context.Context, request mcp.CallToolRequest, args WebFetchParams) (webFetchOutput, error) {
	result, err := s.handleWebFetch(ctx, args)
	return *result, err
}
