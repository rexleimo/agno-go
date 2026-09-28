# P1（`pkg/hno/graph`）状态审计 —— 切片 19 收口后

记录时间：2026-09-27（首版记于切片 18 收口后；本轮按切片 19 的实测更新 §1/§2/§4/§5）。
审计者：作者本人（未派发独立子代理，用户此前已拒绝对 P1 派发）。
动机：切片 18 的 rex 工作项返回 `completed` 后，按票面 §7 完成判据做一次逐行核对，
发现「B1–B13 全部有测试并通过」这一先前信念**不成立**。本文件用实测输出纠正它。

## 1. 判据 2 与判据 5（四条门禁 + 结构判据）：实测通过

```
go build ./...                                                          exit 0
go vet ./pkg/hno/graph/...                                              exit 0
gofmt -l pkg/hno/graph scripts                                          无输出
go test ./pkg/hno/graph/... -count=1                                    ok  1.972s  exit 0
go test ./pkg/hno/graph/... -count=1 -race                              ok  2.720s  exit 0
go test ./pkg/hno/agent/... ./pkg/hno/runner/... ./internal/session/contract/... -count=1
                                                                        ok 三包    exit 0
go test ./pkg/hno/graph -run TestP1R19_ -count=1 -race -v                exit 0
                                                                        receipt:73a2c954-adb0-4b83-8c3c-e67849d23536
grep -o 'sync\.' pkg/hno/graph/graph.go pkg/hno/graph/scheduler.go | wc -l          0      （B12）
find pkg/hno/graph -name '*.go' -not -name '*_test.go' | xargs wc -l | tail -1  838 total   （B13 ≤ 1500）
go doc -all ./pkg/hno/graph | grep -cE '^(func|type|var|const) '             24       （导出面未放宽）
```

切片 19 之前的数字（`wc -l` 口径）：729 → GREEN 844 → REFACTOR 834 → 写回注释补完 838。

## 2. B 行逐行核对：剩两行，且两行都与既有担保正面对撞（实测，非走查）

首版夹具与输出：`/tmp/p1-audit/main.go`（`go run .`，module `replace` 指向本仓库）。
下表保留**改前实测**，勾销列按切片 19 的最终字节实测填写。

| 票面行 | 首版实测（切片 19 之前） | 现在的结论（切片 19 之后） |
|---|---|---|
| **B2** `entry→{a,b}→join`：join 恰好执行一次、输入含两个前驱键（按前驱名键控）、a/b 均先于 join 完成 | `Validate() = graph: join edge from "a" to "join" is declared but not routable by this engine`；`Run` 原样拒绝、`res=nil` | **已落地**。`TestP1R19_*` 7 个 Test 全 `--- PASS`（`-race`，receipt:73a2c954-adb0-4b83-8c3c-e67849d23536）；判据三要素与牙齿归属见票面 §15.1/§15.2 |
| **B6 第 6 项** Join 前驱数 < 2 → `Validate()` 非 nil | `AddJoin(["a"],"merge")` 交出的是 `not routable` 那句，不是「前驱数」那句 | **已落地**。同一命令下 `TestP1R19_JoinWithFewerThanTwoPredecessorsIsRejectedLocatably`（单前驱/空前驱两子用例）通过，文案点名目标与实收前驱数；被契约授权退役的 R1 格由 `TestP1G_`（命令 B）担保 |
| **B6 第 2 项** 重复节点名 → `Validate()` 非 nil | 同名 `AddNode("a")` 两次后 `Validate() = <nil>`（后者静默覆盖前者） | **已落地（切片 21，票面 §17）**。`AddNode` 记 `dupNames`、`Validate` 末位 `rejectDuplicateNodeNames` 点名拒绝，`Run` 随之拒绝；RED `receipt:5964bbf8…`（D1/D2/D5 判红）→ GREEN `receipt:aaea7171…`（5 Test 全过，`-race`）；既有 10 份测试文件与 `scheduler.go` 逐字节未动；牙齿 m1–m3 杀红、m4 等价登记。副本实验先行（`receipt:330df6ad…`/`receipt:14aa3bd0…`）与 §16.4 的自纠记录保留不改 |
| **B6 第 5 项** 无出口（非输出节点无出边）→ `Validate()` 非 nil | `entry→sink`（sink 无出边、非输出）`Validate() = <nil>` | **经负责人裁决作废（OPT-A，2026-09-27）**，不是「未实现」也不是「已实现」。设计期实测：`/tmp/r19q1/sink` 复核今天确实 `Validate() = <nil>` 且 `Run` 静默成功；票面字面判据加到仓库副本 → 12 个既有顶层 Test 判红（基线 `receipt:7f442439…` exit 0 → `receipt:dd11883e…` exit 1）；最窄非空限定版抓住 `entry→sink`、放行条件终点，仍 10 个判红（`receipt:f594bc89…`）。判红的是 `TestP1G_ConditionalAndDefaultRouting/unconditional-fanout`（`graph_test.go:236` 点名 `"b"`）与 `p1r13:135`（点名 `"x"`）—— 即 §3.4 规则 1 的无条件扇出终点按设计允许输出无人消费。裁决、三条回执与「作废后仍成立的那半句」见票面 §16；导出面保持 24、零代码改动 |

其余行核对为已落地并有测试：B1（`TestP1G_LinearDagRunsChain`）、B3（`TestP1G_ConditionalAndDefaultRouting`）、
B4（`TestP1R13_*` 8 个）、B5（`TestP1G_ValidateRejectsUnconditionalCycle`）、B6 其余项
（悬空边/端点、入口、出口、不可达，`TestP1G_ValidateRejects*`）、B7（`TestP1R15_*`+`TestP1R15B_*`）、
B8（`TestP1R16_*` 8 个）、B9（`TestP1R17_*` 6 个）、B10（`TestP1R14_*` 8 个）、B11（`TestP1R18_*` 7 个）、
B12/B13（§1 的结构命令输出，切片 19 后重测仍为 0 / 838）。

## 3. 为什么「已完成」的信念会出错（根因）

切片序列是**按风险自底向上**排布的：切片 4 把 B2 写进非目标，原文是
「`AddJoin` 的声明在本切片之后仍必须在构建期被拒绝，R1 的 join-only 拒绝行不得删除；
出现任何 Join 路由实现即属超范围」（`v3-test-scope-p1-graph-slice4.json` 的 `outOfScope[0]`）——
它钉的是「本片不许顺带实现」，而不是「B2 作废」。此后每片都只在自己的契约里重申同一句非目标，
而 §11.3 那条 RED 次序表始终躺在票面里没有被逐片勾销。
于是「B 行都在切片 1–16 里闭合了」成了未经核对的推断 —— 票面 §7 第 1 条要的是**逐行背书**，
背书清单从未生成过。本文件就是那份清单。首版按未落地登记的四行里，B2 与 B6 第 6 项已随切片 19 落地
并在 §2 勾销，B6 第 5 项经负责人裁决作废（OPT-A，票面 §16），B6 第 2 项经 §16.4 实测确认为正常
`behavior-delta` 后随切片 21 落地（票面 §17）—— **B1–B13 至此全部逐行背书完毕，P1 闭合。**

## 4. 后续次序（不与已交付字节冲突）

