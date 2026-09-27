# 测试范围契约 — 周期 2：`work-p0b-runner-capability-adoption`

- **work item**: `agno-go-v3-p0b-runner-capabilities`
- **Command**: `rex-test-design` / `design-tests`
- **范围来源**: `artifact:docs/design/v3-p0b-minimal-construction.md`（最小构造缩减后的范围）
- **前置**: `artifact:testability-decision:p0-loop-consolidation-blocked`（原 P0 无诚实 RED，已 replan）

> 本契约是本周期的行为边界。不得由实现、工期或当前测试结果反向改写。

---

## 0. RED 性质声明（先说清，避免误判）

本周期**存在诚实 RED**。与周期 1 不同：周期 1 是等价重构（无新行为），本周期是**新增用户可观察行为**。

**RED 的表现形式是编译期失败**。原因：三项能力均需新增公共 API（`Agent.Config.ToolCallLimit`、`RunOutput.StopReason`、工具声明 `stopLoop` 的字段），在 Go 中测试先行必然先编译失败。

**这构成合法 RED 的判据**（对照 `rex-tdd` §RED.3「因语法、环境、错误夹具或无关依赖失败不得记为合法 RED」）：

| 判据 | 状态 |
|---|---|
| 测试文件本身是**语法正确、可格式化**的 Go | 必须（否则不合法） |
| 失败原因是**目标能力缺失**（字段/方法未定义） | 必须 |
| 失败原因是拼写错误、错误 import、错误夹具 | 不合法 |
| 与本次范围**无关的依赖**导致失败 | 不合法 |

**GREEN 后必须存在行为级绿色证据**：仅「编译通过」不构成 GREEN。§7 完成判据 C4 强制要求行为断言全部通过。

## 1. 用户目标

> 通过公共入口 `(*agent.Agent).Run` 暴露两项新能力（`ToolCallLimit` 截断、工具 `StopLoop` 主动停止）并使 `StopReason` 可观察，且不破坏周期 1 已钉死的等价行为 B1–B11。

## 2. 明确非目标

| # | 非目标 | 理由 |
|---|---|---|
| N1 | **不实现 `HITLBlocked`** | 最小构造第 1 级判定为偶然复杂度，与 `work-p5` 持久化 HITL 重复 |
| N2 | 不实现 HITL 的任何瞬时形态（含"阻塞后不可恢复"） | 同 N1；P5 落地时自行决定 |
| N3 | 不改 `MaxLoops` 语义 | 现有行为，错误码/消息不变 |
| N4 | 不引入图引擎 / 事件化 / StreamMode | 分属 `work-p1`/`p4`/`p5` |
| N5 | 不改 `models.Model` 接口 | 分属 `work-p7` |
| N6 | 不做 `StopReason` 全部 6 个枚举值的冻结 | 本期只暴露实际可达子集，避免提前冻结枚举面 |
| N7 | 不改 `workflow` / `team` | 分属 `work-p6` |

## 3. 范围内行为

公共入口 `(*agent.Agent).Run(ctx, string) (*agent.RunOutput, error)`。
`RunOutput` 当前暴露面（`agent.go:60-72`）：`Status` / `Content` / `Messages` / `Metadata` / `Events`。
**`StopReason` 字段当前不存在** —— 这正是 RED 的落点。

| ID | 行为 | 观察面 | 当前 |
|---|---|---|---|
| **N1** | `ToolCallLimit` 截断：达到上限时未执行的调用回注 tool 消息 | `Messages` 中 `RoleTool` 的 `ToolCallID` 与 `Content` | ❌ 能力缺失 |
| **N2** | `StopLoop`：工具声明后循环在该次工具调用后终止，不再回调模型 | 模型 fake 的调用次数 + `RunOutput` | ❌ 能力缺失 |
| **N3** | `StopReason` 可观察 | `RunOutput.StopReason` | ❌ 字段不存在 |
| R1 | 周期 1 的等价行为 B1–B11 全部不回归 | 既有 44 个测试 | ✅ 基线 `receipt:a7240aa8` |

