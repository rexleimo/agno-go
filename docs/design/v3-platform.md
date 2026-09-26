# Agno-Go v3.0 平台设计（图引擎 + 运行时底座）

> 状态：草案，待评审
> 前序文档：`go-agent-framework-design.md`（v2 设计，M1–M6 未闭环，版本停在 1.2.9）
> 本文取代 v2 设计文档 §4 中「循环归属」「可观测性」两条决策的部分内容，改动之处均标注 **[推翻]**。

---

## 0. 决策摘要

### 0.1 背景事实（均已核实）

| 事实 | 证据 |
|---|---|
| 当前版本 1.2.9，v2 设计文档已写但未发版 | `CHANGELOG.md` 最高 `1.2.9` |
| `pkg/hno/runner` 是**死代码** | 全仓库 `grep hno/runner` 零引用 |
| 存在 **4 份循环实现** | `agent/run.go:60`、`agent/stream.go:259`、`runner/runner.go:180`、`run/loop.go`（仅 team 用） |
| M2 工具类型安全未做 | `pkg/hno/tools/` 无 `reflect`/jsonschema 生成 |
| M5 workflow builder 未做 | 仍是线性 `[]Step` |
| observability 有零件未接线 | retry/breaker/ratelimit 在业务侧零调用 |
| 契约测试是硬约束 | `internal/session/contract` 9 个测试，Go↔Python fixture 对齐 |

**结论：当前最大的风险不是缺功能，而是「半迁移状态」——一半代码按旧路径（agent 内部循环），一半按新路径（runner 死代码）。**

### 0.2 本次推翻的决策

v2 设计文档 §4 的两条否决被推翻：

| v2 文档 §4 决策 | 原「放弃的备选」 | v3 裁定 |
|---|---|---|
| 循环归属 | ❌「图引擎（LangGraph Pregel 在 Go 维护成本过高）」 | ✅ **引入图引擎**。原论据有误：LangGraph 19 万行中引擎内核（`_loop`+`_algo`+`_runner`）仅约 4,400 行；adk `workflow` 包 6,244 行实现 + 13,800 行测试。成本可控 |
| 可观测性 | ❌「自研事件体系」 | ⚠️ **部分推翻**：仅「会话持久化事件化」（G7），**可观测仍用 OTel**。两者并行不冲突 |

### 0.3 三条不变的红线

- 红线 1：构造器 ≤ 10 参数，跨切面走接口 + Options
- 红线 2：循环只有一份，sync/stream 共享内核
- 红线 4：禁止 `map[string]any` 作为**公共 API** 参数（引擎内部可用 `any`，见 §3.4）

---

## 1. 参考来源分配

按「谁解决得更好」选参考，不按语言相似度。

| 能力 | 参考 | 理由 |
|---|---|---|
| 控制流图调度（单消费者零锁） | **adk-go** | Go 原生；producer-consumer 共享单 channel，消费者独占可变状态 → 全程无锁 |
| 图构建期校验 | adk-go | `validation.go` 525 行，成熟 |
| 节点重试（退避 + jitter） | adk-go | 比 LangGraph 简洁，够用 |
| 事件溯源 / HITL interrupt+resume | adk-go | 幂等 re-entry/handoff 语义完整 |
| **动态扇出 `Send`** | **LangGraph** | adk 静态边无法表达运行时决定扇出数量 |
| **节点 Cache / Timeout / Trace 策略** | **LangGraph** | adk 的 RetryConfig 只有退避+jitter |
| **StreamMode 七模式** | **LangGraph** | 产品级流式可观测协议 |
| **Durability 三档** | **LangGraph** | 可靠性/性能旋钮交给用户 |
| **recursion_limit 全局步数上限** | **LangGraph** | 图有环就必须有安全阀 |
| **函数式节点装饰器** | **LangGraph** | 降低入门门槛 |
| **Store 长期记忆** | **LangGraph** | adk 无此概念（见 §7） |

### 明确不抄

- ❌ Pregel 的 BSP 超步屏障 —— 我们要**控制流图**，不是数据流图
- ❌ Checkpoint time-travel / 状态归约 channel —— 我们的状态是对话，不是纯数据
- ❌ 完整 `EventActions` 语义（ArtifactDelta/TransferToAgent/Escalate/Compaction）—— adk 为其云场景长出，我们不需要

---

## 2. 范围

### 2.1 做（v3.0）