1. ~~**切片 19 = B2**：并行扇出 + Join 汇聚屏障的路由，含 B6 第 6 项。~~ **已交付**
   （rex 工作项 `work-p1-graph-r19-join-barrier` 判 `completed`；判据、牙齿归属与观察上限见票面 §15）。
   当年把顺序排在这里的理由成立：Join 的建边形状定下来之后，B6 第 5 项的 sink 判据才不会误拒
   「只通过 Join 出发的节点」。
2. **切片 20 原计划 = B6 第 2、5 项 → 设计期实测后判定两项都不能按票面字面取红；第 5 项已裁决、第 2 项待裁决。**
   - 第 5 项（非输出汇点）：~~需负责人在 OPT-A/OPT-B/OPT-C 之间裁决~~ **负责人 2026-09-27 选 OPT-A（作废该判据）**。
     实测支撑：票面字面判据加到仓库副本让 12 个既有顶层 Test 判红；最窄的非空限定版（只判无条件边的目标）
     抓住 `entry→sink` 且放行条件终点，仍让 **10 个**判红（副本 receipt `7f442439…`/`dd11883e…`/`f594bc89…`，见 §2 该行）。
     判红的子用例名是 `TestP1G_ConditionalAndDefaultRouting/unconditional-fanout`，被点名的是无条件扇出的终点 `b`/`x`
     —— 即 `graph_test.go:236` 担保的是「无条件扇出的终点在构建期放行」，不只是分支终点。
     写回已完成：票面 §16（裁决记录 + 三条回执 + 作废后仍成立的那半句）、票面 §4 B6 行划掉该项、
     母约 §3.5 第 5 项改写。**零代码改动、导出面仍 24、无新增测试文件。**
     实验、命令与回放步骤见 `v3-test-scope-p1-graph-slice20.json` 的 `measuredBeforeDesign`（M1–M3）。
   - 第 2 项（重复节点名）：~~不需要裁决，待实现（切片 21）~~ **已交付（票面 §17）**。本文件第一版把它写成与
     `graph_test.go:953` 的静默覆盖语义「正面对撞、属票面变更」，那是读测试名得出的推断；
     副本实测推翻它 —— `AddNode` 记 `dupNames` + `Validate` 末位拒绝重复名后，
     `go -C /tmp/sinkexp test ./pkg/hno/graph -count=1` 与 `-race` 均 exit 0、0 个判红
     （`receipt:330df6ad…`/`receipt:14aa3bd0…`），且 `AddNode(` 在本包之外零调用方。
     留作契约内容的是取舍而不是冲突：按名字记账会让「跨 Run 故意重注册同名节点替换实现」在 `Validate` 上不可通过。
     详见票面 §16.4。
3. **P1 闭合条件**：~~B1–B5、B7–B13 已背书，B6 第 5 项按 OPT-A 作废，只剩第 2 项一片正常切片~~
   **已满足 —— B1–B13 全部逐行背书（B6 第 5 项作废、第 2 项随切片 21 落地），P1 关闭。**
   之后才进 P2（Timeout/Retry/Cache/Trace/`Send`）。
   注意 `Send` 那一片要顺带裁决 `R19-Q2`：母约 §6 的 map-reduce 示例按现在的 §3.5 第 6 项会被拒绝。
4b. **并行派发模式启用（2026-09-28，负责人指示）**：负责人明确「速度太慢了，可以分配一些任务并行处理，子代理」——
   此前「P1 不派发子代理」的口径由本条更新。硬约束不变：同一时刻只允许一个代理改仓库代码
   （变异脚本会临时改写仓库字节）；`v3-p1-graph-status.md` 由主代理独占维护。
4c. **P4 协议层（G4 StreamMode）已交付（切片 23，子代理并行实现）**：契约
   `v3-test-scope-p1-graph-slice23.json`（D1–D12）按 redProtocol 两段落地——
   RED `receipt:2e128444…`（4 行行为判红）→ GREEN `receipt:691c84f4…`（14 Test 全过）；
   run 包导出符号恰 77、`sync.`=0、LOC 1723≤2000、既有测试零改动、变异 m1–m6 全杀红；
   `RunStreamMode` 落地（StreamMessages 可用，其余六模式 fail-closed 待生产者片）。
   证据链 `docs/design/v3-red-observation-p4g4.md`、`v3-p4-run-g4-green.md`、`-refactor.md`、
   `-review-verdict.json`（verdict pass）。实现披露的契约自纠（baselineBytes 配对错位）已由
   主代理逐文件实测修正（见该契约 correctionNote）。
4d. **P3 第 1 片（G6 Durability）契约已起草（切片 24，子代理并行）**：
   `v3-test-scope-p1-graph-slice24.json`（D1–D9，M1/M2 副本实验 receipt:cc67e981/483ee91f/9deb910c 等，
   设计决定：Checkpointer 接口入 graph 包、Seq 消费者盖章、三档时序全在消费侧保持零锁；
   resume/replay 按 §7 门槛挂起至 P5/G7，登记 S24-SPEC-1）。待工作项派发后实现。
4e. **R19-Q2 裁决材料已登记（切片 23 同期，子代理并行）**：`docs/design/v3-adjudication-r19-q2-send-join.md`——
   实测证据（§6 示例今天被 §3.5 第 6 项拒绝，probe receipt:5421d654；且即便补足两个前驱名，
   Send 字面形状仍过不了可达性检查）+ OPT-A/B/C 后果 + 推荐 OPT-C（显式 Send 汇聚声明）。
   负责人裁决前不得起草 G5 契约。
4. **P2 第 1 片（G3 策略四合一）已交付（2026-09-28）**：切片 22 按契约 redProtocol 两段落地 ——
   API 面（policy.go + 变参 `AddNode`）先落、D1–D14 按 M1 形状取行为红（10 红 / 4 反向对照绿，
   `receipt:d8269809…`），接线 `runNode` 包络与消费者侧 trace 转绿（`receipt:b5afcc0f…`）。
   门禁与结构判据全过：`sync.`=0、非测试 LOC 1085（≤1500）、导出符号恰 35、既有 10 份测试文件
   零改动（`graph_test.go 624f1962…`、`p1r21 a8a936ed…` 复核全同）、邻近三包不红、count=30/count=5
   防碰巧绿过；变异矩阵 8 条杀红 + e1（失败路径 trace 发射）等价登记。证据链：
   `docs/design/v3-red-observation-p2g3.md`、`v3-p2-graph-g3-green.md`、`-refactor.md`、
   `-review-verdict.json`。`R19-Q1`、`S19-STD-1` 按契约 carriedItems 原样继续携带；
   `R19-Q2` 仍留给 `Send` 那片。P2 下一前沿：G4 StreamMode 或 G5 `Send`（需先裁决 R19-Q2）。
4f. **P3 第 1 片（G6 Durability）已交付（切片 24，子代理并行实现）**：契约
   `v3-test-scope-p1-graph-slice24.json`（D1–D9）按 redProtocol 两段落地——RED `receipt:3d22b67c…`
   → GREEN `receipt:04eb9c8c…`（9/9 PASS）；导出符号恰 41、LOC 1219、`sync.`=0、既有测试零改动；
   变异 m1–m8 全杀红 + e1/e2 等价登记；证据链 `docs/design/v3-red-observation-p3g6.md`、
   `v3-p3-graph-g6-green.md`、`-refactor.md`、`-review-verdict.json`（verdict pass）。
   母约 §7 与交付表 P3 行（G6 半边）已写回；G5 半边仍待 R19-Q2 裁决。
