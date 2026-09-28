# 切片 19 REFACTOR（票面 §4 B2：并行扇出 + Join 汇聚屏障）

工作项 `work-p1-graph-r19-join-barrier` / activation `48130ef2-0ff7-4e14-87dc-b826a8acb970` / 阶段 refactor。
GREEN 记录见 `docs/design/v3-p1-graph-r19-green.md`；契约 `docs/design/v3-test-scope-p1-graph-slice19.json`。

绑定场景命令（本文件的 receipt 全部是这一条、逐字、`receipt -- go test …` 直接取）：

```bash
go test ./pkg/hno/graph -run TestP1R19_ -count=1 -race -v
```

| 用途 | exit | receipt |
|---|---|---|
| REFACTOR 之后的最终字节全绿 | 0 | `receipt:8a55bd2f-41e3-4c6d-bd26-654fd539b359` |
| GREEN 阶段实现后（重构前字节） | 0 | `receipt:03c4c832-faed-4e90-bdc7-bfd6788d979d` |

命令 B（下表用到，不是 receipt 命令）：`go test ./pkg/hno/graph -run TestP1G_ -count=1 -race -v`。

## 1. REFACTOR 做了什么（且只做了什么）

GREEN 的实现里有一处真实重复：**「哪些前驱算同一个屏障」这条规则被数了两遍** ——
`graph.go` 的 `rejectUnderfedJoinTargets` 自己扫一遍边表建 `目标 → 去重前驱集合`，
`scheduler.go` 的 `joinWaits` 又扫一遍建同样的集合加反向索引。两处各自实现时，
`AddJoin([]string{"a","a"}, "j")` 这类声明完全可以得到相反的结论（一处去重、一处不去重），
而两个结论都是静默的。

收口：合并为唯一的 `joinBarriers(edges) (waits, feeds)`（放在 `graph.go`，因为它是对已声明边的纯函数），
构建期的前驱数判定与运行期的屏障凑齐都读它；`Run` 侧只保留「进 Run 时算一次」这条不变量的注释。
没有引入新导出符号、新锁、新文件；没有改任何断言。

| 度量 | GREEN 字节 | REFACTOR 最终字节 | 判据 |
|---|---|---|---|
| `graph.go` | `f113895a16e5276e77ba0dc26222c49647e02b18` | `18002942851225ac6c27a0a40b4e24086a6c144f` | — |
| `scheduler.go` | `de005da450c2f70e26e50b8801e4708a96e150c8` | `c878a6c74cad89bbdfc773c21c58952d3944630d` | — |
| 非测试 LOC（`graph.go`+`scheduler.go`） | 844 | **834** | ≤ 1500（票面 :94，B13）✓ |
| `sync.` 计数（非测试文件） | 0 | **0** | 必须为 0（B12，零锁模型）✓ |
| `go doc -all` 导出符号 | 24 | **24** | 不放宽（契约判据 6）✓ |

最终字节上的票面 §7 门禁，逐条实测（不是推断）：`go build ./...` exit 0、
`go test ./pkg/hno/graph/... -count=1 -race` exit 0、`go vet ./pkg/hno/graph/...` exit 0、
`gofmt -l pkg/hno/graph scripts` 无输出。邻近回归（契约判据 7）
`go test ./pkg/hno/agent/... ./pkg/hno/runner/... ./internal/session/contract/... -count=1`
三包全 `ok`，exit 0。

## 2. 变异矩阵（13 条，执行体 `scripts/mutation/p1r19-join-barrier.mjs`）

执行体这次**落在仓库里**（前两轮审查 S17-STD-5 / S18-STD-6 记下的是同一件事：脚本在 `/tmp` 时，
「这行断言有牙齿」这个结论别人无法复现）。它做四件事：锚必须唯一命中（否则判为脚本失效而不是判红）、
每条变异跑两命令、`finally` 之前恢复原文、恢复后按 git blob 哈希自检。

- 命令 A（契约绑定）：`go test ./pkg/hno/graph -run TestP1R19_ -count=1 -race -v`
- 命令 B（承载被退役的 R1 格与端点次序）：`go test ./pkg/hno/graph -run TestP1G_ -count=1 -race -v`