| 块 | 内容 | 参考 | 预估 LOC |
|---|---|---|---|
| G1 | agent 循环收口到 `runner`，消灭重复循环 | 红线 2 | ~1,500 |
| G2 | `pkg/hno/graph` 图引擎 v1 | adk | ~1,200 |
| G3 | 节点策略四合一：Retry/Cache/Timeout/Trace | LangGraph | ~600 |
| G4 | StreamMode 七模式统一流式协议 | LangGraph | ~800 |
| G5 | 动态扇出 `Send` + map-reduce | LangGraph | ~500 |
| G6 | Durability 三档 + recursion_limit | LangGraph | ~400 |
| G7 | 会话事件化 + HITL interrupt/resume | adk | ~1,500 |
| G8 | `Store` 长期记忆（层级 namespace + 向量） | LangGraph | ~1,200 |
| G9 | 观测接线（现有零件接进 G1） | 自有 | ~600 |
| G10 | workflow 迁移到图，公共 API 不变 | — | ~1,200 |
| | **合计** | | **~9,500** |

### 2.2 不做（v3.0 明确排除）

- compaction / 上下文压缩（服务长会话，规模未到）
- trigger（Eventarc/PubSub 事件驱动）
- A2A 增强、auth 凭证抽象
- agent 泛型化 `Agent[D, O]`（破坏性大改，G10 稳定后单独做）
- Pregel / BSP / checkpoint time-travel

---

## 3. G2 图引擎（核心）

### 3.1 包边界

```
pkg/hno/graph/          新增。图引擎，不依赖 workflow/agent
pkg/hno/runner/         G1 收口后，Agent 实现 graph.Node
pkg/hno/workflow/       G10 内部改用 graph 编译，公共 API 不变
```

**依赖方向单向**：`workflow → graph`，`graph` 不反向依赖任何上层包。

### 3.2 公共 API

```go
package graph

// Node 是图节点。刻意极简——节点内部想怎么循环都行。
type Node interface {
    Name() string
    Run(ctx context.Context, in any) (any, error)
}

// NodeFunc 适配普通函数
func NodeFunc(name string, fn func(ctx context.Context, in any) (any, error)) Node

// Predicate 是条件边的路由判定
type Predicate func(out any) bool

// Graph 用 builder 构造，构造期完成全部校验
type Graph struct { /* ... */ }

func New(opts ...Option) *Graph
func (g *Graph) AddNode(n Node) *Graph
func (g *Graph) AddEdge(from, to string) *Graph                      // 无条件
func (g *Graph) AddConditional(from, to string, p Predicate) *Graph // 条件
func (g *Graph) AddDefault(from, to string) *Graph                   // 兜底
func (g *Graph) AddJoin(from []string, to string) *Graph             // 汇聚屏障
func (g *Graph) SetEntry(name string) *Graph
func (g *Graph) SetOutput(name string) *Graph

// 配置
func WithMaxConcurrency(n int) Option      // 0 = 不限
func WithStepLimit(n int) Option           // recursion_limit，默认 1000
func WithRetry(cfg RetryConfig) Option
func WithCache(cfg CacheConfig) Option
func WithDurability(d Durability) Option
func WithStreamMode(m StreamMode) Option

// Validate 构造期校验；Add* 已内建调用，Run 前再校验一次
func (g *Graph) Validate() error

// Run 驱动图执行
func (g *Graph) Run(ctx context.Context, in any) (*Result, error)
```

### 3.3 零锁调度器（核心技术论证）

借鉴 adk `scheduler.go` 的 producer-consumer 模型。**这是 Go 做图引擎成本可控的根本原因。**

```go
type queueItem struct {
    kind    itemKind   // completion | retryTimer
    name    string
    out     any
    err     error
    next    []activation // Send 动态扇出
}

type scheduler struct {
    g       *graph
    queue   chan queueItem    // 唯一共享状态
    runs    map[string]*nodeRun  // consumer-only，无锁
    state   *RunState            // consumer-only，无锁
    cancels map[string]context.CancelFunc // consumer-only，无锁
    pending []activation          // 并发打满时的等待队列
    steps   int                  // recursion_limit 计数
}
```

**不变式**：
1. **生产者**（每节点 goroutine）只做三件事：跑节点、`recover` panic 成 error、往 `queue` 发送。**从不读 queue，从不碰任何 scheduler 字段。**
2. **消费者**（调用 `Run` 的那个 goroutine）是 `queue` 的唯一读者，`runs`/`state`/`cancels`/`pending`/`steps` 的唯一写者。
3. 双方只共享 `queue`（channel 本身并发安全）→ **消费者独占字段全部不需要 mutex**。

