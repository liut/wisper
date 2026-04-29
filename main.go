package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/liut/webpawm/server"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	version = "dev"
)

func main() {
	cobra.OnInitialize(initConfig)

	rootCmd := &cobra.Command{
		Use:   "webpawm",
		Short: "MCP web search server",
		Long: `An MCP server that provides web search and web fetch capabilities.
It supports web search across multiple engines (SearXNG, Google, Bing, Brave, Arxiv) and web page content fetching with HTML to Markdown conversion.`,
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			// Default: run stdio mode
			runStdioServer()
		},
	}

	// Persistent flags (available to all subcommands)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (default: ~/.config/webpawm/config.json, fallback: ~/.webpawm/config.json)")

	// web subcommand
	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Start HTTP and SSE server",
		Long:  "Start the MCP server over HTTP and SSE (Server-Sent Events)",
		Run:   runWebCommand,
	}
	webCmd.Flags().StringP("listen", "l", "", "HTTP listen address (e.g., localhost:8087)")
	_ = viper.BindPFlag("listen_addr", webCmd.Flags().Lookup("listen"))

	// std subcommand
	stdCmd := &cobra.Command{
		Use:   "std",
		Short: "Start stdio server",
		Long:  "Start the MCP server in stdio mode (for use as an MCP tool)",
		Run:   runStdioCommand,
	}

	// gen-cfg subcommand
	genCfgCmd := &cobra.Command{
		Use:   "gen-cfg",
		Short: "Generate default config file",
		Long:  "Generate a default config file at ~/.config/webpawm/config.json",
		Run:   runGenCfgCommand,
	}
	genCfgCmd.Flags().StringP("output", "o", "", "Output path (default: ~/.config/webpawm/config.json)")

	rootCmd.AddCommand(webCmd)
	rootCmd.AddCommand(stdCmd)
	rootCmd.AddCommand(genCfgCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func initConfig() {
	var configPaths []string

	// If --config flag is provided, use it
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		configPaths = []string{cfgFile}
	} else {
		// Default config file locations (following XDG Base Directory Specification)
		viper.SetConfigName("config")
		// XDG config directory (preferred)
		viper.AddConfigPath("$HOME/.config/webpawm")
		// Legacy config directory (backward compatibility)
		viper.AddConfigPath("$HOME/.webpawm")
		viper.AddConfigPath(".")
		// Record the paths we're looking for
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configPaths = append(configPaths, filepath.Join(homeDir, ".config", "webpawm", "config.json"))
			configPaths = append(configPaths, filepath.Join(homeDir, ".webpawm", "config.json"))
		}
		configPaths = append(configPaths, "./config.json")
	}

	// Environment variable settings
	viper.SetEnvPrefix("WEBPAWM")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("max_results", 10)
	viper.SetDefault("log_level", "info")

	// Read config file (non-fatal if not found)
	if err := viper.ReadInConfig(); err != nil {
		var viperErr viper.ConfigFileNotFoundError
		if errors.As(err, &viperErr) {
			slog.Info("Config file not found, using environment variables and defaults", "searched", configPaths)
		} else {
			slog.Warn("Error reading config file, using environment variables and defaults", "file", viper.ConfigFileUsed(), "error", err)
		}
	} else {
		slog.Info("Config file loaded successfully", "file", viper.ConfigFileUsed())
	}
}

func getConfig() *server.Config {
	var config server.Config
	if err := viper.Unmarshal(&config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to unmarshal config: %v\n", err)
		os.Exit(1)
	}
	return &config
}

func runWebCommand(cmd *cobra.Command, args []string) {

	config := getConfig()
	// Ensure URIPrefix and LogLevel are loaded from viper (in case they were set via env var)
	config.URIPrefix = viper.GetString("uri_prefix")
	config.LogLevel = viper.GetString("log_level")

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

func runGenCfgCommand(cmd *cobra.Command, args []string) {
	outputPath, _ := cmd.Flags().GetString("output")

	if outputPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot find home directory: %v\n", err)
			os.Exit(1)
		}
		// Use XDG config directory as default
		outputPath = filepath.Join(homeDir, ".config", "webpawm", "config.json")
	}

	// Create directory if not exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot create directory: %v\n", err)
		os.Exit(1)
	}

	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		fmt.Printf("Config file already exists: %s\n", outputPath)
		fmt.Print("Overwrite? (y/N): ")
		var confirm string
		_, _ = fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			fmt.Println("Aborted.")
			os.Exit(0)
		}
	}

	// Generate default config
	defaultConfig := `{
  "searchxng_url": "",
  "google_api_key": "",
  "google_cx": "",
  "bing_api_key": "",
  "brave_api_key": "",
  "api_key": "",
  "max_results": 10,
  "default_engine": "",
  "listen_addr": "localhost:8087",
  "uri_prefix": "/mcp",
  "log_level": "info",
  "disable_sse": false
}
`

	if err := os.WriteFile(outputPath, []byte(defaultConfig), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot write config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Config file created: %s\n", outputPath)
}

