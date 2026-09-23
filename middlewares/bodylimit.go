package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultMaxBodyBytes applies to every route without an override.
	DefaultMaxBodyBytes int64 = 1 << 20
	// ImageUploadMaxBodyBytes is for the routes that carry a base64 image in
	// their JSON body. Images are capped at 10 MB decoded (see
	// controllers.maxUploadBytes), which is ~13.3 MB as base64, plus the
	// other JSON fields.
	ImageUploadMaxBodyBytes int64 = 15 << 20
)

// MaxBodySize caps request bodies at defaultLimit, or at overrides[route] for
// routes (keyed by their registered path, e.g. "/api/auth/wishes/:wish_id")
// that need more. Without it, handlers' ShouldBindJSON buffers whatever the
// client sends before any size check runs.
//
// It has to be a single global middleware with an override table, rather than
// a global default plus per-route middleware: the global one runs first, and
// would reject a large upload before the route's middleware could raise the
// limit.
func MaxBodySize(defaultLimit int64, overrides map[string]int64) gin.HandlerFunc {
	return func(context *gin.Context) {
		limit := defaultLimit
		if override, ok := overrides[context.FullPath()]; ok {
			limit = override
		}

		// Reject a declared oversize body up front, with a clear status. The
		// MaxBytesReader below catches chunked bodies and false Content-Lengths,
		// which surface in the handler as a body parse error instead.
		if context.Request.ContentLength > limit {
			context.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Request body is too large."})
			context.Abort()
			return
		}

		if context.Request.Body != nil {
			context.Request.Body = http.MaxBytesReader(context.Writer, context.Request.Body, limit)
		}
		context.Next()
	}
}
