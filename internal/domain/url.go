package domain

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

// URL represents a shortened URL in our system
// This is our "domain model" - it contains both data AND behavior (methods)
// In Go, we use structs to define data structures
type URL struct {
	ID          string     // UUID for internal identification
	ShortCode   string     // The short identifier (e.g., "abc123")
	OriginalURL string     // The full URL to redirect to
	CustomAlias *string    // Optional custom alias (pointer = nullable)
	CreatedAt   time.Time  // When the URL was created
	ExpiresAt   *time.Time // Optional expiration time (pointer = nullable)
	Clicks      int64      // Number of times this URL was accessed
	CreatedBy   string     // User/API key that created it
	IsActive    bool       // Soft delete flag
}

// Domain errors - defining errors as constants makes them testable
// and allows callers to check for specific error types
var (
	ErrInvalidURL         = errors.New("invalid URL format")
	ErrEmptyURL           = errors.New("URL cannot be empty")
	ErrShortCodeTooShort  = errors.New("short code must be at least 3 characters")
	ErrURLExpired         = errors.New("URL has expired")
	ErrURLNotActive       = errors.New("URL is not active")
	ErrCustomAliasInvalid = errors.New("custom alias must be alphanumeric and 3-20 characters")
)

// IsExpired checks if the URL has passed its expiration time
// This is a METHOD on the URL struct - it has access to the struct's fields via the receiver (u *URL)
// Methods in Go are functions with a receiver parameter
func (u *URL) IsExpired() bool {
	// If ExpiresAt is nil (not set), the URL never expires
	if u.ExpiresAt == nil {
		return false
	}
	// Check if current time is after expiration time
	return time.Now().After(*u.ExpiresAt)
}

// CanBeAccessed checks if the URL can be used for redirection
// This encapsulates business logic in the domain model
func (u *URL) CanBeAccessed() error {
	if !u.IsActive {
		return ErrURLNotActive
	}
	if u.IsExpired() {
		return ErrURLExpired
	}
	return nil
}

// Validate checks if the URL fields are valid
// This is called before saving to the database
func (u *URL) Validate() error {
	// Check if original URL is empty
	if strings.TrimSpace(u.OriginalURL) == "" {
		return ErrEmptyURL
	}

	// Parse and validate URL format
	// url.Parse is from Go's standard library
	parsedURL, err := url.Parse(u.OriginalURL)
	if err != nil {
		return ErrInvalidURL
	}

	// Ensure URL has a scheme (http:// or https://)
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return ErrInvalidURL
	}

	// Ensure URL has a host (domain)
	if parsedURL.Host == "" {
		return ErrInvalidURL
	}

	// Validate short code length
	if len(u.ShortCode) < 3 {
		return ErrShortCodeTooShort
	}

	// Validate custom alias if provided
	if u.CustomAlias != nil && *u.CustomAlias != "" {
		if !isValidAlias(*u.CustomAlias) {
			return ErrCustomAliasInvalid
		}
	}

	return nil
}

// IncrementClicks increases the click counter
// This is better than directly modifying the field because we can add logic here
// For example, we could add analytics tracking, validation, etc.
func (u *URL) IncrementClicks() {
	u.Clicks++
}

// isValidAlias checks if a custom alias is valid
// Private function (lowercase name) - only used within this package
func isValidAlias(alias string) bool {
	// Alias must be 3-20 characters
	if len(alias) < 3 || len(alias) > 20 {
		return false
	}

	// Alias must be alphanumeric (letters and numbers only)
	for _, char := range alias {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return false
		}
	}

	return true
}

// NewURL is a constructor function that creates a new URL with sensible defaults
// In Go, we use constructor functions instead of class constructors
func NewURL(originalURL, shortCode, createdBy string) *URL {
	return &URL{
		OriginalURL: originalURL,
		ShortCode:   shortCode,
		CreatedAt:   time.Now(),
		CreatedBy:   createdBy,
		IsActive:    true,
		Clicks:      0,
	}
}

// WithCustomAlias is a builder method that sets a custom alias
// This is the "builder pattern" - allows for fluent API design
func (u *URL) WithCustomAlias(alias string) *URL {
	u.CustomAlias = &alias
	return u
}

// WithExpiration sets an expiration time for the URL
func (u *URL) WithExpiration(duration time.Duration) *URL {
	expiresAt := time.Now().Add(duration)
	u.ExpiresAt = &expiresAt
	return u
}
