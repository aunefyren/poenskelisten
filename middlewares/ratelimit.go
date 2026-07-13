package middlewares

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimiter is a simple in-memory, per-key sliding-window limiter. State lives
// in the process, which is fine for a single-binary app.
type rateLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	max    int
	window time.Duration
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		hits:   make(map[string][]time.Time),
		max:    max,
		window: window,
	}
}

// allow records a hit for key and reports whether it is within the limit.
func (r *rateLimiter) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	kept := make([]time.Time, 0, len(r.hits[key])+1)
	for _, t := range r.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	if len(kept) >= r.max {
		r.hits[key] = kept
		return false
	}

	kept = append(kept, now)
	r.hits[key] = kept
	return true
}

// RateLimit limits requests per client IP to max within a sliding window,
// returning 429 when exceeded.
func RateLimit(max int, window time.Duration) gin.HandlerFunc {
	limiter := newRateLimiter(max, window)
	return func(ctx *gin.Context) {
		if !limiter.allow(ctx.ClientIP()) {
			ctx.JSON(http.StatusTooManyRequests, gin.H{
				"error":             "rate_limited",
				"error_description": "Too many requests. Please try again later.",
			})
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
