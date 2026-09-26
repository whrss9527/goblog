package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders sets the headers every response should carry: no MIME
// sniffing, no full URLs leaking to other sites, and no framing by other sites
// (the admin cannot be framed at all, so its buttons cannot be clickjacked).
func SecurityHeaders(c *gin.Context) {
	header := c.Writer.Header()
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	if c.Request.URL.Path == "/admin" || strings.HasPrefix(c.Request.URL.Path, "/admin/") {
		header.Set("X-Frame-Options", "DENY")
	} else {
		header.Set("X-Frame-Options", "SAMEORIGIN")
	}
	c.Next()
}
