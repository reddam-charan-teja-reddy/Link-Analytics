package redis

import (
	"context"
	"testing"
)

func TestLinkCacheNoopWhenRedisUnavailable(t *testing.T) {
	t.Parallel()

	cache := NewLinkCache(nil, 0)
	ctx := context.Background()

	item, err := cache.Get(ctx, "abc123")
	if err != nil {
		t.Fatalf("expected nil error for get without redis, got %v", err)
	}
	if item != nil {
		t.Fatalf("expected nil cache item without redis")
	}

	if err := cache.Set(ctx, "abc123", &CachedLink{OriginalURL: "https://example.com", LinkID: "id"}); err != nil {
		t.Fatalf("expected nil error for set without redis, got %v", err)
	}

	if err := cache.Invalidate(ctx, "abc123"); err != nil {
		t.Fatalf("expected nil error for invalidate without redis, got %v", err)
	}
}
