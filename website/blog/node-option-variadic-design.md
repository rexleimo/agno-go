---
title: "Four Node Policies, One Variadic Seam: A NodeOption Design Story"
description: "How HNO attaches Retry, Timeout, Cache, and Trace to graph nodes through one variadic AddNode option — with zero-value semantics, a fail-closed cache, and consumer-side trace hooks."
date: 2026-09-28
lastUpdated: 2026-09-28
author: HNO Team
category: API design
tags:
  - Go
  - API design
  - retry
  - cache
  - timeout
  - observability
head:
  - - meta
    - name: keywords
      content: "Go variadic options, NodeOption design, node policies, retry timeout cache trace, zero value semantics, fail-closed cache, agent framework API design"
  - - meta
    - property: og:type
      content: article
  - - meta
    - property: og:title
      content: "Four Node Policies, One Variadic Seam: A NodeOption Design Story"
  - - meta
    - property: og:description
      content: "Retry, Timeout, Cache, and Trace as variadic AddNode options: zero-value semantics, fail-closed caching, and serial trace hooks in a zero-lock scheduler."
  - - meta
    - property: article:published_time
      content: "2026-09-28T00:00:00Z"
  - - link
    - rel: canonical
      href: https://hno.rexai.top/blog/node-option-variadic-design
---

# Four Node Policies, One Variadic Seam: A NodeOption Design Story

A graph engine grows policies the way a city grows traffic lights: slowly at
first, then all at once. Nodes need retries, then timeouts, then caching,
then tracing — and the obvious API, one policy struct with fourteen fields,
is exactly how a framework ends up with callers constructing

```go
AddNode(NodeConfig{
    Name: "fetch", Fn: fn, Retry: &RetryPolicy{...}, Timeout: &TimeoutPolicy{...},
    Cache: &CachePolicy{...}, Trace: &TracePolicy{...}, // ...and it keeps going
})
```

just to say "run this node." This is the story of the API we shipped instead
for HNO's graph engine — one variadic seam, zero-value semantics, and the
places where we deliberately chose *not* to validate.

## The seam: `AddNode(n, opts...)`

```go
g.AddNode(
    graph.NodeFunc("fetch", fetchFn),
    graph.WithRetry(graph.RetryConfig{MaxAttempts: 3, InitialDelay: 100 * time.Millisecond}),
    graph.WithTimeout(graph.TimeoutConfig{Timeout: 30 * time.Second}),
    graph.WithCache(graph.CacheConfig{TTL: 10 * time.Minute, Store: store}),
    graph.WithTrace(graph.TraceConfig{Enabled: true, Hook: logEvent}),
)
```

`func (g *Graph) AddNode(n Node, opts ...NodeOption) *Graph`. Variadic
options are an old Go idiom, but the property that matters here is
**retroactive**: every existing `AddNode(n)` call in every caller keeps
compiling and behaving identically. We had used the same move for the
graph-level `WithMaxConcurrency`, so `NodeOption` followed the house
precedent rather than inventing a second style.

The execution envelope the options plug into is fixed and ordered: cache
lookup (a hit skips execution), then up to N timeout-bounded attempts, then
a cache write on success, then trace emission. Retries live inside one
activation — they never consume step-limit budget, because the step valve
counts *dispatched activations*, and a retry is not a dispatch.

## One policy at a time

Naming provenance stated plainly: retry is modeled on adk's `retry.go`;
cache and timeout follow LangGraph's `CachePolicy` and `TimeoutPolicy` —
good designs, borrowed consciously. Trace is ours.

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

Exponential backoff from `InitialDelay` scaled by `BackoffFactor`, capped at
`MaxDelay`; `ShouldRetry` is the gate for another attempt (`nil` never
retries). One detail we care about: backoff waits are cancellation-aware —
`select` against `ctx.Done()` — so an abandoned run stops waiting the moment
it is abandoned, not after the nap.

### Timeout

```go
type TimeoutConfig struct {
    Timeout    time.Duration
    PerAttempt bool
}
```

`PerAttempt: false` installs **one budget for the whole node, retries
included** — the deadline exists before the first attempt. `PerAttempt: true`
mints a fresh deadline per attempt. The distinction is not academic: with a
30-second node budget, three attempts share 30 seconds; with per-attempt
budgets, they get 30 each. The test suite pins the *deadline values*, not
elapsed times, so it stays deterministic on a loaded CI box.

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

No default store ships, on purpose: the graph package depends on no cache
implementation, and adapting your backend is a two-method interface. Without
`KeyFunc`, the key is the SHA-256 of the input's `%#v` rendering — good
enough to be useful, explicit enough to be replaced.

### Trace

```go
type TraceConfig struct {
    Enabled   bool
    RedactIn  bool
    RedactOut bool
    Hook      func(NodeEvent)
}
```

`NodeEvent` carries `{Node, Attempt, Err, In, Out}` — one event per outcome,
on success *and* failure. `RedactIn`/`RedactOut` nil the payloads before
delivery, so a hook wired to a logging sink never sees the prompts.

## Zero value means semantics

Here is the design decision this article exists to defend. Policy configs
are **clamped by guards at read time, not validated at build time**:

