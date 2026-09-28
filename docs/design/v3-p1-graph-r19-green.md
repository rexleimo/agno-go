# 切片 19 GREEN（票面 §4 B2：并行扇出 + Join 汇聚屏障）

工作项 `work-p1-graph-r19-join-barrier` / activation `48130ef2-0ff7-4e14-87dc-b826a8acb970` / 阶段 green。
契约：`docs/design/v3-test-scope-p1-graph-slice19.json`（D1–D7）。
本文件记录 GREEN 阶段；随后的 REFACTOR 合并了一处重复聚合实现，最终字节与完整变异矩阵见
`docs/design/v3-p1-graph-r19-refactor.md` §1（本文件 §2 里的两份哈希是 GREEN 时的字节，已被取代）。
本片绑定场景命令逐条为（本文件里每一条 receipt 都是这一条命令、`receipt -- go test …` 直接取，无 `sh -c` 包装）：

```bash
go test ./pkg/hno/graph -run TestP1R19_ -count=1 -race -v
```

## 1. GREEN 判据与实测

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| D1–D7 全绿（契约绑定命令，逐字） | `go test ./pkg/hno/graph -run TestP1R19_ -count=1 -race -v` | 0 | `receipt:03c4c832-faed-4e90-bdc7-bfd6788d979d` |
| 前一步 RED（同一命令，实现之前） | 同上 | 1 | `receipt:ebd73748-a3ba-4e30-99e1-73ae95452fbb` |

RED→GREEN 的差别只有实现字节：`pkg/hno/graph/graph.go` 与 `pkg/hno/graph/scheduler.go`（见 §2），
外加 §4 披露的一处测试自身缺陷修复。

`-v` 逐行（`receipt:03c4c832…` 面）：7 个 Test 全部 `--- PASS`，含 D5 的两个子用例
（`合法汇聚图放行` / `对照：端点不存在的汇聚声明仍被拒且点名那个端点`）与 D6 的两个子用例
（`单前驱` / `空前驱`），末行 `ok github.com/rexleimo/agno-go/pkg/hno/graph 2.413s`。

### 1.1 防「碰巧绿」：重复与全量

这些不是 receipt 命令（只为排除时序侥幸），但都真实执行过：

| 观察 | 命令 | 结果 |
|---|---|---|
| 屏障在重复调度下稳定 | `go test ./pkg/hno/graph -run TestP1R19_ -count=30 -race -timeout 300s` | `ok 1.968s` exit 0 |
| 既有 8 份测试与新面同跑不互相拆台 | `go test ./pkg/hno/graph/... -count=5 -race -timeout 400s` | `ok 6.835s` exit 0 |
| 邻近包不红（契约判据 7） | `go test ./pkg/hno/agent/... ./pkg/hno/runner/... ./internal/session/contract/... -count=1` | 三包全 `ok`，exit 0 |

## 2. 实现差异（有边界）

只动两份非测试文件；本片净增 **+115 行**（结构判据 B13 的 729 → 844，仍 ≤ 1500）。
基线 blob 未入库（未 `hash-object -w`），故给不出逐字节 diff，下面是变更点清单与最终字节。

### 2.1 `pkg/hno/graph/graph.go`（514 → 563 行，`a8e46a3a…` → `f113895a16e5276e77ba0dc26222c49647e02b18`）

1. `edgeJoin` 常量注释从「汇聚屏障，尚未实现路由」改为真实的激活语义（§3.4 规则 3）。
2. `routable()`：从三条 `||` 改为 `switch` 白名单，`edgeJoin` 入列，`default: return false` 保留
   fail-closed 的默认拒绝一侧（票面 :161-168 的「新增种类默认被拒，放行必须是有意识的一次修改」）。
   写成 `switch` 而不是继续加 `||`，是为了让「放行」这件事在 diff 上是一个必须表态的分支，
   不是一个尾部条件。
3. `AddJoin` 的文档改为真实语义：全部前驱完成后激活一次、输入是 `map[string]any`（键为前驱名、
   值为该前驱本次输出）、凑不齐的屏障不把「已完成的那些」当结论、前驱 < 2 被 `Validate` 拒绝。
   这是契约 writeBackOwed 第 2 项要求的注释收口（旧注释自己承诺「路由落地后随之改为真实语义」）。
4. `Validate()` 新增一步 `rejectUnderfedJoinTargets()`，位置在逐条边的端点/谓词/可路由检查之后、
   `rejectUnreachableNodes()` 之前，并把 `Validate` 的次序注释同步为三步。次序是测试钉住的：
   - 端点检查必须在前，否则 `graph_test.go:423-424` 的 `join-to-missing` / `join-from-missing`
     两行会收到前驱数不足的文案，D5 的反向对照（点名 `"ghost"`）也会失去担保。
   - 可达性检查在后，是因为「前驱数为 0」的 `AddJoin(nil, "j")` 在公共面不留任何声明，
     只能由 `j` 无入边这一形状表达（见 §5 观察上限）。
   新文案：`graph: join target "x" must declare at least 2 distinct predecessors, got 1`。
   前驱按名字去重后计数；多个目标不足时按字典序点第一个（与 `rejectUnreachableNodes` 同规矩）。

