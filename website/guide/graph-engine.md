# Graph Engine - Zero-Lock Control-Flow Graphs

Build DAGs, parallel branches, conditional routes, join barriers, and runtime
fan-out on a scheduler that carries no locks at all.

---

## What is the Graph Engine?

The **Graph Engine** (`pkg/hno/graph`) is HNO's control-flow graph kernel. You
declare nodes and five kinds of edges; the engine validates the whole graph at
build time, then drives execution from an entry node to a converged result.

It is the v3 orchestration line, and it is **not** a rewrite of the
[Workflow](/guide/workflow) primitives. Both APIs coexist today:

- `pkg/hno/workflow` — the v1 step-based orchestrator (Step/Condition/Loop/
  Parallel/Router).
- `pkg/hno/graph` — the v3 graph engine described here (DAG, conditional and
  default routing, join barriers, node policies, durability).

Pick the package you need; do not mix them in one pipeline.

### Key Features

- **Five routing kinds**: unconditional, conditional, default (fallback),
  join (barrier), and join-send (fan-out convergence) edges.
- **Dynamic fan-out**: a running node can dispatch `Send` work items — count
  and inputs decided at runtime — and an explicit `AddJoinSend(source, target)`
  declaration converges them into one aggregate activation.
- **Build-time validation**: dangling edges, duplicate node names, unreachable
  nodes, unbreakable cycles, and underfed joins are rejected by `Validate()`
  with attributable error strings — before anything runs.
- **Runtime guardrails**: a step-limit valve (default 1000), a panic barrier,
  cancellation normalization, and an optional max-concurrency FIFO.
- **Zero-lock scheduler**: one consumer goroutine owns all mutable state;
  producers only send on a channel. No `sync.` primitives appear anywhere in
  the engine files, and the full test suite passes under `-race`.
- **Node policies**: retry, timeout, cache, and trace attach per node via a
  variadic option — see [Node Policies](/guide/node-policies).
- **Durability**: sync/async/exit checkpoint tiers via a caller-supplied sink —
  see [Graph Durability](/advanced/graph-durability).

---

## The Zero-Lock Model

Most graph executors guard shared state with mutexes. This engine does not
need them, by construction:

- Each node activation runs on its own **producer goroutine**. Producers
  execute node code and send exactly one completion item on a channel. They
  never write scheduler state.
- The goroutine that called `Run` is the **single consumer**. It is the only
  reader of the queue and the only writer of the pending list, the running
  counter, results, and step accounting.

State that only one goroutine touches needs no lock. You can audit the claim
directly on the engine sources (`graph.go`, `scheduler.go`, `policy.go`,
`durability.go`):

```bash
grep -o 'sync\.' pkg/hno/graph/graph.go pkg/hno/graph/scheduler.go \
  pkg/hno/graph/policy.go pkg/hno/graph/durability.go | wc -l
# 0
```

---

## A Complete Graph

```go
package main

import (
	"context"
	"fmt"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

func main() {
	g := graph.New(
		graph.WithStepLimit(500),    // default 1000; 0/negative is rejected by Validate
		graph.WithMaxConcurrency(8), // 0/negative means unlimited; full slots queue FIFO
	)

	// two parallel branches fanning out of "fetch"
	g.AddNode(graph.NodeFunc("fetch", func(ctx context.Context, in any) (any, error) {
		return fmt.Sprintf("rows for %v", in), nil
	}))
	g.AddNode(graph.NodeFunc("left", func(ctx context.Context, in any) (any, error) {
		return fmt.Sprintf("left(%v)", in), nil
	}))
	g.AddNode(graph.NodeFunc("right", func(ctx context.Context, in any) (any, error) {
		return fmt.Sprintf("right(%v)", in), nil
	}))
	g.AddNode(graph.NodeFunc("merge", func(ctx context.Context, in any) (any, error) {
		// join input is map[string]any keyed by predecessor name
		parts := in.(map[string]any)
		return fmt.Sprintf("merged: %v", parts), nil
	}))
	g.AddNode(graph.NodeFunc("route", func(ctx context.Context, in any) (any, error) {
		return in, nil
	}))
	g.AddNode(graph.NodeFunc("heavy", func(ctx context.Context, in any) (any, error) {
		return fmt.Sprintf("heavy(%v)", in), nil
	}))
	g.AddNode(graph.NodeFunc("light", func(ctx context.Context, in any) (any, error) {
		return fmt.Sprintf("light(%v)", in), nil
	}))

	g.AddEdge("fetch", "left")
	g.AddEdge("fetch", "right")
	g.AddJoin([]string{"left", "right"}, "merge")
	g.AddEdge("merge", "route")
	g.AddConditional("route", "heavy", func(out any) bool {
		return len(fmt.Sprint(out)) > 40 // pick a branch by payload shape
	})
	g.AddDefault("route", "light")

	g.SetEntry("fetch")
	g.SetOutput("merge") // merge's return value is the run's output;
	// heavy/light still run after it — the output node need not be terminal

	res, err := g.Run(context.Background(), "demo")
	if err != nil {
		panic(err)
	}
	fmt.Println(res.Output())        // the output node's return value
	out, ok := res.Value("route")    // any completed node's output
	fmt.Println(out, ok)
	fmt.Println(res.Completed())     // completed node names, lexicographically sorted
}
```

