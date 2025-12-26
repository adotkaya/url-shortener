package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"url-shortener/internal/domain"
	"url-shortener/internal/repository"
)

// Cache interface for URL caching
// Using an interface allows for easy testing and swapping implementations
type Cache interface {
	GetURL(ctx context.Context, shortCode string) (*domain.URL, error)
	SetURL(ctx context.Context, shortCode string, url *domain.URL) error
	DeleteURL(ctx context.Context, shortCode string) error
}

// URLService handles business logic for URL operations
// This is the SERVICE LAYER - it sits between HTTP handlers and repositories
//
// WHY HAVE A SERVICE LAYER?
// 1. Business Logic: Complex operations that involve multiple repositories
// 2. Transaction Management: Coordinate multiple database operations
// 3. Validation: Business rule validation beyond simple field validation
// 4. Reusability: Same logic can be used by HTTP API, gRPC, CLI, etc.
type URLService struct {
	urlRepo   repository.URLRepository
	clickRepo repository.ClickRepository
	cache     Cache // Redis cache for performance
}

// NewURLService creates a new URL service
func NewURLService(urlRepo repository.URLRepository, clickRepo repository.ClickRepository, cache Cache) *URLService {
	return &URLService{
		urlRepo:   urlRepo,
		clickRepo: clickRepo,
		cache:     cache,
	}
}

// CreateShortURL creates a new shortened URL
// This method orchestrates multiple operations:
// 1. Generate or validate short code
// 2. Check for collisions
// 3. Validate the URL
// 4. Save to database
func (s *URLService) CreateShortURL(ctx context.Context, originalURL, customAlias, createdBy string, expiresIn time.Duration) (*domain.URL, error) {
	// Determine the short code (custom alias or generated)
	var shortCode string
	if customAlias != "" {
		// Check if custom alias is already taken
		exists, err := s.urlRepo.ExistsCustomAlias(ctx, customAlias)
		if err != nil {
			return nil, fmt.Errorf("failed to check custom alias: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("custom alias already exists: %s", customAlias)
		}
		shortCode = customAlias
	} else {
		// Generate a unique short code
		var err error
		shortCode, err = s.generateUniqueShortCode(ctx, 6)
		if err != nil {
			return nil, fmt.Errorf("failed to generate short code: %w", err)
		}
	}

	// Create the URL domain object
	url := domain.NewURL(originalURL, shortCode, createdBy)

	// Set custom alias if provided
	if customAlias != "" {
		url.WithCustomAlias(customAlias)
	}

	// Set expiration if provided
	if expiresIn > 0 {
		url.WithExpiration(expiresIn)
	}

	// Validate the URL (business rules)
	if err := url.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Save to database
	if err := s.urlRepo.Create(ctx, url); err != nil {
		return nil, fmt.Errorf("failed to create URL: %w", err)
	}

	// Store in cache for fast access
	// We don't fail if caching fails - it's not critical
	if err := s.cache.SetURL(ctx, shortCode, url); err != nil {
		fmt.Printf("Warning: failed to cache URL: %v\n", err)
	}

	return url, nil
}

// GetURL retrieves a URL by its short code or custom alias
// Implements CACHE-ASIDE PATTERN for performance
func (s *URLService) GetURL(ctx context.Context, shortCode string) (*domain.URL, error) {
	// STEP 1: Check cache first (cache-aside pattern)
	cachedURL, err := s.cache.GetURL(ctx, shortCode)
	if err == nil && cachedURL != nil {
		// Cache hit! Return immediately
		// This is ~50x faster than database lookup
		if err = cachedURL.CanBeAccessed(); err != nil {
			return nil, err
		}
		return cachedURL, nil
	}

	// STEP 2: Cache miss - get from database
	url, err := s.urlRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		// If not found, try custom alias
		url, err = s.urlRepo.GetByCustomAlias(ctx, shortCode)
		if err != nil {
			return nil, fmt.Errorf("URL not found: %s", shortCode)
		}
	}

	// Check if URL can be accessed (not expired, active)
	if err := url.CanBeAccessed(); err != nil {
		return nil, err
	}

	// STEP 3: Store in cache for next time
	// Don't fail if caching fails - it's not critical
	if err := s.cache.SetURL(ctx, shortCode, url); err != nil {
		fmt.Printf("Warning: failed to cache URL: %v\n", err)
	}

	return url, nil
}

// RecordClick records a click event and increments the counter
// This demonstrates a TRANSACTION-like operation across multiple tables
func (s *URLService) RecordClick(ctx context.Context, shortCode, ipAddress, userAgent, referer string) error {
	// Get the URL first to get its ID
	url, err := s.urlRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return fmt.Errorf("URL not found: %w", err)
	}

	// Increment the click counter atomically
	if err := s.urlRepo.IncrementClicks(ctx, shortCode); err != nil {
		return fmt.Errorf("failed to increment clicks: %w", err)
	}

	// Create click event for analytics
	click := domain.NewURLClick(url.ID, ipAddress, userAgent, referer)

	// TODO: Add geolocation lookup here
	// For now, we'll leave it empty
	// In production, you'd use a service like MaxMind GeoIP2

	if err := s.clickRepo.Create(ctx, click); err != nil {
		// Log the error but don't fail the request
		// Analytics is important but not critical for the redirect to work
		// This is a design decision: availability > consistency for analytics
		fmt.Printf("Warning: failed to record click event: %v\n", err)
	}

	return nil
}

// GetURLStats retrieves analytics for a URL
func (s *URLService) GetURLStats(ctx context.Context, shortCode string) (*domain.URL, []*domain.URLClick, error) {
	// Get the URL
	url, err := s.urlRepo.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, nil, fmt.Errorf("URL not found: %w", err)
	}

	// Get recent clicks (last 100)
	clicks, err := s.clickRepo.GetByURLID(ctx, url.ID, 100, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get clicks: %w", err)
	}

	return url, clicks, nil
}

// DeleteURL soft-deletes a URL
func (s *URLService) DeleteURL(ctx context.Context, id string) error {
	return s.urlRepo.Delete(ctx, id)
}

// generateUniqueShortCode generates a cryptographically random short code
// and ensures it doesn't collide with existing codes
func (s *URLService) generateUniqueShortCode(ctx context.Context, length int) (string, error) {
	// Try up to 10 times to generate a unique code
	// Collisions are rare with 6 characters (62^6 = 56 billion possibilities)
	for i := 0; i < 10; i++ {
		code := generateShortCode(length)

		// Check if it exists
		exists, err := s.urlRepo.ExistsShortCode(ctx, code)
		if err != nil {
			return "", err
		}

		if !exists {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique short code after 10 attempts")
}

// generateShortCode generates a random alphanumeric string
// Uses crypto/rand for cryptographically secure randomness
func generateShortCode(length int) string {
	// Base64 URL-safe encoding uses: A-Z, a-z, 0-9, -, _
	// We'll use a custom charset for better readability (no confusing characters)
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// Generate random bytes
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based (less secure but works)
		return base64.URLEncoding.EncodeToString([]byte(time.Now().String()))[:length]
	}

	// Map random bytes to charset
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}

	return string(bytes)
}
