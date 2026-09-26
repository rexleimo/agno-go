# 最小构造评估 — `work-p0b-runner-capability-adoption`

- **work item**: `agno-go-v3-p0b-runner-capabilities`
- **Command**: `rex-minimal-construction` / `minimize`
- **前置范围变更**: `artifact:testability-decision:p0-loop-consolidation-blocked`

## 结论摘要

**「实现三项能力」是错误的问题陈述。** 三项中两项在 `pkg/hno/runner` 已完整实现，Agent 侧只需适配；第三项（HITLBlocked）与 `work-p5` 的持久化 HITL 重复，属应删除的偶然复杂度。

最小构造 = **一个 Agent↔runner 适配器 + 一个 bool 透传 + 一个字段透传 + 一个枚举字段**，而非三套新功能。

## 复用阶梯逐级评估

### 第 1 级：需求中的偶然复杂度可否删除？

| 候选 | 判断 | 依据 |
|---|---|---|
| `ToolCallLimit` | **保留** | 真实成本防护。当前 Agent 只能靠 `MaxLoops` 间接约束，工具调用次数无独立上限 |
| `StopLoop` | **保留** | 与既有终止路径语义不同：`MaxLoops` 是**框架**截断，`StopLoop` 是**工具/agent 主动**声明「不必再探索」。二者不可互相替代 |
| `HITLBlocked` | **删除** | ⚠️ **偶然复杂度**。P0b 的审批阻断是**瞬时停止、不可恢复**；`work-p5` 的 HITL 是**持久化中断 + schema 校验 + 幂等 resume**。先建瞬时版再在 P5 整体替换，属典型返工 |

**HITLBlocked 删除理由（可核验）**：
```
$ grep -n "HITLBlocked" docs/design/v3-delivery-ticket.json
14:  ... runner 的 ToolCallLimit / StopLoop / HITLBlocked 增量能力在本次收口中【不启用】...
41:  "工具可声明 HITLBlocked，触发后以可观察状态退出而非静默完成",
153: work-p5 outcome: "...HITL 支持 schema 校验、幂等恢复与重入/交接两种恢复模式..."
```
同一能力在 P0b（瞬时）与 P5（持久）被登记两次。**保留 P0b 版本会制造一次确定的废弃。**

### 第 2 级：仓库已有代码能否复用？——**能，且是主体**

`runner.Config` 需要的全部输入（`pkg/hno/runner/runner.go`）：
```
Model, Tools, MaxTurns, ToolCallLimit, MessageBuilder, ToolExecutor, OnStep, Logger
```

Agent 侧既有资产对照：

| runner 需求 | Agent 现状 | 差距 |
|---|---|---|
| `ToolExecutor` | `executeToolCalls`（`tool_executor.go:36`）、`executeOneTool`（:74）、`buildFunctionIndex`（:17） | 签名不符：现返回 `error` 并直写 `Memory`；runner 要求返回 `[]ToolCallOutcome` |
| `MessageBuilder` | `updateSystemMessage`（`message_builder.go:5`） | 签名不符：现为 `([]*Message, string) []*Message`；runner 要求 `Build(ctx, messages, tools) (*InvokeRequest, error)` |
| `ToolCallLimit` | 无 | **零新增逻辑**，纯透传 |
| 三项能力的循环逻辑 | `runner.go` 已实现（`StopLoop`@260、`HITLBlocked`@263、`StopReason`@286、limit 截断@240-256） | **零新增循环逻辑** |

**结论**：本工作项的主体是**写适配器**，不是写功能。

### 第 3 级：语言 / 标准库能否解决？

`context` + `errgroup` 可支撑并发工具执行，但 `Agent.executeToolCalls` 已用 `sync.WaitGroup` + 有序 `results` 切片实现等价语义且有测试（`TestExecuteToolCalls_Concurrent`）。**沿用现有实现**，不引入 `errgroup`。

### 第 4 级：已安装依赖能否解决且不扩大耦合？

`hashicorp/golang-lru`（缓存）已在用，但与本工作项无关。采用即扩大耦合。**不采用。**

### 第 5 级：局部表达式能否保持可读与可测？

- `StopLoop` 透传 = 在 `toolResult`（现为 `{callID, message}`）加一个 `stopLoop bool` 并在适配器里读出。**可，且是本工作项唯一的新增字段。**
- `StopReason` 暴露 = `RunOutput` 加一个 `StopReason string` 字段。**可。**
- `ToolCallLimit` 透传 = `Agent.Config` 加 `ToolCallLimit int`，构造时传入 `runner.Config`。**可。**

### 第 6 级：都不成立时的最小新构造

不适用（第 2 级已成立）。若强行不写适配器，唯一「更小」的做法是把 `runner` 内联进 `agent` —— **拒绝**，那等于把死代码换个位置，红线 2 依然违规。

## 被拒绝的选项

