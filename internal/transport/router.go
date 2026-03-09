package transport

import (
	"github.com/gabxlbellior/recommendation-service/internal/config"
	"github.com/gabxlbellior/recommendation-service/internal/handler"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Health         *handler.HealthHandler
	Recommendation *handler.RecommendationHandler
}

func NewRouter(cfg config.Config, handlers Handlers) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/health", handlers.Health.Health)
	router.GET("/users/:user_id/recommendations", handlers.Recommendation.GetUserRecommendations)
	router.GET("/recommendations/batch", handlers.Recommendation.GetBatchRecommendations)

	return router
}