### N1 精确期望值（依据 `runner.go:230-256` 实测，非推测）

- `executed := countToolMessages(allMessages)` —— **跨轮累计**，不是单轮计数
- `remaining <= 0` → 立即 `StopLimitReached` 终止，**且不再追加任何 tool 消息**
- `len(calls) > remaining` → 截断执行，`limitHit = true`
- 被跳过的每个调用追加 `{Role: RoleTool, ToolCallID: <原ID>, Content: "tool call limit reached; call not executed"}`
- 终止原因 `StopLimitReached`（值 `"limit_reached"`）

> **口径回写（2026-09-27，P0-stream 收口轮）**：上面两条期望值已被取代，实现里已不存在
> `countToolMessages`。当前口径见 `docs/design/v3-test-scope-p0stream-v2.md` §4：
> 计数改为**本次运行累计**（历史 tool 消息不占额度），且 `remaining <= 0` 时**不再静默丢弃**——
> 模型本轮请求的每个调用都按 `limitHit` 语义回注配对的 tool 消息，避免历史里留下无人应答的 tool call。
> 本节原文保留，作为该能力首次落地时的实测记录。

## 4. 范围外行为

- `work-p5` 的持久化 HITL interrupt/resume / schema 校验 / 幂等恢复
- `work-p1` 图引擎、`work-p4` StreamMode、`work-p6` workflow 迁移
- 周期 1 的 B12–B15 缺口（工具错误/PreHook/PostHook 错误码特征化）—— 那属 `work-p0`，本周期**不混入**

## 5. 允许修改的测试缝

| 允许 | 禁止 |
|---|---|
| `pkg/hno/agent/*_test.go` 新增测试 | 修改既有断言使其匹配新实现 |
| `pkg/hno/runner/*_test.go` 新增测试 | 删改 `runner` 既有测试（三项能力的循环逻辑已有覆盖） |
| fake `models.Model`（复用既有 `mockModel` 模式） | `t.Skip` / 注释断言 / 放宽期望值 |
| fake `toolkit.Toolkit` 工具 | 只断言 fake 被调用而不验证 `RunOutput` |
| `httptest` 真实地址 | 以内部状态机（`StateAwaitModel` 等）为断言目标 |
| 断言 `RunOutput` 与 fake 模型的**可观察调用次数** | 断言 `runner` 的私有字段或内部函数调用次数 |

**公共入口**：`(*agent.Agent).Run`。
「fake 模型被调用几次」是**用户可观察结果**（它决定了用户为本次运行付了几次模型费），不是内部探针 —— 这是允许的。

## 6. 验收行为 → 公共入口 → 可观察断言 映射

| 行为 | 公共入口 | 场景 setup | 可观察断言 |
|---|---|---|---|
| **N1** | `Agent.Run` | 模型一轮返回 3 个 tool call；`Agent.Config.ToolCallLimit = 2` | `Messages` 中 `RoleTool` 恰好 3 条：2 条内容为真实执行结果，第 3 条 `ToolCallID` 等于被跳过的原始 ID 且 `Content == "tool call limit reached; call not executed"`；`StopReason == "limit_reached"` |
| **N1b** | `Agent.Run` | 先用满 2 次上限，再进入第二轮仍有 tool call | 循环在 `remaining <= 0` 时终止，**第二轮不再追加任何 tool 消息**（断言 `RoleTool` 总数仍为 2） |
| **N2** | `Agent.Run` | 1 个工具声明 `stopLoop`；模型第二轮仍会返回 tool call | fake 模型**只被调用 1 次**；`Messages` 中有该工具的结果消息；`StopReason == "stop_after_tool_call"` |
| **N3** | `Agent.Run` | 上述 N1 / N2 场景 | `RunOutput.StopReason` 取值等于 `runner.StopReason` 字符串常量 |
| **R1** | 同上 | 沿用既有 44 个测试 | 既有断言**零改动** |

