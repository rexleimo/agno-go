# Run Events 与 Stream Modes API 参考

`pkg/hno/run` 的事件协议：流模式选择、六种图时代事件类型及其规范 JSON wire 形
状。引擎侧的检查点缝见[图持久化](/zh/advanced/graph-durability)。

导入 / Import:

```go
import "github.com/rexleimo/agno-go/pkg/hno/run"
```

## StreamMode

**签名 / Signature:**
```go
type StreamMode int

const (
    StreamValues      StreamMode = iota // 0: 每步后的完整状态
    StreamUpdates                       // 1: 各节点增量
    StreamMessages                      // 2: token 级 LLM 消息
    StreamTasks                         // 3: 任务开始/结束（含结果与错误）
    StreamCheckpoints                   // 4: 检查点事件
    StreamDebug                         // 5: checkpoints + tasks 并集
    StreamCustom                        // 6: 节点内主动写入的自定义载荷
)
```

`StreamMode` 选择一次流式运行吐出**哪一族事件**。序数被测试钉死，把它们当作
稳定标识符而不是排名。

**诚实状态 —— fail-closed，不是假装接线：** 今天只有 `StreamMessages` 有已接线
的生产者（token 级内容事件，即 `RunStream` 背后的行为）。选择其余任何模式（单
独或与 `StreamMessages` 混入）会让 `Agent.RunStreamMode` 在**流启动之前**返回包
装 `run.ErrUnsupportedStreamMode` 的错误 —— 绝不静默降级为零事件流。其余模式
是协议占位：常量、事件类型与 wire 名称已落地；生产者待后续切片接线。

## ErrUnsupportedStreamMode

**签名 / Signature:**
```go
var ErrUnsupportedStreamMode = errors.New("run: stream mode has no wired producer")
```

用 `errors.Is` 匹配：

```go
_, err := agent.RunStreamMode(ctx, "hello", run.StreamTasks)
if errors.Is(err, run.ErrUnsupportedStreamMode) {
    // 该模式的生产者尚未接线 —— 契约上 fail-closed
}
// agent: stream mode 3 has no wired producer: run: stream mode has no wired producer
```

## Agent.RunStreamMode

**签名 / Signature:**
```go
func (a *Agent) RunStreamMode(ctx context.Context, input string, modes ...run.StreamMode) (*RunStreamResult, error)
```

三态语义 / three-state semantics:

| 给定的模式 | 行为 |
|---|---|
| 不给 | 默认 `run.StreamMessages` —— 与 `RunStream` 完全一致。 |
| 仅 `StreamMessages` | 产出与 `RunStream` 完全相同的事件序列。 |
| 其余任一模式（含与 Messages 混入） | 启动流之前 fail-closed 返回 `ErrUnsupportedStreamMode`。 |

返回的 `RunStreamResult` 携带两个 channel：

```go
type RunStreamResult struct {
    Events <-chan run.BaseRunOutputEvent // 增量内容事件
    Done   <-chan RunStreamDone          // 终态 Output/Err/StopReason
}
```

### 与 RunStream 的关系

`Agent.RunStream` 现在是**纯包装**：

```go
func (a *Agent) RunStream(ctx context.Context, input string) (*RunStreamResult, error) {
    return a.RunStreamMode(ctx, input, run.StreamMessages)
}
```

签名不变、事件序列不变 —— 模式选择（含未接线模式的 fail-closed 判定）住在
`RunStreamMode`。既有的 `RunStream` 调用代码一行都不用改。

## 六种事件类型

六者共用同一信封：携带 wire 名称的 `Event`（或旧字段 `event_type`）字符串 +
`created_at` Unix 秒时间戳。构造函数在构造时盖章种类与 UTC 时间戳；序列化侧对
种类做 canonical 兜底 —— 即使零值字面量也以正确的 wire 名称序列化（两层防御）。

### NodeStartedEvent

```go
type NodeStartedEvent struct {
    RunID   string `json:"run_id,omitempty"`
    Node    string `json:"node,omitempty"`
    Attempt int    `json:"attempt,omitempty"`
    Input   any    `json:"input,omitempty"`
}

func NewNodeStartedEvent(runID, node string, attempt int, input any) *NodeStartedEvent
```

wire 名称：`node_started`。

### NodeCompletedEvent

