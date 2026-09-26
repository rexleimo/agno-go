# GREEN 实现差异记录 — `work-p0b-runner-capability-adoption`

- **work item**: `agno-go-v3-p0b-runner-capabilities`
- **Command**: `rex-tdd` / `stageId: green`
- **RED 回执**: `receipt:bdc53fe3-5cb2-4630-ada8-992ac163eadb`（exit 1）
- **GREEN 回执**: `receipt:c2b796b1-6ce7-4ac3-9aa5-d1ccb8fc3542`（exit 0）

## 1. 变更清单

```
 pkg/hno/agent/agent.go           |  30 ++++---
 pkg/hno/agent/config.go          |  30 ++++---
 pkg/hno/agent/run.go             | 183 ++++++++++++++++++++++++--------
 pkg/hno/runner/runner.go         |  43 +++++----
 pkg/hno/tools/toolkit/toolkit.go |   1 +
 5 files changed, 171 insertions(+), 116 deletions(-)
新增：pkg/hno/agent/runner_adapter.go
新增：pkg/hno/agent/p0b_red_test.go（RED 测试，未在 GREEN 阶段改动）
```

### 三项能力的来源

| 能力 | 实现方式 | 新增逻辑量 |
|---|---|---|
| `ToolCallLimit` | `agent.Config` / `Agent` 加字段，透传至 `runner.Config`；截断逻辑复用 `runner.go:230-256` | 透传 |
| `StopLoop` | `toolkit.Function.StopLoop bool`；判定复用 `runner.go:295-301` | 1 字段 |
| `StopReason` | `RunOutput.StopReason` 取 `runner.StopReason` 字符串 | 1 字段 |
| 循环内核 | `Agent.Run` 改走 `runner.Runner`，新增 `runner_adapter.go` 提供 `MessageBuilder` / `ToolExecutor` 适配器 | 接线 |

**未实现 `HITLBlocked`** —— 与最小构造第 1 级判定一致（与 `work-p5` 重复）。

## 2. GREEN 期间解决的四个缺陷

GREEN 并非一次通过。四轮修复，每轮都由真实测试失败驱动：

### 缺陷 1：limit 截断时 tool_call 失去配对响应
`ToolCallLimit` 截断后，被跳过的调用没有对应 `tool_response`，`RunOutput.Messages` 缺一条，且下一轮发给 provider 会是**未配对的 tool_call**。

- 首版修复（事后回捞）**失败并被推翻**：`runnerMessages[initialCount:]` 把 adapter 已写入 `a.Memory` 的消息又加了一遍，导致 3→5、2→4。
- 根因是**同一批消息有两个写入者**。最终采用加法式 `runner.Config.OnSkippedToolCalls` 钩子，agent 侧让「已执行」与「被跳过」走**同一条** `a.Memory.Add` 路径，每条 tool 消息有且仅有一个写入者。

### 缺陷 2：assistant 消息从未写入 Memory
改接 runner 时丢失 `a.Memory.Add(assistantMsg, ...)`，导致 5 个既有测试失败 + panic（"expected 2 messages, got 1"）。经 `OnStep` 的 `StateAwaitModel` 事件恢复，时序与旧版一致（先写 assistant、再执行工具）。

### 缺陷 3：`MaxLoops` 耗尽契约断裂
旧版返回 error `"max tool calling loops reached"`；runner 返回 `StopLimitReached` + nil error。由于 `runner_test.go:231` **显式钉死**轮次耗尽时 `reason == StopLimitReached`，不可新增枚举区分，故在 agent 侧以 `stopReason == StopLimitReached && turn >= a.MaxLoops` 判定并还原旧错误。

不与 P0B 冲突：P0B 用默认 `MaxLoops=10`，在第 1～3 轮即撞 `ToolCallLimit`，`turn < 10`。

### 缺陷 4：`extractReasoning` / 缓存写入 / `Metadata["loops"]` 丢失
随接线条一并恢复。

## 3. 约束合规核验

| 判据 | 结果 | 核验方式 |
|---|---|---|
| C1 4 条 P0B 行为断言通过 | ✅ | `-run P0B -v` 逐条 PASS |
| C2 既有测试断言零改动 | ✅ | `git diff --name-only \| grep _test.go` 为空 |
| C3 契约层 9 测试全绿 | ✅ | `go test ./internal/session/contract/...` exit 0 |
| C4 非仅编译通过 | ✅ | 4 个用例实际执行并 PASS，非 build-only |
| C5 `-race` 无告警 | ✅ | `go test -race ./pkg/hno/agent/...` exit 0 |
| C6 未实现 `HITLBlocked` | ✅ | `grep -rn HITLBlocked pkg/hno/agent/` 为空 |

### `runner.go` 改动性质

`--numstat` 显示 `26 adds / 17 dels`。逐行审计确认 17 处删除**全部是 gofmt 对齐**：新增较长的 `onSkippedToolCalls` 字段后，`Runner` 结构体块与构造字面量块被 gofmt 重新对齐；另有一处等价重写 `skipped := response.ToolCalls[len(callsToRun):]` 后 `range skipped`（同一表达式、同一循环）。

**无字段删除、无逻辑删除、无消息内容变更**，`pkg/hno/runner` 行为契约与 `runner_test.go` 均未变。

## 4. GREEN 验收输出

```
gofmt -l pkg/hno/agent pkg/hno/runner pkg/hno/tools/toolkit   →  无输出
go build ./...                                                  →  exit 0
go test ./pkg/hno/agent/... -count=1                           →  ok   2.440s
go test ./pkg/hno/agent/... -run P0B -count=1 -v               →  4 PASS
go test ./pkg/hno/runner/... ./internal/session/contract/...    →  ok / ok
go test -race ./pkg/hno/agent/... -count=1                     →  ok   1.417s
```

## 5. 遗留与后续

- `pkg/hno/runner` 由死代码转为 `Agent` 的实际循环内核，红线 2「删除 `pkg/hno/runner`」**已不再适用**，应回写进 `v3-platform.md`。
- `StopLoop` 的优先级低于 `HITLBlocked`（`runner.go:294-301`）这一分支在 Agent 侧不可达，属 P5 范畴。
- `work-p0` 收口时，`run/loop.go` 与 `agent/stream.go` 的循环是否一并收口，需重新评估（取决于 P0b 已铺好的适配器能否复用）。
