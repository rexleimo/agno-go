package observability

import (
	"errors"
	"sync"
	"time"
)

// CircuitBreaker opens after consecutive failures, then allows a probe
// after a cooldown. Half-open probes reset on success.
//
// CircuitBreaker 在连续失败后断开，冷却后允许探测。
// 半开探测成功后复位。
type CircuitBreaker struct {
	mu          sync.Mutex
	maxFailures int
	cooldown    time.Duration
	failures    int
	state       int // 0=closed, 1=open, 2=half-open
	openedAt    time.Time
}

// NewCircuitBreaker creates a breaker with the given failure threshold.
// NewCircuitBreaker 创建给定失败阈值的熔断器。
func NewCircuitBreaker(maxFailures int, cooldown time.Duration) *CircuitBreaker {
	if maxFailures <= 0 {
		maxFailures = 5
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &CircuitBreaker{maxFailures: maxFailures, cooldown: cooldown}
}

// ErrOpen is returned when the breaker is open.
// ErrOpen 在熔断器断开时返回。
var ErrOpen = errors.New("circuit breaker open")

// Allow reports whether a call may proceed.
// Allow 报告调用是否允许进行。
func (b *CircuitBreaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case 0: // closed / 关闭
		return true
	case 1: // open / 断开
		if time.Since(b.openedAt) >= b.cooldown {
			b.state = 2 // half-open / 半开
			return true
		}
		return false
	default: // half-open / 半开：只允许一次探测
		return false
	}
}

// Success records a successful call.
// Success 记录一次成功调用。
func (b *CircuitBreaker) Success() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.state = 0
}

// Failure records a failed call.
// Failure 记录一次失败调用。
func (b *CircuitBreaker) Failure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.state == 2 || b.failures >= b.maxFailures {
		b.state = 1
		b.openedAt = time.Now()
	}
}
