# R19-Q2 裁决材料 — G5 Send 动态扇出与 §3.5 第 6 项（Join 前驱数 ≥ 2）的冲突

> 冲突登记：`docs/design/v3-test-scope-p1-graph.md` §15.5（切片 19 写回）与 `docs/design/v3-platform.md` §6 注记
> 涉及判据：母约 §3.5 第 6 项（切片 19 已交付）、票面 §15（切片 19 屏障语义）、§6 的 map-reduce 示例（G5，P2 未落地）
> 本材料只登记冲突原文、实测证据与三个选项的后果；**裁决权在负责人**，本文不代作决定，也不因本文改动任何判据、示例或测试。

`R19-Q2` 是一条「两段已交付/已登记的文字不可能同时成立」的冲突：母约 §6 的 map-reduce 示例用
**单个前驱名**的 `AddJoin` 表达「同名前驱被 `Send` 运行时扇出多次后聚合」；而切片 19 交付的
§3.5 第 6 项把**前驱数 < 2** 的汇聚声明在构建期拒绝。构建期唯一可数的是「声明的去重前驱节点数」，
它数不出运行期激活次数 —— 两个判据各自诚实，合在一起示例过不了 `Validate`。G5（`Send`）切片契约
起草前必须先对本案给出裁决结论（切片 23 契约 `carriedItems.R19-Q2` 已把它列为 G5 的设计门槛）。

## 1. 冲突原文与实测

### 1.1 两段冲突原文

**其一，母约 §6 的 map-reduce 示例**（`docs/design/v3-platform.md:365–376`，承重的一行是 :372）：

```go
    AddNode(graph.NodeFunc("summarize", /* ... */)).
    AddJoin([]string{"summarize"}, "reduce").   // reduce 聚合全部 summarize 输出
```

`fanout` 靠 `Send{Node: "summarize", In: it}` 在运行时把同一个目标名扇出 N 次；`AddJoin` 只声明了
**一个**前驱名，聚合语义完全依赖运行期激活次数。母约自己在 :379–384 已登记这段示例跑不通、
与 §3.5 第 6 项相冲（未裁决，即本案），并写明「切片 19 未擅自改动本示例」。

**其二，§3.5 第 6 项 as delivered**（`docs/design/v3-platform.md:249–250`）：

> 6. Join 节点的前驱数 < 2 —— **已交付**（切片 19）：`Validate` 点名那个目标并给出实收的去重前驱数，
> `Run` 随之拒绝；判据形状与打红它的变异见票面 §15.1 第 3 点、§15.2 倒数第二行。

原始登记在票面 §15.5（`docs/design/v3-test-scope-p1-graph.md:502–506`）。**行号漂移披露**：登记时引的
是 `v3-platform.md:341`，本次复核该示例行已漂移至 :372（母约在切片 20–22 写回后增长）；冲突本身一字未变。

### 1.2 探针实测

副本实验在 `/tmp/r19q2`：`module github.com/rexleimo/agno-go` + `go 1.24`（工具链 go1.26.3），
`pkg/hno/graph/*.go` 全部 14 个文件原样拷入，与仓库逐字节相同（`git hash-object --no-filters` 前 8 位：
graph.go `72974bf6`、scheduler.go `873f4a82`、policy.go `6d71fa47`，副本与仓库两边一致）；副本上
`go -C /tmp/r19q2 test ./pkg/hno/graph -count=1` → `ok` exit 0，回执
`receipt:041ab361-29c2-4f23-8bb6-2eb0a4795125` —— 副本是完整的、未被改动判据的引擎。

探针 `cmd/probe/main.go` 只调公共 API。**先说明示例今天为什么只能用普通节点写**：`NodeFunc` 的签名是
`func(ctx context.Context, in any) (any, error)`（`pkg/hno/graph/graph.go:42`，单返回值），§6 设想的
`Sender` 三返回值 `(out, sends, err)` 属 P2 尚未落地 —— 所以 fanout/summarize 只能按普通 `NodeFunc` 建，
静态拓扑上只有 `AddJoin` 声明的那一条边。这意味着**无论本案怎么裁，这段示例都需要 G5 的缝才能跑**
（判据只是它今天先撞上的那一堵墙）。

