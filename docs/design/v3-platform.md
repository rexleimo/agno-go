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

> **基线时效（2026-09-27 回写）**：上表是 v3 规划起点的快照，其中两行已随 P0 / P0b / P0-stream 落地而失效——
> `pkg/hno/runner` 不再是死代码（agent 的同步与流式两条路径都经 `agent/kernel.go` 驱动它），
> 循环实现已从 4 份降为 2 份（`runner` 内核 + team 使用的 `run.Loop`）。
> 原文保留以供追溯，后续读者请以当前代码为准，勿据这两行推断。

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
| G8 | `Store` 长期记忆（层级 namespace + 向量）**——D2 已裁 OPT-a1：收窄为最小核， vectordb 适配层不进 v3.0** | LangGraph | ~1,200 |
| G9 | 观测接线（现有零件接进 G1）**——第 1 片已交付（切片 25：runner 模型路径 retry/breaker + model 级 span）** | 自有 | ~600 |
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
    cancels map[string]context.CancelFunc // consumer-only，无锁（P1 切片 17 后修订：节点不拿派生 ctx 时不需要，见下方「取消」）
    pending []activation          // 并发打满时的等待队列
    steps   int                  // recursion_limit 计数
}
```

**不变式**：
1. **生产者**（每节点 goroutine）只做三件事：跑节点、`recover` panic 成 error、往 `queue` 发送。**从不读 queue，从不碰任何 scheduler 字段。**
2. **消费者**（调用 `Run` 的那个 goroutine）是 `queue` 的唯一读者，`runs`/`state`/`cancels`/`pending`/`steps` 的唯一写者。
3. 双方只共享 `queue`（channel 本身并发安全）→ **消费者独占字段全部不需要 mutex**。

**为什么这在 Go 里比 Python 简单**：Python 版必须用 `asyncio.Lock` / `asyncio.Queue` 保护这些状态；Go 的「单 goroutine 独占可变状态」是天然模式，**把锁的问题在结构上消除了**。

**取消**：在途节点返回 `context.Canceled`，consumer 把「调用方已经取消」归一化成**取消结论**，而不是转述节点交出的业务错误。归一化的判据不在结论文本，而在**咨询位置**：consumer 必须在每一条交出结论的路径上现问一次 `ctx` —— ① 每轮循环开头、进入 `select` 之前；② 从 `queue` 取到一项之后、判定它算不算结论之前；③ 派发循环把激活交给节点之前。②是必需的，因为 `select` 在两个 case 同时就绪时随机挑选，「调用方已 cancel」与「节点交出错误」会同时成立；③是必需的，因为消费者忙于派发循环期间不回到 `select`，那段时间里的取消只有这里看得见。在 `Run` 入口读一次快照、后续都复用那份快照，**不算咨询**。

> 本段原文写的是「`cancelAll()` 遍历 `cancels` 调 `context.CancelFunc`；…统一归一化为取消而非错误」。P1 切片 17（票面 B9）落地后按实测回写两处（证据：`docs/design/v3-p1-graph-r17-refactor.md` §7、`docs/design/v3-p1-graph-r17-review-verdict.json`、票面 §13）：
> 1. **`cancelAll()` / `cancels` 未实现，且当前形状下不需要**：节点拿到的就是调用方那一份 `ctx`（`runNode` 直接传 `s.ctx`），取消由 ctx 树自动传播到在途节点，无需按节点注册取消函数。只有当节点改拿**派生** ctx 时它才成为必需 —— 即 P2 G3 的 Timeout/Retry（每个节点自带超时子 ctx，取消才需要被聚合）。
> 2. **「统一」不适用于成功路径（`running == 0`），且这一取舍目前不带断言**：在节点阻塞期间取消的形状里量不到「取消后仍交出成功 Result」的窗口（0/300 轮）；而「极早取消」下交出成功 Result 的 26/60 轮，在公共面上与「图确实先收敛」不可区分。该路径是否也必须报取消，登记为未裁决项（`S17-SPEC-3` / `R17-UNADJ-1`）挂 review，不用「统一」二字把它写成已决。

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

**规则 3（Join 汇聚屏障）已由切片 19 落地**，判据、牙齿归属与观察上限见 `v3-test-scope-p1-graph.md` §15。
落到代码上的三条要如实记下：

- 「所有前驱 Completed」的判据是**去重前驱名集合被键覆盖**：`joinBarriers` 在进入 `Run` 时从拓扑快照算
  一次，运行中改图不会给已开始的这次执行增删屏障（与 `stepLimit`/`maxConcurrency` 同一套「取一次值」不变量）。
  「聚合前驱输出」的形状是 `map[string]any` —— 键 = 前驱名、值 = 该前驱本次输出，交给激活的就是这个 map 本身。
- 前驱因条件环重跑时键集合不再增长，于是屏障会被**再次**满足，汇聚目标可重跑；票面没有一行要求
  「每张图上每个屏障至多 fire 一次」。含 `edgeJoin` 的环不被 §3.5 第 7 项的构建期环检查拒绝
  （汇聚边按可断开处理），它落到步数安全阀。
- 「未收敛由安全阀结束」只在图仍在产生激活时成立：若声明过的汇聚前驱始终没跑且再无待派激活，
  今天交出的是 `err==nil` 而 `Output()==nil` 的 Result，该形状未裁决（`R19-Q1`）。

**红线 4 冲突的处理**：`Node.Run(in any) (any, error)` 用了 `any`，这是**引擎契约**，不是用户配置的公共 API 参数。为降低代价：

- v1 提供泛型包装 `graph.Typed[TIn, TOut]`（Go 1.24 泛型），用户可获得编译期类型安全
- schema 校验（`InputSchema`/`OutputSchema`，借鉴 adk）留 v3.1

**运行期类型不符的归因由包装层给出，调度器不参与**（切片 18 已交付，判据见 `v3-test-scope-p1-graph.md` §14）：
`Typed` 在包装期算出期望类型、在断言失败时交出带引号节点名 + 期望/实收类型的错误，`nil` 实收渲染为「无值」。
编译期那一半由泛型签名本身担保，不走运行期测试缝；调度器只透传节点错误，因此取消归一化（§3.3）
与类型归因是两条互不重叠的路径。观察上限：错误文本只担保「可定位」，不担保「报的是值的类型而非其指针形式」。

### 3.5 构建期校验（借鉴 adk `validation.go`，精简版）

`Validate()` 必查项（全部在 `Add*` 时增量检查，`Run` 前全量复查）：

1. 悬空边（`from`/`to` 指向不存在的节点）
2. 重复节点名 —— **已交付**（切片 21）：同一张图上重复注册同名节点时，`Validate()` 末位返回
   `graph: node name %q is declared more than once` 并点名那个名字，`Run` 随之拒绝、不跑半张图；
   记账在 `AddNode`（builder 无错误通道，增量检查的唯一诚实形状），map 覆盖语义保留以维持
   in-flight Run 的捕获隔离。判据、牙齿与观察上限见票面 §17。
3. 无入口 / 入口不存在
4. 不可达节点（从入口 BFS 不可达 → 报错而非静默）
5. 无出口（无出边的非输出节点）—— **裁决为不作校验项**（负责人 2026-09-27 选 OPT-A）：
   §3.4 路由核心的规则 1 允许无条件扇出的终点节点输出无人消费，条件边与兜底边也天然产生分支终点，
   因此该判据与本引擎的路由模型不相容。实测：把字面判据加到仓库副本上 12 个既有顶层 Test 判红，
   最窄的非空限定版（只判无条件边的目标）仍 10 个判红，其中 `TestP1G_ConditionalAndDefaultRouting/unconditional-fanout`
   与 `p1r13_step_limit_test.go` 的「合法的扇出图」子用例就是被误拒的那一类。要把这类意外悬空汇点变得可判，
   必须先引入显式终点声明（`graph.END` 一类），那是新的公共面表达力而非一条校验规则。
   数字、命令与三种形状见票面 §16 与 `v3-test-scope-p1-graph-slice20.json` 的 M1–M3。
6. Join 节点的前驱数 < 2 —— **已交付**（切片 19）：`Validate` 点名那个目标并给出实收的去重前驱数，
   `Run` 随之拒绝；判据形状与打红它的变异见票面 §15.1 第 3 点、§15.2 倒数第二行。
   本节第 2 项已随切片 21 交付（见上），第 5 项按上条裁决作废 —— **§3.5 八项至此全项闭合**。
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

// TraceConfig 控制节点级事件记录
type TraceConfig struct {
    Enabled   bool
    RedactIn  bool  // 脱敏输入
    RedactOut bool
    Hook      func(NodeEvent) // sink（切片 22 补出：没有 sink 的 trace 无法被任何调用方观察）
}
```

