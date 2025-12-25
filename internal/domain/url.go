// ============================================================================
// DOMAIN LAYER - URL.GO
// ============================================================================
// This file defines the URL domain model and its business logic.
//
// KEY CONCEPTS:
// 1. Domain-Driven Design (DDD) - Business logic lives in domain models
// 2. Structs - Custom data types (like C# classes)
// 3. Methods - Functions attached to structs (like C# instance methods)
// 4. Pointers - Nullable fields and memory efficiency
// 5. Builder Pattern - Fluent API for object construction
//
// .NET COMPARISON: This is like a C# entity class with validation methods
// ============================================================================

package domain

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

// ============================================================================
// URL STRUCT - THE DOMAIN MODEL
// ============================================================================
// URL represents a shortened URL in our system.
// This is our "domain model" - it contains both DATA and BEHAVIOR (methods).
//
// STRUCT vs CLASS:
// Go structs are similar to C# classes but simpler:
// - No inheritance (composition over inheritance)
// - No constructors (use factory functions instead)
// - Methods are defined separately with receivers
//
// POINTERS FOR NULLABLE FIELDS:
// - string     → Required field (cannot be null)
// - *string    → Optional field (can be nil)
// - time.Time  → Required timestamp
// - *time.Time → Optional timestamp
//
// .NET EQUIVALENT:
//
//	public class URL {
//	    public string ID { get; set; }
//	    public string ShortCode { get; set; }
//	    public string OriginalURL { get; set; }
//	    public string? CustomAlias { get; set; }  // Nullable
//	    public DateTime CreatedAt { get; set; }
//	    public DateTime? ExpiresAt { get; set; }  // Nullable
//	    public long Clicks { get; set; }
//	    public string CreatedBy { get; set; }
//	    public bool IsActive { get; set; }
//	}
//
// ============================================================================
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

// ============================================================================
// DOMAIN ERRORS - SENTINEL ERRORS
// ============================================================================
// Defining errors as package-level variables (sentinel errors) allows:
// 1. Callers to check for specific error types using errors.Is()
// 2. Consistent error messages across the application
// 3. Type-safe error handling
//
// USAGE:
//
//	if errors.Is(err, domain.ErrURLExpired) {
//	    // Handle expired URL specifically
//	}
//
// .NET EQUIVALENT:
// public class URLExpiredException : Exception { }
// throw new URLExpiredException("URL has expired");
// ============================================================================
var (
	ErrInvalidURL         = errors.New("invalid URL format")
	ErrEmptyURL           = errors.New("URL cannot be empty")
	ErrShortCodeTooShort  = errors.New("short code must be at least 3 characters")
	ErrURLExpired         = errors.New("URL has expired")
	ErrURLNotActive       = errors.New("URL is not active")
	ErrCustomAliasInvalid = errors.New("custom alias must be alphanumeric and 3-20 characters")
)

// ============================================================================
// METHODS - BEHAVIOR ATTACHED TO THE STRUCT
// ============================================================================
// IsExpired checks if the URL has passed its expiration time.
//
// METHOD SYNTAX:
// func (u *URL) IsExpired() bool
//
//	↑      ↑
//	|      └─ Receiver: gives method access to struct fields
//	└──────── Pointer receiver: can read/modify the struct
//
// POINTER RECEIVER (*URL) vs VALUE RECEIVER (URL):
// - *URL: Can modify the struct, more efficient for large structs
// - URL:  Read-only copy, safe but less efficient
//
// RULE OF THUMB: Use pointer receivers unless you have a reason not to
//
// .NET EQUIVALENT:
//
//	public bool IsExpired() {
//	    if (ExpiresAt == null) return false;
//	    return DateTime.Now > ExpiresAt.Value;
//	}
//
// ============================================================================
func (u *URL) IsExpired() bool {
	// POINTER DEREFERENCING:
	// u.ExpiresAt is *time.Time (pointer to time.Time)
	// *u.ExpiresAt dereferences the pointer to get the actual time.Time value

	// If ExpiresAt is nil (not set), the URL never expires
	if u.ExpiresAt == nil {
		return false
	}
	// Check if current time is after expiration time
	return time.Now().After(*u.ExpiresAt) // * = dereference pointer
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
