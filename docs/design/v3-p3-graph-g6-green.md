# 切片 24 GREEN（母约 §7 G6：Durability 持久化档位）

工作项 `work-p3-graph-g6-durability` / 契约 `docs/design/v3-test-scope-p1-graph-slice24.json`（D1–D9）。
RED 观察见 `docs/design/v3-red-observation-p3g6.md`；变异矩阵与契约对账见
`docs/design/v3-p3-graph-g6-refactor.md`。本片绑定场景命令逐字：

```bash
go test ./pkg/hno/graph -run TestP3G6_ -count=1 -race -v
```

## 1. GREEN 判据与实测

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| D1–D9 全绿（契约绑定命令，逐字） | `go test ./pkg/hno/graph -run TestP3G6_ -count=1 -race -v` | 0 | `receipt:04eb9c8c-f01f-480f-9a73-fa19d8d55ac5` |
| 最终字节收尾复跑（变异矩阵恢复并复核哈希之后） | 同上 | 0 | `receipt:3b49530f-fe84-4b11-ae49-b36f649056a4` |
| 前一步 RED（redProtocol 第一段，调度器不消费） | 同上 | 1 | `receipt:3d22b67c-1e21-44f4-80fe-c7dbf474ad6f` |

`-v` 逐行：9 个 Test（含 D6/D7/D8 的三档/两档子用例）全部 `--- PASS`，末行
`ok github.com/rexleimo/agno-go/pkg/hno/graph`。RED→GREEN 之间只有实现字节
（`durability.go` 补三段方法 + `scheduler.go` 接线），测试文件零改动。

## 2. 实现差异（redProtocol 第二段：接线）

基线（切片 22 收口态）→ 最终：

| 文件 | 基线 | 最终 | 变更 |
|---|---|---|---|
| `pkg/hno/graph/durability.go` | 无 | `da57a17e…` | 新增：`Durability` 三常量 + `Checkpoint` + `Checkpointer` + `WithDurability/WithCheckpointer` + `beginCommits/record/finishCommits`（114 行） |
| `pkg/hno/graph/graph.go` | `72974bf6…` | `98b8bdad…` | config 加 `durability/checkpointer` 两字段（共 2 行，契约 sourceSeam 措辞） |
| `pkg/hno/graph/scheduler.go` | `873f4a82…` | `4641de52…` | scheduler 快照 `durability/checkpointer` 两字段 + 提交侧状态（seq/exitQueue/commits/flushDone）；`Run` 尾部 `beginCommits` → `consume` → `finishCommits`；consume 成功路径 `complete` 之后调 `record` |
| `pkg/hno/graph/p3g6_durability_test.go` | 无 | `bdeb8ef9…` | 本片新测试，D1–D9（契约 allowedTestSeam.add） |

实现要点（契约 designConsequence 六条决策的落点）：

1. **sink 声明在 graph 包内**：`Checkpointer` 接口（`Append(ctx, Checkpoint) error`）+
   `Checkpoint` 三字段最小集（Seq/Node/Output），沿 CacheStore 先例；graph 不依赖
   internal/session，存储适配留给调用方。
2. **Seq 由消费者盖章**：`record` 在提交时机 `s.seq++` 后写进条目，从 1 连续递增，
   次序即本次 Run 的完成处理次序；条目流 append-only，无读回 API。
3. **三档时机全在消费者侧**：Sync 的「下一步开始前」落在 consume 循环里 `complete` 之后、
   下一轮 dispatch 之前；Async 交后台冲刷 goroutine（与引擎只共享 commits 通道，零锁不破），
   `finishCommits` 在 Run 返回前 close + 收敛等待；Exit 只积累 `exitQueue`、退出一次落齐。
4. **失败路径语义**：已完成节点的条目在失败/撞阀退出时照常交付（`finishCommits` 在 runErr
   非 nil 时照样 flush，只跳过 sink 错误的上交）；sink 错误永不静默——Sync 即时使 Run 失败
   （consume `return nil, err`），Exit/Async 在退出收尾上交（`errors.Is` 可归因），且
   runErr 优先、不变异成双重报错。
5. **默认档 = DurabilitySync 由 iota 零值直接给出**，不新增 Validate 行；声明了档位而未挂
   sink = 不持久化、零影响（三段方法各自 nil 早退），不报错。
6. **本片不交付**：恢复/重放（S24-SPEC-1，挂 P5 G7 门槛）、条目到 run 包 CheckpointEvent
   的生产者桥接、落盘格式/编码/TTL。

