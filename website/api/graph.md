# Graph API Reference

Signatures for `pkg/hno/graph`, the zero-lock control-flow graph engine. See
the [Graph Engine guide](/guide/graph-engine) for concepts and
[Node Policies](/guide/node-policies) for policy semantics.

Import:

```go
import "github.com/rexleimo/agno-go/pkg/hno/graph"
```

## graph.New

Create an empty graph with scheduler options applied.

**Signature:**
```go
func New(opts ...Option) *Graph
```

**Example:**
```go
g := graph.New(
    graph.WithStepLimit(500),
    graph.WithMaxConcurrency(8),
    graph.WithDurability(graph.DurabilitySync),
    graph.WithCheckpointer(sink),
)
```

When `WithStepLimit` is not passed, the step budget defaults to 1000. The
graph package depends on no other HNO package.

## Options

### WithStepLimit

**Signature:**
```go
func WithStepLimit(n int) Option
```

Caps the number of dispatched activations per `Run`. Default 1000 when unset.
`n` must be positive: `0` or negative is rejected by `Validate()` together
with the declaration itself.

### WithMaxConcurrency

**Signature:**
```go
func WithMaxConcurrency(n int) Option
```

Caps in-flight activations. `0` or negative means unlimited — unlimited is
the guard's own semantics. When slots are full, new activations wait in a
FIFO queue; each completion dispatches the next one.

### WithDurability

**Signature:**
```go
func WithDurability(d Durability) Option
```

Selects the checkpoint commit tier (`DurabilitySync`, `DurabilityAsync`, or
`DurabilityExit`). Read once at `Run` entry. See
[Graph Durability](/advanced/graph-durability).

### WithCheckpointer

**Signature:**
```go
func WithCheckpointer(cp Checkpointer) Option
```

Attaches the persistence sink. `nil` (default) means no persistence and no
error. Fails closed: a sink error never lets `Run` hand out a conclusion.

## Node, NodeFunc, Typed

**Signature:**
```go
type Node interface {
    Name() string
    Run(ctx context.Context, in any) (any, error)
}

func NodeFunc(name string, fn func(ctx context.Context, in any) (any, error)) Node

func Typed[TIn, TOut any](name string, fn func(ctx context.Context, in TIn) (TOut, error)) Node
```

`NodeFunc` adapts an `any`-typed function. `Typed` provides compile-time
typing for the node body; on an input whose dynamic type does not match
`TIn`, the node returns an attributable error instead of panicking:

```
graph: node "upper": input type mismatch: expected string, got int
```

A `nil` input renders as `no value` in that message.

**Example:**
```go
n := graph.Typed[string, int]("count", func(ctx context.Context, in string) (int, error) {
    return len(in), nil
})
g.AddNode(n)
```

## Send, Sender, SenderFunc

**Signature:**
```go
type Send struct {
    Node string // target node of this dispatch
    In   any    // this dispatch's own input
}

type Sender interface {
    SendRun(ctx context.Context, in any) (out any, sends []Send, err error)
}

func SenderFunc(name string, fn func(ctx context.Context, in any) (any, []Send, error)) Node
```

`Send` is one runtime-decided fan-out dispatch: the count of dispatches and
their inputs are runtime data. `Sender` is an optional extension interface —
a node implementing it emits `sends` alongside its regular output within the
same activation. The method is deliberately not named `Run`: `Node.Run`
already occupies the two-value signature, and one type cannot implement both
shapes. On every activation the scheduler type-asserts on the producer path:
assertion success runs `SendRun` and dispatches the returned sends; failure
runs `Node.Run` with no dispatch — ordinary nodes are unaffected.
`SenderFunc` adapts a function into such a node; its `Run` view discards
sends, and the engine never routes `Sender` nodes through `Run`.

**Example:**
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

Fail-closed: a `Send` naming an unregistered node fails `Run` with a named
error, and all targets are checked before any dispatch — no partial fan-out:

```
graph: node "fanout" sent to unregistered node "ghost"
```

Each dispatched `Send` is a normal activation and consumes one step. Barrier
and convergence semantics live in `AddJoinSend` below and in the
[Graph Engine guide](/guide/graph-engine).

## Predicate

**Signature:**
```go
type Predicate func(out any) bool
```

The routing test for a conditional edge; the argument is the source node's
output for this completion. A conditional edge declared with a `nil`
predicate is rejected by `Validate()`.

## Graph builder methods

