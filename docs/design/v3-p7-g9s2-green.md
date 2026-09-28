# 切片 32 GREEN（P7 第 2 片，母约 G9：run/agent 级 invoke_agent span）

工作项 `work-p7-observability` / 契约 `docs/design/v3-test-scope-p7-g9-agent-span.json`（D1–D11）。
RED 观察与两段制披露见 `docs/design/v3-red-observation-p7g9s2.md`。本片绑定场景命令逐字：

```bash
go test ./pkg/hno/agent -run TestP7G9S2_ -count=1 -race -v
```

## 1. GREEN 判据与实测

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| D1–D8 全绿（契约绑定命令，逐字，-race -v） | 同上 | 0 | `receipt:5363712d-86d3-4446-864c-a8f7e412dde7`（8 `--- PASS` / 0 `--- FAIL`） |
| 绑定命令裸退码形式 | `go test ./pkg/hno/agent -run TestP7G9S2_ -count=1 -race` | 0 | `receipt:ee5fe579-bfe0-4537-8669-1d6da5a3e497` |
| 防碰巧绿：绑定面 `-race -count=30` | `go test ./pkg/hno/agent -race -count=30` | 0 | `receipt:534b4656-ea26-4a64-ab65-1a8c6a5aec30` |
| 邻居门：runner + observability `-race` | `go test ./pkg/hno/runner ./pkg/hno/observability -race` | 0 | `receipt:76e7ef20-4178-496e-9ee3-b275eb3e16f1` |
| 既有测试全量（含 15 份既有测试文件） | `go test ./pkg/hno/agent -count=1` | 0 | `receipt:99c2989f-d21f-4926-9bfe-c662fe56103c` |
| 既有测试全量 -race | `go test -race ./pkg/hno/agent -count=1` | 0 | `receipt:af65f645-1cd8-4a19-bbe6-21a0079b56da` |
| build 三门包 | `go build ./pkg/hno/agent ./pkg/hno/runner ./pkg/hno/observability` | 0 | `receipt:37e62120-51b3-475f-bdcf-6b037141b431` |
| vet 三门包 | `go vet ./pkg/hno/agent ./pkg/hno/runner ./pkg/hno/observability` | 0 | `receipt:d9fd3228-6b03-4d95-9f6f-0916278cda86` |
| `gofmt -l pkg/hno/agent` | 无输出（0 行） | 0 | （非 receipt 记录） |

⚠️ build 门偏差申报：契约 verificationCommands 第 3 条写 `go build ./...`；编排方硬约束
「兄弟片正在改 pkg/hno/store 与 pkg/hno/workflow+graph，不跑它们的测试、不跑 `./...`」，
故 build/vet 收窄到三个门包（agent/runner/observability）——本片改动只在 pkg/hno/agent，
且该包的下游消费面（agentos/examples）不 import 本缝的新私有符号，收窄不掩盖任何本片字节。

## 2. 实现差异（恰在 sourceSeam 声明范围内）

| 文件 | 状态 | 内容 |
|---|---|---|
| `pkg/hno/agent/kernel.go` | 改（94 → 117 行；`f68fa510…` → `1114030e…`） | 唯一新增符号位点：私有方法 `runKernel(ctx, runID, *runner.Runner, []*types.Message)`，体内三行 = StartAgentSpan / defer span.End() / return r.Run(spanCtx, messages)；import 块加 pkg/hno/observability（包内已有该依赖，无新边） |
| `pkg/hno/agent/run.go` | 改（行中性 191；`b445f570…` → `d0d975b9…`） | :132 一行改为经 `a.runKernel(ctx, runID, r, messages)` |
| `pkg/hno/agent/stream.go` | 改（行中性 284；`4abb334b…` → `51943b69…`） | :134 一行改为经 `a.runKernel(ctx, runID, r, …)`（位于既有 goroutine :106 之内） |
| `pkg/hno/agent/p7g9s2_agent_span_test.go` | 新增（563 行，package agent_test 黑盒） | D1–D8 测试：8 个测试函数、4 个 t.Run 子场景、t.Fatal 31 / t.Error 19；夹具 = 手写稳定 fake model（同步/流式两形脚本、工具调用与 usage 固定）+ 手写 weather toolkit + tracetest.InMemoryExporter，零网络零真模型 |
| `scripts/mutation/p7g9s2-agent-span.mjs` | 新增 | 本片变异执行体，8 杀红 + 等价/缺口各 1（见 refactor 文档 §2） |

**零既有测试改动**（allowedTestSeam.modify 为空的判据）；**pkg/hno/observability 与
pkg/hno/runner 零字节改动**（锚对账见 §3）——零件已存在，本片只补调用点。

## 3. 锚与结构（契约判据 2/5，全部在最终字节上）

锚对账（`git hash-object --no-filters`，逐字节）：

- 8 份锚（observability 2 + runner 2 + 发射缝 stream_producers.go 及其 2 份测试 + tool_executor.go）：**8/8 与契约 baselineBytes.anchors 逐字节全同**——尤其切片 31 的发射缝 `2f16a32b…` 与 `stream_tasks_test.go 107e4b9c…`、`runstreammode_test.go 7c359976…` 未动。
- 8 份 out-of-seam（agent 包内其余非测试文件）：**8/8 与契约 outOfSeamNoChangeExpected 逐字节全同**。

