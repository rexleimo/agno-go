# 切片 19 RED 观察（票面 B2：并行扇出 + Join 汇聚屏障）

工作项 `work-p1-graph-r19-join-barrier` / activation `48130ef2-0ff7-4e14-87dc-b826a8acb970` /
stage `red`。契约 `docs/design/v3-test-scope-p1-graph-slice19.json`（D1–D7）。
测试文件 `pkg/hno/graph/p1r19_join_barrier_test.go`（新增，`8bf4b2f9296eb5781e7f6243c234a1baada0c92f`）。

## 1. 绑定场景命令与回执

逐字命令（cwd `/Users/rex/codes/agno-go`，一律 `receipt -- go test …` 直接取，`executable` 记录为 `go`）：

```
go test ./pkg/hno/graph -run TestP1R19_ -count=1 -race -v
```

| 回执 | exit | 关键输出 |
|---|---|---|
| `receipt:279aa8da-e3b3-4fc3-9c8b-a31059f1461b` | 1 | 仅 D1 候选时：`--- FAIL: TestP1R19_JoinBarrierRunsExactlyOnce`，stdout sha256 `e94ff372…6ea9`（decide-testability 用的 redCandidate） |
| `receipt:ebd73748-a3ba-4e30-99e1-73ae95452fbb` | 1 | D1–D7 全量时：7 个顶层 Test 全部 `--- FAIL`，`FAIL github.com/rexleimo/agno-go/pkg/hno/graph 2.055s`，stdout sha256 `dcd4c7da…5b47` |

确定性：同命令 `-count=5 -race` 复跑得 `35` 行 `--- FAIL: TestP1R19_*`，即 7 行 × 5 轮全红，
无竞争窗口依赖（对照切片 17 的 D2 需要 600 轮取样）。

## 2. 逐行观察（失败输出取自 `receipt:ebd73748` 的 `-v` 正文）

| 行 | Test（文件行号） | 观察到的实际结果 | 与缺失行为是否一致 |
|---|---|---|---|
| D1 | `TestP1R19_JoinBarrierRunsExactlyOnce`（`:71`） | `含 Join 汇聚屏障的合法图被 Validate 拒绝：graph: join edge from "a" to "join" is declared but not routable by this engine` | 一致 —— B2 的图根本进不了路由 |
| D2 | `…JoinInputIsKeyedByPredecessorName`（`:124`） | `Run 未执行汇聚屏障图：…not routable by this engine` | 一致 |
| D3 | `…JoinActivatesOnlyAfterBothPredecessorsFinished`（`:193`） | 同上 | 一致 |
| D4 | `…JoinOutputFlowsToDownstreamAndProjection`（`:240`） | `Run 未执行「汇聚后再下游」的图：…not routable` | 一致 |
| D5 | `…LegalJoinGraphValidatesNilWhileIllegalStillRejected` 两个子用例（`:273`、`:289`） | 正例 `Validate() = …not routable…, want nil`；反例 `错误 "…not routable…" 没有点名不存在的汇聚前驱 "ghost"` | 一致 —— 收缩（放行合法 Join）与不变式（端点仍拒且可定位）今天都拿不到 |
| D6 | `…JoinWithFewerThanTwoPredecessorsIsRejectedLocatably`：`单前驱` 判红（`:327`），`空前驱` **判绿** | 单前驱：`错误 "…not routable…" 仍是「本种类不被路由」那句，不是前驱数不足的判定` | 一致（契约 :measuredBeforeDesign 已实测到这一退化） |
| D7 | `…UnexecutedPredecessorEndsAtStepLimitNotSilence`（`:363`） | `这张带安全阀的汇聚图被 Validate 拒绝：…not routable` | 一致 |

**green-at-red 披露**：D6 的 `空前驱` 子用例在产品字节变更前就通过 —— `AddJoin(nil,"join")` 不声明任何边，
于是 `join` 落到不可达检查上，错误里既有 `"join"` 也不含 `not routable`。契约把 D6 的主体判据写在
`单前驱` 形状上（票面 :65 的「前驱数 < 2」），`空前驱` 只是同一规则的下界对照；它不作为本行的牙齿，
GREEN 之后仍保留为护栏。这不是「靠删断言/放宽拿红」：该行断言在 RED 时未做任何修改。

## 3. 失败原因（red-failure-reason）

三条代码事实共同构成这一片 RED 的原因，全部可核对：

