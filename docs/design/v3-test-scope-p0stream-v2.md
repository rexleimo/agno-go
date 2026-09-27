# 测试范围契约 v2 — `work-p0-stream-loop-consolidation-v2`

- **workflowActivationId**: `e23e2188-4031-49c1-8d83-202573ecec6b`
- **Command**: `rex-test-design` / `stageId: design-tests` / `activationId: a3b23f0f-abd6-4424-abfa-913421a70ce9`
- **重做对象**: `artifact:docs/design/v3-test-scope-p0stream.md`（v1 契约）与 `artifact:docs/design/v3-p0stream-green.md`（v1 实现）
- **票面来源**: `artifact:docs/design/v3-delivery-ticket.json#work-p0-stream-loop-consolidation`

> 本契约界定本轮测什么、从哪里观察。不得由实现反推。

---

## 0. 为什么有 v2：v1 契约的两条反向条款

v1 实现未达成票面判据，根因在契约本身写进了与交付票**方向相反**的条款，实现只是忠实地满足了契约：

| v1 条款 | v1 原文 | 与票面的冲突 | v2 处置 |
|---|---|---|---|
| v1 §2 N2 | 「**不给** `pkg/hno/runner` 加流式能力」 | 票面判据 1：`RunStream` 内部**改走 `pkg/hno/runner`**，不再使用 stream.go 自有循环 | **废除**。改为 B1：内核持有唯一循环 |
| v1 §7 C8 / §9.6 | 「`RunStreamResult` **未**新增 `StopReason`」被列为**通过**判据，且禁止加在该类型上 | 票面判据 2：`RunStreamResult` **暴露** `StopReason` | **废除**。改为 B5：经 `RunStreamResult` 暴露，且要求无数据竞争 |
| v1 §2 N7 | 「不改 `Agent.Run`（同步路径）」 | 不冲突，但使 ToolCallLimit 的 per-run 计数缺陷在两条路径同时保留 | **收窄**：同步路径仅允许接受 per-run 计数修正（B2），其余行为不动 |

结论：v1 把「收口循环」降级为「给重复循环补齐三项能力」，且用一个自造约束把票面要求标成禁止项。v2 以交付票为唯一范围来源。

## 1. 公共入口与观察面

```go
func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error)          // run.go:16
func (a *Agent) RunStream(ctx context.Context, input string) (*RunStreamResult, error) // stream.go:40

type RunStreamResult struct {   // agent.go:89
    Events <-chan run.BaseRunOutputEvent
    Done   <-chan RunStreamDone
}
type RunStreamDone struct {     // agent.go:80
    Output     *RunOutput
    Err        error
    StopReason string
}
```

**本轮新增/变更的观察面**：

1. `RunStreamResult.StopReason() string` — 方法而非字段。理由：`RunStreamResult` 在流开始时就返回，彼时终止原因未知；用字段会制造数据竞争（v1 §9.6 的担忧成立，但结论「所以不加」是错的，正确做法是加**受互斥保护、在 `Done` 发送前写入**的访问器）。约定：**仅在从 `Done` 收到终值之后调用**。
2. `RunOutput.StopReason`（`agent.go:75`，已存在）在流式路径同样填充；两条路径填同一个内核返回值。
3. `RunStreamDone.StopReason` 保持存在，取值与上述一致。

**必须保持不变的观察面**：`Done.Output.Messages`、`Done.Output.Content`、`Done.Err`、`Events` 的内容与顺序、`stream_aggregator.go` 的增量聚合语义、`streamMockModel.Invoke` 的 `"unexpected sync invoke"` 守卫（见 §5）。

## 2. 用户目标

> 让 `RunStream` 与 `Run` 共用 `pkg/hno/runner` 的**同一个循环状态机**：tool 循环只有一处实现；流式路径通过内核的回合调用接缝完成「流式调用 + 聚合 + 事件扇出」。对外可观察结果：同一场景下两条路径的**完整消息序列**与**终止原因**一致；`ToolCallLimit` 按「每次运行」计数；`StopReason` 可从 `RunStreamResult` 读到，且失败路径不再谎报 `no_tool_calls`。

## 3. 明确非目标