---

## Routing Semantics

When a node completes, the consumer decides which successors to activate.
All matching successors of one node run concurrently as separate activations.

### 1. Unconditional edges — `AddEdge(from, to)`

`from` completing always activates `to`, with `from`'s output as input.

### 2. Conditional edges — `AddConditional(from, to, pred)`

Activated only when `pred(out)` returns `true` for this run's output. Multiple
conditional edges from the same source are **not mutually exclusive** — every
predicate that hits activates its target. A `nil` predicate is a declaration
with no routing input and is rejected at build time; the engine never treats it
as "never fires" and silently skip the edge.

### 3. Default edges — `AddDefault(from, to)`

The fallback: activated only when **no** unconditional or conditional edge from
`from` fired on this completion. Concrete hits suppress the default.

### 4. Join edges — `AddJoin(from, to)`

Declares a barrier: `to` is activated once, after **every** declared
predecessor in `from` has completed. Its input is not one predecessor's output
but a `map[string]any` keyed by predecessor name.

- A join with fewer than 2 distinct predecessors is not a barrier at all —
  `Validate()` names the target and rejects it. Use `AddEdge` instead.
- When predecessors re-run because of a conditional cycle, the key set does not
  grow, so the join target is activated again with the refreshed map.
- Treat the join input map as read-only for the duration of your node's `Run`;
  do not retain the reference beyond it. The engine currently hands the same
  map object it keeps updating (a snapshot hand-off is tracked as an open
  engineering item).
- Honesty note: if a barrier's predecessors never all run and no activations
  remain, today's `Run` returns a nil error with an empty output rather than a
  diagnosis. That shape is a known open adjudication item, not a contract.

### 5. Join-send edges — `AddJoinSend(source, target)`

Declares convergence for **runtime** fan-out: `to` is activated once after
every `Send` dispatched under `source` in the current wave completes, with an
aggregate input. Unlike a join edge it has no declared predecessor list — the
workload arrives via `Send` values that exist only at runtime. The full story
(lifecycle, zero-dispatch semantics, and when to prefer it over `AddJoin`) is
in the Dynamic Fan-out section below.

---

## Dynamic Fan-out — `Send` and `AddJoinSend`

Static edges commit to a fixed shape before `Run` starts. The `Send` seam
completes the picture: a node running at runtime decides **how many**
dispatches to emit and **what input** each carries — the half a static edge
cannot express. Together with the `AddJoinSend` convergence declaration it
gives you the map-reduce shape without pre-declaring a single branch.

### The map-reduce story

