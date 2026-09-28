# P1 切片 18 · REFACTOR 记录（B11 可归因类型不符）

工作项 `work-p1-graph-r18-typed`，契约 `docs/design/v3-test-scope-p1-graph-slice18.json`。
绑定场景命令逐字节为 `go test ./pkg/hno/graph -run TestP1R18_ -count=1 -race -v`（cwd 仓库根）。

- RED：`receipt:3f831afa-731c-4021-b4db-1f0e4efeb5dd`（exit 1）
- GREEN：`receipt:8674cf3f-6981-40de-b2dd-bd5297c07b22`（exit 0）
- REFACTOR：`receipt:e110a8db-4aa2-4567-bc41-b6112383b15b`（exit 0，stdoutSha256=`a0a76dd56d12ed4117d62b442c2a9908…`）

三者字节相同（graph.go `a8e46a3a2f0baaa699ae60f26498c6f5c1c851cb`）：本阶段是**零实现改动**的
整理判定 + 变异取证，不是又一次改写。

## 1. 整理判定：不动结构，理由与判据同形

契约 completionCriteria 第 1 条要求「实现选择与判据冲突时停下上报」，本节把候选整理逐项判掉：

| 候选 | 判定 | 理由 |
|---|---|---|
| 抽出 `nodeError(name, format…)` 之类的共享错误构造器 | **不做** | 全类型不符只有 `graph.go:502` 一个写点（同文件另 8 处 `%q` 错误分别是构建期校验：`:255`、`:258`、`:262`、`:265`、`:268`、`:271`、`:328`、`:370`，语义不同形、参数不同序）。单点抽 helper 属契约 explicitNonGoals 反对的投机抽象，且会把 D2–D5 的判据面从 `Typed` 引到 helper 上 |
| 复用既有 `%q` 引号惯例 | **已复用** | 新文案 `graph: node %q: …` 与 `:328` 的 `graph: unreachable node %q: …` 同形，这正是 D2/D4 能反向断言「引号边界」的前提（§11.1 先例） |
| 把 `expected` 挪进闭包内每次算 | **不做** | 现在在包装期算一次，成本更低且不引入新状态；变异 `kappa` 证明判据对「算几次」不敏感 |
| 合并 `actualTypeLabel` 进 `Typed` | **不做** | nil 分支是判据 D5 的独立要求（实测 `%!s(<nil>)` 不满足 got 正则），单列函数让 GREEN 的判据来源可读；它是未导出符号，导出面计数仍 24 |
| 给 `Result` 或 `Run` 加类型不符的结构化字段 | **不做** | 属新增导出面（R18-Q1 未裁决），D8 的 `exportedSymbols=24` 会破 |

## 2. 变异矩阵（11 条，执行体 `/tmp/p1-r18/mutants.mjs`，逐条输出存 `/tmp/p1-r18/mutants-<id>.out.txt`）

纪律：每条变异用精确串锚替换当前字节 → 跑绑定命令 → `finally` 之前的同一写回把原文恢复 →
`git hash-object --no-filters` 自检恢复后仍为 `a8e46a3a…`（下表 `restored` 列全为 true，
脚本末行 `finalHash … identical true`）。`red` 列取绑定命令 `-v` 输出里 `--- FAIL: TestP1R18_*`
的 Test 名，不靠推断。

| id | 写坏形态 | exit | 判红的 Test 行 | restored |
|---|---|---|---|---|
| alpha | 整段归因删除，退回基线常量文案 `graph: input type mismatch`（同时删掉 `expected` 以免构建失败） | 1 | D2、D3、D4、D5 | true |
| beta | 两槽同词：`got` 槽填期望类型（= 契约判据 (b)「只报期望不报实际」） | 1 | **D3、D5（D2 仍绿）** | true |
| gamma | 只有类型没有标签：`node "x": string int` | 1 | D3、D5（D2 仍绿） | true |
| delta | `got` 槽打印 `reflect.TypeOf(in)`（= 契约判据 (c)） | 1 | **D5 独杀** | true |
| epsilon | 包上 `%w context.Canceled`（= 契约判据 (d)，票面 §13.1 反向牙齿同族） | 1 | **D7 独杀** | true |
| zeta | 节点名不带引号（`node %s:` 前缀粘连） | 1 | D2、D4、D5 | true |
| eta | 名字槽填期望类型名（`%q` 给 `expected`） | 1 | D2、D4、D5 | true |
| theta | 包级共享状态：总是点名第一个被构造的 `Typed` 节点 | 1 | D2、D4、D5 | true |
| lambda | 业务错误与类型分支共用错误构造（D6 的 teethMutant） | 1 | **D6 独杀** | true |
| mu | 丢弃 TOut、交出 `nil, nil`（D1 的 teethMutant） | 1 | D1、D6、D7 | true |
| kappa | 期望类型打印成指针类型（`*string` 而非 `string`） | **0** | —— 无（等价变异） | true |

