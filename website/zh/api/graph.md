# Graph API 参考 / Graph API Reference

`pkg/hno/graph`（零锁控制流图引擎）的签名清单。概念见[图引擎指南](/zh/guide/graph-engine)，
策略语义见[节点策略](/zh/guide/node-policies)。/ Signatures for `pkg/hno/graph`,
the zero-lock control-flow graph engine.

导入 / Import:

```go
import "github.com/rexleimo/agno-go/pkg/hno/graph"
```

## graph.New

创建空图并应用调度选项。/ Create an empty graph with scheduler options applied.

**签名 / Signature:**
```go
func New(opts ...Option) *Graph
```

**示例 / Example:**
```go
g := graph.New(
    graph.WithStepLimit(500),
    graph.WithMaxConcurrency(8),
    graph.WithDurability(graph.DurabilitySync),
    graph.WithCheckpointer(sink),
)
```

未传 `WithStepLimit` 时步数预算默认 1000。graph 包不依赖任何其他 HNO 包。

## Options

### WithStepLimit

**签名 / Signature:**
```go
func WithStepLimit(n int) Option
```

限制单次 `Run` 的派发激活数。未设置默认 1000。`n` 必须为正数：`0` 或负数会被
`Validate()` 连同声明一起拒绝。

### WithMaxConcurrency

**签名 / Signature:**
```go
func WithMaxConcurrency(n int) Option
```

限制在途激活数。`0` 或负数表示不限 —— 这是守卫自身给出的语义。槽位打满时新激
活按 FIFO 排队，每完成一个派发下一个。

### WithDurability

**签名 / Signature:**
```go
func WithDurability(d Durability) Option
```

选择检查点提交档位（`DurabilitySync` / `DurabilityAsync` / `DurabilityExit`）。
进入 `Run` 时读取一次。见[图持久化](/zh/advanced/graph-durability)。

### WithCheckpointer

**签名 / Signature:**
```go
func WithCheckpointer(cp Checkpointer) Option
```

挂载持久化 sink。`nil`（默认）即不持久化、不报错。fail-closed：sink 错误绝不
让 `Run` 交出结论。

## Node、NodeFunc、Typed

**签名 / Signature:**
```go
type Node interface {
    Name() string
    Run(ctx context.Context, in any) (any, error)
}

func NodeFunc(name string, fn func(ctx context.Context, in any) (any, error)) Node

func Typed[TIn, TOut any](name string, fn func(ctx context.Context, in TIn) (TOut, error)) Node
```

`NodeFunc` 适配 `any` 类型函数。`Typed` 为节点体提供编译期类型安全；输入动态类
型与 `TIn` 不符时返回可归因错误而不是 panic：

```
graph: node "upper": input type mismatch: expected string, got int
```

`nil` 输入在该文案中渲染为 `no value`。

**示例 / Example:**
```go
n := graph.Typed[string, int]("count", func(ctx context.Context, in string) (int, error) {
    return len(in), nil
})
g.AddNode(n)
```

## Send、Sender、SenderFunc

**签名 / Signature:**
```go
type Send struct {
    Node string // 本次派发的目标节点
    In   any    // 该次派发独有的输入
}

type Sender interface {
    SendRun(ctx context.Context, in any) (out any, sends []Send, err error)
}

func SenderFunc(name string, fn func(ctx context.Context, in any) (any, []Send, error)) Node
```

`Send` 是一次运行期决定的扇出派发：派发数量与各自输入都是运行期数据。
`Sender` 是可选扩展面 —— 实现它的节点在同一次激活内、在常规输出之外派发
`sends`。方法名刻意不叫 `Run`：`Node.Run` 已占用两值签名，一个类型不可能
同时实现两种形状。调度器在生产者路径对每次激活做类型断言：断言成功走
`SendRun`、其返回的 sends 被派发；断言失败走 `Node.Run`、没有任何派发 ——
普通节点不受影响。`SenderFunc` 把函数适配成这样的节点；它的 `Run` 视图丢弃
sends，引擎对 `Sender` 节点从不走 `Run`。

**示例 / Example:**
```go
g.AddNode(graph.SenderFunc("fanout", func(ctx context.Context, in any) (any, []graph.Send, error) {
    items := in.([]string)
    sends := make([]graph.Send, 0, len(items))
    for _, it := range items {
        sends = append(sends, graph.Send{Node: "summarize", In: it})
    }
    return nil, sends, nil
}))
```