All builder methods return `*Graph` for chaining. They have no error channel:
structural problems are reported by `Validate()`.

### AddNode

**Signature:**
```go
func (g *Graph) AddNode(n Node, opts ...NodeOption) *Graph
```

Registers a node, optionally attaching node policies (see below). Registering
the same name again replaces the node and its policy set in the table, but the
duplicate is recorded and `Validate()` rejects the graph afterward. In-flight
runs are isolated by a topology snapshot.

### AddEdge

**Signature:**
```go
func (g *Graph) AddEdge(from, to string) *Graph
```

Unconditional edge: when `from` completes, `to` is always activated with
`from`'s output as input.

### AddConditional

**Signature:**
```go
func (g *Graph) AddConditional(from, to string, p Predicate) *Graph
```

Conditional edge: activated only when `p(fromOutput)` is true. Multiple
conditional edges from one source are not mutually exclusive. `p` must not be
`nil`.

### AddDefault

**Signature:**
```go
func (g *Graph) AddDefault(from, to string) *Graph
```

Fallback edge: activated only when no unconditional or conditional edge from
`from` fired on this completion.

### AddJoin

**Signature:**
```go
func (g *Graph) AddJoin(from []string, to string) *Graph
```

Barrier declaration: `to` is activated once after every distinct predecessor
in `from` has completed. The activation input is a `map[string]any` keyed by
predecessor name. Fewer than 2 distinct predecessors is rejected by
`Validate()` — that shape has no barrier to wait on and belongs to `AddEdge`.

### AddJoinSend

**Signature:**
```go
func (g *Graph) AddJoinSend(source, target string) *Graph
```

Declares the convergence for runtime fan-out: after every `Send` dispatched
under `source` in the current wave has completed, `target` is activated once
with an aggregate input — a `map[string][]any` keyed by source name, values
in dispatch order. Build time checks endpoint existence and contributes a
`source -> target` reachability edge (the source itself counts as reachable);
it does not consult the `AddJoin` predecessor-count rule — the two
declarations keep separate criteria, and `AddJoin`'s semantics are unchanged.
At runtime the barrier settles when `source`'s pending-`Send` count returns
to zero. Waves settle independently: if the source re-runs and dispatches
again, the barrier re-arms and the target receives only the new wave. If no
`Send` ever names `source`, the target is not activated (the zero-dispatch
note in the guide describes the resulting `Run` shape honestly). Multiple
targets per source and multiple sources per target are allowed; the latter
fires once only when all source counts are zero simultaneously.

### SetEntry / SetOutput

**Signature:**
```go
func (g *Graph) SetEntry(name string) *Graph
func (g *Graph) SetOutput(name string) *Graph
```

`SetEntry` names the node that receives `Run`'s input. `SetOutput` names the
node whose return value becomes `Result.Output()`. Both must reference
registered nodes reachable from the entry, or `Validate()` rejects.

## Validate

**Signature:**
```go
func (g *Graph) Validate() error
```

Runs the full build-time catalog: declaration-level checks (step budget,
entry, output, edge endpoints, predicates, routable edge kinds), then
underfed joins, then unreachable nodes, then unbreakable cycles, then
duplicate node names. Errors are stable, attributable, and name the offending
declaration; each check reports one item (lexicographically first for
map-sourced sets). `Run` calls `Validate()` before executing.

## Warnings

**Signature:**
```go
func (g *Graph) Warnings() []error
```

Non-fatal build-time hints, deliberately separate from `Validate()`.
Read-only: never mutates config, never affects execution. The current warning
fires when a graph containing a cycle has no explicit `WithStepLimit`:

```
graph: cycle "a" -> "b" -> "a" has no explicit budget: WithStepLimit was not set,
so the default stepLimit=1000 is what bounds this graph
```

## Run

**Signature:**
```go
func (g *Graph) Run(ctx context.Context, in any) (*Result, error)
```

Validates, snapshots the topology, and drives execution from the entry node
to convergence. Runtime guarantees:

- Step-limit valve: exceeding the budget returns an error wrapping
  `ErrStepLimitExceeded`; the activation that would cross the limit is never
  started.
- Panic barrier: a node panic becomes an error naming the node
  (`graph: node "..." panicked while running: ...`).
- Cancellation normalization: when caller cancellation and a node error
  coincide, `Run` returns `ctx.Err()`.