| Declaration | Effective semantics |
|---|---|
| No `WithRetry` / `RetryConfig{}` | 1 attempt — `MaxAttempts < 1` clamps to 1 |
| `ShouldRetry: nil` | never retry |
| No `WithTimeout` / `Timeout <= 0` | no time limit |
| No `WithCache` / `Store: nil` | no caching |
| No `WithTrace` / `Enabled: false` / `Hook: nil` | no events |

"Zero value is a working configuration" is the Go idiom applied to policy:
*don't retry*, *don't limit*, *don't cache* are all real states a caller
wants, and each is exactly one zero value away. No new `Validate()` rules
were added for policies — zero, as a count, is part of the contract.

And yet the same engine **rejects** `WithStepLimit(0)` outright:

```
graph: WithStepLimit(0) is not a usable step limit: a positive number of steps is required
```

Why the asymmetry? Because a non-positive step limit has no readable meaning:
interpreted as "unlimited," it disarms the safety valve precisely on the
cyclic graphs that need it most; interpreted as "zero steps," it rejects
every run at the entry. Both readings are wrong, so the declaration is wrong.
Policy zero values, by contrast, each map to exactly one sensible state. Two
postures — clamp what is unambiguous, reject what is not — chosen per knob.
A single rule ("always validate" or "always clamp") would have made one of
these two knobs worse.

## The fail-closed cache

Only successful results enter the cache. A failing node writes nothing.

The alternative — caching errors to "avoid repeating failing work" — turns
one transient outage into a persistent one: the next run receives the cached
failure as if it were a conclusion. When the mutation harness flipped this
line (mutant m6: "failures also written to cache"), the suite went red on
exactly the test that pins it (D11, the failure-never-cached case). That is
the difference between a stated invariant and a tested one.

## Why trace lives on the consumer side

The engine is a single-consumer, zero-lock scheduler: producer goroutines run
nodes and send completions on a channel; one consumer owns all mutable state.
Trace events are emitted **by that consumer**, serially, after the outcome is
known — which buys three things:

1. The hook contract needs no synchronization. A plain function, no mutex, no
   channel of its own.
2. The zero-lock invariant survives — `sync.` count in the engine files stays
   at zero.
3. Events arrive in completion-processing order, which is the same order the
   scheduler itself believes in.

The cost is honest: hooks run on the scheduling path, so keep them fast, and
a slow hook slows every node completion. For heavy sinks, buffer inside your
hook.

## What the mutation matrix caught

The policies slice shipped with eight mutants, all killed:

| Mutant | Sabotage | Caught by |
|---|---|---|
| m1 | retry loop disabled (always break) | D1/D3/D5/D8 |
| m2 | `ShouldRetry` inverted | D1/D3/D4/D5/D8 |
| m3 | timeout clamping disabled | D6/D8 |
| m4 | `PerAttempt` degraded to a single deadline | D8 |
| m5 | cache lookup disabled | D9/D10/D11 |
| m6 | failures also cached | D11 |
| m7 | completion trace emission removed | D12/D13 |
| m8 | redaction branch removed | D13 |

One equivalence was registered rather than claimed: e1 (the failure-path
trace emission) — the harness could not distinguish it with a
behavior-observable test, so it is on record as equivalent rather than
silently "passing." m2 is the one I would have bet against: inverting a
predicate is the kind of bug that makes *some* tests pass harder and others
fail, and it still killed five test groups. m4 is the subtlest: everything
still times out, attempts still happen — only the *shape* of the deadlines is
wrong, and only the test that asserts deadline values notices.

## What this design does not prove

- **Producer-goroutine reentrancy is not addressed.** Policies execute inside
  the producer goroutine that runs the node; what happens when a node's own
  body spawns work that re-enters policy paths (e.g., a nested graph with the
  same cache store) is caller territory. The engine makes no reentrancy
  claims.
- **The cache seam ships no store.** "Adapt your backend in two methods" is a
  feature for footprint and a to-do for you; there is no Redis/Postgres
  adapter in the box, and TTL semantics are whatever your `SetAny` does with
  the argument.
- **`Jitter` exists in the config struct but the landed scheduler does not
  apply it yet.** We would rather document the field honestly today than
  pretend thundering-herd protection that is not wired.
- **Trace is node-level, not run-level.** Distributed tracing of full runs is
  a separate observability line; `NodeEvent` hooks are the node-scoped seam.

## Reproduce it

```bash
go test ./pkg/hno/graph -run TestP2G3_ -count=1 -race -v
grep -o 'sync\.' pkg/hno/graph/graph.go pkg/hno/graph/scheduler.go \
  pkg/hno/graph/policy.go pkg/hno/graph/durability.go | wc -l   # expect 0
```

The full policy walkthrough with runnable examples is the
[Node Policies guide](/guide/node-policies); signatures live in the
[Graph API reference](/api/graph).

## Continue reading

- [Previous: Building a LangGraph in Go — a Zero-Lock Graph Engine](/blog/zero-lock-graph-engine)
- [Graph Engine guide](/guide/graph-engine)
- [All blog articles](/blog/)