结构读数（逐条实读到值）：

| 读数 | 值 | 契约要求 |
|---|---|---|
| 包内非测试 observability.StartAgentSpan 站点 | 1（仅 kernel.go） | 1 |
| run.go / stream.go 的 `r.Run(` | 0 / 0 | 0/0 |
| run.go / stream.go 的 `a.runKernel(` | 1 / 1 | 1/1 |
| kernel.go 七类 sync 原语 | 0×7 | 全 0 |
| kernel.go `go func` | 0 | 0 |
| stream.go `go func` 站点 | 2（与基线同；本片零新增 goroutine） | 2 |
| `go doc -all ./pkg/hno/agent` 导出条目 | 21 | 21（新增导出符号 0） |
| 行首 `t.Parallel` 计数（整包） | 0 | 0 |
| 包内非测试 sync 原语总数 | 3（与基线同） | 3 |
| agent 非测试 LOC | **1750**（kernel 117 / run 191 / stream 284） | ≤ 1757（预演 1750；+23 全在 kernel.go，run/stream 行中性） |

零 tracer 代价（同命令 `-benchtime=200x -count=2` 在 HEAD 副本与最终字节副本各两轮，
非判据、不为省它引入 guard）：基线 **326 allocs/op、43721–43754 B/op**；最终
**332 allocs/op、44195–44231 B/op** ⇒ 固定开销 **+6 allocs、≈ +441…510 B/op（≈1.1%）**；
ns/op 本机噪声大，按未加权指标登记。no-op span 的 End 在 defer 中被真实调用而全程无
panic（D8 两侧各一次运行即是其执行证据）。

## 4. 每行由哪段实现担保

| 行 | 断言的公共事实 | 实现落点 | 测试 |
|---|---|---|---|
| D1 | 一次被驱动的同步运行发且只发一条 invoke_agent；agent.name=配置名（与自动 id 可分辨）、agent.run_id=RunOutput.RunID、gen_ai.system=hno、已结束 | runKernel 的 StartAgentSpan(ctx, a.Name, runID) | RunEmitsOneInvokeAgentSpanWithIdentityAttributes |
| D2 | 同步路径父子链：chat×2 + execute_tool×1 全部 ChildOf invoke_agent 且同 TraceID；子属性不被父覆写 | spanCtx 交给 r.Run → runner 逐尝试 chat span（resilience.go:40）与本包 execute_tool span（tool_executor.go:103）天然成子 | SyncPathParentsChatAndToolSpans |
| D3 | 流式（Messages）跨 goroutine 认父 + 存活期包含 + agent.run_id==done.Output.RunID | stream.go :134 的调用点在既有 goroutine 内经同一缝 | StreamPathParentsChatSpansAcrossTheGoroutine |
| D4 | StreamTasks 的并发工具批 execute_tool 仍认父（缝不与发射缝争抢 ctx） | 同一 spanCtx 经 emitter.tasksExecutor(a) 传入生产者 | TasksStreamToolSpanIsAChildToo（含通道事件活性正控） |
| D5 | 父时钟区间包住每个子（sync + stream 两场景）——把「驱动点之外 End」证伪的互补牙齿 | span 的开合都在 runKernel 内、与驱动重合 | AgentSpanContainsEveryChild（sync/stream 两子场景，p7g9s2Contains） |
| D6 | 未接线模式 fail-closed 且 span 零增量（正控 1 条 → 未接线后仍 1 条、chat 仍 1 条） | fail-closed 在 runStreamMode 归一阶段（agent.go:128），先于任何驱动，缝无需分支 | UnwiredStreamModeOpensNoSpan |
| D7 | span 存活期跟随驱动：被放弃且不取消 → 已结束 span 为 0（宁缺不假报）；取消 → 恰 1 条、已结束、chat 认父 | defer span.End() 由驱动拥有 | SpanLifetimeFollowsTheDrive（abandoned/cancel 两子场景） |
| D8 | 零 tracer 零扰动：两全局 provider 状态下掩码后 RunOutput JSON 逐字节相等；SDK 侧恰 1 条为正控 | 无条件发射（零 guard），no-op span 零外溢 | NoopTracerKeepsWireBytesIdentical（p7g9s2TracerInstalled 前置探针） |
| D9/D10/D11 | 结构门 / tracer 隔离 / 邻居与只读门 | 见 §3 表与锚对账 | 结构读数 + grep/go doc/wc + 邻居 receipt |

## 5. 本阶段未闭环

- 变异矩阵结果与契约逐条对账见 `docs/design/v3-p7-g9s2-refactor.md`。
- `golangci-lint` 欠项沿用；切片 2–32 未提交沿用（S18-SPEC-2）。
- 写回：母约 §10 P7 行与 §3 G9 行已改（P7 四级全清）；切片 25 契约的 S25-DEFER-1 结清
  登记与状态文档 v3-p1-graph-status.md 条目由编排方收口（本契约 writeBackOwed 第 3/4 条）。
