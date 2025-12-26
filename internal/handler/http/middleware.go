package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"url-shortener/internal/metrics"

	"github.com/google/uuid"
)

// Middleware is a function that wraps an http.Handler
// This is the MIDDLEWARE PATTERN in Go
// Middleware can:
// 1. Execute code before the handler
// 2. Execute code after the handler
// 3. Modify the request or response
// 4. Short-circuit the request (e.g., authentication failure)

// LoggingMiddleware logs HTTP requests with structured logging
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Call the next handler
			next.ServeHTTP(wrapped, r)

			// Log after the request is processed
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

// RateLimitMiddleware adds rate limiting to protect against abuse
// Uses token bucket algorithm with Redis for distributed rate limiting
func RateLimitMiddleware(limiter RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract identifier (IP address)
			// In production, you might use API keys instead
			ip := extractIP(r)

			// Check rate limit
			allowed, remaining, resetTime, err := limiter.Allow(r.Context(), ip)
			if err != nil {
				// If rate limiting fails, allow the request (fail open)
				// This prevents rate limiting issues from breaking the service
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers (standard practice)
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.MaxRequests()))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

			if !allowed {
				// Rate limit exceeded
				retryAfter := int(time.Until(resetTime).Seconds())
				if retryAfter < 0 {
					retryAfter = 0
				}

				metrics.RecordRateLimited()

				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			// Request allowed
			metrics.RecordRateLimitAllowed()
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter interface for rate limiting
type RateLimiter interface {
	Allow(ctx context.Context, key string) (allowed bool, remaining int, resetTime time.Time, err error)
	MaxRequests() int
}

// extractIP extracts the client IP address from the request
// Handles X-Forwarded-For header for proxies/load balancers
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header (set by some proxies)
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	// Remove port if present (e.g., "127.0.0.1:12345" -> "127.0.0.1")
	ip := r.RemoteAddr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}

	return ip
}

// MetricsMiddleware records Prometheus metrics for HTTP requests
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Track in-flight requests
		metrics.HTTPRequestsInFlight.Inc()
		defer metrics.HTTPRequestsInFlight.Dec()

		// Wrap response writer to capture status code
		wrapped := &metricsResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapped.statusCode)

		// Simplify endpoint for better cardinality
		endpoint := simplifyEndpoint(r.URL.Path)

		metrics.HTTPRequestDuration.WithLabelValues(
			r.Method,
			endpoint,
			status,
		).Observe(duration)

		metrics.HTTPRequestsTotal.WithLabelValues(
			r.Method,
			endpoint,
			status,
		).Inc()
	})
}

// metricsResponseWriter wraps http.ResponseWriter to capture status code
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (m *metricsResponseWriter) WriteHeader(code int) {
	m.statusCode = code
	m.ResponseWriter.WriteHeader(code)
}

// simplifyEndpoint reduces cardinality by grouping similar endpoints
func simplifyEndpoint(path string) string {
	// Root path
	if path == "/" {
		return "/"
	}

	// API endpoints
	if strings.HasPrefix(path, "/api/v1/urls/") {
		if strings.HasSuffix(path, "/stats") {
			return "/api/v1/urls/:id/stats"
		}
		return "/api/v1/urls/:id"
	}

	if path == "/api/v1/urls" {
		return "/api/v1/urls"
	}

	// Health check
	if path == "/health/live" {
		return "/health/live"
	}

	// Metrics endpoint
	if path == "/metrics" {
		return "/metrics"
	}

	// Static files
	if strings.HasPrefix(path, "/static/") {
		return "/static/*"
	}

	// Short code redirect (catch-all)
	return "/:shortcode"
}
