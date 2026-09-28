---
title: "节点策略四合一：Retry/Timeout/Cache/Trace 的一次 variadic 设计"
description: "HNO 如何用一条变参 AddNode 选项缝把重试、超时、缓存、追踪挂到图节点上 —— 零值即语义、fail-closed 缓存、消费侧 trace hook。"
date: 2026-09-28
lastUpdated: 2026-09-28
author: HNO Team
category: API design
tags:
  - Go
  - API 设计
  - 重试
  - 缓存
  - 超时
  - 可观测性
head:
  - - meta
    - name: keywords
      content: "Go 变参选项, NodeOption 设计, 节点策略, 重试超时缓存追踪, 零值语义, fail-closed 缓存, Agent 框架 API 设计"
  - - meta
    - property: og:type
      content: article
  - - meta
    - property: og:title
      content: "节点策略四合一：Retry/Timeout/Cache/Trace 的一次 variadic 设计"
  - - meta
    - property: og:description
      content: "重试、超时、缓存、追踪作为 AddNode 的变参选项：零值即语义、fail-closed 缓存、零锁调度器里的串行 trace hook。"
  - - meta
    - property: article:published_time
      content: "2026-09-28T00:00:00Z"
  - - link
    - rel: canonical
      href: https://hno.rexai.top/zh/blog/node-option-variadic-design
---

# 节点策略四合一：Retry/Timeout/Cache/Trace 的一次 variadic 设计

图引擎长策略的方式和城市长红绿灯一样：起初很慢，然后一下子全来了。节点先需要
重试，然后是超时，然后是缓存，然后是追踪 —— 而最顺手的 API，一个十四个字段的
策略大结构，恰恰是框架走向这种调用方的路：

```go
AddNode(NodeConfig{
    Name: "fetch", Fn: fn, Retry: &RetryPolicy{...}, Timeout: &TimeoutPolicy{...},
    Cache: &CachePolicy{...}, Trace: &TracePolicy{...}, // ……还能继续加
})
```

就为了说一句「跑这个节点」。这篇讲我们为 HNO 图引擎实际发布的替代 API —— 一
条变参缝、零值即语义、以及几处刻意**不**做校验的地方。

## 缝：`AddNode(n, opts...)`

```go
g.AddNode(
    graph.NodeFunc("fetch", fetchFn),
    graph.WithRetry(graph.RetryConfig{MaxAttempts: 3, InitialDelay: 100 * time.Millisecond}),
    graph.WithTimeout(graph.TimeoutConfig{Timeout: 30 * time.Second}),
    graph.WithCache(graph.CacheConfig{TTL: 10 * time.Minute, Store: store}),
    graph.WithTrace(graph.TraceConfig{Enabled: true, Hook: logEvent}),
)
```

`func (g *Graph) AddNode(n Node, opts ...NodeOption) *Graph`。变参选项是 Go 的
老习惯，但这里要紧的性质是**向后兼容**：所有调用方既有的 `AddNode(n)` 原样编
译、原样运行。图级 `WithMaxConcurrency` 已经用过同一招，`NodeOption` 是沿家族
先例，不是另起炉灶。

选项插进的那个执行包络次序固定：缓存查询（命中即跳过执行）、至多 N 次带期限
的尝试、成功后写缓存、最后发 trace。重试留在一次激活里 —— 不消耗步数预算，
因为步数阀计的是*派发的激活数*，重试不是派发。

## 逐策略一节

命名出处直说：重试参考 adk 的 `retry.go`；缓存与超时沿 LangGraph 的
`CachePolicy`/`TimeoutPolicy` —— 好设计，有意识地借用。Trace 是我们自己的。

### Retry

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

从 `InitialDelay` 起按 `BackoffFactor` 指数退避、`MaxDelay` 封顶；
`ShouldRetry` 是是否再试的门槛（`nil` 从不重试）。一个我们在意的细节：退避等
待对取消敏感 —— `select` 监听 `ctx.Done()` —— 被放弃的运行在被放弃的那一刻停
止等待，而不是睡醒再说。

### Timeout

```go
type TimeoutConfig struct {
    Timeout    time.Duration
    PerAttempt bool
}
```

`PerAttempt: false` 为**整个节点装一个预算、重试全算在内** —— 期限在第一次尝
试之前就存在。`PerAttempt: true` 每次尝试铸一个新期限。区别不是学院式的：
节点预算 30 秒时三次尝试共享 30 秒；逐次预算则各有 30 秒。测试钉的是*期限值*
而不是耗时，在负载飘忽的 CI 机器上仍然确定。

### Cache

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

刻意不附带默认 store：graph 包不依赖任何缓存实现，接你的后端就是实现一个两方
法接口。不给 `KeyFunc` 时，键是输入 `%#v` 渲染的 SHA-256 —— 够有用，也够你明
确地替换掉。

### Trace

```go
type TraceConfig struct {
    Enabled   bool
    RedactIn  bool
    RedactOut bool
    Hook      func(NodeEvent)
}
```

`NodeEvent` 携带 `{Node, Attempt, Err, In, Out}` —— 每个结果一条事件，成功与
失败*都*发。`RedactIn`/`RedactOut` 在送达前把载荷置 nil，接到日志 sink 的
hook 永远看不到 prompt。

## 零值即语义

这是本文要辩护的设计决定。策略配置**在读取时由 guard 钳制，不在构建期校验**：

