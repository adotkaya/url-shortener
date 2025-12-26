package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the application
// Using promauto automatically registers metrics with the default registry

var (
	// ==================== HTTP METRICS ====================

	// HTTPRequestDuration tracks the duration of HTTP requests
	// Histogram allows us to calculate percentiles (P50, P95, P99)
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTPRequestsTotal counts total HTTP requests
	// Counter only goes up, useful for calculating rates
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTPRequestsInFlight tracks currently processing requests
	// Gauge can go up and down
	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
	)

	// ==================== CACHE METRICS ====================

	// CacheHitsTotal counts cache hits
	CacheHitsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	// CacheMissesTotal counts cache misses
	CacheMissesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	// CacheOperationDuration tracks cache operation latency
	CacheOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_operation_duration_seconds",
			Help:    "Duration of cache operations in seconds",
			Buckets: []float64{.0001, .0005, .001, .0025, .005, .01, .025, .05},
		},
		[]string{"operation"}, // get, set, delete
	)

	// ==================== RATE LIMITING METRICS ====================

	// RateLimitedRequestsTotal counts rate-limited requests
	RateLimitedRequestsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "rate_limited_requests_total",
			Help: "Total number of rate-limited requests",
		},
	)

	// RateLimitAllowedRequestsTotal counts allowed requests
	RateLimitAllowedRequestsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "rate_limit_allowed_requests_total",
			Help: "Total number of requests allowed by rate limiter",
		},
	)

	// ==================== BUSINESS METRICS ====================

	// URLsCreatedTotal counts URLs created
	URLsCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "urls_created_total",
			Help: "Total number of URLs created",
		},
	)

	// RedirectsTotal counts successful redirects
	RedirectsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "redirects_total",
			Help: "Total number of successful redirects",
		},
	)

	// ClicksRecordedTotal counts analytics events
	ClicksRecordedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "clicks_recorded_total",
			Help: "Total number of click events recorded",
		},
	)

	// ActiveURLsGauge tracks number of active URLs
	ActiveURLsGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_urls",
			Help: "Number of active (non-expired) URLs",
		},
	)

	// ==================== DATABASE METRICS ====================

	// DatabaseQueryDuration tracks database query latency
	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Duration of database queries in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"operation"}, // create, get, update, delete
	)

	// DatabaseErrorsTotal counts database errors
	DatabaseErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_errors_total",
			Help: "Total number of database errors",
		},
		[]string{"operation"},
	)
)

// RecordCacheHit increments cache hit counter
func RecordCacheHit() {
	CacheHitsTotal.Inc()
}

// RecordCacheMiss increments cache miss counter
func RecordCacheMiss() {
	CacheMissesTotal.Inc()
}

// RecordURLCreated increments URL creation counter
func RecordURLCreated() {
	URLsCreatedTotal.Inc()
}

// RecordRedirect increments redirect counter
func RecordRedirect() {
	RedirectsTotal.Inc()
}

// RecordClickRecorded increments click recording counter
func RecordClickRecorded() {
	ClicksRecordedTotal.Inc()
}

// RecordRateLimited increments rate-limited requests counter
func RecordRateLimited() {
	RateLimitedRequestsTotal.Inc()
}

// RecordRateLimitAllowed increments allowed requests counter
func RecordRateLimitAllowed() {
	RateLimitAllowedRequestsTotal.Inc()
}