生效归属（每条验收行至少一次被证伪，契约 completionCriteria 第 2 条）：
D1 ← {mu}；D2 ← {alpha, zeta, eta, theta}；D3 ← {alpha, beta, gamma}；
D4 ← {alpha, zeta, eta, theta}；D5 ← {alpha, beta, gamma, delta, zeta, eta, theta}；
D6 ← {lambda, mu}；D7 ← {epsilon, mu}。

## 3. 三处诚实更正（契约预测与实测不一致，按实测收口）

1. **契约判据 (a)「节点名写死为入口节点 → D4 红、D2 绿」在 `Typed` 内部不可达**：包装层看不到
   入口、注册序与前驱，唯一可达的「点错名」形态是包级共享状态（`theta`）或名字槽被类型名顶掉
   （`eta`），而这两种都同时打红 D2 —— 因为夹具按文件顺序构造节点，`theta` 捕获到的是 D1 的
   `"len"`。⇒ 登记 `R18-GAP-1`：**D4 的两条反向断言（不含 `"a"`、不含 `"seed"`）至今没有 solo 杀手**，
   D4 的归因能力全靠它与 D2 一起红来担保。要真正隔离 D4，需要一个能读到「另一个节点名」的写坏点
   （例如归因改由调度器/图侧提供，即 P2 之后的形状），本片按 explicitNonGoals 不动 scheduler，
   故不建该夹具。这不是「D4 无牙齿」：`alpha/zeta/eta/theta` 四条都能打红它，只是都不能只打红它。
2. **`mu` 第一次运行记为构建失败而不是判红**：初版写 `_, _ = fn` 而漏了 `typed`，触发
   `declared and not used`，绑定命令返回 `[build failed]` 且 `-v` 里没有任何 `--- FAIL` 行。
   按 rex-tdd 的 RED 第 3 条同类纪律，基础设施失败不得计入合法杀红；已改为 `_, _ = fn, typed`
   重跑，得到上表 D1、D6、D7 三行。**中途那次构建失败的 exit 同样是 1，若只看退出码就会被误记为一次杀红**
   —— 因此本矩阵的判红列只认 `-v` 输出里的 Test 名，退出码单独列。
3. **`kappa` 是等价变异，不是漏网的形态**：`reflect.TypeOf((*TIn)(nil))` 交出 `*string`，
   而 D3 的正则 `expected[^,]*string` 允许标签与类型词之间出现任意非逗号字符，`got` 侧亦然，
   所以整族判绿。这暴露的是本片判据的观察上限（「标签里出现类型词」不等于「报的是值的类型而不是其
   指针形式」），与切片 17 把 σ、ξ 移入观测上限的处置一致：**登记为观测上限，不追加断言去堵**，
   因为要堵它就得钉死措辞，而契约 D2/D3 刻意只钉「可定位」这一最小性质。

## 4. 测试差异审查（test-diff-review）

`git diff` 面上本片新增只有一份测试文件（`p1r18_typed_test.go`，`10bce61b…`，基线不存在）
与 `graph.go` 的一处错误构造，未触碰既有 7 份测试文件的任何断言（逐字哈希不变，见契约 baselineBytes）。
逐条核对：

1. **只用公共入口**：7 个 Test 的观察面是 `graph.New/WithStepLimit/AddNode/NodeFunc/Typed/AddEdge/
   SetEntry/SetOutput/Run` + `Result.Output/Value/Completed` + `errors.Is` + 错误文本。
   没有一处读 `cfg/plan/scheduler/pending/running`，没有 `queue` 长度或 goroutine 数（票面 :40）。
