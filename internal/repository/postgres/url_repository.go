package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"url-shortener/internal/domain"
	"url-shortener/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// urlRepository is the PostgreSQL implementation of repository.URLRepository
// The lowercase name means it's private to this package
// We return it as the interface type (repository.URLRepository) for abstraction
type urlRepository struct {
	db *pgxpool.Pool // Connection pool for database connections
}

// NewURLRepository creates a new PostgreSQL URL repository
// This is a constructor function that returns the interface type
//
// CONNECTION POOLING:
// Instead of opening a new connection for each query (slow!),
// we maintain a pool of reusable connections. This dramatically improves performance.
func NewURLRepository(db *pgxpool.Pool) repository.URLRepository {
	return &urlRepository{db: db}
}

// Create inserts a new URL into the database
func (r *urlRepository) Create(ctx context.Context, url *domain.URL) error {
	// SQL query with placeholders ($1, $2, etc.) to prevent SQL injection
	// RETURNING id returns the generated UUID after insertion
	query := `
		INSERT INTO urls (
			short_code, original_url, custom_alias, created_at,
			expires_at, created_by, is_active, clicks
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING id
	`

	// QueryRow executes the query and scans the result into url.ID
	// ctx is used for timeouts and cancellation
	err := r.db.QueryRow(
		ctx,
		query,
		url.ShortCode,
		url.OriginalURL,
		url.CustomAlias, // Can be nil (NULL in database)
		url.CreatedAt,
		url.ExpiresAt, // Can be nil (NULL in database)
		url.CreatedBy,
		url.IsActive,
		url.Clicks,
	).Scan(&url.ID)

	if err != nil {
		// Wrap the error with context for better debugging
		return fmt.Errorf("failed to create URL: %w", err)
	}

	return nil
}

// GetByShortCode retrieves a URL by its short code
func (r *urlRepository) GetByShortCode(ctx context.Context, shortCode string) (*domain.URL, error) {
	query := `
		SELECT id, short_code, original_url, custom_alias, created_at,
		       expires_at, clicks, created_by, is_active
		FROM urls
		WHERE short_code = $1 AND is_active = true
	`

	url := &domain.URL{}

	// QueryRow returns a single row
	err := r.db.QueryRow(ctx, query, shortCode).Scan(
		&url.ID,
		&url.ShortCode,
		&url.OriginalURL,
		&url.CustomAlias, // pgx handles NULL -> nil conversion automatically
		&url.CreatedAt,
		&url.ExpiresAt,
		&url.Clicks,
		&url.CreatedBy,
		&url.IsActive,
	)

	if err != nil {
		// pgx.ErrNoRows is returned when no rows match the query
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("URL not found: %s", shortCode)
		}
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	return url, nil
}

// GetByID retrieves a URL by its UUID
func (r *urlRepository) GetByID(ctx context.Context, id string) (*domain.URL, error) {
	query := `
		SELECT id, short_code, original_url, custom_alias, created_at,
		       expires_at, clicks, created_by, is_active
		FROM urls
		WHERE id = $1
	`

	url := &domain.URL{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&url.ID,
		&url.ShortCode,
		&url.OriginalURL,
		&url.CustomAlias,
		&url.CreatedAt,
		&url.ExpiresAt,
		&url.Clicks,
		&url.CreatedBy,
		&url.IsActive,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("URL not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	return url, nil
}

// GetByCustomAlias retrieves a URL by its custom alias
func (r *urlRepository) GetByCustomAlias(ctx context.Context, alias string) (*domain.URL, error) {
	query := `
		SELECT id, short_code, original_url, custom_alias, created_at,
		       expires_at, clicks, created_by, is_active
		FROM urls
		WHERE custom_alias = $1 AND is_active = true
	`

	url := &domain.URL{}
	err := r.db.QueryRow(ctx, query, alias).Scan(
		&url.ID,
		&url.ShortCode,
		&url.OriginalURL,
		&url.CustomAlias,
		&url.CreatedAt,
		&url.ExpiresAt,
		&url.Clicks,
		&url.CreatedBy,
		&url.IsActive,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("URL not found: %s", alias)
		}
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	return url, nil
}

// Update modifies an existing URL
func (r *urlRepository) Update(ctx context.Context, url *domain.URL) error {
	query := `
		UPDATE urls
		SET original_url = $1, custom_alias = $2, expires_at = $3, is_active = $4
		WHERE id = $5
	`

	// Exec executes a query that doesn't return rows
	result, err := r.db.Exec(
		ctx,
		query,
		url.OriginalURL,
		url.CustomAlias,
		url.ExpiresAt,
		url.IsActive,
		url.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update URL: %w", err)
	}

	// Check if any rows were affected
	if result.RowsAffected() == 0 {
		return fmt.Errorf("URL not found: %s", url.ID)
	}

	return nil
}

// Delete performs a soft delete (sets is_active = false)
func (r *urlRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE urls SET is_active = false WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete URL: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("URL not found: %s", id)
	}

	return nil
}

// IncrementClicks atomically increases the click counter
// ATOMIC OPERATION: This happens in a single database operation,
// preventing race conditions when multiple requests access the same URL simultaneously
func (r *urlRepository) IncrementClicks(ctx context.Context, shortCode string) error {
	query := `
		UPDATE urls
		SET clicks = clicks + 1
		WHERE short_code = $1 AND is_active = true
	`

	result, err := r.db.Exec(ctx, query, shortCode)
	if err != nil {
		return fmt.Errorf("failed to increment clicks: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("URL not found or inactive: %s", shortCode)
	}

	return nil
}

// ExistsShortCode checks if a short code already exists
func (r *urlRepository) ExistsShortCode(ctx context.Context, shortCode string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, shortCode).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check short code existence: %w", err)
	}

	return exists, nil
}

// ExistsCustomAlias checks if a custom alias is already taken
func (r *urlRepository) ExistsCustomAlias(ctx context.Context, alias string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM urls WHERE custom_alias = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, alias).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check custom alias existence: %w", err)
	}

	return exists, nil
}

// InitDB initializes the database connection pool
// This is called once at application startup
func InitDB(ctx context.Context, dsn string, maxConns, minConns int, maxLifetime time.Duration) (*pgxpool.Pool, error) {
	// Parse the connection string and create a config
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure connection pool settings
	config.MaxConns = int32(maxConns)          // Maximum number of connections
	config.MinConns = int32(minConns)          // Minimum number of idle connections
	config.MaxConnLifetime = maxLifetime       // Maximum lifetime of a connection
	config.MaxConnIdleTime = 30 * time.Minute  // Close idle connections after 30 minutes
	config.HealthCheckPeriod = 1 * time.Minute // Check connection health every minute

	// Create the connection pool
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
