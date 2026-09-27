# REFACTOR / 测试差异审查 — `work-p0-stream-loop-consolidation-v2`

- **Command**: `rex-tdd` / `stageId: refactor` / `activationId: 14a9faae-5113-4064-9047-3be51eb1b6f1`
- **GREEN 回执**: `receipt:74e6a186-868d-472a-a6ce-c43e51ea78c6`（exit 0）
- **REFACTOR 回执**: `receipt:2ddfc18c-f0ca-4a86-a652-ba7042c008df`（exit 0，场景命令逐字一致）
- **变异回执**: `receipt:269e5960-f9ba-4286-b7b9-1e44a211d455` / `receipt:9444a9af-06f3-4bf0-8d9d-2f001db3c57c` / `receipt:f11cdb63-356a-404f-b830-d7c431507f80`（均 exit 1）

## 1. refactor 阶段改动

| 改动 | 理由 |
|---|---|
| 删除 `stream.go` 的 `singleDoneChannel` | 无任何调用方（`grep` 仅剩自身与 `agent.go` 里一条指向它的悬空注释），循环收口后彻底成为死代码 |
| 删除 `agent.go:115` 的悬空注释 | 同上，注释描述的对象已不存在 |
| 回退 v1 在 `tool_executor.go` 引入的 `executeToolCallsWithStopLoop` 拆分 | 流式改走内核后不再需要；`tool_executor.go` 现与 HEAD 逐字一致（不在 `git diff --stat` 中） |
| 保留 `maxLoopsExceeded` 与 `assistantMessage` 两个小函数 | 它们是同步与流式当前唯一的共用判定点；内联回去即重新制造两处可漂移的副本 |

refactor 前后 P0SV2 场景命令均为 exit 0；`-race` 下 agent/runner/contract 三包全绿。

## 2. 测试差异审查：新断言是否仍约束用户行为

`pkg/hno/agent/p0stream_v2_red_test.go` 的 5 个用例全部只经公共入口观察：`Agent.Run`、`Agent.RunStream`、`RunStreamResult.StopReason()`、`RunStreamDone.{Output,Err,StopReason}`、fake model 的 `InvokeStream` 次数。无一处引用 `runner` 私有字段或状态机内部变量；没有删除、跳过或放宽任何既有断言（`git diff --name-only | grep _test.go | grep -v p0stream_v2` 为空）。

**发现的一处强度不足（如实记录，不掩盖）**：`TestP0SV2_SyncAndStreamProduceSameMessageSequence` 断言的是「两条路径一致」。单独变异 `kernel.go` 的 `OnSkippedToolCalls`（不再向 Memory 回注 skipped 消息）时，P0SV2 场景命令 **exit 0**（`receipt:b78d1964-e2d1-487b-801d-f87701358253`），因为两条路径同时变错、序列仍然相等。该缺陷由 v1 的 `TestP0S_ToolCallLimitTruncatesInStream` 杀掉（`receipt:f11cdb63-356a-404f-b830-d7c431507f80` exit 1，输出 `p0stream_red_test.go:143 期望恰好 3 条 RoleTool 消息，实际为 2`）。

**结论**：parity 测试必须与绝对期望测试**同时存在**才有意义，两个文件都不得删。这一点应写进交付票，否则后续周期很容易只留 parity 而失去担保。

## 3. 变异验证结果

| # | 变异 | 期望 | 实际 | 回执 |
|---|---|---|---|---|
| M1 | 内核 `executed` 改回按历史计数（复现 v1 缺陷） | P0SV2 计数用例失败 | `TestP0SV2_ToolCallLimitCountsPerRunInStream` 与 `...InSync` 双双失败（c1 无结果消息、最终内容为空、回合数 1） | `receipt:269e5960-f9ba-4286-b7b9-1e44a211d455` exit 1 |
| M2 | 流式失败分支硬编码回报 `no_tool_calls`（复现 v1 缺陷） | B5 用例失败 | `TestP0SV2_ModelFailureStopReasonIsNotNoToolCalls` 失败，输出实际值 `"no_tool_calls"` | `receipt:9444a9af-06f3-4bf0-8d9d-2f001db3c57c` exit 1 |
| M3 | 移除 skipped 消息回注 | 组合用例失败 | P0SV2 子集存活（见 §2），组合 `-run 'P0SV2\|P0S'` 失败 | `receipt:f11cdb63-356a-404f-b830-d7c431507f80` exit 1 |

每次变异都先备份原文件到 `/tmp`，运行后 `cp` 还原并复跑场景命令确认 exit 0；还原后 `kernel.go`/`stream.go`/`runner.go` 与备份一致（`stream.go` 死代码删除发生在备份之前，还原结果即当前 refactor 后版本）。

## 4. 由审查提出的两项后续

1. `v3-test-scope-p0stream-v2.md` §7 C8 的判据文案与本契约 §1 自相矛盾（详见 GREEN 记录 §5），需在 review 阶段改写为「无导出的可写 `StopReason` 字段；竞态由 `-race` 判据守」。
2. 交付票 `work-p0-stream-loop-consolidation` 判据 1 的 `grep -c 'for {'` 形式判据需要收窄为「tool 循环归零」：`streamOnce` 保留的是**单回合内的分块泵**（`stream.go:220`），把它也搬进内核属 P4 StreamMode 的范围。
