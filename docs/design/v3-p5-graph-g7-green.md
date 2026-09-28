# 切片 27 GREEN（母约 §9 G7：图引擎 HITL Interrupt/Resume 核心）

工作项 `work-p5-graph-g7-hitl-core` / 契约 `docs/design/v3-test-scope-p1-graph-slice27.json`（D1–D15）。
RED 观察见 `docs/design/v3-red-observation-p5g7.md`；实现差异、变异矩阵与契约对账见
`docs/design/v3-p5-graph-g7-refactor.md`。本片绑定场景命令逐字：

```bash
go test ./pkg/hno/graph -run TestP5G7_ -count=1 -race -v
```

## 1. GREEN 判据与实测

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| D1–D15 全绿（契约绑定命令，逐字，最终字节） | `go test ./pkg/hno/graph -run TestP5G7_ -count=1 -race -v` | 0 | `receipt:ad6c744c-e211-442f-8232-f088ebf3d295` |
| 前一步 RED（redProtocol 第一段，调度器不消费） | 同上 | 1 | `receipt:e428c1fc-090c-4fc6-b48a-af4242e59762` |

`-v` 逐行：15 个 Test（含 D5/D7/D11/D13 的子用例）全部 `--- PASS`，末行
`ok github.com/rexleimo/agno-go/pkg/hno/graph`。RED→GREEN 之间只有实现字节（`hitl.go` 的
`Resume` 实体与 `newScheduler`、`scheduler.go` 的挂起捕获/排干/组装、`durability.go` 的
`commit` 抽取与挂起路径 sink 并入、`graph.go` 的 `pending` 字段），测试文件零改动
（blob `7587689b…` 在 e428c1fc 与 ad6c744c 之间逐字节一致）。

## 2. 实现差异（redProtocol 第二段：接线）

基线（切片 26 收口态）→ 最终：

| 文件 | 基线 | 最终 | 变更 |
|---|---|---|---|
| `pkg/hno/graph/hitl.go` | 无 | `75756a14…` | 新增：`Interrupt/InterruptMode` 两档 + `Suspension/WaitingNode` + 三哨兵 + `RequestInterrupt/InterruptResponse` + `Resume` 实体 + `newScheduler`（Run/Resume 共用构造点）+ `validateResponse` 子集校验（291 行） |
| `pkg/hno/graph/scheduler.go` | `fcd552c5…` | `dfd6c1ba…` | 挂起态三字段（消费者独占）；`Run` 改走 `newScheduler` 并捕获挂起产物入 `g.pending`；consume 辨认中断载体→`suspend` 入册、排干期派发冻结、`running==0 && suspended` 交 `suspensionError`；`runNode` 对中断短路（重试不吞） |
| `pkg/hno/graph/durability.go` | `da57a17e…` | `11075cfe…` | `EntryKind` 两常量 + `Checkpoint` 扩展 `Kind/Input/Interrupt`；`record` 的档位开关抽取为 `commit`（完成条目与中断条目同一时机语义）；`finishCommits` 挂起路径 sink 错误 `errors.Join` 并入（双可达） |
| `pkg/hno/graph/graph.go` | `7d7bb8d8…` | `efa9a4ba…` | `Graph` 加 `pending *Suspension` 一字段（挂起账，调用链独占写） |
| `pkg/hno/graph/p5g7_hitl_test.go` | 无 | `7587689b…` | 本片新测试，D1–D15（契约 allowedTestSeam.add） |

实现要点（契约 designConsequence 七条语义的落点）：

1. **中断面是哨兵错误**：`RequestInterrupt(i)` 返回 `*interruptSignal` 作为 `Node.Run` 的 error，
   穿过既有 `queueItem.err` 通道抵达唯一消费者——生产者零写调度器字段（零锁不破），`Node.Run`
   签名零改动（与 Sender 同一可选扩展先例）。
2. **挂起形状 fail-closed**：首个中断在 consume 被辨认后派发冻结、在途节点排干收账，`running==0`
   时组装 `*Suspension`（全部待答 + 已完成 `Values/Completed` 字典序 + 内部账目 `seq/steps`），
   `Run` 交 `(nil, suspension)`——无半截 Result（R19-Q1 形状规避）。
3. **Resume 三语义**：响应集必须恰好覆盖全部待答中断（缺 → 包 `ErrInvalidResponse` 点名 ID、
   多 → 未知 ID 错误，均在消费挂起账之前返回，挂起保留）；逐中断 `validateResponse`（诚实子集，
   超集声明报不支持）；按 `Mode` 归 Rerun（`In` 原样重入，响应经 ctx 取值器）或 Handoff
   （走 complete+record 同一条记账路径，节点体不执行）。恢复可再次挂起（重悬链）。
4. **重试不吞中断**：`runNode` 在 `pol.retry.should` 判定之前对 `interruptFrom(err)` 短路——
   重试不重发、退避不占用（D8 实测 calls==1）。
5. **持久化复用切片 24**：挂起条目 `EntryInterrupt`（Node/Input/Interrupt 全量）经 `commit`
   走同一三档时机语义；sink 失败不互掩——Sync 在 `suspensionError` 处、Async/Exit 在
   `finishCommits` 处 `errors.Join`（`errors.Is` 对 `ErrSuspended` 与 sink 错误同时成立）。
6. **预算跨恢复累计**：`Suspension` 携带 `seq/steps`，`Resume` 以 `newScheduler(ctx, seq, steps)`
   重建——一次逻辑运行一个预算（D12 恢复段撞 `ErrStepLimitExceeded`）。
