package geoip

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	cacheTTL              = 1 * time.Hour
	negativeCacheTTL      = 30 * time.Second
	breakerFailureLimit   = 5
	breakerOpenDuration   = 30 * time.Second
	maxResolverRetryCount = 2
)

type Result struct {
	Country string `json:"country"`
	City    string `json:"city"`
}

type cacheEntry struct {
	result    Result
	expiresAt time.Time
}

type Resolver struct {
	client              *http.Client
	cache               map[string]cacheEntry
	mu                  sync.RWMutex
	consecutiveFailures int
	circuitOpenUntil    time.Time
}

type ipAPIResponse struct {
	Country string `json:"country_name"`
	City    string `json:"city"`
}

func New() *Resolver {
	return &Resolver{
		client: &http.Client{Timeout: 2 * time.Second},
		cache:  make(map[string]cacheEntry),
	}
}

func (r *Resolver) Resolve(ip string) Result {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return Result{Country: "Local", City: "Local"}
	}

	if cached, ok := r.getCached(ip); ok {
		return cached
	}

	if r.isCircuitOpen() {
		return Result{Country: "Unknown", City: "Unknown"}
	}

	var lastErr error
	for attempt := 0; attempt <= maxResolverRetryCount; attempt++ {
		result, err := r.resolveRemote(ip)
		if err == nil {
			r.recordSuccess()
			r.setCache(ip, result, cacheTTL)
			return result
		}

		lastErr = err
		if attempt < maxResolverRetryCount {
			time.Sleep(time.Duration((attempt+1)*100) * time.Millisecond)
		}
	}

	r.recordFailure()
	r.setCache(ip, Result{Country: "Unknown", City: "Unknown"}, negativeCacheTTL)
	if lastErr != nil {
		log.Printf("geoip resolve failed ip=%s err=%v", ip, lastErr)
	}

	return Result{Country: "Unknown", City: "Unknown"}
}

func (r *Resolver) resolveRemote(ip string) (Result, error) {
	resp, err := r.client.Get(fmt.Sprintf("https://ipapi.co/%s/json/", url.PathEscape(ip)))
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("upstream status %d", resp.StatusCode)
	}

	var upstream ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&upstream); err != nil {
		return Result{}, err
	}

	result := Result{Country: upstream.Country, City: upstream.City}
	if result.Country == "" {
		result.Country = "Unknown"
	}
	if result.City == "" {
		result.City = "Unknown"
	}

	return result, nil
}

func (r *Resolver) getCached(ip string) (Result, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.cache[ip]
	if !ok || time.Now().After(entry.expiresAt) {
		return Result{}, false
	}

	return entry.result, true
}

func (r *Resolver) setCache(ip string, result Result, ttl time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cache[ip] = cacheEntry{result: result, expiresAt: time.Now().Add(ttl)}
	r.evictExpiredLocked()
}

func (r *Resolver) isCircuitOpen() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return !r.circuitOpenUntil.IsZero() && time.Now().Before(r.circuitOpenUntil)
}

func (r *Resolver) recordSuccess() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.consecutiveFailures = 0
	r.circuitOpenUntil = time.Time{}
}

func (r *Resolver) recordFailure() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.consecutiveFailures++
	if r.consecutiveFailures >= breakerFailureLimit {
		r.circuitOpenUntil = time.Now().Add(breakerOpenDuration)
		r.consecutiveFailures = 0
	}
}

func (r *Resolver) evictExpiredLocked() {
	if len(r.cache) <= 1000 {
		return
	}

	now := time.Now()
	for k, v := range r.cache {
		if now.After(v.expiresAt) {
			delete(r.cache, k)
		}
	}

	if len(r.cache) > 1000 {
		for k := range r.cache {
			delete(r.cache, k)
			if len(r.cache) <= 1000 {
				break
			}
		}
	}
}