## 7. 完成判据

| # | 判据 | 机器可判定 |
|---|---|---|
| C1 | N1/N1b/N2/N3 四条行为断言全部存在且通过 | 零退出 receipt |
| C2 | 既有 44 个 agent 测试 + runner 包测试全绿，**断言零改动** | 零退出 receipt + `git diff` 无既有测试行的修改 |
| C3 | `internal/session/contract` 9 测试全绿 | 零退出 receipt |
| C4 | **GREEN 阶段不只是编译通过**：C1 的行为断言必须实际执行并通过 | 聚焦测试输出中含 PASS 记录，非 `[no test files]` / 非仅 build |
| C5 | `-race` 无告警 | 零退出 receipt |
| C6 | `HITLBlocked` **未被实现**（最小构造第 1 级的删除必须保持） | `grep -rn "HITLBlocked" pkg/hno/agent/` 为空 |

## 8. 最小纵向切片

**切片：N1（`ToolCallLimit` 截断）**。

**为什么它足以代表本周期目标**：

1. 它横跨接线面最宽的一段：`Agent.Config` 新字段 → 适配器 → `runner` 循环的 `limitHit` 分支 → 消息回注 → `RunReason`。一条测试即覆盖 P0b 主体。
2. 它的期望值**完全确定且可精确定义**（见 §3），不存在"断言写成当前输出"的空间。
3. 它是最容易在接线时**静默失效**的一项：若适配器漏传 `ToolCallLimit`，代码照常编译、照常跑，只是 limit 不生效 —— 只有这个断言能抓住。
4. N2 共享同一「循环提前终止」形态，N3 是一次字段透传，均可在同一纵向切片内低成本覆盖。

## 9. 禁止假通过条款

任一发生即判本周期失败：

1. 删除 / `t.Skip` / 注释任何既有断言
2. 放宽既有期望值以匹配新实现
3. 只让测试编译通过而不验证行为（见 C4）
4. 为通过测试而修改非测试生产代码中的**契约**（如改 `runner` 的 limit 语义）
5. 断言内部状态机或私有字段作为行为证据
6. 把 `work-p5` 的 HITL 形态偷偷带进来（违反 N1/N2）
7. 复活 `HITLBlocked`（违反 C6）
8. 把周期 1 的 B12–B15 缺口混入本周期「顺便测一下」

## 10. 预期 testability 结论

| 字段 | 值 |
|---|---|
| 类型 | **`behavior-delta`** |
| 理由 | N1/N2/N3 是**新增的用户可观察行为**，当前 `Agent` 经核实不具备（`grep ToolCallLimit\|StopLoop\|HITLBlocked pkg/hno/` 仅命中 `pkg/hno/runner` 死代码） |
| `redCandidate.publicEntry` | `(*agent.Agent).Run` |
| `redCandidate` 必须包含 | 公共入口、场景 setup、精确命令、预期、实际观察、失败原因、非零退出 `receiptRef` |
| 禁止 | 不得复用周期 1 的零退出基线充当 RED；不得以自然语言或未执行命令替代 `receiptRef` |

## 11. 聚焦测试命令与回执

```bash
# RED / GREEN 共用同一聚焦命令
go test ./pkg/hno/agent/... -run 'ToolCallLimit|StopLoop|StopReason' -count=1

# 邻近回归
go test ./pkg/hno/agent/... ./pkg/hno/runner/... -count=1
go test ./internal/session/contract/... -count=1
go test -race ./pkg/hno/agent/... -count=1
```

```bash
rex-harness receipt --root /Users/rex/codes/agno-go -- go test ./pkg/hno/agent/... -run 'ToolCallLimit|StopLoop|StopReason' -count=1
```