2. **无 sleep、无时序前提**：`grep -nE 'time\.(Sleep|After|NewTimer)' p1r18_typed_test.go`
   → 1 处命中且是 `:19` 的注释（声明它不参与定序），**调用为 0**（票面 :75）。类型不符是确定性事件，
   D2–D5 在 `-count=20 -race` 下逐轮同集判红、修复后逐轮同集判绿。
3. **没有把期望值改成当前错误输出**：契约 forbiddenShortcuts 第 1 条禁的
   `strings.Contains(err, "input type mismatch")` 在本文件里 0 命中；`input type mismatch` 只出现在
   实现侧文案与注释中，断言用的是引号包裹的节点名与 expected/got 两个标签。
4. **没有放宽或跳过**：无 `t.Skip`、无 `|| true`、无容差；D5 的 got 渲染允许三种字样是**设计期就写进契约**的
   可接受渲染集合（`no value | <nil> | nil`），不是修复后回填的容差 —— 基线上这四项全部 match=false（实测见
   `docs/design/v3-p1-graph-r18-green.md` §1 第 3 条）。
5. **没有用内部调用次数代替行为**：无 mock 计数、无「Typed 被调用了几次」的断言；
   `Node.Name()` 只用于表驱动夹具自指（`SetEntry(tc.node.Name())`），不参与任何判定。
6. **归因两面各管一段**：类型标签判据只扫描剥离引号片段后的文本（`p1r18Labels`），
   点名判据只看引号片段。这样「节点名里含类型词」不能替类型标签顶包，反之亦然；
   `beta/gamma/delta` 三条变异正是在这个切分下才只打红 D3/D5 而不打红 D2。

## 5. 判据与测量（最终字节复跑）

| 项 | 结果 |
|---|---|
| 绑定场景 | `receipt:e110a8db-4aa2-4567-bc41-b6112383b15b` exit 0，7/7 PASS |
| 重复确定性 | `go test ./pkg/hno/graph -run TestP1R18_ -count=20 -race` → `ok … 1.424s`，0 FAIL，0 DATA RACE |
| 包级回归 | `go test ./pkg/hno/graph -race -count=1` → `ok … 2.529s`（切片 1–17 的 55 个 Test 一个不少、断言未放宽） |
| 静态 | `gofmt -l pkg/hno/graph/` 无输出；`go vet ./pkg/hno/graph/` 通过 |
| 结构 | `sync_dot_graph=0 sync_dot_scheduler=0 recover_scheduler=1 ctx_checks_in_scheduler=4 loc_nonblank_nontest=663 testfunc_total=62`；`exportedSymbols=24` |
| 覆盖 | `Typed 100.0%`（基线 0.0%）、`actualTypeLabel 100.0%`、`total 99.1%`（基线 96.6%） |
| 字节 | graph.go `a8e46a3a…`（本片唯一产品改动文件）、scheduler.go `0ce6c3b3…` 逐字不变、测试文件 `10bce61b…` |

`golangci-lint` 仍未运行（本机 `make lint` 工具链未确认）—— 如实登记，不写成已过。

## 6. 未闭环项

1. `R18-GAP-1`（D4 无 solo 杀手，见 §3.1）；`R18-GAP-2`（`kappa` 型等价变异，按观测上限处置）。
2. `R18-Q1`（是否新增导出哨兵/As 目标，使归因不依赖错误文本）、`R18-Q2`（nil fn 是否升级为构建期拒绝）
   —— 两项都属票面 §1/§6 的未裁决面，留给负责人裁决，本片不动导出面也不动 `Validate()`。
3. `writeBackOwed`：票面 :70 的「可判定」需精确化为「可归因 = 点名节点（引号边界）+ expected/got 两槽位 +
   nil 渲染为无值」；母约 §3.4「跨节点值以 `any` 流动」应补一句「类型不符的归因由包装层给出，调度器不参与」。
4. 结构判据执行体仍在 `/tmp/p1-b9/struct.sh`（S17-STD-5 债务），本片沿用同一脚本，未新增。
5. 切片 17 遗留的 `R17-GAP-1/2/3`、`S17-SPEC-3`、`R17-UNADJ-1/2/3` 等未裁决项与本片无关，未顺带收口。
