# 切片 26 GREEN（母约 §6 G5：动态扇出 Send + AddJoinSend）

工作项 `work-p3-graph-g5-send` / 契约 `docs/design/v3-test-scope-p1-graph-slice26.json`（D1–D9）。
RED 观察见 `docs/design/v3-red-observation-p3g5.md`；变异矩阵与契约对账见
`docs/design/v3-p3-graph-g5-refactor.md`。本片绑定场景命令逐字：

```bash
go test ./pkg/hno/graph -run TestP3G5_ -count=1 -race -v
```

## 1. GREEN 判据与实测

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| D1–D9 全绿（契约绑定命令，逐字） | `go test ./pkg/hno/graph -run TestP3G5_ -count=1 -race -v` | 0 | `receipt:dfab55ac-9d7c-43c2-b61a-7d1c00f0f143` |
| 最终字节收尾复跑（变异矩阵恢复并复核哈希之后） | 同上 | 0 | `receipt:b5f24ece-15ca-4d62-91a3-da2ca8e18675` |
| 前一步 RED（redProtocol 第一段，仅 API 面） | 同上 | 1 | `receipt:259ee969-74ae-4340-a37f-ae45bca76154` |

`-v` 逐行：9 个 Test 全部 `--- PASS`，末行 `ok github.com/rexleimo/agno-go/pkg/hno/graph`，
`-race` 无 DATA RACE。RED→GREEN 之间只有实现字节（`scheduler.go` 的屏障接线），测试文件零改动。

## 2. 实现差异（redProtocol 第二段：接线）

基线（切片 24 收口态，契约 baselineBytes）→ 最终：

| 文件 | 基线 | 最终 | 变更 |
|---|---|---|---|
| `pkg/hno/graph/send.go` | 无 | `dcd04661…` | 新增（87 行）：`Send`/`Sender`（`SendRun` 可选扩展面）+ `senderFunc`/`SenderFunc` 适配器 + `AddJoinSend` 声明 + `joinSendTables`（构建期收三张表，对偶 `joinBarriers`） |
| `pkg/hno/graph/graph.go` | `98b8bdad…` | `7d7bb8d8…` | `edgeJoinSend` 枚举 + `routable()` 白名单项（fail-closed 默认拒绝一侧保留）+ `label()` 文案 + 可达性贡献（BFS 前把声明 source 播种为可达，共 10 行） |
| `pkg/hno/graph/scheduler.go` | `4641de52…` | `fcd552c5…` | queueItem 加 `sends/via/idx`、activation 加 `via/idx`、scheduler 加 7 个 send 屏障字段（全消费者独占）；`Run` 进门调 `joinSendTables` 建表；`consume` 在 `record` 之后调 `absorbSends`；`runNode/tryNode` 增 `[]Send` 返回值、`tryNode` 做 Sender 类型断言；新增 `absorbSends`/`joinSendReady` |
| `pkg/hno/graph/p3g5_send_test.go` | 无 | `c003a537…` | 本片新测试，D1–D9（契约 allowedTestSeam.add），两段之间零改动 |

实现要点（契约 designConsequence 五条钉死语义的落点；形态承自契约 M2 预演副本 /tmp/g5exp，
本片在其上逐字节复核并修一处 gofmt 文档注释空格——见 refactor 文档 §1）：

1. **Sender 是可选扩展面，不改 `Node.Run` 签名**（forbiddenShortcuts 第 1 条）：`SendRun(ctx,in)(any,[]Send,error)`
   独立方法名，`senderFunc` 同时实现两个视角（`Run` 是丢弃 sends 的常规视图，引擎对 Sender 节点
   从不走它）。调度器在 `tryNode` 对每次激活做类型断言：成功走 `SendRun`，失败走 `Node.Run`
   ——普通节点零改变（D8 + m1）。
2. **`AddJoinSend(source, target)` 构建期只查端点存在 + 贡献可达性**：不经 §3.5 第 6 项、不与
   `AddJoin` 判据互通（D6 + 既有 TestP1R19 全绿）；`routable()` 白名单显式加入 `edgeJoinSend`
   （fail-closed 的「新增种类默认拒绝」一侧保留）。可达性上，声明边本身照常进邻接表，且
   **source 被播种为可达**——Send 的目标名是运行期值，构建期无法反驳「会有 Sender 指名它」
   （与条件边不计谓词的同一乐观先例），否则 map-reduce 形状整体被判不可达（D5 + m3）。
3. **运行期屏障按「source 名下 pending Send 计数归零」判凑齐**：记账键 = Send 的地址名
   （`sendPending[X]` 数在途的、以 X 为地址的 Send 激活）——「source 名下」即派发账本里记在
   source 名下的那些 Send；每次派发按声明序记 `idx` 槽位，每个激活完成回填输出并递减，归零时
   `joinSendReady` 以聚合输入（`map[string][]any`，键 = source 名，内层 slice 按派发序——
   S19-STD-1 的别名形状在此结构性不存在，激活时移交新建聚合）激活 target 一次（D1/D3）。
   多波扇出：计数由 0 转正时复位「本波已激活」守卫，逐波独立凑齐（D9 + m6）。
4. **零派发 → target 不激活**：计数从未转正就没有归零事件，Run 交 err==nil 而 Output()==nil
   ——与 R19-Q1 同族的既有形状，本片只登记不裁决（D2 + m5；联动义务：R19-Q1 若日后裁为
   fail-closed，D2 同改）。