**为什么这在 Go 里比 Python 简单**：Python 版必须用 `asyncio.Lock` / `asyncio.Queue` 保护这些状态；Go 的「单 goroutine 独占可变状态」是天然模式，**把锁的问题在结构上消除了**。

**取消**：`cancelAll()` 遍历 `cancels` 调 `context.CancelFunc`；在途节点返回 `context.Canceled`，consumer 统一归一化为取消而非错误。

**并发上限**：`maxConcurrency > 0 && len(runs) >= maxConcurrency` → 激活进 `pending`（`NodePending`），每次 completion 后 `tryDispatchPending()` FIFO 派发。

**步数上限**：`steps >= stepLimit` → 返回 `ErrStepLimitExceeded`（对应 LangGraph `GraphRecursionError`）。**图有环就必须有。**

### 3.4 路由核心（借鉴 adk `findSuccessors`）

```
对 current 的所有出边：
  1. 无条件边          → 全部激活
  2. 条件边            → Predicate(out)==true 激活
  3. Join 目标         → 仅当所有前驱 Completed，聚合前驱输出后激活
  4. 若无任何具体边命中 → 激活 AddDefault 边
  5. 扇出时给每个分支打 sub-branch（隔离 LLM 历史）
```

**红线 4 冲突的处理**：`Node.Run(in any) (any, error)` 用了 `any`，这是**引擎契约**，不是用户配置的公共 API 参数。为降低代价：

- v1 提供泛型包装 `graph.Typed[TIn, TOut]`（Go 1.24 泛型），用户可获得编译期类型安全
- schema 校验（`InputSchema`/`OutputSchema`，借鉴 adk）留 v3.1

### 3.5 构建期校验（借鉴 adk `validation.go`，精简版）

`Validate()` 必查项（全部在 `Add*` 时增量检查，`Run` 前全量复查）：

1. 悬空边（`from`/`to` 指向不存在的节点）
2. 重复节点名
3. 无入口 / 入口不存在
4. 不可达节点（从入口 BFS 不可达 → 报错而非静默）
5. 无出口（无出边的非输出节点）
6. Join 节点的前驱数 < 2
7. **环检测**（区分「合法环」与「死循环」：合法环需由条件边构成，无条件边成环 = 错误）
8. `stepLimit` 校验：存在无条件环时若未设 `stepLimit` → 警告

---

## 4. G3 节点策略四合一

```go
// RetryConfig 借鉴 adk retry.go（含 jitter）
type RetryConfig struct {
    MaxAttempts   int
    InitialDelay  time.Duration
    BackoffFactor float64
    MaxDelay      time.Duration
    Jitter        float64
    ShouldRetry   func(error) bool   // 默认不重试确定性输入错误
}

// CacheConfig 借鉴 LangGraph CachePolicy
type CacheConfig struct {
    KeyFunc func(in any) string  // 默认 hash(in)
    TTL     time.Duration       // 0 = 永不过期
    Store   CacheStore          // 复用现有 cache.Provider
}

// CacheStore 是泛化后的缓存契约。
// 现有 cache.Provider（152 LOC）泛化到 any，ModelResponse 场景为其特化。
type CacheStore interface {
    GetAny(ctx context.Context, key string) (any, bool, error)
    SetAny(ctx context.Context, key string, v any, ttl time.Duration) error
}

// TimeoutConfig 借鉴 LangGraph TimeoutPolicy
type TimeoutConfig struct {
    Timeout   time.Duration
    PerAttempt bool // true=每次尝试超时；false=整个节点超时
}

// TraceConfig 控制节点级 span 记录
type TraceConfig struct {
    Enabled  bool
    RedactIn bool  // 脱敏输入
    RedactOut bool
}
```

**与现有 `cache` 包共存**：`cache.Provider` 保留不变（agent 的 LLM 响应缓存继续用），新增 `CacheStore` 接口，graph 节点用泛化版。

---

## 5. G4 StreamMode 七模式

```go
type StreamMode int
const (
    StreamValues    StreamMode = iota // 每步后的完整状态
    StreamUpdates                     // 仅各节点返回的增量
    StreamMessages                    // token 级 LLM 消息
    StreamTasks                       // 任务开始/结束（含结果与错误）
    StreamCheckpoints                 // 检查点事件
    StreamDebug                       // checkpoints + tasks
    StreamCustom                      // 节点内主动写入
)
```

**落地方式**：在 `pkg/hno/run` 的事件体系上扩展，**保留现有 `run_content`/`run_completed`**（向后兼容）：

