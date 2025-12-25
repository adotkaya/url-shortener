// ============================================================================
// MIDDLEWARE.GO - CROSS-CUTTING CONCERNS
// ============================================================================
// Middleware functions wrap HTTP handlers to add functionality like logging,
// authentication, CORS, panic recovery, etc.
//
// KEY CONCEPTS:
// 1. Middleware Pattern - Composable request/response processing
// 2. Function Closures - Functions that capture variables from outer scope
// 3. Higher-Order Functions - Functions that take/return other functions
// 4. Panic Recovery - Preventing server crashes
// 5. Context Propagation - Passing request-scoped data
//
// MIDDLEWARE SIGNATURE:
// func(http.Handler) http.Handler
//      ↑                ↑
//      |                └─ Returns wrapped handler
//      └───────────────── Takes handler to wrap
//
// .NET EQUIVALENT: ASP.NET Core middleware pipeline
// ============================================================================

package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// LOGGING MIDDLEWARE
// ============================================================================
// LoggingMiddleware logs HTTP requests with structured logging.
//
// FUNCTION CLOSURE EXPLAINED:
// This function returns a function that returns a function!
//
//	func LoggingMiddleware(logger) func(http.Handler) http.Handler {
//	     ↑                              ↑
//	     |                              └─ Returns middleware function
//	     └────────────────────────────────── Takes logger parameter
//
// WHY CLOSURE?
// The returned function "closes over" the logger variable, meaning it
// captures and remembers the logger even after LoggingMiddleware returns.
//
// .NET EQUIVALENT:
//
//	public class LoggingMiddleware {
//	    private readonly ILogger _logger;
//	    public LoggingMiddleware(ILogger logger) { _logger = logger; }
//	    public async Task InvokeAsync(HttpContext context, RequestDelegate next) {
//	        var start = DateTime.Now;
//	        await next(context);
//	        var duration = DateTime.Now - start;
//	        _logger.LogInformation("Request completed in {Duration}ms", duration);
//	    }
//	}
//
// ============================================================================
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			// (http.ResponseWriter doesn't expose the status code by default)
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Call the next handler in the chain
			next.ServeHTTP(wrapped, r)

			// Log after the request is processed (we now have the status code)
			duration := time.Since(start)
			logger.Info("HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration_ms", duration.Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequestIDMiddleware adds a unique request ID to each request
// This is crucial for DISTRIBUTED TRACING and debugging
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate a unique request ID
		requestID := uuid.New().String()

		// Add to response headers for client-side tracking
		w.Header().Set("X-Request-ID", requestID)

		// Add to context so handlers can access it
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RecoveryMiddleware recovers from panics and returns a 500 error
// This prevents the entire server from crashing due to a panic in a handler
func RecoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("Panic recovered",
						"error", err,
						"path", r.URL.Path,
						"method", r.Method,
					)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware adds CORS headers
// CORS (Cross-Origin Resource Sharing) allows web apps from different domains to access your API
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow all origins in development (restrict in production!)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// TimeoutMiddleware adds a timeout to requests
// This prevents slow clients or handlers from tying up server resources
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create a channel to signal when the handler completes
			done := make(chan struct{})

			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				// Handler completed successfully
				return
			case <-ctx.Done():
				// Timeout occurred
				http.Error(w, "Request timeout", http.StatusRequestTimeout)
			}
		})
	}
}

// Chain combines multiple middleware functions
// This is a helper to make middleware composition cleaner
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		// Apply middleware in reverse order so they execute in the order specified
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}
