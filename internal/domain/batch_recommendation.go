package domain

import "time"

type BatchRecommendationResult struct {
	UserID          int64                `json:"user_id"`
	Recommendations []RecommendationItem `json:"recommendations,omitempty"`
	Status          string               `json:"status"`
	Error           string               `json:"error,omitempty"`
	Message         string               `json:"message,omitempty"`
}

type BatchRecommendationSummary struct {
	SuccessCount     int   `json:"success_count"`
	FailedCount      int   `json:"failed_count"`
	ProcessingTimeMs int64 `json:"processing_time_ms"`
}

type BatchRecommendationMetadata struct {
	GeneratedAt time.Time `json:"generated_at"`
}

type BatchRecommendationResponse struct {
	Page       int                         `json:"page"`
	Limit      int                         `json:"limit"`
	TotalUsers int64                       `json:"total_users"`
	Results    []BatchRecommendationResult `json:"results"`
	Summary    BatchRecommendationSummary  `json:"summary"`
	Metadata   BatchRecommendationMetadata `json:"metadata"`
}