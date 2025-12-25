package repository

import (
	"context"
	"url-shortener/internal/domain"
)

// URLRepository defines the interface for URL data access
// This is the "Repository Pattern" - it abstracts data storage
//
// WHY USE AN INTERFACE?
// 1. Testability: We can create mock implementations for testing
// 2. Flexibility: We can swap PostgreSQL for MongoDB without changing business logic
// 3. Dependency Inversion: High-level code doesn't depend on low-level database details
//
// In Go, interfaces are satisfied implicitly - any type that implements these methods
// automatically satisfies the interface (no "implements" keyword needed)
type URLRepository interface {
	// Create inserts a new URL into the database
	// context.Context is used for cancellation, timeouts, and passing request-scoped values
	Create(ctx context.Context, url *domain.URL) error

	// GetByShortCode retrieves a URL by its short code (e.g., "abc123")
	// Returns the URL and a boolean indicating if it was found
	GetByShortCode(ctx context.Context, shortCode string) (*domain.URL, error)

	// GetByID retrieves a URL by its UUID
	GetByID(ctx context.Context, id string) (*domain.URL, error)

	// GetByCustomAlias retrieves a URL by its custom alias
	GetByCustomAlias(ctx context.Context, alias string) (*domain.URL, error)

	// Update modifies an existing URL
	Update(ctx context.Context, url *domain.URL) error

	// Delete performs a soft delete (sets is_active = false)
	Delete(ctx context.Context, id string) error

	// IncrementClicks increases the click counter for a URL
	// This is done atomically in the database to avoid race conditions
	IncrementClicks(ctx context.Context, shortCode string) error

	// ExistsShortCode checks if a short code already exists
	// Used to prevent collisions when generating short codes
	ExistsShortCode(ctx context.Context, shortCode string) (bool, error)

	// ExistsCustomAlias checks if a custom alias is already taken
	ExistsCustomAlias(ctx context.Context, alias string) (bool, error)
}

// ClickRepository defines the interface for analytics data access
type ClickRepository interface {
	// Create inserts a new click event
	Create(ctx context.Context, click *domain.URLClick) error

	// GetByURLID retrieves all clicks for a specific URL
	GetByURLID(ctx context.Context, urlID string, limit, offset int) ([]*domain.URLClick, error)

	// GetClickCount returns the total number of clicks for a URL
	GetClickCount(ctx context.Context, urlID string) (int64, error)

	// GetClickStats returns aggregated statistics (clicks per day, top countries, etc.)
	// This would return a custom stats struct
	// GetClickStats(ctx context.Context, urlID string) (*ClickStats, error)
}
