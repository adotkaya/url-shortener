package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"url-shortener/internal/domain"
)

// URLService interface defines the service methods needed by the handler
// Using an interface instead of concrete type allows for easy mocking in tests
type URLService interface {
	CreateShortURL(ctx context.Context, originalURL, customAlias, createdBy string, expiresIn time.Duration) (*domain.URL, error)
	GetURL(ctx context.Context, shortCode string) (*domain.URL, error)
	RecordClick(ctx context.Context, shortCode, ipAddress, userAgent, referer string) error
	GetURLStats(ctx context.Context, shortCode string) (*domain.URL, []*domain.URLClick, error)
	DeleteURL(ctx context.Context, id string) error
}

// Handler holds dependencies for HTTP handlers
// This is DEPENDENCY INJECTION - we pass dependencies through the constructor
// instead of using global variables or creating them inside handlers
type Handler struct {
	urlService URLService
	logger     *slog.Logger
	baseURL    string // Base URL for generating short URLs (e.g., "http://localhost:8080")
}

// NewHandler creates a new HTTP handler
func NewHandler(urlService URLService, logger *slog.Logger, baseURL string) *Handler {
	return &Handler{
		urlService: urlService,
		logger:     logger,
		baseURL:    baseURL,
	}
}

// Request/Response DTOs (Data Transfer Objects)
// These are separate from domain models because:
// 1. API contracts should be stable even if domain models change
// 2. We might want to expose/hide certain fields
// 3. We can add API-specific validation

type CreateURLRequest struct {
	URL            string `json:"url"`
	CustomAlias    string `json:"custom_alias,omitempty"`
	ExpiresInHours int    `json:"expires_in_hours,omitempty"`
}

type CreateURLResponse struct {
	ID          string     `json:"id"`
	ShortCode   string     `json:"short_code"`
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type URLStatsResponse struct {
	ID           string      `json:"id"`
	ShortCode    string      `json:"short_code"`
	OriginalURL  string      `json:"original_url"`
	Clicks       int64       `json:"clicks"`
	CreatedAt    time.Time   `json:"created_at"`
	ExpiresAt    *time.Time  `json:"expires_at,omitempty"`
	RecentClicks []ClickInfo `json:"recent_clicks"`
}

type ClickInfo struct {
	ClickedAt   time.Time `json:"clicked_at"`
	CountryCode string    `json:"country_code,omitempty"`
	City        string    `json:"city,omitempty"`
}

// CreateURL handles POST /api/v1/urls
func (h *Handler) CreateURL(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse request body
	var req CreateURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	defer r.Body.Close()

	// Validate required fields
	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "URL is required")
		return
	}

	// Calculate expiration duration
	var expiresIn time.Duration
	if req.ExpiresInHours > 0 {
		expiresIn = time.Duration(req.ExpiresInHours) * time.Hour
	}

	// Call service layer
	url, err := h.urlService.CreateShortURL(
		r.Context(),
		req.URL,
		req.CustomAlias,
		"anonymous", // TODO: Get from authentication
		expiresIn,
	)
	if err != nil {
		h.logger.Error("Failed to create URL", "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build response
	response := CreateURLResponse{
		ID:          url.ID,
		ShortCode:   url.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", h.baseURL, url.ShortCode),
		OriginalURL: url.OriginalURL,
		CreatedAt:   url.CreatedAt,
		ExpiresAt:   url.ExpiresAt,
	}

	respondSuccess(w, http.StatusCreated, response, "URL created successfully")
}

// RedirectURL handles GET /{shortCode}
func (h *Handler) RedirectURL(w http.ResponseWriter, r *http.Request) {
	// Extract short code from path
	shortCode := r.URL.Path[1:] // Remove leading "/"

	if shortCode == "" {
		respondError(w, http.StatusBadRequest, "Short code is required")
		return
	}

	// Get URL from service
	url, err := h.urlService.GetURL(r.Context(), shortCode)
	if err != nil {
		h.logger.Warn("URL not found", "short_code", shortCode, "error", err)
		respondError(w, http.StatusNotFound, "URL not found")
		return
	}

	// Record the click asynchronously (don't block the redirect)
	// This is a common pattern: analytics shouldn't slow down the user experience
	go func() {
		// Extract analytics data from request
		ipAddress := r.RemoteAddr
		userAgent := r.UserAgent()
		referer := r.Referer()

		if err := h.urlService.RecordClick(r.Context(), shortCode, ipAddress, userAgent, referer); err != nil {
			h.logger.Error("Failed to record click", "error", err)
		}
	}()

	// Perform the redirect
	// http.StatusFound (302) is a temporary redirect
	// http.StatusMovedPermanently (301) is a permanent redirect
	// We use 302 because URLs might expire or change
	http.Redirect(w, r, url.OriginalURL, http.StatusFound)
}

// GetURLStats handles GET /api/v1/urls/{shortCode}/stats
func (h *Handler) GetURLStats(w http.ResponseWriter, r *http.Request) {
	// Extract short code from path
	// In a real app, you'd use a router like gorilla/mux or chi
	// For now, we'll parse it manually
	shortCode := r.URL.Path[len("/api/v1/urls/"):]
	if len(shortCode) > 6 {
		shortCode = shortCode[:len(shortCode)-6] // Remove "/stats"
	}

	// Get stats from service
	url, clicks, err := h.urlService.GetURLStats(r.Context(), shortCode)
	if err != nil {
		h.logger.Error("Failed to get stats", "error", err)
		respondError(w, http.StatusNotFound, "URL not found")
		return
	}

	// Build response
	recentClicks := make([]ClickInfo, 0, len(clicks))
	for _, click := range clicks {
		recentClicks = append(recentClicks, ClickInfo{
			ClickedAt:   click.ClickedAt,
			CountryCode: click.CountryCode,
			City:        click.City,
		})
	}

	response := URLStatsResponse{
		ID:           url.ID,
		ShortCode:    url.ShortCode,
		OriginalURL:  url.OriginalURL,
		Clicks:       url.Clicks,
		CreatedAt:    url.CreatedAt,
		ExpiresAt:    url.ExpiresAt,
		RecentClicks: recentClicks,
	}

	respondSuccess(w, http.StatusOK, response, "")
}

// HealthCheck handles GET /health/live
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}