```go
package main

import (
	"context"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

func main() {
	g := graph.New().
		AddNode(graph.SenderFunc("fanout", func(ctx context.Context, in any) (any, []graph.Send, error) {
			items := in.([]string) // count and payloads are runtime data
			sends := make([]graph.Send, 0, len(items))
			for _, it := range items {
				sends = append(sends, graph.Send{Node: "summarize", In: it})
			}
			return nil, sends, nil
		})).
		AddNode(graph.NodeFunc("summarize", func(ctx context.Context, in any) (any, error) {
			return summarizeOne(in.(string)), nil // one activation per Send; input = that Send's In
		})).
		AddNode(graph.NodeFunc("reduce", func(ctx context.Context, in any) (any, error) {
			// aggregate input: map[string][]any keyed by source name,
			// values in dispatch order
			outs := in.(map[string][]any)["summarize"]
			return mergeAll(outs), nil
		})).
		AddJoinSend("summarize", "reduce"). // declare the convergence
		SetEntry("fanout").
		SetOutput("reduce")

	res, err := g.Run(context.Background(), []string{"alpha", "beta", "gamma"})
	_ = res
	_ = err
}
```

What happens on `Run`:

1. `fanout` activates once and returns three `Send` values. Each is
   dispatched as an ordinary activation of `summarize` with its own `In` —
   three items, three activations, three independent outputs.
2. Every completion under `summarize` is booked against the
   `AddJoinSend("summarize", "reduce")` barrier. When the pending count for
   `summarize` returns to zero, `reduce` is activated **once**, with a
   `map[string][]any` keyed by source name and the outputs in dispatch order.
3. `reduce`'s return value is the run output.

Note there is no `AddEdge("fanout", "summarize")` — the `Send` values *are*
the dispatch. And without the `AddJoinSend` declaration, `Validate()` would
reject `reduce` as unreachable: the declaration contributes a
`summarize -> reduce` reachability edge, and `summarize` itself counts as
reachable (a `Send` target name is a runtime value, so build time cannot
refute that some `Sender` will name it — the same optimistic precedent that
lets reachability ignore predicates).

### Runtime semantics, pinned

- **`Sender` is an optional extension interface.** Its method is deliberately
  not named `Run`: `Node.Run` already occupies the two-value signature, and
  one type cannot implement both. The scheduler type-asserts every activation
  on the producer path — `Sender` nodes go through `SendRun` and their sends
  get dispatched; ordinary nodes go through `Node.Run` and are unaffected.
  `SenderFunc` adapts a function into such a node.
- **Fail-closed targeting.** A `Send` naming an unregistered node fails the
  run with a named error, and all targets are checked before any dispatch —
  no partial fan-out. Build time cannot do this check (targets exist only at
  runtime), so it is a runtime check that refuses to silently drop:

  ```
  graph: node "fanout" sent to unregistered node "ghost"
  ```

- **Every `Send` is a billed activation.** Dispatched sends and the
  join-send target activation each consume one step, so unbounded fan-out (a
  loop that re-dispatches forever) is caught by the step-limit valve
  (`errors.Is(err, graph.ErrStepLimitExceeded)`), not by memory pressure.
- **Multi-wave fan-out settles per wave.** If the source re-runs in a
  conditional cycle and dispatches again, the barrier re-arms: each wave
  converges independently and the target receives only that wave's outputs.
- **Multiple declarations compose.** One source may declare several targets
  (each fires once when the source's count hits zero); one target may declare
  several sources (it fires once only when all of their counts are zero at
  the same time).
- **Ordinary edges coexist.** A source may also keep unconditional or
  conditional outgoing edges — both paths fire, each by its own semantics:
  normal successors activate per the routing rules while the join-send target
  waits for the sends.

### Zero dispatch is not an activation

If the source activates but returns no sends (an empty batch is normal data,
not an error), the pending count never leaves zero, no completion triggers
the barrier check, and the target is not activated. If no other activations
remain, `Run` returns `err == nil` with `res.Output() == nil` — the same
shape as an `AddJoin` barrier whose predecessors never all complete. That is
today's pinned behavior and a known open adjudication item (the R19-Q1
family), not a bug-fix promise; check `res.Value("reduce")` when a
zero-dispatch wave is possible.

### AddJoin vs AddJoinSend: when to use which