## 3. 门禁（契约判据 4/5/6/8，全部在最终字节上）

| 门禁 | 结果 | receipt |
|---|---|---|
| `go build ./...` | exit 0 | （非 receipt 记录） |
| `go test ./pkg/hno/graph/... -count=1` | `ok` exit 0 | `receipt:d8b78d99-e739-45ff-ae5a-fdfc9e141f41` |
| `go test -race ./pkg/hno/graph/... -count=1` | `ok` exit 0，无 DATA RACE | `receipt:898ad3f9-d10c-483e-86fc-ec094e1a9722` |
| `go vet ./pkg/hno/graph/...` | exit 0 | （非 receipt 记录） |
| `gofmt -l pkg/hno/graph` | 无输出 | （非 receipt 记录） |
| 邻近回归（agent/runner/session-contract）`-count=1` | 三包全 `ok` exit 0 | `receipt:b52032ac-d0c7-4c8b-9fae-13fc3e2aa874` |
| 防碰巧绿：绑定命令 `-count=30 -race` | `ok` exit 0 | `receipt:dc787347-22a5-4cb1-98e4-1c6c30cd265b` |
| 防碰巧绿：整包 `-count=5 -race` | `ok` exit 0（首跑撞 R17 既有夹具竞态一次，见 refactor 文档 §2.2；复跑 exit 0） | `receipt:21238297-f4bf-4f0e-a14d-1601b8ae7728` |
| `sync.` 计数（graph/scheduler/policy/durability 四文件） | **0** | `grep -o … wc -l` |
| 非测试 LOC | **1219**（policy 138 + graph 625 + durability 114 + scheduler 342；判据 ≤1500） | `find … xargs wc -l` |
| 导出符号 | **41**（判据恰好 41：+3 类型 Durability/Checkpoint/Checkpointer + 2 Option + 1 const 块；三常量在 const 块内缩进不被 ^ 锚 grep 计入） | `go doc -all … grep -cE '^(func|type|var|const) '` |

既有 11 份测试文件与 `policy.go` 逐字节复核（`git hash-object --no-filters`）：`policy.go 6d71fa47…`、
`graph_test.go 624f1962…`、`p1r13 bc4ab8e7…`、`p1r14 0ed607db…`、`p1r15 3ea06d27…`、`p1r15b 0285ab35…`、
`p1r16 6934a4f2…`、`p1r17 cbc8b692…`、`p1r18 10bce61b…`、`p1r19 8b4b7234…`、`p1r21 a8a936ed…`、
`p2g3 ec390cc7…` —— 与契约 baselineBytes 全同，零改动。

## 4. 每行由哪段实现担保

| 行 | 断言的公共事实 | 实现落点 |
|---|---|---|
| D1 | 三常量 iota 序 Sync<Async<Exit、零值即 Sync；四符号存在 | durability.go const 块（零值即默认档）+ 类型/Option 声明；符号存在性另由测试文件整体编译钉住 |
| D2/D3 | 默认档=Sync 的行为钉；Sync 每节点完成后、下一步开始前已落 | consume 循环 `complete` 之后、下一轮 dispatch 之前调 `record`；record 的 Sync 分支即时 `Append`；单链上提交 happen-before 派发后继，采样确定 |
| D4 | Exit 运行中零条目、返回前一次落齐 | record 的 default 分支只积累 `exitQueue`；`finishCommits` 的 Exit 分支返回前逐条落齐 |
| D5 | Async 返回前落齐、不丢、Seq 序不乱 | `beginCommits` 冲刷 goroutine（只共享 commits 通道）+ `finishCommits` 的 close + 收敛等待；Seq 由消费者盖章故与完成处理次序一致 |
| D6 | 条目流与 Result 对账（每完成节点恰一条、Seq 连续、Node 集==Completed()、Output 同值） | complete 成功路径恰调一次 record；`s.seq` 消费者侧连递；`item.out` 与 `result.values` 同源 |
| D7 | sink 失败 fail-closed（Sync 即时、Exit 退出上交、res=nil） | record Sync 分支包装上交（consume `return nil, err`）；`finishCommits` 在 runErr==nil 时上交 sink 错误 |
| D8 | nil sink 零影响（反向锚） | `beginCommits/record/finishCommits` 三处 nil 早退；无全局默认 sink；零新增 Validate 行 |
| D9 | 撞阀保留已提交条目、不给未跑节点补条目 | dispatch 的步数判据只拒绝派发、不触碰提交侧；已完成的 a 在撞阀前已经 record；未跑节点无 complete 即无 record |
