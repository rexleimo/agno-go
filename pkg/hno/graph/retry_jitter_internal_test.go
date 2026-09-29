package graph

import (
	"math/rand"
	"testing"
	"time"
)

// 切片 35（RetryConfig.Jitter 接线）的白盒确定性测试。分布密度按切片 22 的
// 预登记不进公共面；这里只钉三件可确定断言的事：界限、恒等、可复现。

// TestApplyJitterStaysUnderBoundAndVaries 观察契约 D1：Jitter>0 时 500 次采样
// 全部落在 [0, d) 且至少出现两个不同值（不是恒等返回）。
func TestApplyJitterStaysUnderBoundAndVaries(t *testing.T) {
	const d = 100 * time.Millisecond
	rnd := rand.New(rand.NewSource(42))
	seen := map[time.Duration]bool{}
	for i := 0; i < 500; i++ {
		got := applyJitter(d, 1.0, rnd)
		if got < 0 || got >= d {
			t.Fatalf("采样 %d 越界：%v，want [0, %v)", i, got, d)
		}
		seen[got] = true
	}
	if len(seen) < 2 {
		t.Fatalf("500 次采样只有 %d 个不同值，抖动没有生效", len(seen))
	}
}

// TestApplyJitterIdentityOnNonPositive 观察契约 D2：Jitter<=0 或 d<=0 恒等返回。
func TestApplyJitterIdentityOnNonPositive(t *testing.T) {
	const d = 50 * time.Millisecond
	rnd := rand.New(rand.NewSource(7))
	if got := applyJitter(d, 0, rnd); got != d {
		t.Fatalf("jitter=0 输出 %v，want 恒等 %v", got, d)
	}
	if got := applyJitter(d, -1.5, rnd); got != d {
		t.Fatalf("负 jitter 输出 %v，want 恒等 %v", got, d)
	}
	if got := applyJitter(0, 1.0, rnd); got != 0 {
		t.Fatalf("d=0 输出 %v，want 0", got)
	}
}

// TestApplyJitterDeterministicWithFixedSource 观察契约 D3：同一固定种子的输出
// 序列可复现（rnd 注入语义；全局随机源不可复现，故注入是契约的一部分）。
func TestApplyJitterDeterministicWithFixedSource(t *testing.T) {
	draw := func() []time.Duration {
		rnd := rand.New(rand.NewSource(42))
		out := make([]time.Duration, 16)
		for i := range out {
			out[i] = applyJitter(80*time.Millisecond, 0.5, rnd)
		}
		return out
	}
	a, b := draw(), draw()
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("第 %d 个采样不可复现：%v vs %v", i, a[i], b[i])
		}
	}
}

// TestRetryEnvelopeAppliesJitter 观察契约 D4 的结构面：退避计算点经 applyJitter
// —— Jitter>0 时退避永不放大（上限仍为折减值），Jitter=0 时恰为折减值。
// 真实时钟的休眠时长按切片 22 预登记不进断言（R17 教训：负载下 elapsed 不可信）。
func TestRetryEnvelopeAppliesJitter(t *testing.T) {
	c := RetryConfig{InitialDelay: 40 * time.Millisecond, BackoffFactor: 2, Jitter: 1.0}
	rnd := rand.New(rand.NewSource(99))
	for attempt := 1; attempt <= 5; attempt++ {
		base := c.delayFor(attempt)
		got := applyJitter(base, c.Jitter, rnd)
		if got < 0 || got > base {
			t.Fatalf("attempt %d：抖动后 %v 越出 [0, %v)", attempt, got, base)
		}
	}
}