| 选项 | 拒绝理由 |
|---|---|
| 在 Agent 侧独立实现三项能力 | 制造第 5 份循环实现，直接违反红线 2 |
| 把 `runner` 内联进 `agent` 包 | 只是移动死代码，不解决唯一真相源问题 |
| 保留 `HITLBlocked` 于 P0b | 与 `work-p5` 重复，制造确定的废弃代码（见第 1 级） |
| 用 `errgroup` 重写并发工具执行 | 现有 `WaitGroup`+有序切片已满足且有测试，替换无收益且放大 diff |
| 在 P0b 中同时暴露 `StopReason` 的全部 6 个枚举值 | 一次只暴露当前可达的子集，避免提前冻结枚举面 |

## 最小实现清单

| # | 改动 | 性质 | 新增逻辑量 |
|---|---|---|---|
| 1 | `toolResult` 增加 `stopLoop bool` | 新增字段 | 极小 |
| 2 | Agent 侧实现 `runner.MessageBuilder` 适配器 | 适配 | 小（已有 `updateSystemMessage` 可内联） |
| 3 | Agent 侧实现 `runner.ToolExecutor` 适配器 | 适配 | 中（需把 `Memory` 直写改为返回 outcome） |
| 4 | `Agent.Config` 增加 `ToolCallLimit`，透传至 `runner.Config` | 透传 | 极小 |
| 5 | `RunOutput` 增加 `StopReason` 字段 | 透传 | 极小 |
| 6 | **仅** `Agent.Run` 内部改走 `runner.Run`；`Agent.RunStream` **不纳入本周期** | 接线（收窄） | — |
| 7 | **不实现 `HITLBlocked`** | 删除 | 0 |
| 8 | **不实现 `runner.Config.OnSkippedToolCalls`** → 实现为 `pkg/hno/runner` 的**导出构造函数** | 增补（见下） | 极小 |

**对照原范围**：原 P0b 承诺「三项能力 + StopReason 暴露」；最小构造为「两项能力 + StopReason 暴露」。**范围缩减 1/3，且缩减的是确定会废弃的部分。**

### 第 6 项收窄裁定（2026-09-26，`review` 阶段 SPEC-1）

原第 6 项写作「`Agent.Run` / `RunStream` 内部改走 `runner.Run`」。实现未覆盖 `RunStream`：

```
$ grep -n "runner\." pkg/hno/agent/stream.go
（零命中）
$ sed -n '89p' pkg/hno/agent/stream.go
		for loopCount < a.MaxLoops {
```

**裁定：修订规格，不扩大实现。** 理由：

1. `RunStream` 涉及增量事件聚合（`stream_aggregator.go`）与分块聚合语义，接入 runner 需另行设计映射，风险显著高于收益。
2. `work-p0`（循环收口）本就要重评「`stream.go` 的循环是否一并收口」，在那里一次性处理比现在做更合适。
3. 本周期主体是「让 `runner` 从死代码变成实际内核」，该目标已达成。

**由此产生的能力缺口（必须随范围一并记录，不得静默丢失）**：

> 用户若通过 `RunStream` 调用 Agent，则 `ToolCallLimit`、`StopLoop`、`StopReason` **三项能力均不生效** —— 流式路径仍走 `stream.go:89` 的自有循环。
> 该缺口已登记为 `work-p0-stream-loop-consolidation`（见 `v3-delivery-ticket.json`）。

### 第 8 项增补裁定（2026-09-26，`review` 阶段 SPEC-3）

GREEN 阶段为消除「同一批 tool 消息两个写入者」的缺陷，引入了 `runner.Config.OnSkippedToolCalls`。该字段**不在**原清单 7 项内，属实现超出规格。

**裁定：保留该能力，但把公共面从「回调」改为「共享构造函数」**，理由是回调把 runner 的内部决策外泄给宿主，而规格要求的只是「跳过消息的构造语义唯一」。见 `SPEC-2`。

## 已知沉淀约束

- `pkg/hno/runner` 现有测试（`runner_test.go`）覆盖三项能力的循环逻辑，**不得删改**；适配器接线后这些测试须仍全绿。
- `pkg/hno/agent` 既有 44 个测试**断言不得改写**（见 `artifact:docs/design/v3-test-scope-p0.md` §10）。
- 契约层 `internal/session/contract` 9 个测试**不得变红**。
- `work-p5` 落地时若需要审批阻断的**瞬时**语义，须由 P5 自行决定，不得反向要求 P0b 复活。

## Evidence refs

- `artifact:docs/design/v3-delivery-ticket.json`（P0b 与 P5 范围对照）
- `artifact:docs/design/v3-test-scope-p0.md`（B1–B11 等价行为，不得破坏）
- `artifact:docs/design/v3-testability-decision-p0-blocked.json`（范围变更来源）
- `artifact:docs/design/v3-baseline-evidence.md#2`（runner 为死代码的证据）
