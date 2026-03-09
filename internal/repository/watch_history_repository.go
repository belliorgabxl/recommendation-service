package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/gabxlbellior/recommendation-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type WatchHistoryRepository interface {
	GetUserWatchHistoryWithGenres(ctx context.Context, userID int64, limit int) ([]domain.WatchHistoryWithGenre, error)
	GetWatchHistoriesForUsers(ctx context.Context, userIDs []int64, perUserLimit int) (map[int64][]domain.WatchHistoryWithGenre, error)
	GetWatchedContentIDs(ctx context.Context, userID int64) (map[int64]struct{}, error)
	GetLatestWatchedAt(ctx context.Context, userID int64) (*time.Time, error)
}

type PostgresWatchHistoryRepository struct {
	db *pgxpool.Pool
}

func NewWatchHistoryRepository(db *pgxpool.Pool) WatchHistoryRepository {
	return &PostgresWatchHistoryRepository{db: db}
}

func (r *PostgresWatchHistoryRepository) GetUserWatchHistoryWithGenres(ctx context.Context, userID int64, limit int) ([]domain.WatchHistoryWithGenre, error) {
	if limit < 1 {
		limit = 50
	}

	const query = `
		SELECT
			uwh.user_id,
			uwh.content_id,
			c.genre,
			uwh.watched_at
		FROM user_watch_history uwh
		INNER JOIN content c ON c.id = uwh.content_id
		WHERE uwh.user_id = $1
		ORDER BY uwh.watched_at DESC, uwh.id DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("get user watch history with genres: %w", err)
	}
	defer rows.Close()

	histories := make([]domain.WatchHistoryWithGenre, 0, limit)

	for rows.Next() {
		var item domain.WatchHistoryWithGenre
		if err := rows.Scan(
			&item.UserID,
			&item.ContentID,
			&item.Genre,
			&item.WatchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user watch history row: %w", err)
		}

		histories = append(histories, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user watch history rows: %w", err)
	}

	return histories, nil
}

func (r *PostgresWatchHistoryRepository) GetWatchHistoriesForUsers(ctx context.Context, userIDs []int64, perUserLimit int) (map[int64][]domain.WatchHistoryWithGenre, error) {
	result := make(map[int64][]domain.WatchHistoryWithGenre)

	if len(userIDs) == 0 {
		return result, nil
	}

	if perUserLimit < 1 {
		perUserLimit = 50
	}

	const query = `
		WITH ranked_history AS (
			SELECT
				uwh.user_id,
				uwh.content_id,
				c.genre,
				uwh.watched_at,
				ROW_NUMBER() OVER (
					PARTITION BY uwh.user_id
					ORDER BY uwh.watched_at DESC, uwh.id DESC
				) AS rn
			FROM user_watch_history uwh
			INNER JOIN content c ON c.id = uwh.content_id
			WHERE uwh.user_id = ANY($1::bigint[])
		)
		SELECT
			user_id,
			content_id,
			genre,
			watched_at
		FROM ranked_history
		WHERE rn <= $2
		ORDER BY user_id ASC, watched_at DESC
	`

	rows, err := r.db.Query(ctx, query, userIDs, perUserLimit)
	if err != nil {
		return nil, fmt.Errorf("get watch histories for users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.WatchHistoryWithGenre
		if err := rows.Scan(
			&item.UserID,
			&item.ContentID,
			&item.Genre,
			&item.WatchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan watch histories for users row: %w", err)
		}

		result[item.UserID] = append(result[item.UserID], item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watch histories for users rows: %w", err)
	}

	return result, nil
}

func (r *PostgresWatchHistoryRepository) GetWatchedContentIDs(ctx context.Context, userID int64) (map[int64]struct{}, error) {
	const query = `
		SELECT DISTINCT content_id
		FROM user_watch_history
		WHERE user_id = $1
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get watched content ids: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]struct{})

	for rows.Next() {
		var contentID int64
		if err := rows.Scan(&contentID); err != nil {
			return nil, fmt.Errorf("scan watched content id row: %w", err)
		}
		result[contentID] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watched content id rows: %w", err)
	}

	return result, nil
}

func (r *PostgresWatchHistoryRepository) GetLatestWatchedAt(ctx context.Context, userID int64) (*time.Time, error) {
	const query = `
		SELECT watched_at
		FROM user_watch_history
		WHERE user_id = $1
		ORDER BY watched_at DESC, id DESC
		LIMIT 1
	`

	var watchedAt time.Time
	err := r.db.QueryRow(ctx, query, userID).Scan(&watchedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest watched at: %w", err)
	}

	return &watchedAt, nil
}