### 2.2 `pkg/hno/graph/scheduler.go`（215 → 281 行，`0ce6c3b3…` → `de005da450c2f70e26e50b8801e4708a96e150c8`）

1. 新增包级函数 `joinWaits(edges)`：把汇聚声明收成「目标 → 去重前驱集合」与它的反向索引
   （前驱 → 目标列表），一次全边扫描算完，双向都去重。
2. `Run` 在 `capturePlan()` 之后调用它一次，把结果装进 `scheduler`。与 `stepLimit`、
   `maxConcurrency` 同一条不变量：进入 Run 时取一次值，运行中改图不能给已开始的这次 Run
   增删屏障。
3. `scheduler` 新增三个字段 `waits` / `feeds` / `joinCollected`，全部只由唯一消费者读写，
   因此**没有引入任何锁**：`grep -o 'sync\.' $(find pkg/hno/graph -name '*.go' -not -name '*_test.go')`
   在最终字节上计数 **0**（契约判据 6，票面 §3.3 零锁模型不破）。
4. `complete(item)` 在原有 `successors(...)` 之后追加 `joinReady(...)` 的结果，两路都只 append 到
   `pending`，消费者独占写的形状不变。
5. `joinReady(name, out)`：把该前驱的输出按名字记进聚合 map；只有收集到的键集合覆盖了目标声明的
   全部前驱时，才以那个 map 为输入激活目标。「凑齐」判据用键集合覆盖而不是计数器归零——计数器要
   把重复声明与前驱重跑都算对，而集合本身就是公共面看到的那个形状（D2 断言的 map）。
6. `successors` 的注释补一段：`edgeJoin` 在这里没有分支**不是**漏判，激活时刻归屏障；
   两侧各收一半种类，新增种类时两侧都要表态。

### 2.3 导出面

`go doc -all ./pkg/hno/graph | grep -cE '^(func|type|var|const) '` 在最终字节上仍为 **24**，
本片没有新增导出符号（屏障的形状完全经由已有的 `AddJoin` + `Node.Run(in any)` 表达）。

## 3. 每行由哪段实现担保

| 行 | 断言的公共事实 | 实现落点 |
|---|---|---|
| D1 | join 恰好进入一次；`Output()=="joined"`；`Completed()` 里 join 一次 | `joinReady` 的覆盖判据 + 消费者独占 `pending` |
| D2 | join 的输入是 `map[string]any`，按前驱名键控且只含两个前驱键 | `collected[name] = out` 与 `activation{in: collected}` |
| D3 | a、b 都先于 join 完成 | 覆盖判据在 `complete` 内、`running--` 之后才可能激活 |
| D4 | join 的输出继续作为普通前驱往下游流并成为 `Output()` | `complete` 先记 `result.values`/`output`，`successors` 对 join 的出边照常路由 |
| D5 | 合法汇聚图 `Validate()==nil`；端点不存在的汇聚声明仍被点名拒绝且 `Run` 拒绝 | `routable()` 放行 `edgeJoin`，端点检查仍在最前 |
| D6 | 前驱 < 2 的汇聚目标被独立、可定位地拒绝（不再是「本种类不路由」那句） | 新 `rejectUnderfedJoinTargets()` |
| D7 | 前驱本次没跑完 ⇒ 走步数安全阀，不静默、不挂死 | 屏障永不凑齐时 `pending` 由其他激活维持，`dispatch` 的既有预算判据收口 |

## 4. GREEN 阶段发现并修掉的测试自身缺陷（不是弱化断言）

D1 的信号清点原来写成 `for range joinEntered`，而 `joinEntered` 从不关闭 ⇒ Run 返回后该 `range`
永久阻塞。RED 阶段这条不可达（`Run` 先返回「not routable」错误，测试在 :74 就 `t.Fatalf` 退出），
所以直到实现变绿才暴露为整包挂死（一次 `go test` 卡住 5 分钟以上、CPU 0.2s）。

处置：改为**有界非阻塞清点**（`select` + `default` 跳出），断言语义逐字不变（`entered != 1` 仍是
「恰好一次」）。这不触碰契约的任何期望值，也不是把期望改成当前实际输出。
测试文件哈希因此从 RED 阶段的 `8bf4b2f9296eb5781e7f6243c234a1baada0c92f` 变为
`8b4b7234ce92df2f936a1544cf24545d65ce0a9b`（386 行）。其余 7 份既有测试文件逐字节未动：
`bc4ab8e7…` `0ed607db…` `3ea06d27…` `0285ab35…` `6934a4f2…` `cbc8b692…` `10bce61b…`，
与契约 `baselineBytes` 一致（契约判据 4 满足）。