| | `AddJoin(from []string, to)` | `AddJoinSend(source, target)` |
|---|---|---|
| Predecessors decided | at build time (the declared list) | at runtime (whatever `Send` names the source this wave) |
| Barrier settles when | every declared predecessor has completed | the source's pending-`Send` count returns to zero |
| Target input | `map[string]any` keyed by predecessor name | `map[string][]any` keyed by source name, outputs in dispatch order |
| Build-time criteria | fewer than 2 distinct predecessors is rejected by `Validate()` | endpoints exist; contributes a `source -> target` reachability edge; no predecessor count |
| Use it when | the set of branches is fixed in code | the workload count and payloads are runtime data (map-reduce, per-item scattering) |

The two declarations deliberately keep separate criteria: `AddJoin`'s
underfed-join rejection is untouched, and an `AddJoinSend` declaration never
consults it.

---

## Build-Time Validation

`Validate()` runs automatically at the top of `Run`, or explicitly on demand.
The rule is fail-closed: an unusable declaration is rejected with an error that
names it, never silently dropped into a graph that "runs" with missing pieces.

| # | Rejected at build time | Error text (verbatim) |
|---|---|---|
| 1 | `WithStepLimit(0)` or negative | `graph: WithStepLimit(0) is not a usable step limit: a positive number of steps is required` |
| 2 | Entry not registered | `graph: entry node "..." does not exist` |
| 3 | Output node not registered | `graph: output node "..." does not exist` |
| 4 | Edge endpoint not a node | `graph: unconditional edge from node "..." does not exist` (same shape for `to node`, and for conditional/default/join/join-send edges) |
| 5 | Conditional edge without predicate | `graph: conditional edge from "..." to "..." declares no predicate` |
| 6 | Edge kind this engine cannot route | `graph: ... edge from "..." to "..." is declared but not routable by this engine` |
| 7 | Join with < 2 distinct predecessors | `graph: join target "..." must declare at least 2 distinct predecessors, got 1` |
| 8 | Node unreachable from entry | `graph: unreachable node "...": not reachable from entry "..." through any declared edge` |
| 9 | Cycle where every edge is always taken | `graph: unconditional cycle "a" -> "b" -> "a": every edge on it is always taken, so Run cannot terminate` |
| 10 | Duplicate node name | `graph: node name "..." is declared more than once` |

Notes on the catalog:

- **Cycles are allowed.** Only cycles whose every edge is always taken
  (unconditional edges, plus default edges with no conditional alternative on
  the source) are rejected, because `Run` could never terminate. A cycle that
  any conditional edge can break is legal; the step limit bounds the worst case
  where a predicate is always true.
- **Reachability counts declared edges only**, regardless of kind or predicate:
  build time cannot evaluate predicates, and treating "reachable" as "actually
  ran" would outlaw every conditional branch. One optimistic precedent: the
  source of an `AddJoinSend` declaration counts as reachable (its traffic
  arrives via runtime `Send` values build time cannot see), and the
  declaration contributes a `source -> target` edge.
- **Errors are attributable and stable.** Most checks report one item at a time
  (lexicographically first for map-sourced sets), so the same broken graph
  always produces the same message you can paste into an issue. Long cycles are
  rendered as a head of names plus `… -> "last" (N nodes)` instead of dumping
  thousands of names.
- **Duplicate registration is a ledger, not an immediate panic.** A later
  `AddNode` with the same name still replaces the earlier node in the table
  (in-flight runs are isolated by a topology snapshot), but the name is
  recorded and `Validate()` refuses the graph afterward. Builder methods return
  `*Graph` and have no error channel — ledger-then-recheck is the only honest
  incremental shape for this API.
- **Order is observable and pinned by tests**: declaration-level checks first
  (budget, entry, output, endpoints, predicates, routable kinds), then join
  underfeeding, then reachability, then unbreakable cycles, then duplicate
  names. A graph that is both unreachable and cyclic reports reachability.

### `Warnings()` — the non-fatal channel

`Validate()` rejects only what cannot run. `Warnings()` reports what *can* run
but deserves a second look, and the two are deliberately separate:

```go
g := graph.New() // no WithStepLimit declared
g.AddNode(graph.NodeFunc("a", fnA))
g.AddNode(graph.NodeFunc("b", fnB))
g.AddEdge("a", "b")
g.AddConditional("b", "a", func(out any) bool { return shouldLoop(out) }) // breakable cycle

for _, w := range g.Warnings() {
	fmt.Println(w)
}
// graph: cycle "a" -> "b" -> "a" has no explicit budget: WithStepLimit was not set,
//        so the default stepLimit=1000 is what bounds this graph
```

