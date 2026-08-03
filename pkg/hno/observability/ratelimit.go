package observability

import (
	"context"
	"sync"
	"time"
)

// RateLimiter is a simple token bucket limiter.
// RateLimiter 是简单的令牌桶限流器。
type RateLimiter struct {
	mu     sync.Mutex
	rate   float64 // tokens per second / 每秒令牌数
	burst  float64 // max bucket size / 桶容量
	tokens float64
	last   time.Time
}

// NewRateLimiter creates a limiter with the given rate (tokens/sec) and burst.
// NewRateLimiter 创建给定速率（令牌/秒）和容量的限流器。
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	if rate <= 0 {
		rate = 1
	}
	if burst <= 0 {
		burst = 1
	}
	return &RateLimiter{
		rate:   rate,
		burst:  float64(burst),
		tokens: float64(burst),
		last:   time.Now(),
	}
}

// Allow reports whether a token is available immediately.
// Allow 报告是否立即可用令牌。
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(r.last).Seconds()
	r.last = now
	r.tokens += elapsed * r.rate
	if r.tokens > r.burst {
		r.tokens = r.burst
	}
	if r.tokens >= 1 {
		r.tokens--
		return true
	}
	return false
}

// Wait blocks until a token is available or ctx is done.
// Wait 阻塞直到获得令牌或 ctx 结束。
func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		if r.Allow() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second / time.Duration(r.rate)):
		}
	}
}
