# 切片 28 GREEN（母约 §9 G7 第 2 片：会话事件侧车 + 派生视图 + 跨进程恢复）

工作项 `work-p5-session-sidecar` / 契约 `docs/design/v3-test-scope-p1-graph-slice28.json`（D1–D10）。
RED 观察与取消事件披露见 `docs/design/v3-red-observation-p5s28.md`。本片绑定场景命令逐字：

```bash
go test ./pkg/hno/session/sidecar/... ./internal/hitlbridge/... -run TestP5S28_ -count=1 -race -v
```

## 1. GREEN 判据与实测

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| D1–D10 全绿（契约绑定命令，逐字） | 同上 | 0 | `receipt:2a7761b3-85f6-4b04-a58f-a91323b0374a` |
| 接手时中间态（6 过/2 挂） | 同上 | 1 | `receipt:531c79d9-7cad-452a-a6f5-24fa3cfdaa34` |
| 邻近面（agent/runner/session-contract/session 全家） | `go test ./pkg/hno/agent/... ./pkg/hno/runner/... ./internal/session/contract/... ./pkg/hno/session/... -count=1` | 0 | `receipt:66e68495-c5a5-4cf5-aef9-8d008fe4fd98` |
| 防碰巧绿：绑定 `-count=30 -race` | ok | 0 | `receipt:b09c9acb-253f-4897-8657-11c504f3b02a` |
| 防碰巧绿：sidecar+bridge+graph 整包 `-count=5 -race` | ok | 0 | `receipt:3c595bef-2bfa-4e89-98f5-ced59d46d37d` |

## 2. 实现差异（有边界）

| 文件 | 状态 | 内容 |
|---|---|---|
| `pkg/hno/session/sidecar/sidecar.go` | 新增（246 行） | Sidecar 接口 + MemorySidecar + Event/SuspendedRun/InterruptRecord/WaitingNodeRecord + 三派生视图 + Snapshot/RestoreSnapshot（模拟重启路径） |
| `internal/hitlbridge/bridge.go` | 新增（134 行） | Capture/Install/ResumeSaved + fail-open(事件)/fail-closed(挂起记录) 分流 + 记录生命周期（成功即清除、安装不覆盖活挂起） |
| `pkg/hno/graph/restore.go` | 新增（44 行，契约 sourceSeam 的条件性入口） | Suspension.ResumeAccount（挂起时 Seq/Steps 账目读侧）+ RestorePending（把侧车记录重建的 Suspension 安装到全新重建的图实例） |
| `pkg/hno/session/sidecar/p5g7_sidecar_test.go` + `internal/hitlbridge/p5g7_wiring_test.go` | 新增（191/407 行） | D1–D10 测试；字节恒等行用真实 `pkg/hno/session.Session` + MemoryStorage 夹具 |

修复记录：接手时 `MemorySidecar.PendingInterrupts` 是半成品存根（聚合收集后 `return nil, nil`），
补上等待项展平 6 行——两个失败测试同源于此。

## 3. 门禁（契约判据 3–6/8，全部在最终字节上）

| 门禁 | 结果 | receipt |
|---|---|---|
| `go build ./...`；`go vet` 两新包；`gofmt -l` 三处 | 全部干净 | （非 receipt 记录） |
| 邻近面（含 session 全家与契约测试） | 全 `ok` exit 0 | `receipt:66e68495-c5a5-4cf5-aef9-8d008fe4fd98` |
| 结构判据：sidecar 导出 **18**、bridge 导出 **3**（恰等契约 counters） | 达标 | `go doc -all … grep -c` |
| 新包非测试 LOC | **380**（契约 377 + 条件性入口另计的口径内；restore.go 另 44 行） | `wc -l` |
| 8 个 marshal 边界锚文件 | **逐字节全同**（D10/OPT-1 不变量成立） | 逐文件 `git hash-object --no-filters` 对账契约 anchors |
| `sync.` 计数 | 仅 MemorySidecar 自有 `sync.RWMutex`（契约允许项），新缝无新增锁语义 | `grep` |
| 防碰巧绿 | 见 §1 表两行 | — |

## 4. 每行由哪段实现担保

| 行 | 断言的公共事实 | 实现落点 | 测试 |
|---|---|---|---|
| D1 | 侧车写入后 Session 对外 JSON marshal 逐字节不变；对照锚：Storage.Update 会改字节 | 侧车旁路容器（永不路由过 Storage.Update） | ByteIdentity |
| D2 | 事件单调 Seq、挂起记录 CRUD、半截记录拒收、快照→新实例水合 | AppendEvent/Save/Get/Clear + Snapshot/RestoreSnapshot | StoreRoundTripAndViews、SnapshotRestart、InvalidRecordRejected |
| D3 | 三派生视图（Events/EventsByRun/PendingInterrupts 跨 run 聚合） | 三个视图方法 | StoreRoundTripAndViews |
| D4 | 跨进程恢复 Rerun 档端到端（中断→Capture→快照水合→图重建→ResumeSaved） | bridge Capture/ResumeSaved + restore.go RestorePending | CrossProcessResumeRoundTrip |
| D5 | Handoff 档跨重启：节点不偷跑、响应成输出 | RestorePending 保留 Waiting 的 Mode + 恢复调度 | HandoffAcrossRestart |
| D6 | 切片 27 校验语义跨重启保持（坏响应 → ErrInvalidResponse 且挂起保留） | Resume 路径原样复用 | CrossProcessResumeRoundTrip 子用例 |
| D7 | 引擎账目（Seq/Steps）跨重启逐值回装 | ResumeAccount/RestorePending 原值装回 | CrossProcessResumeRoundTrip 子用例 |
| D8 | 幂等 + 成功后清除（PendingInterrupts 随之清空） | ResumeSaved 成功路径 ClearSuspendedRun | ResuspendChainLifecycle |
| D9 | 事件 fail-open / 挂起记录 fail-closed 分流 | bridge 双语义 + failing 双层夹具 | FailureSemantics |
| D10 | 8 锚逐字节 + 允许包契约测试绿 | 侧车旁路设计 | ByteIdentity + 邻近 receipt |

## 5. 本阶段未闭环

- 变异矩阵结果与契约逐条对账见 `docs/design/v3-p5-s28-refactor.md`。
- `golangci-lint` 欠项沿用；切片 2–28 未提交沿用（S18-SPEC-2）。