```go
// run.Events 现有实现（agent.go:70）
type Events []BaseRunOutputEvent   // 只有 2 种事件 → 需扩展

// v3 扩展为
type Event interface {
    EventType() string
    Timestamp() time.Time
}
// 新增：NodeStartedEvent / NodeCompletedEvent / TaskErrorEvent /
//       CheckpointEvent / StateUpdateEvent / CustomEvent
```

**迁移影响评估（已核实）**：`RunStreamResult` 在 `pkg/`、`cmd/`、`examples/` 中**无外部消费方**（仅 `agent` 包内自用），因此改签名的爆炸半径可控。`output.Events` 的消费方是 `team/inheritance.go`、`workflow/run.go`、`workflow/step.go` —— 扩展新事件类型对它们是**增量兼容**（旧类型不变）。

---

## 6. G5 动态扇出 Send

```go
// Send 是节点在运行时决定的一次扇出派发
type Send struct {
    Node string   // 目标节点
    In   any      // 该次派发独有的输入
}

// 支持 Send 的节点
type Sender interface {
    Run(ctx context.Context, in any) (out any, sends []Send, err error)
}
```

**用法（map-reduce 模式）**：

```go
g := graph.New().
    AddNode(graph.NodeFunc("fanout", func(ctx context.Context, in any) (any, error) {
        var items []string = // ...
        sends := make([]graph.Send, 0, len(items))
        for _, it := range items {
            sends = append(sends, graph.Send{Node: "summarize", In: it})
        }
        return nil, sends, nil
    })).
    AddNode(graph.NodeFunc("summarize", /* ... */)).
    AddJoin([]string{"summarize"}, "reduce").   // reduce 聚合全部 summarize 输出
    SetEntry("fanout").
    SetOutput("reduce")
```

**与静态扇出的区别**：静态 `AddParallel(from, tos...)` 目标编译期定死；`Send` 的**数量和参数运行时决定**。这是 adk 静态边做不到的。

---

## 7. G6 Durability + StepLimit

```go
type Durability int
const (
    DurabilitySync  Durability = iota // 下一步开始前同步落盘
    DurabilityAsync                    // 下一步执行时异步落盘
    DurabilityExit                     // 仅图退出时落盘
)
```

v1 语义：控制 `graph` 在每个节点完成后向 `Session` 存储提交事件的时机。默认 `DurabilitySync`（最安全），低延迟场景可降级。

**StepLimit**：默认 1000，超限返回 `ErrStepLimitExceeded`。**这是图有环场景的必要安全阀**（我们现状：`workflow.Loop` 条件写反会死循环到 ctx 超时）。

---

## 8. G8 Store 长期记忆

> ⚠️ **此项引入全新概念，是 v3 中范围最大、破坏性最强的一块。建议单独决策（见 §12 决策点 D2）。**

```go
package store

// Item 是存储的键值对，带元数据与时间戳
type Item struct {
    Key       string
    Namespace []string       // 层级命名空间，如 ("users","profiles")
    Value     []byte
    CreatedAt time.Time
    UpdatedAt time.Time
}

// Store 提供跨会话/跨线程的长期记忆
type Store interface {
    Get(ctx context.Context, ns []string, key string) (*Item, error)
    Put(ctx context.Context, item *Item) error
    Delete(ctx context.Context, ns []string, key string) error
    List(ctx context.Context, ns []string) ([]*Item, error)
    Search(ctx context.Context, ns []string, query string, k int) ([]*Item, error) // 需 Embedder
}
```

**与现有概念分工**（避免重叠）：

| 现有 | 职责 | 与 Store 的边界 |
|---|---|---|
| `pkg/hno/memory` | 对话历史（进程内，agent 私有） | Store 跨会话、持久、可共享 |
| `pkg/hno/knowledge` + `vectordb` | RAG 文档库 | knowledge 是**非结构化文档**；Store 是**结构化用户事实**（偏好、画像） |
| `session.State` | 会话级 KV | Store 是跨会话长期 |

**后端**：v1 提供内存实现 + 复用现有 `session/db` 的 5 种后端之一（建议 Postgres）。

---

## 9. G7 会话事件化 + HITL

### 9.1 最小事件模型（**不抄 adk 完整 EventActions**）

