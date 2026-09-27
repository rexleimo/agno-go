# 测试范围契约 — `work-p0-stream-loop-consolidation`

- **workflowActivationId**: `f166216c-061d-47ec-b149-b78c3e02d507`
- **Command**: `rex-test-design` / `stageId: design-tests`
- **范围来源**: `artifact:docs/design/v3-p0stream-minimal-construction.md`（最小构造裁定：方案 B）
- **基线回执**: `receipt:039e7fdb-7673-4731-b31e-d0838e82374d`（exit 0）

> 本契约界定本周期测什么、从哪里观察。不得由实现反推。

---

## 0. 公共入口与观察面

```go
func (a *Agent) RunStream(ctx context.Context, input string) (*RunStreamResult, error)

type RunStreamResult struct {   // agent.go:88
    Events <-chan run.BaseRunOutputEvent
    Done   <-chan RunStreamDone
}
type RunStreamDone struct {     // agent.go:80
    Output *RunOutput
    Err    error
}
```

**本周期新增观察面**：`RunStreamDone.StopReason`（按最小构造 §3 第 4 项；**不加在 `RunStreamResult` 上**，因后者在流开始时即返回，值那时未知）。

**既有且必须保持的观察面**：`Done.Output.Messages`（`a.Memory` 为真相源）、`Done.Output.Content`、`Done.Err`、模型 `InvokeStream` 的调用次数、`Events` 的内容与顺序。

## 1. 用户目标

> 通过 `RunStream` 的公共入口，让流式路径获得与 `Run` 相同的三项能力：`ToolCallLimit` 截断、工具 `StopLoop` 主动终止、`StopReason` 可观察；且不弱化 `stream_aggregator` 的既有聚合语义与事件顺序。

## 2. 明确非目标

| # | 非目标 | 理由 |
|---|---|---|
| N1 | **不修改** `stream_aggregator.go` 的聚合语义与事件顺序 | 交付票硬判据；聚合器仅 44 行且已有 3 个测试钉住 |
| N2 | **不给 `pkg/hno/runner` 加流式能力** | 最小构造裁定：runner 无流式支持，强行统一会摧毁 `RunStream` |
| N3 | **不改** `runner` 状态机与 11 个既有 runner 测试 | 保持爆炸半径受控 |
| N4 | **不实现** `HITLBlocked` | 与 P0b 一致，已删除的偶然复杂度 |
| N5 | 不改 `MaxLoops` 语义与错误码 | 既有行为 |
| N6 | 不引入新依赖 | — |
| N7 | 不改 `Agent.Run`（同步路径） | P0b 已交付，本周期只补流式 |

## 3. 范围内行为

| ID | 行为 | 观察面 | 当前 |
|---|---|---|---|
| **N1** | 流式路径按 `Config.ToolCallLimit` 截断，回注 skipped tool 消息 | `Done.Output.Messages` 的 `RoleTool` `ToolCallID` 与 `Content` | ❌ 能力缺失 |
| **N2** | 流式路径尊重工具 `StopLoop`，该次工具调用后终止且不再回调模型 | 模型 `InvokeStream` 调用次数 + `Done` | ❌ 能力缺失 |
| **N3** | `RunStreamDone.StopReason` 可观察 | `RunStreamDone.StopReason` | ❌ 字段不存在 |
| **R1** | 等价性：既有 43 agent + 11 runner + 9 contract 测试全绿且断言零改动 | 既有测试 | ✅ 基线 `receipt:039e7fdb` |

### 精确期望值（依据 `runner.go` 与 `stop.go`，非推测）

- 截断时跳过的调用产生 `{Role: RoleTool, ToolCallID: <原始ID>, Content: "tool call limit reached; call not executed"}`
- 该消息由 runner/策略**直接构造，不经 `FormatResult`**，故为裸文案
- 真实执行结果经 `toolkit.FormatResult`（`json.Marshal`），字符串**带引号**，如 `"\"result1\""`
- `StopReason` 取值：`"limit_reached"` / `"stop_after_tool_call"` / `"no_tool_calls"`（`stop.go:10,13,16`）
- `ToolCallLimit` 为**跨轮累计**；`remaining <= 0` 时终止且**不再追加任何 tool 消息**

## 4. 范围外行为

`work-p0-loop-consolidation` 的其余循环收口；图引擎、事件化、StreamMode 七模式（分属 P1/P4/P5）。

## 5. 允许修改的测试缝

| 允许 | 禁止 |
|---|---|
| `pkg/hno/agent/` 下**新增**测试文件（如 `p0stream_red_test.go`） | 修改既有测试的任何一行 |
| 在 agent 包内自建流式 fake model | 断言 `runner` 私有字段或内部状态机 |
| 断言 `Done`、`Events`、fake 的 `InvokeStream` 次数 | 断言 goroutine/channel 内部时序 |
| 以「sync 与 stream 消息序列一致」的对照断言 | 以内部结构共用替代行为对照 |

**必须复用而非重造的既有事实**：既有 `streamToolLoopTest` 的 `streamMockModel` 已把 `Invoke` 实现为返回 `errors.New("unexpected sync invoke")`。这是**一道现成的守卫** —— 若有人把流式路径改接 `runner`（它调 `Invoke`），既有测试会立刻失败。**不得放宽或绕过它。**

