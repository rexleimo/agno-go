# 切片 25 GREEN（母约 G9：观测接线）

工作项 `work-p7-g9-observability` / 契约 `docs/design/v3-test-scope-p7-g9-observability.json`（D1–D13）。
RED 观察见 `docs/design/v3-red-observation-p7g9.md`；变异矩阵与契约对账见
`docs/design/v3-p7-g9-refactor.md`。本片绑定场景命令逐字：

```bash
go test ./pkg/hno/runner -run 'TestP7G9_' -count=1 -race -v
```

## 1. GREEN 判据与实测

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| D1–D13 全绿（契约绑定命令，逐字） | `go test ./pkg/hno/runner -run 'TestP7G9_' -count=1 -race -v` | 0 | `receipt:669c0c6d-251d-46d9-8d06-5390310c0d7e` |
| 前一步 RED（redProtocol 第一段，调用点未接线） | 同上 | 1 | `receipt:aa8c6103-3d91-4a2f-ba74-1e9050c90275` |

`-v` 逐行：13 个 Test 全部 `--- PASS`，末行 `ok github.com/rexleimo/agno-go/pkg/hno/runner`。
RED→GREEN 之间只有实现一行（调用点 `r.invoker.InvokeTurn` → `r.invokeTurn`），测试文件零改动。

## 2. 实现差异（redProtocol 第二段：接线）

基线（契约 baselineBytes，切片 24 后收口态）→ 最终：

| 文件 | 基线 | 最终 | 变更 |
|---|---|---|---|
| `pkg/hno/runner/runner.go` | `970c701d…` | `18fe1f80…` | Config 增两个可选字段（Retry/Breaker，各带 GoDoc 语义）；Runner 增四个私有字段；`New` 接线（provider/modelName 取自必填 Model）；`StateAwaitModel` 调用点换 `r.invokeTurn` |
| `pkg/hno/runner/resilience.go` | 无 | `2d6c944c…` | 本片新缝：`clampAttempts`（guard 钳制）+ `invokeTurn`（breaker 门禁 → 每尝试 chat span → 重试循环 → 一次逻辑熔断记账），全私有、零锁 |
| `pkg/hno/runner/p7g9_resilience_test.go` | 无 | `1c25b730…` | 本片新测试，D1–D13（契约 allowedTestSeam.add） |
| `scripts/mutation/p7g9-observability.mjs` | 无 | — | 本片变异执行体（9 杀红 + 2 等价登记） |

实现要点（契约 designConsequence 八条的落点）：

1. **挂载边界**：Run 的 StateAwaitModel 里唯一 TurnInvoker 调用点；同步与流式共用这条缝
   （流式调用方把聚合藏在自己的 Invoker 里，缝在 Invoker 边界之外天然覆盖两路）；
   工具阶段不包（工具韧性归 ToolExecutor 属主，以既有 execute_tool span 反向对照保护）。
2. **零值即无操作**：nil Retry=恰一次、nil Breaker=不设门；MaxAttempts<1 在缝内钳制为 1 次
   （沿切片 22 WithMaxConcurrency 的 guard 钳制先例；裸 observability.Retry 对 0 是零迭代
   静默成功——M1 SHAPE-2——钳制是接线的责任，不是对零件的修改）。零新增 Validate 行。
3. **失败语义**：Run 独占唯一 wrap 点 `runner: model invoke: %w`——重试耗尽交最后一次错误、
   经这一个 wrap 交出（不双包，StopReason 维持 model_failure）；breaker-open 由缝内返回
   **裸 `observability.ErrOpen` 哨兵**，Run 的 wrap 是唯一前缀、errors.Is 可达
   （M2 实测的「model invoke 双前缀」缺陷以此定形修正）。
4. **breaker 与 retry 的组合**：门禁在重试循环之外（open 不重试、不烧尝试）；记账在重试循环
   之后、每逻辑回合恰一次 Success/Failure（重试把瞬态故障挡在熔断记账之外）。
5. **chat span 按尝试计**：每次尝试都是真实模型调用——失败尝试 `RecordError`，有响应才
   `SetUsage`；sink 就是 OTel 全局 TracerProvider（经既有 `observability.Tracer()/StartChatSpan/
   SetUsage` 到达，no-op 默认零开销），不新设 sink 接口。
