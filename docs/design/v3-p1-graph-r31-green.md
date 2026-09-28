# 切片 31 第一段 + 第二段 GREEN：`StreamTasks` 生产者接线（P4 第 2 片）

契约：`docs/design/v3-test-scope-p1-graph-slice31.json`（D1–D10，两段制 redProtocol）。
绑定场景命令（所有收口回执逐字这一条，cwd 仓库根）：

```
go test ./pkg/hno/agent -run TestP4S31_ -count=1 -race -v
```

回执工具只记 exit 与输出哈希，因此每个「读到的值」都由**同一条命令的明文复跑**给出，
文件路径逐条列出；两条不成对时以明文文件为准（纪律来自状态文档 §5 的两次脑补教训）。

## 1. 两段边界（契约 completionCriteria 1）

| 段 | 内容 | 回执 | 结果 |
|---|---|---|---|
| 段 0 | 只有测试文件，生产字节为开工态 | `receipt:3deb2151-60e5-4a52-814a-09292bfae800` | exit 1，D1–D6/D8/D9 全红（D7 绿：既有用为绿） |
| 段 1 | 模式表 + `streamEmitter` + Tasks 分支最小可编译形状 | `receipt:4781e1a7-c84c-4f2b-88d2-bb6523e00e8c` | exit 1，**D1/D2/D7 绿，D3/D4/D5/D6/D8/D9 行为红** |
| 段 2 | 精确分类 + 批次次序 + 截断静默 | `receipt:d5b15e9f-f846-4997-a1bc-934704397187` | exit 0，9 条顶级行全绿 |

段 0→段 1→段 2 之间**测试文件零改动**：收口哈希 `107e4b9cd8c0ef3b9f82b4b36f04d8f83b03a7d5`，
且段 1 的判据行号（315/355/465/490/530/629/634/678）与收口文件里同一批断言所在行**逐一对应**
（分别落在 D3、D4、D5 两段、D6、D8 两段、D9 之内）——行号对齐是机器可核的第二条担保，
不靠「我记得没改过」。

段 1 的六条红必须是**行为红而非前提红/守卫红**，实测判据行（`/tmp/s31/seg1-raw.txt`）：

```
stream_tasks_test.go:315: stream kinds = [] (0 events), want [node_started node_started node_completed task_error] (4 events)
stream_tasks_test.go:355: stream kinds = [] (0 events), want [node_started node_started node_completed task_error] (4 events)
stream_tasks_test.go:465: stream kinds = [] (0 events), want [node_started task_error] (2 events)
stream_tasks_test.go:490: stream kinds = [] (0 events), want [node_started node_completed] (2 events)
stream_tasks_test.go:530: stream kinds = [] (0 events), want [... 8 events)
stream_tasks_test.go:629: stream kinds = [run_content run_content] (2 events), want [node_started node_started node_completed task_error run_content run_content] (6 events)
stream_tasks_test.go:634: stream kinds = [] (0 events), want [node_started node_started node_completed task_error] (4 events)
stream_tasks_test.go:678: stream kinds = [] (0 events), want [node_started node_completed] (2 events)
```

零事件上通道 = 「今天 Tasks 一律 fail-closed、通道上任何任务事件都不存在」，与契约 M1 的
预演形状一致；无一条是夹具守卫（本夹具的守卫文案只有 `InvokeStream` 超脚本时报的 fixture bug，
段 1 输出里 0 次出现）。

## 2. 收口门禁（全部 exit 0，最终字节）