> **已交付（切片 22）**：四个策略经 `AddNode` 的变参 `NodeOption`（`WithRetry/WithTimeout/WithCache/WithTrace`）
> 挂载，执行缝在 scheduler 的 `runNode` 包络——重试与期限在生产者 goroutine 内生效（不产生新激活、
> 不消耗步数预算）、缓存查询在生产者侧、trace 事件由消费者串行发射；零锁模型未破（`sync.` 计数 0）。
> 取值语义沿 WithMaxConcurrency 先例「guard 钳制、不进 Validate」：MaxAttempts<1 即 1 次、Timeout<=0
> 即不限时、`ShouldRetry=nil` 一律不重试（fail-closed）、缓存只存成功结果。判据 D1–D14、变异矩阵与
> 观察上限见 `docs/design/v3-test-scope-p1-graph-slice22.json` 与 `docs/design/v3-p2-graph-g3-*.md`。

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

> **已交付（切片 23，协议层）**：`pkg/hno/run/modes.go`（StreamMode 七常量 + 六个新事件 wire 名 + `ErrUnsupportedStreamMode`）、`pkg/hno/run/stream_events.go`（六事件类型 + New\* 构造函数 + canonical JSON）、`decodeEvent` 精确匹配块（先于 contains 归一化，次序由 D12 钉死）、`Agent.RunStreamMode` 选择器（`pkg/hno/agent/agent.go`；`RunStream` 成为它的 `StreamMessages` 纯包装）。旧 `run_content`/`run_completed` 的 wire 形状与既有测试逐字节零改动。判据 D1–D12、两段 RED/GREEN、6 条变异矩阵见 `docs/design/v3-test-scope-p1-graph-slice23.json` 与 `docs/design/v3-red-observation-p4g4.md` / `v3-p4-run-g4-green.md` / `v3-p4-run-g4-refactor.md`。
>
> **七模式解禁状态**：
>
> | 模式 | 状态 |
> |---|---|
> | `StreamMessages` | **已接**（`runStreamMessages` 生产者，`RunStream`/`RunStreamMode` 同形） |
> | `StreamValues` | fail-closed 待生产者片（graph 节点事件桥接） |
> | `StreamUpdates` | fail-closed 待生产者片（graph 节点事件桥接） |
> | `StreamTasks` | fail-closed 待生产者片 |
> | `StreamCheckpoints` | fail-closed 待生产者片（检查点生产者） |
> | `StreamDebug` | fail-closed 待生产者片（依赖 Tasks/Checkpoints 的并集发射） |
> | `StreamCustom` | fail-closed 待生产者片（节点内自定义写入入口） |
>
> 其余六模式在 `RunStreamMode` 上返回包装 `ErrUnsupportedStreamMode` 的错误、不启动流；各自生产者接线片落地时解禁（S23-SPEC-1）。

