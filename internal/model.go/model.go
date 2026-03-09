package model

import "github.com/gabxlbellior/recommendation-service/internal/domain"

type Input struct {
	User         *domain.User
	WatchHistory []domain.WatchHistoryWithGenre
	Candidates   []domain.Content
	Limit        int
}