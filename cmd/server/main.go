// ============================================================================
// MAIN.GO - APPLICATION ENTRY POINT
// ============================================================================
// This is where your application starts. Think of it like Program.cs in .NET.
//
// KEY CONCEPTS DEMONSTRATED:
// 1. Dependency Injection (DI) - Manually wiring up dependencies
// 2. Layered Architecture - Handler → Service → Repository
// 3. Graceful Shutdown - Properly closing resources on exit
// 4. Middleware Chaining - Composing cross-cutting concerns
// 5. Connection Pooling - Reusing database connections
//
// LEARNING TIP: Read this file top-to-bottom to understand the startup flow
// ============================================================================

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
	// ========================================================================
	// STEP 1: LOAD CONFIGURATION
	// ========================================================================
	// Configuration comes from environment variables (12-factor app principle)
	// This allows the same code to run in dev, staging, and production
	// with different settings (database URLs, ports, etc.)
	//
	// .NET EQUIVALENT: IConfiguration in ASP.NET Core
	// ========================================================================
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// ========================================================================
	// STEP 2: INITIALIZE STRUCTURED LOGGER
	// ========================================================================
	// Structured logging outputs JSON with key-value pairs instead of plain text
	// This makes logs searchable and parseable by tools like Elasticsearch
	//
	// Example output:
	// {"time":"2025-12-25T15:00:00Z","level":"INFO","msg":"Starting URL Shortener","environment":"dev","port":"8080"}
	//
	// .NET EQUIVALENT: ILogger<T> with Serilog
	// ========================================================================
	appLogger := logger.New(cfg.App.LogLevel)
	appLogger.Info("Starting URL Shortener",
		"environment", cfg.App.Environment,
		"port", cfg.Server.Port,
	)

	// ========================================================================
	// STEP 3: INITIALIZE DATABASE CONNECTION POOL
	// ========================================================================
	// CONNECTION POOLING EXPLAINED:
	// Instead of creating a new connection for each query (slow!), we maintain
	// a pool of reusable connections. This dramatically improves performance.
	//
	// Pool Configuration:
	// - MaxOpenConns: 25  → Maximum concurrent connections
	// - MaxIdleConns: 5   → Keep 5 connections ready (warm pool)
	// - ConnMaxLifetime: 5m → Recycle connections every 5 minutes
	//
	// WHY POOLING MATTERS:
	// Creating a connection requires: DNS lookup, TCP handshake, SSL negotiation,
	// authentication. This can take 50-100ms. With pooling, queries take ~1-5ms.
	//
	// .NET EQUIVALENT: DbContext with connection pooling (automatic in EF Core)
	// ========================================================================
	ctx := context.Background()
	db, err := postgres.InitDB(
		ctx,
		cfg.Database.DatabaseDSN(), // Connection string: "host=localhost port=5432 user=..."
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
		cfg.Database.ConnMaxLifetime,
	)
	if err != nil {
		appLogger.Error("Failed to connect to database", "error", err)
		log.Fatalf("Database connection failed: %v", err)
	}
	// DEFER: Ensures db.Close() runs when main() exits, even if panic occurs
	// Similar to C#'s using statement or try-finally
	defer db.Close()
	appLogger.Info("Database connection established")

	// ========================================================================
	// STEP 4: DEPENDENCY INJECTION (DI) - BUILD THE DEPENDENCY GRAPH
	// ========================================================================
	// We manually wire up dependencies here. In Go, there's no built-in DI
	// container like .NET's IServiceCollection. This is intentional - it keeps
	// things explicit and traceable.
	//
	// DEPENDENCY FLOW:
	// Database Pool → Repositories → Service → Handler
	//
	// WHY MANUAL DI?
	// - Explicit: You can see exactly what depends on what
	// - Simple: No magic, no reflection, no runtime surprises
	// - Testable: Easy to inject mocks for testing
	//
	// .NET EQUIVALENT:
	// services.AddScoped<IURLRepository, PostgresURLRepository>();
	// services.AddScoped<IURLService, URLService>();
	// ========================================================================

	// Initialize repositories (Data Access Layer)
	// These handle all database operations
	urlRepo := postgres.NewURLRepository(db)
	clickRepo := postgres.NewClickRepository(db)

	// Initialize services (Business Logic Layer)
	// This orchestrates operations across multiple repositories
	urlService := service.NewURLService(urlRepo, clickRepo)

	// Initialize HTTP handler (Presentation Layer)
	// This handles HTTP requests and responses
	baseURL := fmt.Sprintf("http://localhost:%s", cfg.Server.Port)
	handler := httpHandler.NewHandler(urlService, appLogger.Logger, baseURL)

	// ========================================================================
	// STEP 5: SET UP HTTP ROUTES
	// ========================================================================
	// Go's standard library uses http.ServeMux for routing
	// It's simple but limited (no path parameters like /users/:id)
	// For production, consider using chi, gorilla/mux, or gin
	//
	// .NET EQUIVALENT: app.MapControllers() in ASP.NET Core
	// ========================================================================
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/v1/urls", handler.CreateURL)
	mux.HandleFunc("/api/v1/urls/", handler.GetURLStats) // Trailing slash matches /api/v1/urls/*

	// Health check endpoint (for Kubernetes liveness probes)
	mux.HandleFunc("/health/live", handler.HealthCheck)

	// Redirect route (catch-all for short codes)
	// IMPORTANT: This must be registered LAST because "/" matches everything
	mux.HandleFunc("/", handler.RedirectURL)

	// ========================================================================
	// STEP 6: APPLY MIDDLEWARE CHAIN
	// ========================================================================
	// MIDDLEWARE PATTERN:
	// Middleware wraps handlers to add cross-cutting concerns like logging,
	// authentication, CORS, etc. Each middleware can:
	// 1. Execute code BEFORE the handler
	// 2. Execute code AFTER the handler
	// 3. Modify the request or response
	// 4. Short-circuit the request (e.g., return 401 Unauthorized)
	//
	// EXECUTION ORDER (outside-in):
	// Request → Recovery → Logging → RequestID → CORS → Handler → Response
	//
	// WHY THIS ORDER?
	// - Recovery: Outermost to catch panics from all other middleware
	// - Logging: Log all requests, including those that panic
	// - RequestID: Add ID early so all logs include it
	// - CORS: Add headers before handler runs
	//
	// .NET EQUIVALENT:
	// app.UseExceptionHandler();
	// app.UseLogging();
	// app.UseCors();
	// ========================================================================
	finalHandler := httpHandler.Chain(
		httpHandler.RecoveryMiddleware(appLogger.Logger),  // Catches panics, prevents server crash
		httpHandler.LoggingMiddleware(appLogger.Logger),   // Logs every request with duration
		httpHandler.RequestIDMiddleware,                   // Adds unique ID to each request
		httpHandler.CORSMiddleware,                        // Adds CORS headers for browser requests
	)(mux)

	// ========================================================================
	// STEP 7: CREATE HTTP SERVER
	// ========================================================================
	// TIMEOUTS EXPLAINED:
	// - ReadTimeout: Max time to read request headers and body (prevents slow clients)
	// - WriteTimeout: Max time to write response (prevents slow handlers)
	// - IdleTimeout: Max time to keep connection alive between requests
	//
	// WHY TIMEOUTS MATTER:
	// Without timeouts, a slow client can tie up server resources indefinitely,
	// leading to resource exhaustion and denial of service.
	//
	// .NET EQUIVALENT: Kestrel server configuration in Program.cs
	// ========================================================================
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      finalHandler,
		ReadTimeout:  cfg.Server.ReadTimeout,  // Default: 10s
		WriteTimeout: cfg.Server.WriteTimeout, // Default: 10s
		IdleTimeout:  cfg.Server.IdleTimeout,  // Default: 120s
	}

	// ========================================================================
	// STEP 8: START SERVER IN BACKGROUND (GOROUTINE)
	// ========================================================================
	// GOROUTINES EXPLAINED:
	// A goroutine is a lightweight thread managed by the Go runtime.
	// Creating a goroutine is cheap (~2KB stack), so you can have thousands.
	//
	// WHY START IN GOROUTINE?
	// ListenAndServe() blocks forever. By running it in a goroutine, we can
	// continue to the graceful shutdown code below.
	//
	// .NET EQUIVALENT: Task.Run(() => app.Run())
	// ========================================================================
	go func() {
		appLogger.Info("Server starting", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("Server failed", "error", err)
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// ========================================================================
	// STEP 9: GRACEFUL SHUTDOWN - WAIT FOR INTERRUPT SIGNAL
	// ========================================================================
	// GRACEFUL SHUTDOWN EXPLAINED:
	// When you press Ctrl+C or send SIGTERM (kill command), the OS sends a
	// signal to the process. We catch this signal and shut down gracefully:
	//
	// 1. Stop accepting new requests
	// 2. Wait for in-flight requests to complete (up to 30 seconds)
	// 3. Close database connections
	// 4. Exit cleanly
	//
	// WHY THIS MATTERS:
	// - Data Integrity: Don't lose data from incomplete requests
	// - User Experience: Don't drop active connections
	// - Resource Cleanup: Close files, connections, etc.
	//
	// CHANNELS EXPLAINED:
	// Channels are Go's way of communicating between goroutines.
	// - make(chan os.Signal, 1): Create a buffered channel (capacity 1)
	// - signal.Notify(quit, ...): Send signals to this channel
	// - <-quit: Block until a value is received (wait for signal)
	//
	// .NET EQUIVALENT: IHostApplicationLifetime.ApplicationStopping
	// ========================================================================
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block here until Ctrl+C or kill signal

	appLogger.Info("Shutting down server...")

	// ========================================================================
	// STEP 10: GRACEFUL SHUTDOWN - DRAIN CONNECTIONS
	// ========================================================================
	// Give in-flight requests 30 seconds to complete
	// If they don't finish in time, force shutdown
	//
	// CONTEXT WITH TIMEOUT:
	// context.WithTimeout creates a context that automatically cancels after
	// the specified duration. This is passed to Shutdown() to enforce the deadline.
	// ========================================================================
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel() // Always call cancel to free resources

	// Attempt graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("Server forced to shutdown", "error", err)
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	appLogger.Info("Server exited gracefully")

	// ========================================================================
	// CONGRATULATIONS! 🎉
	// ========================================================================
	// You've just traced through a production-grade Go application startup!
	//
	// KEY TAKEAWAYS:
	// ✅ Configuration from environment variables (12-factor app)
	// ✅ Structured logging with JSON output
	// ✅ Database connection pooling for performance
	// ✅ Manual dependency injection (explicit and traceable)
	// ✅ Middleware for cross-cutting concerns
	// ✅ Graceful shutdown for clean resource cleanup
	//
	// NEXT STEPS:
	// 1. Read internal/domain/url.go to understand domain models
	// 2. Read internal/repository/repository.go to understand the repository pattern
	// 3. Read internal/service/url_service.go to understand business logic
	// 4. Read internal/handler/http/handler.go to understand HTTP handling
	// ========================================================================
}