6. **回合预算**：重试内嵌于单个逻辑回合，turn++/MaxTurns 记数不动（沿切片 22「重试不消耗
   步数预算」同一原则）；span 属性 provider/model 取自 Config.Model 的 GetProvider/GetName。

## 3. 门禁（契约判据 4/5/6/7，全部在最终字节上）

| 门禁 | 结果 | receipt |
|---|---|---|
| `go build ./pkg/hno/runner/... ./pkg/hno/observability/...` | exit 0 | （非 receipt 记录） |
| `go test ./pkg/hno/runner/... ./pkg/hno/observability/... -count=1` | `ok` exit 0 | `receipt:afbe63e4-6205-438b-8c88-772cd6a60168` |
| `go test -race ./pkg/hno/runner/... ./pkg/hno/observability/... -count=1` | `ok` exit 0，无 DATA RACE | `receipt:c17f4bd4-ca29-4008-b933-da49b26277ff` |
| `go vet ./pkg/hno/runner/... ./pkg/hno/observability/...` | exit 0 | （非 receipt 记录） |
| `gofmt -l pkg/hno/runner pkg/hno/observability` | 无输出 | （非 receipt 记录） |
| 邻近回归 agent/ graph/ internal/session/contract `-count=1` | 三组全 `ok` exit 0 | `receipt:8e7b5d10-7120-4112-bc59-b9ce7101ebac` |
| 防碰巧绿：绑定命令 `-count=30 -race` | `ok` exit 0 | `receipt:908c02ff-499c-4b68-abb0-31056ac97e0d` |
| 防碰巧绿：两包整包 `-count=5 -race` | `ok` exit 0 | `receipt:26bc6f79-0c68-4599-bc5b-4682d68d9d65` |
| runner 导出符号（`go doc -all` 口径） | **23**（判据恰 23，零新增导出） | `go doc -all ./pkg/hno/runner \| grep -cE '^(func\|type\|var\|const) '` |
| runner 非测试 LOC | **582**（判据 ≤700；runner.go 400 + resilience.go 69 + 既有 113） | `find … xargs wc -l` |
| 新缝文件 `sync.` 计数 | **0**（breaker 由调用方持有传入，runner 不新增锁） | `grep -c 'sync\.' resilience.go` |

既有文件逐字节复核（`git hash-object --no-filters`，与契约 baselineBytes 全同）：
`runner_test.go 15d85c1f…`、`state.go 194c8992…`、`stop.go a62d85e7…`、`tool_batch.go 21984492…`、
`tool_limit_run_test.go ab2d2d28…`、observability 九文件（`breaker ac4fb9b8…`、`cost 3ef5556a…`、
`cost_test 0c29f6d3…`、`prompt_cache 1d460fd0…`、`ratelimit 92a23db3…`、`resilience_test 55b4ff38…`、
`retry f2c2568e…`、`tracing ccf7014a…`、`tracing_test d83c09f5…`）——observability/models/types
零字节改动。

## 4. 每行由哪段实现担保

| 行 | 断言的公共事实 | 实现落点 |
|---|---|---|
| D1/D3 | 重试至成功 / 耗尽交末错且 wrap 恰一次 | invokeTurn 的重试循环 + Run 唯一 wrap 点 |
| D2/D11 | 零值恰 1 次、错误串同形（禁用观测零开销） | nil Retry/Breaker 的零值直通路径 |
| D4 | MaxAttempts=0 钳制为真实 1 次（不静默成功） | clampAttempts guard 钳制 |
| D5/D7 | 熔断开断 fail-fast（ErrOpen、调用冻结）/ open 不进重试 | 循环外的 breaker.Allow() 门禁 + 裸哨兵 |
| D6 | 冷却后放行探测、成功复位 | Allow() 的 half-open 探测 + Success() 记账 |
| D8 | 熔断按逻辑回合记账（每回合恰 1 次） | 重试循环之后的 Success/Failure |
| D9/D10 | model 级 chat span 带 usage / 按尝试计且失败记 error | 尝试闭包内 StartChatSpan + SetUsage/RecordError |
| D12/D13 | 自定义 Invoker 同样受益 / 重试不耗 MaxTurns | 缝在 TurnInvoker 边界 + 回合内嵌重试 |
