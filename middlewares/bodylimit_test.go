package middlewares

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// bodyLimitRouter has a default limit of 10 bytes, raised to 20 on /large.
// Its handler echoes how many body bytes it could read, or 400 if reading hit
// the limit.
func bodyLimitRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MaxBodySize(10, map[string]int64{"/large/:id": 20}))
	handler := func(context *gin.Context) {
		if context.Request.Body == nil {
			context.String(http.StatusOK, "0")
			return
		}
		data, err := io.ReadAll(context.Request.Body)
		if err != nil {
			context.String(http.StatusBadRequest, err.Error())
			return
		}
		context.String(http.StatusOK, "%d", len(data))
	}
	router.POST("/", handler)
	router.POST("/large/:id", handler)
	return router
}

func TestMaxBodySize(t *testing.T) {
	cases := []struct {
		name           string
		path           string
		bodyLength     int
		hideLength     bool // simulate a chunked body with no Content-Length
		wantStatus     int
		wantBodyPrefix string
	}{
		{"within default", "/", 10, false, 200, "10"},
		{"declared over default", "/", 11, false, 413, `{"error"`},
		{"undeclared over default", "/", 11, true, 400, "http: request body too large"},
		{"override raises limit", "/large/1", 15, false, 200, "15"},
		{"override raises limit, undeclared", "/large/1", 15, true, 200, "15"},
		{"declared over override", "/large/1", 21, false, 413, `{"error"`},
		{"undeclared over override", "/large/1", 21, true, 400, "http: request body too large"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			router := bodyLimitRouter()
			request := httptest.NewRequest("POST", c.path, bytes.NewReader(bytes.Repeat([]byte("x"), c.bodyLength)))
			if c.hideLength {
				request.ContentLength = -1
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if recorder.Code != c.wantStatus || !strings.HasPrefix(recorder.Body.String(), c.wantBodyPrefix) {
				t.Errorf("got %d %q, want %d %q...", recorder.Code, recorder.Body.String(), c.wantStatus, c.wantBodyPrefix)
			}
		})
	}
}

func TestMaxBodySizeNoBody(t *testing.T) {
	router := bodyLimitRouter()
	request := httptest.NewRequest("POST", "/", nil)
	request.Body = nil
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != 200 {
		t.Errorf("status = %d, want 200 for a request with no body", recorder.Code)
	}
}
