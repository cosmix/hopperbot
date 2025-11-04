package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rudderlabs/hopperbot/internal/slack"
	"github.com/rudderlabs/hopperbot/pkg/cache"
	"github.com/rudderlabs/hopperbot/pkg/config"
	"github.com/rudderlabs/hopperbot/pkg/constants"
	"github.com/rudderlabs/hopperbot/pkg/health"
	"github.com/rudderlabs/hopperbot/pkg/metrics"
	"github.com/rudderlabs/hopperbot/pkg/middleware"
	"go.uber.org/zap"
)

// Build information set via ldflags at compile time.
// Example: go build -ldflags "-X main.version=1.0.0 -X main.commit=abc123 -X main.buildTime=2024-01-01T00:00:00Z"
var (
	version   = "dev"     // Application version (e.g., "1.0.0", "v1.2.3")
	commit    = "unknown" // Git commit hash (short or full)
	buildTime = "unknown" // Build timestamp in RFC3339 format
)

// VersionInfo contains build and version information.
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

func main() {
	// Create production logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load configuration", zap.Error(err))
	}

	// Initialize metrics
	m := metrics.Init()
	logger.Info("metrics initialized")

	// Initialize Slack handler
	handler := slack.NewHandler(cfg, logger)
	handler.SetMetrics(m)

	logger.Info("initializing bot and fetching client list from Notion")
	if err := handler.Initialize(); err != nil {
		logger.Fatal("failed to initialize handler", zap.Error(err))
	}
	logger.Info("bot initialization complete")

	// Initialize cache manager for periodic and manual cache refresh
	cacheMgr := cache.NewManager(handler, m, logger, cfg.CacheRefreshInterval)
	handler.SetCacheManager(cacheMgr)
	cacheMgr.Start()
	logger.Info("cache manager started",
		zap.Duration("refresh_interval", cfg.CacheRefreshInterval),
	)

	// Initialize health manager
	healthMgr := health.NewManager(logger)

	// Register liveness check (basic server health)
	healthMgr.RegisterLivenessCheck("server", health.AlwaysHealthyChecker())

	// Register readiness checks (dependencies)
	healthMgr.RegisterReadinessCheck("notion_api", health.NotionHealthChecker(func(ctx context.Context) error {
		return handler.NotionClient().HealthCheck(ctx)
	}))

	healthMgr.RegisterReadinessCheck("client_cache", health.ClientCacheChecker(
		handler.GetClientCount,
		10, // Expect at least 10 clients as a sanity check
	))

	logger.Info("health checks registered")

	// Setup HTTP handlers with middleware
	// Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Health check endpoints
	http.HandleFunc("/health", healthMgr.LivenessHandler())
	http.HandleFunc("/ready", healthMgr.ReadinessHandler())

	// Version endpoint
	http.HandleFunc("/version", versionHandler())

	// Slack endpoints with full middleware stack
	http.HandleFunc("/slack/command", middleware.Chain(
		handler.HandleSlashCommand,
		func(next http.HandlerFunc) http.HandlerFunc {
			return middleware.WithLogging(logger, next)
		},
		func(next http.HandlerFunc) http.HandlerFunc {
			return middleware.WithTimeout(30*time.Second, logger, m, next)
		},
		func(next http.HandlerFunc) http.HandlerFunc {
			return middleware.WithMetrics("/slack/command", m, next)
		},
		func(next http.HandlerFunc) http.HandlerFunc {
			return middleware.WithRecovery(logger, m, next)
		},
	))

	http.HandleFunc("/slack/interactive", middleware.Chain(
		handler.HandleInteractive,
		func(next http.HandlerFunc) http.HandlerFunc {
			return middleware.WithLogging(logger, next)
		},
		func(next http.HandlerFunc) http.HandlerFunc {
			return middleware.WithTimeout(30*time.Second, logger, m, next)
		},
		func(next http.HandlerFunc) http.HandlerFunc {
			return middleware.WithMetrics("/slack/interactive", m, next)
		},
		func(next http.HandlerFunc) http.HandlerFunc {
			return middleware.WithRecovery(logger, m, next)
		},
	))

	http.HandleFunc("/slack/options", middleware.Chain(
		handler.HandleOptionsRequest,
		func(next http.HandlerFunc) http.HandlerFunc {
			return middleware.WithLogging(logger, next)
		},
		func(next http.HandlerFunc) http.HandlerFunc {
			return middleware.WithTimeout(30*time.Second, logger, m, next)
		},
		func(next http.HandlerFunc) http.HandlerFunc {
			return middleware.WithMetrics("/slack/options", m, next)
		},
		func(next http.HandlerFunc) http.HandlerFunc {
			return middleware.WithRecovery(logger, m, next)
		},
	))

	port := os.Getenv("PORT")
	if port == "" {
		port = constants.DefaultPort
	}

	// Configure server with explicit timeouts
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      nil, // uses DefaultServeMux
		ReadTimeout:  constants.ServerReadTimeout,
		WriteTimeout: constants.ServerWriteTimeout,
		IdleTimeout:  constants.ServerIdleTimeout,
	}

	// Setup graceful shutdown handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Run server in a goroutine
	go func() {
		logger.Info("starting Hopperbot server",
			zap.String("version", version),
			zap.String("commit", commit),
			zap.String("build_time", buildTime),
			zap.String("port", port),
			zap.String("metrics_endpoint", "/metrics"),
			zap.String("health_endpoint", "/health"),
			zap.String("readiness_endpoint", "/ready"),
			zap.String("version_endpoint", "/version"),
			zap.String("options_endpoint", "/slack/options"),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed to start", zap.Error(err))
		}
	}()

	// Block until shutdown signal
	<-stop
	logger.Info("shutdown signal received, initiating graceful shutdown")

	// Stop cache manager
	cacheMgr.Stop()
	logger.Info("cache manager stopped")

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), constants.GracefulShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("error during graceful shutdown", zap.Error(err))
	} else {
		logger.Info("server shutdown complete")
	}
}

// versionHandler returns an HTTP handler for the /version endpoint.
// Returns build information including version, commit hash, and build time.
func versionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		info := VersionInfo{
			Version:   version,
			Commit:    commit,
			BuildTime: buildTime,
			GoVersion: "go1.21+", // Minimum required Go version
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(info)
	}
}
