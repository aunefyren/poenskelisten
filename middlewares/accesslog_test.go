package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// The access log mustn't hold a code that resets a password, verifies an
// e-mail address or opens a public wishlist, even when no route matches.
// Other paths and IDs are kept, for tracing a request.
func TestAccessLogLeavesOutCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logged strings.Builder
	router := gin.New()
	router.Use(AccessLog(&logged, router.Routes))
	ok := func(ctx *gin.Context) { ctx.Status(http.StatusOK) }
	router.GET("/reset/:resetCode", ok)
	router.POST("/verify/:code", ok)
	router.GET("/lists/:wishlist_id", ok)
	router.GET("/public/:wishlist_hash", ok)

	cases := []struct{ name, method, target, want string }{
		{"status", http.MethodGet, "/reset/SECRET", `| 200 |`},
		{"secret parameter", http.MethodGet, "/reset/SECRET", `"/reset/:resetCode"`},
		{"query string", http.MethodGet, "/lists/42?code=SECRET", `"/lists/42"`},
		{"ordinary ID", http.MethodGet, "/lists/42", `"/lists/42"`},
		{"HEAD on a secret route", http.MethodHead, "/public/SECRET", `"/public/:wishlist_hash"`},
		{"wrong method", http.MethodGet, "/verify/SECRET", `"/verify/:code"`},
		{"slash inside a code", http.MethodGet, "/public/SEC%2FRET", `"/public/:wishlist_hash"`},
		{"unknown path", http.MethodGet, "/nowhere", `"/nowhere"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			logged.Reset()
			router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(c.method, c.target, nil))
			line := logged.String()
			if strings.Contains(line, "SECRET") || !strings.Contains(line, c.method) || !strings.Contains(line, c.want) {
				t.Errorf("logged %q, want %s %s and no code", line, c.method, c.want)
			}
			if strings.Count(line, "[GIN] ") != 1 {
				t.Errorf("logged %q, want one line", line)
			}
		})
	}
}

func TestSecretRoutesOf(t *testing.T) {
	routes := gin.RoutesInfo{
		{Method: http.MethodGet, Path: "/api/open/users/reset/:resetCode"},
		{Method: http.MethodGet, Path: "/files/*code"},
		{Method: http.MethodGet, Path: "/groups/:group_id"},
		{Method: http.MethodGet, Path: "/login"},
	}
	got := secretRoutesOf(routes)
	want := []secretRoute{
		{prefix: "/api/open/users/reset/", route: "/api/open/users/reset/:resetCode"},
		{prefix: "/files/", route: "/files/*code"},
	}
	if len(got) != len(want) {
		t.Fatalf("secretRoutesOf = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("secretRoutesOf[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}