The cycle above is legal (a conditional edge can break it, so `Validate()`
passes) — but if the predicate is always true, the run will grind against the
default budget of 1000 steps. The warning tells you that number is the only
thing standing there, because you never chose it.

The one warning today fires when a graph with cycles has no explicit
`WithStepLimit`: the caller is running on a default (1000) they never chose.
`Warnings()` is read-only — it never mutates config, never feeds into
`Validate()`, and calling it repeatedly returns the same answer. Calling danger
illegal would decide for the caller that the graph cannot run; a warning only
states the fact.

---

## Runtime Guardrails

Build-time checks cannot see predicates or node behavior. The runtime carries
its own valves, all enforced by the single consumer:

### Step-limit valve

Every dispatched activation consumes one step. When the count reaches the
limit before the graph converges, `Run` returns an error wrapping
`graph.ErrStepLimitExceeded`:

```go
if errors.Is(err, graph.ErrStepLimitExceeded) { ... }
// graph: step limit 1000 reached before the graph converged: graph: step limit exceeded
```

Counting happens **before** an activation is handed to a node: an activation
that would cross the limit is never started, so no unbillable side effects
escape. Retries inside a node policy do not consume steps (they happen inside
one activation) — see [Node Policies](/guide/node-policies).

### Panic barrier

A panic inside any node is recovered above the node's frame (the node's own
`defer`s run first) and converted into an error naming the node:

```
graph: node "fetch" panicked while running: runtime error: index out of range [5]
```

The goroutine still reports through the queue; it never writes scheduler
state directly.

### Cancellation normalization

Pass a cancellable context. When the caller cancels and a node error lands at
the same moment, `Run` reports the **cancellation** (`ctx.Err()`), not the node
error — the consumer re-checks the context before treating a dequeued item as a
verdict.

### Max-concurrency FIFO

`WithMaxConcurrency(n)` caps in-flight activations. When all slots are busy,
new activations wait in a FIFO queue; each completion frees one slot and the
next activation is dispatched. `0` or negative means unlimited — that is the
guard's own semantics, not an omission.

### Topology snapshot per Run

`Run` executes the topology that passed `Validate()` at entry, captured as a
snapshot. Mutating the builder after `Run` starts affects only future runs —
a node's stray `AddEdge` cannot smuggle an unvalidated declaration into the
run in flight. `stepLimit`, `maxConcurrency`, and durability settings are read
once at entry under the same invariant.

---

## Typed Nodes

`NodeFunc` is `any`-typed by design — the graph does not understand your data.
When you want compile-time typing at the leaves, `Typed` adapts a concrete
function:

```go
upper := graph.Typed[string, string]("upper", func(ctx context.Context, in string) (string, error) {
	return strings.ToUpper(in), nil
})
g.AddNode(upper)
```

If the input's dynamic type does not match `TIn`, the node returns an
attributable error instead of panicking:

```
graph: node "upper": input type mismatch: expected string, got int
```

A `nil` input renders as `no value` rather than formatting noise.

---

## Roadmap

Only landed behavior is documented on this page. One adjacent capability is
designed but **not yet delivered**:

- **Resume / replay from checkpoints, and HITL (`RequestInterrupt` /
  `Resume`)** — the durability seam today is commit-side only; reading
  checkpoints back to resume a suspended graph, and human-in-the-loop
  interrupt/resume, are planned for G7 (contract:
  [v3-test-scope-p1-graph-slice27.json](https://github.com/rexleimo/HNO/blob/main/docs/design/v3-test-scope-p1-graph-slice27.json)).
  See [Graph Durability](/advanced/graph-durability).

---

## Next Steps

- Attach [Node Policies](/guide/node-policies) (retry, timeout, cache, trace)
  to individual nodes.
- Persist completed nodes with [Graph Durability](/advanced/graph-durability).
- Browse signatures in the [Graph API Reference](/api/graph).
- Compare with [Workflow](/guide/workflow) for the v1 step-based orchestrator.
