package postgres

import (
	"context"
	"fmt"

	"url-shortener/internal/domain"
	"url-shortener/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// clickRepository is the PostgreSQL implementation for analytics
type clickRepository struct {
	db *pgxpool.Pool
}

// NewClickRepository creates a new PostgreSQL click repository
func NewClickRepository(db *pgxpool.Pool) repository.ClickRepository {
	return &clickRepository{db: db}
}

// Create inserts a new click event into the database
func (r *clickRepository) Create(ctx context.Context, click *domain.URLClick) error {
	query := `
		INSERT INTO url_clicks (
			url_id, clicked_at, ip_address, user_agent,
			referer, country_code, city
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) RETURNING id
	`

	err := r.db.QueryRow(
		ctx,
		query,
		click.URLID,
		click.ClickedAt,
		click.IPAddress,
		click.UserAgent,
		click.Referer,
		click.CountryCode,
		click.City,
	).Scan(&click.ID)

	if err != nil {
		return fmt.Errorf("failed to create click event: %w", err)
	}

	return nil
}

// GetByURLID retrieves clicks for a specific URL with pagination
func (r *clickRepository) GetByURLID(ctx context.Context, urlID string, limit, offset int) ([]*domain.URLClick, error) {
	query := `
		SELECT id, url_id, clicked_at, ip_address, user_agent,
		       referer, country_code, city
		FROM url_clicks
		WHERE url_id = $1
		ORDER BY clicked_at DESC
		LIMIT $2 OFFSET $3
	`

	// Query returns multiple rows, so we use Query instead of QueryRow
	rows, err := r.db.Query(ctx, query, urlID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get clicks: %w", err)
	}
	defer rows.Close() // Always close rows to free resources

	// Collect all clicks into a slice
	var clicks []*domain.URLClick
	for rows.Next() {
		click := &domain.URLClick{}
		err := rows.Scan(
			&click.ID,
			&click.URLID,
			&click.ClickedAt,
			&click.IPAddress,
			&click.UserAgent,
			&click.Referer,
			&click.CountryCode,
			&click.City,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan click: %w", err)
		}
		clicks = append(clicks, click)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating clicks: %w", err)
	}

	return clicks, nil
}

// GetClickCount returns the total number of clicks for a URL
func (r *clickRepository) GetClickCount(ctx context.Context, urlID string) (int64, error) {
	query := `SELECT COUNT(*) FROM url_clicks WHERE url_id = $1`

	var count int64
	err := r.db.QueryRow(ctx, query, urlID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get click count: %w", err)
	}

	return count, nil
}
