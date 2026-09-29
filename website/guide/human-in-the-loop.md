# Human-in-the-Loop - Interrupt and Resume

Pause a running graph to ask a human, persist the suspension, and resume
in-process or in a brand-new process — with schema-checked responses,
idempotent resume, and two recovery modes.

---

## What is HITL here?

A node can halt the run by returning a sentinel from
[`RequestInterrupt`](#requestinterrupt-interrupt-error). The engine turns that
into a suspension instead of a failure: `Run` returns `(nil, err)` where
`err` wraps [`ErrSuspended`](#sentinels), and the
[`*Suspension`](#the-suspension-value) carries every pending interrupt plus
the state completed so far. Nothing half-done is handed back — a suspended run
is never a `Result`, and never a plain error.

```go
g := graph.New().
    AddNode(graph.NodeFunc("draft", func(ctx context.Context, in any) (any, error) {
        return "draft-v1", nil
    })).
    AddNode(graph.NodeFunc("approve", func(ctx context.Context, in any) (any, error) {
        return nil, graph.RequestInterrupt(graph.Interrupt{
            InterruptID:    "approve-release",
            Message:        "Release to production?",
            ResponseSchema: map[string]any{
                "type": "object",
                "required": []string{"approved"},
                "properties": map[string]any{
                    "approved": map[string]any{"type": "boolean"},
                },
            },
            Mode: graph.ResumeRerun,
        })
    })).
    AddNode(graph.NodeFunc("ship", func(ctx context.Context, in any) (any, error) {
        approved, _ := graph.InterruptResponse(ctx, "approve-release")
        if approved.(bool) {
            return "shipped", nil
        }
        return "held", nil
    })).
    AddEdge("draft", "approve").
    AddEdge("approve", "ship").
    SetEntry("draft").
    SetOutput("ship")

res, err := g.Run(ctx, nil)
// err wraps graph.ErrSuspended; errors.As gives you *graph.Suspension.
```

## The Suspension value

`Run` returns `(nil, err)` on suspension. Inspect it with the standard
errors chain:

- `errors.Is(err, graph.ErrSuspended)` — this run is waiting on humans.
- `errors.As(err, &suspension)` — read `suspension.Interrupts` (the pending
  [`Interrupt`](#interrupt) list) and `suspension.Completed` (nodes that
  finished before the halt).

A suspension is durable state, not a courtesy message. Combined with
[Durability](/advanced/graph-durability) (`WithDurability` +
`WithCheckpointer`), every suspension is committed as an `EntryInterrupt`
checkpoint entry alongside the completed-node entries, so the waiting state
survives a process restart.

## Resume: schema-checked, idempotent, two modes

```go
res, err := g.Resume(ctx, map[string]any{
    "approve-release": map[string]any{"approved": true},
})
```

`Resume` enforces the three semantics the design pinned:

1. **Schema-checked.** Each response is validated against that interrupt's
   `ResponseSchema` (an honest subset: `type: object`, `required`,
   `properties.<name>.type`). A mismatch returns an error wrapping
   `ErrInvalidResponse` and keeps the suspension intact — fix the response
   and retry.
2. **Idempotent.** Resuming with nothing to resume — never suspended,
   already completed — returns `ErrNothingToResume`. Repeated calls are
   no-ops, never a second execution.
3. **Two modes, per interrupt.** `ResumeRerun` re-executes the waiting node
   with the response reachable via
   [`InterruptResponse`](#interruptresponse-ctx-interruptid-any-any-bool).
   `ResumeHandoff` does not run the node again; the response becomes its
   output and flows to successors.

The response set must cover exactly the pending interrupts — a missing ID or
an unknown ID is rejected by name. Resume may suspend again (a re-suspension
chain is a first-class shape), and the step budget continues across the whole
logical run.

## Cross-process resume

The engine keeps `Resume` working on the same `*Graph`. To resume in a fresh
process, pair it with the session sidecar (`pkg/hno/session/sidecar`) and the
bridge (`internal/hitlbridge`): capture the suspension into the sidecar
store, rebuild the graph from its declaration, and install the stored
suspension before calling `Resume`. The sidecar keeps your `Session` JSON
byte-identical — it never routes through `Storage.Update`.

## Sentinels

| Sentinel | Meaning |
|---|---|
| `ErrSuspended` | The run halted awaiting human response (carried by `*Suspension`). |
| `ErrNothingToResume` | Nothing to resume: never suspended or already completed. |
| `ErrInvalidResponse` | A response failed its `ResponseSchema`; the suspension is preserved. |

Two runtime errors are also pinned: an empty `InterruptID` and a duplicate
`InterruptID` among concurrently pending interrupts are both rejected with a
node-attributed error.

## What Is Deliberately Not Here Yet

- The schema subset is exactly `type` + `required` + `properties.<name>.type`.
  Full JSON Schema (nesting, `anyOf`, patterns) is out of scope.
- The events bridge (emitting `run` protocol events from engine suspensions)
  is a separate, optional slice.

## Related Pages

- [Durability](/advanced/graph-durability) — the checkpoint seam suspensions
  ride on.
- [Graph Engine](/guide/graph-engine) — routing, validation, valves.
