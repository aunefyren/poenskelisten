package middlewares

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func allowed(limiter *rateLimiter, key string) bool {
	ok, _ := limiter.allow(key)
	return ok
}

// fakeClock lets a test move the limiter's time forward without sleeping.
func fakeClock(limiter *rateLimiter) *time.Time {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	limiter.lastSweep = now
	return &now
}

func TestRateLimiterAllowsUpToMax(t *testing.T) {
	limiter := newRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !allowed(limiter, "1.2.3.4") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if allowed(limiter, "1.2.3.4") {
		t.Error("request over the limit should be blocked")
	}
}

func TestRateLimiterPerKey(t *testing.T) {
	limiter := newRateLimiter(1, time.Minute)

	if !allowed(limiter, "a") {
		t.Error("first request for key a should be allowed")
	}
	if allowed(limiter, "a") {
		t.Error("second request for key a should be blocked")
	}
	// A different key has its own budget.
	if !allowed(limiter, "b") {
		t.Error("first request for key b should be allowed")
	}
}

func TestRateLimiterWindowExpiry(t *testing.T) {
	limiter := newRateLimiter(1, 20*time.Millisecond)

	if !allowed(limiter, "k") {
		t.Fatal("first request should be allowed")
	}
	if allowed(limiter, "k") {
		t.Fatal("second request within window should be blocked")
	}
	time.Sleep(30 * time.Millisecond)
	if !allowed(limiter, "k") {
		t.Error("request after the window should be allowed again")
	}
}

func rateLimitTestContext() *gin.Context {
	ctx, _ := rateLimitTestRecorder()
	return ctx
}

// rateLimitTestRecorder is rateLimitTestContext for tests that inspect the
// response, all from the same client IP.
func rateLimitTestRecorder() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Request.RemoteAddr = "203.0.113.1:12345"
	return ctx, w
}

func TestRateLimitMiddlewareAllowsWithinLimit(t *testing.T) {
	handler := RateLimit(2, time.Minute)

	ctx := rateLimitTestContext()
	handler(ctx)
	if ctx.IsAborted() {
		t.Error("first request should not be aborted")
	}
}

func TestRateLimitMiddlewareBlocksOverLimit(t *testing.T) {
	handler := RateLimit(1, time.Minute)

	first := rateLimitTestContext()
	handler(first)
	if first.IsAborted() {
		t.Fatal("first request should not be aborted")
	}

	// Same middleware instance (same limiter state), same client IP.
	second := rateLimitTestContext()
	handler(second)
	if !second.IsAborted() {
		t.Error("second request over the limit should be aborted")
	}
	if second.Writer.Status() != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", second.Writer.Status())
	}
}

func TestRateLimiterRetryAfterCountsFromOldestHit(t *testing.T) {
	limiter := newRateLimiter(2, time.Minute)
	now := fakeClock(limiter)

	limiter.allow("k")
	*now = now.Add(20 * time.Second)
	limiter.allow("k")
	*now = now.Add(10 * time.Second)

	ok, retryAfter := limiter.allow("k")
	if ok {
		t.Fatal("third request should be blocked")
	}
	// The first hit leaves the window 60s after it was made, 30s from now.
	if retryAfter != 30*time.Second {
		t.Errorf("retryAfter = %v, want 30s", retryAfter)
	}
}

func TestRateLimiterSweepDropsIdleKeys(t *testing.T) {
	limiter := newRateLimiter(5, time.Minute)
	now := fakeClock(limiter)

	limiter.allow("idle")
	*now = now.Add(30 * time.Second)
	limiter.allow("active")

	// One window after the last sweep, "idle" has no hit left inside the window
	// but "active" does.
	*now = now.Add(31 * time.Second)
	limiter.allow("trigger")

	if _, ok := limiter.hits["idle"]; ok {
		t.Error("idle key should have been swept")
	}
	if _, ok := limiter.hits["active"]; !ok {
		t.Error("active key should be kept")
	}
	if _, ok := limiter.hits["trigger"]; !ok {
		t.Error("key hit during the sweep should be kept")
	}
}

func TestRateLimiterSweepWaitsAWindow(t *testing.T) {
	limiter := newRateLimiter(5, time.Minute)
	now := fakeClock(limiter)

	limiter.allow("old")
	// Expired, but less than a window since the last sweep: left alone for now.
	*now = now.Add(time.Minute)
	limiter.lastSweep = now.Add(-time.Second)
	limiter.allow("other")

	if _, ok := limiter.hits["old"]; !ok {
		t.Error("sweep should not run more than once per window")
	}
}

func TestRateLimitMiddlewareSetsRetryAfter(t *testing.T) {
	handler := RateLimit(1, time.Minute)

	handler(rateLimitTestContext())
	ctx, w := rateLimitTestRecorder()
	handler(ctx)

	got := w.Header().Get("Retry-After")
	if got != "60" {
		t.Errorf("Retry-After = %q, want \"60\"", got)
	}
}

func TestRateLimitBodies(t *testing.T) {
	cases := []struct {
		name    string
		handler gin.HandlerFunc
		want    map[string]string
	}{
		{"oauth", RateLimit(1, time.Minute), map[string]string{
			"error":             "rate_limited",
			"error_description": "Too many requests. Please try again later.",
		}},
		{"api", RateLimitAPI(1, time.Minute), map[string]string{
			"error": "Too many attempts. Please wait a while and try again.",
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.handler(rateLimitTestContext())
			ctx, w := rateLimitTestRecorder()
			c.handler(ctx)

			if w.Code != http.StatusTooManyRequests {
				t.Fatalf("status = %d, want 429", w.Code)
			}
			var body map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not JSON: %v", err)
			}
			if len(body) != len(c.want) {
				t.Errorf("body = %v, want %v", body, c.want)
			}
			for key, value := range c.want {
				if body[key] != value {
					t.Errorf("body[%q] = %q, want %q", key, body[key], value)
				}
			}
		})
	}
}