fail-closed：`Send` 指向未注册节点会让 `Run` 失败并点名，且派发前整体校验
全部目标 —— 不做部分派发：

```
graph: node "fanout" sent to unregistered node "ghost"
```

每个派发的 `Send` 都是一次普通激活、消耗一步。屏障与汇聚语义见下文
`AddJoinSend` 与[图引擎指南](/zh/guide/graph-engine)。

## Predicate

**签名 / Signature:**
```go
type Predicate func(out any) bool
```

条件边的路由判定，入参是源节点本次完成时的输出。`nil` 谓词的条件边会被
`Validate()` 拒绝。

## Graph builder 方法

所有 builder 方法返回 `*Graph` 支持链式调用，没有错误通道：结构问题由
`Validate()` 汇报。

### AddNode

**签名 / Signature:**
```go
func (g *Graph) AddNode(n Node, opts ...NodeOption) *Graph
```

注册节点，可选挂载节点策略（见下）。同名再注册会在表里替换该节点及其策略表，
但撞名被记账、`Validate()` 随后拒绝。进行中的 Run 持有拓扑快照、不受影响。

### AddEdge

**签名 / Signature:**
```go
func (g *Graph) AddEdge(from, to string) *Graph
```

无条件边：`from` 完成必然激活 `to`，输入即 `from` 的输出。

### AddConditional

**签名 / Signature:**
```go
func (g *Graph) AddConditional(from, to string, p Predicate) *Graph
```

条件边：仅当 `p(from输出)` 为 true 时激活。同一源的多条条件边互不排斥。`p`
不得为 `nil`。

### AddDefault

**签名 / Signature:**
```go
func (g *Graph) AddDefault(from, to string) *Graph
```

兜底边：仅当 `from` 本次完成时没有任何无条件边或条件边命中时激活。

### AddJoin

**签名 / Signature:**
```go
func (g *Graph) AddJoin(from []string, to string) *Graph
```

屏障声明：`to` 在 `from` 中每个去重前驱都完成后被激活一次。激活输入是以前驱名
为键的 `map[string]any`。去重前驱少于 2 个会被 `Validate()` 拒绝 —— 那种形状没
有屏障可等，应该用 `AddEdge`。

### AddJoinSend

**签名 / Signature:**
```go
func (g *Graph) AddJoinSend(source, target string) *Graph
```

为运行期扇出声明汇聚：等 `source` 名下本轮派发的全部 `Send` 激活完成后，
`target` 被激活一次，输入是聚合形式 —— 以 source 名为键的 `map[string][]any`，
值按派发序排列。构建期只查端点存在，并贡献一条 `source -> target` 可达性边
（source 本身被视为可达）；它不经 `AddJoin` 的前驱数判据 —— 两类声明各用各的
判据，`AddJoin` 的语义原样保留。运行期屏障按「`source` 名下在途 `Send` 计数
归零」判凑齐。多波各自独立：source 因环重跑再派发时屏障重新武装，target 只
收到当波的输出。本轮从未有 `Send` 指名 `source` 时，target 不被激活（由此
形成的 `Run` 收场形状见指南中的零派发说明）。同一 source 可声明多个 target、
同一 target 可声明多个 source；后者要等全部 source 的计数同时归零才激活
一次。

### SetEntry / SetOutput

**签名 / Signature:**
```go
func (g *Graph) SetEntry(name string) *Graph
func (g *Graph) SetOutput(name string) *Graph
```

`SetEntry` 指定接收 `Run` 输入的入口节点。`SetOutput` 指定其返回值成为
`Result.Output()` 的输出节点。两者必须引用从入口可达的已注册节点，否则
`Validate()` 拒绝。

## Validate

**签名 / Signature:**
```go
func (g *Graph) Validate() error
```

执行完整构建期清单：声明层检查（步数预算、入口、输出、边端点、谓词、可路由边
种类）、汇聚前驱不足、不可达节点、不可断开的环、重复节点名。文案稳定、可归因、
点名出问题的声明；各检查一次报一项（map 来源集合取字典序第一）。`Run` 执行前
自动调用。

## Warnings

**签名 / Signature:**
```go
func (g *Graph) Warnings() []error
```

