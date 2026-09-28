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
- **切片 21 的 checkout 事故（已恢复，票面 §17.4）**：变异阶段一次 `git checkout graph.go` 把
  切片 2–21 的未提交字节回退到 HEAD 存根，靠切片 20 留在 `/tmp/sinkexp` 的逐字节副本恢复
  （`b7790b9f…` 复核全同）。`S18-SPEC-2` 的「无提交授权」单拷贝风险由口头变成了实测事故；
  在拿到提交授权之前，任何变异/实验脚本都不得对本仓工作树执行 git 破坏性命令。
- 沿用：`R18-Q1`（是否新增导出小类型/哨兵承载归因）、`R18-Q2`（`Typed(name,nil)` 是否升级为构建期拒绝）、
  `S18-SPEC-2`（HEAD `0387000` 只含首片存根，切片 2–19 未提交；**无提交授权**）、
  `R17-GAP-1/2/3`、`R18-GAP-1`、`S17-SPEC-3`/`R17-UNADJ-1`、`S17-STD-5`+`S18-STD-6`
  （`struct.sh`、前两片的 `mutants.mjs`、本审计的 `/tmp/p1-audit` 仍在 /tmp，未入库）。

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
