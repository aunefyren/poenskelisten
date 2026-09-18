package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiterAllowsUpToMax(t *testing.T) {
	limiter := newRateLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !limiter.allow("1.2.3.4") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if limiter.allow("1.2.3.4") {
		t.Error("request over the limit should be blocked")
	}
}

func TestRateLimiterPerKey(t *testing.T) {
	limiter := newRateLimiter(1, time.Minute)

	if !limiter.allow("a") {
		t.Error("first request for key a should be allowed")
	}
	if limiter.allow("a") {
		t.Error("second request for key a should be blocked")
	}
	// A different key has its own budget.
	if !limiter.allow("b") {
		t.Error("first request for key b should be allowed")
	}
}

func TestRateLimiterWindowExpiry(t *testing.T) {
	limiter := newRateLimiter(1, 20*time.Millisecond)

	if !limiter.allow("k") {
		t.Fatal("first request should be allowed")
	}
	if limiter.allow("k") {
		t.Fatal("second request within window should be blocked")
	}
	time.Sleep(30 * time.Millisecond)
	if !limiter.allow("k") {
		t.Error("request after the window should be allowed again")
	}
}

func rateLimitTestContext() *gin.Context {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Request.RemoteAddr = "203.0.113.1:12345"
	return ctx
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