判红列取 `-v` 输出里的 `--- FAIL:` Test 名；`exit` 单独一列（构建失败同样 exit 1，只看退出码会误记杀红。
本次 13 条里 `BUILD_FAILED` 计数为 0）。`restored` 全为 true，脚本末行 `finalCheck identical=true`。

| id | 写坏形态 | exitA | exitB | 命令 A 判红的 Test | 命令 B 判红的 Test |
|---|---|---|---|---|---|
| m1 | 凑齐判据整段失效（`if len(collected) < len(waits[target])` → `if false`）：每个前驱完成就激活一次 | 1 | 0 | D1、D2、D3、D4 四个 Test 全名（见下注） | —— |
| m2 | 激活时交出「最后一个前驱的输出」而不是聚合 map | 1 | 0 | **只有 D2** | —— |
| m3 | 聚合输入按汇聚目标名键控（`collected[target] = out`） | 1 | 0 | D1、D2、D3、D4 | —— |
| m4 | 前驱数检查读空表（等于该检查整段不存在） | 1 | 1 | D6 | R1 拒绝表（退役后的新格） |
| m5 | 阈值放宽为「至少 1 个前驱」 | 1 | 1 | D6 | R1 拒绝表 |
| m6 | 票面 :77 反例的另一半：在 `successors` 里把 Join 当无条件边直接激活（屏障逻辑仍在） | 1 | 0 | D1、D2、D3、D4 | —— |
| m7 | 屏障算完了但从不交给调度器（`joinReady` 结果被丢弃） | 1 | 0 | D1、D2、D3、D4 | —— |
| m8 | 汇聚目标自己的出边被忽略（join 的输出不再往下游走） | 1 | 0 | **只有 D4** | —— |
| m9 | 次序变异：前驱数检查挪到逐条边的端点检查**之前** | **0** | 1 | —— | `TestP1G_ValidateRejectsDanglingEndpointForEveryEdgeKind` |
| m10 | 把 Join 收回「不可路由」白名单（等于本片的路由没落地） | 1 | 1 | D1–D7 全部 7 个 Test | R1 拒绝表 |
| m11 | 安全阀错误不再包装哨兵（`%w` → `%v`） | 1 | 0 | **只有 D7** | —— |
| e1 | 多个不足前驱的目标同时存在时不再按字典序点名 | 0 | 0 | —— 无（观察上限） | —— 无 |
| e2 | 前驱名不去重、按声明条数算屏障 | 0 | 0 | —— 无（观察上限） | —— 无 |
| e3 | 反向索引按逆序累加，放弃声明顺序 | 0 | 0 | —— 无（观察上限） | —— 无 |

D1–D7 的 Test 全名对应：D1 `TestP1R19_JoinBarrierRunsExactlyOnce`、
D2 `…JoinInputIsKeyedByPredecessorName`、D3 `…JoinActivatesOnlyAfterBothPredecessorsFinished`、
D4 `…JoinOutputFlowsToDownstreamAndProjection`、D5 `…LegalJoinGraphValidatesNilWhileIllegalStillRejected`、
D6 `…JoinWithFewerThanTwoPredecessorsIsRejectedLocatably`、D7 `…UnexecutedPredecessorEndsAtStepLimitNotSilence`。

### 2.1 生效归属（契约 completionCriteria 第 2 条：每行至少一次被证伪）

- D1 ← {m1, m3, m6, m7, m10}
- D2 ← {m1, **m2（solo）**, m3, m6, m7, m10}
- D3 ← {m1, m3, m6, m7, m10}
- D4 ← {m1, m3, m6, m7, **m8（solo）**, m10}
- D5 ← {m10}
- D6 ← {m4, m5, m10}
- D7 ← {m10, **m11（在本片两条命令内 solo）**}
- 退役后的 R1 格（`join-with-one-predecessor`）← {m4, m5, m10}