```go
package session

type EventKind int
const (
    EventInput      EventKind = iota // 用户输入
    EventModelCall                    // 模型请求
    EventModelResponse                // 模型响应
    EventToolCall                     // 工具调用
    EventToolResult                   // 工具结果
    EventInterrupt                    // HITL 中断请求
    EventInterruptResponse            // HITL 恢复响应
)

type Event struct {
    ID           string
    Kind         EventKind
    InvocationID string
    Author       string
    Branch       string
    Timestamp    time.Time
    Payload      any
}
```

**与 adk 的差异（刻意的）**：不含 `ArtifactDelta` / `TransferToAgent` / `Escalate` / `Compaction` / `RequestedToolConfirmations` —— 那些是 adk 云场景的产物。

**向后兼容**：`RunOutput` 保留为**派生视图**，由事件流投影生成。现有 `Session.Runs` 不变。

### 9.2 HITL

```go
// Interrupt 挂在事件上，持久化
type Interrupt struct {
    InterruptID    string
    Message        string
    ResponseSchema map[string]any  // JSON Schema
    Payload        any
}

// Resume 恢复（借鉴 adk resume.go 的幂等语义）
func (g *Graph) Resume(ctx context.Context, responses map[string]any) (*Result, error)
```

**必须实现的三个语义**（借鉴 adk，adk 花了 980 LOC 在这上面）：
1. **schema 校验**：响应不匹配 `ResponseSchema` → 返回错误，节点保持 Waiting，调用方可修正后重试
2. **幂等**：`resolvedCount == 1` 才是首次恢复；重复 Resume 同一 `InterruptID` 是 no-op（返回 `ErrNothingToResume`）
3. **两种恢复模式**：`Rerun`（重入，节点拿到响应重跑）vs `Handoff`（交接，响应作为节点输出给后继）

### 9.3 契约层影响 ⚠️

`internal/session/contract` 有 9 个 Go↔Python fixture 对齐测试。**事件化会改变 session 存储形态**。

**应对**：
- v1：**不改 `Session` 的对外 JSON 结构**，事件流作为**新增的内部存储 + 派生视图**
- 若必须改，需同步更新 `internal/session/contract` 的 fixture，并评估是否破坏 agno-python 互操作
- **这是一个需要显式拍板的决策点（§12 D1）**

---

## 10. 分阶段里程碑

| 阶段 | 内容 | 前置 | 可交付物 | 验收 |
|---|---|---|---|---|
| **P0** | G1 循环收口 | — | agent 全部走 `runner`；删除 run.go/stream.go 内的循环 | `runner` 被引用；`grep -c "for {" agent/` ≤ 1；`make test` 绿 |
| **P1** | G2 图引擎 v1 | P0 | `pkg/hno/graph` | 5 类图（DAG/并行/汇聚/条件/环）端到端测试；`-race` 全绿 |
| **P2** | G3 策略四合一 | P1 | Retry/Cache/Timeout/Trace | 每个策略独立测试 + 组合测试 |
| **P3** | G5 Send + G6 Durability/StepLimit | P1 | 动态扇出 + 安全阀 | map-reduce 端到端；环死循环被 StepLimit 拦住 |
| **P4** | G4 StreamMode | P0 | 七模式 + 旧类型兼容 | 旧 `run_content`/`run_completed` 行为不变 |
| **P5** | G7 事件化 + HITL | P0 + 决策 D1 | 事件存储 + Resume | 审批场景跨进程恢复；重复 Resume 幂等；**契约测试绿** |
| **P6** | G10 workflow 迁移 | P1–P4 | 线性 []Step 编译为链式图 | **现有用户零改动**；`make test` 绿 |
| **P7** | G9 观测接线 | P0 | retry/breaker 接进 runner | span 覆盖 run/agent/model/tool |
| **P8** | G8 Store | 决策 D2 | 长期记忆 | 与 knowledge/vectordb 分工清晰 |

**关键路径**：P0 → P1 → (P2/P3/P4 并行) → P6

**P0 必须最先做**：它是红线 2 的兑现，且不做的话 P1 之后所有 agent 侧改动都在两套循环上打补丁。

---

## 11. 破坏性变更清单