| 声明 | 生效语义 |
|---|---|
| 不带 `WithRetry` / `RetryConfig{}` | 1 次尝试 —— `MaxAttempts < 1` 钳为 1 |
| `ShouldRetry: nil` | 从不重试 |
| 不带 `WithTimeout` / `Timeout <= 0` | 不限时 |
| 不带 `WithCache` / `Store: nil` | 不缓存 |
| 不带 `WithTrace` / `Enabled: false` | 零事件 |

「零值是可用配置」是 Go 惯用法在策略上的应用：*不重试*、*不限时*、*不缓存*都是
调用方真实想要的状态，而每个都恰好离一个零值一步。策略没有新增任何 `Validate()`
规则 —— 零，这个计数本身是契约的一部分。

而同一个引擎却把 `WithStepLimit(0)` **直接拒绝**：

```
graph: WithStepLimit(0) is not a usable step limit: a positive number of steps is required
```

为什么不对称？因为非正的步数上限没有可读的含义：读成「不限」，它恰好在最需要
安全阀的环图上把阀拆了；读成「零步」，它在入口就把每次运行拒掉。两种读法都
错，所以声明本身错。策略零值则各自映射到恰好一种合理状态。两种口径 —— 无歧义
的钳制、有歧义的拒绝 —— 按旋钮分别选定。一条统一规则（「一律校验」或「一律
钳制」）必会让这两个旋钮之一变差。

## fail-closed 的缓存

只有成功结果进缓存。失败的节点什么都不写。

另一种做法 —— 把错误也缓存以「避免重复失败工作」—— 会把一次瞬时故障变成持
久故障：下一次运行把缓存里的失败当结论收下。变异工装翻转这一行时（变异 m6：
「失败也写缓存」），套件恰好在钉死它的那条测试上判红（D11，失败绝不入缓存）。
这就是「说出来的不变量」与「测出来的不变量」的差别。

## trace 为什么放消费侧

引擎是单消费者零锁调度器：生产者 goroutine 跑节点、向 channel 发完成项；一个
消费者独占全部可变状态。trace 事件由**那个消费者**在结果确定后**串行**发射
—— 这买到三件事：

1. hook 契约不需要同步。一个普通函数，没有锁，没有自己的 channel。
2. 零锁不变量存活 —— 引擎文件里的 `sync.` 计数保持为零。
3. 事件按完成处理次序到达，与调度器自己相信的次序相同。

代价同样诚实：hook 跑在调度路径上，hook 慢就拖慢每次节点完成。重 sink 请在你
的 hook 里做缓冲。

## 变异矩阵抓住了什么

策略片随附八条变异，全部杀红：

| 变异 | 破坏方式 | 被谁抓住 |
|---|---|---|
| m1 | 重试循环失效（恒 break） | D1/D3/D5/D8 |
| m2 | `ShouldRetry` 取反 | D1/D3/D4/D5/D8 |
| m3 | 期限钳制失效 | D6/D8 |
| m4 | `PerAttempt` 退化成单次期限 | D8 |
| m5 | 缓存查询失效 | D9/D10/D11 |
| m6 | 失败也写缓存 | D11 |
| m7 | 完成 trace 发射被摘 | D12/D13 |
| m8 | 脱敏分支被摘 | D13 |

一条等价项是登记出来的而不是宣称的：e1（失败路径的 trace 发射）—— 工装无法
用行为可观察的测试区分它，于是如实登记为等价，而不是静默「通过」。m2 是我最
不敢押注的那条：反转一个谓词是那种让*某些*测试「更绿」、另一些判红的 bug，它
仍然杀掉了五个测试组。m4 最隐蔽：一切照样超时、尝试照样发生 —— 只有期限的
*形状*错了，只有断言期限值的那条测试看得见。

## 这个设计证明不了什么

- **不处理生产者 goroutine 内的重入。** 策略在跑节点的生产者 goroutine 内生
  效；节点自己再派生工作、重入策略路径（比如嵌套图共用同一个 cache store）是
  调用方领地。引擎不做重入承诺。
- **缓存缝不附带 store。** 「两方法接你的后端」对足迹是优点、对你是待办；盒子里
  没有 Redis/Postgres 适配器，TTL 语义就是你的 `SetAny` 对那个参数做的事。
- **`Jitter` 字段在 config 结构体里存在，但已落地的调度器尚未应用它。** 与其
  假装有雷群保护，不如今天就把话说明白。
- **trace 是节点级，不是运行级。** 完整运行的分布式追踪是另一条可观测性线；
  `NodeEvent` hook 是节点范围的缝。

## 复现

```bash
go test ./pkg/hno/graph -run TestP2G3_ -count=1 -race -v
grep -o 'sync\.' pkg/hno/graph/graph.go pkg/hno/graph/scheduler.go \
  pkg/hno/graph/policy.go pkg/hno/graph/durability.go | wc -l   # 期望 0
```

完整的策略走读与可运行示例见[节点策略指南](/zh/guide/node-policies)；签名在
[Graph API 参考](/zh/api/graph)。

## 延伸阅读

- [上一篇：在 Go 里造一个 LangGraph —— 零锁图引擎](/zh/blog/zero-lock-graph-engine)
- [图引擎指南](/zh/guide/graph-engine)
- [全部博客文章](/zh/blog/)
