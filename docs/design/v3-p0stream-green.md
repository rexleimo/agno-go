# GREEN 实现差异记录 — `work-p0-stream-loop-consolidation`

- **workflowActivationId**: `f166216c-061d-47ec-b149-b78c3e02d507`
- **Command**: `rex-tdd` / `stageId: green`
- **RED 回执**: `receipt:090fdbf3-650d-49cf-875f-2c56ce101f00`（exit 1，行为级断言失败）
- **GREEN 回执**: `receipt:d701b62b-eaec-42cb-8fab-53a26169b437`（exit 0）

## 1. 变更清单

```
新增 pkg/hno/runner/tool_batch.go        共享纯函数（DecideToolBatch + NewToolCallLimitMessage + 共享文案常量）
改动 pkg/hno/agent/stream.go             循环接策略 + 跳过消息 + StopLoop + 每处 RunStreamDone 填 StopReason
改动 pkg/hno/agent/agent.go              RunStreamDone 加 StopReason（RunStreamResult 不变）
改动 pkg/hno/agent/tool_executor.go      内部执行方法可返回 stopLoop
改动 pkg/hno/runner/runner.go            截断分支改调用共享函数（见 §3 审计）
追加 pkg/hno/agent/p0stream_red_test.go  末尾加 TestP0S_StopReasonObservable
```

三项能力的来源：

| 能力 | 实现 | 新增逻辑量 |
|---|---|---|
| `ToolCallLimit`（流式） | stream 循环按 Memory 累计 tool 消息数算 `executed`，调 `runner.DecideToolBatch`，只执行 `toRun`，`skipped` 经 `runner.NewToolCallLimitMessage` 写入 Memory | 接线 |
| `StopLoop`（流式） | `tool_executor` 内部方法回传 `stopLoop`；该轮结果全写完后终止，不进下一轮 | 接线 |
| `StopReason` | `RunStreamDone.StopReason`，stream.go 全部 doneCh 发送点填充 | 字段 |

## 2. GREEN 验收输出

```
gofmt -l pkg/hno/agent pkg/hno/runner                      →  无输出
5 个 P0S 用例（-v）                                          →  全 PASS
go test ./pkg/hno/agent/... -count=1                        →  ok
go test ./pkg/hno/runner/... -count=1                       →  ok
go test ./internal/session/contract/... -count=1             →  ok  （9 个契约测试）
go test -race ./pkg/hno/agent/... -count=1                  →  ok，无竞态
```

5 个用例：`ToolCallLimitTruncatesInStream` / `ToolCallLimitExhaustedAcrossRounds` / `StopLoopHaltsStream` / `SyncAndStreamProduceSameToolMessages` / `StopReasonObservable`。

## 3. 约束合规核验

| 判据 | 结果 | 核验 |
|---|---|---|
| C1 五项行为断言通过 | ✅ | `-run P0S -v` 全 PASS |
| C2 既有测试断言零改动 | ✅ | `git diff --name-only \| grep _test.go \| grep -v p0stream` 为空 |
| C3 `stream_aggregator.go` 零改动 | ✅ | `git diff --stat` 该文件为空 |
| C4 `runner` 仅加法式 | ⚠️ 见 §4 审计 | `--numstat` 显示 runner.go 6 adds / 19 dels |
| C5 `streamMockModel` sync 守卫未被放宽 | ✅ | 既有测试文件零改动 |
| C6 非仅编译通过 | ✅ | 5 个用例实际执行 PASS |
| C7 `-race` 无告警 | ✅ | exit 0 |
| C8 `RunStreamResult` 未加 `StopReason` | ✅ | 该 struct 仍只有 `Events`/`Done` |
| N7 `run.go`（同步路径）零改动 | ✅ | `git diff --stat` 为空 |
| N4 无 `HITLBlocked` | ✅ | `grep -rn HITLBlocked pkg/hno/agent/` 为空 |

## 4. `runner.go` 19 处删除的逐行审计

**结论：不是字面意义的「纯加法」，但是最小构造第 2 项明确要求的抽取，语义逐字等价。**

19 处删除分两类：

1. **内联截断计算**（`callsToRun := response.ToolCalls` / `limitHit := false` / `if r.toolCallLimit > 0 {...}` 共 8 行）→ 被一行 `callsToRun, skipped, limitHit := DecideToolBatch(executed, r.toolCallLimit, response.ToolCalls)` 取代。
2. **内联跳过消息构造**（5 行字面量拼装）→ 被 `NewToolCallLimitMessage(c)` 取代。

**等价性推理**（最敏感的 exhausted 分支）：

- 原版：`remaining <= 0` 时在 `Execute` 之前直接 `break`，不追加任何消息。
- 新版：`DecideToolBatch` 返回 `(nil, nil, true)`，guard `limitHit && len(callsToRun)==0 && len(skipped)==0` 为真，同样在 `Execute` 之前 `break`，不追加消息。

且 `DecideToolBatch` 仅在 `remaining<=0` 时返回 `(nil,nil,true)`（其余情况 `toRun` 或 `skipped` 至少一者非空），故 guard 不会误触发。截断分支的 `onSkippedToolCalls` 调用与 `allMessages` 追加位置、顺序均未变。

**既有 11 个 runner 测试全绿**，为该等价性提供了机器判据。

**净收益**：跳过消息文案 `tool call limit reached; call not executed` 现在只存在于 `tool_batch.go` 的一个常量中，runner 与 stream 共用构造器。这正是 P0b 审查记录为 **SPEC-2（judgement）** 的漂移风险（当时 agent 侧复制了 runner 的字面量），本周期顺带消除了它。

## 5. 行为与测试的对应

`SyncAndStreamProduceSameToolMessages` 是本周期最关键的一条：它**不靠「结构上共用一个循环」来保证一致**，而是用同一场景分别跑 `Run` 与 `RunStream`、比对 `RoleTool` 消息的 `ToolCallID`+`Content` 排序后序列。这是判据 5 的行为级验证。

## 6. 遗留

- 交付票 `work-p0-stream-loop-consolidation` 的第 1、3 条判据文案需按最小构造 §4 修正（依赖图不变）。
- `v3-platform.md` 的红线 2「删除 `pkg/hno/runner`」已因 P0b/P0stream 失效，待回写。
- 与 P0b 相同的自检义务移交 `refactor` / `review` 阶段：变异测试确认新断言真能挡住缺陷。