| # | 变更 | 影响 | 迁移 |
|---|---|---|---|
| BC-1 | `Agent.Run` 内部改走 `runner` | 行为需逐项对齐（cache/hitl/hooks 交织） | 对外签名不变；补齐行为等价测试 |
| BC-2 | `RunStream` 返回值变更（G4） | 已核实：`pkg/`+`cmd/`+`examples/` 无外部消费方 | 新增 `RunStreamMode`，保留 `RunStream` 包装为 `StreamMessages` |
| BC-3 | `pkg/hno/workflow` 内部实现换引擎 | 性能特征变化 | **公共 API 不变**；`Node.Execute(ctx, *ExecutionContext)` 签名保留 |
| BC-4 | `session.Session` 新增事件流 | 若改 JSON 结构则影响契约层 | 见决策 D1；v1 默认不改对外结构 |
| BC-5 | `graph` 包引入 `any` 契约 | 与红线 4 张力 | 提供 `graph.Typed[TIn,TOut]` 泛型包装；schema 校验留 v3.1 |

---

## 12. 待决策点

| # | 决策 | 选项 | 我的倾向 |
|---|---|---|---|
| **D1** | 会话事件化是否改动 `Session` 对外 JSON？ | (a) 不改，事件为内部存储+派生视图（安全，但契约层无感）<br>(b) 改，同步更新 Python fixture（暴露能力，但可能破坏互操作） | **(a) 先做 a**，b 作为独立任务 |
| **D2** | G8 Store 是否进 v3.0？ | (a) 进（+1,200 LOC，全新概念，范围最大）<br>(b) 延到 v3.1 | **(a) 进** —— 「能记住用户」是产品差异化，但需先定与 knowledge 的边界 |
| **D3** | `graph.Node` 用 `any` 还是泛型？ | (a) `any` + `Typed[TIn,TOut]` 包装<br>(b) 纯泛型 | **(a)** —— 引擎需要异构节点，纯泛型会锁死组合性 |
| **D4** | P0 是否要先合入主干？ | (a) 是，单独发一个 minor<br>(b) 全部做完一起发 major | **(a)** —— 死代码与 4 份循环是持续风险，不应压到 major 发布 |

---

## 13. 风险登记

| 风险 | 等级 | 缓解 |
|---|---|---|
| 图引擎与 agent 循环职责重叠 | 高 | **明确定界**：图管「节点间控制流」，节点管「节点内循环」。节点内的 tool loop **不上图为环**，除非需要跨节点循环 |
| P0 收口时行为漂移（cache/hitl/hooks） | 高 | P0 必须补齐行为等价测试（golden cases），逐项对齐 `run.go` 现有语义 |
| 条件边成环导致死循环 | 中 | StepLimit 强制默认 1000；构建期检测「无条件边成环」直接报错 |
| 契约层被事件化破坏 | 中 | D1 决策；P5 阶段契约测试必须全绿才可合入 |
| 并发 bug 难以测试 | 中 | 参照 adk 实现:测试 ≈ 1:2.2 配比；`-race` 为 CI 必过项 |
| 图引擎做成「第二个 workflow」 | 中 | 验收标准必须包含并行/汇聚/条件路由/跨节点环——**静态线性能表达的不算 v1 达标** |

---

## 附：证据索引

本设计的关键数字均来自源码实测，非估计：

| 数字 | 来源 |
|---|---|
| adk workflow 非测试 6,244 LOC / 测试 13,800 LOC | `wc -l /tmp/fwcmp/adk-go/workflow` |
| 图引擎核心 6 文件 2,134 LOC | `graph.go`+`scheduler.go`+`base_node.go`+`branch.go`+`edgebuilder.go`+`validation.go` |
| 单消费者零锁模型 | `adk-go/workflow/scheduler.go` 注释「Concurrency model: producer-consumer over eventQueue」 |
| 路由语义 | `adk-go/workflow/scheduler.go:986 findSuccessors` |
| HITL 幂等语义 | `adk-go/workflow/resume.go`（`answeredThisTurn`/`ErrNothingToResume`） |
| 节点重试 | `adk-go/workflow/retry.go`（`CalculateDelay`/`ShouldRetry`） |
| `Send` 动态扇出 | `langgraph/types.py:732` |
| Store 长期记忆 | `langgraph/checkpoint/store/base/__init__.py` |
| Durability 三档 | `langgraph/types.py:98` |
| StreamMode 七种 | `langgraph/types.py:131` |
| 四合一节点策略 | `langgraph/types.py:427 RetryPolicy` / `:530 CachePolicy` / `:461 TimeoutPolicy` / `:542 TracePolicy` |
| recursion_limit | `langgraph/errors.py:67 GraphRecursionError` |
| 我们 4 份循环 | `agent/run.go:60`、`agent/stream.go:259`、`runner/runner.go:180`、`run/loop.go` |
| runner 死代码 | 全仓库 `grep hno/runner` 零引用 |
| 契约层 9 测试 | `internal/session/contract/contract_test.go` |
