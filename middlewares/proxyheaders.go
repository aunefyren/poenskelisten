package middlewares

import (
	"sync"

	"aunefyren/poenskelisten/logger"

	"github.com/gin-gonic/gin"
)

// WarnUntrustedProxyHeaders logs once when a request carries a forwarding
// header from a peer that isn't a trusted proxy. Such headers are ignored, so
// behind an unconfigured reverse proxy every visitor shares the proxy's IP:
// rate limits then apply to everyone at once, and sessions record the proxy's
// address. The warning is how an operator finds out to set trustedproxies.
func WarnUntrustedProxyHeaders() gin.HandlerFunc {
	var once sync.Once
	return func(ctx *gin.Context) {
		forwarded := ctx.GetHeader("X-Forwarded-For") != "" || ctx.GetHeader("X-Real-IP") != ""
		// ClientIP only differs from the peer address when the header was used.
		if forwarded && ctx.ClientIP() == ctx.RemoteIP() {
			once.Do(func() {
				logger.Log.Warn("ignoring X-Forwarded-For/X-Real-IP from " + ctx.RemoteIP() + ", which isn't a trusted proxy. If Pønskelisten runs behind a reverse proxy, set trustedproxies to the proxy's address so rate limits and sessions see each visitor's own IP.")
			})
		}
		ctx.Next()
	}
}
