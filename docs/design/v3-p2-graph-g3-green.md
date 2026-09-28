# 切片 22 GREEN（母约 §4 G3：节点策略四合一）

工作项 `work-p2-graph-g3-node-policies` / 契约 `docs/design/v3-test-scope-p1-graph-slice22.json`（D1–D14）。
RED 观察见 `docs/design/v3-red-observation-p2g3.md`；变异矩阵与契约对账见
`docs/design/v3-p2-graph-g3-refactor.md`。本片绑定场景命令逐字：

```bash
go test ./pkg/hno/graph -run TestP2G3_ -count=1 -race -v
```

## 1. GREEN 判据与实测

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| D1–D14 全绿（契约绑定命令，逐字） | `go test ./pkg/hno/graph -run TestP2G3_ -count=1 -race -v` | 0 | `receipt:b5afcc0f-985a-409e-b32e-8cbe6bc1b194` |
| 前一步 RED（redProtocol 第一段，语义未接线） | 同上 | 1 | `receipt:d8269809-2eb9-406b-8662-cc18c72f112b` |

`-v` 逐行：14 个 Test 全部 `--- PASS`，末行 `ok github.com/rexleimo/agno-go/pkg/hno/graph`。
RED→GREEN 之间只有实现字节（scheduler.go 接线），测试文件零改动。

## 2. 实现差异（redProtocol 第二段：接线）

基线（切片 21 收口态）→ 最终：

| 文件 | 基线 | 最终 | 变更 |
|---|---|---|---|
| `pkg/hno/graph/policy.go` | 无 | `6d71fa47…` | 新增：四个 Config + NodeOption + With* + 取值钳制 helper（约 145 行） |
| `pkg/hno/graph/graph.go` | `b7790b9f…` | `72974bf6…` | `Graph` 加 policies 表、`AddNode` 变参挂载（对既有调用源兼容）、`New` 初始化 |
| `pkg/hno/graph/scheduler.go` | `c878a6c7…` | `873f4a82…` | plan 携带 policies；`runNode` 包络（缓存查询 → 重试循环内带期限尝试 → 成功才写缓存）；`queueItem` 捎带 in/attempt；消费者两处发射 trace |
| `pkg/hno/graph/p2g3_policies_test.go` | 无 | `ec390cc7…` | 本片新测试，D1–D14（契约 allowedTestSeam.add） |

实现要点（契约 designConsequence 的落点）：

1. **重试/期限在生产者 goroutine 内包络**：重试的是「同一次激活的执行」，不产生新激活、
   不消耗步数预算（D5）；退避等待随时让位给调用方取消（`select` ctx.Done）。
2. **PerAttempt=false 是整个节点的期限预算**（循环外挂一次，含全部重试尝试）；PerAttempt=true
   每次尝试新建期限（D8 钉的是递增的 deadline 值，不是耗时）。
3. **缓存查询在生产者侧、成功才写缓存**（失败入库 = 把失败当结论交给下一次 Run，D11 钉死）。
4. **trace 由消费者串行发射**（`complete` 与错误分支两处），零锁模型未破：`sync.` 计数仍 0。
5. **零值即语义**：未声明的策略走各自 guard 钳制（MaxAttempts<1→1 次、Timeout<=0→不限时、
   未声明缓存/追踪→不生效），本片零新增 Validate 行。

## 3. 门禁（契约判据 4/5/6/8，全部在最终字节上）

| 门禁 | 结果 | receipt |
|---|---|---|
| `go build ./...` | exit 0 | （非 receipt 记录） |
| `go test ./pkg/hno/graph/... -count=1` | `ok` exit 0 | `receipt:b45262e0-cfcb-400c-b66d-d002f246c0ba` |
| `go test -race ./pkg/hno/graph/... -count=1` | `ok` exit 0，无 DATA RACE | `receipt:5e54d8c6-4590-4655-8ebf-c471c368f9a5` |
| `go vet ./pkg/hno/graph/...` | exit 0 | （非 receipt 记录） |
| `gofmt -l pkg/hno/graph` | 无输出 | （非 receipt 记录） |
| 邻近回归（agent/runner/session-contract） | 三包全 `ok` exit 0 | `receipt:c21e6091-757c-4ad0-b294-971905172858` |
| 防碰巧绿：绑定命令 `-count=30 -race` | `ok` exit 0 | `receipt:4d26a57d-bd7f-4fb4-808a-56b1bfa53969` |
| 防碰巧绿：整包 `-count=5 -race` | `ok` exit 0 | `receipt:5bcf938b-a8eb-4d93-ae50-0472e9c2fa40` |
| `sync.` 计数（graph/scheduler/policy 三文件） | **0** | `grep -o … wc -l` |
| 非测试 LOC | **1085**（判据 ≤1500） | `find … xargs wc -l` |
| 导出符号 | **35**（判据恰好 35：+11 见契约 baselineBytes.counters） | `go doc -all … grep -c` |

既有 10 份测试文件逐字节复核（`git hash-object --no-filters`）：`graph_test.go 624f1962…`、
`p1r13 bc4ab8e7…`、`p1r14 0ed607db…`、`p1r15 3ea06d27…`、`p1r15b 0285ab35…`、`p1r16 6934a4f2…`、
`p1r17 cbc8b692…`、`p1r18 10bce61b…`、`p1r19 8b4b7234…`、`p1r21 a8a936ed…` —— 与契约 baselineBytes 一致。

## 4. 每行由哪段实现担保

| 行 | 断言的公共事实 | 实现落点 |
|---|---|---|
| D1/D3 | 重试至成功 / 耗尽交末错，次数恰为 MaxAttempts | runNode 重试循环 + attempts() 钳制 |
| D2/D4 | 零值不重试；ShouldRetry 门槛与 nil=从不 | should() 的 nil 判 + 循环 break 条件 |
| D5 | 重试不消耗步数预算 | 包络在同一次激活内，dispatch 计步不变 |
| D6/D7/D8 | 期限送达节点 / 未声明不挂 / PerAttempt 逐次新建 | tryNode 的 WithTimeout 与 runNode 的 base 分支 |
| D9/D10/D11 | 缓存命中 / 按键分流 / 失败不入库 | runNode 首尾的 GetAny / SetAny（成功才写） |
| D12/D13/D14 | 事件送达 / 脱敏 / 关闭零事件 | queueItem 捎带 + 消费者两处 emit + emit 内脱敏 |
