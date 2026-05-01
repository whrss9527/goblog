package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type ipRecord struct {
	count    int
	resetAt  time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	records map[string]*ipRecord
	max     int
	window  time.Duration
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		records: make(map[string]*ipRecord),
		max:     max,
		window:  window,
	}
}

func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()

		rl.mu.Lock()
		rec, ok := rl.records[ip]
		now := time.Now()
		if !ok || now.After(rec.resetAt) {
			rl.records[ip] = &ipRecord{count: 1, resetAt: now.Add(rl.window)}
			rl.mu.Unlock()
			ctx.Next()
			return
		}
		rec.count++
		if rec.count > rl.max {
			rl.mu.Unlock()
			ctx.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		rl.mu.Unlock()
		ctx.Next()
	}
}
