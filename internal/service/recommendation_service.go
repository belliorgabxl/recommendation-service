package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gabxlbellior/recommendation-service/internal/cache"
	"github.com/gabxlbellior/recommendation-service/internal/domain"
	"github.com/gabxlbellior/recommendation-service/internal/model.go"
	"github.com/gabxlbellior/recommendation-service/internal/repository"
)

var (
	ErrInvalidUserID = errors.New("invalid user id")
	ErrInvalidLimit  = errors.New("invalid limit")
)

type RecommendationService interface {
	GetUserRecommendations(ctx context.Context, userID int64, limit int) (*domain.RecommendationResponse, error)
}

type recommendationService struct {
	userRepo            repository.UserRepository
	contentRepo         repository.ContentRepository
	watchHistoryRepo    repository.WatchHistoryRepository
	scorer              *model.Scorer
	recommendationCache cache.RecommendationCache
	cacheTTL            time.Duration
}

func NewRecommendationService(
	userRepo repository.UserRepository,
	contentRepo repository.ContentRepository,
	watchHistoryRepo repository.WatchHistoryRepository,
	scorer *model.Scorer,
	recommendationCache cache.RecommendationCache,
	cacheTTL time.Duration,
) RecommendationService {
	return &recommendationService{
		userRepo:            userRepo,
		contentRepo:         contentRepo,
		watchHistoryRepo:    watchHistoryRepo,
		scorer:              scorer,
		recommendationCache: recommendationCache,
		cacheTTL:            cacheTTL,
	}
}

func (s *recommendationService) GetUserRecommendations(ctx context.Context, userID int64, limit int) (*domain.RecommendationResponse, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}

	if limit <= 0 || limit > 50 {
		return nil, ErrInvalidLimit
	}

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	latestWatchedAt, err := s.watchHistoryRepo.GetLatestWatchedAt(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get latest watched at: %w", err)
	}

	cacheKey := cache.BuildRecommendationKey(userID, limit, latestWatchedAt)

	if s.recommendationCache != nil {
		cachedResp, hit, cacheErr := s.recommendationCache.Get(ctx, cacheKey)
		if cacheErr == nil && hit {
			cachedResp.Metadata.CacheHit = true
			return cachedResp, nil
		}
	}

	history, err := s.watchHistoryRepo.GetUserWatchHistoryWithGenres(ctx, userID, 50)
	if err != nil {
		return nil, fmt.Errorf("get user watch history: %w", err)
	}

	candidateLimit := calculateCandidateLimit(limit)
	candidates, err := s.contentRepo.GetUnwatchedContent(ctx, userID, candidateLimit)
	if err != nil {
		return nil, fmt.Errorf("get candidate content: %w", err)
	}

	recommendations, err := s.scorer.Score(ctx, model.Input{
		User:         user,
		WatchHistory: history,
		Candidates:   candidates,
		Limit:        limit,
	})
	if err != nil {
		return nil, err
	}

	resp := &domain.RecommendationResponse{
		UserID:          userID,
		Recommendations: recommendations,
		Metadata: domain.RecommendationMetadata{
			CacheHit:    false,
			GeneratedAt: time.Now().UTC(),
			TotalCount:  len(recommendations),
		},
	}

	if s.recommendationCache != nil {
		_ = s.recommendationCache.Set(ctx, cacheKey, resp, s.cacheTTL)
	}

	return resp, nil
}

func calculateCandidateLimit(limit int) int {
	if limit <= 0 {
		return 100
	}

	candidateLimit := limit * 10
	if candidateLimit < 100 {
		candidateLimit = 100
	}
	if candidateLimit > 300 {
		candidateLimit = 300
	}

	return candidateLimit
}