4g. **D1 裁决材料已登记（P5/G7 前置，子代理并行）**：`docs/design/v3-adjudication-d1-session-events.md`——
   实测基线 `receipt:6738b4c9…`，并发现 **contract fixtures 实为 skip-green**（loader 指向
   `/Users/rex/codes/contract-fixtures`，仓库内从未存在，9/9 SKIP——互操作基线比名面弱）；
   四个 marshal 边界盘点（唯一定整结构序列化在 surreal `sessionToPayload`）；OPT-1 探针证明
   侧车存储下对外 JSON 逐字节不变。推荐 OPT-1（v1 内部存储 + 派生视图）。负责人裁决前不起草 G7 契约。
4h. **P7 第 1 片（G9 观测接线）契约已起草（切片 25，子代理并行）**：
   `docs/design/v3-test-scope-p7-g9-observability.json`（13 行 D 映射）；M2 在副本上完整预演两段制
   （阶段 1 恰 9 红/4 绿 → 接线后全绿，receipt:7308e63a/5bd127dc）；零新增导出符号（私有 seam：
   `StateAwaitModel` 处单点 `TurnInvoker` 包络，breaker→span→retry）；M1 顺带量得
   **observability.Retry(MaxAttempts=0) 静默不调用 fn** 的既有隐患（契约 M1 实测）。
   tool 阶段已有 span（反向对照钉住）；run/agent 阶段推迟至 P7 第 2 片（S25-DEFER-1）。
4i. **两项裁决已落（2026-09-28，负责人）**：R19-Q2 = OPT-C（显式 `AddJoinSend(source, target)`，
   构建期不经 §3.5 第 6 项，运行期屏障按「source 的 pending Send 归零」判凑齐）→ G5 解锁；
   D1 = OPT-1（v1 侧车存储 + 派生视图，不改 Session 对外 JSON）→ G7 解锁。
   母约 §6/§9.3/§12 与两份材料文档均已标注；G5/G7 契约起草随即并行派发。
4j. **第三轮并行派发（2026-09-28）**：切片 25 G9 实现（唯一改仓库代理）+ G5 契约起草 +
   G7 契约起草 + 文档/blog 更新计划，四路并行。
4k. **P7 第 1 片（G9 观测接线）已交付（切片 25，子代理并行实现）**：契约
   `v3-test-scope-p7-g9-observability.json`（D1–D13）两段落地——RED `receipt:aa8c6103…`（9 红/4 绿）
   → GREEN `receipt:669c0c6d…`（13/13）；零新增导出符号（私有 seam：runner `StateAwaitModel` 单点
   `invokeTurn` 包络 breaker→span→retry）；变异 m1–m9 杀红 + e1/e2 等价登记（e2 顺带量得 M2 缺陷
   形态的观察上限 S25-SPEC-1）；证据链 `v3-red-observation-p7g9.md`、`v3-p7-g9-green.md`、
   `-refactor.md`、`-review-verdict.json`（pass）。run/agent 阶段 span 留第 2 片（S25-DEFER-1）。
4l. **切片 26（G5 Send+AddJoinSend）与切片 27（G7 HITL 核心）契约已起草（主代理据被取消子代理的
   完整 /tmp 实验写成）**：26 = `v3-test-scope-p1-graph-slice26.json`（D1–D9；R19-Q2=OPT-C 落地形态：
   Sender.SendRun 可选面 + AddJoinSend 声明 + pending-Send 归零屏障；副本 0 判红 + 10 观察面绿）；
   27 = `v3-test-scope-p1-graph-slice27.json`（D1–D15；D1=OPT-1 第 1 片：RequestInterrupt 哨兵 +
   Suspension/ErrSuspended + Resume 三语义 + EntryInterrupt 持久化；副本 0 判红 + 15 观察面绿；
   G7 拆三片登记：引擎核心→会话侧车→事件桥接）。**两片都改 pkg/hno/graph，实现必须串行
   （26 先行、27 重取 baseline）**。契约起草时发生过一次回执 ID 尾段脑补错误（10+4 条），
   逐条对账修正 —— 与票面 §16.5 同源的教训再次实测成立：截断列表不得当完整 ID 用。
4m. **文档/blog 更新计划已登记（子代理并行）**：`docs/design/v3-docs-blog-update-plan.md`——
   website 对 v3 图引擎线零覆盖（grep=0）；P-now 可立即并行执行 5 个新页面（graph-engine/
   node-policies/durability/api×2，全部用已落地导出面），blog 两篇现在可写（零锁图引擎、
   节点策略四合一），第三篇（HITL）等切片 27 落地；en-first + zh mirror，ja/ko 不在范围。
4n. **P3 第 2 片（G5 Send + AddJoinSend）已交付（切片 26，子代理并行实现）—— P3 全清**：
   契约 `v3-test-scope-p1-graph-slice26.json`（D1–D9）两段落地——RED `receipt:259ee969…` →
   GREEN `receipt:dfab55ac…`（9/9）；导出符号恰 45、LOC 1435、`sync.`=0、既有测试零改动；
   变异 m1–m6 杀红 + e1 等价登记；母约 §6 示例改写为可跑通形态、交付表 P3 行标记全清。
   证据链 `v3-red-observation-p3g5.md`、`v3-p3-graph-g5-green.md`、`-refactor.md`、
   `-review-verdict.json`（pass）。实现披露两处如实登记：阶段 1 的 D2 因「无派发即无激活」
   空转绿（牙齿由变异 m5 在 GREEN 字节上补证）；待派 Send 账本的键读法按契约 M2 实测形状钉死。
   **P6（G10 workflow 迁移）的前置 P1–P4 至此全部满足。**
   切片 27 契约 baselineBytes 已按本片收口字节重取（graph.go 7d7bb8d8…、scheduler.go fcd552c5…、
   send.go dcd04661…），串行约束解除。
4o. **website 文档与 blog 已上线（子代理并行，14 个新文件 + 注册）**：guide/graph-engine、
   guide/node-policies、advanced/graph-durability、api/graph、api/run-events 五页（en+zh 镜像）
   + blog 两篇（零锁图引擎、NodeOption variadic 设计，en+zh），blog index 与侧栏已注册，
   `npm run docs:build` 全绿。诚实披露三处：`RetryConfig.Jitter` 已声明未接线（文档标注）；
   计划草案的拓扑示例撞 R19-Q1 未决形状（改为扇出 join 并如实标注）；RunStreamMode 入参为
   string。Send/HITL 文档段按「落地一片写一片」等 26/27 收口后补（26 已落，可补）。
4p. **D2 裁决材料已登记（子代理并行）**：`docs/design/v3-adjudication-d2-store.md`——
   实测关键发现：**Store 与 knowledge 重叠为零**（knowledge 只有 Loader/Chunker 摄取面，
   无任何存储/检索），真正相邻的是 vectordb（可组合但四个表示损失点 + 跨 namespace 同名 Key
   在 Document.ID 上后写覆盖的实测冲突）；母约「破坏性最强」按零消费方实测不成立。
   推荐 **OPT-a1**：进 v3.0 但收窄为最小核（≈1,000–1,300 行，接口+namespace+注入
   vectordb.EmbeddingFunction+内存与单一 Postgres 后端），vectordb 适配挂 v3.1；
   七条边界必须先写进 G8 契约的 explicitNonGoals。待负责人裁决。
