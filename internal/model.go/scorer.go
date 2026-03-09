package model

import (
	"context"
	"errors"
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/gabxlbellior/recommendation-service/internal/domain"
)

var ErrModelUnavailable = errors.New("model unavailable")

type Scorer struct{}

func NewScorer() *Scorer {
	return &Scorer{}
}

func (s *Scorer) Score(ctx context.Context, input Input) ([]domain.RecommendationItem, error) {
	if input.User == nil {
		return nil, errors.New("user is required")
	}

	if err := simulateModelLatency(ctx, input.User.ID); err != nil {
		return nil, err
	}

	if shouldFailModel(input.User.ID) {
		return nil, ErrModelUnavailable
	}

	genrePreference := buildGenrePreference(input.WatchHistory)
	recommendations := make([]domain.RecommendationItem, 0, len(input.Candidates))

	for _, candidate := range input.Candidates {
		score := s.calculateScore(input.User.ID, candidate, genrePreference)

		recommendations = append(recommendations, domain.RecommendationItem{
			ContentID:       candidate.ID,
			Title:           candidate.Title,
			Genre:           candidate.Genre,
			PopularityScore: candidate.PopularityScore,
			Score:           round(score, 6),
		})
	}

	sort.SliceStable(recommendations, func(i, j int) bool {
		if recommendations[i].Score == recommendations[j].Score {
			if recommendations[i].PopularityScore == recommendations[j].PopularityScore {
				return recommendations[i].ContentID < recommendations[j].ContentID
			}
			return recommendations[i].PopularityScore > recommendations[j].PopularityScore
		}
		return recommendations[i].Score > recommendations[j].Score
	})

	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > len(recommendations) {
		limit = len(recommendations)
	}

	return recommendations[:limit], nil
}

func (s *Scorer) calculateScore(userID int64, candidate domain.Content, genrePreference map[string]float64) float64 {
	popularityComponent := candidate.PopularityScore * 0.40
	genreComponent := genrePreference[candidate.Genre] * 0.35
	recencyComponent := recencyFactor(candidate.CreatedAt) * 0.15
	explorationComponent := deterministicNoise(userID, candidate.ID) * 0.10

	return popularityComponent + genreComponent + recencyComponent + explorationComponent
}

func buildGenrePreference(history []domain.WatchHistoryWithGenre) map[string]float64 {
	preference := make(map[string]float64)
	if len(history) == 0 {
		return preference
	}

	counts := make(map[string]int)
	total := 0

	for _, item := range history {
		if item.Genre == "" {
			continue
		}
		counts[item.Genre]++
		total++
	}

	if total == 0 {
		return preference
	}

	for genre, count := range counts {
		preference[genre] = float64(count) / float64(total)
	}

	return preference
}

func recencyFactor(createdAt time.Time) float64 {
	days := time.Since(createdAt).Hours() / 24
	if days < 0 {
		days = 0
	}

	return 1.0 / (1.0 + (days / 365.0))
}

func deterministicNoise(userID, contentID int64) float64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(strconv.FormatInt(userID, 10) + ":" + strconv.FormatInt(contentID, 10)))

	base := float64(h.Sum64()%10000) / 10000.0
	return (base * 0.10) - 0.05
}

func shouldFailModel(userID int64) bool {
	h := fnv.New64a()
	_, _ = h.Write([]byte("model-fail:" + strconv.FormatInt(userID, 10)))
	value := h.Sum64() % 1000
	return value < 15
}

func simulateModelLatency(ctx context.Context, userID int64) error {
	h := fnv.New64a()
	_, _ = h.Write([]byte("latency:" + strconv.FormatInt(userID, 10)))

	latencyMs := 30 + int(h.Sum64()%21)

	timer := time.NewTimer(time.Duration(latencyMs) * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func round(v float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(v*pow) / pow
}