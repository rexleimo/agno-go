# Graph Durability - Checkpoint Tiers

Control when a running graph commits completed-node events to your persistent
sink: per node, in the background, or once at exit.

---

## Overview

The graph engine's durability seam answers one question: **after a node
completes, when does its event reach the sink?** It is commit-side only —
the engine records what happened; it does not (yet) read checkpoints back.
You compose two declarations:

```go
g := graph.New(
	graph.WithDurability(graph.DurabilitySync),     // the tier
	graph.WithCheckpointer(mySink),                 // the sink
)
```

- `WithDurability` selects one of three commit tiers. The zero value is
  `DurabilitySync` — an undeclared graph runs at the **safest** tier, not the
  fastest.
- `WithCheckpointer` attaches the sink. No sink (`nil`) means no persistence
  at all: the graph runs identically and nothing errors.

Both are read once at `Run` entry, like `WithStepLimit` and
`WithMaxConcurrency` — mutating the builder mid-run cannot rewrite the tier of
the run in flight.

---

## The Checkpointer Contract

```go
type Checkpoint struct {
	Seq    int    // stamped by the consumer, from 1, contiguous
	Node   string
	Output any
}

type Checkpointer interface {
	Append(ctx context.Context, cp Checkpoint) error
}
```

The seam is declared inside the graph package on purpose: the engine does not
depend on any storage package. Adapting Postgres, a log file, or an in-memory
slice is your one-method implementation.

The entry stream is **append-only**:

- One entry per completed node, exactly once, with the node's output for that
  completion.
- `Seq` is stamped by the single consumer goroutine at commit time — it starts
  at 1, increments contiguously, and its order is the order completions were
  processed in this `Run`. Nodes that re-run under a conditional cycle commit
  again with fresh sequence numbers.
- There is no read-back API in this seam. Consuming the stream (tailing a log,
  querying a table) is the sink's business.

---

## The Three Tiers

| Tier | When entries reach the sink | Trade-off |
|---|---|---|
| `DurabilitySync` (default, zero value) | Synchronously, right after each node completes and **before the next dispatch round** | Safest: an entry is durably behind before any successor starts. Adds one sink round-trip per node. |
| `DurabilityAsync` | By a background flush goroutine; all entries flushed before `Run` returns | Dispatch is never blocked. Entries are never lost and never reordered, but a successor may already be running while its predecessor's entry is still in flight — do not rely on sink visibility for ordering. |
| `DurabilityExit` | Accumulated in memory; flushed once, in sequence, right before `Run` returns | Zero sink traffic during the run. A crash mid-run loses everything since process start. |

Tier timing is entirely consumer-side, which is why it costs no locks: the
`Async` flusher goroutine shares only a channel of pending commits with the
engine, and the single consumer remains the only writer of commit state.

### Sync in detail

`Sync` means "happen-before successors": the consumer calls `Append` after
recording the completion and before dispatching the next round. On a chain
`a -> b`, `a`'s entry is committed before `b` is handed its activation.

### Async in detail

Entries flow to the flusher immediately but without blocking dispatch. When
`Run` returns — success or failure — the flusher has been drained and joined.
Losing entries or committing them out of `Seq` order is not possible; what is
**not** promised is that a successor's body can observe its predecessor's
entry in the sink.

### Exit in detail

The run itself sees zero sink calls. Everything lands in one ordered burst at
the end. Useful for cheap, restartable batch graphs where per-node durability
buys nothing.

---

## Failure Semantics (Fail-Closed)

A sink failure is never silent, and it never lets `Run` hand out a conclusion
built on top of lost persistence:

- **Sync**: the first failed `Append` fails `Run` immediately. No successor is
  dispatched on top of an unpersisted completion.
- **Async / Exit**: the failure is surfaced during exit finalization — `Run`
  fails rather than returning success with quietly missing entries.
- The error wraps whatever your sink returned, so `errors.Is` / `errors.As`
  attribute it:

```
graph: checkpoint sink: postgres: connection refused
```

Two boundaries keep failure reporting honest:

- **A run error wins.** If the graph itself failed, that primary error is
  returned; a concurrent sink error does not mutate it into a double report.
- **Completed work is still committed on failure paths.** If a run fails (node
  error, step limit, cancellation, panic barrier), entries for nodes that did
  complete are still flushed — the point of durability is to record what
  happened, including when it ended badly.

---

## Example

```go
package main

import (
	"context"
	"fmt"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

type logSink struct{ entries []graph.Checkpoint }

func (s *logSink) Append(ctx context.Context, cp graph.Checkpoint) error {
	fmt.Printf("seq=%d node=%s out=%v\n", cp.Seq, cp.Node, cp.Output)
	s.entries = append(s.entries, cp)
	return nil
}

func main() {
	sink := &logSink{}

	g := graph.New(
		graph.WithDurability(graph.DurabilitySync),
		graph.WithCheckpointer(sink),
	)
	g.AddNode(graph.NodeFunc("a", func(ctx context.Context, in any) (any, error) {
		return "A-out", nil
	}))
	g.AddNode(graph.NodeFunc("b", func(ctx context.Context, in any) (any, error) {
		return "B-out", nil
	}))
	g.AddEdge("a", "b")
	g.SetEntry("a")
	g.SetOutput("b")

	res, err := g.Run(context.Background(), "input")
	fmt.Println(res.Output(), err, len(sink.entries))
	// seq=1 node=a out=A-out   (printed before b is dispatched)
	// seq=2 node=b out=B-out
	// B-out <nil> 2
}
```

---

## What Is Deliberately Not Here Yet

Honest scope, as of this page:

- **Resume / replay has landed (G7).** Suspended runs are committed as
  `EntryInterrupt` entries on this seam and restored from them:
  [Human-in-the-Loop](/guide/human-in-the-loop) covers in-process `Resume`
  and cross-process restart via the session sidecar.
- **No bridging to stream events.** Checkpoints are engine-side entries; the
  `run.StreamCheckpoints` protocol event family
  ([Run Events API](/api/run-events)) is a separate protocol layer whose
  producer wiring is still pending.
- **No storage format, encoding, or TTL policy.** Your sink decides what
  `Append` means physically.

---

## Next Steps

- See [Graph Engine](/guide/graph-engine) for routing, validation, and runtime
  guardrails.
- Attach [Node Policies](/guide/node-policies) to the same nodes you
  checkpoint.
- Browse signatures in the [Graph API Reference](/api/graph).