4q. **P5 第 1 片（G7 HITL 引擎核心）已交付（切片 27，子代理并行实现）**：契约
   `v3-test-scope-p1-graph-slice27.json`（D1–D15）两段落地——RED `receipt:e428c1fc…`（14 红/1 锚绿）
   → GREEN `receipt:ad6c744c…`（15/15）；导出符号恰 60、LOC 恰 1800（实现者把副本超限的 1860
   压回上限，S27-SPEC-1）、`sync.`=0；变异 m1–m7 杀红 + e1/e2 登记；RequestInterrupt 哨兵、
   Suspension/ErrSuspended、Resume 三语义（校验保留挂起/幂等/Rerun+Handoff）、EntryInterrupt
   持久化、预算跨恢复、重试不吞中断全部落地。证据链 `v3-red-observation-p5g7.md`、
   `v3-p5-graph-g7-green.md`、`-refactor.md`、`-review-verdict.json`（pass）。母约 §9.2 与
   交付表 P5 行（第 1 片）已写回。
4r. **切片 28（G7 第 2 片：会话事件侧车 + 派生视图 + 跨进程恢复）契约已起草（子代理并行）**：
   `v3-test-scope-p1-graph-slice28.json`——侧车落位 `pkg/hno/session/sidecar`（四个 marshal
   边界零触碰，D1 探针 [B2] 纪律成文）+ `internal/hitlbridge`（Capture/Install/ResumeSaved），
   M2 在副本全预演（21 观察面含跨进程恢复双档、fail-open/fail-closed 分野）。
   **依赖登记**：跨进程恢复需要切片 27 的 pending 恢复入口（ResumeAccount/RestorePending 一族），
   该入口不在 27 的交付面内 —— 切片 28 实现时按最小入口补齐（严格串行、重取 baseline）。
4s. **website 的 Send 段与交叉链接已补（子代理并行，切片 26 落地后）**：guide/graph-engine
   （en+zh）新增 Dynamic Fan-out 章节（map-reduce、零派发如实标注、AddJoin vs AddJoinSend
   取舍表）、api/graph 增 Send/Sender/SenderFunc/AddJoinSend 条目、D9 交叉链接修正
   （guide/index 四抽象、workflow 双 API 共存注记、architecture 零锁模型小节）、blog 首篇
   roadmap 行改为「已落地」；docs:build 绿。
4t. **P5 第 2 片（G7 会话侧车 + 跨进程恢复）已交付（切片 28，子代理取消后主代理接手补完）—— P5 全清**：
   契约 `v3-test-scope-p1-graph-slice28.json`（D1–D10）。子代理 GREEN 中途被系统取消（无返回），
   已落盘四个实现/测试文件 + 条件性入口 `graph/restore.go`（ResumeAccount/RestorePending，44 行）；
   主代理接手：中间态回执 `receipt:531c79d9…`（6 过/2 挂，PendingInterrupts 存根是真实红）→
   补 6 行展平 → 全绿 `receipt:2a7761b3…`。8 个 marshal 边界锚逐字节全同（OPT-1 不变量）；
   变异 m1–m10 杀红（m9 精确复现接手时的红形态——红绿红闭环）+ e1/e2 等价；邻近面与
   `-count=30`/整包 `-count=5 -race` 防碰巧绿全过。母约 §9/§10 P5 行标注全清，D1 材料补落地指针。
   **P5（G7 事件化+HITL）关闭：审批场景跨进程恢复、重复 Resume 幂等、契约测试绿三条验收实测达成。**
   过程披露与 salvage 纪律见 `v3-red-observation-p5s28.md` §1、`v3-p5-s28-refactor.md` §4。
4u. **提交授权落地 —— 未提交序列入库（2026-09-28，负责人「你来推进一下」）**：此前 201 个路径（179 未跟踪 +
    22 修改）悬在工作树里，`S18-SPEC-2` 的「无提交授权」已由切片 21 的 `git checkout` 事故升级为实测风险。
    现按包逻辑分七片本地提交（**未 push**）：`0da4013` graph（P1/P2/P3/P5 全部引擎字节 + R13–R21/P2G3/P3G5/P3G6/P5G7
    测试）、`f8e6e04` run（G4 协议层）、`28f2c92` runner（G9 观测缝）、`cf1e247` session 侧车 + hitlbridge、
    `f7a57dd` 变异执行体入库（S19-STD-6 的「未接进 make」仍开着）、`2c7718a` docs/design 证据链、`17f6845` website。
    提交前门禁实测：`go build ./...` exit 0、`go vet ./pkg/hno/... ./internal/...` exit 0、
    `gofmt -l` 对**本次触及的包**零输出（全仓 30 个 gofmt-dirty 文件均为存量、本片一字未动）、
    受影响 12 包 `go test -count=1` 全 `ok`。**全仓 `go test ./... -count=1` 则三轮里一轮判红**，定位到
    `pkg/hno/graph` 的 R17 族在负载下稳定复现（见 §5 末条 S24-STD-2 升级）——故本条绿灯口径限定为
    「build/vet/gofmt + 受影响 12 包」，不含 graph 全包在负载下的绿灯。新字节做了 secret 抽查（API key/AKIA/私钥/密码模式）零命中。
    **`yarn.lock` 故意不提交**：其 11 行改动是 win32-x64 → darwin-arm64 的平台二进制翻转（本机装 docs 依赖所致），
    提交会把 CI 侧的期望平台条目改掉。自此刻起，任何变异/实验脚本对仓库字段的恢复都有了真实 HEAD 可退。
4v. **D2 已裁 + G8 契约（切片 29）已起草（2026-09-28）**：负责人选 **OPT-a1**（进 v3.0，收窄为最小核 +
    单一 Postgres 后端，vectordb 适配层不进 v3.0）。契约 `docs/design/v3-test-scope-p8-g8-store.json`
    （`work-p8-store`，30 键与 slice26/28 键序逐位一致）：D1–D11（草图逐字形状/Embedder 注入 fail-closed/
    错误分类学/(ns,key) 复合身份防 [A4] 后写覆盖/upsert 字节恒等/Search 排名与确定性序/向量为写入期派生列/
    Postgres 八项模式清单 + 禁 import session·pgx/线性打分 v1 形状 + 禁 pgvector/十二锚点字节恒等 + 零接线/
    分工表以导出白名单验收）+ 14 条 explicitNonGoals（承接裁决材料 §3 的七条边界）。
    **回执纪律已复核**：契约引用 30 条唯一 receipt，逐条对 `.rex-harness/receipts/` 存在性校验 **0 缺失**
    （起草者自报曾有一条 UUID 中段写错并已按真实文件修正——与 §4l/§6 同源教训第三次出现，本次被自查拦下）。
    母约 §8 警示行、§2.1 G8 行、§10 P8 行、§12 D2 行均已按裁决写回。**待落裁**：契约的 ADJ-1…ADJ-6
    （包落位与 `internal/session/store` 重名、namespace 语法、`List` 分页、`Value` 文本/二进制、
    Search 是否回 score、批量写入面）——**这六项未裁前 G8 实现不得开工**。
    另两条实测披露：① 相邻四包基线的 48.3% 覆盖含水分（chromadb + openai 共 7 个 skip 假绿，redisdb 整包挂 build tag 根本没编译）；
    ② 本机 `-race` 下 pgx 链不上，故 D8/D9 的真 Postgres 行为只能由 sqlmock 面证明，CI 侧需补真库跑。