1. `pkg/hno/graph/graph.go:137-139` 的 `edgeKind.routable()` 是白名单，`edgeJoin` 不在其中；
   `Validate()` 在 `graph.go:271` 把不可路由种类整条声明拒掉，因此 D1–D5、D7 的图在构建期就出局。
2. `pkg/hno/graph/scheduler.go:194-215` 的 `successors` 的 switch 只有 `edgeUnconditional`、
   `edgeConditional`、`edgeDefault` 三支；全文没有任何汇聚记账（`grep -n 'join\|Join' scheduler.go` → 0 命中），
   所以「凑齐之前不激活」这条 B2 的核心不变式在实现里不存在。
3. D6 的失败原因与前两条不同：它不是「没实现所以放行」，而是「没实现所以先撞上种类拒绝」，
   前驱数判据因此今天不可区分 —— 这正是契约 §measuredBeforeDesign 第二条登记的内容。

因此本 RED 观察到的全部是**目标行为缺失**，没有语法、环境或夹具失败：同一命令的编译通过、
`go vet ./pkg/hno/graph/` 无输出，且既有 62 个测试（8 份文件）在产品字节不变的前提下全绿：

```
go test ./pkg/hno/graph -count=1 -race -run 'TestP1G_|TestP1R1[3-8]'   →  ok  2.518s
```

基线字节复核（RED 时零改动）：`graph.go a8e46a3a…`、`scheduler.go 0ce6c3b3…`、
`graph_test.go 425ea030…`、p1r13 `bc4ab8e7…`、p1r14 `0ed607db…`、p1r15 `3ea06d27…`、
p1r15b `0285ab35…`、p1r16 `6934a4f2…`、p1r17 `cbc8b692…`、p1r18 `10bce61b…`。

## 4. D7 夹具为什么必须是「图仍在往前走」的形状

契约 D7 点名票面 :63 的步数安全阀作为「屏障凑不齐」的兜底。第一版夹具（只有
`entry --cond(false)--> a`、`entry-->b`、`AddJoin([a,b],join)`）拿不到那个机制：`a` 不激活 ⇒
`join` 不进 pending ⇒ 某一刻 `pending` 空且 `running==0`，`consume`（`scheduler.go:94-97`）会直接走
**成功分支**，永远到不了 `dispatch` 里的预算判据（`scheduler.go:136-138`）。所以 RED 观察用的夹具补了
一条会计数的 `b` 条件自环，让图在屏障未凑齐时仍持续消耗步数，安全阀才真的落下。
这是夹具形状修正，不是断言修正：D7 的期望（`errors.Is(err, ErrStepLimitExceeded)`、`res==nil`、
有界阀内返回）逐字未动。

## 5. 由第 4 节发现的新风险（登记为 R19-Q1，不在本片私自加断言）

上面那段推理指向一个 GREEN 之后才成为可能的形状：**Join 的声明前驱永不执行、且图再无待派激活时，
`Run` 会走成功分支交出 `err==nil` 而 `Output()==nil` 的 Result** —— 票面 :161-168 的 fail-closed 精神
与切片 9 起反复钉住的「静默空结果」正是这一形状。

- 当前状态：**未实测**。今天这类图在构建期就被 `not routable` 拒掉，公共面上观察不到；
  依据只是 `scheduler.go:94-97` 的成功分支与 `complete`/`successors` 的既有路径。
- 本片处置：不为它新增断言（契约 D1–D7 里没有这一行，私自补等于在 TDD 阶段扩大测试范围），
  也不把它当作已解决问题写进 GREEN 说明。
- 需要 rex-harness 决定的事项：`R19-Q1` —— 该形状是否属于 B2 的验收面；若属于，需要一条新的
  `rex-test-design` 周期把「凑不齐且已静默」与 D7 的「凑不齐但仍在走」分成两行，并定下报什么错误
  （新的哨兵？复用 `ErrStepLimitExceeded`？点名 join 目标的一条可归因错误，沿用 §11.1 先例）。
  GREEN 阶段我会在真实字节上实测这个形状，把「预测」换成观察再回报。

## 6. 未闭环

- `make lint` / `golangci-lint` 本片仍未运行（沿 `S18-STD-6`）。
- 变异牙齿矩阵在 REFACTOR 阶段执行，RED 阶段不做实现改动、也不声称任何检测力。
- `p1r19_join_barrier_test.go` 里的 `time.After` 只作停摆上界阀（`:43`），`grep time.Sleep` 命中 0；
  若 GREEN 后任何一行依赖它判绿，须按票面 :75 判为不合法证据。