| # | 非目标 | 理由 |
|---|---|---|
| N1 | 不改 `stream_aggregator.go` 的聚合语义与事件顺序 | 票面硬判据（`AggregateResponseStream` 三个测试逐字不变） |
| N2 | 不实现 `HITLBlocked` 流式语义 | P0b 已裁定删除该偶然复杂度；内核保留枚举值但不接入 |
| N3 | 不做 P4 `StreamMode` 七模式、不做图引擎/事件化/Store | 分属 `work-p4-streammode`、P1/P5 |
| N4 | 不给流式路径接入响应缓存 | 同步路径缓存属既有行为，本轮不扩大 |
| N5 | 不改错误返回**形状** | `Run` 返回 `(nil, err)`、`RunStream` 经 `Done.Err`，是既有的、票面未要求变更的差异 |
| N6 | 不改 `MaxLoops` 的错误码与语义 | 既有行为（`run.go:180` 的 MaxLoops 判定保持） |
| N7 | 不引入新依赖 | — |

## 4. 范围内行为

| ID | 行为 | 观察面 | v1 状态 |
|---|---|---|---|
| **B1** | 循环唯一性：tool 循环只存在于内核；`stream.go` 不再自有循环 | 结构判据 §7-C1 + B6 的六场景 parity 行为判据 | ❌ v1 未做（`stream.go:91` 仍在） |
| **B2** | `ToolCallLimit` 为 **per-run** 计数：历史里的 tool 消息不占用本轮额度 | `Run` 与 `RunStream` 的 `Output.Messages` 中**本次运行新增**的 tool 消息 | ❌ 两条路径都按历史计数（v1 §3 反而写明「跨轮累计」含历史） |
| **B3** | 流式截断回注 skipped tool 消息，`ToolCallID` 与原文案配对 | `RoleTool` 消息的 `ToolCallID`+`Content` | ✅ v1 已交付，保持 |
| **B4** | 流式尊重工具 `StopLoop`，该轮后不再调用模型 | fake `InvokeStream` 次数 + 消息保留 | ✅ v1 已交付，保持 |
| **B5** | `StopReason` 经 `RunStreamResult` 暴露；失败路径不得上报 `no_tool_calls` | `RunStreamResult.StopReason()` + `Done.StopReason` | ❌ v1 放在 `RunStreamDone`，且错误路径硬编码默认值（`stream.go:87`） |
| **B6** | 同一场景下 sync 与 stream 的**完整消息序列**（角色顺序 + `ToolCallID` + `Content`）逐条一致 | 两个 `Output.Messages` 对比 | ⚠️ v1 只比 `RoleTool` 子集，测不出 B1/B2/B5 的分叉 |
| **B7** | 流式事件顺序与分块边界不因接入内核而改变 | 既有 stream 测试 + 两轮场景的 `Events` 序列 | ✅ v1 未触及，须继续为绿 |
| **R1** | 既有测试断言零改动 | 既有 11 runner + 既有 agent/contract 测试 | ✅ 基线 |

### 精确期望值（依据代码，非推测）

- skipped 文案唯一来源 `pkg/hno/runner/tool_batch.go:5` 常量 `tool call limit reached; call not executed`；`run.go:155` 的字面量副本本轮消除（B2 同属内核单一实现的必然结果）。
- 真实工具结果经 `toolkit.FormatResult`（`json.Marshal`），带引号，如 `"\"A\""`。
- 终止原因取值以 `pkg/hno/runner/stop.go:10-25` 为准：`no_tool_calls` / `limit_reached` / `stop_after_tool_call` / `hitl_blocked` / `requirements_pending` / `cancelled`。
- 内核在 `remaining <= 0` 时终止且不追加任何 tool 消息（`DecideToolBatch` 返回 `(nil,nil,true)`）。
- **计数口径（本轮变更）**：`ToolCallLimit` 在内核内按**本次运行**累计（本轮真实执行数 + 被跳过数），`Run` 传入的历史消息中的 `RoleTool` **不占额度**。v1 §3 把「跨轮累计」写成含历史是缺陷来源，本轮作废该口径。
- 失败路径的终止原因（B5-b 落地要求，本轮在 `pkg/hno/runner/stop.go` 新增两个枚举值）：
  - 模型调用失败 → `model_failure`（新）
  - 工具执行失败 → `tool_failure`（新）
  - 上下文取消 → `cancelled`（既有，`TestRunnerCancellation` 断言保持）
  - 任何失败路径**都不得**上报 `no_tool_calls`，也不得留空。

