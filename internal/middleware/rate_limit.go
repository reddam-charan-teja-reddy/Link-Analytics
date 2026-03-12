package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

var sharedRateLimitRedis *redis.Client

func SetRateLimitRedisClient(client *redis.Client) {
	sharedRateLimitRedis = client
}

type rateEntry struct {
	count   int
	resetAt time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateEntry
	limit   int
	window  time.Duration
	scope   string
	lastGC  time.Time
}

func newRateLimiter(limit int, window time.Duration, scope string) *rateLimiter {
	return &rateLimiter{
		entries: make(map[string]rateEntry),
		limit:   limit,
		window:  window,
		scope:   scope,
	}
}

func (r *rateLimiter) allow(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if now.Sub(r.lastGC) > 5*time.Minute {
		r.gc(now)
		r.lastGC = now
	}

	key := r.scope + ":" + ip

	entry, ok := r.entries[key]
	if !ok || now.After(entry.resetAt) {
		r.entries[key] = rateEntry{count: 1, resetAt: now.Add(r.window)}
		return true
	}

	if entry.count >= r.limit {
		return false
	}

	entry.count++
	r.entries[key] = entry
	return true
}

func (r *rateLimiter) gc(now time.Time) {
	for key, entry := range r.entries {
		if now.After(entry.resetAt) {
			delete(r.entries, key)
		}
	}
}

func RateLimit(limit int, window time.Duration, scope string) gin.HandlerFunc {
	limiter := newRateLimiter(limit, window, scope)

	return func(c *gin.Context) {
		if sharedRateLimitRedis != nil {
			key := "rl:" + scope + ":" + c.ClientIP()
			count, err := sharedRateLimitRedis.Incr(c.Request.Context(), key).Result()
			if err == nil {
				if count == 1 {
					_ = sharedRateLimitRedis.Expire(c.Request.Context(), key, window).Err()
				}
				if int(count) <= limit {
					c.Next()
					return
				}

				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
				return
			}

			// Fallback to local limiter if Redis is temporarily unavailable.
		}

		if limiter.allow(c.ClientIP()) {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
	}
}
