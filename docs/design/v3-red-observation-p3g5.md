# 切片 26 RED 观察（母约 §6 G5：动态扇出 Send + AddJoinSend）

工作项 `work-p3-graph-g5-send` / 契约 `docs/design/v3-test-scope-p1-graph-slice26.json`（D1–D9）。
测试文件：`pkg/hno/graph/p3g5_send_test.go`（`package graph_test`，只用导出 API；计数与输入记录
均为夹具自有状态，契约 forbiddenObservations 逐条遵守：不观察调度器内部、不 `time.Sleep` 定序、
不断言 map 迭代序——聚合断的是 slice 的派发序）。

## 1. redProtocol 两段制的第一段

按契约 redProtocol：第一段只落 API 面——`send.go` 的 `Send`/`Sender`/`senderFunc`/`SenderFunc`/
`AddJoinSend`/`joinSendTables`（声明侧整体）、`graph.go` 的 `edgeJoinSend` 枚举 + `routable()`
白名单项 + `label()` 文案 + 可达性贡献（source 视为可达并从它继续传播）、`scheduler.go` 的
状态字段（queueItem 的 `sends/via/idx`、activation 的 `via/idx`、scheduler 的 7 个 send 屏障字段）。
**不属于第一段**：调度器对字段的全部消费——`Run` 不调 `joinSendTables`、`consume` 不调
`absorbSends`、`tryNode` 不做 Sender 断言。绑定命令可编译（零编译失败），行为行按今天缺掉
派发/屏障的真实形状判红。

本片绑定场景命令逐字（所有 receipt 都是这一条命令、直接取，无 `sh -c` 包装）：

```bash
go test ./pkg/hno/graph -run TestP3G5_ -count=1 -race -v
```

## 2. RED 实测（receipt:259ee969-74ae-4340-a37f-ae45bca76154，exit 1）

**判红 5 行**（D1/D3/D4/D7/D9，全部落在「派发与屏障缺位」的公共面形状上）：

| 行 | 失败文案（摘，`p3g5_send_test.go` 行号） |
|---|---|
| D1 | :61 `sum 执行次数 = 0, want 3（3 个 Send 各激活一次、各收独有输入）`；:64 `reduce 执行次数 = 0, want 1（屏障恰激活一次）` |
| D3 | :154 `sum 执行次数 = 0, want 2（join-send 路不吞并 Send 派发）`；:157 `reduce 执行次数 = 0, want 1` |
| D4 | :188 `Run err = <nil>, want 包 ErrStepLimitExceeded（扇出激活照常计步）`——没有派发就没有新激活，图在 1 步内「收敛」，安全阀根本轮不到触发 |
| D7 | :264 `Run err = nil, want 运行期点名未注册目标（fail-closed，不静默丢）`——sends 被 Run 视图丢弃，未注册目标无从暴露 |
| D9 | :354 `target 激活波数 = 0, want 2（每波各自凑齐、逐波激活）` |

**天然绿 4 行**（按契约与承接指令的预期，如实登记）：

- **D5**（可达性双向）：可达性贡献属于第一段的 API 面（契约 sourceSeam 把它划在 graph.go 一侧），
  withJoin `Validate()==nil` 随第一段落地面即绿；withoutJoin 的 unreachable 拒绝是切片 1 既有行为。
  本行的牙齿在 GREEN 字节上由变异 m3 证明（贡献摘掉 → withJoin 判红）。
- **D2**（零派发）：第一段上「没有任何派发发生」使「target 不激活 + err==nil + Output()==nil」
  **平凡成立**（fan 的 Run 视图丢弃空 sends 后图直接收敛）——本行在第一段是真空绿，不是实现绿；
  其牙齿在 GREEN 字节上由变异 m5 证明（零派发改判激活 → 判红）。这与切片 24 的 D1「API 面落地
  即转绿」同族：公共面存在性行的牙齿在变异侧。
- **D6**（AddJoin 判据原样保留）：反向对照，两段都绿。
- **D8**（非 Sender 零影响）：反向锚，两段都绿；另一半牙齿是既有全部测试零改动全过（门禁面）。

第一段字节上绑定命令 exit 1 且全部失败落在行为断言上（零编译失败、零环境失败）。

## 3. 承接

第二段（`scheduler.go` 接线：`Run` 从快照调 `joinSendTables` 建三张表、`consume` 在 `record`
之后调 `absorbSends`、`runNode/tryNode` 增 `[]Send` 返回值与 Sender 类型断言、新增
`absorbSends`/`joinSendReady`）与 GREEN 见 `docs/design/v3-p3-graph-g5-green.md`；变异矩阵与
契约对账见 `v3-p3-graph-g5-refactor.md`。两段之间测试文件零改动
（`p3g5_send_test.go` 两段同字节，`git hash-object` = `c003a537…`）。