```go
type NodeCompletedEvent struct {
    RunID   string `json:"run_id,omitempty"`
    Node    string `json:"node,omitempty"`
    Attempt int    `json:"attempt,omitempty"`
    Output  any    `json:"output,omitempty"`
}

func NewNodeCompletedEvent(runID, node string, attempt int, output any) *NodeCompletedEvent
```

wire 名称：`node_completed`。

### TaskErrorEvent

```go
type TaskErrorEvent struct {
    RunID   string `json:"run_id,omitempty"`
    Node    string `json:"node,omitempty"`
    Message string `json:"message,omitempty"`
}

func NewTaskErrorEvent(runID, node, message string) *TaskErrorEvent
```

wire 名称：`task_error`。

### CheckpointEvent

```go
type CheckpointEvent struct {
    RunID   string `json:"run_id,omitempty"`
    Label   string `json:"label,omitempty"`
    Payload any    `json:"payload,omitempty"`
}

func NewCheckpointEvent(runID, label string, payload any) *CheckpointEvent
```

wire 名称：`checkpoint`。载荷解释属于生产者（引擎侧 `graph.Checkpoint` 缝是预期
来源；桥接尚未接线）。

### StateUpdateEvent

```go
type StateUpdateEvent struct {
    RunID string `json:"run_id,omitempty"`
    Node  string `json:"node,omitempty"`
    Patch any    `json:"patch,omitempty"`
}

func NewStateUpdateEvent(runID, node string, patch any) *StateUpdateEvent
```

wire 名称：`state_update`。`StreamUpdates` 的预期事件族。

### CustomEvent

```go
type CustomEvent struct {
    RunID   string `json:"run_id,omitempty"`
    Subtype string `json:"subtype,omitempty"`
    Data    any    `json:"data,omitempty"`
}

func NewCustomEvent(runID, subtype string, data any) *CustomEvent
```

wire 名称：`custom`。`StreamCustom` 的预期事件族 —— 节点主动写入的载荷。

## Wire 名称

```go
const (
    EventTypeNodeStarted   = "node_started"
    EventTypeNodeCompleted = "node_completed"
    EventTypeTaskError     = "task_error"
    EventTypeCheckpoint    = "checkpoint"
    EventTypeStateUpdate   = "state_update"
    EventTypeCustom        = "custom"
)
```

信封示例：

```json
{
  "event": "node_completed",
  "created_at": 1790000000,
  "run_id": "run-42",
  "node": "summarize",
  "attempt": 1,
  "output": "summary text"
}
```

## Events、BaseRunOutputEvent 与旧兼容

```go
type Events []BaseRunOutputEvent

type BaseRunOutputEvent interface {
    EventType() string
    Timestamp() time.Time
}
```

`Events` 支持异构事件数组的序列化与反序列化。解码规则：

- **精确 wire 名匹配先于旧版模糊归一化求值。** 次序是刻意的且被测试钉死：
  `node_completed` 含子串 `completed`，若旧匹配器先跑，它会被吞进
  `RunCompletedEvent`。
- **旧事件零改动。** `run_content`（`EventTypeRunContent` / `RunContentEvent` /
  `NewRunContentEvent(runID, agentID, role, content, sequence)`）与
  `run_completed`（`EventTypeRunCompleted` / `RunCompletedEvent` /
  `NewRunCompletedEvent(...)`）保持 wire 形状、宽容匹配（大小写不敏感、去空白、
  子串）与 team 变体（`NewTeamRunContentEvent`）。既有持久化历史逐字节往返不
  变。
- **未知种类兜底**到 `GenericRunEvent`，它保留原始载荷并实现基础接口。

载荷类型固有的一个 JSON 往返注意点：`any` 类型字段（`Output`、`Data`、
`Patch`……）里的数字解码后是 `float64` —— 与 `float64(42)` 比较，而不是
`int(42)`。

## 示例

```go
res, err := myAgent.RunStreamMode(ctx, "explain the graph engine")
if err != nil {
    log.Fatal(err) // 只有流启动错误会落在这里
}
for {
    select {
    case ev, ok := <-res.Events:
        if !ok {
            res.Events = nil
            continue
        }
        fmt.Println(ev.EventType(), ev.Timestamp().UTC())
    case done := <-res.Done:
        if done.Err != nil {
            log.Fatal(done.Err)
        }
        fmt.Println("final:", done.Output.Content)
        return
    }
}
```

## 相关页面

- [Graph API 参考](/zh/api/graph)
- [图引擎指南](/zh/guide/graph-engine)
- [Agent API](/zh/api/agent)（`RunStream` 与 agent 面）
