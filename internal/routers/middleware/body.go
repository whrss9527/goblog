package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// LimitBody caps the request body at n bytes. It has to run before anything
// reads the form (the CSRF check does), or a huge upload would be parsed
// into memory and temporary files first.
func LimitBody(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		c.Next()
	}
}
