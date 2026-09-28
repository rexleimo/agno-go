# 节点策略 - Retry、Timeout、Cache、Trace

通过一个变参缝 `AddNode(n, opts...)` 把执行策略挂到单个图节点上。

---

## 变参缝

节点接受任意数量的 `NodeOption`。既有的单参 `AddNode(n)` 调用完全不受影响；
策略逐节点全可选：

```go
g.AddNode(
	graph.NodeFunc("summarize", summarizeFn),
	graph.WithRetry(graph.RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		BackoffFactor: 2.0,
		MaxDelay:      2 * time.Second,
	}),
	graph.WithTimeout(graph.TimeoutConfig{
		Timeout: 30 * time.Second,
	}),
	graph.WithCache(graph.CacheConfig{
		TTL:   10 * time.Minute,
		Store: myStore,
	}),
	graph.WithTrace(graph.TraceConfig{
		Enabled:  true,
		RedactIn: true,
		Hook:     func(ev graph.NodeEvent) { log.Printf("%+v", ev) },
	}),
)
```

同一节点名再次注册时，后注册者的 options 完整接管该节点的策略表。

四个策略在调度器的单激活包络内按固定次序生效：

1. **缓存查询** —— 命中则完全跳过节点执行。
2. **至多 N 次带期限的尝试** —— 重试循环。
3. **缓存写入** —— 仅成功时。
4. **trace 发射** —— 结果确定后由消费者 goroutine 发出。

重试停留在一次激活内：不产生新激活、不消耗步数预算。退避等待随时让位给调用
方的取消。

---

## 重试 —— `WithRetry(RetryConfig)`

```go
type RetryConfig struct {
	MaxAttempts   int
	InitialDelay  time.Duration
	BackoffFactor float64
	MaxDelay      time.Duration
	Jitter        float64
	ShouldRetry   func(error) bool
}
```

语义：

- 重试的是**同一次激活**，至多 `MaxAttempts` 次；耗尽或 `ShouldRetry` 拒绝时
  交出最后一次的错误。
- `ShouldRetry` 是是否再试的门槛；`nil` 表示从不重试（第一次的错误直接生效）。
- 退避：第 *k* 次尝试等待 `InitialDelay * BackoffFactor^(k-1)`，封顶
  `MaxDelay`。`InitialDelay <= 0` 则完全不等待。
- 退避等待期间取消 context 立即中断，返回 `ctx.Err()`。
- 诚实说明：`Jitter` 字段在 config 结构体里存在，但已落地的调度器尚未应用它。

---

## 超时 —— `WithTimeout(TimeoutConfig)`

```go
type TimeoutConfig struct {
	Timeout    time.Duration
	PerAttempt bool
}
```

期限以派生 `context` 的形式送达节点 —— 尊重 `ctx` 的节点就尊重预算。两种形状：

| 设置 | 含义 |
|---|---|
| `PerAttempt: false`（默认） | **整个节点**一个预算，覆盖**全部**重试尝试。期限在重试循环之外一次性挂好。 |
| `PerAttempt: true` | **每次尝试一个全新期限** —— 每次重试都拿到完整的 `Timeout`。 |

`Timeout <= 0` 表示不限时 —— 零值是可用且刻意的配置。

---

## 缓存 —— `WithCache(CacheConfig)`

```go
type CacheConfig struct {
	KeyFunc func(in any) string
	TTL     time.Duration
	Store   CacheStore
}

type CacheStore interface {
	GetAny(ctx context.Context, key string) (any, bool, error)
	SetAny(ctx context.Context, key string, v any, ttl time.Duration) error
}
```

- **键**：给 `KeyFunc` 就用领域语义的键；不给则用输入 `%#v` 渲染的 SHA-256
  十六进制摘要。
- **存储**：引擎不附带默认 store。用你的后端实现两方法的 `CacheStore` 接口即
  可 —— graph 包刻意不依赖任何缓存实现。
- **TTL** 透传给 `SetAny`；如何解释是 store 的事。
- **fail-closed 缓存**：只有成功结果会被写入。失败的节点绝不进缓存 —— 把错误
  缓存进去，等于把那次错误当作结论交给下一次运行。缓存命中立即返回，计为一次
  尝试。

---

## 追踪 —— `WithTrace(TraceConfig)`