非致命构建期提示，与 `Validate()` 刻意分离。只读：不改配置、不影响执行。当前唯
一一条提示在含环图未显式 `WithStepLimit` 时发出：

```
graph: cycle "a" -> "b" -> "a" has no explicit budget: WithStepLimit was not set,
so the default stepLimit=1000 is what bounds this graph
```

## Run

**签名 / Signature:**
```go
func (g *Graph) Run(ctx context.Context, in any) (*Result, error)
```

校验、快照拓扑，从入口驱动执行到收敛。运行期保证：

- 步数阀：超预算返回包装 `ErrStepLimitExceeded` 的错误；会让步数越界的激活根本
  不会被启动。
- panic 屏障：节点 panic 转为点名节点的错误
  （`graph: node "..." panicked while running: ...`）。
- 取消归一：调用方取消与节点错误同时成立时返回 `ctx.Err()`。
- Send 目标校验：指向未注册节点的 `Send` 会让 Run 失败并点名
  （`graph: node "..." sent to unregistered node "..."`），派发前整体校验。
- 快照隔离：入口之后的 builder 修改只影响未来运行。

## Result

**签名 / Signature:**
```go
type Result struct { /* 未导出字段 */ }

func (r *Result) Output() any
func (r *Result) Value(name string) (any, bool)
func (r *Result) Completed() []string
```

- `Output()` 返回输出节点的值（未产出过则 nil）。
- `Value(name)` 返回任意已完成节点的输出及是否产出过。
- `Completed()` 返回按字典序排序的已完成节点名 —— 不是完成顺序：并行分支之间
  没有确定的全局时序，不要按时间序做断言。

## ErrStepLimitExceeded

**签名 / Signature:**
```go
var ErrStepLimitExceeded = errors.New("graph: step limit exceeded")
```

步数阀包装的哨兵错误，用 `errors.Is` 匹配。

## 节点策略

通过 `AddNode` 的变参选项挂载 / Attached through `AddNode`'s variadic options:

```go
func WithRetry(c RetryConfig) NodeOption
func WithTimeout(c TimeoutConfig) NodeOption
func WithCache(c CacheConfig) NodeOption
func WithTrace(c TraceConfig) NodeOption
```

### RetryConfig

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

`MaxAttempts < 1`（或不配置重试）钳制为 1 次 —— 「不重试」即零值语义。
`ShouldRetry == nil` 从不重试。退避等待让位给 context 取消。

### TimeoutConfig

```go
type TimeoutConfig struct {
    Timeout    time.Duration
    PerAttempt bool
}
```

`Timeout <= 0` 表示不限时。`PerAttempt: false` 是覆盖全部重试的整节点预算；
`PerAttempt: true` 每次尝试一个新期限。以派生 context 送达节点。

### CacheConfig / CacheStore

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

默认键是输入 `%#v` 形式的 SHA-256 十六进制。fail-closed：只有成功结果写入；
缓存命中跳过执行。引擎不附带默认 store —— 实现 `CacheStore` 即适配缝。

### TraceConfig / NodeEvent

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

事件由消费者 goroutine 串行发射（hook 无需同步），成功与失败路径都发。
`RedactIn`/`RedactOut` 在送达前置 nil 载荷字段。

## Durability、Checkpoint、Checkpointer

**签名 / Signature:**
```go
type Durability int

const (
    DurabilitySync  Durability = iota // 默认（零值）：下一步派发前同步落
    DurabilityAsync                   // 后台冲刷，Run 返回前排空
    DurabilityExit                    // 积累，退出时一次落齐
)

type Checkpoint struct {
    Seq    int
    Node   string
    Output any
}

type Checkpointer interface {
    Append(ctx context.Context, cp Checkpoint) error
}
```

`Seq` 由消费者从 1 连续盖章，次序即完成处理次序。sink 失败 fail-closed，可经包
装错误归因（`graph: checkpoint sink: ...`）。这条缝尚不含恢复/重放。见
[图持久化](/zh/advanced/graph-durability)。

## 相关页面

- [图引擎指南](/zh/guide/graph-engine)
- [节点策略指南](/zh/guide/node-policies)
- [图持久化](/zh/advanced/graph-durability)
- [Workflow API](/zh/api/workflow)（v1 步骤编排器）
