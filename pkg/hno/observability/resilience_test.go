package observability

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_SuccessFirstTry(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), DefaultRetryConfig(), func(ctx context.Context) error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetry_RetriesThenSucceeds(t *testing.T) {
	calls := 0
	cfg := DefaultRetryConfig()
	cfg.MaxAttempts = 3
	cfg.Jitter = false
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_ExhaustsAttempts(t *testing.T) {
	calls := 0
	cfg := DefaultRetryConfig()
	cfg.MaxAttempts = 2
	cfg.Jitter = false
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		calls++
		return errors.New("always fails")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestRetry_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Retry(ctx, DefaultRetryConfig(), func(ctx context.Context) error {
		return errors.New("boom")
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRateLimiter_AllowsUpToBurst(t *testing.T) {
	rl := NewRateLimiter(1000, 3) // high rate, burst 3
	allowed := 0
	for i := 0; i < 5; i++ {
		if rl.Allow() {
			allowed++
		}
	}
	if allowed != 3 {
		t.Errorf("expected 3 allowed (burst), got %d", allowed)
	}
}

func TestRateLimiter_Refills(t *testing.T) {
	rl := NewRateLimiter(1000, 1) // 1000 tokens/sec, burst 1
	if !rl.Allow() {
		t.Fatal("first token should be available")
	}
	if rl.Allow() {
		t.Fatal("second token should not be available immediately")
	}
	time.Sleep(5 * time.Millisecond)
	if !rl.Allow() {
		t.Error("token should refill after ~1ms")
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	rl := NewRateLimiter(1000, 1)
	_ = rl.Allow() // consume
	start := time.Now()
	if err := rl.Wait(context.Background()); err != nil {
		t.Fatalf("wait: %v", err)
	}
	if time.Since(start) < time.Millisecond {
		t.Error("wait returned too fast")
	}
}

func TestCircuitBreaker_OpensAndRecovers(t *testing.T) {
	cb := NewCircuitBreaker(3, 50*time.Millisecond)

	// Closed: all allowed.
	// 关闭：全部允许。
	for i := 0; i < 5; i++ {
		if !cb.Allow() {
			t.Fatal("should be allowed while closed")
		}
	}

	// 3 failures → open.
	// 3 次失败 → 断开。
	cb.Failure()
	cb.Failure()
	cb.Failure()
	if cb.Allow() {
		t.Fatal("should be open after 3 failures")
	}

	// After cooldown, probe allowed; success resets.
	// 冷却后允许探测；成功复位。
	time.Sleep(60 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("probe should be allowed after cooldown")
	}
	cb.Success()
	if !cb.Allow() {
		t.Error("should be closed again after success")
	}
}

func TestPromptCache_Basic(t *testing.T) {
	c := NewPromptCache(10)
	key := c.Key("hello world")
	if _, ok := c.Get(key); ok {
		t.Fatal("empty cache should miss")
	}
	c.Put(key, CacheEntry{Response: "hi"})
	entry, ok := c.Get(key)
	if !ok || entry.Response != "hi" {
		t.Errorf("cache miss after put: %+v", entry)
	}
	if c.Len() != 1 {
		t.Errorf("Len = %d, want 1", c.Len())
	}
}

func TestPromptCache_EvictsOldest(t *testing.T) {
	c := NewPromptCache(2)
	c.Put("a", CacheEntry{Response: "1"})
	c.Put("b", CacheEntry{Response: "2"})
	c.Put("c", CacheEntry{Response: "3"})
	if c.Len() != 2 {
		t.Errorf("Len = %d, want 2", c.Len())
	}
	if _, ok := c.Get("a"); ok {
		t.Error("oldest entry should be evicted")
	}
	if _, ok := c.Get("c"); !ok {
		t.Error("newest entry should remain")
	}
}

func TestPromptCache_KeyStable(t *testing.T) {
	c := NewPromptCache(10)
	if c.Key("same") != c.Key("same") {
		t.Error("key should be deterministic")
	}
	if c.Key("same") == c.Key("different") {
		t.Error("different prompts should differ")
	}
}