```go
type TraceConfig struct {
	Enabled   bool
	RedactIn  bool
	RedactOut bool
	Hook      func(NodeEvent)
}

type NodeEvent struct {
	Node    string
	Attempt int
	Err     error
	In      any
	Out     any
}
```

- 每个节点结果恰发一条事件 —— 成功与失败路径都发（失败节点的事件携带
  `Err`）。
- 事件由**消费者 goroutine 串行发射**，因此 hook 自身不需要任何同步，零锁模型
  不被破坏。hook 要快：它跑在调度路径上。
- `RedactIn` / `RedactOut` 在送达前把 `In` / `Out` 置 nil，prompt、密钥之类
  的载荷不会到达 sink。
- `Enabled: false` 或 `nil` hook 零发射 —— 一个真正关得上的开关。

---

## 零值即语义

策略配置在读取时由 guard 钳制，而不是构建期校验。每个 config 的零值都是可
用、刻意的配置 —— 「不重试 / 不限时 / 不缓存 / 不追踪」就是不声明该选项时
直接得到的东西：

| 声明 | 生效语义 |
|---|---|
| 不带 `WithRetry` | 1 次尝试，不重试 |
| `RetryConfig{}`（零值） | 1 次尝试 —— `MaxAttempts < 1` 钳制为 1 |
| `ShouldRetry: nil` | 从不重试 |
| 不带 `WithTimeout` | 不限时 |
| `TimeoutConfig{}` / `Timeout <= 0` | 不限时 |
| 不带 `WithCache` / `Store: nil` | 不缓存 |
| `KeyFunc: nil` | 输入 `%#v` 的 SHA-256 |
| 不带 `WithTrace` / `Enabled: false` / `Hook: nil` | 零事件 |

这与图级 `WithStepLimit(0)` 被 `Validate()` **直接拒绝**形成刻意对比：非正的
步数上限没有可读的含义（既不是「不限」也不是「零步」），而一个在关键时刻把自己
关掉的安全阀比构建期报错更糟。策略零值则都映射到「该策略未生效」—— 恰好只有
一种合理解读的状态。两种口径按旋钮分别选定，而不是一条规则强行统一。

---

## 策略在哪里执行

- **重试、超时、缓存**在执行该节点的生产者 goroutine 内运行。它们不产生激
  活、不触碰调度器状态。
- **trace** 事件由唯一消费者发射 —— 正因如此，hook 契约可以不靠锁就承诺串行
  送达。
- 整个包络在步数记账上算一次激活：重试五次的节点仍然只消耗一步。

---

## 完整示例

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

type memStore struct{ m map[string]any }

func (s *memStore) GetAny(ctx context.Context, key string) (any, bool, error) {
	v, ok := s.m[key]
	return v, ok, nil
}
func (s *memStore) SetAny(ctx context.Context, key string, v any, ttl time.Duration) error {
	s.m[key] = v
	return nil
}

func main() {
	store := &memStore{m: map[string]any{}}
	calls := 0

	flaky := graph.Typed[string, string]("flaky", func(ctx context.Context, in string) (string, error) {
		calls++
		if calls < 3 {
			return "", fmt.Errorf("transient upstream error %d", calls)
		}
		return "ok:" + in, nil
	})

	g := graph.New()
	g.AddNode(flaky,
		graph.WithRetry(graph.RetryConfig{
			MaxAttempts:   4,
			InitialDelay:  10 * time.Millisecond,
			BackoffFactor: 2.0,
			ShouldRetry:   func(err error) bool { return true },
		}),
		graph.WithTimeout(graph.TimeoutConfig{Timeout: 5 * time.Second}),
		graph.WithCache(graph.CacheConfig{TTL: time.Minute, Store: store}),
		graph.WithTrace(graph.TraceConfig{
			Enabled: true,
			Hook:    func(ev graph.NodeEvent) { fmt.Printf("node=%s attempt=%d err=%v\n", ev.Node, ev.Attempt, ev.Err) },
		}),
	)
	g.SetEntry("flaky")
	g.SetOutput("flaky")

	res, err := g.Run(context.Background(), "payload")
	fmt.Println(res.Output(), err) // ok:payload <nil> —— 第三次尝试成功
}
```

---

## 下一步

- 读[图引擎指南](/zh/guide/graph-engine)了解路由与校验。
- 看[图持久化](/zh/advanced/graph-durability)—— 提交侧持久化与策略可在相同
  节点上组合。
- 查阅[Graph API 参考](/zh/api/graph)。
