package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aunefyren/poenskelisten/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus/hooks/test"
)

// proxyWarnings sends each request (header name → value, empty for none)
// through one router trusting trusted, and returns how many warnings were logged.
func proxyWarnings(t *testing.T, trusted []string, headers ...map[string]string) int {
	t.Helper()
	hook := test.NewLocal(logger.Log)
	t.Cleanup(hook.Reset)

	router := gin.New()
	if err := router.SetTrustedProxies(trusted); err != nil {
		t.Fatalf("SetTrustedProxies: %v", err)
	}
	router.Use(WarnUntrustedProxyHeaders())
	router.GET("/", func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	for _, h := range headers {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "172.18.0.2:5000"
		for name, value := range h {
			req.Header.Set(name, value)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
	}

	count := 0
	for _, entry := range hook.AllEntries() {
		if strings.Contains(entry.Message, "trustedproxies") {
			count++
		}
	}
	return count
}

func TestWarnUntrustedProxyHeaders(t *testing.T) {
	xff := map[string]string{"X-Forwarded-For": "198.51.100.7"}
	realIP := map[string]string{"X-Real-IP": "198.51.100.7"}
	none := map[string]string{}

	cases := []struct {
		name    string
		trusted []string
		headers []map[string]string
		want    int
	}{
		{"untrusted peer with X-Forwarded-For warns once", nil, []map[string]string{xff, xff, realIP}, 1},
		{"untrusted peer with X-Real-IP warns", nil, []map[string]string{realIP}, 1},
		{"no forwarding header stays quiet", nil, []map[string]string{none, none}, 0},
		{"trusted proxy stays quiet", []string{"172.18.0.0/16"}, []map[string]string{xff, realIP}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := proxyWarnings(t, c.trusted, c.headers...); got != c.want {
				t.Errorf("warnings = %d, want %d", got, c.want)
			}
		})
	}
}