7. **并发纪律沿用 builder 口径**：同一张图上的 Run/Resume 调用链不并发；挂起账（`g.pending`）
   由调用链独占写，不导出任何内部 pending 态。

## 3. 门禁（契约判据 4/5/6/8，全部在最终字节上）

| 门禁 | 结果 | receipt |
|---|---|---|
| `go build ./...` / `go vet ./pkg/hno/graph/...` / `gofmt -l pkg/hno/graph` | exit 0 / exit 0 / 无输出 | （非 receipt 记录） |
| `go test ./pkg/hno/graph/... -count=1` | `ok` exit 0 | `receipt:021f711e-4861-402e-8a30-305d0c679950` |
| `go test -race ./pkg/hno/graph/... -count=1` | `ok` exit 0，无 DATA RACE | `receipt:23f6b4e3-3a17-46d2-961f-1c3369af79d6` |
| 邻近回归（agent/runner/session-contract）`-count=1` | 三包全 `ok` exit 0 | `receipt:60f20a74-4c5b-46ac-9c09-b1eb4022dd07` |
| 防碰巧绿：绑定命令 `-count=30 -race` | `ok` exit 0 | `receipt:207c88b2-a43a-4fce-be85-35a912143292` |
| 防碰巧绿：整包 `-count=5 -race` | `ok` exit 0（一次通过，未撞 S24-STD-2） | `receipt:0dcf9dd7-4a26-49f9-a694-67b673e41725` |
| `sync.` 计数（graph/scheduler/hitl） | **0** | `grep -o … wc -l` |
| 非测试 LOC | **1800**（policy 138 + graph 639 + durability 146 + scheduler 499 + send 87 + hitl 291；判据 ≤1800，取整压线，见 refactor 文档 §1 的字节经济过程） | `find … xargs wc -l` |
| 导出符号 | **60**（判据恰好 60：切片 26 后基线 45 +15——hitl.go 的 11 符号 + 2 个 Suspension 方法 + durability 的 `EntryKind` 类型 + 1 个 const 块；常量缩进不被 `^` 锚 grep 计入） | `go doc -all … grep -cE '^(func|type|var|const) '` |

既有 13 份测试文件与 `policy.go`、`send.go` 逐字节复核（`git hash-object --no-filters`）：
`policy.go 6d71fa47…`、`send.go dcd04661…`（两者 MUST 不动，未动）、`graph_test.go 624f1962…`、
`p1r13 bc4ab8e7…`、`p1r14 0ed607db…`、`p1r15 3ea06d27…`、`p1r15b 0285ab35…`、`p1r16 6934a4f2…`、
`p1r17 cbc8b692…`、`p1r18 10bce61b…`、`p1r19 8b4b7234…`、`p1r21 a8a936ed…`、`p2g3 ec390cc7…`、
`p3g5 c003a537…`、`p3g6 bdeb8ef9…`——与切片 26 收口基线全同，零改动。

## 4. 每行由哪段实现担保

| 行 | 断言的公共事实 | 实现落点 |
|---|---|---|
| D1 | 中断 → `(nil, err)` 且 Is/As 三断言，已完成前驱留在 Suspension | consume 辨认载体 → `suspend` 入册 → `suspensionError` 组装（Values/Completed 与 Result 同源同形） |
| D2 | 校验不过 → `ErrInvalidResponse` 且挂起保留可重试 | `Resume` 的 `validateResponse` 失败路径在 `g.pending` 消费之前返回 |
| D3 | Rerun 带响应重跑、同输入、图跑完 | default 档 `activation{name, in: w.In}` 入 pending；`ctx` 携带响应集 |
| D4 | Handoff 不偷跑节点体、响应即输出喂后继 | Handoff 分支走 `complete+record`（attempt=0），不入激活队列 |
| D5 | 二次/无处恢复 → `ErrNothingToResume` | `g.pending` 消费置 nil + `Resume` 头部 nil 早退 |
| D6 | 响应集恰好覆盖（缺失/未知点名、挂起保留） | known 集合双向核对，均在消费挂起账前返回 |
| D7 | 挂起落 `EntryInterrupt` 全量条目、普通完成 Kind 零值不变 | `suspend` 经 `commit` 落条目；完成条目 `Kind` 恒零值 |
| D8 | 重试不吞中断（calls==1） | `runNode` 的 `interruptFrom` 短路先于 `retry.should` |
| D9 | 并行双中断一个不丢、Join 照常 | `waitings` 累加所有排干期到达的中断；恢复后 joinReady 既有键控路径（S19-STD-1 口径） |
| D10 | 重悬链 | Resume 复用同一 consume：重入体再中断 → 新 Suspension → `g.pending` 再入账 |
| D11 | 空/重复 InterruptID 运行期点名 | `suspend` 的两条守卫（空 ID 点名节点；重复 ID 点名两节点） |
| D12 | 预算跨恢复累计 | `Suspension.steps` → `newScheduler(ctx, seq, steps)` |
| D13 | sink 失败与挂起双 Is；Exit 档挂起照常冲刷 | `suspensionSinkErr` Join（Sync）+ `finishCommits` 挂起分支 Join（Async/Exit）+ Exit 落齐不受 runErr 影响 |
| D14 | 非中断节点零影响（反向锚） | 全部新逻辑挂在中断载体/挂起态分支上，普通路径字节未动（既有测试零改动全绿） |
| D15 | `InterruptResponse` 公共取值器；普通 Run 恒查不到 | `ctx` 仅在 `Resume` 注入 responsesKey；D15 的负半边由 a 节点在普通 Run 内采样钉住 |
