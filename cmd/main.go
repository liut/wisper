package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/liut/wisper"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	listen    = flag.String("listen", "", "HTTP listen address (e.g., localhost:8080). If not provided, runs in stdio mode")
	searchxng = flag.String("searchxng", "", "SearXNG base URL (e.g., https://searchx.ng)")
	googleKey = flag.String("google-key", "", "Google Custom Search API key")
	googleCX  = flag.String("google-cx", "", "Google Search Engine ID")
	bingKey   = flag.String("bing-key", "", "Bing Search API key")
	maxResults = flag.Int("max-results", 10, "Maximum number of search results")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "This program runs a MCP web search server.\n")
		fmt.Fprintf(os.Stderr, "If no options are provided, it runs in stdio mode (for use as an MCP tool).\n")
		fmt.Fprintf(os.Stderr, "If --listen is provided, it starts an HTTP SSE server.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()

	config := wisper.Config{
		SearchXNGURL: *searchxng,
		GoogleAPIKey: *googleKey,
		GoogleCX:     *googleCX,
		BingAPIKey:   *bingKey,
		MaxResults:   *maxResults,
	}
	server := wisper.NewWebSearchServer(config)

	// If --listen is provided, start HTTP SSE server
	if *listen != "" {
		startHTTPServer(server, *listen)
		return
	}

	// Otherwise, run in stdio mode
	startStdioServer(server)
}

func startHTTPServer(server *wisper.WebSearchServer, addr string) {
	mcpServer := server.CreateMcpServer()

	handler := mcp.NewSSEHandler(func(request *http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	fmt.Printf("Starting Wisper MCP server (HTTP SSE mode)...\n")
	fmt.Printf("  Listen: %s\n", addr)
	fmt.Printf("  SearXNG: %s\n", *searchxng)
	fmt.Printf("  Google: %s\n", *googleKey != "" && *googleCX != "")
	fmt.Printf("  Bing: %s\n", *bingKey != "")
	fmt.Printf("  Max Results: %d\n", *maxResults)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func startStdioServer(server *wisper.WebSearchServer) {
	mcpServer := server.CreateMcpServer()

	fmt.Printf("Starting Wisper MCP server (stdio mode)...\n")
	fmt.Printf("  SearXNG: %s\n", *searchxng)
	fmt.Printf("  Google: %s\n", *googleKey != "" && *googleCX != "")
	fmt.Printf("  Bing: %s\n", *bingKey != "")
	fmt.Printf("  Max Results: %d\n", *maxResults)

	if err := mcpServer.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server error: %v", err)
	}
}
