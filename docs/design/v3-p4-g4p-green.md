# 切片 34 GREEN（P4 第 3 片，母约 G4：剩余五模式生产者接线，P4 全清）

工作项 `work-p4-g4-producers` / 契约 `docs/design/v3-test-scope-p4-g4-producers.json`（D1–D9）。
RED 观察见 `docs/design/v3-red-observation-p4g4p.md`。本片绑定场景命令逐字：

```bash
go test ./pkg/hno/agent -run TestP4G4P_ -count=1 -race -v
```

## 1. GREEN 判据与实测

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| D1–D8 全绿（8 个测试函数 `--- PASS`，契约绑定命令） | 同上 | 0 | `receipt:4082e7da-164d-4c67-b25a-99b510f4c327` |
| 基线（开工时 agent+run plain） | `go test ./pkg/hno/agent ./pkg/hno/run -count=1` | 0 | `receipt:8210683a-ba1e-4cc1-9319-f40be844d6ad` |
| 基线（开工时 agent+run race） | `go test -race ./pkg/hno/agent ./pkg/hno/run -count=1` | 0 | `receipt:1b7e30c7-103b-4b16-b5fe-e9e58a647ff6` |
| 门禁合跑：build + plain + race + vet + gofmt | `go build ./... && go test ./pkg/hno/agent ./pkg/hno/run -count=1 && go test -race … && go vet … && gofmt -l … \| wc -l`（gofmt 输出 0 行） | 0 | `receipt:f9d72be6-bd80-4442-b4bb-080168730fc9` |
| 邻近消费面回归（不跑 pkg/hno/graph 测试） | `go test ./pkg/hno/team/... ./pkg/hno/workflow/... ./internal/session/contract/... -count=1` | 0 | `receipt:03ae0b3a-99cc-4911-913b-bc605711ef2c` |
| 防碰巧绿：绑定 `-count=30 -race` | `go test ./pkg/hno/agent -run TestP4G4P_ -count=30 -race` | 0 | `receipt:029f1102-c46f-450b-8cda-5424a73ccdbb` |
| 防碰巧绿：agent+run `-count=5 -race` | `go test ./pkg/hno/agent ./pkg/hno/run -count=5 -race` | 0 | `receipt:c6326052-df6b-4eb9-bd56-a529800be23e` |

## 2. 实现清单（契约 sourceSeam 边界内）

| 文件 | 状态 | 内容 |
|---|---|---|
| `pkg/hno/agent/stream_producers.go` | 修改（146 → 289 行） | wiredStreamModes 扩到七；`enabled()` 增 Debug 并集分支（Checkpoints ∪ Tasks，不暗含 Messages）；`taskStarted`/`taskFinished` 族门内移；`tasksExecutor` → `streamExecutor`（Tasks∨Custom 双族选择）；新增 `turnCompleted`（Updates→Values→Checkpoints 固定次序）+ `snapshot`（新 map，Values 与 Checkpoint 不共享载荷）；新增 `turnObserver`（接内核 void 钩子，取消即丢事件不另设错误面）；新增 Custom 写透（`customWriterKey`/`customWriter`（Mutex 串行化整个 emit）/`withCustomWriter`/`custom`）；新增导出 `agent.WriteCustomEvent`（包内唯一新增导出符号） |
| `pkg/hno/agent/stream.go` | 修改（恰两行调用面） | `newKernel(...)` 的 `nil` → `emitter.turnObserver(ctx, messageCount)`；`emitter.tasksExecutor(a)` → `emitter.streamExecutor(a)` |
| `pkg/hno/agent/stream_producers_test.go` | 新增（676 行，package agent_test） | TestP4G4P_* 八个测试函数覆盖 D1–D8；自含夹具（p4g4pModel 脚本化流式 fake + weather/boom/scribe 三类处理器），零网络零真模型 |

**kernel.go / run.go / runner_adapter.go / tool_executor.go / agent.go 零字节改动**（OnStep 的
onAssistantTurn 参数与记账时序是现成缝）；**pkg/hno/run 与 pkg/hno/runner 零字节改动**（六事件
构造函数全部复用）；**pkg/hno/graph 零接触、零测试**。

