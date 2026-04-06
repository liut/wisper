package main

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
		Long: `Wisper is an MCP server that provides web search capabilities.
It supports multiple search engines including SearXNG, Google, Bing, and Arxiv.`,
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			// Default: run stdio mode
			runStdioServer()
		},
	}

	// Persistent flags (available to all subcommands)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (default: ~/.webpawm/config.json)")

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
		Long:  "Generate a default config file at ~/.webpawm/config.json",
		Run:   runGenCfgCommand,
	}
	genCfgCmd.Flags().StringP("output", "o", "", "Output path (default: ~/.webpawm/config.json)")

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
		// Default config file locations
		viper.SetConfigName("config")
		viper.AddConfigPath("$HOME/.webpawm")
		viper.AddConfigPath(".")
		// Record the paths we're looking for
		homeDir, err := os.UserHomeDir()
		if err == nil {
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
	// Bind flag to viper
	_ = viper.BindPFlag("listen_addr", cmd.Flags().Lookup("listen"))

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
		outputPath = filepath.Join(homeDir, ".webpawm", "config.json")
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

	srv := server.NewWebServer(*config)
	mcpServer := srv.CreateMcpServer()

	// Create HTTP/SSE server
	httpServer := mcpserver.NewStreamableHTTPServer(mcpServer)

	// Use ServeMux to handle different paths
	mux := http.NewServeMux()
	uriPrefix := config.URIPrefix

	// HTTP mode endpoint: prefix/mcp
	if uriPrefix != "" {
		mux.Handle(uriPrefix+"/mcp", httpServer)
	} else {
		mux.Handle("/mcp", httpServer)
	}

	// Wrap with API key auth and logging middleware
	var handler http.Handler = mux
	if config.APIKey != "" {
		handler = apiKeyAuthMiddleware(config.APIKey, handler, logger)
	}
	handler = loggingMiddleware(logger, handler)

	// Print endpoints
	httpEndpoint := config.ListenAddr
	if uriPrefix != "" {
		httpEndpoint = config.ListenAddr + uriPrefix
	}

	fmt.Printf("Starting Webpawm MCP server (HTTP and SSE mode)...\n")
	fmt.Printf("  Listen: %s\n", config.ListenAddr)
	fmt.Printf("  HTTP endpoint: http://%s/mcp\n", httpEndpoint)
	fmt.Printf("  Log Level: %s\n", config.LogLevel)
	fmt.Printf("  SearXNG: %s\n", config.SearchXNGURL)
	googleEnabled := config.GoogleAPIKey != "" && config.GoogleCX != ""
	fmt.Printf("  Google: %v\n", googleEnabled)
	fmt.Printf("  Bing: %v\n", config.BingAPIKey != "")
	fmt.Printf("  Brave: %v\n", config.BraveAPIKey != "")
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

	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
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
			"duration", time.Since(start).String(),
			"client", r.RemoteAddr,
		)
	})
}

// apiKeyAuthMiddleware validates X-API-Key header or Authorization: Bearer header
func apiKeyAuthMiddleware(validAPIKey string, next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientAPIKey := r.Header.Get("X-API-Key")
		if clientAPIKey == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				clientAPIKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if subtle.ConstantTimeCompare([]byte(clientAPIKey), []byte(validAPIKey)) != 1 {
			if logger != nil {
				logger.Warn("authentication failed",
					"reason", "invalid key",
					"client", r.RemoteAddr,
					"path", r.URL.Path,
				)
			}
			w.Header().Set("WWW-Authenticate", `Bearer realm="API", error="invalid_token"`)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Unauthorized", "message": "API key required. Use X-API-Key header or Authorization: Bearer <key>"}`))
			return
		}

		next.ServeHTTP(w, r)
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
	srv := server.NewWebServer(*config)
	startStdioServer(srv)
}

func startStdioServer(srv *server.WebServer) {
	mcpServer := srv.CreateMcpServer()

	fmt.Printf("Starting Webpawm MCP server (stdio mode)...\n")

	if err := mcpserver.ServeStdio(mcpServer); err != nil {
		log.Printf("Server error: %v", err)
	}
}
