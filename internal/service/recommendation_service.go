package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gabxlbellior/recommendation-service/internal/cache"
	"github.com/gabxlbellior/recommendation-service/internal/domain"
	"github.com/gabxlbellior/recommendation-service/internal/model.go"
	"github.com/gabxlbellior/recommendation-service/internal/repository"
)

var (
	ErrInvalidUserID    = errors.New("invalid user id")
	ErrInvalidLimit     = errors.New("invalid limit")
	ErrInvalidPage      = errors.New("invalid page")
	ErrInvalidBatchSize = errors.New("invalid batch size")
)

const (
	maxSingleLimit                    = 50
	defaultBatchRecommendationLimit   = 10
	defaultBatchHistoryLimit          = 50
	defaultBatchPopularContentPool    = 300
	defaultBatchWorkerCount           = 8
	defaultBatchPerUserProcessTimeout = 250 * time.Millisecond
	maxBatchPageSize                  = 100
)

type RecommendationService interface {
	GetUserRecommendations(ctx context.Context, userID int64, limit int) (*domain.RecommendationResponse, error)
	GetBatchRecommendations(ctx context.Context, page, limit int) (*domain.BatchRecommendationResponse, error)
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

	if limit <= 0 || limit > maxSingleLimit {
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

func (s *recommendationService) GetBatchRecommendations(ctx context.Context, page, limit int) (*domain.BatchRecommendationResponse, error) {
	if page <= 0 {
		return nil, ErrInvalidPage
	}

	if limit <= 0 || limit > maxBatchPageSize {
		return nil, ErrInvalidBatchSize
	}

	startedAt := time.Now()

	totalUsers, err := s.userRepo.CountUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	users, err := s.userRepo.ListUsers(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	response := &domain.BatchRecommendationResponse{
		Page:       page,
		Limit:      limit,
		TotalUsers: totalUsers,
		Results:    []domain.BatchRecommendationResult{},
		Summary: domain.BatchRecommendationSummary{
			SuccessCount:     0,
			FailedCount:      0,
			ProcessingTimeMs: 0,
		},
		Metadata: domain.BatchRecommendationMetadata{
			GeneratedAt: time.Now().UTC(),
		},
	}

	if len(users) == 0 {
		response.Summary.ProcessingTimeMs = time.Since(startedAt).Milliseconds()
		return response, nil
	}

	userIDs := make([]int64, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	historiesByUser, err := s.watchHistoryRepo.GetWatchHistoriesForUsers(ctx, userIDs, defaultBatchHistoryLimit)
	if err != nil {
		return nil, fmt.Errorf("get watch histories for users: %w", err)
	}

	watchedContentIDsByUser, err := s.watchHistoryRepo.GetWatchedContentIDsForUsers(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("get watched content ids for users: %w", err)
	}

	popularCandidates, err := s.contentRepo.GetPopularContent(ctx, defaultBatchPopularContentPool)
	if err != nil {
		return nil, fmt.Errorf("get popular content: %w", err)
	}

	results := s.processBatchUsers(
		ctx,
		users,
		historiesByUser,
		watchedContentIDsByUser,
		popularCandidates,
		defaultBatchRecommendationLimit,
	)

	successCount := 0
	failedCount := 0

	for _, result := range results {
		if result.Status == "success" {
			successCount++
		} else {
			failedCount++
		}
	}

	response.Results = results
	response.Summary = domain.BatchRecommendationSummary{
		SuccessCount:     successCount,
		FailedCount:      failedCount,
		ProcessingTimeMs: time.Since(startedAt).Milliseconds(),
	}

	return response, nil
}

type batchJob struct {
	Index int
	User  domain.User
}

type batchResult struct {
	Index  int
	Result domain.BatchRecommendationResult
}

func (s *recommendationService) processBatchUsers(
	ctx context.Context,
	users []domain.User,
	historiesByUser map[int64][]domain.WatchHistoryWithGenre,
	watchedContentIDsByUser map[int64]map[int64]struct{},
	popularCandidates []domain.Content,
	recommendationLimit int,
) []domain.BatchRecommendationResult {
	if len(users) == 0 {
		return []domain.BatchRecommendationResult{}
	}

	jobs := make(chan batchJob)
	resultsCh := make(chan batchResult, len(users))

	workerCount := minInt(defaultBatchWorkerCount, len(users))

	var wg sync.WaitGroup
	wg.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()

			for job := range jobs {
				resultsCh <- batchResult{
					Index: job.Index,
					Result: s.processBatchUser(
						ctx,
						job.User,
						historiesByUser[job.User.ID],
						watchedContentIDsByUser[job.User.ID],
						popularCandidates,
						recommendationLimit,
					),
				}
			}
		}()
	}

	for idx, user := range users {
		jobs <- batchJob{
			Index: idx,
			User:  user,
		}
	}
	close(jobs)

	wg.Wait()
	close(resultsCh)

	orderedResults := make([]domain.BatchRecommendationResult, len(users))
	for item := range resultsCh {
		orderedResults[item.Index] = item.Result
	}

	return orderedResults
}

func (s *recommendationService) processBatchUser(
	ctx context.Context,
	user domain.User,
	history []domain.WatchHistoryWithGenre,
	watchedContentIDs map[int64]struct{},
	popularCandidates []domain.Content,
	recommendationLimit int,
) domain.BatchRecommendationResult {
	userCtx, cancel := context.WithTimeout(ctx, defaultBatchPerUserProcessTimeout)
	defer cancel()

	filteredCandidates := filterCandidatesForUser(
		popularCandidates,
		watchedContentIDs,
		calculateCandidateLimit(recommendationLimit),
	)

	recommendations, err := s.scorer.Score(userCtx, model.Input{
		User:         &user,
		WatchHistory: history,
		Candidates:   filteredCandidates,
		Limit:        recommendationLimit,
	})
	if err != nil {
		return buildBatchFailureResult(user.ID, err)
	}

	return domain.BatchRecommendationResult{
		UserID:          user.ID,
		Recommendations: recommendations,
		Status:          "success",
	}
}

func filterCandidatesForUser(
	candidates []domain.Content,
	watchedContentIDs map[int64]struct{},
	limit int,
) []domain.Content {
	if limit <= 0 {
		limit = 100
	}

	filtered := make([]domain.Content, 0, limit)

	for _, candidate := range candidates {
		if _, watched := watchedContentIDs[candidate.ID]; watched {
			continue
		}

		filtered = append(filtered, candidate)

		if len(filtered) >= limit {
			break
		}
	}

	return filtered
}

func buildBatchFailureResult(userID int64, err error) domain.BatchRecommendationResult {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return domain.BatchRecommendationResult{
			UserID:  userID,
			Status:  "failed",
			Error:   "model_inference_timeout",
			Message: "recommendation generation exceeded timeout limit",
		}

	case errors.Is(err, model.ErrModelUnavailable):
		return domain.BatchRecommendationResult{
			UserID:  userID,
			Status:  "failed",
			Error:   "model_unavailable",
			Message: "recommendation model is temporarily unavailable",
		}

	default:
		return domain.BatchRecommendationResult{
			UserID:  userID,
			Status:  "failed",
			Error:   "internal_error",
			Message: "failed to generate recommendations",
		}
	}
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
