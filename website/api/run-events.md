# Run Events & Stream Modes API Reference

The `pkg/hno/run` event protocol: stream-mode selection, the six graph-era
event types, and their canonical JSON wire form. See
[Graph Durability](/advanced/graph-durability) for the engine-side checkpoint
seam this protocol will eventually surface.

Import:

```go
import "github.com/rexleimo/agno-go/pkg/hno/run"
```

## StreamMode

**Signature:**
```go
type StreamMode int

const (
    StreamValues      StreamMode = iota // 0: full state after each step
    StreamUpdates                       // 1: per-node increments
    StreamMessages                      // 2: token-level LLM messages
    StreamTasks                         // 3: task start/end with results and errors
    StreamCheckpoints                   // 4: checkpoint events
    StreamDebug                         // 5: checkpoints + tasks union
    StreamCustom                        // 6: node-emitted custom payloads
)
```

A `StreamMode` selects **which family of events** a streaming run emits.
Ordinals are pinned by tests; treat them as stable identifiers, not as
rankings.

**Honest status — fail-closed, not pretend-wired:** today only
`StreamMessages` has a wired producer (token-level content events, the
behavior behind `RunStream`). Selecting any other mode (alone or mixed with
`StreamMessages`) makes `Agent.RunStreamMode` return an error wrapping
`run.ErrUnsupportedStreamMode` **before any stream starts** — it never
silently degrades into an empty stream. The remaining modes are protocol
placeholders: their constants, event types, and wire names are landed; their
producers are pending future slices.

## ErrUnsupportedStreamMode

**Signature:**
```go
var ErrUnsupportedStreamMode = errors.New("run: stream mode has no wired producer")
```

Match with `errors.Is`:

```go
_, err := agent.RunStreamMode(ctx, "hello", run.StreamTasks)
if errors.Is(err, run.ErrUnsupportedStreamMode) {
    // this mode's producer is not wired yet — fail-closed by contract
}
// agent: stream mode 3 has no wired producer: run: stream mode has no wired producer
```

## Agent.RunStreamMode

**Signature:**
```go
func (a *Agent) RunStreamMode(ctx context.Context, input string, modes ...run.StreamMode) (*RunStreamResult, error)
```

Three-state semantics:

| Modes given | Behavior |
|---|---|
| none | Defaults to `run.StreamMessages` — identical to `RunStream`. |
| `StreamMessages` (alone) | Produces exactly the `RunStream` event sequence. |
| any other mode (including mixed with `StreamMessages`) | Fails closed with `ErrUnsupportedStreamMode` before starting a stream. |

The returned `RunStreamResult` carries two channels:

```go
type RunStreamResult struct {
    Events <-chan run.BaseRunOutputEvent // incremental content events
    Done   <-chan RunStreamDone          // terminal Output/Err/StopReason
}
```

### Relationship to RunStream

`Agent.RunStream` is now a **pure wrapper**:

```go
func (a *Agent) RunStream(ctx context.Context, input string) (*RunStreamResult, error) {
    return a.RunStreamMode(ctx, input, run.StreamMessages)
}
```

Same signature, same event sequence — mode selection (and the fail-closed
verdict for unwired modes) lives in `RunStreamMode`. Nothing about existing
`RunStream` code needs to change.

## The Six Event Types

All six share the same envelope: an `Event` (or legacy `event_type`) string
carrying the wire name and a `created_at` Unix-seconds timestamp. Constructors
stamp the type and UTC timestamp at build time; serialization re-canonicalizes
the type as a second layer of defense, so even a zero-value literal marshals
with its correct wire name.

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

Wire name: `node_started`.

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

Wire name: `node_completed`.

### TaskErrorEvent

```go
type TaskErrorEvent struct {
    RunID   string `json:"run_id,omitempty"`
    Node    string `json:"node,omitempty"`
    Message string `json:"message,omitempty"`
}

func NewTaskErrorEvent(runID, node, message string) *TaskErrorEvent
```

Wire name: `task_error`.

### CheckpointEvent

```go
type CheckpointEvent struct {
    RunID   string `json:"run_id,omitempty"`
    Label   string `json:"label,omitempty"`
    Payload any    `json:"payload,omitempty"`
}

func NewCheckpointEvent(runID, label string, payload any) *CheckpointEvent
```

Wire name: `checkpoint`. The payload's interpretation belongs to the producer
(the engine-side `graph.Checkpoint` seam is the intended source; the bridge is
not wired yet).

### StateUpdateEvent

```go
type StateUpdateEvent struct {
    RunID string `json:"run_id,omitempty"`
    Node  string `json:"node,omitempty"`
    Patch any    `json:"patch,omitempty"`
}

func NewStateUpdateEvent(runID, node string, patch any) *StateUpdateEvent
```

Wire name: `state_update`. Intended family for `StreamUpdates`.

### CustomEvent

```go
type CustomEvent struct {
    RunID   string `json:"run_id,omitempty"`
    Subtype string `json:"subtype,omitempty"`
    Data    any    `json:"data,omitempty"`
}

func NewCustomEvent(runID, subtype string, data any) *CustomEvent
```

Wire name: `custom`. Intended family for `StreamCustom` — payloads a node
writes on its own initiative.

## Wire Names

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

Example envelope:

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

## Events, BaseRunOutputEvent, and Legacy Compatibility

```go
type Events []BaseRunOutputEvent

type BaseRunOutputEvent interface {
    EventType() string
    Timestamp() time.Time
}
```

`Events` marshals and unmarshals heterogeneous event arrays. Decoding rules:

- **Exact wire-name matches are evaluated before the legacy fuzzy
  normalization.** This ordering is deliberate and pinned: `node_completed`
  contains the substring `completed`, and if the legacy matcher ran first it
  would swallow it into a `RunCompletedEvent`.
- **Legacy events are unchanged.** `run_content`
  (`EventTypeRunContent` / `RunContentEvent` /
  `NewRunContentEvent(runID, agentID, role, content, sequence)`) and
  `run_completed` (`EventTypeRunCompleted` / `RunCompletedEvent` /
  `NewRunCompletedEvent(...)`) keep their wire form, their tolerant matching
  (case-insensitive, whitespace-trimmed, substring-based), and their team
  variants (`NewTeamRunContentEvent`). Existing persisted history round-trips
  exactly as before.
- **Unknown kinds fall back** to `GenericRunEvent`, which preserves the raw
  payload while implementing the base interface.

One JSON round-trip caveat inherent to the payload types: `any`-typed fields
(`Output`, `Data`, `Patch`, ...) that hold numbers decode as `float64` —
compare against `float64(42)`, not `int(42)`.

## Example

```go
res, err := myAgent.RunStreamMode(ctx, "explain the graph engine")
if err != nil {
    log.Fatal(err) // only stream-start errors land here
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

## Related Pages

- [Graph API Reference](/api/graph)
- [Graph Engine guide](/guide/graph-engine)
- [Agent API](/api/agent) for `RunStream` and the agent surface