---

## 6. G5 动态扇出 Send

**已交付（切片 26）**：`Send` + `Sender`（`SendRun` 可选扩展面）+ `SenderFunc` 适配器 + 显式汇聚声明
`AddJoinSend(source, target)`——R19-Q2 裁决 OPT-C 的落地形态。契约
`docs/design/v3-test-scope-p1-graph-slice26.json`（D1–D9）；证据 `docs/design/v3-p3-graph-g5-green.md`。

```go
// Send 是节点在运行时决定的一次扇出派发
type Send struct {
    Node string   // 该次派发的目标节点
    In   any      // 该次派发独有的输入
}

// Sender 是节点的可选扩展面。方法名刻意不是 Run：Node.Run 已占用两返回值签名，
// 一个类型不可能同时实现 2 值与 3 值的 Run——调度器对每次激活做类型断言，
// 断言成功走 SendRun（sends 被派发），失败走 Node.Run（普通节点零改变）。
type Sender interface {
    SendRun(ctx context.Context, in any) (out any, sends []Send, err error)
}

// SenderFunc 用函数构造一个支持运行期扇出的 Node
func SenderFunc(name string, fn func(ctx context.Context, in any) (any, []Send, error)) Node
```

**用法（map-reduce 模式，可跑通形态）**：

```go
g := graph.New().
    AddNode(graph.SenderFunc("fanout", func(ctx context.Context, in any) (any, []graph.Send, error) {
        items := in.([]string) // 数量与参数都在运行期决定
        sends := make([]graph.Send, 0, len(items))
        for _, it := range items {
            sends = append(sends, graph.Send{Node: "summarize", In: it})
        }
        return nil, sends, nil
    })).
    AddNode(graph.NodeFunc("summarize", func(ctx context.Context, in any) (any, error) {
        return summarizeOne(in.(string)), nil // 每片独立激活，输入 = 该次 Send.In
    })).
    AddNode(graph.NodeFunc("reduce", func(ctx context.Context, in any) (any, error) {
        // 聚合输入 map[string][]any：键 = 声明的 source 名，值 = 按派发序的各次 summarize 输出
        outs := in.(map[string][]any)["summarize"]
        return mergeAll(outs), nil
    })).
    AddJoinSend("summarize", "reduce"). // 等 summarize 名下本轮全部 Send 激活完成后，以聚合输入激活 reduce 一次
    SetEntry("fanout").
    SetOutput("reduce")
```

