---
title: "Human-in-the-Loop for Graphs: Interrupt, Persist, Resume — Across Processes"
description: "How the v3 engine turns a node's RequestInterrupt into a durable suspension with schema-checked, idempotent resume — and why the suspension is never a half-done Result."
date: 2026-09-29
lastUpdated: 2026-09-29
author: HNO Team
category: Go engineering
tags:
  - Go
  - HITL
  - graph
  - durability
  - agent workflows
---

Every agent framework promises "pause for human approval." Most implement it
as a callback that blocks a goroutine, or a half-finished result the caller
has to know how to interpret. Both approaches fall apart the moment the
process restarts or the approver takes a lunch break.

This post is about how the v3 engine landed the third attempt: a suspension
that is durable state, a resume that is schema-checked and idempotent, and a
cross-process path that survives the thing approvals always assume — time.

## The shape that fails: half results and blocking callbacks

Two anti-shapes dominate. The blocking callback holds a goroutine and a
context hostage for as long as the human takes; cancel the run and the
approval question evaporates. The half-result returns a `Result` with a flag
set — but a `Result` that only traversed part of the graph is exactly the
"conclusion from a graph the caller never wrote" failure this engine's
build-time validation work spent twenty-one slices eliminating.

So the engine draws a hard line: **a suspended run is not a `Result`.**
`Run` returns `(nil, err)` where `err` satisfies `errors.Is(err,
ErrSuspended)`, and `errors.As` yields a `*Suspension` carrying every pending
interrupt and every node that completed before the halt. Callers cannot
accidentally treat a waiting run as a finished one — the type system of the
error chain makes the distinction un-ignorable.

## RequestInterrupt: the smallest possible node surface

Nodes do not get a new interface or a changed signature. A node that needs a
human returns a sentinel:

```go
return nil, graph.RequestInterrupt(graph.Interrupt{
    InterruptID: "approve-release",
    ResponseSchema: map[string]any{
        "type":     "object",
        "required": []string{"approved"},
        "properties": map[string]any{
            "approved": map[string]any{"type": "boolean"},
        },
    },
    Mode: graph.ResumeRerun,
})
```

Two details carry most of the weight. First, `RequestInterrupt` is a sentinel
error, so the retry envelope deliberately does **not** swallow it — a node
with `MaxAttempts: 5` that interrupts executes exactly once and suspends.
Second, each `Interrupt` carries its own `Mode`: `ResumeRerun` re-runs the
node with the response reachable via `InterruptResponse(ctx, id)`;
`ResumeHandoff` never executes it again — the response becomes its output and
flows to successors. Approval flows want handoff; "regenerate with feedback"
flows want rerun. Per interrupt, not per graph.

## Resume: checked, idempotent, honest about failure

`Resume(ctx, responses)` enforces three semantics, each with a mutation-tested
guarantee:

1. **Schema-checked.** Responses are validated against the interrupt's
   schema — an honest subset (`type`, `required`, `properties.<name>.type`),
   not a pretend JSON Schema. A bad response returns `ErrInvalidResponse`
   and leaves the suspension intact. The caller fixes the answer and retries;
   nothing was consumed.
2. **Idempotent.** `ErrNothingToResume` for double-resume and for
   never-suspended alike. The tempting shortcut — return an empty `Result` —
   would be the same "silent no-conclusion" shape the join-barrier work
   flagged, so it is rejected by name instead.
3. **Two modes, per interrupt.** One resume can answer a rerun interrupt and
   a handoff interrupt in the same call; a parallel fan-out can suspend two
   branches at once and resume both.

## Durability: the suspension is checkpoint data

The suspension rides the durability seam from the previous engine release:
suspending commits an `EntryInterrupt` entry through the same
`Checkpointer` that records completed nodes, under whichever tier
(Sync/Async/Exit) the graph declared. A failing sink does not mask the
suspension — both errors are `errors.Is`-able. The step budget continues
across resume, because a run interrupted for six hours is still one logical
run.

For cross-process resume, a session sidecar (`pkg/hno/session/sidecar`) plus
a small bridge captures the suspension into storage without touching the
session's outward JSON — byte-identical, verified against the marshal
anchors — and a fresh process rebuilds the graph and installs the stored
suspension before `Resume`. The end-to-end test interrupts, snapshots,
"restarts", and resumes in both modes.

## What the mutation matrix says

The slice's 13-variant matrix kills every recovery guarantee: removing the
capture turns interrupts into plain errors; removing schema validation lets a
bad response consume the suspension; inverting the retry guard burns retry
budget on a question no one answered yet; resetting the budget per resume
turns a six-hour approval into a fresh 1000-step allowance. Two equivalent
mutants are registered honestly — including one coverage gap (stop-status
spans) queued for follow-up.

## Where to read the code

- Guide: [Human-in-the-Loop](/guide/human-in-the-loop)
- API: [Graph — HITL](/api/graph)
- Durability seam: [Graph Durability](/advanced/graph-durability)
- Evidence: `docs/design/v3-p5-graph-g7-green.md`,
  `docs/design/v3-p5-s28-green.md`, and the slice contracts they cite.