## 5. 范围内/外的测试缝

| 允许 | 禁止 |
|---|---|
| `pkg/hno/agent/` 下**新增**测试文件（本轮：`p0stream_v2_red_test.go`） | 修改/删除/跳过既有测试的任何一行断言 |
| `pkg/hno/runner/` 下**新增**测试文件（本轮：`tool_limit_run_test.go`） | 断言 runner 私有字段、状态机内部变量或 goroutine 时序 |
| 以 fake model 的 `InvokeStream` 次数、`Output.Messages`、`StopReason()`、`Events` 序列作为断言对象 | 以「结构上共用一个循环」替代 B6 的行为对照 |
| 用 `grep` 做 B1 的**结构**判据，并同时要求 B6 行为判据成立 | 只靠 `grep` 宣布收口完成（v1 的教训：判据可被绕过，行为分叉照样绿） |
| 在 agent 包内自建流式 fake（含会返回 error 的 `InvokeStream`） | 放宽 `streamMockModel.Invoke` 的 `"unexpected sync invoke"` 守卫 |

**现成守卫（必须保持）**：`pkg/hno/agent/stream_tool_loop_test.go` 的 `streamMockModel.Invoke` 返回 `errors.New("unexpected sync invoke")`。B1 落地后，流式路径经内核回合接缝调用的仍是 `InvokeStream`，该守卫**继续为绿**即为「接入内核却没把流式退化成同步调用」的行为证据。若有人让内核在流式模式下走 `Invoke`，既有测试立即失败。

## 6. 验收行为 → 公共入口 → 可观察断言 映射

| 行为 | 公共入口 | 场景 setup | 可观察断言 |
|---|---|---|---|
| **B2-sync** | `Run` | fake 第 1 轮返回 1 个 tool call，`ToolCallLimit = 1`；**事先向 Memory 预置 2 条 `RoleTool` 历史消息** | 本次运行新增的 tool 消息内容为**执行结果**（带引号），而非 skipped 文案；`StopReason == "no_tool_calls"`（第二轮 fake 返回文本） |
| **B2-stream** | `RunStream` | 同上，流式 fake | 同上，经 `Done.Output.Messages` 观察 |
| **B3** | `RunStream` | 单轮 3 个 tool call（`c1`/`c2`/`c3`），`ToolCallLimit = 2` | `RoleTool` 恰 3 条：`c1`→`"\"A\""`、`c2`→`"\"B\""`、`c3`→skipped 文案；`StopReason() == "limit_reached"`；`Done.Err == nil` |
| **B4** | `RunStream` | 工具 `p0s_a` 声明 `StopLoop`，fake 第 2 轮仍会返回内容 | `InvokeStream` 次数 `== 1`；`c1` 结果消息保留；`StopReason() == "stop_after_tool_call"` |
| **B5-a** | `RunStream` | 首轮返回最终文本 | `result.StopReason() == "no_tool_calls"` 且 `Done.StopReason` 相同且 `Output.StopReason` 相同 |
| **B5-b** | `RunStream` | fake 的 `InvokeStream` 直接返回 error | `StopReason() == "model_failure"`（不得为 `no_tool_calls` 或空）；`Done.Err` 非 nil；`Done.Output == nil` |
| **B6** | `Run` + `RunStream` | 三套场景各自同构：① 3 call + limit 2 ② 跨轮耗尽 ③ StopLoop | 两条路径**本次运行新增消息**的 `(Role, ToolCallID, Content)` **有序**序列逐条相等；`StopReason` 相等 |
| **B7** | `RunStream` | 两轮：第 1 轮 tool call、第 2 轮文本，各含多个 content chunk | `Events` 内容事件按 chunk 顺序拼接等于最终 `Output.Content` 的前缀序列；`sequence` 单调递增；既有 3 个 aggregator 测试与 2 个 tool-loop 测试零改动全绿 |
| **R1** | — | 既有测试 | 断言零改动 + 全绿 |