GREEN 文档 §5 记下过一个欠项：m1 之下 D2/D4 的判红是**通过测试自有的 10s 有界阀**拿到的
（第二次激活把容量为 1 的信号通道写满 ⇒ 生产者阻塞 ⇒ `running` 归不了零），
也就是「执行停下来了」而不是「输入形状错了」。本次由 m2（输入不是聚合 map）与 m8
（汇聚目标的出边被吞）各自 solo 打红补上，两者都在 10s 阀之外判红。

## 3. 三处诚实更正 / 观察上限

1. **m3 不是「键名错位」的 solo 杀手**：把聚合键写成目标名以后，a、b 两次完成会写进同一个键，
   `len(collected)` 永远是 1 ⇒ 屏障永不凑齐。所以 m3 同时打红 D1/D3/D4，它的判红不能单独归因给
   「键控错了」这条性质；D2 的键控牙齿由 m2 单独担保。
2. **D5 的反向对照不担保「前驱数检查在端点检查之后」这条次序**：m9 把两步互换后，
   命令 A 全绿（D5 用的 `AddJoin(["a","ghost"])` 有两个去重前驱，前驱数检查根本不触发，
   `"ghost"` 仍由端点检查点名）。真正担保这条次序的是既有 `graph_test.go:407` 那组表
   （命令 B 判红）。这与 GREEN 文档 §2.1 第 4 点的说法一致，但现在有实测支撑而不是设计推断。
3. **m11 的杀红不唯一**：整包跑一遍时 `%w → %v` 还会打红
   `TestP1R13_StepLimitStopsBreakableCycle`、`TestP1R13_LimitRefusesActivationsAtDispatch`、
   `TestP1R13_StepLimitDefaultsWhenUnset`、`TestP1R13_EachRunGetsItsOwnBudget`、
   `TestP1R16_CapDoesNotConsumeStepBudget`、`TestP1R17_UncancelledBusinessErrorStillSurfacesVerbatim`
   （实测 7 行 FAIL，含 D7）。哨兵包装因此是跨切片冗余担保的；D7 在「本片的两条命令」内是 solo。
4. **e1/e2/e3 是观察上限，不是漏网的缺陷形态**（与切片 17 把 σ、ξ 移入上限的处置一致，不追加断言去堵）：
   没有任何一行会同时声明两个「前驱不足」的目标（e1）；没有任何一行用重复名字声明前驱（e2）；
   屏障前进顺序在公共面上只能通过时序观察，而票面 :40、:75 明确禁止把时序当证据（e3）。
   对应的未担保性质分别是：多目标时文案点名的稳定性、前驱名的去重语义、`feeds` 的声明顺序。
5. **两种「数不出来」的汇聚声明在公共面上不留事实，最终字节上各自实测过一句**
   （独立 module 在仓库外 `/tmp` 经 `replace` 引本包，只调公共 API）：
   - `AddJoin(nil, "join")`（D6 的 `空前驱` 子用例的形状）收到的是
     `graph: unreachable node "join": not reachable from entry "entry" through any declared edge`
     —— 即它由**可达性**检查担保，而不是由本片新增的前驱数检查担保。空前驱在边表上一条声明都不留，
     与「节点注册了但没有入边」同形；D6 的断言（点名 `"join"`、不含 `not routable`）照样成立，
     但归因要说清。
   - `AddJoin([]string{"a","a"}, "join")` 收到的是
     `graph: join target "join" must declare at least 2 distinct predecessors, got 1`
     —— 去重语义是真实生效的（否则会是 `got 2` 并放行），但 e2 显示没有任何一行钉它。
     登记为未担保性质：若哪天改成按声明条数计数，本片全绿。


## 4. 测试差异审查（test-diff-review）

本片测试面 = 新文件 `pkg/hno/graph/p1r19_join_barrier_test.go`（386 行，`8b4b7234…`）
+ 被授权的 `graph_test.go` 一格（`425ea030…` → `624f1962…`，1348 → 1352 行）+ 其余 7 份零改动。
逐条核对，全部在最终字节上实测：