命令（回执 executable=`go`、args 头两格 `-C /tmp/r19q2`、cwd 记录为仓库根）：
`node /tmp/rex-receipt.mjs -- go -C /tmp/r19q2 run ./cmd/probe` →
**`receipt:5421d654-4a9a-4804-b033-4c66f6861670`**（exit 0）。输出逐行：

```
== [A] 母约§6示例形状 AddJoin(["summarize"],"reduce") 无静态扇出边
[A] Validate() = graph: join target "reduce" must declare at least 2 distinct predecessors, got 1
== [A2] 同上但前驱补足2个 仍无静态边
[A2] Validate() = graph: unreachable node "reduce": not reachable from entry "fanout" through any declared edge
== [B] 对照 AddJoin(["summarize","summarize2"],"reduce") + 静态边
[B] Validate() = <nil>
[B] Run err = <nil>, Output() = joined{summarize,summarize2}
== [C] TestP1R19 单前驱形状 AddJoin(["a"],"join")
[C] Validate() = graph: join target "join" must declare at least 2 distinct predecessors, got 1
```

四条读数：

- **[A]** 母约 §6 示例按今天的 API 尽可能照抄后，`Validate` 给出的正是第 6 项的文案，点名目标
  `"reduce"`、报出实收去重前驱数 1 —— **示例今天过不了 `Validate`，这条冲突是实测事实而非走查推断**。
  （且因为 `Validate` 的判序是先汇聚屏障后可达性，`graph.go:310` 在 :313 之前，示例拿到的是 join 文案。）
- **[B]** 对照形状：补到 `["summarize","summarize2"]` 两个**不同名**前驱并加静态扇出边后
  `Validate() = <nil>`，`Run` 里屏障等齐两个前驱、reduce 收到的聚合 map 两个键都在 ——
  证明判据数的是**去重后的声明前驱节点数**，数不出同名激活次数。
- **[A2]** 只把前驱补足 2 个、仍不给静态扇出边（最贴近 `Send` 运行时派发的形状）：
  越过第 6 项后倒在可达性检查 —— 运行期派发没有静态边，构建期 BFS 到不了 summarize/reduce。
  这是「示例本来就需要 G5 的缝」的实测另一半，也封死了「只改示例不引入运行期派发就既能跑又保留原意图」的路。
- **[C]** 钉住判据的那条测试的单前驱子用例形状，错误点名 join 目标 —— 与 [A] 同文案，
  即**改示例（OPT-A）与改判据（OPT-B/C）打的注定是同一行测试**。

### 1.3 判据的钉子与运行期落点（本案动它时必须一起动的部分）

- 构建期钉子：`TestP1R19_JoinWithFewerThanTwoPredecessorsIsRejectedLocatably`
  （`pkg/hno/graph/p1r19_join_barrier_test.go:308`，本次 `grep -rn` 已核实存在），子用例「单前驱/空前驱」，
  要求错误点名目标、且不得退化成「本种类不被路由」那句。判据落点 `rejectUnderfedJoinTargets`
  （`graph.go:356`，文案 `graph: join target %q must declare at least 2 distinct predecessors, got %d`），
  计数来源 `joinBarriers`（`graph.go:328`，按名字去重 —— 注释自认它是「哪些前驱算同一个屏障」的唯一定义）。
- 运行期落点：`joinReady`（`scheduler.go:277` 起）——凑齐判据是**键集合覆盖**
  （`len(collected) < len(s.waits[target])` → 继续等），同名前驱激活两次时 `collected[name] = out`
  （:285）是**覆盖写**，激活拿到的是累加器 map 本体（:289 交出，即 `S19-STD-1` 登记的同对象形状）。
  也就是说：就算把构建期放行，今天的运行期屏障也会在**第一次**同名激活完成时就 fire，N−1 份输出
  直接丢失 —— 「同名多次激活计入汇聚」不是一个改判据数字就能得到语义，它需要新的运行期记账。

## 2. 三个选项

### OPT-A 改示例