唯一被授权的既有测试改动照 `allowedTestSeam.retires` 执行：`graph_test.go` 的
`TestP1G_UnsupportedEdgeDeclarationsAreRejected` 表里 `join-only` 一格换成
`join-with-one-predecessor`（`wantPhrases` 改为 `{"graph: join target", "at least 2", "\"merge\""}`），
反向对照子用例「只使用已实现边类型的图仍然合法」逐字保留，整块 Test 未删除
（forbiddenShortcuts 第 4 条）。`graph_test.go` 1348 → 1352 行，`425ea030…` → `624f19624c6f4fa3c04ee6619e722b03de83336c`。

## 5. 一次生效归属抽查（完整变异矩阵在 REFACTOR 阶段）

GREEN 阶段先跑一条最能代表票面 :77 反例的形态：把 `joinReady` 的凑齐判据改成 `if false`，
即「每个前驱完成都激活一次屏障」（朴素实现）。`-v` 实测（非 receipt 运行）：

- `TestP1R19_JoinBarrierRunsExactlyOnce` 判红：`joinEntered … 2 次, want 恰好 1 次`（:94）。
- `TestP1R19_JoinActivatesOnlyAfterBothPredecessorsFinished` 判红：`只清点到自己已完成的 1 个前驱, want 2`（:208）。
- `TestP1R19_JoinInputIsKeyedByPredecessorName`、`TestP1R19_JoinOutputFlowsToDownstreamAndProjection`
  各在 **10.00s** 判红：第二次激活把容量为 1 的信号通道写满 ⇒ 生产者阻塞 ⇒ `running` 归不了零 ⇒
  测试自有的有界阀 `p1r19Valve` 触发。**这两行的牙齿目前落在「挂死被有界阀抓住」上**，
  REFACTOR 阶段要补一条不依赖通道容量的形态，把「输入形状错了」与「执行停下来了」分开归因。
- D5、D6、D7 在该变异下全绿（它们判的是构建期与未收敛路径，屏障放宽不触及）。
- 恢复：`cp /tmp/scheduler.r19green.bak` 后 `git hash-object --no-filters` 复核仍为
  `de005da450c2f70e26e50b8801e4708a96e150c8`，再跑绑定命令 `ok 3.770s` exit 0。

## 6. 观察上限与未裁决项（登记，不追加断言去堵）

1. **`AddJoin(nil, "j")` 不留声明**：空前驱在边表上没有任何事实可查，D6 的 `空前驱` 子用例由
   「`j` 无入边 ⇒ 不可达」这条既有检查担保（实测文案点名 `"j"` 且不含 `not routable`，性质符合断言）。
   「空前驱」与「节点注册了但没有任何入边」在公共面同形，与前序切片登记的「注册名为空串」同类。
2. **R19-Q1（已由预测转为实测）**：声明过的汇聚前驱本次没执行、且图上再无其他待派激活时，
   `Run` 交出 `err==nil` 而 `Output()==nil` 的 Result。最终字节上的真实运行（在仓库外 `/tmp` 起的
   独立 module，经 `replace` 引本仓 `pkg/hno/graph`，只调公共 API）：
   `err=<nil> resNil=false output=<nil> completed=[a entry] Value(join)=<nil>(ok=false)`。
   票面 D7 钉的是「图还在往前走 ⇒ 安全阀」这一形状，没覆盖这一形状，本片按契约**不新增断言**，
   登记为需要新的 `rex-test-design` 裁决的收口问题（它同样是票面 :161-168 fail-closed 精神的一个缺口）。
3. **母约 §3.4 的 map-reduce 示例与票面 :65 冲突**（实测发现，非推断）：
   `docs/design/v3-platform.md:341` 写的是 `AddJoin([]string{"summarize"}, "reduce")` —— 单个前驱名、
   靠 `Send` 运行时扇出多次来聚合。按 :65 的「Join 前驱数 < 2 构建期拒绝」，这条示例现在会被本片
   的 `rejectUnderfedJoinTargets` 拒绝。构建期唯一可数的是「声明的去重前驱节点数」，所以两者不可能
   同时成立。需要一次显式裁决（改示例、或给 `Send` 那一片引入「同名前驱的多次激活也算汇聚对象」的
   另一种屏障），登记为 **R19-Q2**；本片未动母约示例。
4. **重复激活的屏障**：前驱因环重跑时，聚合 map 的键数不再增长 ⇒ 目标会被再次激活。
   票面没有任何一行要求「一张图上每个屏障至多fire一次」，且无条件环已被 `rejectUnconditionalCycles`
   挡在构建期，条件环由安全阀兜住（不静默）。本片刻意不加 `fired` 守卫：加了就没有契约行担保它的必要性。
5. **含 `edgeJoin` 的环**：`alwaysTakenAdjacency` 只收无条件边与「无条件兜底边」，汇聚边按可断开处理，
   因此 `a →(join) j →(join) a` 这类声明不被构建期环检查拒绝，落到步数安全阀。这是既有判据的自然延伸，
   本片未改其语义。

## 7. 本阶段未闭环

- `golangci-lint` / `make lint` 仍未运行（与前序切片同样的欠项）。
- 完整变异矩阵（每条验收行至少一次被证伪 + 等价变异登记）在 REFACTOR 阶段交付，脚本要落在仓库内
  可复现的位置（见 S17-STD-5 / S18-STD-6 的教训）。
