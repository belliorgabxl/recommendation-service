package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/gabxlbellior/recommendation-service/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	fixedSeed    int64 = 20260309
	userCount          = 200
	contentCount       = 1000
	historyCount       = 5000
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN())
	if err != nil {
		log.Fatalf("failed to create pg pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping postgres: %v", err)
	}

	if err := seed(ctx, pool); err != nil {
		log.Fatalf("seed failed: %v", err)
	}

	log.Println("seed completed successfully")
}

func seed(ctx context.Context, pool *pgxpool.Pool) error {
	r := rand.New(rand.NewSource(fixedSeed))
	baseTime := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		TRUNCATE TABLE user_watch_history, content, users RESTART IDENTITY CASCADE;
	`); err != nil {
		return fmt.Errorf("truncate tables: %w", err)
	}

	if err := seedUsers(ctx, tx, r, baseTime); err != nil {
		return err
	}

	genreToContentIDs, err := seedContent(ctx, tx, r, baseTime)
	if err != nil {
		return err
	}

	if err := seedWatchHistory(ctx, tx, r, baseTime, genreToContentIDs); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	log.Printf("seeded users=%d content=%d watch_history=%d", userCount, contentCount, historyCount)
	return nil
}

func seedUsers(ctx context.Context, tx pgx.Tx, r *rand.Rand, baseTime time.Time) error {
	countries := []string{"TH", "US", "GB", "JP", "SG", "AU", "KR"}
	subscriptions := []string{"free", "basic", "premium"}

	rows := make([][]any, 0, userCount)

	for i := 0; i < userCount; i++ {
		age := 18 + r.Intn(48)
		country := weightedCountry(r, countries)
		subscription := weightedSubscription(r, subscriptions)
		createdAt := baseTime.Add(-time.Duration(r.Intn(900*24)) * time.Hour)

		rows = append(rows, []any{
			age,
			country,
			subscription,
			createdAt,
		})
	}

	_, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"users"},
		[]string{"age", "country", "subscription_type", "created_at"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("copy users: %w", err)
	}

	return nil
}

func seedContent(ctx context.Context, tx pgx.Tx, r *rand.Rand, baseTime time.Time) (map[string][]int64, error) {
	genres := []string{
		"action",
		"drama",
		"comedy",
		"sci-fi",
		"documentary",
		"thriller",
		"romance",
		"animation",
	}

	rows := make([][]any, 0, contentCount)
	genreToContentIDs := make(map[string][]int64, len(genres))

	for i := 0; i < contentCount; i++ {
		genre := weightedGenre(r, genres)
		title := fmt.Sprintf("%s Content %04d", genreTitlePrefix(genre), i+1)

		// long-tail distribution: many low-popularity items, few high-popularity items
		popularityScore := 0.01 + 0.99*math.Pow(r.Float64(), 2.8)
		if popularityScore > 1 {
			popularityScore = 1
		}

		createdAt := baseTime.Add(-time.Duration(r.Intn(730*24)) * time.Hour)

		rows = append(rows, []any{
			title,
			genre,
			popularityScore,
			createdAt,
		})

		// because we TRUNCATE ... RESTART IDENTITY before CopyFrom
		contentID := int64(i + 1)
		genreToContentIDs[genre] = append(genreToContentIDs[genre], contentID)
	}

	_, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"content"},
		[]string{"title", "genre", "popularity_score", "created_at"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return nil, fmt.Errorf("copy content: %w", err)
	}

	return genreToContentIDs, nil
}

func seedWatchHistory(ctx context.Context, tx pgx.Tx, r *rand.Rand, baseTime time.Time, genreToContentIDs map[string][]int64) error {
	genres := []string{
		"action",
		"drama",
		"comedy",
		"sci-fi",
		"documentary",
		"thriller",
		"romance",
		"animation",
	}

	userPreferences := make(map[int64][]string, userCount)

	for userID := int64(1); userID <= userCount; userID++ {
		first := genres[r.Intn(len(genres))]
		second := genres[r.Intn(len(genres))]
		for second == first {
			second = genres[r.Intn(len(genres))]
		}
		userPreferences[userID] = []string{first, second}
	}

	rows := make([][]any, 0, historyCount)

	for i := 0; i < historyCount; i++ {
		userID := int64(1 + r.Intn(userCount))
		preferred := userPreferences[userID]

		genre := pickPreferredGenre(r, preferred, genres)
		contentIDs := genreToContentIDs[genre]
		if len(contentIDs) == 0 {
			continue
		}

		contentID := contentIDs[r.Intn(len(contentIDs))]
		watchedAt := baseTime.Add(-time.Duration(r.Intn(365*24)) * time.Hour)

		rows = append(rows, []any{
			userID,
			contentID,
			watchedAt,
		})
	}

	_, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"user_watch_history"},
		[]string{"user_id", "content_id", "watched_at"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("copy user_watch_history: %w", err)
	}

	return nil
}

func weightedCountry(r *rand.Rand, countries []string) string {
	n := r.Intn(100)
	switch {
	case n < 40:
		return "TH"
	case n < 55:
		return "US"
	case n < 65:
		return "JP"
	case n < 75:
		return "SG"
	case n < 85:
		return "GB"
	case n < 93:
		return "AU"
	default:
		return countries[r.Intn(len(countries))]
	}
}

func weightedSubscription(r *rand.Rand, _ []string) string {
	n := r.Intn(100)
	switch {
	case n < 60:
		return "free"
	case n < 85:
		return "basic"
	default:
		return "premium"
	}
}

func weightedGenre(r *rand.Rand, genres []string) string {
	n := r.Intn(100)
	switch {
	case n < 22:
		return "drama"
	case n < 40:
		return "action"
	case n < 54:
		return "comedy"
	case n < 66:
		return "thriller"
	case n < 76:
		return "sci-fi"
	case n < 84:
		return "romance"
	case n < 92:
		return "documentary"
	default:
		return genres[r.Intn(len(genres))]
	}
}

func pickPreferredGenre(r *rand.Rand, preferred []string, all []string) string {
	n := r.Intn(100)
	switch {
	case n < 65:
		return preferred[0]
	case n < 90:
		return preferred[1]
	default:
		return all[r.Intn(len(all))]
	}
}

func genreTitlePrefix(genre string) string {
	switch genre {
	case "action":
		return "Action"
	case "drama":
		return "Drama"
	case "comedy":
		return "Comedy"
	case "sci-fi":
		return "SciFi"
	case "documentary":
		return "Doc"
	case "thriller":
		return "Thriller"
	case "romance":
		return "Romance"
	case "animation":
		return "Animation"
	default:
		return "Content"
	}
}
