package main

import (
	"context"
	"log"
	"time"

	"github.com/gabxlbellior/recommendation-service/internal/cache"
	"github.com/gabxlbellior/recommendation-service/internal/config"
	"github.com/gabxlbellior/recommendation-service/internal/handler"
	"github.com/gabxlbellior/recommendation-service/internal/model.go"
	"github.com/gabxlbellior/recommendation-service/internal/repository"
	"github.com/gabxlbellior/recommendation-service/internal/service"
	"github.com/gabxlbellior/recommendation-service/internal/transport"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := repository.NewPostgresPool(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to connect postgres: %v", err)
	}
	defer dbPool.Close()

	recommendationCache, err := cache.NewRedisRecommendationCache(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}
	defer recommendationCache.Close()

	userRepo := repository.NewUserRepository(dbPool)
	contentRepo := repository.NewContentRepository(dbPool)
	watchHistoryRepo := repository.NewWatchHistoryRepository(dbPool)

	scorer := model.NewScorer()

	recommendationService := service.NewRecommendationService(
		userRepo,
		contentRepo,
		watchHistoryRepo,
		scorer,
		recommendationCache,
		cfg.RecommendationCacheTTL,
	)

	healthHandler := handler.NewHealthHandler(cfg)
	recommendationHandler := handler.NewRecommendationHandler(recommendationService)

	router := transport.NewRouter(cfg, transport.Handlers{
		Health:         healthHandler,
		Recommendation: recommendationHandler,
	})

	addr := ":" + cfg.AppPort
	log.Printf("starting server on %s (%s)", addr, cfg.AppEnv)

	if err := router.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
