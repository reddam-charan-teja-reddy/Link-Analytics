package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CachedLink struct {
	OriginalURL  string  `json:"original_url"`
	LinkID       string  `json:"link_id"`
	SourceLinkID *string `json:"source_link_id,omitempty"`
}

type LinkCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewLinkCache(client *redis.Client, ttl time.Duration) *LinkCache {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &LinkCache{client: client, ttl: ttl}
}

func (c *LinkCache) Get(ctx context.Context, hash string) (*CachedLink, error) {
	if c == nil || c.client == nil {
		return nil, nil
	}

	data, err := c.client.Get(ctx, cacheKey(hash)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}

	var cached CachedLink
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("unmarshal cached link: %w", err)
	}
	return &cached, nil
}

func (c *LinkCache) Set(ctx context.Context, hash string, link *CachedLink) error {
	if c == nil || c.client == nil {
		return nil
	}

	data, err := json.Marshal(link)
	if err != nil {
		return fmt.Errorf("marshal cached link: %w", err)
	}
	return c.client.Set(ctx, cacheKey(hash), data, c.ttl).Err()
}

func (c *LinkCache) Invalidate(ctx context.Context, hash string) error {
	if c == nil || c.client == nil {
		return nil
	}

	return c.client.Del(ctx, cacheKey(hash)).Err()
}

func cacheKey(hash string) string {
	return "link:" + hash
}