4w. **G10（P6）两次派发未跑完 → 收窄为范围摸底材料（第三次派发，在途）**：第一次被系统取消，第二次子代理
    烧到 150 轮上限**未落盘任何文件**，其最后动作是「在 /tmp 副本里加两套内核的逐字段差分测试」——把实现期
    验证手段当成了契约前置，是本轮要纠正的形状。我实测的张力：`pkg/hno/workflow` 153 导出符号 / 3,263 非测试行 /
    5,373 测试行，而母约给 G10 估 ~1,200；workflow 与 graph **今天零耦合**（grep 零命中），workflow 的仓库内消费者
    只有 `cmd/examples/workflow_demo`、`workflow_history`。故第三轮改为交付一份有界材料
    `docs/design/v3-adjudication-p6-g10-scope.md`（体裁照 D1/D2 成功先例），并下**≤45 次工具调用**硬线 +
    明禁差分测试台，把「第一片换哪条执行路径 / 哪些 workflow 能力图侧今天表达不了 / 是否会长出双内核语义」
    三问量清楚再立契约。同期在途（均为 docs-only 并行，不改仓库代码）：R17 负载加固契约、G4 生产者接线契约。
    串行纪律不变（§4b）：同一时刻只允许一个代理改仓库代码，本文件由主代理独占维护。
4x. **交付前沿同步（2026-09-28）**：`docs/design/v3-delivery-ticket.json` 的 `work-p2-node-policies`、
    `work-p3-send-durability`、`work-p5-event-hitl`、`work-p6-workflow-migration` 四条阻塞理由写的是
    「依赖 work-p1-graph-engine 未完成」，随 P1–P5 收口已全部过期，故移入 `frontier.ready`，
    `blocked` 只剩 `work-p9-release`（前置 P6/P7/P8 尚未交付）。`node scripts/validate-delivery-ticket.mjs`
    判 **VALID**（`frontier.ready` 11 / `blocked` 1 / 12 workItems / 七组并行组），
    回执 `receipt:b7ed56ca-5143-49c6-89d0-823fe1d6fc10`（exit 0）。
4y. **G8 六项形状已放行并回填契约（2026-09-28，负责人一次性放行包 A,A,A,A,B,B）**：
    `docs/design/v3-test-scope-p8-g8-store.json` 新增 `adjudicationRuling`（逐条记录裁决与理由），
    D 行由 11 条扩到 **13 条**——D1 形状锚从「恰五方法」改为**七方法**（草图五 + `SearchScored` + `PutMany`，
    登记为对母约 §8 草图的一次**显式扩面**，草图原文保留作裁决出处），新增 D12（`SearchScored` 与 `Search`
    同集合同序只差分数 + 非文本剔除必须经分数通道可见）与 D13（`PutMany` 嵌入调用数与条数解耦、**整批原子**）；
    explicitNonGoals 第 10 条由「不加批量面」改写为「扩面只放行这两处 + 禁把 `PutMany` 写成逐条 Embed 的语法糖」。
    **派生子决定一条并如实登记**：ADJ-6/B 原文把「批量部分失败」列为待裁，负责人按包放行未逐条表决，
    该子决定由主代理按 D5 同族（单条 Put 已承诺「失败不留半截行」）裁为**整批原子**，写在 D13 的
    `independentlyFailable` 里可测；改意的成本是换 D13 一处断言。母约 §8 已同步写回。
    **G8 实现片因此解冻**，但按 §4b 串行纪律仍排在 R17（切片 30）与 G4 生产者（切片 31）之后。
4z. **G10 范围摸底材料已落地（第三次派发，37 次工具调用收在 45 硬预算内）**：
    `docs/design/v3-adjudication-p6-g10-scope.md`。关键实测：workflow 的**唯一线性内核**是
    `executeSteps`（`executor.go:27`，跑在 `run.Loop` 上，全仓仅 `workflow.go:222` 一处调用），
    condition/loop/router/parallel 各自递归；workflow 与 graph 双向零 import；
    基线 `go test ./pkg/hno/workflow/... -count=1 -v` = **131 PASS / 0 FAIL**（`receipt:312de106-ff8e-4006-8918-8b0720d2d824`）、
    两个示例编译（`receipt:d52c41ef-2953-4e31-a569-5bb0686e4203`）、34 项字节锚（`receipt:61e7faf5-4b46-4db5-b791-9d15cb887ae0`）；
    窄探针（`/tmp/g10probe/cmd/linprobe`）证链式图与旧核**输出逐字相同、resume 相同**，适配器净 78 行（入库估 120–180），
    而 β（控制流全换）外推 **1,800–2,900 行**，两值都不是母约 :88 的 ~1,200。材料推荐 **OPT-α**（只换线性链）。
  - **本材料带回一条必须升格的发现（不只是 G10 的事）**：探针在扇出形状上 `-race` **实测 DATA RACE**
    （`receipt:7e8029a8-5ad6-4ce7-8de3-c4f3fb5c10a8` exit 1），根因是把**同一个可变 `*ExecutionContext`**
    交给并发节点，与 §5 挂着未裁的 **`S19-STD-1`**（引擎交出的是调度器继续写入的那个累加器 map 本体）同源。
    另有两处真实行为差：错误串丢 `[UNKNOWN] ` 前缀且报的是最后一个成功步、同名 step ID 被 `Validate` 拒绝
    （切片 21 的重复名拒绝行与 workflow 的 step 命名习惯相冲）。**读法**：OPT-α 只换线性链故不引并发，
    这两条在 α 下不成为缺陷；一旦走 β，`S19-STD-1` 就从「未裁的引擎内部取舍」变成「用户可见的数据竞争」。
    因此 `S19-STD-1` 的裁决时机应提前到 G10 范围裁定同一轮，不排到它后面。

4aa. **G10 范围已裁 OPT-β，写回母约与摸底材料（负责人 2026-09-28）**：选 **OPT-β「控制流全换」**，
    **逆着摸底材料 §3 的「推荐 OPT-α」**。已写回三处：母约 §2.1 G10 行（成本 `~1,200` → 实测外推
    `≈1,800–2,900`，合计行 `~9,500` → `~10,100–11,200`）、母约 §10 P6 行（交付列换成 β 口径，并把三项开工前置
    写进验收列）、`v3-adjudication-p6-g10-scope.md` 新增 §5「落裁登记」。**α 的推荐依据仍然有效，所以它的三条前置
    在 β 下从风险提示升格为阻塞项**：`S19-STD-1`（`ExecutionContext` 引用/值语义，只有负责人能裁，
    且 `[P6]` 的 DATA RACE 已实测 `receipt:7e8029a8-5ad6-4ce7-8de3-c4f3fb5c10a8` exit 1）、
    互斥分支原语落点（workflow 侧互补谓词 vs 给 graph 加 `Branch`/`Router` 导出面）、
    Router 未命中语义（β 采 today 报错，须在适配器复刻 `router.go:63`，不许借道 `AddDefault` 静默兜底）。
    **结论：P6 在 `S19-STD-1` 落裁前不建切片契约**——这条与 §4z 的读法一致，现在成为裁决的一部分而不是建议。

