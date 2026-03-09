package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/gabxlbellior/recommendation-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	GetUserByID(ctx context.Context, userID int64) (*domain.User, error)
	ListUsers(ctx context.Context, page, limit int) ([]domain.User, error)
	CountUsers(ctx context.Context) (int64, error)
}

type PostgresUserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) GetUserByID(ctx context.Context, userID int64) (*domain.User, error) {
	const query = `
		SELECT id, age, country, subscription_type, created_at
		FROM users
		WHERE id = $1
	`

	var user domain.User

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Age,
		&user.Country,
		&user.SubscriptionType,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &user, nil
}

func (r *PostgresUserRepository) ListUsers(ctx context.Context, page, limit int) ([]domain.User, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	offset := (page - 1) * limit

	const query = `
		SELECT id, age, country, subscription_type, created_at
		FROM users
		ORDER BY id ASC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0, limit)

	for rows.Next() {
		var user domain.User
		if err := rows.Scan(
			&user.ID,
			&user.Age,
			&user.Country,
			&user.SubscriptionType,
			&user.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user rows: %w", err)
	}

	return users, nil
}

func (r *PostgresUserRepository) CountUsers(ctx context.Context) (int64, error) {
	const query = `SELECT COUNT(*) FROM users`

	var count int64
	if err := r.db.QueryRow(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}

	return count, nil
}
