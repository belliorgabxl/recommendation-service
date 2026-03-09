package domain

import "time"

type WatchHistory struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	ContentID int64     `json:"content_id" db:"content_id"`
	WatchedAt time.Time `json:"watched_at" db:"watched_at"`
}

type WatchHistoryWithGenre struct {
	UserID    int64     `json:"user_id" db:"user_id"`
	ContentID int64     `json:"content_id" db:"content_id"`
	Genre     string    `json:"genre" db:"genre"`
	WatchedAt time.Time `json:"watched_at" db:"watched_at"`
}
