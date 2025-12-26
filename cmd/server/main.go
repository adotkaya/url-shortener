package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"url-shortener/internal/config"
	httpHandler "url-shortener/internal/handler/http"
	"url-shortener/internal/ratelimit"
	"url-shortener/internal/repository/postgres"
	redisrepo "url-shortener/internal/repository/redis"
	"url-shortener/internal/service"
	"url-shortener/pkg/logger"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	appLogger := logger.New(cfg.App.LogLevel)
	appLogger.Info("Starting URL Shortener",
		"environment", cfg.App.Environment,
		"port", cfg.Server.Port,
	)

	// Initialize database connection
	ctx := context.Background()
	db, err := postgres.InitDB(
		ctx,
		cfg.Database.DatabaseDSN(),
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
		cfg.Database.ConnMaxLifetime,
	)
	if err != nil {
		appLogger.Error("Failed to connect to database", "error", err)
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()
	appLogger.Info("Database connection established")

	// Initialize Redis connection
	redisClient, err := redisrepo.InitRedis(
		cfg.Redis.RedisAddr(),
		cfg.Redis.Password,
		cfg.Redis.DB,
	)
	if err != nil {
		appLogger.Error("Failed to connect to Redis", "error", err)
		log.Fatalf("Redis connection failed: %v", err)
	}
	defer redisClient.Close()
	appLogger.Info("Redis connection established")

	// Initialize cache
	cache := redisrepo.NewCache(redisClient, cfg.Redis.CacheTTL)

	// Initialize repositories (Data Access Layer)
	urlRepo := postgres.NewURLRepository(db)
	clickRepo := postgres.NewClickRepository(db)

	// Initialize services (Business Logic Layer)
	urlService := service.NewURLService(urlRepo, clickRepo, cache)

	// Initialize HTTP handler (Presentation Layer)
	baseURL := fmt.Sprintf("http://localhost:%s", cfg.Server.Port)
	handler := httpHandler.NewHandler(urlService, appLogger.Logger, baseURL)

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Serve static files (CSS, JS, images)
	httpHandler.SetupStaticFiles(mux)

	// API routes
	mux.HandleFunc("/api/v1/urls", handler.CreateURL)
	mux.HandleFunc("/api/v1/urls/", handler.GetURLStats) // Note: trailing slash for path matching

	// Health check
	mux.HandleFunc("/health/live", handler.HealthCheck)

	// Metrics endpoints (must be before catch-all)
	mux.HandleFunc("/metrics", httpHandler.ServeMetricsPage) // Styled page for viewing
	mux.Handle("/metrics-raw", promhttp.Handler())           // Raw metrics for Prometheus

	// API Documentation (must be before catch-all)
	mux.HandleFunc("/api/docs", httpHandler.ServeSwagger)
	mux.HandleFunc("/api/openapi.json", httpHandler.ServeOpenAPISpec)

	// UI and redirect routes
	// This must be last because it matches everything
	mux.HandleFunc("/", handler.ServeUI)

	// Initialize rate limiter
	rateLimiter := ratelimit.NewTokenBucketLimiter(
		redisClient,
		cfg.App.RateLimitPerMinute,
		time.Minute,
		cfg.App.RateLimitPerMinute+20, // Allow burst of 20 extra requests
	)

	// Apply middleware
	// Middleware is applied in reverse order (last middleware wraps first)
	var finalHandler http.Handler = mux

	// Only apply rate limiting if enabled in config
	if cfg.App.RateLimitEnabled {
		finalHandler = httpHandler.RateLimitMiddleware(rateLimiter)(finalHandler)
		appLogger.Info("Rate limiting enabled", "requests_per_minute", cfg.App.RateLimitPerMinute)
	}

	// Apply other middleware
	finalHandler = httpHandler.Chain(
		httpHandler.RecoveryMiddleware(appLogger.Logger),
		httpHandler.LoggingMiddleware(appLogger.Logger),
		httpHandler.RequestIDMiddleware,
		httpHandler.CORSMiddleware,
	)(finalHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      finalHandler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		appLogger.Info("Server starting", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("Server failed", "error", err)
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	// This is GRACEFUL SHUTDOWN - we wait for existing requests to complete
	// before shutting down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	// Create a deadline for shutdown (30 seconds)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("Server forced to shutdown", "error", err)
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	appLogger.Info("Server exited gracefully")
}
