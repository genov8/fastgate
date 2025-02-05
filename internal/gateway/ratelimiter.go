package gateway

import (
	"sync"
	"time"
)

type RateLimiter struct {
	limits map[string]*rateLimit
	mu     sync.Mutex
}

type rateLimit struct {
	limit     int
	interval  time.Duration
	count     int
	lastReset time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limits: make(map[string]*rateLimit),
	}
}

func (rl *RateLimiter) AllowRequest(key string, limit, interval int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if _, exists := rl.limits[key]; !exists {
		rl.limits[key] = &rateLimit{
			limit:     limit,
			interval:  time.Duration(interval) * time.Second,
			lastReset: now,
		}
	}

	limitData := rl.limits[key]

	if now.Sub(limitData.lastReset) > limitData.interval {
		limitData.count = 0
		limitData.lastReset = now
	}

	if limitData.count < limitData.limit {
		limitData.count++
		return true
	}

	return false
}