**与静态扇出的区别**：静态 `AddParallel(from, tos...)` 目标编译期定死；`Send` 的**数量和参数运行时决定**。这是 adk 静态边做不到的。

**语义（切片 26 契约 designConsequence 五条钉死）**：`AddJoinSend(source, target)` 构建期不数前驱名数、
不经 §3.5 第 6 项、不与 `AddJoin` 判据互通（单前驱 `AddJoin` 照旧被拒），只查端点存在并贡献
source→target 一条可达性边（source 本身被视为可达——Send 的目标名是运行期值，构建期无法反驳
「会有 Sender 指名它」）；运行期屏障按「source 名下 pending Send 计数归零」判凑齐，归零时以聚合输入
激活 target 一次，多波扇出（source 因环重跑再派发）逐波独立凑齐；零派发 → target 不激活，Run 交
err==nil 而 Output()==nil（R19-Q1 同族，联动登记）；Send 指向未注册节点 → 运行期点名错误（fail-closed，
不静默丢）。原「示例跑不通、与 §3.5 第 6 项相冲（R19-Q2）」的注记已由裁决（OPT-C）+ 本片落地关闭：
材料与三选项存档于 `docs/design/v3-adjudication-r19-q2-send-join.md`；判据原文见
`v3-test-scope-p1-graph.md` §15.5。

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

