// 节点级策略（母约 §4，G3 策略四合一）：Retry 借鉴 adk retry.go，Cache/Timeout 借鉴
// LangGraph 的 CachePolicy/TimeoutPolicy，Trace 是节点级事件记录。
//
// 挂载走 AddNode 的变参 NodeOption（对既有单参调用源兼容）；取值语义沿 WithMaxConcurrency
// 先例「guard 钳制、不进 Validate」：MaxAttempts<1 即 1 次、Timeout<=0 即不限时，
// 「不重试/不限时/不缓存」都是零值直接给出的语义，引擎不为策略配置新增构建期拒绝。
//
// 执行缝在 scheduler 的 runNode 包络：重试与期限在生产者 goroutine 内生效，缓存查询也在
// 生产者侧；trace 事件由消费者串行发射。全程不引入任何锁（票面 §3.3 零锁模型）。
package graph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// RetryConfig 借鉴 adk retry.go（母约 §4）。
type RetryConfig struct {
	MaxAttempts   int
	InitialDelay  time.Duration
	BackoffFactor float64
	MaxDelay      time.Duration
	Jitter        float64
	ShouldRetry   func(error) bool
}

// CacheConfig 借鉴 LangGraph CachePolicy（母约 §4）。
type CacheConfig struct {
	KeyFunc func(in any) string
	TTL     time.Duration
	Store   CacheStore
}

// CacheStore 泛化缓存契约（母约 §4）。graph 不依赖 internal/cache，适配留给调用方。
type CacheStore interface {
	GetAny(ctx context.Context, key string) (any, bool, error)
	SetAny(ctx context.Context, key string, v any, ttl time.Duration) error
}

// TimeoutConfig 借鉴 LangGraph TimeoutPolicy（母约 §4）。
type TimeoutConfig struct {
	Timeout    time.Duration
	PerAttempt bool
}

// TraceConfig 控制节点级事件记录。Hook 是母约 §4 草图之外由本片补出的 sink：
// 没有 sink 的 trace 无法被任何调用方观察（切片 22 契约 designConsequence 第 5 条）。
type TraceConfig struct {
	Enabled   bool
	RedactIn  bool
	RedactOut bool
	Hook      func(NodeEvent)
}

// NodeEvent 是 Hook 收到的单条节点级事件，由消费者 goroutine 串行发出（零锁不破）。
type NodeEvent struct {
	Node    string
	Attempt int
	Err     error
	In      any
	Out     any
}

// NodeOption 挂载节点级策略；AddNode 变参接收，既有单参调用源兼容。
type NodeOption func(*nodePolicy)

type nodePolicy struct {
	retry   *RetryConfig
	timeout *TimeoutConfig
	cache   *CacheConfig
	trace   *TraceConfig
}

func WithRetry(c RetryConfig) NodeOption     { return func(p *nodePolicy) { p.retry = &c } }
func WithTimeout(c TimeoutConfig) NodeOption { return func(p *nodePolicy) { p.timeout = &c } }
func WithCache(c CacheConfig) NodeOption     { return func(p *nodePolicy) { p.cache = &c } }
func WithTrace(c TraceConfig) NodeOption     { return func(p *nodePolicy) { p.trace = &c } }

// attempts 落实「guard 钳制、不进 Validate」的取值语义（WithMaxConcurrency 先例）：
// 未声明或 MaxAttempts<1 一律 1 次，即「不重试」是零值直接给出的语义。
func (c *RetryConfig) attempts() int {
	if c == nil || c.MaxAttempts < 1 {
		return 1
	}
	return c.MaxAttempts
}

func (c *RetryConfig) should(err error) bool {
	return c != nil && c.ShouldRetry != nil && c.ShouldRetry(err)
}

func (c *RetryConfig) delayFor(tries int) time.Duration {
	if c == nil || c.InitialDelay <= 0 {
		return 0
	}
	d := c.InitialDelay
	for i := 1; i < tries && c.BackoffFactor > 1; i++ {
		d = time.Duration(float64(d) * c.BackoffFactor)
		if c.MaxDelay > 0 && d > c.MaxDelay {
			return c.MaxDelay
		}
	}
	return d
}

func (c *TimeoutConfig) deadline() (time.Duration, bool) {
	if c == nil || c.Timeout <= 0 {
		return 0, false
	}
	return c.Timeout, true
}

func (c *CacheConfig) keyFor(in any) string {
	if c == nil {
		return ""
	}
	if c.KeyFunc != nil {
		return c.KeyFunc(in)
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%#v", in)))
	return hex.EncodeToString(sum[:])
}

func (c *TraceConfig) emit(ev NodeEvent) {
	if c == nil || !c.Enabled || c.Hook == nil {
		return
	}
	if c.RedactIn {
		ev.In = nil
	}
	if c.RedactOut {
		ev.Out = nil
	}
	c.Hook(ev)
}
