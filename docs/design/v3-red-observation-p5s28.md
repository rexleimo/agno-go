# 切片 28 RED 观察（母约 §9 G7 第 2 片：会话事件侧车 + 派生视图 + 跨进程恢复）

工作项 `work-p5-session-sidecar` / 契约 `docs/design/v3-test-scope-p1-graph-slice28.json`（D1–D10）。
测试文件：`pkg/hno/session/sidecar/p5g7_sidecar_test.go` + `internal/hitlbridge/p5g7_wiring_test.go`
（`TestP5S28_*`，外部包，只用导出 API）。

## 1. redProtocol 与一次如实披露

本片 redProtocol 两段制。**披露**：实现由子代理执行，在 GREEN 阶段中途被系统取消——
第一段（API 面，D2–D9 行为红、D1/D10 绿）的 RED receipt 未及落盘。取消时工作树留有已落地的
四个实现/测试文件与 6 过 2 挂的中间态。主代理接手后按诚实口径补证：

- **中间态回执 `receipt:531c79d9-7cad-452a-a6f5-24fa3cfdaa34`（exit 1）**：接手时刻的真实字节上，
  8 个 Test 中 6 过 2 挂——`StoreRoundTripAndViews`（PendingInterrupts 视图返回空，want [i-1]）与
  `CrossProcessResumeRoundTrip`（重启后 pending 视图为空）。这两条行为红就是「派生视图接线未完成」
  的直接观察，替代缺失的阶段 1 RED 作为 D3 行的红证据；其余行的红证据由契约 M2 的副本实测
  （21 观察面）与中间态共同覆盖。
- 修复：`MemorySidecar.PendingInterrupts` 聚合函数是半成品存根（收集+排序后 `return nil, nil`），
  补上等待项展平（6 行）。修复后全绿。

## 2. 绑定命令与实测

本片绑定场景命令逐字（所有 receipt 都是这一条命令直接取，无 `sh -c` 包装）：

```bash
go test ./pkg/hno/session/sidecar/... ./internal/hitlbridge/... -run TestP5S28_ -count=1 -race -v
```

| 阶段 | exit | receipt |
|---|---|---|
| 中间态（取消时字节：6 过/2 挂） | 1 | `receipt:531c79d9-7cad-452a-a6f5-24fa3cfdaa34` |
| 修复后全绿（最终字节） | 0 | `receipt:2a7761b3-85f6-4b04-a58f-a91323b0374a` |

`-v` 逐行：8 个 Test 全部 `--- PASS`（ByteIdentity / StoreRoundTripAndViews /
InvalidRecordRejected / SnapshotRestart / CrossProcessResumeRoundTrip / HandoffAcrossRestart /
FailureSemantics / ResuspendChainLifecycle，含子用例覆盖 D1–D10）。

## 3. 承接

GREEN 门禁、变异矩阵与契约逐条对账见 `docs/design/v3-p5-s28-green.md` 与
`docs/design/v3-p5-s28-refactor.md`。
