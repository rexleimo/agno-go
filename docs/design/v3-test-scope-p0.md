# 测试范围契约 — 周期 1：`work-p0-loop-consolidation`

- **work item**: `agno-go-v3-platform`
- **Command**: `rex-test-design` / `design-tests`（activation `09b6726e-63ec-40cd-9825-21c3e5323c07`）
- **决策依据**: `artifact:docs/design/v3-decision-ticket.md`、`artifact:docs/design/v3-delivery-ticket.json`
- **基线证据**: `artifact:docs/design/v3-baseline-evidence.md`

> 本契约是本周期的**行为边界**。不得由实现、工期或当前测试结果反向改写。
> 目标或非目标需要变化时，向 harness 报告测试范围变更需求并停止。

---

## 0. 范围事实核查（推翻前一版草稿的关键发现）

草稿初版曾把 `ToolCallLimit` / `StopLoop` / `StopHITLBlocked` 列为「Agent 现有但无测试的行为」。
**经源码核实这是错的**，已修正：

```bash
$ grep -rn "ToolCallLimit\|StopLoop\|HITLBlocked" pkg/hno/ --include='*.go' | cut -d/ -f1-3 | sort -u
pkg/hno/runner/runner.go
pkg/hno/runner/stop.go
pkg/hno/runner/runner_test.go

$ grep -n "ToolCallLimit\|StopLoop\|HITLBlocked\|MaxLoops" pkg/hno/agent/*.go | grep -v _test
（仅 MaxLoops，无其余三项）
```

**事实**：当前在线的 `Agent` **只有 `MaxLoops` 一个循环控制概念**。
`ToolCallLimit` / `StopLoop` / `StopHITLBlocked` / `StopReason` **只存在于 `pkg/hno/runner`（死代码）及其测试中**。

**推论（本契约的形状由此决定）**：

- 收口到 runner **天然附带三项当前 Agent 不具备的能力**。
- 若在 P0 一并启用，P0 就不是等价重构，而是**新增用户可观察行为** → 需 RED 优先 TDD，与「P0 单独发 minor」（D4）冲突。
- 故本周期裁定：**P0 只做严格等价，不启用 runner 的增量能力**；其采纳拆为独立工作项 `work-p0b-runner-capability-adoption`，走 RED 优先 TDD。

## 1. 本周期性质

**行为保持型重构**（strict equivalence）。无新增用户可观察行为。
预期 `decide-testability` 结论为 **`behavior-preserving-hardening`**，前提是先取得零退出的真实基线回执。

## 2. 用户目标

> 让 `pkg/hno/runner` 从死代码变为 agent 唯一循环内核，循环实现从 4 份降为 2 份，且 `Agent.Run` / `Agent.RunStream` 的用户可观察行为逐项等价。

## 3. 范围内行为与已有覆盖

`RunOutput` 暴露面 = 断言目标。**注意：`runner.StopReason` 当前不在 `RunOutput` 中暴露，不得作为断言对象。**

| ID | 行为 | 观察面 | 已有测试 | 状态 |
|---|---|---|---|---|
| B1 | 无工具调用 → 单次模型调用完成 | `Status`/`Content` | `TestAgent_Run_SimpleResponse` | ✅ 已覆盖 |
| B2 | 有工具调用 → 执行后回到模型 | `Messages` 轮次结构 | `TestAgent_Run_WithToolCalls` | ✅ 已覆盖 |
| B3 | `MaxLoops` 截断 | `HnoError.Code==ErrCodeUnknown` + 消息 | `TestAgent_Run_MaxLoops` | ✅ 已覆盖 |
| B4 | 缓存命中 → 不调模型 | `Metadata["cache_hit"]==true` | `TestAgent_Run_UsesCache` | ✅ 已覆盖 |
| B5 | 上下文取消 | `Status==cancelled` + `ErrCodeCancelled` | `TestAgent_Run_ContextCancelled`、`TestRunStreamCancellation` | ✅ 已覆盖 |
| B6 | 运行事件产出 | `Events` | `TestAgent_Run_EmitsEvents` | ✅ 已覆盖 |
| B7 | 空输入拒绝 | `ErrCodeInvalidInput` | `TestAgent_Run_EmptyInput` | ✅ 已覆盖 |
| B8 | 流式执行工具循环 | `Events`/`Messages` | `TestRunStreamExecutesToolCalls` | ✅ 已覆盖 |
| B9 | 工具/历史消息存储开关 | `len(Messages)` | 6 个 `Test*Storage*` | ✅ 已覆盖 |
| B10 | 临时指令仅本次生效 | 二次 `Run` 的消息内容 | `TestAgent_Run_AutoClearsTempInstructions` | ✅ 已覆盖 |
| B11 | RunContext 贯穿 hooks 与 tools | 工具收到的 ctx | `TestRun_PropagatesRunContextToHooksAndTools` | ✅ 已覆盖 |
| B12 | 并发工具执行的有序回填 | `Messages` 中 tool 消息顺序 | `TestExecuteToolCalls_Concurrent` | ⚠️ 部分覆盖，未断言**跨轮次的整体顺序** |
| B13 | 工具执行错误 → `Agent.Run` 返回 `ErrCodeToolExecution` | 返回 error 的 `Code` | — | ❌ **零覆盖** |
| B14 | PreHooks 失败 → `ErrCodeInputCheck` | 返回 error 的 `Code` | — | ❌ **零覆盖** |
| B15 | PostHooks 失败 → `ErrCodeOutputCheck` | 返回 error 的 `Code` | — | ❌ **零覆盖** |

