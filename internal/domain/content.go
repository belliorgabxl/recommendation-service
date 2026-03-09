package domain

import "time"

type Content struct {
	ID              int64     `json:"id" db:"id"`
	Title           string    `json:"title" db:"title"`
	Genre           string    `json:"genre" db:"genre"`
	PopularityScore float64   `json:"popularity_score" db:"popularity_score"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}