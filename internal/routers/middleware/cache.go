package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// cacheImmutable is for URLs that carry a content fingerprint (?v=…, added
	// by the "asset" template func): a changed file gets a new URL.
	cacheImmutable = "public, max-age=31536000, immutable"
	// cacheShort is for plain static URLs: cached for a while, then revalidated
	// (http.FileServer answers If-Modified-Since with 304).
	cacheShort = "public, max-age=86400, stale-while-revalidate=604800"
)

// StaticCache sets Cache-Control on GET/HEAD requests below the given path
// prefixes. Without it browsers re-request every stylesheet, script and image
// on each page view.
func StaticCache(prefixes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" {
			path := c.Request.URL.Path
			for _, prefix := range prefixes {
				if !strings.HasPrefix(path, prefix) {
					continue
				}
				if c.Query("v") != "" {
					c.Header("Cache-Control", cacheImmutable)
				} else {
					c.Header("Cache-Control", cacheShort)
				}
				break
			}
		}
		c.Next()
	}
}

// ImmutableCache marks responses below the given prefixes as never changing:
// uploaded images get a new name every time, so a copy can be kept for good.
func ImmutableCache(prefixes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" {
			for _, prefix := range prefixes {
				if strings.HasPrefix(c.Request.URL.Path, prefix) {
					c.Header("Cache-Control", cacheImmutable)
					break
				}
			}
		}
		c.Next()
	}
}
