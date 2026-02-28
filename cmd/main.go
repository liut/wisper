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
	printVersion = flag.Bool("version", false, "print version information")
	printHelp    = flag.Bool("help", false, "print help information")
	printUsage  = flag.Bool("usage", false, "print usage information")
	listen      = flag.String("listen", "", "HTTP listen address (e.g., localhost:8080). If not provided, runs in stdio mode")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "This program runs a MCP web search server.\n")
		fmt.Fprintf(os.Stderr, "If no options are provided, it runs in stdio mode (for use as an MCP tool).\n")
		fmt.Fprintf(os.Stderr, "If --listen is provided, it starts an HTTP SSE server.\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, wisper.UsageString("WISPER"))
	}
	flag.Parse()

	// Load configuration from environment variables first
	config, err := wisper.LookupConfig("WISPER")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Override with command-line flags if provided
	if *listen != "" {
		config.ListenAddr = *listen
	}

	// Create server
	server := wisper.NewWebSearchServer(*config)

	// Determine run mode based on --listen flag
	if config.ListenAddr != "" {
		startHTTPServer(server, config)
		return
	}

	// Otherwise, run in stdio mode
	startStdioServer(server)
}

func startHTTPServer(server *wisper.WebSearchServer, config *wisper.Config) {
	mcpServer := server.CreateMcpServer()

	handler := mcp.NewSSEHandler(func(request *http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	fmt.Printf("Starting Wisper MCP server (HTTP SSE mode)...\n")
	fmt.Printf("  Listen: %s\n", config.ListenAddr)
	fmt.Printf("  SearXNG: %s\n", config.SearchXNGURL)
	fmt.Printf("  Google: %s\n", config.GoogleAPIKey != "" && config.GoogleCX != "")
	fmt.Printf("  Bing: %s\n", config.BingAPIKey != "")
	fmt.Printf("  Max Results: %d\n", config.MaxResults)

	if err := http.ListenAndServe(config.ListenAddr, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func startStdioServer(server *wisper.WebSearchServer) {
	mcpServer := server.CreateMcpServer()

	fmt.Printf("Starting Wisper MCP server (stdio mode)...\n")

	if err := mcpServer.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server error: %v", err)
	}
}
