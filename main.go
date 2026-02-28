package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/liut/wisper/server"
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

	// gen-cfg subcommand
	genCfgCmd := &cobra.Command{
		Use:   "gen-cfg",
		Short: "Generate default config file",
		Long:  "Generate a default config file at ~/.wisper/config.json",
		Run:   runGenCfgCommand,
	}
	genCfgCmd.Flags().StringP("output", "o", "", "Output path (default: ~/.wisper/config.json)")

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
	viper.SetDefault("log_level", "info")

	// Read config file (non-fatal if not found)
	if err := viper.ReadInConfig(); err != nil {
		var viperErr viper.ConfigFileNotFoundError
		if !errors.As(err, &viperErr) {
			fmt.Fprintf(os.Stderr, "Warning: error reading config file: %v\n", err)
		}
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
	// Bind flag to viper
	viper.BindPFlag("listen_addr", cmd.Flags().Lookup("listen"))

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
		outputPath = filepath.Join(homeDir, ".wisper", "config.json")
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
		fmt.Scanln(&confirm)
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
  "max_results": 10,
  "default_engine": "",
  "listen_addr": "localhost:8080",
  "uri_prefix": "",
  "log_level": "info"
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
	logger := setupLogger(config.LogLevel)

	server := server.NewWebSearchServer(*config)
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

	// Wrap with logging middleware
	handler := loggingMiddleware(logger, mux)

	// Print endpoints
	httpEndpoint := config.ListenAddr
	if uriPrefix != "" {
		httpEndpoint = config.ListenAddr + uriPrefix
	}

	fmt.Printf("Starting Wisper MCP server (HTTP and SSE mode)...\n")
	fmt.Printf("  Listen: %s\n", config.ListenAddr)
	fmt.Printf("  HTTP endpoint: http://%s/mcp\n", httpEndpoint)
	fmt.Printf("  SSE endpoint:  http://%s/mcp/sse\n", httpEndpoint)
	fmt.Printf("  Log Level: %s\n", config.LogLevel)
	fmt.Printf("  SearXNG: %s\n", config.SearchXNGURL)
	googleEnabled := config.GoogleAPIKey != "" && config.GoogleCX != ""
	fmt.Printf("  Google: %v\n", googleEnabled)
	fmt.Printf("  Bing: %v\n", config.BingAPIKey != "")
	fmt.Printf("  Max Results: %d\n", config.MaxResults)

	if err := http.ListenAndServe(config.ListenAddr, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// setupLogger creates an slog.Logger based on the LogLevel config
func setupLogger(level string) *slog.Logger {
	var slogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn", "warning":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	}))
}

// loggingMiddleware logs HTTP requests using slog
func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration", time.Since(start).Milliseconds(),
			"client", r.RemoteAddr,
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func runStdioServer() {
	config := getConfig()
	server := server.NewWebSearchServer(*config)
	startStdioServer(server)
}

func startStdioServer(server *server.WebSearchServer) {
	mcpServer := server.CreateMcpServer()

	fmt.Printf("Starting Wisper MCP server (stdio mode)...\n")

	if err := mcpServer.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server error: %v", err)
	}
}