4ab. **失败代理留下的两份契约已校验入库（`receipt` 抽查 + 逐锚复核）**：起草切片 30/31 的两个代理都撞到轮次上限
    （状态为 failed），但两份 JSON 均已完整落盘。主代理逐条独立复核而非采信其自述：
    - **切片 30（R17 负载加固，8 行 D1–D8）全项对上**：`p1r17_cancel_normalization_test.go` = `cbc8b692a84ed5ce23b461a8ed40b9ba3870d386`、
      `scheduler.go` = `dfd6c1ba1c2a9c35bf4ccb613ee986dd80a7c736`、`go doc -all` 导出面 = 64，三个锚逐字命中；
      它引用的 12 个断言行号（`:216/:219/:223/:226`、`:259/:262/:265/:268`、`:419/:436/:439/:443`）逐个落在原文上，
      `p1r17RaceRounds = 100`（`:61`）与「5.8%／35 of 600」的叙述同源；它声称的加固缝 `DurabilitySync` +
      `WithCheckpointer`（`durability.go:21/:60/:64`）**确是既有导出面**，故 D7 的「零新增导出」不是空话；
      11 个红轮的原始文件在 `/tmp/r17exp/` 可数（`grep -l FAIL` 恰 11 份）；17 个引用回执全部存在。
    - **切片 31（G4 StreamTasks 生产者，10 行 D1–D10）事实全对、指针有假**：图目录逐文件哈希的 md5
      = `ae3c22445eb8167f500ad832513046ca`（契约自己纠过的现值）复采逐字相同；三个新事件构造函数
      `NewNodeStartedEvent/NewNodeCompletedEvent/NewTaskErrorEvent` 在 `pkg/hno/run` 之外**确为零调用方**；
      `allowedTestSeam` 承诺的唯一一行改动锚点 `runstreammode_test.go:182` 确为 `run.StreamTasks,`。
      但它引用了 **3 个磁盘上不存在的回执 ID**（`5a6a371f…`／`26609c5f…`／`95a9161e…`）。
      同时段（08:00 段）回执有 63 份之多，故不是被清理，而是**凭印象转写**——讽刺的是该契约 `notes`
      自己写着「所有 ID 现取现贴……写完抽查 `.rex-harness/receipts/` 对应 JSON 存在」。
      处置：`5a6a371f` 的 4 处（含 notes 的 exit≠0 归类短形式）改指同一条
      `-v -run TestP4G4_RunStreamModeUnsupportedModesFailClosed` 的真实回执 `0eede7da-…`（exit 1），
      并补登其 exit 0 复跑 `eb7079a2-…`（两者命令逐字相同，差在候选副本是否已落那一行删除，
      这对读数正是 seam「只有一行」的最小性证明）；另两个无可追溯声明，换成校验时现取的
      `daa972c7-9c65-48c6-81d2-4a869832001a`（图字节锚）与 `9826367c-f909-4682-957f-e24a8db089f8`
      （run/agent/runner 计数基线 77/21/23）。修后复扫：36 个唯一 ID **全部命中**，键数仍 30。

4ac. **切片 30/31 的负载/接线加固片与 G10 之外的一项程序级欠账已登记**：见 §5 新增的 **`S30-EVD-1`**
    （证据指针对账，含 `node scripts/evidence-audit.mjs` 这条命令与 16 个真正无法核验的引用分布）。
    它与 `S19-STD-6`（变异脚本未接进 `make`）是同一类欠账：**纪律有散文、没有可跑的检查**。
    本轮未把它接进 `make`——因为现存 16 条 NOWHERE 引用会让门禁直接 exit 1，先登记、后由负责人定「历史引用是否豁免」。

4ad. **交付票的 P6 条目随 β 改写并回到阻塞态**：`work-p6-workflow-migration` 的 outcome/completionCriteria
    原文写的是 **α 口径**（「现有线性 Steps 工作流内部编译为链式图」「Step 内 Parallel/Loop/Condition/Router 容器节点行为等价」），
    裁决换成 β 之后这段文字会变成实现片的错误靶子，故一并改写：outcome 改为「四类控制流全部进图、六个横切面留在 workflow」，
    判据补三条硬约束（`S19-STD-1` 落裁结果 + `-race` 无竞态、互斥分支原语按落裁实现并带反向判据、Router 未命中复刻 `router.go:63` 报错），
    并把摸底 §4 的 7 项未测清单写成「必须逐条执行并留现取回执」的判据。
    frontier 相应从 ready 移回 **blocked**（原因写明先决项是 `S19-STD-1`，次决项是分支原语落点），
    成员唯一性自证：12 个 workItem ＝ ready 10 + blocked 2，逐一对应。
    校验：`node scripts/validate-delivery-ticket.mjs` → **VALID**（`receipt:41b5586d-c7b3-48ad-86c6-b04b70e9b894` exit 0）。
    过程如实登记：第一版把成本注记写成 work item 上的 `notes` 字段，被 harness schema 拒为
    `delivery work item contains unknown field: notes`（`WORK_ITEM_KEYS` 只有 id/title/outcome/completionCriteria/
    verification/evidenceRefs/dependsOn，见 `~/.rexcil/harness-cli/rex-harness/src/domain/planning-artifact.mjs:16`），
    已把该注记改写成判据、成本数字移到阻塞原因，validator 复跑转绿。

