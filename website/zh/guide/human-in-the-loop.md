# Human-in-the-Loop - 中断与恢复

让运行中的图停下来问人、把挂起持久化，然后在同进程或全新进程里恢复——
响应带 schema 校验、恢复幂等、双档恢复语义。

---

## 这里的 HITL 是什么？

节点通过返回 [`RequestInterrupt`](#requestinterrupt-interrupt-error) 的哨兵
错误来暂停运行。引擎把它变成「挂起」而不是「失败」：`Run` 返回 `(nil, err)`，
`err` 可 `errors.Is` 到 [`ErrSuspended`](#sentinels)，[`*Suspension`](#the-suspension-value)
携带全部待答中断与已完成状态。不会交出半截结论——挂起既不是 `Result`，
也不是普通错误。

```go
g := graph.New().
    AddNode(graph.NodeFunc("draft", func(ctx context.Context, in any) (any, error) {
        return "draft-v1", nil
    })).
    AddNode(graph.NodeFunc("approve", func(ctx context.Context, in any) (any, error) {
        return nil, graph.RequestInterrupt(graph.Interrupt{
            InterruptID:    "approve-release",
            Message:        "要发布到生产环境吗？",
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
// err 包着 graph.ErrSuspended；errors.As 可取出 *graph.Suspension。
```

## Suspension 值

挂起时 `Run` 返回 `(nil, err)`。用标准 errors 链检查：

- `errors.Is(err, graph.ErrSuspended)` —— 这次运行在等人。
- `errors.As(err, &suspension)` —— 读 `suspension.Interrupts`（待答的
  [`Interrupt`](#interrupt) 清单）与 `suspension.Completed`（挂起前已完成
  的节点）。

挂起是持久化状态，不是客套话。配合 [Durability](/zh/advanced/graph-durability)
（`WithDurability` + `WithCheckpointer`），每次挂起都会作为 `EntryInterrupt`
检查点条目与已完成节点条目一起提交，等待状态可跨进程重启存活。

## Resume：schema 校验、幂等、双档

```go
res, err := g.Resume(ctx, map[string]any{
    "approve-release": map[string]any{"approved": true},
})
```

`Resume` 强制三条定死的语义：

1. **schema 校验。** 每个响应按对应中断的 `ResponseSchema` 校验（诚实的
   子集：`type: object`、`required`、`properties.<name>.type`）。不匹配返回
   包着 `ErrInvalidResponse` 的错误且挂起原样保留——修正响应后重试即可。
2. **幂等。** 无处可恢复时（从未挂起、已完成）返回 `ErrNothingToResume`。
   重复调用是 no-op，绝不会引发第二次执行。
3. **每个中断各选一档。** `ResumeRerun` 让等待节点带着响应重新执行（响应经
   [`InterruptResponse`](#interruptresponse-ctx-interruptid-any-any-bool) 可
   取）；`ResumeHandoff` 不再执行该节点，响应直接成为它的输出流向后继。

响应集必须恰好覆盖全部待答中断——缺 ID 或多给未知 ID 都会被点名拒绝。
Resume 可以再次挂起（重悬链是一等形状），步数预算跨整个逻辑运行累计。

## 跨进程恢复

引擎保证 `Resume` 在同一个 `*Graph` 上工作。要在全新进程恢复，配 合会话
侧车（`pkg/hno/session/sidecar`）与桥接层（`internal/hitlbridge`）：把挂起
捕获进侧车存储、按声明重建图、装回存储的挂起态后再调 `Resume`。侧车保证
你的 `Session` JSON 逐字节不变——它绝不路由过 `Storage.Update`。

## 哨兵错误

| 哨兵 | 含义 |
|---|---|
| `ErrSuspended` | 运行已挂起等人（由 `*Suspension` 携带）。 |
| `ErrNothingToResume` | 无处可恢复：从未挂起或已完成。 |
| `ErrInvalidResponse` | 响应未通过 `ResponseSchema`；挂起保留。 |

两条运行期错误也已钉死：空 `InterruptID`、并发待答中断里的重复
`InterruptID`，都会被可定位到节点的错误拒绝。

## 刻意还没有的

- schema 子集就是 `type` + `required` + `properties.<name>.type`。完整
  JSON Schema（嵌套、`anyOf`、pattern）不在范围。
- 事件桥接（把引擎挂起发射成 `run` 协议事件）是独立的可选片。

## 相关页面

- [Durability](/zh/advanced/graph-durability) —— 挂起所骑的检查点缝。
- [Graph Engine](/zh/guide/graph-engine) —— 路由、校验、安全阀。