## 3. 既有测试迁移（恰三份，契约 allowedTestSeam.modify 授权）

| 文件 | 授权改动 |
|---|---|
| `runstreammode_test.go` | unsupported 名单五模式 → 未知序数 `run.StreamMode(99)`、`run.StreamMode(7)`；mixed 子用例 StreamValues → 99；注释补迁移说明。循环体三判据零改动 |
| `stream_tasks_test.go` | unwired 名单同上；「debug is not accepted as a tasks alias」转正为「debug is wired as the checkpoints∪tasks family」（断言流启动，完整并集判据在本片新文件）；p4s31 夹具符号零改动 |
| `p7g9s2_agent_span_test.go` | D6 负控探针 StreamValues → `run.StreamMode(99)`（断言形状零改动，仅模式名同步） |

其余 38 份既有测试文件 + 41 条锚逐字节同哈希（见 §4）。

## 4. 结构读数（契约 D9，收口 receipt:8837f5d3-5a55-4f59-9f07-8569bd8c3057，明文复跑同值）

| 指标 | 开工 | 收口 | 判据 |
|---|---|---|---|
| run 包（exported/nontestLOC/allLOC/sync） | 77/889/1723/1 | 77/889/1723/1 | 零改动 ✓ |
| runner 包 | 23/582/1601/0 | 23/582/1601/0 | 零改动 ✓ |
| agent 导出符号 | 21 | **22** | 恰 +1 = WriteCustomEvent ✓ |
| agent 非测试 LOC | 1749 | **1894** | ≤ 天花板 1920 ✓（余量 26） |
| agent 非测试 sync. | 3 | **4** | 恰 +1 = customWriter.Mutex ✓ |
| 三族构造函数非 run 包调用点 | 0 | 4（全在 stream_producers.go:147/152/158/242） | 只落 agent ✓ |
| graph 目录 manifest md5 | f8f2feed8ee5226b4f3a08919ecf6ff2 | 同值（receipt:f7885f4b 尾段） | 前后同命令同输出 ✓ |
| 锚 41 条 | 契约 baselineBytes | 逐字节全同（diff 实测 41/41；receipt:f7885f4b） | ✓ |
| changedBefore 5 条 | 开工真值 | 全部变动，且 git status 改动面恰为 5 修改 + 2 新增 | ✓ |

## 5. D 行 → 实测落点

| 行 | 测试 | 关键读数 |
|---|---|---|
| D1 | FiveModesAreWiredAndStartStreams | 五模式各 err==nil、result 非 nil、streamCalls≥1 |
| D2 | UnknownModesStillFailClosed | 序数 99/7 三判据 + 「stream mode 99」文案形状 + 混选整体 fail-closed |
| D3 | UpdatesEmitPerTurnDelta | `[state_update ×2]`；patch 逐字节 `{"delta":"","turn":1}` / `{"delta":"final","turn":2}`；Node 空；账本 = 通道+run_completed |
| D4 | ValuesEmitRunningStatePerTurn | `{"content":"","messages":2,"turn":1}` / `{"content":"final","messages":5,"turn":2}`；Updates+Values 同场键形互异 |
| D5 | CheckpointsEmitPerTurnSnapshot | label turn-1/turn-2；payload==values 快照；三族并选 6 条逐位 [update,values,checkpoint]×2 |
| D6 | DebugIsTheCheckpointsTasksUnion | Debug-only `[checkpoint, started×2, completed, task_error, checkpoint]` 无 run_content；Debug+Checkpoints 等长；Debug+Messages 回合 2 checkpoint 落 content 后 |
| D7 | CustomWriterWritesFromToolHandlers | handler 写两条 `[custom ×2]`（subtype/data 逐等）；bare ctx 与 Messages-only 写入都报错 |
| D8 | MessagesAndTasksBehaviorUnchanged | Messages `[run_content ×2]`（序列号 0/1）、Tasks 四条、并集六条、RunStream 包装不变 |