| 门禁 | 回执 | 明文读数 |
|---|---|---|
| 绑定命令 `-count=1 -race -v` | `receipt:d5b15e9f-f846-4997-a1bc-934704397187` | `/tmp/s31/final/r1-raw.txt`：9 条 `--- PASS: TestP4S31_`、0 条 FAIL、0 条 `DATA RACE` |
| 防碰巧绿 `-count=30 -race -v` | `receipt:94ad35fd-f482-403e-b428-7ab3fbddeb25` | `/tmp/s31/final/r2-raw.txt`：270 条 PASS（9×30）、0 FAIL、0 `DATA RACE` |
| `./pkg/hno/agent ./pkg/hno/run -count=5 -race` | `receipt:a34fc717-b50e-4432-aba0-a43a88449395` | exit 0 |
| `go build ./...` | `receipt:74982c41-a205-4015-bc33-dc59cb6101c8` | exit 0 |
| 三包 `-count=1` | `receipt:4184d4e4-9f93-4565-9410-febaf296c74c` | run/agent/runner 三行 `ok` |
| 三包 `-race -count=1` | `receipt:d8f0127d-73cf-4083-9fe1-3e4402cc47c7` | 三行 `ok`，无 `DATA RACE` |
| `go vet` 三包 | `receipt:768dd0a9-d193-4324-8e07-2aa026ee39f1` | 空输出 |
| `gofmt -l` 三包 | `receipt:12fdb6e5-2013-4e1a-8e8e-8ec46de7f79e` | 空输出（新增两文件同判） |
| 邻近消费面 team/workflow/session-contract | `receipt:81b5953c-0568-4eb3-9b49-a4d324d53394` | 三行 `ok`（工具批次被它们消费，实测不红） |

契约命令表第 3 条明写「不得跑 `pkg/hno/graph` 测试」，本片全程未对该包执行任何 test/build 动作。

## 3. 结构判据（D10）

`receipt:6d88cf17-3b8f-41d5-88bb-859fd47bdd8a`（命令与契约 verificationCommands[3] 逐字同形）
的明文复跑读数：

```
== run      exported=77 nontestLOC=889 allLOC=1723 syncAll=1 syncNontest=1
== agent    exported=21 nontestLOC=1727 allLOC=6217 syncAll=5 syncNontest=3
== runner   exported=23 nontestLOC=582 allLOC=1601 syncAll=0 syncNontest=0
```

- run / runner 与契约等值目标逐位相同（`77/889/1723/1`、`23/582/1601/0`）。
- agent `exported=21` ⇒ **本片新增导出符号 0**（`go doc -all` 计数差值，非目测；与契约
  `receipt:9826367c` 基线同值）。
- agent 非测试 LOC `1727` = 契约上限 1727，**贴顶通过**。实现初稿实测 1731 超限 4 行，
  处置是压缩 `stream_producers.go` 里三处双语注释的冗余（`wiredStreamModes` /
  `normalizeStreamModes` / `taskToolExecutor` / `streamEmitter`），不改判据、不申请放宽上限。
- 新文件 `sync.` 计数 0；agent 非测试 `sync.` 实测 **3** 而非契约预测 4 —— 见 §5 口径登记 (1)。
- 任务族构造点（`receipt:9e06fb0e-180d-4df7-87f0-8c1c90a49f20`，开工实测为 0）收口后只落在
  `pkg/hno/agent/stream_producers.go:95/104/106`，`pkg` 内其余命中全在 `_test`。

## 4. 字节锚（三段核对）

| 集合 | 期望 | 实测 |
|---|---|---|
| `baselineBytes.anchors`（31 份） | 逐字节不动 | **31/31 全同**（采集 `receipt:98498dcc-a45a-4cc8-bb80-370457e56248`） |
| `changedBefore`（7 份） | 全部变动且差异只落在 sourceSeam | **7/7 变动**（before→after 见 §6 表） |
| `outOfSeamNoChangeExpected`（5 份） | 不动 | **5/5 同哈希** |
| `newFileExpectedAbsent`（2 份） | 只允许这两个新增名 | `git status --porcelain pkg/hno/agent` 只多出 `stream_producers.go` 与 `stream_tasks_test.go` |
| 图目录清单 md5 | 契约锚 `ae3c22445eb8167f500ad832513046ca` | 实测 `639f37dbb16f4a20ce2e796e4586caac` ⇒ **漂移已定位来路，非本片所为**，见 §5 登记 (2) |

## 5. 两条口径登记（不靠重算蒙过去）

