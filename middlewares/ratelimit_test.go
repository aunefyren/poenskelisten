package middlewares

import (
	"testing"
	"time"
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