1. **只用公共入口**：观察面是 `graph.New/WithStepLimit/AddNode/NodeFunc/AddEdge/AddConditional/
   AddJoin/SetEntry/SetOutput/Validate/Run` + `Result.Output/Value/Completed` + `errors.Is` + 错误文本。
   `grep -nE 'scheduler|pending|running|plan\.|queue|goroutine|reflect\.|unsafe'` 在新文件里只有
   1 处命中，且是 :18 的注释（声明这些面被票面 :40 禁止），**调用为 0**。
2. **无 sleep、无时序前提**：`grep -nE 'time\.Sleep|t\.Skip'` → 0 命中；`time.` 的 3 处命中里
   只有 :43 是调用（`time.After(p1r19Valve)`，10s 停摆上界阀），另两处是 :21/:24 的声明与常量。
   屏障的先后前提由测试自有握手表达（D3 的「返回前投信号 + join 非阻塞清点」），不是等时长。
3. **没有把期望改成当前实际输出**：`Output()=="joined"`、`len(byName)!=2`、`got!=any(2)`、
   `errors.Is(err, graph.ErrStepLimitExceeded)` 都是先于实现写下的契约期望（RED 记录
   `docs/design/v3-red-observation-p1r19.md`，实现之前它们逐条判红）。
4. **每行可独立失败**：D5/D6 用 `t.Run` 子用例且各自断言（不是末尾一次整体断言），
   m4/m5 只打红 D6、m9 只打红命令 B、m2 只打红 D2 —— 独立失败不是推断，是上表的实测列。
5. **没有放宽期望**：契约 forbiddenShortcuts 第 2 条禁的「恰好一次 → 至少一次」未发生
   （D1 判的是 `entered != 1`）；第 3 条禁的「D2 只断非 nil 输入」未发生（键名与键数都钉）。
6. **GREEN 阶段唯一一次测试自身修改**是 D1 的 `for range` 挂死修复（改有界非阻塞清点），
   断言语义逐字不变；这不是把期望改成当前输出，而是将一处「RED 阶段不可达」的测试基础设施缺陷
   修好（ rex-tdd RED 第 3 条的同类纪律反过来用：不合法的挂死不能留作 GREEN）。

## 5. 已知欠项（不藏）

- `golangci-lint` / `make lint` 仍未运行（前序切片同样欠着）。
- R19-Q1（声明过的前驱没跑且再无待派激活 ⇒ `err==nil` 且 `Output()==nil`）本片**不新增断言**，
  已在 GREEN 文档 §6.2 用真实运行登记；需要新的 `rex-test-design` 裁决。
- R19-Q2（母约 `v3-platform.md:341` 的 map-reduce 示例 `AddJoin(["summarize"],"reduce")`
  与票面 :65「前驱数 < 2 拒绝」冲突）需要显式裁决，本切片未动母约示例。
- 契约 writeBackOwed 的票面回写（B2 行精确化 + 新增 §15 + 状态文档 §2 的 B2 行勾销）
  按前序切片的做法在审查之后执行 —— **已执行**：票面 §4 表 B2 行与新增 §15、母约 §3.4 规则 3 与
  §3.5 第 6 项、状态文档 §1/§2/§4/§5。
- 写回阶段另外只改了 `AddJoin` 的**文档注释**（审查 S19-SPEC-5：「未收敛由安全阀结束」只在图仍在
  产生激活时成立）。因此本文件 §1 记的最终字节 `18002942851225ac6c27a0a40b4e24086a6c144f` 是写回前态，
  写回后为 `graph.go 4cb363b729cddf7541df8d1b1900c8240dff93f2`（`scheduler.go` 等其余三份不变）；
  注释改动不动行为，但写回字节上重跑并全部实测通过：绑定命令
  `receipt:73a2c954-adb0-4b83-8c3c-e67849d23536`（exit 0）、`go test ./pkg/hno/graph/... -count=1 -race`
  exit 0、`go build ./...` / `go vet ./pkg/hno/graph/...` / `gofmt -l pkg/hno/graph scripts` 全清、
  `node scripts/mutation/p1r19-join-barrier.mjs --check` 报 13 条锚点仍唯一命中。
  最终字节表与可重放计数见票面 §15.6。
