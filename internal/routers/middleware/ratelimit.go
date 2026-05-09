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
	rl := &RateLimiter{
		records: make(map[string]*ipRecord),
		max:     max,
		window:  window,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, rec := range rl.records {
			if now.After(rec.resetAt) {
				delete(rl.records, ip)
			}
		}
		rl.mu.Unlock()
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
