package middlewares

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimiter is a simple in-memory, per-key sliding-window limiter. State lives
// in the process, which is fine for a single-binary app.
type rateLimiter struct {
	mu        sync.Mutex
	hits      map[string][]time.Time
	max       int
	window    time.Duration
	lastSweep time.Time
	now       func() time.Time
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		hits:      make(map[string][]time.Time),
		max:       max,
		window:    window,
		lastSweep: time.Now(),
		now:       time.Now,
	}
}

// allow records a hit for key and reports whether it is within the limit. When
// it is not, retryAfter is how long until the oldest hit leaves the window.
func (r *rateLimiter) allow(key string) (allowed bool, retryAfter time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	cutoff := now.Add(-r.window)
	r.sweep(now, cutoff)

	kept := make([]time.Time, 0, len(r.hits[key])+1)
	for _, t := range r.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	if len(kept) >= r.max {
		r.hits[key] = kept
		return false, kept[0].Add(r.window).Sub(now)
	}

	kept = append(kept, now)
	r.hits[key] = kept
	return true, 0
}

// sweep drops keys with no hit inside the window. Without it every client IP
// ever seen would stay in the map for the life of the process. Running at most
// once per window keeps the full scan amortized. Callers hold r.mu.
func (r *rateLimiter) sweep(now, cutoff time.Time) {
	if now.Sub(r.lastSweep) < r.window {
		return
	}
	r.lastSweep = now
	for key, times := range r.hits {
		if len(times) == 0 || !times[len(times)-1].After(cutoff) {
			delete(r.hits, key)
		}
	}
}

// RateLimit limits requests per client IP to max within a sliding window,
// returning 429 with an OAuth-style error body. Use it on the /oauth/*
// endpoints, whose clients expect RFC 6749-shaped errors.
func RateLimit(max int, window time.Duration) gin.HandlerFunc {
	return rateLimit(max, window, gin.H{
		"error":             "rate_limited",
		"error_description": "Too many requests. Please try again later.",
	})
}

// RateLimitAPI is RateLimit for /api routes: the 429 body carries a readable
// message under "error", since the frontend shows that field to the user as is.
func RateLimitAPI(max int, window time.Duration) gin.HandlerFunc {
	return rateLimit(max, window, gin.H{
		"error": "Too many attempts. Please wait a while and try again.",
	})
}

// Every route sharing one returned handler shares its budget, so register a
// single instance on related routes to give them a common limit.
func rateLimit(max int, window time.Duration, body gin.H) gin.HandlerFunc {
	limiter := newRateLimiter(max, window)
	return func(ctx *gin.Context) {
		allowed, retryAfter := limiter.allow(ctx.ClientIP())
		if !allowed {
			seconds := int(math.Ceil(retryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			ctx.Header("Retry-After", strconv.Itoa(seconds))
			ctx.JSON(http.StatusTooManyRequests, body)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
