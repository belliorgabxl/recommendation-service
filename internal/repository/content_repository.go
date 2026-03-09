package repository

import (
	"context"
	"fmt"

	"github.com/gabxlbellior/recommendation-service/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ContentRepository interface {
	GetUnwatchedContent(ctx context.Context, userID int64, limit int) ([]domain.Content, error)
	GetPopularContent(ctx context.Context, limit int) ([]domain.Content, error)
}

type PostgresContentRepository struct {
	db *pgxpool.Pool
}

func NewContentRepository(db *pgxpool.Pool) ContentRepository {
	return &PostgresContentRepository{db: db}
}

func (r *PostgresContentRepository) GetUnwatchedContent(ctx context.Context, userID int64, limit int) ([]domain.Content, error) {
	if limit < 1 {
		limit = 100
	}

	const query = `
		SELECT
			c.id,
			c.title,
			c.genre,
			c.popularity_score,
			c.created_at
		FROM content c
		WHERE NOT EXISTS (
			SELECT 1
			FROM user_watch_history uwh
			WHERE uwh.user_id = $1
			  AND uwh.content_id = c.id
		)
		ORDER BY c.popularity_score DESC, c.created_at DESC, c.id ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get unwatched content: %w", err)
	}
	defer rows.Close()

	contents := make([]domain.Content, 0, limit)

	for rows.Next() {
		var content domain.Content
		if err := rows.Scan(
			&content.ID,
			&content.Title,
			&content.Genre,
			&content.PopularityScore,
			&content.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan unwatched content row: %w", err)
		}

		contents = append(contents, content)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unwatched content rows: %w", err)
	}

	return contents, nil
}

func (r *PostgresContentRepository) GetPopularContent(ctx context.Context, limit int) ([]domain.Content, error) {
	if limit < 1 {
		limit = 100
	}

	const query = `
		SELECT
			id,
			title,
			genre,
			popularity_score,
			created_at
		FROM content
		ORDER BY popularity_score DESC, created_at DESC, id ASC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get popular content: %w", err)
	}
	defer rows.Close()

	contents := make([]domain.Content, 0, limit)

	for rows.Next() {
		var content domain.Content
		if err := rows.Scan(
			&content.ID,
			&content.Title,
			&content.Genre,
			&content.PopularityScore,
			&content.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan popular content row: %w", err)
		}

		contents = append(contents, content)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate popular content rows: %w", err)
	}

	return contents, nil
}
