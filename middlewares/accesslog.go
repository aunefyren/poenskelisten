package middlewares

import (
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
)

// secretRouteParams are the route parameters whose value grants access by
// itself: a password reset code, an e-mail verification code and the link
// to a public wishlist.
var secretRouteParams = map[string]bool{"resetCode": true, "code": true, "wishlist_hash": true}

// AccessLog writes a line per request to out, in the format of gin's own
// logger, but leaves out what lets someone reading the log in: the query
// string, which carries a reset link's code or an OAuth code, and the value
// of the secret route parameters, logged as the route instead.
func AccessLog(out io.Writer) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()

		path := ctx.Request.URL.Path
		for _, param := range ctx.Params {
			if secretRouteParams[param.Key] {
				path = ctx.FullPath()
				break
			}
		}
		fmt.Fprintf(out, "[GIN] %s | %3d | %13v | %15s | %-7s %#v\n%s",
			start.Format("2006/01/02 - 15:04:05"),
			ctx.Writer.Status(),
			time.Since(start),
			ctx.ClientIP(),
			ctx.Request.Method,
			path,
			ctx.Errors.ByType(gin.ErrorTypePrivate).String(),
		)
	}
}
