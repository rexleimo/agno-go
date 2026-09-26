# 审查发现处置记录 — P0b blocking 项

- **来源**: `artifact:docs/design/v3-p0b-review-verdict.json`（verdict: `changes-requested`）
- **处置范围**: 三项 blocking（SPEC-1 / CONFLICT-1 / STD-1）
- **裁定人**: 用户选择路线 **B**（修订规格，不扩大实现）

## 处置结果

| ID | 严重度 | 处置 | 证据 |
|---|---|---|---|
| **SPEC-1** `RunStream` 未接入 runner | hard | **修订规格**（路线 B）+ 缺口已登记为独立工作项 | `v3-p0b-minimal-construction.md` 第 6 项收窄裁定；`v3-delivery-ticket.json` 新增 `work-p0-stream-loop-consolidation` |
| **CONFLICT-1** 交付票要求 `HITLBlocked` | hard | **更新交付票**，移除该条 | `v3-delivery-ticket.json` P0b `completionCriteria`（6 条，HITLBlocked 已删） |
| **STD-1** 复述性注释 | hard | **已修复** | `run.go` 两条注释删除；全量测试绿、`-race` 干净、`gofmt` 干净 |
| STD-2 请求构造三处重复 | judgement | 未修（记入待办） | 见下 |
| STD-3 `Run` 210 行 Divergent Change | judgement | 未修（记入待办） | 见下 |
| STD-4 `StopReason` 用 `string` | judgement | 不修，理由见下 | 见下 |
| SPEC-2 复制 runner 跳过消息字面量 | judgement | **已并入新工作项完成判据** | `work-p0-stream-loop-consolidation` 要求 sync/stream 消息序列一致 |
| SPEC-3 `OnSkippedToolCalls` 属规格外新增 | judgement | **补记进规格** | `v3-p0b-minimal-construction.md` 第 8 项增补裁定 |

## 关键决策：SPEC-1 为什么修订规格而不是补实现

`RunStream` 未接入 runner 属实。裁定**不扩大本周期实现**，理由：

1. `RunStream` 涉及增量事件聚合（`stream_aggregator.go`）与分块聚合语义，接入 runner 需另行设计映射，风险显著高于收益。
2. `work-p0` 本就要重评「`stream.go` 的循环是否一并收口」，在那里一次性处理比现在做更合适。
3. 本周期主体「让 runner 从死代码变成实际内核」已达成。

**该裁定产生的能力缺口，已显式登记而非静默丢弃：**

> 用户若通过 `RunStream` 调用 Agent，`ToolCallLimit`、`StopLoop`、`StopReason` **三项能力均不生效**。
> 缺口由 `work-p0-stream-loop-consolidation` 承接，并已加进 `work-p4-streammode` 与 `work-p9-release` 的依赖链。

## 交付票变更明细

- P0b `title`：`ToolCallLimit / StopLoop / HITLBlocked` → `ToolCallLimit / StopLoop / StopReason`
- P0b `outcome`：三项能力 → 两项，并写明 HITLBlocked 的删除理由
- P0b `completionCriteria`：删除 HITLBlocked 条目，新增「范围限于 `Agent.Run`」条目
- **新增** `work-p0-stream-loop-consolidation`（dependsOn: P0b；被 P4、P9 依赖）
- `frontier.blocked` 新增该工作项的阻塞理由
- `parallelGroups` 新增独立分组（不与 P0b 并发，避免同时改适配器语义）

校验：`node scripts/validate-delivery-ticket.mjs` → `VALID`，12 work items，无环，frontier 一致。
回执 `receipt:19a41976-7748-4b1c-9b62-647239288d49`（exit 0）。

## 未修项的处置

**STD-2（请求构造三处重复）** 与 **STD-3（`Run` 210 行）** 是判断性发现，不属 blocking。二者共同指向同一根因：`run.go` 承担了过多职责。已并入 `work-p0` 收口项的前置考虑 —— 该项本来就要重评循环收口，届时一并处理。

**STD-4（`StopReason` 用 `string` 而非类型）** 明确不修：`RunOutput` 是对外 JSON 序列化的 DTO，字段带 `json:"stop_reason,omitempty"` 标签，用 `string` 是正确选择。改用类型反而会让 DTO 依赖内部包。已在 verdict 中保留记录以备后续争议。

## 遗留事项

**已补齐（2026-09-26）**：`artifact:docs/design/v3-p0b-test-diff-review.md` 用变异测试实证的三处断言缺口已全部关闭，既有断言零改动（纯追加），且用同一批变异重跑确认已被抓住：

- A `stopLoop` 丢弃工具结果 → 被抓（`p0b_red_test.go:292`）
- B 3 条消息全变 skip 文案 → 被抓（SingleRound + MultiRound）
- C 交换 `call1`/`call2` 结果（条数对、身份错）→ 被抓

补写过程中发现一处**我自己的期望值错误**（非生产缺陷）：`toolkit.FormatResult` 对 handler 返回值做 `json.Marshal`，字符串带引号，故期望值应为 `"\"result1\""` 而非 `"result1"`。已在测试中加注说明。

**仍未覆盖**：`v3-p0b-test-diff-review.md` 缺口 4 ——

1. `StopLoop` 与 `ToolCallLimit` 同时触发时的优先级（`runner.go:294-301` 的 switch 顺序）
2. 多个工具中仅一个声明 `StopLoop` 时，其余工具是否仍执行
3. 工具未找到 / 参数非法 与 `ToolCallLimit` 的交互

这三项建议并入 `work-p0` 收口项。

**STD-2 / STD-3 仍未修**（判断性、非 blocking）：请求构造三处重复、`Run` 210 行职责过载，已并入 `work-p0` 前置考虑。
