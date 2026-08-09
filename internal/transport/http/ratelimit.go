package http

import (
	"sync"
	"time"
)

type visitor struct {
	count       int
	windowStart time.Time
}

type RateLimiter struct {
	mu sync.Mutex

	visitors map[string]*visitor

	limit  int
	window time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
	}
}

func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	v, exists := l.visitors[key]

	if !exists || now.Sub(v.windowStart) >= l.window {
		l.visitors[key] = &visitor{
			count:       1,
			windowStart: now,
		}

		return true
	}

	if v.count >= l.limit {
		return false
	}

	v.count++

	return true
}
