package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gabxlbellior/recommendation-service/internal/config"
	"github.com/gabxlbellior/recommendation-service/internal/domain"
	"github.com/redis/go-redis/v9"
)

type RecommendationCache interface {
	Get(ctx context.Context, key string) (*domain.RecommendationResponse, bool, error)
	Set(ctx context.Context, key string, value *domain.RecommendationResponse, ttl time.Duration) error
	Close() error
}

type RedisRecommendationCache struct {
	client *redis.Client
}

func NewRedisRecommendationCache(ctx context.Context, cfg config.Config) (*RedisRecommendationCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr(),
		Password:     cfg.RedisPassword,
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 2,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &RedisRecommendationCache{
		client: client,
	}, nil
}

func (c *RedisRecommendationCache) Get(ctx context.Context, key string) (*domain.RecommendationResponse, bool, error) {
	raw, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("redis get key %q: %w", key, err)
	}

	var resp domain.RecommendationResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, false, fmt.Errorf("unmarshal cached recommendation response: %w", err)
	}

	return &resp, true, nil
}

func (c *RedisRecommendationCache) Set(ctx context.Context, key string, value *domain.RecommendationResponse, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal recommendation response: %w", err)
	}

	if err := c.client.Set(ctx, key, payload, ttl).Err(); err != nil {
		return fmt.Errorf("redis set key %q: %w", key, err)
	}

	return nil
}

func (c *RedisRecommendationCache) Close() error {
	return c.client.Close()
}

func BuildRecommendationKey(userID int64, limit int, latestWatchedAt *time.Time) string {
	version := "0"
	if latestWatchedAt != nil {
		version = strconv.FormatInt(latestWatchedAt.UTC().Unix(), 10)
	}

	return "rec:user:" +
		strconv.FormatInt(userID, 10) +
		":limit:" +
		strconv.Itoa(limit) +
		":v:" +
		version
}