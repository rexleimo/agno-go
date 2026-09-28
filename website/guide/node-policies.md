# Node Policies - Retry, Timeout, Cache, Trace

Attach execution policies to individual graph nodes through one variadic
seam: `AddNode(n, opts...)`.

---

## The Variadic Seam

A node accepts any number of `NodeOption` values. Existing single-argument
`AddNode(n)` calls are unaffected; policies are entirely opt-in per node:

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

When the same node name is registered again, the later registration's options
fully replace the earlier policy set for that node.

The four policies execute inside the scheduler's per-activation envelope, in
this order:

1. **Cache lookup** — a hit skips node execution entirely.
2. **Up to N timeout-bounded attempts** — the retry loop.
3. **Cache write** — only on success.
4. **Trace emission** — from the consumer goroutine, after the outcome is
   known.

Retries stay inside one activation: they produce no new activations and
consume no step-limit budget. Backoff waits always yield to caller
cancellation.

---

## Retry — `WithRetry(RetryConfig)`

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

Semantics:

- Retries the **same activation** up to `MaxAttempts` times; the last error is
  returned when attempts are exhausted or `ShouldRetry` says stop.
- `ShouldRetry` is the gate for another attempt; `nil` means "never retry"
  (the first error stands).
- Backoff: attempt *k* waits `InitialDelay * BackoffFactor^(k-1)`, capped at
  `MaxDelay`. `InitialDelay <= 0` disables waiting entirely.
- During a backoff wait, cancelling the context aborts immediately with
  `ctx.Err()`.
- Honesty note: the `Jitter` field exists in the config struct but the landed
  scheduler does not apply it yet.

---

## Timeout — `WithTimeout(TimeoutConfig)`

```go
type TimeoutConfig struct {
	Timeout    time.Duration
	PerAttempt bool
}
```

The deadline is delivered to your node as a derived `context` — nodes that
respect `ctx` respect the budget. Two shapes:

| Setting | Meaning |
|---|---|
| `PerAttempt: false` (default) | One budget for the **whole node**, covering **all** retry attempts. The deadline is installed once, before the retry loop. |
| `PerAttempt: true` | A **fresh deadline per attempt** — every retry gets its own full `Timeout`. |

`Timeout <= 0` means no time limit — the zero value is a usable, deliberate
configuration.

---

## Cache — `WithCache(CacheConfig)`

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

- **Keys**: supply `KeyFunc` for domain-aware keys; without it the key is the
  SHA-256 hex digest of the input's `%#v` rendering.
- **Store**: the engine ships no default store. Implement the two-method
  `CacheStore` interface against whatever backend you use — the graph package
  deliberately depends on no cache implementation.
- **TTL** is passed through to `SetAny`; interpretation belongs to the store.
- **Fail-closed caching**: only successful results are written. A failed node
  never enters the cache — caching an error would hand that error to the next
  run as if it were a conclusion. A cache hit returns immediately and counts
  as one attempt.

---

## Trace — `WithTrace(TraceConfig)`

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

- The hook receives one event per node outcome — on success and on failure
  (a failed node's event carries `Err`).
- Events are emitted **serially by the consumer goroutine**, so a hook never
  needs its own synchronization, and the zero-lock model is preserved. Keep
  hooks fast: they run on the scheduling path.
- `RedactIn` / `RedactOut` nil out `In` / `Out` before delivery, so payloads
  such as prompts or secrets never reach the sink.
- `Enabled: false` or a `nil` hook emits nothing — an off switch that is
  actually off.

---

## Zero Values Mean Semantics

Policy configs are clamped by guards at read time, not validated at build
time. The zero value of every config is a working, deliberate configuration —
"no retry / no time limit / no caching / no tracing" is what you get by simply
not declaring the option:

| Declaration | Effective semantics |
|---|---|
| No `WithRetry` | 1 attempt, no retries |
| `RetryConfig{}` (zero value) | 1 attempt — `MaxAttempts < 1` clamps to 1 |
| `ShouldRetry: nil` | Never retry |
| No `WithTimeout` | No time limit |
| `TimeoutConfig{}` / `Timeout <= 0` | No time limit |
| No `WithCache` / `Store: nil` | No caching |
| `KeyFunc: nil` | SHA-256 of `%#v` of the input |
| No `WithTrace` / `Enabled: false` / `Hook: nil` | No events |

This is a deliberate contrast with the graph-level `WithStepLimit(0)`, which
`Validate()` **rejects** outright: a non-positive step limit has no readable
meaning (it is neither "unlimited" nor "zero steps"), and a safety valve that
silently turns itself off at the moment it matters most is worse than a build
error. Policy zero values, on the other hand, all map to "the policy is simply
not active" — a state with exactly one sensible reading. Two postures, chosen
per knob, not one rule forced everywhere.

---

## Where Policies Execute

- **Retry, timeout, and cache** run inside the producer goroutine that executes
  the node. They never spawn activations and never touch scheduler state.
- **Trace** events are emitted by the single consumer, which is why the hook
  contract can promise serial delivery without a lock.
- The whole envelope stays within one activation for step-accounting purposes:
  a node that retries five times still consumes exactly one step.

---

## Complete Example

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
	fmt.Println(res.Output(), err) // ok:payload <nil> — third attempt succeeded
}
```

---

## Next Steps

- Read the [Graph Engine guide](/guide/graph-engine) for routing and
  validation.
- See [Graph Durability](/advanced/graph-durability) for commit-side
  persistence, which composes with policies on the same nodes.
- Browse signatures in the [Graph API Reference](/api/graph).
