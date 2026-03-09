package handler

import (
	"net/http"
	"time"

	"github.com/gabxlbellior/recommendation-service/internal/config"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	cfg config.Config
}

func NewHealthHandler(cfg config.Config) *HealthHandler {
	return &HealthHandler{cfg: cfg}
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":      "ok",
		"service":     h.cfg.AppName,
		"environment": h.cfg.AppEnv,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	})
}