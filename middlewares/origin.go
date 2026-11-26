package middlewares

import (
	"aunefyren/poenskelisten/config"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

// SameOrigin refuses a browser request sent by a page that isn't one of
// Pønskelisten's own origins. It guards the endpoints that turn a password,
// a code or the sign-in cookie into a session or a consent: any site can
// post a form to them, and the cookie's SameSite=Lax protects neither a
// sign-in, which needs no cookie, nor a request from another site of the
// same domain. Browsers always send Origin, or at least Sec-Fetch-Site, with
// such a request; a client sending neither isn't a browser.
func SameOrigin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if origin := ctx.GetHeader("Origin"); origin != "" {
			if !isOwnOrigin(origin) {
				refuseCrossOrigin(ctx)
				return
			}
		} else if site := ctx.GetHeader("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
			refuseCrossOrigin(ctx)
			return
		}
		ctx.Next()
	}
}

// isOwnOrigin compares origins the way a browser writes them: the external
// URL comes from the configuration as typed, possibly with a capital letter,
// a default port or a trailing path.
func isOwnOrigin(origin string) bool {
	normalized, err := config.NormalizeOrigin(origin)
	if err != nil {
		return false
	}
	for _, allowed := range config.AllowedOrigins() {
		parsed, err := url.Parse(allowed)
		if err != nil {
			continue
		}
		if own, err := config.NormalizeOrigin(parsed.Scheme + "://" + parsed.Host); err == nil && own == normalized {
			return true
		}
	}
	return false
}

func refuseCrossOrigin(ctx *gin.Context) {
	ctx.JSON(http.StatusForbidden, gin.H{"error": "Sign in from the address Pønskelisten is set up on."})
	ctx.Abort()
}
