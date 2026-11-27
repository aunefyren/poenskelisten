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
		if !fromOwnPage(ctx) {
			ctx.JSON(http.StatusForbidden, gin.H{"error": crossOriginMessage})
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

// SameOriginForm is SameOrigin for an endpoint a plain HTML form posts to,
// such as the consent page: the browser shows the refusal as the page, so it
// is plain text rather than JSON.
func SameOriginForm() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !fromOwnPage(ctx) {
			ctx.String(http.StatusForbidden, crossOriginMessage)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

// crossOriginMessage covers sign-in, MFA, e-mail verification and consent
// alike, so it names no single one of them.
const crossOriginMessage = "Open Pønskelisten on the address it is set up on and try again."

// fromOwnPage reports whether a request came from one of Pønskelisten's own
// pages, or from a client that isn't a browser.
func fromOwnPage(ctx *gin.Context) bool {
	if origin := ctx.GetHeader("Origin"); origin != "" {
		return isOwnOrigin(origin)
	}
	site := ctx.GetHeader("Sec-Fetch-Site")
	return site == "" || site == "same-origin" || site == "none"
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
