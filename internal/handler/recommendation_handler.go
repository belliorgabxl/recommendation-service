package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gabxlbellior/recommendation-service/internal/model.go"
	"github.com/gabxlbellior/recommendation-service/internal/repository"
	"github.com/gabxlbellior/recommendation-service/internal/service"
	"github.com/gin-gonic/gin"
)

type RecommendationHandler struct {
	recommendationService service.RecommendationService
}

func NewRecommendationHandler(recommendationService service.RecommendationService) *RecommendationHandler {
	return &RecommendationHandler{
		recommendationService: recommendationService,
	}
}

func (h *RecommendationHandler) GetUserRecommendations(c *gin.Context) {
	userIDParam := c.Param("user_id")
	userID, err := strconv.ParseInt(userIDParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "invalid_parameter",
				"message": "user_id must be a valid integer",
			},
		})
		return
	}

	limit := 10
	limitParam := c.Query("limit")
	if limitParam != "" {
		parsedLimit, err := strconv.Atoi(limitParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "invalid_parameter",
					"message": "limit must be a valid integer",
				},
			})
			return
		}
		limit = parsedLimit
	}

	resp, err := h.recommendationService.GetUserRecommendations(c.Request.Context(), userID, limit)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidUserID):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "invalid_parameter",
					"message": "user_id must be greater than 0",
				},
			})
			return

		case errors.Is(err, service.ErrInvalidLimit):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "invalid_parameter",
					"message": "limit must be between 1 and 50",
				},
			})
			return

		case errors.Is(err, repository.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "user_not_found",
					"message": "user not found",
				},
			})
			return

		case errors.Is(err, model.ErrModelUnavailable):
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"code":    "model_unavailable",
					"message": "recommendation model is temporarily unavailable",
				},
			})
			return

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "internal_error",
					"message": "internal server error",
				},
			})
			return
		}
	}

	c.JSON(http.StatusOK, resp)
}