## 7. 完成判据

| # | 判据 | 判定方式 |
|---|---|---|
| C1 | **结构**：`stream.go` 不再持有 tool 循环 | `grep -c 'for loopCount' pkg/hno/agent/stream.go` 为 0；`grep -n 'runner\.' pkg/hno/agent/stream.go` 命中内核构造；票面命令 `grep -c 'for {' pkg/hno/agent/stream.go` 不再命中循环体 |
| C2 | **行为**：B2-sync/B2-stream/B3/B4/B5-a/B5-b/B6/B7 全部存在且通过 | 零退出 receipt（聚焦命令 §10） |
| C3 | 既有测试断言零改动 | `git diff` 中既有 `_test.go` 无改动行；全绿 receipt |
| C4 | `stream_aggregator.go` 零改动 | `git diff --stat -- pkg/hno/agent/stream_aggregator.go` 为空 |
| C5 | `streamMockModel.Invoke` 守卫未被放宽 | `grep` 确认 `"unexpected sync invoke"` 仍在，且该测试为绿 |
| C6 | ToolCallLimit 的 skipped 文案只有内核一处 | `grep -rn 'tool call limit reached' pkg/hno/` 仅命中 `runner/tool_batch.go` 与其测试 |
| C7 | `StopReason` 暴露无数据竞争 | `go test -race ./pkg/hno/agent/...` 零退出 receipt |
| C8 | `RunStreamResult` 不新增可写字段（只加受保护的方法） | `git diff pkg/hno/agent/agent.go` 中 struct 仅通道字段 |
| C9 | 不止编译通过：C2 的断言实际执行并 PASS | `-v` 输出含各用例 PASS |

## 8. 最小纵向切片

**切片：B2-stream（带历史的 `ToolCallLimit` per-run 计数）**，随后 B5-b，最后 B6。

**为何足以代表本轮目标**：

1. 它**同时**要求循环在内核内（否则得改 stream 的计数逻辑，等于 v1 的重复实现）与内核计数语义正确，一条测试就能区分「真收口」与「假收口」。
2. v1 的实现对它必然失败：`stream.go:154` 从整个 Memory 数 tool 消息，预置历史即误判额度耗尽——这是 v1 无法用接线蒙过去的断言。
3. 期望值完全确定（执行结果 vs skipped 文案），没有「把断言写成当前输出」的空间。
4. B5-b 紧随其后：它抓住 v1 第二个失败点（错误路径硬编码 `no_tool_calls`），且不依赖 B1。
5. B6 必须最后做：它需要在同一内核上已有两条路径。

## 9. 禁止假通过条款

任一发生即判本轮失败：

1. 删除 / `t.Skip` / 注释 / 放宽任何既有断言（含 `"unexpected sync invoke"` 守卫）
2. 保留 `stream.go` 的自有循环，仅把 v1 的实现换个名字宣布收口
3. 用「两条路径都从历史计数」的自洽来实现 B6 parity（parity 必须建立在 per-run 语义上，即两边都改对）
4. 只比 `RoleTool` 子集就宣布 B6 完成
5. 用 `grep` 结构判据替代 B6 行为判据（C1 与 C2 必须同时成立）
6. 把 `StopReason` 作为 `RunStreamResult` 的**可写字段**暴露（竞态）
7. 在错误/取消路径上报 `no_tool_calls` 作为终止原因
8. 为通过测试修改 `stream_aggregator` 的聚合或事件顺序
9. 复活 `HITLBlocked` 流式语义
10. 借本轮扩大 P4 StreamMode 或缓存范围

## 10. 聚焦测试命令与回执

```bash
# RED / GREEN 共用
go test ./pkg/hno/agent/... ./pkg/hno/runner/... -run 'P0SV2|P0S|ToolLimitRun' -count=1

# 邻近回归
go test ./pkg/hno/agent/... ./pkg/hno/runner/... -count=1
go test ./internal/session/contract/... -count=1
go test -race ./pkg/hno/agent/... -count=1

# B1 结构判据
grep -c 'for loopCount' pkg/hno/agent/stream.go
```
