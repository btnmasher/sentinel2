package middleware

import (
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
}

func NewRateLimiter(r rate.Limit, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: map[string]*rate.Limiter{},
		rate:     r,
		burst:    burst,
	}
}

func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	limiter, ok := r.limiters[key]
	if !ok {
		limiter = rate.NewLimiter(r.rate, r.burst)
		r.limiters[key] = limiter
	}
	r.mu.Unlock()
	return limiter.Allow()
}

func RateLimit(limiter *RateLimiter, keyFunc func(*core.RequestEvent) string) func(*core.RequestEvent) error {
	return func(c *core.RequestEvent) error {
		key := keyFunc(c)
		if key == "" {
			key = clientIP(c)
		}
		if !limiter.Allow(key) {
			return router.NewTooManyRequestsError("Ratelimit hit, stop spamming us nerds", nil)
		}
		return c.Next()
	}
}

func clientIP(c *core.RequestEvent) string {
	return c.RealIP()
}

func LimitPerHour(count int) rate.Limit {
	if count <= 0 {
		return rate.Every(time.Hour)
	}
	return rate.Every(time.Hour / time.Duration(count))
}

func LimitPerMinute(count int) rate.Limit {
	if count <= 0 {
		return rate.Every(time.Minute)
	}
	return rate.Every(time.Minute / time.Duration(count))
}