- **语义**：map-reduce 示例改用 ≥2 个不同前驱名（每个分片一个节点名），或不用 `AddJoin` 表达聚合
  （普通边 + 用户层聚合节点）。
- **实现代价**：最小 —— 只改母约 §6 一段示例文字，代码零改动。
- **对既有判据的影响**：零。第 6 项、`TestP1R19` 两条钉子、`joinBarriers` 语义全部原样。
- **对 fail-closed 精神的影响**：无损 —— 「声明了屏障却凑不出屏障」继续被构建期挡下。
- **代价**：动态扇出的聚合没有一等表达。示例原本的意图（**同一个逻辑前驱被 `Send` 激活 N 次、
  N 份输出都进 reduce**）失去官方形态 —— 分片数量是运行期决定的，「为每个分片起不同节点名」
  在真 map-reduce 里根本写不出来；示例从「G5 卖点的官方形态」退化为「绕开判据的形状」。
- **适用前提**：负责人若裁定 map-reduce 聚合不做一等表达（明确降级），本项即刻可执行。

### OPT-B 同名前驱计数屏障

- **语义**：把屏障凑齐判据对 Send 场景从「去重前驱节点数」扩为「运行期激活实例数」：
  单前驱名 + `Send` 扇出 N 次 → 等全部 N 次完成后再激活 target 一次。
- **实现代价**：大，且分三层。
  1. 构建期：§3.5 第 6 项必须为「目标是 Send 扇出对象」的声明**让路或分叉**。但 `Send` 属 P2 未落地，
     构建期今天无从知道一个 `AddJoin` 目标将来会不会被扇出 —— 同一条 `AddJoin(["a"], "j")` 的合法性
     将取决于一个尚未存在/运行期才成立的事实，第 6 项「点名目标 + 实收前驱数」的文案也随之分叉。
  2. 运行期：`joinReady` 的覆盖写（`collected[name] = out`）装不下 N 个同名激活 —— 要么丢（语义骗人），
     要么给每次派发身份并改聚合输入形状（破坏票面 §15.1 担保的「键 = 前驱名」公共面形状）。
  3. 与 `S19-STD-1` 耦合：激活拿到的是累加器本体（`scheduler.go:285` 写、:289 交）；激活实例从
     「每名一次」变「每名 N 次」后，那竞态窗口从理论形状变成 Send 场景的必然路径，
     快照修法（`maps.Clone`/重建 map）从可选变成前置。
- **对既有判据的影响**：正面改写第 6 项与 `TestP1R19` 钉住的边界（`AddJoin` 注释里「单前驱就是一条
  无条件边」的立论对 Send 目标不再成立）；`joinBarriers` 作为屏障唯一定义要一分为二，
  恰好制造 `graph.go:326–327` 注释自己警告过的「两处各数一遍、结论相反且都静默」那类裂缝。
- **对 fail-closed 精神的影响**：凑不齐的新路径（派发的激活没全部完成且再无待派激活）与
  `R19-Q1`（屏障不收敛时 `err==nil` 而 `Output()==nil` 的**未裁决**形状）正面相遇 ——
  在 R19-Q1 裁决前落 OPT-B，等于把一个未裁决形状扩成两个；且让构建期校验依赖运行期事实，
  破坏「§3.5 只数声明」的职责边界。

### OPT-C 显式 Send 汇聚声明

- **语义**：新增公共面声明形态（暂名 `AddJoinSend(source, target)`）：「等 `source` 本轮派发的全部
  `Send` 激活完成后激活 `target` 一次」。构建期不数前驱名（**不经** §3.5 第 6 项 —— 它声明的是
  派发源，不是前驱集合），运行期屏障按「source 的 pending Send 计数归零」判凑齐。
- **实现代价**：中 —— 新公共面声明 + 调度器新状态（每 source 的 pending 计数），需要自己的切片契约
  （RED 天然可取：方法今天不存在，调用即编译失败）。聚合输出形状可独立设计（按 `Send.In` 或派发序号
  给键），不必挤进 `map[前驱名]any`。