4ae. **切片 30（R17 夹具负载加固）实现收口，`S24-STD-2` 的加固欠项结清**：三段证据分成两份账本
    —— `docs/design/v3-p1-graph-r30-green.md`（串行门禁 + 负载协议 + 同二进制 A/B）与
    `docs/design/v3-p1-graph-r30-refactor.md`（10 条变异矩阵 + 六条 solo 杀手各 5/5 独立执行）。
    改动仍然只在测试文件与 `scripts/`（`allowedModifiedTestFiles: 1` 兑现），生产字节逐位不动。
    加固形把三行负载敏感行的「前提如何建立」换成 **Sync 提交停靠点**
    （`WithCheckpointer`+`WithDurability(DurabilitySync)`，消费者被钉在 `complete` 之后、
    下一轮 `dispatch` 之前那个可复算的位置），26 处 `t.Error*` 判据行经机器 `diff` 证为逐字未动，
    四个常量（`p1r17Grace/Succ/StepBudget/RaceRounds`）取值不变。
    - **实测读数**：`-count=1 -race -v` exit 0（`receipt:ab8987aa-67df-4c2e-bf60-42334a420747`）、
      `-count=30 -race` exit 0 且 180 条 PASS×0 FAIL（`receipt:92813c00-c02f-4f5e-8aa4-1e67efe4e4af`）、
      整包 `-count=5 -race` **连续 3 轮全绿**（`receipt:5efe7f82-0b8e-4500-bfe9-039af1016944` /
      `receipt:a2dac9f4-a3f6-47f9-93ff-85b37e7edf7e` / `receipt:b3fa2e87-8bef-47f6-b42e-6294bf3afc65`）、
      负载协议 **16/16 轮 0 红**（`receipt:f5337f72-cf86-495f-b0d1-ed922e64b50d` 取 10 轮 +
      `receipt:94ad9f73-aa42-435b-9a70-cf3ad2f28fb9` 追加 6 轮，loadavg 爬到 21.38），
      完整矩阵 exit 0（`receipt:dcc7911c-a203-46d4-bbfa-680183a26ff6`），5/5 计数
      （`receipt:f0824b09-cff2-4597-adbe-f3f170734de0`）。50 份原始输出里 **0 条 DATA RACE**。
    - **A/B 对照（契约 #2 要求的那一次，同二进制、同负载曲线）**：原始夹具 **1/6 轮红**、加固形 **0/6**
      （`receipt:79cb40a1-ebbf-4853-91be-78636fc0d19e`）。那一红按契约 D8 分型为**前提红**：
      `p1r17o…:211`「取消之前 Run 已经交出…派发窗口没建立」——安全阀抢在测试取消之前撞上，
      证明的是原夹具**连前提都建不起来**，不是引擎答错。这正是 `S24-STD-2` 那句
      「(a) 守卫读的是过期瞬时值」的直接实证：**读法 (a) 证实**；
      **(b)（引擎在单次 dispatch 内真的不回头咨询 ctx）在本片可构造的位置上不成立**——
      m1/m2 那两条注入让 `HardD1` 稳定红 5/5，说明现行字节答对靠的正是循环头那两条判据的存在，
      而不是靠「取消恰好先到」。
    - **执行体入库**：`scripts/mutation/p1r17-cancel-window.mjs`（矩阵 + `--check` 锚唯一性 +
      逐条 `git hash-object` 自检 + 原始输出落盘）与 `scripts/mutation/p1r17-load-rounds.sh`
      （负载协议）。`S17-STD-5`/`S18-STD-6` 对这两份脚本而言关闭，对 `struct.sh` 仍欠；
      `S19-STD-6`（变异脚本未接进 `make`）**本片不结**，见 §5。
    - **过程如实登记（本片踩到的一条、值得所有判定脚本共用）**：矩阵第一版给十条变异都记了 exit=1，
      包括登记为「等价变异、应当全绿」的 m11 —— 根因在脚本自己：`"test"` 被前置了两遍，
      `go test test ./pkg/hno/graph …` 打出 `FAIL test [setup failed]` 并非零退出，而测试本体照常跑完打印 PASS。
      当时「m11 全绿」与「十条各有牙齿」两句都是**没被验证过的叙述**。修法是三条自检进代码：
      认 `[setup failed]`、非零退出且无判红行判 `NO_ASSERTION_RED`（不计牙齿）、
      命中守卫文案判 `PREMISE_RED`（不计牙齿）。修后 m11 才是真 `exit 0`，其余接手行不变。
      教训与切片 26 的 D2 空转绿同型，只是这次藏在判定脚本里：**判定脚本自己也要有牙齿。**

## 5. 本片仍带着的未裁决项（沿用，不借本审计变成已决）


- **`S19-STD-1`（本片新增，已从静态读代码升级为实测）**：运行期交给激活的聚合 map，与调度器
  继续写入的那个累加器是**同一个对象**（`scheduler.go:220` 写、:224 交出）。探针实测：前驱因条件环
  重跑时三次激活收到的输入地址相同（`0xc00018a000`），节点保存引用后在 `Run` 结束读到的是
  最后一次前驱输出（`b:B3`）而非自己被激活时看到的那一份。修法是交出快照（`maps.Clone` 或重建 map），
  但那是新的用户可观察行为，切片 19 的契约行不覆盖它，本片不实现。需一次独立的
  `rex-test-design`（`behavior-delta`）取红。
- `R19-Q1`（汇聚前驱没跑且再无待派激活时，`Run` 交出 `err==nil` 而 `Output()==nil`）与
  `R19-Q2`（母约 :341 的单前驱 `AddJoin` 示例与 §3.5 第 6 项相冲），均见票面 §15.5。
- `S19-STD-6`：变异执行体已入库（`scripts/mutation/p1r19-join-barrier.mjs`、切片 21 的
  `scripts/mutation/p1r21-duplicate-node.mjs`）但未接进 `make`；
  `golangci-lint` / `make lint` 至今从未运行。
- **切片 22 的一次未复现裸 FAIL（观察中）→ 已有同族线索**：收口核查时全包 plain `-count=1`
  出现一次无包名的裸 `FAIL`，7 次复跑均无法复现（`v3-p2-graph-g3-refactor.md` §2.1）。切片 24
  收口的 anti-flake 首轮又单次判红 `TestP1R17_CancellationWinsOverStepLimitDuringDispatch`
  （其前置守卫分支），隔离复现实验证明 **基线字节同样偶发**（6 核合成负载下 2/3 轮判红）——
  即 R17 夹具对负载敏感是既有性质，与切片 22/24 的接缝无关（登记 S24-STD-2，
  `v3-p3-graph-g6-refactor.md` §2.2）。两案并案观察：若裸 FAIL 再现，优先在合成负载下重放
  R17 场景并取 `-v` 全量。
- **S24-STD-2 复发实测（2026-09-28 第三轮并行收口时）**：三个子代理并行跑测后机器负载均值
  **223**（来源为宿主用户应用：WindowServer/ZCode 渲染器/外部 dev server/Telegram/Chrome/vitest，
  非本仓测试进程），`TestP1R17_*` 三条再次判红 —— 失败文案仍全是夹具自己的前置守卫
  （「派发窗口没建立」「先加强夹具而不是放宽判据」），且 graph 包字节与切片 24 收口态逐字节
  相同（`98b8bdad…`/`4641de52…` 复核）。结论：环境性复发，非回归。**欠项登记**：R17 夹具需要
  一次负载加固片（把「取消时仍在派发」的前提从时序竞赛改成确定性形状 —— 夹具自己的守卫文案
  已经指出方向），独立成契约再做，不在收口里顺手改。
- **S24-STD-2 升级为可稳定复现，且红形态不止「夹具前提」这一种（2026-09-28，提交后全仓门禁实测）**：
  七片提交落地后跑 `go test ./... -count=1`，三轮里一轮出现**无包名的裸 FAIL**（与切片 22 那次同形态，
  这次复现了），定位到 `pkg/hno/graph`；随后 `go test ./pkg/hno/graph -count=10` 在负载均值
  **100.98 → 120.44**（三路契约子代理并行 + 本仓全仓测试）下**稳定判红**，
  回执 `receipt:196ed0f4-9ac9-4516-9da0-ef0348fc4bc2`（exit=1，点名
  `TestP1R17_CancellationWinsOverStepLimitDuringDispatch` 与
  `TestP1R17_CancellationIsNotDisguisedAsNodeBusinessErrorAcrossRounds`）。
  **诚实披露：这不再是单纯的「夹具前提没建立」**——`-count=10` 里出现了一条前提**成立**却仍判红的形态：
  `p1r17:211` 的守卫量到「取消时刻已启动 1/2000 个后继」（远小于预算 100，即安全阀确实在取消之后才该撞上），
  而 `p1r17:219/:223` 报出 Run 交出的是 `step limit exceeded` 而非 `context.Canceled`。
  两条读法都还没被排除：**(a) 守卫读的是过期瞬时值**——`ran=1` 不能约束 `cancel()` 落地那一刻消费者已提交了多少步，
  所以「前提成立」这个自证本身就是时序竞赛的产物（正是 §5 上条要求加固的方向）；
  **(b) 引擎在单次 dispatch 循环内真的不回头咨询 ctx**，于是步数阀的错误抢在取消之前交给调用方
  （那属 `S17-SPEC-3` / `R17-UNADJ-1` 未裁决的同族，母约 :192 已登记「取消后是否也必须报取消」未决）。
  **(a)/(b) 的分辨属 R17 加固片（切片 30）的契约职责**：加固后的夹具必须能*证明*取消时刻预算尚未耗尽
  （不是读一个可能过期的计数），并让「预算在取消之后才被耗尽」成为测试可控的事件。
  在此之前，任何在负载下撞到这三条红的收口都**不得**记为回归，也不得记为已通过。
  - **已结清（2026-09-28，切片 30 收口，见 §4ae）**：加固形落成 Sync 提交停靠点，负载协议 16/16 轮 0 红
    且 0 条前提红，同二进制 A/B 下原始夹具 1/6 轮红（那一红按 D8 分型为**前提红**）。
    **(a) 证实、(b) 在本片可构造的位置上不成立**；「负载协议纳入收口口径」已落成仓库命令
    `bash scripts/mutation/p1r17-load-rounds.sh 14 10 'TestP1R17_'`。
    本条此前那句「不得记为回归、也不得记为已通过」的临时禁令随之撤销：R17 族现在**在负载下也可记为已通过**。