5. **Send 指向未注册节点 → 运行期点名错误**：`absorbSends` 先整体校验全部 sends、任何一个未注册
   即整批拒绝（不做部分派发），错误 `graph: node %q sent to unregistered node %q`，res=nil
   ——构建期判不了运行时才出现的目标名（D7 + m4）。

与 durability 的交点按契约 observabilityLimit 第 4 条：join-send 的每次激活照常计一步、
其完成照常产检查点条目（`absorbSends` 在 `record` 之后），本片零专属条目种类。

## 3. 门禁（契约判据 4/5/6/8，全部在最终字节上）

| 门禁 | 结果 | receipt |
|---|---|---|
| `go build ./...` | exit 0 | （非 receipt 记录） |
| `go test ./pkg/hno/graph/... -count=1` | `ok` exit 0 | `receipt:4a0865d4-8ff6-479f-af77-0c1803a2d0cb` |
| `go test -race ./pkg/hno/graph/... -count=1` | `ok` exit 0，无 DATA RACE | `receipt:3c8f5315-683a-4adf-843d-728743e70d4e` |
| `go vet ./pkg/hno/graph/...` | exit 0 | （非 receipt 记录） |
| `gofmt -l pkg/hno/graph` | 无输出 | （非 receipt 记录） |
| 邻近回归（agent/runner/session-contract）`-count=1` | 三包全 `ok` exit 0 | `receipt:79f630bb-2a64-4cc6-a8d0-fba26ecaa854` |
| 防碰巧绿：绑定命令 `-count=30 -race` | `ok` exit 0 | `receipt:c85e8a42-5cfc-4214-b393-19bc9bfc4341` |
| 防碰巧绿：整包 `-count=5 -race` | `ok` exit 0（一次过，无 R17 复发） | `receipt:e487bbfd-d9d9-4d2c-972e-3d6cc78443c3` |
| `sync.` 计数（graph/scheduler/send 三文件） | **0** | `grep -o … wc -l` |
| 非测试 LOC | **1435**（graph 636 + scheduler 460 + policy 138 + durability 114 + send 87；判据 ≤1500，恰为契约 expectedFinalLoc） | `wc -l` |
| 导出符号 | **45**（41 → 45：+类型 `Send`/`Sender` + `func SenderFunc` + 方法 `AddJoinSend`） | `go doc -all … grep -cE '^(func\|type\|var\|const) '` |

既有 12 份测试文件与 `policy.go`/`durability.go` 逐字节复核（`git hash-object --no-filters`）：
`policy.go 6d71fa47…`、`durability.go da57a17e…`（= 契约 baselineBytes，零改动）、
`graph_test.go 624f1962…`、`p1r13 bc4ab8e7…`、`p1r14 0ed607db…`、`p1r15 3ea06d27…`、
`p1r15b 0285ab35…`、`p1r16 6934a4f2…`、`p1r17 cbc8b692…`、`p1r18 10bce61b…`、
`p1r19 8b4b7234…`、`p1r21 a8a936ed…`、`p2g3 ec390cc7…`、`p3g6 bdeb8ef9…` ——
与切片 24 收口字节全同，零改动（契约 completionCriteria 第 3 条；票面「13 份」按实数 12 份
登记，见 refactor 文档 §3 判据 3 的对账说明）。

## 4. 每行由哪段实现担保

| 行 | 断言的公共事实 | 实现落点 |
|---|---|---|
| D1 | 3 个 Send 各激活一次、各收独有输入；reduce 恰一次收全 3 份（派发序） | `tryNode` 的 Sender 断言 + `absorbSends` 方向二（派发记账）+ 方向一（回填/递减）+ `joinSendReady`（归零激活，slice 按派发序）；变异 m1/m2 杀红 |
| D2 | 零派发：target 不激活、err==nil、res!=nil 而 Output()==nil | 记账只在「有 Send 派发」时发生，无归零事件即无激活；R19-Q1 既有收敛形状；变异 m5 杀红 |
| D3 | 普通出边与 join-send 声明共存互不吞并 | `successors()` 与 send 屏障两条路各自独立 append pending（`complete` 与 `absorbSends` 分置）；m1/m2 杀红 |
| D4 | 无界扇出被步数安全阀拦住（errors.Is(ErrStepLimitExceeded)、res=nil） | Send 激活与屏障激活都经 `pending` → `dispatch` 计步（切片 13 的判据原样吃新激活）；m1/m5 杀红 |
| D5 | withJoin Validate()==nil；withoutJoin 点名 unreachable | `rejectUnreachableNodes` 的 edgeJoinSend 播种 + 声明边照常入邻接表；变异 m3 杀红 |
| D6 | 单前驱 AddJoin 仍被 §3.5 第 6 项同文案拒绝 | `joinBarriers`/`rejectUnderfedJoinTargets` 零改动（edgeJoinSend 不进 join 表）+ 既有 TestP1R19 钉 |
| D7 | 未注册目标运行期点名（graph: 前缀 + ghost）、res=nil、不做部分派发 | `absorbSends` 方向二的整体校验先于任何入队；变异 m4（静默丢弃形态）杀红 |
| D8 | 非 Sender 节点行为不变（次数/输出/Completed）+ 既有测试零改动全过 | `tryNode` 断言失败即走 `Node.Run`、无任何记账；既有 12 份测试文件哈希未动 |
| D9 | 多波逐波凑齐、输入只含当波（waves==[1,2]、长度 1/2） | 计数 0→1 时复位 `sendFired`、激活后聚合器置 nil（下一波从零积累）；变异 m6（守卫不复位）/m5 杀红 |