> ✅ **Durability 三档已交付（切片 24，契约 `v3-test-scope-p1-graph-slice24.json` D1–D9）**。缝形状（契约 designConsequence 的落点）：`Checkpointer` 接口（`Append(ctx, Checkpoint) error`）与 `Checkpoint` 条目（Seq/Node/Output 最小集）**声明在 `graph` 包内**（沿 CacheStore 先例，不依赖 internal/session，存储适配留给调用方）；**Seq 由消费者 goroutine 在提交时机盖章**，从 1 连续递增，条目流 append-only；**三档时机全在消费者侧**——Sync 在 complete 之后、下一轮 dispatch 之前同步落（默认档，零值即 Sync），Async 交后台冲刷 goroutine（与引擎只共享 commits 通道，**零锁保持**）、Run 返回前冲刷完毕，Exit 只积累、退出一次落齐；sink 失败 **fail-closed**（`errors.Is` 可归因到 sink 交的错误，失败路径上 runErr 优先不被掩盖）；已完成节点的条目在失败/撞阀退出时照常交付；声明档位而未挂 sink = 不持久化、不报错，零新增 Validate 行。**恢复/重放（读回续跑）未交付，按 §7 门槛挂至 P5/G7**（S24-SPEC-1）。实现 `pkg/hno/graph/durability.go`；证据 `docs/design/v3-p3-graph-g6-green.md`（变异矩阵 `scripts/mutation/p3g6-durability.mjs`）。

**StepLimit**：默认 1000，超限返回 `ErrStepLimitExceeded`。**这是图有环场景的必要安全阀**（我们现状：`workflow.Loop` 条件写反会死循环到 ctx 超时）。**已随切片 13 交付**（`defaultStepLimit=1000`、`WithStepLimit` 构建期校验、`p1r13_step_limit_test.go` 8 个 Test 锚定）。

---

## 8. G8 Store 长期记忆

> ⚠️ ~~此项引入全新概念，是 v3 中范围最大、破坏性最强的一块。建议单独决策（见 §12 决策点 D2）。~~
> **D2 已裁决（2026-09-28，OPT-a1 最小核）**，下方草图随之收窄。警示按实测修正：Store 今天在全仓库
> **零消费方**，故爆炸半径为零，「破坏性最强」不成立（`v3-adjudication-d2-store.md` §1.3/§3）；
> 与 `knowledge` 重叠实测为零（该包只有 Loader/Chunker 摄取面）、与 `memory.Memory` 结构性不可互换、
> `embeddings` 三个 provider 已是现成的 `vectordb.EmbeddingFunction` 注入件（草图「需 Embedder」不自建抽象）。
> 契约：`docs/design/v3-test-scope-p8-g8-store.json`（切片 29）。
> ✅ **D2 的六项形状取舍已于 2026-09-28 一次性放行（包 A,A,A,A,B,B，负责人）**：
> 落位 `pkg/hno/store` 且 `internal/session/store` **不改名**（同名异义作为已接受代价，internal 不可 import）；
> namespace **精确匹配**（段规则：非空 + UTF-8 + 禁 U+001F + 段长 ≤128/深度 ≤16），前缀检索留 v3.1；
> `List` 签名不动、**全量不截断**（任何实现不得静默截断，分页 v3.1 以末位 variadic 新增）；
> `Value []byte` **允许任意字节**，检索只对合法 UTF-8 打分；
> **草图下面的接口由此扩为七方法**——加 `SearchScored(ctx, ns, query, k) ([]SearchHit, error)`（分数必须回传，
> 否则调用方看不见上一条的非文本剔除、v3.1 上 pgvector 也无距离回传通道）与
> `PutMany(ctx, items []*Item) error`（一次 `Embed` 整批：逐条嵌入等于导入 5,000 条 = 5,000 次真 provider 调用；
> 派生的「批量部分失败」语义按 D5 同族裁为**整批原子**）。下方五方法草图作为裁决原文保留，扩面已登记在契约 D1。

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
    Mode           InterruptMode   // ResumeRerun（默认）| ResumeHandoff —— 引擎调度语义，切片 27 补出的第五字段
}