**(1) agent 非测试 `sync.` 4 → 3。** 契约预测 4 的口径是「既有的三处 + 新生产者一处」。实际
落位把 `executeToolCalls` 与 `agentToolExecutor.Execute` 收拢进同一个 `runToolBatch`
（`tool_executor.go:51-85`），`WaitGroup` 从两处变一处。锁计数**严格优于**预测，本片意图
（不新增锁、并发批次不加序列化）达成；登记为口径偏差而不是偷偷重算成 4。

**(2) 图目录锚值漂移。** 用同一条命令在历史提交上复算：
`b44b588`（契约起草时的 HEAD）⇒ `ae3c22445eb8167f500ad832513046ca` **逐字符复现**；
`HEAD`（`4d58ea5`，含兄弟片 `0387000` 落进 `pkg/hno/graph/` 的 scheduler 加固与新增 R17 测试）
⇒ `639f37dbb16f4a20ce2e796e4586caac`。同时 `git status --porcelain pkg/hno/graph` 与
`git diff --numstat HEAD -- pkg/hno/graph` 双双为空 ⇒ 本片实现窗口内图目录字节零变动。
结论：锚的绝对常数会被兄弟片合法落地作废；后续任何用该 md5 的片必须按
「**实现前后同一条命令同输出**」判，并把起草时的 HEAD 一起写进契约。采集回执
`receipt:e90876aa-5e1c-4c82-93e8-2f31b54ed754`。

## 6. 最终字节清单（`git hash-object --no-filters`）

| 文件 | 开工（契约 changedBefore） | 收口 |
|---|---|---|
| `pkg/hno/agent/agent.go` | `9813b86af2e7c455dec6c7c36f187f0bf7627da0` | `54b5dad514d280f89e1587d11875d3602abc528a` |
| `pkg/hno/agent/stream.go` | `bd3583dc512b0aa7c81874e0cb566ba42b36ebe4` | `4abb334bb9cecd11b98b4db28f7a378e8fccbfb5` |
| `pkg/hno/agent/kernel.go` | `6140b5803e44c534264dd807c0cc090e98330e7f` | `f68fa51023ec8b1237247ba2a67b77240f27b4a4` |
| `pkg/hno/agent/runner_adapter.go` | `8a7f36da27f0f186f86591a76b91c36793f8f386` | `9f05be715e8c36b361184c76083710196e66381d` |
| `pkg/hno/agent/tool_executor.go` | `c2e9ec1c85cf889c5d8f55c0192deb28719bafd2` | `22f2228fe39e5475f00b35d14c8ec923d89364ec` |
| `pkg/hno/agent/run.go` | `8c2dea551cefd7035822879d568af318691a63df` | `b445f570b705961aadbe166c7b3fe600c8b65ea6` |
| `pkg/hno/agent/runstreammode_test.go` | `6c809c23d81949b196fd213ab5199d74047ebdab` | `7c3599763fabbfc193e4e3f3c6b4a293c109d06d`（`+1/-1`：删 `run.StreamTasks,` + 一行迁移注释） |
| `pkg/hno/agent/stream_producers.go` | 不存在 | `2f16a32b060338110c379ece747fe8d846956e9e`（145 行） |
| `pkg/hno/agent/stream_tasks_test.go` | 不存在 | `107e4b9cd8c0ef3b9f82b4b36f04d8f83b03a7d5`（693 行） |
| `scripts/mutation/p4s31-tasks-producer.mjs` | 不存在 | `a5488069e66a9ad53d4568711847d8f5508e79b5`（矩阵执行体） |

`git diff --numstat HEAD -- pkg/hno/agent` 的七行改动量：
`agent.go 12/15`、`kernel.go 12/1`、`run.go 1/1`、`runner_adapter.go 1/35`、
`runstreammode_test.go 1/1`、`stream.go 28/30`、`tool_executor.go 59/16` —— 净增来自
`runToolBatch` 的收拢与新文件，`runner_adapter.go` 的 `-35` 是把原批次实现搬进
`tool_executor.go`，不是删除行为。
