package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/liut/wisper"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

func main() {
	cobra.OnInitialize(initConfig)

	rootCmd := &cobra.Command{
		Use:   "wisper",
		Short: "MCP web search server",
		Long: `Wisper is an MCP server that provides web search capabilities.
It supports multiple search engines including SearXNG, Google, Bing, and Arxiv.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Default: run stdio mode
			runStdioServer()
		},
	}

	// Persistent flags (available to all subcommands)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (default: ~/.wisper/config.json)")

	// web subcommand
	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Start HTTP and SSE server",
		Long:  "Start the MCP server over HTTP and SSE (Server-Sent Events)",
		Run:   runWebCommand,
	}
	webCmd.Flags().StringP("listen", "l", "", "HTTP listen address (e.g., localhost:8080)")
	viper.BindPFlag("listen_addr", webCmd.Flags().Lookup("listen"))

	// std subcommand
	stdCmd := &cobra.Command{
		Use:   "std",
		Short: "Start stdio server",
		Long:  "Start the MCP server in stdio mode (for use as an MCP tool)",
		Run:   runStdioCommand,
	}

	rootCmd.AddCommand(webCmd)
	rootCmd.AddCommand(stdCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func initConfig() {
	// If --config flag is provided, use it
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Default config file locations
		viper.SetConfigName("config")
		viper.AddConfigPath("$HOME/.wisper")
		viper.AddConfigPath(".")
	}

	// Environment variable settings
	viper.SetEnvPrefix("WISPER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("max_results", 10)

	// Read config file (non-fatal if not found)
	if err := viper.ReadInConfig(); err != nil {
		var viperErr viper.ConfigFileNotFoundError
		if !errors.As(err, &viperErr) {
			fmt.Fprintf(os.Stderr, "Warning: error reading config file: %v\n", err)
		}
	}
}

func getConfig() *wisper.Config {
	var config wisper.Config
	if err := viper.Unmarshal(&config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to unmarshal config: %v\n", err)
		os.Exit(1)
	}
	return &config
}

func runWebCommand(cmd *cobra.Command, args []string) {
	// Bind flag to viper
	viper.BindPFlag("listen_addr", cmd.Flags().Lookup("listen"))

	config := getConfig()
	// Ensure URIPrefix is loaded from viper (in case it was set via env var)
	config.URIPrefix = viper.GetString("uri_prefix")

	// Use flag value if provided, otherwise use config/env value
	listenAddr := viper.GetString("listen_addr")
	if listenAddr == "" {
		listenAddr = config.ListenAddr
	}

	if listenAddr == "" {
		fmt.Fprintf(os.Stderr, "Error: listen address is required. Use --listen flag or set listen_addr in config.\n")
		os.Exit(1)
	}

	config.ListenAddr = listenAddr
	startHTTPServer(config)
}

func runStdioCommand(cmd *cobra.Command, args []string) {
	runStdioServer()
}

func startHTTPServer(config *wisper.Config) {
	server := wisper.NewWebSearchServer(*config)
	mcpServer := server.CreateMcpServer()

	// Create handlers for different transport modes
	sseHandler := mcp.NewSSEHandler(func(request *http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	httpHandler := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	// Use ServeMux to handle different paths
	mux := http.NewServeMux()
	uriPrefix := config.URIPrefix

	// HTTP mode endpoint: prefix/mcp
	if uriPrefix != "" {
		mux.Handle(uriPrefix+"/mcp", httpHandler)
		mux.Handle(uriPrefix+"/mcp/sse", sseHandler)
	} else {
		mux.Handle("/mcp", httpHandler)
		mux.Handle("/mcp/sse", sseHandler)
	}

	// Print endpoints
	httpEndpoint := config.ListenAddr
	if uriPrefix != "" {
		httpEndpoint = config.ListenAddr + uriPrefix
	}

	fmt.Printf("Starting Wisper MCP server (HTTP and SSE mode)...\n")
	fmt.Printf("  Listen: %s\n", config.ListenAddr)
	fmt.Printf("  HTTP endpoint: http://%s/mcp\n", httpEndpoint)
	fmt.Printf("  SSE endpoint:  http://%s/mcp/sse\n", httpEndpoint)
	fmt.Printf("  SearXNG: %s\n", config.SearchXNGURL)
	googleEnabled := config.GoogleAPIKey != "" && config.GoogleCX != ""
	fmt.Printf("  Google: %v\n", googleEnabled)
	fmt.Printf("  Bing: %v\n", config.BingAPIKey != "")
	fmt.Printf("  Max Results: %d\n", config.MaxResults)

	if err := http.ListenAndServe(config.ListenAddr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func runStdioServer() {
	config := getConfig()
	server := wisper.NewWebSearchServer(*config)
	startStdioServer(server)
}

func startStdioServer(server *wisper.WebSearchServer) {
	mcpServer := server.CreateMcpServer()

	fmt.Printf("Starting Wisper MCP server (stdio mode)...\n")

	if err := mcpServer.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server error: %v", err)
	}
}