func startHTTPServer(config *server.Config) {
	// Setup logger based on LogLevel
	logger := server.SetupLogger(config.LogLevel)

	srv := server.NewWebServer(*config)
	mcpServer := srv.CreateMcpServer()

	uriPrefix := strings.TrimSuffix(config.URIPrefix, "/")

	// Create StreamableHTTP server (always enabled)
	streamSvr := mcpserver.NewStreamableHTTPServer(mcpServer)

	// Use ServeMux to handle different paths
	mux := http.NewServeMux()

	// HTTP mode endpoint: prefix/mcp
	mux.Handle(uriPrefix+"/stream", streamSvr)
	if len(uriPrefix) == 0 { // old path
		mux.Handle(uriPrefix+"/mcp", streamSvr)
	}

	// SSE endpoint (optional)
	if !config.DisableSSE {
		sseSvr := mcpserver.NewSSEServer(mcpServer,
			mcpserver.WithStaticBasePath(uriPrefix),
		)
		mux.Handle(uriPrefix+"/sse", sseSvr.SSEHandler())
		mux.Handle(uriPrefix+"/message", sseSvr.MessageHandler())
	}

	// Wrap with API key auth, security headers and logging middleware
	var handler http.Handler = mux
	if config.APIKey != "" {
		handler = server.APIKeyAuthMiddleware(config.APIKey, handler, logger)
	}
	handler = server.LoggingMiddleware(logger, handler)
	handler = server.SecurityHeadersMiddleware(handler)

	// Print endpoints
	httpEndpoint := config.ListenAddr
	if uriPrefix != "" {
		httpEndpoint = config.ListenAddr + uriPrefix
	}

	fmt.Fprintf(os.Stderr, "Starting Webpawm MCP server (HTTP and SSE mode)...\n")
	fmt.Fprintf(os.Stderr, "  Listen: %s\n", config.ListenAddr)
	fmt.Fprintf(os.Stderr, "  Endpoint: http://%s/stream\n", httpEndpoint)
	if !config.DisableSSE {
		fmt.Fprintf(os.Stderr, "  Endpoint(SSE): http://%s/sse\n", httpEndpoint)
	}
	fmt.Fprintf(os.Stderr, "  Log Level: %s\n", config.LogLevel)
	fmt.Fprintf(os.Stderr, "  SearXNG: %s\n", config.SearchXNGURL)
	googleEnabled := config.GoogleAPIKey != "" && config.GoogleCX != ""
	fmt.Fprintf(os.Stderr, "  Google: %v\n", googleEnabled)
	fmt.Fprintf(os.Stderr, "  Bing: %v\n", config.BingAPIKey != "")
	fmt.Fprintf(os.Stderr, "  Brave: %v\n", config.BraveAPIKey != "")
	fmt.Fprintf(os.Stderr, "  Max Results: %d\n", config.MaxResults)

	httpServer := &http.Server{
		Addr:    config.ListenAddr,
		Handler: handler,
	}

	// Start server in a goroutine
	go func() {
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("Server failed", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "err", err)
	}
	slog.Info("Server stopped")
}

func runStdioServer() {
	config := getConfig()
	srv := server.NewWebServer(*config)
	startStdioServer(srv)
}

func startStdioServer(srv *server.WebServer) {
	mcpServer := srv.CreateMcpServer()

	fmt.Printf("Starting Webpawm MCP server (stdio mode)...\n")

	if err := mcpserver.ServeStdio(mcpServer); err != nil {
		slog.Error("Server failed", "err", err)
	}
}
