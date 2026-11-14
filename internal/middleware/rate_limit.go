package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimiter is a simple in-memory fixed-window limiter keyed by a caller
// identifier (usually the client IP). It is per-process: good enough to blunt
// brute-force and abuse on a single instance without requiring extra infra.
// For a multi-instance deployment a shared store (e.g. Redis) would be needed
// to make the limit global; this still bounds each instance independently.
type rateLimiter struct {
	mu       sync.Mutex
	window   time.Duration
	limit    int
	counters map[string]*rlEntry
}

type rlEntry struct {
	count int
	reset time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		window:   window,
		limit:    limit,
		counters: make(map[string]*rlEntry),
	}
	go rl.janitor()
	return rl
}

// janitor periodically drops expired counters so the map does not grow without bound.
func (rl *rateLimiter) janitor() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		rl.mu.Lock()
		for k, e := range rl.counters {
			if now.After(e.reset) {
				delete(rl.counters, k)
			}
		}
		rl.mu.Unlock()
	}
}

// allow reports whether the key may proceed within the current window, and if not,
// the number of seconds until the window resets.
func (rl *rateLimiter) allow(key string) (bool, int) {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	e, ok := rl.counters[key]
	if !ok || now.After(e.reset) {
		rl.counters[key] = &rlEntry{count: 1, reset: now.Add(rl.window)}
		return true, 0
	}
	if e.count >= rl.limit {
		return false, int(time.Until(e.reset).Seconds()) + 1
	}
	e.count++
	return true, 0
}

// RateLimitByIP returns a middleware that allows at most `limit` requests per
// `window` per client IP. When exceeded it aborts with 429 and a Retry-After
// header. Each call constructs an independent limiter, so give each sensitive
// endpoint its own instance.
func RateLimitByIP(limit int, window time.Duration) gin.HandlerFunc {
	rl := newRateLimiter(limit, window)
	return func(c *gin.Context) {
		ok, retry := rl.allow(c.ClientIP())
		if !ok {
			c.Header("Retry-After", strconv.Itoa(retry))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "Too many requests. Please try again later.",
			})
			return
		}
		c.Next()
	}
}
