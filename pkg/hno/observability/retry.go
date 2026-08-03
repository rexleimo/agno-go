package observability

import (
	"context"
	"time"
)

// RetryConfig configures the retry policy.
// RetryConfig 配置重试策略。
type RetryConfig struct {
	MaxAttempts int           // total attempts incl. first / 总尝试次数（含首次）
	InitialWait time.Duration // first backoff / 首次退避
	MaxWait     time.Duration // cap on backoff / 退避上限
	Jitter      bool          // add random jitter / 是否加随机抖动
}

// DefaultRetryConfig returns a sane default (3 attempts, 200ms→2s).
// DefaultRetryConfig 返回合理默认值（3 次尝试，200ms→2s）。
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{MaxAttempts: 3, InitialWait: 200 * time.Millisecond, MaxWait: 2 * time.Second, Jitter: true}
}

// Retry runs fn, retrying on error with exponential backoff.
// Retry 运行 fn，失败时按指数退避重试。
func Retry(ctx context.Context, cfg RetryConfig, fn func(context.Context) error) error {
	var err error
	wait := cfg.InitialWait
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err = fn(ctx); err == nil {
			return nil
		}
		if attempt == cfg.MaxAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff(wait, cfg)):
		}
		if wait *= 2; cfg.MaxWait > 0 && wait > cfg.MaxWait {
			wait = cfg.MaxWait
		}
	}
	return err
}

func backoff(wait time.Duration, cfg RetryConfig) time.Duration {
	if !cfg.Jitter {
		return wait
	}
	// Deterministic-ish jitter: ±20%.
	// 确定性抖动：±20%。
	jitter := time.Duration(int64(wait) / 5 * (int64(hashSeed())%2*2 - 1))
	return wait + jitter
}

var hashCounter int64

func hashSeed() int64 {
	hashCounter++
	return hashCounter * 2654435761 % 1000
}
