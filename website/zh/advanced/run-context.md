---
title: 运行上下文
description: 通过 hooks、工具与模型调用传播 run_context_id，并在事件中关联。
---

# 运行上下文（Run Context）

HNO 在一次运行中传播上下文标识，流式事件会携带 `run_context_id` 便于端到端关联。

## SSE 事件

`POST /api/v1/agents/{id}/run?stream_events=true` 会输出 `run_start`、`reasoning`、`token`、`complete`、`error` 等事件，均包含 `run_context_id`。

## 代码使用

```go
ctx := agent.WithRunContext(context.Background(), "rc-123")
out, err := myAgent.Run(ctx, "hello")
```

在 Hook 或 Toolkit 中读取：

```go
id, _ := agent.RunContextID(ctx)
```

## 说明

- 未显式传入时，HTTP 层会为每次运行自动注入。
- 取消/超时会沿 Context 传递到 hooks、tools 与模型调用。

