package middlewares

import (
	"github.com/gin-gonic/gin"
)

// ContentSecurityPolicy is sent on every response. script-src is 'self' only:
// the frontend carries no inline <script> blocks, on* handler attributes or
// javascript: URLs (markup names registered actions instead, see
// web/js/actions.js, and TestFrontendHasNoInlineScript keeps it that way), so an
// injected handler or script tag is refused by the browser rather than run.
// connect-src and img-src keep fetch/XHR/beacons and image loads on this origin
// (or inline data); object-src/base-uri shut off plugin and <base> tricks;
// frame-ancestors stops clickjacking.
//
// style-src still allows 'unsafe-inline': the markup leans heavily on style
// attributes, and injected CSS can't run script.
//
// form-action is deliberately absent: the OAuth consent form POSTs to
// /oauth/consent, which redirects to a third-party client's redirect_uri, and
// browsers apply form-action to that redirect.
const ContentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
	"font-src 'self' https://fonts.gstatic.com; " +
	"img-src 'self' data: blob:; " +
	"connect-src 'self'; " +
	"worker-src 'self'; " +
	"manifest-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'self'; " +
	"frame-ancestors 'none'"

// SecurityHeaders sets the CSP plus the companion headers that harden the
// browser side of every page. Referrer-Policy is same-origin rather than the
// browser default because public wishlist URLs carry their access hash, and
// clicking a wish link from such a page must not leak it to the shop.
func SecurityHeaders() gin.HandlerFunc {
	return func(context *gin.Context) {
		header := context.Writer.Header()
		header.Set("Content-Security-Policy", ContentSecurityPolicy)
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "same-origin")
		context.Next()
	}
}