- **切片 21 的 checkout 事故（已恢复，票面 §17.4）**：变异阶段一次 `git checkout graph.go` 把
  切片 2–21 的未提交字节回退到 HEAD 存根，靠切片 20 留在 `/tmp/sinkexp` 的逐字节副本恢复
  （`b7790b9f…` 复核全同）。`S18-SPEC-2` 的「无提交授权」单拷贝风险由口头变成了实测事故；
  在拿到提交授权之前，任何变异/实验脚本都不得对本仓工作树执行 git 破坏性命令。
- 沿用：`R18-Q1`（是否新增导出小类型/哨兵承载归因）、`R18-Q2`（`Typed(name,nil)` 是否升级为构建期拒绝）、
  `S18-SPEC-2`（HEAD `0387000` 只含首片存根，切片 2–19 未提交；**无提交授权**）**已关闭（2026-09-28，见 §4u：
  七片本地提交 `0da4013`…`17f6845`，未 push）**、
  `R17-GAP-1`**已闭合（切片 30：m9「整条删除 ②」现在是 `HardD2` 的 solo 杀手，5/5 实测；见票面 §13.6 第 1 条）**、
  `R17-GAP-2`（σ 等价变异，加固形下仍 `exit 0` ⇒ **本片未给它牙齿**，继续挂账）、
  `R17-GAP-3`（超时形状，**本片未触及**）、
  `R18-GAP-1`、`S17-SPEC-3`/`R17-UNADJ-1`、`S17-STD-5`+`S18-STD-6`
  （`struct.sh`、前两片的 `mutants.mjs`、本审计的 `/tmp/p1-audit` 仍在 /tmp，未入库；
  切片 30 的两份执行体已入库，这条欠账的范围因此缩小到 `struct.sh` 与 p1-b9 那七份）。

- **`S30-EVD-1`（新登记，程序级证据完整性，2026-09-28 全仓扫描实测）**：把 `docs/design/v3-*.json` 里所有
  `receipt:` 前缀的 UUID 与 `.rex-harness/` 对账。**这条对账已落成仓库命令 `node scripts/evidence-audit.mjs`**
  （存在 NOWHERE 引用即 exit 1）。登记口径为 `docs/design/v3-*.json`（唯一 ID 去重后 711 个被引用，127 个不指向
  `receipts/<uuid>.json`）；脚本默认把 `*.md` 一并扫描，故其行数为 149，**两种口径下 NOWHERE 同为 16**。
  **分层读数**：
  - **94 个**在 harness 的别处存在（`activations/*.json` 内的 `activationId`／`evidence/*.ndjson` 的证据 id），
    即**跑过、记过，只是前缀写错了命名空间**——缺陷在引用格式，不在证据本身；
  - **17 个**是 `activations/<uuid>.json` 的真实文件名，同样属前缀误用；
  - **16 个**在整个 `.rex-harness/` 里**任何角色都找不到**，才是真正无法核验的断言。
    分布集中于早期片：`v3-p1-graph-review-verdict.json` 6 个，`slice7-*` 3 个，`slice8-*` 1 个，
    `slice13/14-review-verdict` 与 `slice13/6-testability-decision` 各 1 个，`slice6-test-scope` 1 个。
    切片 30 干净（17/17 命中），切片 31 原有 3 个已在入库前换成真实回执（见 §4ab）。
  - **结构性成因（比个别假 id 更要紧）**：`.gitignore:62` 把 `.rex-harness/` 整目录排除入库，
    故 650 份回执**只存在于这台机器**，提交与克隆都不携带证据。任何「收口已实测」的表述对换机/换人都不可复核。
    这不是要立刻改 `.gitignore` 的授权——把它作为待裁项登记：**是否把收口所需的回执摘要（命令 + exit + stdout sha256）
    随票面写进仓库文本**，使证据至少随提交迁移一次。
  - **纪律修正（对本轮之后的所有派发有效）**：引用 harness 对象时必须写对前缀（`receipt:` / `activation:` / `evidence:`），
    并在停笔前跑一次自动对账而不是自述「已抽查」；切片 31 的 `notes` 自写了这条纪律仍有 3 个脑补 id，
    说明「靠散文提醒」不解决问题，对账必须是**一条命令**（已落成 `scripts/evidence-audit.mjs`，本轮实测 exit 1）。

## 6. 过程如实登记（写作本文件时的自纠）

切片 20 的 `design-tests` 阶段第一版把副本实验写成 `receipt:efa1ac82…` 与 `receipt:b917dd0a…`，
而 `.rex-harness/receipts/` 里对这两个 id `grep -rn` **零命中** —— 它们没有实际回执，是凭印象写下的引用。
本轮已在副本上重跑实验、取了三条真实回执（副本基线 `receipt:7f442439-8bce-4f23-b44f-a5e6d0ca0375` exit 0，
字面版 `receipt:dd11883e-75ad-4d88-844e-2139f94a2f6c` exit 1，限定版 `receipt:f594bc89-77f3-4ca3-bcb2-16fdc621d5ec` exit 1），
替换了 §2 与契约里的假 id，并把 12 / 10 两个数字改为实测值。同时如实登记两条：

1. 第一版对冲突根因的描述偏窄（写成「条件边/兜底边的分支终点」）。实测显示**无条件扇出的终点**
   也被同一判据命中，因此限定版也救不回来 —— 已在 §2 与契约 M3 改写。
2. 实验全程在 `/tmp/sinkexp` 的副本里做，结束时把副本逐文件恢复并与仓库比对
   （两边 `git hash-object --no-filters pkg/hno/graph/*.go` 的 md5 同为 `1250071191a7ddb8…`，
   副本复跑 exit 0），仓库字节未被实验改动。
3. **第二处自纠（同一轮的后续实验）**：本文件与票面第一版还把 B6 第 2 项写成「与 `graph_test.go:953`
   的静默覆盖语义正面对撞、属票面变更、需裁决」。那是读测试名得出的推断，没有跑。副本实测（`-count=1` 与
   `-race` 均 exit 0、0 个判红）推翻了对撞结论：第 2 项是一片正常可取的 `behavior-delta`。
   教训与 §7 第 1 条同源 —— **凡「某判据会与既有测试对撞」的断言，都要用一次副本执行来担保，不得由走查供应。**
