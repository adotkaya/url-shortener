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
	"url-shortener/internal/repository/postgres"
	"url-shortener/internal/service"
	"url-shortener/pkg/logger"
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

	// Initialize repositories (Data Access Layer)
	urlRepo := postgres.NewURLRepository(db)
	clickRepo := postgres.NewClickRepository(db)

	// Initialize services (Business Logic Layer)
	urlService := service.NewURLService(urlRepo, clickRepo)

	// Initialize HTTP handler (Presentation Layer)
	baseURL := fmt.Sprintf("http://localhost:%s", cfg.Server.Port)
	handler := httpHandler.NewHandler(urlService, appLogger.Logger, baseURL)

	// Set up HTTP routes
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/v1/urls", handler.CreateURL)
	mux.HandleFunc("/api/v1/urls/", handler.GetURLStats) // Note: trailing slash for path matching

	// Health check
	mux.HandleFunc("/health/live", handler.HealthCheck)

	// Redirect route (catch-all for short codes)
	// This must be last because it matches everything
	mux.HandleFunc("/", handler.RedirectURL)

	// Apply middleware
	// Middleware is applied in reverse order (last middleware wraps first)
	finalHandler := httpHandler.Chain(
		httpHandler.RecoveryMiddleware(appLogger.Logger),
		httpHandler.LoggingMiddleware(appLogger.Logger),
		httpHandler.RequestIDMiddleware,
		httpHandler.CORSMiddleware,
	)(mux)

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