**缺口定位**：B13/B14/B15 是当前 `Agent` **真实存在但零覆盖**的错误分类路径。
它们位于 `run.go` 的错误分支上，收口时必然被触碰，且错误码是**用户可观察**的（`types.HnoError.Code`）。

**已排除的伪缺口**：`ToolCallLimit` / `StopLoop` / `HITLBlocked` **不属于本周期**（当前 Agent 无此能力），已移至 `work-p0b-runner-capability-adoption`。

## 4. 明确非目标

| # | 非目标 | 理由 |
|---|---|---|
| N1 | 不改对外签名 `Agent.Run`/`RunStream`/`RunStreamResult` | D4；破坏性变更留 v3.0 主体 |
| N2 | 不启用 runner 的 ToolCallLimit / StopLoop / HITLBlocked | 见 §0；拆至 P0b |
| N3 | 不暴露 `StopReason` 到 `RunOutput` | 同上 |
| N4 | 不改语义（循环次数、缓存、取消、错误码） | 等价重构 |
| N5 | 不引入图引擎 / 事件化 / StreamMode | 分属 `work-p1`/`p4`/`p5` |
| N6 | 不动 hooks 与 guardrails 触发时机 | 前置/后置语义不变 |
| N7 | 不动 workflow / team 行为 | 分属 `work-p6` |

## 5. 范围外行为

- `work-p0b` 的三项增量能力（tool call limit / 工具请求停止 / 审批阻断）
- `work-p5` 的**可恢复** HITL interrupt/resume（本周期仅要求 B5 取消路径，不涉及审批恢复）
- workflow / team 编排、图引擎、StreamMode、Store

## 6. 允许修改的测试缝

| 允许 | 禁止 |
|---|---|
| `pkg/hno/agent/*_test.go` 新增特征化测试（B13/B14/B15 为主） | 修改任何**非测试**生产代码来让测试通过 |
| `pkg/hno/runner/*_test.go` | 改写既有断言或放宽期望值 |
| 注入 `models.Model` fake（复用既有 `mockModel` 模式） | `t.Skip` / 删除用例 / 注释断言 |
| 构造 `toolkit.Toolkit` fake 工具与 hook | 只断言 mock 被调用而不验证 `RunOutput` 或返回 error |
| `httptest` 真实地址 | 以内部调用次数替代行为断言 |

**公共入口**：`(*agent.Agent).Run` 与 `(*agent.Agent).RunStream`。
`runner` 的内部状态机（`StateAwaitModel`/`StateToolCallsPending`）**不是**可观察面。

## 7. 验收行为 → 公共入口 → 可观察断言 映射