- Send targeting: a `Send` naming an unregistered node fails the run with a
  named error (`graph: node "..." sent to unregistered node "..."`), checked
  before any dispatch.
- Snapshot isolation: builder mutations after entry affect only future runs.

## Result

**Signature:**
```go
type Result struct { /* unexported fields */ }

func (r *Result) Output() any
func (r *Result) Value(name string) (any, bool)
func (r *Result) Completed() []string
```

- `Output()` returns the output node's value (nil if it never produced one).
- `Value(name)` returns any completed node's output and whether it produced
  one.
- `Completed()` returns completed node names sorted lexicographically — not
  completion order, because parallel branches have no deterministic global
  ordering. Do not assert on timing order.

## ErrStepLimitExceeded

**Signature:**
```go
var ErrStepLimitExceeded = errors.New("graph: step limit exceeded")
```

Sentinel wrapped by the step-limit valve; match with `errors.Is`.

## Node policies

Attached through `AddNode`'s variadic options:

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

`MaxAttempts < 1` (or no retry config) clamps to 1 attempt — "no retry" is
the zero-value semantics. `ShouldRetry == nil` never retries. Backoff waits
yield to context cancellation.

### TimeoutConfig

```go
type TimeoutConfig struct {
    Timeout    time.Duration
    PerAttempt bool
}
```

`Timeout <= 0` means no limit. `PerAttempt: false` is one budget for the whole
node including all retries; `PerAttempt: true` gives each attempt a fresh
deadline. Delivered to the node as a derived context.

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

Default key is the SHA-256 hex of the input's `%#v` form. Fail-closed: only
successful results are written; a cache hit skips execution. The engine ships
no default store — implementing `CacheStore` is the adaptation seam.

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

Events are emitted serially by the consumer goroutine (no hook
synchronization needed), on both success and failure paths.
`RedactIn`/`RedactOut` nil the payload fields before delivery.

## Durability, Checkpoint, Checkpointer

**Signature:**
```go
type Durability int

const (
    DurabilitySync  Durability = iota // default (zero value): sync before next dispatch
    DurabilityAsync                   // background flush, drained before Run returns
    DurabilityExit                    // accumulate, flush once at exit
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

`Seq` is stamped by the consumer from 1, contiguously, in completion
processing order. Sink failures fail closed and are attributable via the
wrapped error (`graph: checkpoint sink: ...`). Resume/replay is not part of
this seam yet. See [Graph Durability](/advanced/graph-durability).

## Human-in-the-Loop (HITL)

### RequestInterrupt(i Interrupt) error

Call inside a node body to halt the run. The engine converts the returned
sentinel into a suspension: `Run` returns `(nil, err)` with
`errors.Is(err, ErrSuspended)` and `errors.As(err, *Suspension)`.

`Interrupt` fields: `InterruptID` (required, unique among pending), `Message`,
`ResponseSchema` (subset: `type` + `required` + `properties.<name>.type`),
`Payload`, `Mode` (`ResumeRerun` default | `ResumeHandoff`).

### Resume(ctx context.Context, responses map[string]any) (*Result, error)

Resume a suspended graph. The response set must cover exactly the pending
interrupt IDs. Each response is validated against its `ResponseSchema`;
failures return an error wrapping `ErrInvalidResponse` and preserve the
suspension. `ResumeRerun` re-executes the waiting node with the response
reachable via `InterruptResponse(ctx, id)`; `ResumeHandoff` skips the node and
feeds the response to successors as its output. Repeated resume with nothing
pending returns `ErrNothingToResume`. Budget continues across resume.

### InterruptResponse(ctx context.Context, interruptID string) (any, bool)

Read a resolved response inside a re-run node body.

### Sentinels

`ErrSuspended` (run awaiting human), `ErrNothingToResume` (idempotent
no-resume), `ErrInvalidResponse` (schema mismatch, suspension preserved).

### Suspension

Carried by the `Run` error via `errors.As`: `.Interrupts` (pending list) and
`.Completed` (nodes finished before the halt). Durable via
[Durability](/advanced/graph-durability) — suspensions commit as
`EntryInterrupt` checkpoint entries. Cross-process restart pairs with
`pkg/hno/session/sidecar` + `internal/hitlbridge`.

## Related Pages

- [Graph Engine guide](/guide/graph-engine)
- [Node Policies guide](/guide/node-policies)
- [Graph Durability](/advanced/graph-durability)
- [Workflow API](/api/workflow) for the v1 step-based orchestrator