- **对既有判据的影响**：零 —— `AddJoin` 与第 6 项、`TestP1R19`、`joinBarriers` 全部原样；
  新形态与旧屏障不共用凑齐定义，「构建期按名去重、运行期按实例数」的对账裂缝被结构性排除。
- **对 fail-closed 精神的影响**：保持 —— 凑不齐走既有步数安全阀与 `R19-Q1` 的既有形状，
  不新增静默成功路径；派发源不存在或不是 `Sender` 时构建期可拒绝（与第 6 项同一精神：
  数不出结论就在构建期说出来，而不是跑完后沉默）。
- **代价**：公共面 +1 声明、调度器 +1 状态、文档 +1 概念；`Send` 未落地前它是「先到」的空声明，
  与 `Send` 同片落地才诚实 —— 这正是它需要自己切片契约的原因。

## 3. 推荐与理由

**推荐 OPT-C**，次选 OPT-B，OPT-A 保留为「明确不做一等聚合」时的兜底。推荐理由：

1. **fail-closed 不让路**：两个判据各管各的声明形态，不需要为 Send 弱化或分叉第 6 项；
   凑不齐仍走既有安全阀，不新增 `R19-Q1` 之外的静默成功路径。
2. **语义显式**：「等本轮全部 Send 完成」是用户写得出来的意图；单前驱 `AddJoin` 依赖读者自行知道
   「这个名字会被 Send 扇出」是隐式约定 —— 与切片 18（B11）确立的取向一致：归因显式、引擎不猜。
3. **构建期/运行期职责清晰、不与既有判据对撞**：构建期只数声明（第 6 项原样），运行期新屏障有自己的
   计数来源。这正是票面 §16（切片 20，B6 第 5 项）裁决先例的取向：判据与既有担保相撞时先显式裁决，
   不擅自改判据，也不让示例悬空。
4. **与 R19-Q1 解耦**：OPT-B 把凑不齐的新路径直接压在 R19-Q1 的未裁决形状上；OPT-C 无论 R19-Q1
   之后怎么裁都不需要返工。

**次选 OPT-B** 的唯一理由是公共面不为 Send 增加第二种声明；若选它，必须与 `R19-Q1` 裁决、
`S19-STD-1` 快照修法排进同一次契约，不能单独落，否则上面三层代价会以「静默丢输出」的形状先爆。

**权属声明**：裁决权在负责人。本材料只登记冲突与实测，**不擅自改任何判据、示例或测试**
（本文件的写入是本轮唯一的仓库改动；实验全程在 `/tmp/r19q2`，仓库工作树除回执目录外逐字节未动）。
在负责人给出裁决结论前，G5 切片契约不得起草；本文亦不预写任何一方落地后的文案或测试期望。

## 4. 登记

- 本材料由切片 23 契约 `docs/design/v3-test-scope-p1-graph-slice23.json` 的 `carriedItems.R19-Q2`
  同期登记引用（该条「裁决材料已单独成文」即本文；G5 契约起草前必须先有负责人对该材料的裁决）。
- 状态文档 `docs/design/v3-p1-graph-status.md` §5 的指针由主代理维护，本文不代管、不代改。
- 母约 §6 的冲突注记（`v3-platform.md:379–384`）与本文互为出处；裁决落地后由承接片的写回更新母约、
  票面与判据，本文作为裁决时的实测底稿存档，不再改写。
- 独立性披露：本文为单代理自查材料（取证、实验、成文同一代理），未派发独立审查；
  探针与副本均在 `/tmp`，不入库；引述的回执 id 均为 `/tmp/rex-receipt.mjs` 实际打印：
  `receipt:5421d654-4a9a-4804-b033-4c66f6861670`（探针）、`receipt:041ab361-29c2-4f23-8bb6-2eb0a4795125`（副本基线）。

## 5. 裁决记录

**负责人 2026-09-28 选 OPT-C**（显式 `AddJoinSend(source, target)` 声明形态）。构建期不数前驱名数、不经 §3.5 第 6 项；运行期屏障按「source 的 pending Send 归零」判凑齐。母约 §6 已同步标注；G5 切片契约据此起草，示例随 G5 落地改写。R19-Q2 关闭。