// Resume 恢复（借鉴 adk resume.go 的幂等语义）
func (g *Graph) Resume(ctx context.Context, responses map[string]any) (*Result, error)
```

**必须实现的三个语义**（借鉴 adk，adk 花了 980 LOC 在这上面）：
1. **schema 校验**：响应不匹配 `ResponseSchema` → 返回错误，节点保持 Waiting，调用方可修正后重试
2. **幂等**：`resolvedCount == 1` 才是首次恢复；重复 Resume 同一 `InterruptID` 是 no-op（返回 `ErrNothingToResume`）
3. **两种恢复模式**：`Rerun`（重入，节点拿到响应重跑）vs `Handoff`（交接，响应作为节点输出给后继）

> ✅ **引擎核心已交付（切片 27，2026-09-28，契约 `v3-test-scope-p1-graph-slice27.json`）**。三语义落点：
> （1）**schema 校验**——`Resume` 逐中断校验 `ResponseSchema`（诚实子集：`type=object` + `required` +
> `properties.<name>.type`；子集之外的声明报不支持而非静默放过），不过 → 包 `ErrInvalidResponse`
> 且挂起原样保留，节点保持 Waiting 可修正重试；（2）**幂等**——挂起账由 `Graph` 独占（`pending`），
> 无处可恢复/重复 Resume → `ErrNothingToResume`；响应集必须**恰好覆盖**全部待答中断（多给/少给都
> 点名 InterruptID，不静默取交集）；（3）**Rerun/Handoff**——`Interrupt.Mode`（草图四字段外补出的
> 字段）各归其档：Rerun 带响应重入（响应经 `InterruptResponse(ctx, id)` 公共取值器可见）、Handoff
> 节点体不再执行、响应直接成为其输出喂后继。挂起形状：`Run`/`Resume` 交 `(nil, *Suspension)`（可
> `errors.Is(ErrSuspended)` / `errors.As` 取全部待答中断与已完成状态），无半截 Result；重试不吞中断
> （`RequestInterrupt` 是不可重试终态）；空/并发重复 InterruptID 运行期点名；挂起经切片 24
> Checkpointer 以 `EntryInterrupt` 条目持久化（sink 失败与挂起经 `errors.Join` 双可达）；步数预算跨
> 恢复累计。**切片拆分登记**：第 1 片 = 引擎内 HITL 核心 + 挂起检查点条目（本片）；第 2 片 = 会话
> 侧车存储 + 派生视图 + 跨进程恢复接线（依 D1 裁决 OPT-1，契约另立）；第 3 片（可选）= run 事件流
> 与 CheckpointEvent 生产者桥接（S23-SPEC-1 交点）。实现 `pkg/hno/graph/hitl.go`；证据
> `docs/design/v3-p5-graph-g7-green.md`（变异矩阵 `scripts/mutation/p5g7-hitl.mjs`）。

### 9.3 契约层影响 ⚠️

`internal/session/contract` 有 9 个 Go↔Python fixture 对齐测试。**事件化会改变 session 存储形态**。

**应对**：
- v1：**不改 `Session` 的对外 JSON 结构**，事件流作为**新增的内部存储 + 派生视图**
- 若必须改，需同步更新 `internal/session/contract` 的 fixture，并评估是否破坏 agno-python 互操作
- **已拍板（2026-09-28，负责人选 OPT-1/v1）**：不改 `Session` 对外 JSON；事件流为侧车内部存储 +
  派生视图，实现细节（侧车挂在哪个 Store、派生视图形态）由 G7 切片契约定。材料与实测
  （含 fixtures 实为 skip-green 的发现）见 `docs/design/v3-adjudication-d1-session-events.md`。
- **第 2 片已交付（切片 28）**：侧车落位 `pkg/hno/session/sidecar`（旁路容器，四个 marshal 边界
  8 锚逐字节不变）+ `internal/hitlbridge`（Capture/Install/ResumeSaved，事件 fail-open、
  挂起记录 fail-closed）+ 引擎条件性恢复入口 `graph/restore.go`（ResumeAccount/RestorePending）。
  P5 验收「审批场景跨进程恢复」实测达成（双档端到端）。

---

## 10. 分阶段里程碑

| 阶段 | 内容 | 前置 | 可交付物 | 验收 |
|---|---|---|---|---|
| **P0** | G1 循环收口 | — | agent 全部走 `runner`；删除 run.go/stream.go 内的循环 | `runner` 被引用；`grep -c "for {" agent/` ≤ 1；`make test` 绿 |
| **P1** | G2 图引擎 v1 | P0 | `pkg/hno/graph` | 5 类图（DAG/并行/汇聚/条件/环）端到端测试；`-race` 全绿 |
| **P2** | G3 策略四合一 | P1 | ~~Retry/Cache/Timeout/Trace~~ **已交付（切片 22）** | 每个策略独立测试 + 组合测试（D1–D14 + 9 条变异矩阵） |
| **P3** | G5 Send + G6 Durability/StepLimit | P1 | 动态扇出 + 安全阀 | **全清**：**G5 Send 已交付（切片 26：动态扇出 Send + AddJoinSend，R19-Q2 裁决 OPT-C 落地，map-reduce 端到端 D1–D9 + 6 杀红变异）**；**G6 Durability 已交付（切片 24）**；StepLimit 已随切片 13 交付 |
| **P4** | G4 StreamMode | P0 | 七模式 + 旧类型兼容 **协议层已交付（切片 23）**；六模式生产者接线待各自成片 | 旧 `run_content`/`run_completed` 行为不变 |
| **P5** | G7 事件化 + HITL | ~~P0 + 决策 D1~~ **全清（切片 27 引擎核心 + 28 侧车/跨进程恢复）** | 事件存储 + Resume **全部交付（27：引擎核心；28：`pkg/hno/session/sidecar` + `internal/hitlbridge` + `graph/restore.go`，D1–D10 + 10 杀红变异）** | 审批场景跨进程恢复**实测达成（双档端到端）**；重复 Resume 幂等**实测达成（跨重启）**；**契约测试绿（8 边界锚逐字节）** |
| **P6** | G10 workflow 迁移 | P1–P4 | 线性 []Step 编译为链式图 | **现有用户零改动**；`make test` 绿 |
| **P7** | G9 观测接线 | P0 | retry/breaker 接进 runner **模型阶段已交付（切片 25，D1–D13 + 9 杀红变异矩阵）**；run/agent 级 span 待第 2 片（S25-DEFER-1） | span 覆盖状态表：**model=已接于 runner**（每尝试 chat span + usage 归集）；**tool=已在 agent 既有**（execute_tool）；**run/agent=待 P7 第 2 片**（pkg/hno/agent） |
| **P8** | G8 Store | ~~决策 D2~~ **已裁决（OPT-a1，2026-09-28）** | 长期记忆最小核：接口 + namespace + 内存与单一 Postgres 后端（契约 `v3-test-scope-p8-g8-store.json` 切片 29 已起草，D1–D11） | 与 knowledge/vectordb 分工清晰（契约以导出白名单验收）；ADJ-1…ADJ-6 待落裁后开工 |

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
| **D1** | 会话事件化是否改动 `Session` 对外 JSON？ | (a) 不改，事件为内部存储+派生视图（安全，但契约层无感）<br>(b) 改，同步更新 Python fixture（暴露能力，但可能破坏互操作） | **已裁决（2026-09-28）：(a)**，实测材料见 `v3-adjudication-d1-session-events.md` |
| **D2** | G8 Store 是否进 v3.0？ | (a) 进（+1,200 LOC，全新概念，范围最大）<br>(b) 延到 v3.1 | **已裁决（2026-09-28）：(a) 进，收窄为 OPT-a1 最小核**（接口 + namespace + `ErrNotFound` + 注入 `vectordb.EmbeddingFunction` + 内存后端 + 单一 Postgres 后端，≈1,000–1,300 行；**vectordb/chromadb/redisdb 适配层不进 v3.0**）。「破坏性最强」经实测修正为不成立（Store 今天零消费方），真实风险在边界文案。实测材料 `v3-adjudication-d2-store.md`；契约 `v3-test-scope-p8-g8-store.json`（切片 29，D1–D11 + 14 条 explicitNonGoals），其 ADJ-1…ADJ-6 六项形状取舍待负责人落裁 |
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