## 6. 验收行为 → 公共入口 → 可观察断言 映射

| 行为 | 公共入口 | 场景 setup | 可观察断言 |
|---|---|---|---|
| **N1** | `RunStream` | 流式 fake 第 1 轮流式返回 **3 个** tool call（`c1`/`c2`/`c3`）；`Config.ToolCallLimit = 2` | `Done.Output.Messages` 中 `RoleTool` 恰 **3** 条；逐条身份+内容校验：`c1`→`"\"r1\""`、`c2`→`"\"r2\""`、`c3`→`"tool call limit reached; call not executed"`；`Done.StopReason == "limit_reached"`；`Done.Err == nil` |
| **N1b** | `RunStream` | 跨轮耗尽：上限 2，第 2 轮仍有 tool call | `RoleTool` 总数仍为 2，**无新增 tool 消息**；`StopReason == "limit_reached"` |
| **N2** | `RunStream` | 1 个工具声明 `StopLoop`；第 2 轮 fake 仍会流式返回内容 | fake `InvokeStream` 调用次数 **== 1**；`Messages` 中该工具结果存在（内容带引号）；`StopReason == "stop_after_tool_call"` |
| **N3** | `RunStream` | 首轮即返回最终文本（无 tool call） | `Done.StopReason == "no_tool_calls"` |
| **N4（对照）** | `Run` + `RunStream` | 同一场景：3 个 tool call + limit 2 | 两条路径产出的 `RoleTool` 消息序列（按 `ToolCallID`+`Content` 排序后）**完全相等**；终止原因相等 |
| **R1** | 同上 | 沿用既有测试 | 既有断言**零改动** |

## 7. 完成判据

| # | 判据 | 机器可判定 |
|---|---|---|
| C1 | N1/N1b/N2/N3/N4 行为断言全部存在且通过 | 零退出 receipt |
| C2 | 既有 43 agent + 11 runner + 9 contract 全绿，**断言零改动** | 零退出 receipt + `git diff` 无既有测试行修改 |
| C3 | `stream_aggregator.go` **零改动** | `git diff --stat -- pkg/hno/agent/stream_aggregator.go` 为空 |
| C4 | `pkg/hno/runner/runner.go` 仅允许**加法式**新增（新增纯函数），状态机与既有断言零改动 | 逐行审计 + `runner_test.go` 全绿 |
| C5 | `streamMockModel.Invoke` 的 `"unexpected sync invoke"` 守卫未被放宽 | `grep` 确认该字符串仍在 |
| C6 | GREEN 不止于编译通过：行为断言实际执行并 PASS | 聚焦测试输出含 PASS 记录 |
| C7 | `-race` 无告警 | 零退出 receipt |
| C8 | `RunStreamResult` **未**新增 `StopReason` 字段（该值在流开始时未知） | `grep` 确认 |

## 8. 最小纵向切片

**切片：N1（流式 `ToolCallLimit` 截断）**。

**为何足以代表本周期目标**：

1. 它横跨接线面最宽：`Config.ToolCallLimit` → 共享策略函数 → 流式循环分支 → skipped 消息回注 → `Done.StopReason`。N2、N3 共享同一条接线。
2. 期望值完全确定，无「断言写成当前输出」的空间。
3. 它最易**静默失效**：若策略函数没接上流式循环，代码照常编译、照常跑，只是 limit 不生效 —— 只有 N1 能抓住。
4. N4 对照测试必须在 N1 之后做，否则没有第二条路径可比。

## 9. 禁止假通过条款

任一发生即判本周期失败：

1. 删除 / `t.Skip` / 注释任何既有断言（含 `streamMockModel` 的 sync 守卫）
2. 放宽既有期望以匹配新实现
3. 只让测试编译通过而不验证行为
4. 为通过测试而修改 `stream_aggregator` 的聚合或事件顺序
5. 用「结构上共用一个循环」替代 N4 的行为对照断言
6. 把 `StopReason` 加到 `RunStreamResult` 上（值在流开始时未知，会制造竞态）
7. 复活 `HITLBlocked`
8. 借本周期「顺便」改同步路径 `Agent.Run`

## 10. 预期 testability 结论

| 字段 | 值 |
|---|---|
| 类型 | **`behavior-delta`** |
| 理由 | N1/N2/N3 是新增用户可观察行为；当前 `RunStream` 经核实**不具备**（`grep` 确认 `stream.go:89` 的循环无任何截断/stop/stopReason 逻辑） |
| `redCandidate.publicEntry` | `(*agent.Agent).RunStream` |
| `redCandidate` 必备 | 公共入口、场景 setup、精确命令、预期、实际观察、失败原因、非零退出 `receiptRef` |
| 禁止 | 不得复用零退出基线充当 RED；不得以自然语言或未执行命令替代 `receiptRef` |

## 11. 聚焦测试命令与回执

```bash
# RED / GREEN 共用同一聚焦命令
go test ./pkg/hno/agent/... -run 'P0S' -count=1

# 邻近回归
go test ./pkg/hno/agent/... ./pkg/hno/runner/... -count=1
go test ./internal/session/contract/... -count=1
go test -race ./pkg/hno/agent/... -count=1
```

```bash
rex-harness receipt --root /Users/rex/codes/agno-go -- go test ./pkg/hno/agent/... -run P0S -count=1
```