| 行为 | 公共入口 | 场景 setup | 可观察断言 |
|---|---|---|---|
| **B13** 工具错误 | `Agent.Run` | fake 工具返回错误 | 返回 `*types.HnoError`；`errors.As` 后 `Code == types.ErrCodeToolExecution`；`RunOutput == nil` |
| **B14** PreHook 失败 | `Agent.Run` | `PreHooks` 含返回 error 的 `HookFunc` | `Code == types.ErrCodeInputCheck`；模型 fake 的 `Invoke` **调用次数为 0**（模型未被调用是用户可观察结果，非内部探针） |
| **B15** PostHook 失败 | `Agent.Run` | `PostHooks` 含返回 error 的 `HookFunc` | `Code == types.ErrCodeOutputCheck`；模型 fake 调用次数为 1；工具 fake 调用次数为 0 |
| **B12** 跨轮次顺序 | `Agent.Run` | 3 个工具各返回可区分标记，跨 2 轮 | `Messages` 中 tool 角色消息的出现顺序等于输入 `ToolCallID` 顺序，且轮次间不交错 |
| B1–B11 | 同上 | 沿用既有测试 | 既有断言**零改动**，仅在切换后复跑 |

## 8. 最小纵向切片

**切片**：**B13（工具执行错误 → `ErrCodeToolExecution`）**。

**为什么它足以代表本周期目标**：

1. 它是当前 `Agent` 真实存在、**零覆盖**、且**用户可观察**（`HnoError.Code`）的行为。
2. 它横跨收口必然触碰的路径：`run.go` 的工具执行错误分支 ↔ `runner` 的 `ToolExecutor.Execute` 错误返回。收口时该分支若丢失，**唯一症状就是错误码从 `TOOL_ERROR` 变成别的**——外部签名不变，所以只有这个断言能抓住。
3. 它不依赖任何内部状态机探针，断言全部落在返回的 error 上。
4. B14/B15 共享同一「错误分类」形态，可在同一纵向切片内低成本扩展。

## 9. 完成判据

| # | 判据 | 机器可判定 |
|---|---|---|
| C1 | B13/B14/B15 特征化测试在**切换前**存在并通过 | 零退出 receipt |
| C2 | 同一批测试在**切换后**仍全绿，且测试 diff 仅新增 | 零退出 receipt + `git diff` 仅新增 |
| C3 | 对外签名零变更 | `git diff` 对签名行无改动 |
| C4 | 既有 44 个 agent 测试全绿 | `go test ./pkg/hno/agent/... -count=1` 零退出 |
| C5 | `-race` 无告警 | `go test -race ./pkg/hno/agent/... -count=1` 零退出 |
| C6 | `runner` 不再是死代码 | `grep -rn 'hno/runner' --include='*.go' . \| grep -v '^./pkg/hno/runner/' \| wc -l` > 0 |
| C7 | 循环实现从 4 份降为 2 份 | `grep -c 'for {' pkg/hno/agent/run.go pkg/hno/agent/stream.go` 均为 0 |

## 10. 禁止假通过条款（硬约束）

任一发生即判本周期失败：

1. 删除或 `t.Skip` 任何既有断言
2. 放宽既有期望值以匹配新实现输出
3. 只断言 fake 被调用，不验证 `RunOutput` 或返回 error
4. 为通过测试而修改非测试生产代码
5. 特征化测试在切换前后内容不一致（只能新增）
6. 以「其他测试也绿」替代 §9 逐条判据
7. 把 `work-p0b` 的新增能力混入本周期以「顺便测一下」

## 11. 预期 testability 结论

| 字段 | 值 |
|---|---|
| 类型 | `behavior-preserving-hardening` |
| 理由 | P0 为严格等价重构，无新增用户可观察行为 |
| 前置 | 必须在真实场景取得**零退出**基线回执（B13/B14/B15 先落在现有实现上） |
| 禁止 | 不得伪造 RED；不得以自然语言、旧日志或未执行命令充当 `receiptRef` |

## 12. 聚焦测试命令与回执

```bash
# 基线（切换前 / 切换后，均须零退出）
go test ./pkg/hno/agent/... ./pkg/hno/runner/... -count=1

# 竞态
go test -race ./pkg/hno/agent/... -count=1

# 死代码与循环份数判据
grep -rn 'hno/runner' --include='*.go' . | grep -v '^./pkg/hno/runner/' | wc -l
grep -c 'for {' pkg/hno/agent/run.go pkg/hno/agent/stream.go
```

回执命令：

```bash
rex-harness receipt --root /Users/rex/codes/agno-go -- go test ./pkg/hno/agent/... ./pkg/hno/runner/... -count=1
```
