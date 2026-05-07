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
