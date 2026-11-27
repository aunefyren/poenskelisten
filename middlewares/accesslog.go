package middlewares

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// secretRouteParams are the route parameters whose value grants access by
// itself: a password reset code, an e-mail verification code and the link
// to a public wishlist.
var secretRouteParams = map[string]bool{"resetCode": true, "code": true, "wishlist_hash": true}

// secretRoute is a route with a secret parameter, with the part of its path
// before that parameter.
type secretRoute struct{ prefix, route string }

// secretRoutesOf lists the routes that have a secret parameter.
func secretRoutesOf(routes gin.RoutesInfo) []secretRoute {
	var found []secretRoute
	for _, r := range routes {
		segments := strings.Split(r.Path, "/")
		for i, segment := range segments {
			if (strings.HasPrefix(segment, ":") || strings.HasPrefix(segment, "*")) && secretRouteParams[segment[1:]] {
				found = append(found, secretRoute{prefix: strings.Join(segments[:i], "/") + "/", route: r.Path})
				break
			}
		}
	}
	return found
}

// AccessLog writes a line per request to out, in the format of gin's own
// logger, but leaves out what lets someone reading the log in: the query
// string, which carries a reset link's code or an OAuth code, and the value
// of the secret route parameters, logged as the route instead. routes gives
// the router's routes; it is read once, at the first request, when every
// route has been registered.
func AccessLog(out io.Writer, routes func() gin.RoutesInfo) gin.HandlerFunc {
	var once sync.Once
	var secret []secretRoute
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()
		once.Do(func() { secret = secretRoutesOf(routes()) })

		fmt.Fprintf(out, "[GIN] %s | %3d | %13v | %15s | %-7s %#v\n%s",
			start.Format("2006/01/02 - 15:04:05"),
			ctx.Writer.Status(),
			time.Since(start),
			ctx.ClientIP(),
			ctx.Request.Method,
			loggedPath(ctx, secret),
			ctx.Errors.ByType(gin.ErrorTypePrivate).String(),
		)
	}
}

// loggedPath is the request's path without its query string, or the route
// when the path holds the value of a secret parameter.
func loggedPath(ctx *gin.Context, secret []secretRoute) string {
	path := ctx.Request.URL.Path
	if route := ctx.FullPath(); route != "" {
		for _, param := range ctx.Params {
			if secretRouteParams[param.Key] {
				return route
			}
		}
		return path
	}
	// No route matched (HEAD, a wrong method, a slash inside a code), so gin
	// set no parameters: check whether the path falls under a secret route.
	for _, r := range secret {
		if strings.HasPrefix(path, r.prefix) {
			return r.route
		}
	}
	return path
}
