---
title: Run Context
description: Correlate events and propagate context through hooks, tools, and model invocations.
---

# Run Context

HNO propagates a per-run context identifier through hooks, tools, and model calls. Streaming events include `run_context_id` for end-to-end correlation.

## SSE Events

The AgentOS `POST /api/v1/agents/{id}/run?stream_events=true` endpoint emits SSE events with:

- `run_start`, `reasoning`, `token`, `complete`, `error`
- Each event carries `run_context_id`

## Programmatic Usage

```go
import (
  "context"
  "github.com/rexleimo/agno-go/pkg/hno/agent"
)

ctx := agent.WithRunContext(context.Background(), "rc-123")
out, err := myAgent.Run(ctx, "hello")
```

Inside hooks or toolkit handlers:

```go
func preHook(ctx context.Context, in *hooks.HookInput) error {
  if id, ok := agent.RunContextID(ctx); ok { /* use id */ }
  return nil
}

fn := &toolkit.Function{ Name: "op", Handler: func(ctx context.Context, args map[string]interface{}) (any, error) {
  id, _ := agent.RunContextID(ctx)
  return map[string]any{"run_context_id": id}, nil
}}
```

## Notes

- If no ID is provided, the HTTP server injects one automatically.
- Context cancellation/timeouts are respected across hooks, tools, and model calls.

