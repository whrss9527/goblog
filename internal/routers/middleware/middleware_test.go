package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	limiter := NewRateLimiter(3, 1*time.Minute)
	router := gin.New()
	router.POST("/test", limiter.Limit(), func(c *gin.Context) {
		c.Status(200)
	})

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	limiter := NewRateLimiter(2, 1*time.Minute)
	router := gin.New()
	router.POST("/test", limiter.Limit(), func(c *gin.Context) {
		c.Status(200)
	})

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestCSRFProtect_GETSetsToken(t *testing.T) {
	token := generateToken()
	assert.Len(t, token, 64)

	token2 := generateToken()
	assert.NotEqual(t, token, token2)
}

func TestSecurityHeaders(t *testing.T) {
	router := gin.New()
	router.Use(SecurityHeaders)
	router.GET("/*path", func(c *gin.Context) { c.Status(200) })

	for path, frame := range map[string]string{"/": "SAMEORIGIN", "/posts/x": "SAMEORIGIN", "/admin": "DENY", "/admin/posts/add": "DENY", "/administrator": "SAMEORIGIN"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		assert.Equal(t, frame, w.Header().Get("X-Frame-Options"), path)
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"), path)
		assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"), path)
	}
}
