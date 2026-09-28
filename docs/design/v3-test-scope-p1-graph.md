# 测试范围契约 — `work-p1-graph-engine`

> 治理来源：`docs/design/v3-delivery-ticket.json#work-p1-graph-engine`（outcome + 6 条 completionCriteria + 4 条 verification）
> 设计定稿：`docs/design/v3-platform.md` §3（G2 图引擎），借鉴依据实测见 `docs/design/v3-baseline-evidence.md` §4/§5
> 前置：`work-p0-stream-loop-consolidation-v2`（循环收口，判据 1–4 已过，2 个既存内核缺陷已修）
> 本契约是本周期的行为边界，不得由实现、工期或当前测试结果反向改写。

## 1. 公共入口与观察面

唯一观察面 = `pkg/hno/graph` 的导出 API，测试写在 **`package graph_test`**（外部测试包）以强制「公共面足以表达全部判据」：

```go
graph.New(opts ...graph.Option) *graph.Graph
(*Graph) AddNode(Node) *Graph
(*Graph) AddEdge(from, to string) *Graph
(*Graph) AddConditional(from, to string, Predicate) *Graph
(*Graph) AddDefault(from, to string) *Graph
(*Graph) AddJoin(from []string, to string) *Graph
(*Graph) SetEntry(name string) *Graph
(*Graph) SetOutput(name string) *Graph
(*Graph) Validate() error
(*Graph) Warnings() []error            // 非致命构建期提示（判据 8）
(*Graph) Run(ctx context.Context, in any) (*Result, error)
graph.NodeFunc(name string, func(ctx context.Context, in any) (any, error)) Node
graph.Predicate func(out any) bool
graph.WithMaxConcurrency(n int) Option // 0 = 不限
graph.WithStepLimit(n int) Option      // recursion_limit，默认 1000（未显式设置时参与判据 8 告警）
graph.Typed[TIn, TOut](...)            // §3.4 编译期类型安全出口
graph.ErrStepLimitExceeded             // 哨兵
```

`Result` 的可观察面（刻意做成与调度时序无关的形状）：

| 方法 | 语义 | 为什么它能被安全断言 |
|---|---|---|
| `Output() any` | `SetOutput` 节点的返回值 | 因果确定 |
| `Value(name string) (any, bool)` | 指定节点的输出 | 入口→输出节点的输入传递由此钉死 |
| `Completed() []string` | 已完成节点名的**字典序**切片 | 并行分支间不做全序断言，避免时序耦合假绿/假红 |

**不允许的观察面**：scheduler 内部字段、`queue` 长度、goroutine 数、私有类型断言、内部计时。

## 2. 用户目标

吸收 adk-go 的**控制流图调度 + 构建期校验 + 单消费者零锁模型**，落地为 `pkg/hno/graph`：一个不依赖 `workflow`/`agent`、可被 P6 编译承载、可被 P3 动态扇出扩展的图引擎内核。价值判据是「有环图不会锁死、非法图在构建期就被拒、并发状态无锁」。

## 3. 明确非目标

- ❌ G3 节点策略四合一（Retry/Cache/Timeout/Trace）→ P2。本契约不测重试行为，`WithRetry`/`WithCache`/`WithDurability`/`WithStreamMode` 选项**不在本轮实现**（避免 §3.2 API 出现无行为的空壳）。
- ❌ G5 `Send` 动态扇出 → P3；§3.3 `queueItem.next` 字段留待该轮引入。
- ❌ G4 StreamMode 七模式 → P4；G7 事件化 + HITL resume → P5；G8 Store → P8；G10 workflow 迁移 → P6。
- ❌ `pkg/hno/runner` 侧 Agent 实现 `graph.Node`（§3.1 的第二行）→ 需 agent 公共 API 决策，另立工作项。
- ❌ `InputSchema`/`OutputSchema` 校验 → 设计已划到 v3.1。
- ❌ 改动 `pkg/hno/agent`、`pkg/hno/runner`、`pkg/hno/workflow` 的任何行为或既有测试。
- ❌ P0 遗留 judgement 项（内核 `StopMaxTurnsReached`/`StopToolCallLimit` 枚举拆分、`StopReason` 越过 agent 边界的类型化）：属 runner/agent 公共 API，与本票 `pkg/hno/graph` 无交集，**不得借本票顺带改动**，另立工作项。

## 4. 范围内行为（每条均为外部调用方可观察）

| # | 行为 | 精确期望 |
|---|---|---|
| B1 | 线性 DAG 执行与输入传递 | `entry→out`：entry 收到 `Run` 的 `in`；out 收到 entry 的输出；`Output()` 等于 out 的返回值；`Completed()==["entry","out"]` |
| B2 | 并行扇出 + Join 汇聚屏障 | `entry→{a,b}→join`：join 恰好执行一次；`join` 的输入含 a、b 两者的输出（按前驱名键控）；a、b 均先于 join 完成。**按 §15 精确化：激活条件是「声明的全部前驱在本次 Run 内完成」这一屏障判据（不是最后一个前驱完成即触发、也不是 §4.1 第 3 条禁止的等固定时长）；聚合输入的形状是 `map[string]any`，键 = 前驱名、值 = 该前驱的输出；屏障凑不齐时由 `WithStepLimit` 安全阀以 `ErrStepLimitExceeded` 结束，不静默交出半成品** |
| B3 | 条件路由 + Default 兜底 | 谓词命中 → 该目标执行且 `Completed()` 不含兜底目标；谓词皆不命中 → 仅兜底目标执行；两条都不命中的非兜底节点不得执行 |
| B4 | 合法环收敛 + stepLimit 安全阀 | 条件边计数环在 N 次后走 false 分支收敛，`Run` 返回 nil error；把 `WithStepLimit` 设为小于所需步数 → `errors.Is(err, graph.ErrStepLimitExceeded)` 且 `Output()` 不可用 |
| B5 | 无条件边成环 = 构建期错误 | `AddEdge` 造出 A→B→A：`Validate()` 返回非 nil，且该 error 可判定为「无条件环」类；`Run` 不得启动它 |
| B6 | 其余 7 项构建期校验 | 悬空边、重复节点名、无入口/入口不存在、不可达节点、~~无出口（非输出节点无出边）~~、Join 前驱数 < 2 → `Validate()` 非 nil；合法图 → `Validate()` 为 nil。**「无出口」一项经 §16 裁决作废（OPT-A，本引擎允许无条件扇出终点与分支终点）；「重复节点名」已随切片 21 落地（§17：`Validate()` 末位点名拒绝，`Run` 随之拒绝），B6 全项闭合** |
| B7 | stepLimit 缺失告警（判据 8） | 含**合法条件环**且未显式 `WithStepLimit` 的图：`Validate()` 为 nil、`Run` 正常、`Warnings()` 含该告警；显式设置后 `Warnings()` 不含它 |
| B8 | 并发上限 + FIFO 等待队列 | `WithMaxConcurrency(1)` 且 a、b 并行：a、b 的执行区间**不重叠**（用节点内 gate 同步，不用 sleep 定序）；激活顺序为声明顺序，即第二个节点观测到的前序完成集合包含第一个 |
| B9 | 取消归一化 | 节点阻塞期间 cancel `ctx`：`Run` 返回错误且取消语义为 `context.Canceled`（`errors.Is`），不得伪装成节点业务错误。**按 §13 精确化：取消须在每条交出结论的路径上现问 `ctx`（三处，见 §13.1），入口快照不算咨询；未取消时也不得反向归一化成取消** |
| B10 | panic 转错误 | 节点内 `panic("boom")`：`Run` 返回非 nil error，进程不崩，测试不中断 |
| B11 | 泛型包装 | `graph.Typed` 包装的节点接到 `TIn`、返回 `TOut`；对错误类型输入返回**可归因**错误而非 panic。**按 §14 精确化：可归因 = 三要素同时出现在错误文本里（引号边界的 offending 节点名 + 期望类型 + 实收类型），且 `nil` 实收须渲染为「无值」而不是格式化噪声；不要求新增导出哨兵（§11.1 先例）** |
| B12 | 零锁结构判据 | `pkg/hno/graph` 非测试文件中 `sync.`（Mutex/RWMutex/Once/WaitGroup）出现次数为 0；可变状态仅由消费者 goroutine 写 |
| B13 | 规模判据 | 引擎核心非测试 LOC ≤ 1500（票面判据 6） |

### 4.1 反例条款（这些情况必须仍然判红）
- 用 `time.Sleep` 给并发场景定序 → 不算证据，必须用 channel gate / WaitGroup-free 的显式握手。
- `Completed()` 若返回插入序 → 并行分支间时序会抖动，禁止改回。
- Join 用「等固定时长再聚合」实现 → B2 断言的输入内容仍可能通过，故 B2 必须同时断言 join 只执行一次与输入含两个前驱键。

## 5. 范围内 / 外的测试缝

**允许**：新增 `pkg/hno/graph/graph_test.go`（`package graph_test`）与后续同包测试文件；新增 `pkg/hno/graph` 下的实现文件。
**RED 夹具声明（诚实记账，非隐藏）**：新包的第一个 RED 无法在「包不存在」上合法成立——按 `rex-tdd` RED 第 3 条，编译失败/`undefined` 属基础设施失败。故 RED 前置一个**零逻辑骨架** `pkg/hno/graph/graph.go`：仅 §1 的导出签名 + `Run`/`Validate` 返回 `ErrNotImplemented` 的占位体，无调度、无路由、无校验逻辑。行为红 = 断言拿到 `ErrNotImplemented` 而非期望输出；GREEN 阶段才填实这些体。
**禁止**：修改或删除任何既有测试；为测试导出内部可变状态；把 `Node.Run` 签名改成带 scheduler 上下文的形式来方便实现；用 `_test.go` 里的实现副本代替真引擎。

## 6. 验收行为 → 公共入口 → 可观察断言 映射（票面 6 条判据全覆盖）

| 票面判据 | 覆盖行为 | 公共入口 | 断言形状 |
|---|---|---|---|
| 1 `Graph/Node/Predicate` 按 §3.2，`Node.Run(ctx, any)(any, error)` | B1, B11 | `New/AddNode/AddEdge/SetEntry/SetOutput/Run`、`Typed` | `Output()` 值相等 + `Value()` 输入传递 + `Completed()` 字典序 |
| 2 单消费者零锁，全程无 mutex | B12, B2, B8 | `Run` | `grep sync.` 计数 0（结构判据）+ 并行/Join/限流行为在 `-race` 下绿 |
| 3 8 项构建期校验 | B5, B6, B7 | `Validate` / `Warnings` | 每种非法图一个表驱动子用例，断 `Validate()` 非 nil；告警项断 `Validate()` nil + `Warnings()` 命中/不命中 |
| 4 `maxConcurrency` 打满进 FIFO 并自动派发 | B8 | `WithMaxConcurrency` + `Run` | 节点内记录 `[name,start,end]` 轨迹，断区间不重叠 + 后启动者观测到前者已完成 |
| 5 五类图各有端到端测试 | B1(DAG)/B2(并行+汇聚)/B3(条件)/B4(环) + B5(非法图拒绝) | `Run` | 每类一图，断 `Output()`/`Completed()`，端到端不碰内部 |
| 6 引擎核心非测试 LOC ≤ 1500 | B13 | — | `find pkg/hno/graph -name '*.go' -not -name '*_test.go' \| xargs wc -l` |

票面 4 条 verification 命令全部作为门禁：`go build ./...`、`go test ./pkg/hno/graph/... -count=1`、`go test -race ./pkg/hno/graph/... -count=1`、`go vet ./pkg/hno/graph/...`。

## 7. 完成判据

1. B1–B13 全部有测试并通过；票面 6 条判据逐条有上表映射的测试或结构命令背书。
2. `go build ./...`、`go test ./pkg/hno/graph/... -count=1`、`go test -race ./pkg/hno/graph/... -count=1`、`go vet ./pkg/hno/graph/...` 四条全 exit 0（零退出回执）。
3. 邻近回归不红：`go test ./pkg/hno/agent/... ./pkg/hno/runner/... ./internal/session/contract/... -count=1` exit 0。
4. `gofmt -l pkg/hno/graph` 无输出。
5. 非测试 LOC ≤ 1500 且 `sync.` 计数为 0 —— 两个数字都要在证据里给出实测输出。

**否定条款**：不接受「测试通过但断言落在 mock 上」；不接受为过判据放宽 B12/B13；不接受删测试、跳测试、`t.Skip`、把期望改成实际错误输出。

## 8. 最小纵向切片

**B1 线性 DAG**（`entry→out` 两节点，断输入传递、输出投影、完成集合）。

为什么它足以代表本次目标：五类图里 DAG 是其余四类的退化形态，而 B1 一旦成立就必须同时存在「consumer 唯一读取完成事件」「activation 后继节点」「前驱输出作为后继输入」「输出节点值投影」四件事——这正是 §3.3 零锁模型与 §3.4 `findSuccessors` 的风险中心。若这一条能红→绿，后续并行/Join/条件/环/校验都是在同一骨架上增量，不再引入新的并发不变式。

## 9. 禁止假通过条款

- 断言只允许走 §1 公共面；任何 `//go:linkname`、内部包别名、导出内部字段都视为假通过。
- 并发类断言（B2/B8/B9）必须在 `-race` 下运行；未跑 `-race` 的绿不算完成。
- 结构判据（B12/B13）必须由命令输出背书，不接受人工声明。
- B7 的「告警」不得实现为 `panic` 或 `log.Fatal`；不得塞进 `error` 返回值污染 `Validate()` 的致命语义。

## 10. 聚焦测试命令与回执

```bash
# RED / GREEN 共用（本契约的聚焦命令，逐字用于 receipt）
go test ./pkg/hno/graph/... -run P1G -count=1

# 邻近回归
go test ./pkg/hno/agent/... ./pkg/hno/runner/... ./internal/session/contract/... -count=1

# 票面 verification
go build ./... ; go vet ./pkg/hno/graph/... ; go test -race ./pkg/hno/graph/... -count=1

# B12 / B13 结构判据
grep -c 'sync\.' $(find pkg/hno/graph -name '*.go' -not -name '*_test.go')
find pkg/hno/graph -name '*.go' -not -name '*_test.go' | xargs wc -l
```

## 11. 切片 2 精确期望（承接 `v3-p1-graph-review-verdict.json` 阻断项）

> 追加原因：`docs/design/v3-p1-graph-review-verdict.json` 的阻断项 SPEC-1/SPEC-2/SPEC-3/SPEC-4 属本契约 B5/B6/B7 与 §1 公共面之内，**不是新增目标**，只是把 B6 的 8 项校验拆成可独立失败的 RED 次序。契约的行为边界未变。

1. **B6-悬空边（本切片唯一 RED）**：出边的 `from` 或 `to` 不在节点集合内时
   - `Validate()` 返回非 nil，且错误文本包含 offending 节点名（可定位，不要求新导出哨兵）；
   - `Run(ctx, in)` 返回非 nil 错误且**不改变已绿切片的可观察行为**（线性图仍绿）；
   - 崩溃禁止条款：非法图必须经错误返回，不得出现 nil 节点 panic（审查实测 `receipt:fea0030e-136a-4c56-bf8a-89cf6965cd3f` 的 SIGSEGV 即为本条的反例）。
2. **构建期拒绝的可达路径澄清（设计注记，非放宽）**：§3.2 的 `Add*` 返回 `*Graph` 而非 `error`，因此「`Add*` 内建校验」在公共面上只能表现为：`Add*` 保存声明，非法声明由 `Validate()` 与 `Run()` 前置复查报错。不为此改 `Add*` 签名（那会偏离 §3.2 与票面判据 1）。
3. **后续 RED 次序**（各自单独取红，不在本切片顺带实现）：B6 其余 6 项 → B5 无条件环 → B7 告警 → B4 默认 `stepLimit=1000` 与安全阀 → B8 并发上限 FIFO → B2 并行+Join → B3 条件路由+Default → B9 取消 → B10 panic 转错误 → B11 Typed。
4. **聚焦命令不变**：`go test ./pkg/hno/graph/... -run P1G -count=1`（与前序 RED/GREEN 逐字同命令，保证回执可比）。

## 12. 切片 3 精确期望（cycle-2 审查阻断项的闭包次序）

> 追加依据：`docs/design/v3-p1-graph-review-verdict.json` 的 `cycle2.axes.spec.findings`。
> 本节只改变 §11.3 的 **RED 取用次序**，不新增、不删除 §4 的任何行为，也不改 §1 的公共面；
> 因此属测试设计职权内的次序修正，不是目标变更。
> 次序修正的理由：§11.3 把路由（B2/B3）排在最后，而 cycle-2 实测证明「声明被静默丢弃」在
> `Validate()` 落地后已从「无校验」升级为「校验通过却产出静默错误的空 Result」——
> 比崩溃更难被调用方发现，故必须最先取红。

### 12.1 本切片行为（两条，各自独立取红）

**R1 声明要么被路由、要么被拒绝（fail-closed 不变式，永久成立）**

- 前置：图只声明 `AddConditional` / `AddDefault` / `AddJoin` 中任意一种、且调度器尚未实现该类边。
- 期望：`Validate()` 返回非 nil，错误文本可定位到 offending 声明（节点名或边类型）；`Run` 返回非 nil；
  两条路径都不得 panic，也不得返回 `Output()==nil` 且 `err==nil` 的 Result。
- 反向：只使用已实现边类型的合法图，`Validate()` 必须仍为 nil（R1 不得变成一律拒绝）。
- 该不变式的理由：§3.5 的精神是「非法/不受支持的声明在构建期被拒」，而不是等运行时静默出错。
  随着 R2/B2 逐类实现，被拒绝的集合会缩小，但「未实现的声明不被静默接受」这一点永不放松。

**R2 条件边与 Default 兜底按 §3.4 规则 2/4 路由（B3）**

`§3.4` 原文：`1. 无条件边 → 全部激活`、`2. 条件边 → Predicate(out)==true 激活`、
`4. 若无任何具体边命中 → 激活 AddDefault 边`。可观察期望（全部只经 §1 公共面断言）：

| 场景 | 图 | 期望 `Completed()`（字典序）与 `Output()` |
|---|---|---|
| 谓词命中 | `a --cond(p=true)--> b`，`a --> c`（Default 指向 c） | 只激活 b；`Completed()` 不含 c；Default 不得因「有条件边未全命中」以外的理由触发 |
| 谓词不命中且无其它具体边 | `a --cond(p=false)--> b`，`a --default--> c` | 只激活 c；b 不执行 |
| 多条条件边命中 | `a --cond(p1=true)--> b`、`a --cond(p2=true)--> d` | b、d 都激活（条件边之间是扇出，不互斥） |
| 无条件边存在时 | `a --> b` 且 `a --cond(p=false)--> x`、`a --default--> c` | b 执行、x 不执行、**c 也不执行**（已有具体边命中 → 不触发兜底） |
| 谓词入参 | `AddConditional` 的 `Predicate` 收到的是源节点本次输出的值 | 用谓词内部记录实参并断言等于 `Result.Value("a")` |

### 12.2 本切片明确不做（留待后续 RED 次序）

- B2 并行 + Join 汇聚屏障（R1 在 Join 未实现期间继续拒绝其声明）。
- B5 无条件边成环的构建期拒绝、B6 余下 4 项（重复名 / 不可达 / 非输出汇点 / Join 前驱数）。
- B4 stepLimit 安全阀与 B7 缺失告警。
- B8 并发上限 FIFO、B9 取消归一化、B10 panic 转错误、B11 `Typed`。
- 契约 §3 的全部非目标不变（G3/G4/G5/G7/G8/G10 与 P0 遗留 judgement 项）。

### 12.3 反例条款（这些实现方式必须仍判红）

- 把 Default 实现成「谓词为假就触发」而忽略规则 4 的「无任何具体边命中」前提 → 第 4 个场景判红。
- 把条件边实现成「命中第一条即停」的互斥选择 → 第 3 个场景判红。
- 让 `Predicate` 收到图输入、节点名或整个 Result 而非该节点输出 → 第 5 个场景判红。
- 用 `Validate()` 一律返回错误来同时满足 R1 与「不静默出错」 → R1 的反向子用例判红。
- 为通过 R2 而先实现 Join 聚合（顺带扩大 diff）：不允许，R2 的图里出现 Join 声明即属超范围。

### 12.4 聚焦命令与完成判据（与本契约前序切片逐字同命令）

```bash
go test ./pkg/hno/graph/... -run P1G -count=1          # RED / GREEN 共用
go test -race ./pkg/hno/graph/... -count=1
go vet ./pkg/hno/graph/... ; gofmt -l pkg/hno/graph
grep -c 'sync\.' $(find pkg/hno/graph -name '*.go' -not -name '*_test.go')   # 必须仍为 0
find pkg/hno/graph -name '*.go' -not -name '*_test.go' | xargs wc -l          # 必须 ≤ 1500
```

R1、R2 各需一条真实非零退出的 RED 回执与一条零退出的 GREEN 回执；R2 的五个场景须在同一测试内以
子用例呈现，且每个子用例可独立失败（不得只在最后做一次整体断言）。

## 13. 切片 17 写回（B9 的咨询点、牙齿归属与实测命中率）

> 追加依据：切片 17 契约 `docs/design/v3-test-scope-p1-graph-slice17.json` 的 `writeBackOwed`（三项），
> 加上该切片 REFACTOR / review 阶段实测出的牙齿归属更正（`docs/design/v3-p1-graph-r17-refactor.md` §3、§7 与
> `docs/design/v3-p1-graph-r17-review-verdict.json`）。
> **方向检查**：本节只把 B9 从「结论正确」收紧为「结论必须在正确的位置得出」，并更正牙齿归属的证据账本；
> 未新增、未删除 §4 的任何行为，未改动 §1 的公共面，未放宽任何一条判据或容差。
> 契约文件按其所代表的那个周期接受后即冻结，因此 §13.3 的归属更正**不回写契约本体**，而是写在这里，
> 作为后续切片读该契约时的生效说明。

### 13.1 B9 的精确措辞：三处咨询点（同时回写母约 §3.3）

「不得伪装成节点业务错误」的判据不在返回的**文本**，而在**问 `ctx` 的位置**。消费者（`Run` 的那个 goroutine）
必须在每一条交出结论的路径上现问一次取消：

| # | 咨询点（最终字节行号） | 这一处非问不可的理由 | 缺掉它时是谁判红（实测） |
|---|---|---|---|
| ① | `select` 的取消分支：消费者 parked 等待、没有任何结论可取时（`scheduler.go:99-100`） | 取消之后节点可能永不返回，`queue` 里也不会有新项；这条分支是「等待中被告知取消」的唯一出口，缺它就不是报错而是挂死 | 拿掉该分支 ⇒ 挂到有界阀为止，由 D5 `:410`「Run 未在有界阀内返回」接住（**接手行是 D5，不是形状最贴近的 D4**） |
| ② | 从 `queue` 取到一项之后、判定它算不算结论之前（`scheduler.go:105`） | 该项已在手上且带节点的业务错误；「调用方已 cancel」与「节点交出错误」这时同时成立，`select` 的随机挑选会随机挑一个说出去，能区分二者的只剩 `ctx` | 把这条守卫改成复用 `consume` 入口的快照 ⇒ D2 `:262` 命中第 17/31/32/37… 轮（`mutants6.mjs` 的 `L_snap`，即契约 λ 的逐字形态）；整条删除在 ③ 还在时走不到差别点（⇒ `R17-GAP-1`） |
| ③ | `dispatch` 循环头、把激活交给节点之前（`scheduler.go:133`） | 消费者忙于派发积压期间不回到 `select`，那段窗口里的取消只有这里看得见 | 只拿掉这一条 ⇒ D1 `:219`+`:223` 把取消报成 `ErrStepLimitExceeded`（`mutants3.mjs` τ）；把咨询整体挪到「pending 排空之后」⇒ 同一行判红（`mutants6.mjs` `R_empty`，即契约 ρ 的形态） |

上述三处合计 4 次取消读取（① 的 `Done` 与 `ctx.Err()` 各一次，②③ 各一次），即结构计数
`ctx_checks_in_scheduler=4`；`L_snap` 那次它降到 3，正是「位置对、读的却是快照」的形状。

配套三条：

- **入口读一次快照、后续复用快照，不算咨询。** `mutants6.mjs` 的 `L_snap` 把 ②③ 两条守卫的位置保持原样、
  只把读取换成复用 `consume` 入口那份快照，D1 与 D2 同时判红。
- **反向同样有牙齿：未取消时不得归一化成取消。** 三种写坏法各有实测落点 —— 一切结论都交 `ctx.Err()`
  （`mutants.mjs` mu）命中 D6；把取消包装进节点原错误、两个哨兵都留（`mutants2.mjs` nu）命中 D6
  `:468`「没取消却被归一化成取消」；未取消时把步数耗尽直接报成取消（`mutants2.mjs` xi）命中 D6
  `:500`+`:503`。归一化只能发生在取消真实成立时。
- **次序不是判据，「在交给节点之前问到」才是。** ο（`mutants.mjs`：预算判据抢在取消判据之前，两条仍都在
  `spawn` 之前）单独跑判绿 —— 同一次循环里谁都还没拿到激活，次序在公共面上不可观察；
  σ（取消后把上限守卫视为失效、继续放出积压）单独跑也判绿 —— 取消一旦成立，③ 先交出结论，
  容量守卫那一步根本执行不到。两条都属等价变异，它给出的结论是 ③ 的牙齿位置在「spawn 之前」，
  不在「预算判据之前」。`dispatch` 上方注释原先把后者说成有牙齿的取舍，属过度声明，已按此改写。

### 13.2 命中率：把早期数字换成绑定夹具上的实测数字

| 编号 | 形状 | 实测（改动前字节 `f2d7fb4b…`） | 用途 |
|---|---|---|---|
| M-4 早期 | 票面 :68 的字面形状（节点阻塞期间 cancel） | 300/300 交出 `context.Canceled` | 证明 §13.1 的三咨询点**不是**由票面原句单独逼出来的：缺口在竞争窗口，不在直白路径 |
| D1 窗口 | 取消落在派发窗口内 | 20/20 确定性交出 `ErrStepLimitExceeded`（`-count=20 -race`），前提自证 0/20 失败 | ①③ 的确定性判据，不依赖运气 |
| D2 竞争 | 消费者 parked、队列里已有一条业务错误 | 35/600 轮 ≈ 5.8%/轮命中「业务错误」出口 ⇒ 跨轮 0 容忍下 20 次执行约 17 次判红 | 竞争行的取样轮数由此定为 100 而不是 30（契约禁止的是减少轮数或把容差写成百分比） |
| M-5 | 取消之后仍在启动的激活数 | 等到首节点后取消：平均 1999.0、最大 1999（succ=2000）；Run 出口 canceled 34 / business 6 | B9 与 B8 的交叉面：取消之后上限不再保护任何东西，故 ③ 必须早于任何交接 |
| M-7 | 极早取消的出口 | plain 26/60、cap 32/60 交出成功 Result；低预算下 41/60 交出 `ErrStepLimitExceeded` | §13.1 末条与「成功路径不咨询」这条取舍的量化根据 |

### 13.3 牙齿归属更正（四处；读切片 17 契约 D2/D4/D5 时以本节为准）

> 希腊字母在本轮的两份账本里**不同义**：契约按字母点名的形态与本轮自建矩阵的同号形态有 5 条不一致，
> 逐条对照表在 `docs/design/v3-p1-graph-r17-refactor.md` §7。因此下面一律以**脚本键 + 形态描述**为准，
> 字母只作为契约原文的引用。

1. **D2.teethMutant**：契约原文点名的形态是「取到 `item.err` 后原样 return、不看 ctx」。
   按该形态单跑（`mutants4.mjs` 的 `chi`，`-count=5`＝500 轮）**判绿**：派发侧守卫 ③ 还在时，
   消费者会先在循环头交出取消，走不到这条守卫 —— 不是漏检，而是该形态在 ③ 存在的前提下不可达。
   D2 迄今唯一的 solo 杀手是契约 λ 的逐字形态（快照 ctx，`mutants6.mjs` 的 `L_snap`）。
   ⇒ 生效归属 **D2 ← { 两条守卫全拿掉（`mutants.mjs` kappa）, 快照 ctx（`L_snap`） }**；
   「把 ② 整条删除」这一形态要可达，缺一个 parked-window 夹具，登记为 `R17-GAP-1`
   （契约原本把 D2 的牙齿写成不可达形态，本轮不再把它记作「D2 无牙齿」）。
2. **D5.teethMutant**：契约指名 σ（取消后绕过上限继续放出积压）与 ρ（只在 pending 空时咨询）。
   σ 在最终字节上单跑 `exit 0`、6 行全绿 ⇒ 等价变异（取消成立时 ③ 先交出，容量守卫执行不到，见 §13.1 末条）；
   它与复合形态 π（`mutants3.mjs`）一起证明 `:419`「取消停在派发之前」那行**没有任何变异可达** ⇒ `R17-GAP-2`。
   ρ 的逐字形态（`mutants6.mjs` 的 `R_empty`）判红的是 **D1** `:219`/`:223`，D5 判绿。
   ⇒ 生效归属 **D5 ← { 拿掉 `select` 取消分支（`:410` 未在有界阀内返回）, 取消被归一化成成功收敛
   （`:413`/`:416` 仍交出 Result） }**；`:419` 那行**保留**（删行等于放宽），但不计入牙齿账。
3. **契约 ξ（用错 ctx 实例）不可观察**：守卫改读自己派生的子 ctx（`mutants6.mjs` 的 `X_child`）判绿 ——
   取消沿父子链传播，公共面上与正确实现无法区分。⇒ 移入观测上限登记（与本轮 σ 同一处置），
   真正断链的写法（节点收到 `context.Background()`）等价于两条守卫全拿掉。
4. **契约指定的杀手行与实测接手行不一致的两处**，一并生效：拿掉 `select` 取消分支由 **D5** 接手（契约写 D4）；
   复合形态 π 由 **D1** 接手（契约写 D5 的 `:419`）。**D3 没有任何 solo 杀手**，它是契约登记的
   green-at-red 护栏行 —— 逐行牙齿强弱以 §13.1 与 `…-refactor.md` §2/§7 为准，不以「该行存在」为准。

### 13.4 母约 §3.3 的同步（已改）

- 「取消」段改为三咨询点 + 快照反例 + 次序非判据；删除「统一归一化」这一越界措辞。
- 原稿的 `cancelAll()` / `cancels` map **未实现，且当前形状下不需要**：节点拿到的就是调用方那一份 `ctx`
  （`runNode` 直接传 `s.ctx`），取消由 ctx 树自动传播。它们只在节点改拿**派生** ctx 时成为必需，
  即 P2 G3 的 Timeout/Retry —— 已把这条依赖写进 §3.3，避免下一个读者按字面去找一个不存在的机制。
- 成功路径（`running == 0`）**不**咨询取消，且这条取舍目前不带断言：M-4 形状下量不到该窗口（0/300），
  而 M-7 交出成功 Result 的那些轮与「图确实先收敛」在公共面上不可区分。是否也必须报取消属未裁决项
  （`S17-SPEC-3` / `R17-UNADJ-1`），按未裁决登记，不借「统一」二字写成已决。

### 13.5 证据与可重放性

- 回执（绑定命令逐字为 `go test ./pkg/hno/graph -run TestP1R17_ -count=1 -race -v`，cwd 仓库根）：
  RED `receipt:276bffa9-133a-4002-9963-2f2c6e42f669`、`receipt:8d1faf62-524a-477b-bde0-2a080db01bc2`、
  `receipt:8851e9e8-b2d2-4a21-955d-9ee09e96600a`（均 exit 1）；GREEN `receipt:88e2073c-dc8d-43dc-a056-6585cb6658b8`
  与加强轮 `receipt:4a826a3b-4ffc-4c03-9c28-4d213942b877`；REFACTOR `receipt:6a9536c7-da6d-4585-9605-731505347f78`。
- 最终字节：`pkg/hno/graph/scheduler.go 0ce6c3b332ddc768c1c5a10d6c1591fc88d67a2a`、
  `pkg/hno/graph/p1r17_cancel_normalization_test.go cbc8b692a84ed5ce23b461a8ed40b9ba3870d386`；
  `graph.go` 本片零改动（`dca23943fd9cade4766ceeffe43b47477b815a61`）。
- 变异留档：`/tmp/p1-b9/mutants{,2,3,4,5,6,7}.mjs` + 同名 `.out.txt`，全部 `restoredIdentical=true`，
  跑后 hash 复核未变；κ/μ/ο/σ 一批的原始输出此前只在会话记录里，`mutants7.out.txt` 是其在最终字节上的留档复跑。
- 披露：`rex-harness receipt` 只存 stdout/stderr 的 sha256，正文由同命令的另一次实跑抄录；
  本节矩阵与测试差异审查均为**作者自查**，未派发独立审查子代理（用户此前已拒绝对 P1 派发）。
- 未闭环：结构计数脚本 `struct.sh` 仍在 `/tmp`（`S17-STD-5`），入库前 §13.2 的 M-5/M-7 与零锁/LOC 判据无法由他人重放；
  `golangci-lint` 未运行。

## 14. 切片 18 写回（B11 的「可归因」三要素、牙齿归属与观察上限）

> 追加依据：切片 18 契约 `docs/design/v3-test-scope-p1-graph-slice18.json` 的 `writeBackOwed`，
> 加上该切片 RED / GREEN / REFACTOR 的实测（`docs/design/v3-red-observation-p1r18.md`、
> `docs/design/v3-p1-graph-r18-green.md`、`docs/design/v3-p1-graph-r18-refactor.md`）与 review 结论
> （`docs/design/v3-p1-graph-r18-review-verdict.json` 的 `S18-SPEC-1`）。
> **方向检查**：本节只把 B11 后半句从「可判定」收紧为「可归因」，并登记两处观察上限与一条未裁决项；
> 未新增、未删除 §4 的任何行为，未放宽任何一条判据，未把 §3 的非目标（`InputSchema`/`OutputSchema`）
> 拉回范围内。契约文件按其所代表的那个周期接受后即冻结，因此 §14.3 的归属更正**不回写契约本体**，
> 写在这里作为后续切片读该契约时的生效说明。

### 14.1 「可判定」→「可归因」：三要素与它们各自的非问不可

票面原句只要求「可判定错误而非 panic」。基线实现（`graph.go dca23943…` 时代 `:495`）交出的正是
`errors.New("graph: input type mismatch")` —— 它**已经**是可判定的：非 panic、非 nil、错误文本稳定。
所以按原句，B11 后半句在切片 18 之前就是绿的，`pkg/hno/graph` 全仓 grep 也证实 `Typed` 从未取过红。
原句缺的是**可定位性**：图上一旦有多个 `Typed` 节点，同一串常量文案无法告诉调用方坏的是哪一个节点、
期望什么类型、实际收到什么类型。§11.1 已确立同类先例（错误文本须包含 offending 节点名，不要求新导出哨兵），
本节按该先例把判据写成三要素，缺一即判红：

| # | 要素 | 落点（最终字节行号） | 缺掉它时是谁判红（实测，`/tmp/p1-r18/mutants.mjs`） |
|---|---|---|---|
| ① | 引号边界的 offending 节点名 | `graph.go:502` 的 `node %q:` | `zeta`（无名版）打红 D2/D4/D5；`eta`（名字槽被类型名顶掉）同样打红 D2/D4/D5 |
| ② | 期望类型与实收类型**分列且不同词** | `:498` 包装期算 `expected`、`:502` 两个独立槽位 | `beta`（两槽同词，即只报期望）打红 D3/D5；`gamma`（只有类型词、无 expected/got 标签）打红 D3/D5 |
| ③ | `nil` 实收渲染为「无值」而非格式化噪声 | `:509-513` `actualTypeLabel` 的 nil 分支 | `delta`（`got` 槽直接打 `reflect.TypeOf(in)`）由 **D5 独杀** |

配套三条判据同源于实测：

- **归因只能落在真正不符的那个节点上。** 三节点链 `seed→a→b`（`b` 的 `TIn` 与前驱输出不符）里，
  错误文本须含 `"b"` 且不得含 `"a"`、`"seed"`（`p1r18_typed_test.go:211/:214/:217`）。
  `theta`（包级共享状态，总是点名第一个被构造的 `Typed` 节点）实测打红 D2/D4/D5。
- **类型不符的结论不得与相邻哨兵混同。** D7（`:304`）三个子测试分别锁住：不被报成
  `ErrStepLimitExceeded`、在 `WithStepLimit(1)` 下不被步数预算顶掉、不被切片 17 的 B9 归一化吃掉。
  `epsilon`（包上 `%w context.Canceled`）由 **D7 独杀** —— 这与 §13.1 的反向牙齿同族，只是方向相反：
  本行管的是「未取消时不得把类型错误说成取消」。
- **节点自身业务错误必须原样穿过。** D6（`:295`）断言业务错误文本里不出现类型不符的形状；
  `lambda`（业务错误与类型分支共用错误构造器）由 **D6 独杀**，`mu`（丢弃 `TOut`、交 `nil, nil`）打红 D1/D6/D7。

### 14.2 实测基线：改前 / 改后的文本形状

同一夹具在两份字节上各跑一次（非推断，`v3-p1-graph-r18-green.md` 的探针表）：

| 场景 | 基线（`dca23943…`） | 最终（`a8e46a3a…`） |
|---|---|---|
| `Typed("needstring", …)` 收到 `int` | `graph: input type mismatch` | `graph: node "needstring": input type mismatch: expected string, got int` |
| 三节点链里 `b` 不符 | 同上常量（不可定位） | `graph: node "b": input type mismatch: expected int, got string` |
| `TIn=io.Reader` 收到 `nil` | 同上常量 | `graph: node "needreader": input type mismatch: expected io.Reader, got no value` |
| `%s` 直接打 `reflect.TypeOf(nil)` | —— | `%!s(<nil>)`（格式化噪声；实测它**不**匹配 D5 的 got 正则，故 ③ 有牙齿） |

### 14.3 牙齿归属更正与观察上限（读切片 18 契约 D1–D7 时以本节为准）

1. **契约判据 (a)「节点名写死为入口节点 ⇒ D4 红、D2 绿」在 `Typed` 内部不可达**：包装层看不到入口、
   注册序与前驱，可达的「点错名」形态只有包级共享状态（`theta`）与名字槽被类型名顶掉（`eta`），
   两者都同时打红 D2（夹具按文件顺序构造，`theta` 捕获到的是 D1 的 `"len"`）。
   ⇒ 登记 `R18-GAP-1`：**D4 的两条反向断言至今没有 solo 杀手**，其归因能力靠与 D2 同红担保。
   这不是「D4 无牙齿」：`alpha/zeta/eta/theta` 四条都能打红它，只是都不能只打红它。
2. **`kappa` 是等价变异，登记为观察上限**：期望类型打印成指针形式（`*string` 而非 `string`）时整族判绿，
   因为 D3 的正则只钉「标签里出现该类型词」。要堵它就得钉死措辞，而 D2/D3 刻意只取「可定位」这一最小性质。
   与切片 17 把 σ、ξ 移入观察上限的处置一致：**不追加断言去堵**。
3. **`mu` 的首次运行是构建失败，不是杀红**：`declared and not used` 使绑定命令返回 `[build failed]`，
   而它的 exit 同样是 1。本矩阵的判红列只认 `-v` 输出里的 `--- FAIL: TestP1R18_*` 名字，退出码单独列。
4. **B11 前半句（编译期类型安全）不由测试担保**：`Typed` 的签名让 `graph.AddNode(graph.Typed[int,int](…))`
   接到不符类型时在**编译期**失败，因此它在票面 §1 的运行期公共面上不可观察。该保证由泛型签名本身提供，
   本节把它写成文字判据而非新增一个「期望编译失败」的测试（后者需要 `go build` 级夹具，超出本片缝）。

生效归属（每条验收行至少一次被证伪）：D1 ← {mu}；D2 ← {alpha, zeta, eta, theta}；D3 ← {alpha, beta, gamma}；
D4 ← {alpha, zeta, eta, theta}；D5 ← {alpha, beta, gamma, delta, zeta, eta, theta}；D6 ← {lambda, mu}；
D7 ← {epsilon, mu}。`kappa` 无归属（等价变异）。

### 14.4 未裁决项（登记，不借写回变成已决）

- `R18-Q1`：三要素目前**寄生在错误文本**上。若要把归因升级为不依赖文本匹配的稳定契约，需一个导出小类型
  或哨兵（`exportedSymbols` 24→25），那会同时改 §1 的公共面与 D8 的结构计数。本片按契约
  `explicitNonGoals` 选择「不加新导出」，故此项留给负责人。
- `R18-Q2`：`Typed(name, nil)` 的节点今天要到运行期才暴露（由切片 14 的 panic 屏障兜住），
  是否在 `Validate()` 里升级为构建期拒绝未裁决。

### 14.5 证据与可重放性

- 回执（绑定命令逐字为 `go test ./pkg/hno/graph -run TestP1R18_ -count=1 -race -v`，cwd 仓库根）：
  RED `receipt:b1222ea4-9a77-42fa-9c5c-9ed5c19df628`（exit 1，`-count=20` 下 D2 前提自证 20/20）与
  阶段级 `receipt:3f831afa-731c-4021-b4db-1f0e4efeb5dd`；GREEN `receipt:8674cf3f-6981-40de-b2dd-bd5297c07b22`（最终字节复跑）；
  REFACTOR `receipt:e110a8db-4aa2-4567-bc41-b6112383b15b`。
- 最终字节：`pkg/hno/graph/graph.go a8e46a3a2f0baaa699ae60f26498c6f5c1c851cb`、
  `pkg/hno/graph/p1r18_typed_test.go 10bce61bd70f2b78a968d585cc81919e8fe68293`；
  `scheduler.go` 本片零改动（`0ce6c3b332ddc768c1c5a10d6c1591fc88d67a2a`）。
- 覆盖率实测：`Typed` 0.0% → **100.0%**，包总覆盖 99.1%。结构计数：
  `loc_nonblank_nontest 649 → 663`（+14，预算 ≤60）、`exportedSymbols` 保持 24、零锁计数保持 0。
- 变异留档：`/tmp/p1-r18/mutants.mjs` + 逐条 `mutants-<id>.out.txt`，11 条全部 `restored=true`
  （`git hash-object --no-filters` 复核回 `a8e46a3a…`）。
- 披露：本片的矩阵与测试差异审查均为**作者自查**，未派发独立审查子代理（用户此前已拒绝对 P1 派发）；
  `redCandidate` 首次提交时误用 `sh -c` 包装，回执记录的 `executable` 是 `sh` 而非 `go`，
  被宿主按「回执与声明场景命令不匹配」拒收（`receipt:a7b51ac5-3097-4d98-b162-18e4d9dbf309`），改为本节逐字形态后重取。
- 未闭环：`golangci-lint` 未运行；`struct.sh` 与 `mutants.mjs` 仍在 `/tmp`（同 `S17-STD-5`），
  入库前 §14.5 的结构计数与矩阵无法由他人重放。

## 15. 切片 19 写回（B2 的三要素、牙齿归属、审查要求落成文字的限制）

工作项 `work-p1-graph-r19-join-barrier` / activation `48130ef2-0ff7-4e14-87dc-b826a8acb970`。
§4 表的 B2 行已按本节改写；读切片 19 契约 D1–D7 与
`docs/design/v3-p1-graph-r19-green.md`、`docs/design/v3-p1-graph-r19-refactor.md` 时以本节为准。
本片把 Join 从「构建期被拒绝的声明」变成「运行期真正路由的屏障」：零新导出符号、零锁、零新实现文件，
`graph.go` 与 `scheduler.go` 之外只动了一格被契约授权退役的既有测试。
母约同步已做：`v3-platform.md` §3.4 规则 3 标注为已落地并写下三条边界，§3.5 第 6 项标注已交付，
§6 的 map-reduce 示例处登记了与 §3.5 第 6 项的冲突（`R19-Q2`，示例本身未擅改）。

### 15.1 三要素：屏障激活条件、聚合输入形状、凑不齐时的兜底

1. **激活条件**：目标 `to` 等的是 `joinBarriers` 从边表算出的**去重前驱名集合**；前驱每完成一个，
   就把它的输出按前驱名记进该目标的聚合 map，只有键集合覆盖 `waits[to]` 时才把 `to` 激活。
   判据用集合覆盖而不是计数归零：计数器要把同名重复声明与前驱重跑都算对，而集合本身就是
   公共面看到的那个形状（`Result.Value("join")` 收到的即它）。两种被禁止的形态都被实测证伪 ——
   把 Join 当无条件边路由（每个前驱完成都激活一次，m1/m6 判红）；「等固定时长再聚合」
   （§4.1 第 3 条禁止的形态，D3 让 join 节点在入口清点已到达的前驱信号，读到 2 才认，不用 sleep）。
2. **聚合输入形状**：`map[string]any`，键 = 前驱名、值 = 该前驱本次输出；交给激活的是这个 map
   本身，**不是**「最后一个前驱的输出」（m2 因此单独打红 D2）。把键写成汇聚目标名时
   `len(collected)` 永远是 1 ⇒ 屏障永不凑齐，所以 m3 的后果是「整族判红」而不是「输入形状略偏」
   （键控这条性质由 m2 单独担保，见 15.3 第 1 点）。
3. **凑不齐时的兜底（有条件，不是无条件成立）**：图仍在产生激活时，未收敛由 `WithStepLimit`
   的安全阀结束，调用方拿到 `errors.Is(err, graph.ErrStepLimitExceeded)`（D7 实测；m11 打红的正是
   哨兵没被 `%w` 包进错误的那类实现）。若声明过的汇聚前驱始终没跑、且再无待派激活，今天交出的是
   `err==nil` 而 `Output()==nil` 的 Result —— 这是票面 :161-168 fail-closed 精神的缺口，未裁决，
   见 15.5 的 R19-Q1。`AddJoin` 的文档注释已按这条边界改写（写回阶段补，字节见 15.6）。

另两条在最终字节上实测、并刻意不加断言的形状：

- **屏障在前驱重跑时会再次激活**：`entry→{a,b}`、`b→(条件，命中两次)c`、`c→b`、
  `AddJoin(["a","b"],"j")` 的探针给出 `joined=3`、`completed=[a b b b c c entry j j j]`。
  票面没有任何一行要求「一张图上每个屏障至多 fire 一次」，本片不加 `fired` 守卫 ——
  加了就没有任何行担保它的必要性。
- **交给节点的 map 是调度器继续写入的那个对象**：同一探针里三次激活收到的输入地址相同
  （`0xc00018a000`），节点保存引用后在 `Run` 结束读到的是最后一次前驱输出（`b:B3`），
  不是它被激活时看到的那一份。这就是 15.5 的 S19-STD-1，本轮由「读代码看出来的」升级为实测。

### 15.2 牙齿归属（每条子断言至少一次被证伪）

命令 A 是契约绑定命令（`TestP1R19_`），命令 B 是承载退役格与端点次序的 `TestP1G_`。
完整矩阵（13 条变异、逐条 exitA/exitB 与判红 Test 名）见 `docs/design/v3-p1-graph-r19-refactor.md` §2，
执行体 `scripts/mutation/p1r19-join-barrier.mjs` 已入库。

| B2 子断言 | 打红它的变异 | 备注 |
|---|---|---|
| join 在本次 Run 恰好执行一次 | m1、m6、m7、m10 | |
| 输入含两个前驱键、以前驱名为键 | **m2（solo）**、m3、m10 | m2 在命令 A 只打红 D2 |
| a、b 均先于 join 完成 | m1、m3、m6、m7、m10 | 观察手段是节点内信号清点，非 sleep（§4.1 第 1 条） |
| join 的输出继续往下游走并进入投影 | **m8（solo）**、m1、m3、m6、m7、m10 | |
| 合法汇聚图 `Validate()` 为 nil、非法声明仍被点名 | m10 | 反向对照不担保检查次序，见 15.3 第 2 点 |
| 前驱数 < 2 构建期拒绝且可定位（即 B6 第 6 项） | m4、m5、m10 | 同时打红退役后的 R1 格（命令 B） |
| 未收敛由安全阀结束、不静默 | **m11**、m10 | m11 在本片两条命令内 solo，整包不唯一，见 15.3 第 3 点 |

### 15.3 三处更正与三条观察上限（读早期记录时以本节为准）

1. **m3 不是「键名错位」的 solo 杀手**：键写成目标名会让两次前驱完成落进同一个键，屏障永不凑齐，
   因此它同时打红 D1/D3/D4。键控性质的独立牙齿是 m2。
2. **「前驱数检查排在逐条边端点检查之后」这条次序不由 D5 担保**：m9 把两步互换后命令 A 全绿
   （D5 用的声明有两个去重前驱，前驱数检查根本不触发，缺失端点仍由端点检查点名），
   判红的是命令 B 里既有的 `TestP1G_ValidateRejectsDanglingEndpointForEveryEdgeKind`
   （`graph_test.go:411`）。这条次序的担保来自既有测试，不来自本片新写的行。
3. **m11 的杀红跨切片冗余**：整包跑一遍时 `%w → %v` 还会打红 `TestP1R13_*`（4 个）、
   `TestP1R16_CapDoesNotConsumeStepBudget`、`TestP1R17_UncancelledBusinessErrorStillSurfacesVerbatim`。
   哨兵包装因此不只由 D7 担保。
4. **三条观察上限（e1/e2/e3，不追加断言去堵）**：多目标不足前驱时文案点名的稳定性、
   前驱名的去重语义、`feeds` 的声明顺序 —— 没有任何 B2 行能在公共面上同时区分它们，
   而时序本身被 :40、:75 禁止当证据。
5. **两种「数不出来」的汇聚声明在公共面上不留事实**，最终字节上各实测过一句：
   `AddJoin(nil, "join")` 收到的是不可达节点那句（空前驱在边表上一条声明都不留，与
   「注册了但无入边」同形，由既有的可达性检查担保）；
   `AddJoin([]string{"a","a"}, "join")` 收到
   `graph: join target "join" must declare at least 2 distinct predecessors, got 1` —— 去重真实生效，但没有行钉它。

### 15.4 审查要求落成文字的两条限制

- **S19-SPEC-4（含 `edgeJoin` 的环在构建期不断环）**：`alwaysTakenAdjacency`（`graph.go:411`）
  只收无条件边与无条件兜底边，汇聚边按「可断开」处理，所以 `a →(join) j →(join) a` 这类声明
  不被 `rejectUnconditionalCycles` 拒绝，落到步数安全阀。写回字节上的仓库外探针
  （`AddEdge("a","b")` + `AddJoin(["a","b"],"b")` + `AddJoin(["a","b"],"a")`、`WithStepLimit(6)`）
  实测：`Validate()` 为 nil，`Run` 交出
  `graph: step limit 6 reached before the graph converged: graph: step limit exceeded`
  且 `errors.Is(err, ErrStepLimitExceeded)` 为真、`res==nil` —— 是安全阀，不是构建期错误。这是既有判据的自然延伸，本片未改其语义，
  但它意味着「Join 声明可以出现在环里」是当前已接受的公共面行为；要收紧须另起一行判据。
- **S19-SPEC-5（「未收敛由安全阀结束」这句原来写得过宽）**：该结论只在图仍在产生激活时成立，
  15.1 第 3 点的另一形状今天是静默的 `err==nil`。`AddJoin` 的文档注释已按此限定（写回阶段的
  注释改动，不动行为；对应的裁决仍是 R19-Q1，不在本片偷偷改掉语义）。

### 15.5 未裁决项（登记，不借写回变成已决）

- `R19-Q1`：声明的汇聚前驱本次没跑、且再无待派激活时，`Run` 交出 `err==nil` 而
  `Output()==nil`。实测：`err=<nil> resNil=false output=<nil> completed=[a entry] Value(join)=<nil>(ok=false)`。
  需要一次 `rex-test-design` 裁决：构建期要求「声明的每个 join 前驱都可从入口到达」，
  还是运行期把「屏障永不凑齐」归一化成错误。本片按契约不加断言。
- `R19-Q2`：母约 `docs/design/v3-platform.md:341` 的 map-reduce 示例写的是
  `AddJoin([]string{"summarize"}, "reduce")` —— 单个前驱名、靠 `Send` 在运行时扇出多次来聚合；
  按票面 :65「Join 前驱数 < 2 构建期拒绝」，这条示例今天会被本片新加的 `rejectUnderfedJoinTargets`
  拒绝。构建期唯一可数的是「声明的去重前驱节点数」，两者不可能同时成立。要么改示例，
  要么在 `Send` 那一片引入第二种屏障（同名前驱的多次激活也计入汇聚）。
- `S19-STD-1`：运行期交给激活的 map 与调度器继续写入的累加器是同一个对象
  （`scheduler.go:220` 写、:224 交），15.1 末条已把它实测出来。修法是交出快照
  （`maps.Clone` 或重建 map），但这需要一次诚实 RED —— 它是新的用户可观察行为，
  不属本片契约行。已同步登记到 `docs/design/v3-p1-graph-status.md`。
- `S19-STD-6`：变异执行体已入库但未接进 `make`；`golangci-lint` / `make lint` 在本工作项从未运行。
- 沿用未裁决：`R18-Q1`、`R18-Q2`、`R17-GAP-1/2/3`、`S17-SPEC-3`/`R17-UNADJ-1`、`S18-SPEC-2`
  （HEAD `0387000` 仍只含首片存根，切片 2–19 全部未提交；**无提交授权**）。

### 15.6 证据与可重放性

- 绑定场景命令逐字为 `go test ./pkg/hno/graph -run TestP1R19_ -count=1 -race -v`（cwd 仓库根，
  `receipt -- go test …` 直接取，无 `sh -c` 包装；本片全部 receipt 都是这一条）：
  D1 候选红 `receipt:279aa8da-e3b3-4fc3-9c8b-a31059f1461b`、阶段级 RED
  `receipt:ebd73748-a3ba-4e30-99e1-73ae95452fbb`（exit 1）、GREEN
  `receipt:03c4c832-faed-4e90-bdc7-bfd6788d979d`（exit 0）、REFACTOR
  `receipt:8a55bd2f-41e3-4c6d-bd26-654fd539b359`（exit 0）、写回字节复跑
  `receipt:73a2c954-adb0-4b83-8c3c-e67849d23536`（exit 0）。
- 最终字节（含 15.4 要求的 `AddJoin` 注释限定）：`pkg/hno/graph/graph.go 4cb363b729cddf7541df8d1b1900c8240dff93f2`、
  `pkg/hno/graph/scheduler.go c878a6c74cad89bbdfc773c21c58952d3944630d`、
  `pkg/hno/graph/graph_test.go 624f19624c6f4fa3c04ee6619e722b03de83336c`、
  `pkg/hno/graph/p1r19_join_barrier_test.go 8b4b7234ce92df2f936a1544cf24545d65ce0a9b`；
  其余 7 份既有测试文件与基线逐字节相同（契约判据 4，`git hash-object --no-filters` 前 8 位：
  p1r13 `bc4ab8e7`、p1r14 `0ed607db`、p1r15 `3ea06d27`、p1r15b `0285ab35`、p1r16 `6934a4f2`、
  p1r17 `cbc8b692`、p1r18 `10bce61b`）。`graph_test.go` 自身的改动只有被契约授权退役的那一格。
  写回前的字节是 `18002942851225ac6c27a0a40b4e24086a6c144f`（REFACTOR 文档 §1 记的那一列），
  与最终字节的差异只有 `AddJoin` 的文档注释。
- 结构计数（命令逐字可重放）：
  `find pkg/hno/graph -name '*.go' -not -name '*_test.go' | xargs wc -l | tail -1` → 838 total
  （票面 :94 的 ≤1500 仍成立；GREEN 阶段是 844，REFACTOR 收到 834，注释补完 838）；
  `grep -o 'sync\.' pkg/hno/graph/graph.go pkg/hno/graph/scheduler.go | wc -l` → 0（B12）；
  `go doc -all ./pkg/hno/graph | grep -cE '^(func|type|var|const) '` → 24（契约判据 6：不放宽，命令逐字取契约）。
- 写回字节上门禁实测（契约判据 5、7）：`go build ./...` exit 0、
  `go test ./pkg/hno/graph/... -count=1` → `ok 1.972s` exit 0、
  `go test -race ./pkg/hno/graph/... -count=1` → `ok 2.720s` exit 0、
  `go vet ./pkg/hno/graph/...` exit 0、`gofmt -l pkg/hno/graph scripts` 无输出、
  `go test ./pkg/hno/agent/... ./pkg/hno/runner/... ./internal/session/contract/... -count=1`
  三包全 `ok` exit 0、`node scripts/mutation/p1r19-join-barrier.mjs --check` 报 13 条锚点全部唯一命中。
- 15.1 末两条形状与 15.5 的 `R19-Q1` 都取自仓库外的独立 module 探针
  （`/tmp/r19q1`，经 `replace` 引本包、只调公共 API），探针本身不入库、其数字不入判据。
- 披露：本片的矩阵、测试差异审查与写回均为**作者自查**，未派发独立审查子代理
  （用户此前已拒绝对 P1 派发），独立性因此降级 —— 见
  `docs/design/v3-p1-graph-r19-review-verdict.json` 的 `independenceDisclosure`。
  另两处过程披露：D1 初稿对未关闭的带缓冲通道 `for range`，会挂住整条命令，改为有界非阻塞 drain
  后断言 `entered != 1` 未变（基础设施修复，不是放宽期望，见 green 文档 §4）；
  15.1 的 `joined` 第一次读到 1 是探针自己没加锁导致的丢计数，加锁后为 3。

## 16. B6 第 5 项的裁决记录（切片 20 设计期实测 + 负责人选 OPT-A）

`work-p1-graph-r20-dead-end-sink` 在 `rex-test-design` 的 `decide-testability` 阶段提交 `blocked`，
宿主返回 `outcome: replan`、`command: null` —— 本片没有进入 `rex-tdd`，没有新增测试文件，也没有实现代码。
阻塞的理由不是「找不到红」，而是「任何诚实的红都要求先修改票面或修改既有担保」。负责人 2026-09-27 选
**OPT-A：作废该判据**，本节按此登记裁决与支撑它的三个数字。

### 16.1 三条实测（仓库字节全程未动）

副本实验在 `/tmp/sinkexp`：把 `pkg/hno/graph` 的 11 个文件与全部测试原样拷进一个
`module github.com/rexleimo/agno-go` 的空壳 module，只在副本的 `graph.go` 里加候选检查并接进
`Validate` 末位。副本命令统一为 `go -C /tmp/sinkexp test ./pkg/hno/graph -count=1`
（回执里 executable=`go`、args 头两格 `-C /tmp/sinkexp`、cwd 记录为仓库根）。

| # | 副本状态 | 观察 | 回执 |
|---|---|---|---|
| 基线 | 与仓库逐字节相同 | `ok`，exit 0 | `receipt:7f442439-8bce-4f23-b44f-a5e6d0ca0375` |
| 字面判据 | 「注册了、没有任何出边、不是输出 → 判红」 | exit 1，`-v` 里 **12 个顶层 Test 判红**（放在不可达检查之前则为 16 个） | `receipt:dd11883e-75ad-4d88-844e-2139f94a2f6c` |
| 最窄非空限定版 | 只判「无条件边的目标 + 无出边 + 非输出」 | exit 1，**仍 10 个顶层 Test 判红** | `receipt:f594bc89-77f3-4ca3-bcb2-16fdc621d5ec` |

今天的公共面形状取自仓库外探针 `/tmp/r19q1/sink`（`replace` 引本包、只调公共 API，跑在切片 19 写回字节上）：
`entry→sink` + `SetOutput("entry")` 收到 `Validate() = <nil>` 且 `Run` 静默成功交出 `output=x`、
`completed=[entry sink]`；两条反向对照（`entry→{a,b}`+`AddJoin(["a","b"],"j")`+`SetOutput("j")`、
`entry→out`+`SetOutput("out")`）同样为 `<nil>`。缺口是真的，只是它不能按字面判据补。

### 16.2 为什么限定版也不是出路

限定版经副本 `cmd/probe` 实测**同时**做到两件事：`entry→sink` 收到
`graph: dead-end node "sink" has no outgoing edge and is not the output`（意外死端抓住了），
条件边两个终点 `b`/`c` 的图收到 `Validate() = <nil>`（分支终点放行了）。但它仍打红 10 个既有 Test，
且判红理由不是分支终点，而是**无条件扇出的终点**：

- `graph_test.go:236` 的断言原文「按 §3.4 合法的路由图被 Validate 拒绝」，判红的子用例名就是
  `TestP1G_ConditionalAndDefaultRouting/unconditional-fanout`，被点名的节点是 `"b"`；
- `p1r13_step_limit_test.go:135` 写的是「合法的扇出图被构建期拒绝」，被点名的是 `"x"`；
- `p1r14_panic_barrier_test.go:149` 的失败形状不同但同源：夹具里的 `bad` 先被判成死端，
  调用方拿到构建期文案而不是运行期 panic 文案；`p1r17_cancel_normalization_test.go:475` 的夹具被同样方式拒绝。

也就是说：§3.4 规则 1 按设计就允许「某个节点的输出没有任何消费者」，而这条设计事实被 10 个既有顶层 Test
担保着。要在不放宽它们的前提下判红意外死端，必须先给公共面增加终点声明（OPT-B 的 `graph.END`），
那是一条新的表达力决策；或者降级为告警（OPT-C）。两者都没有被选中，故本项按 OPT-A 作废。

### 16.3 作废之后仍然成立的那半句

判据作废不等于「A 格的静默不再是问题」。今天没有任何一行判据担保「无出边的非输出节点一定会被执行」，
它与 §15.5 的 `R19-Q1`（屏障凑不齐且再无待派激活时 `Run` 交出 `err==nil` 而 `Output()==nil`）是同一族
「数不出结论就不吭声」的形状。若日后要收紧，起点应是引入终点声明（OPT-B），而不是在现有声明形态上再加一条
只能误伤合法图的检查。母约 §3.5 第 5 项已按本节改写。

### 16.4 B6 第 2 项（重复节点名）—— 先前登记的「正面对撞」是走查推断，已被实测推翻

状态文档与本片契约的第一版写过：第 2 项与
`TestP1G_RunIsolatesTopologyFromConcurrentMutation` 的子用例「运行中覆盖同名节点不得改变本次输出来源」
（`graph_test.go:953`）正面对撞、属票面变更。**该判断错了**，错在只读了测试名与断言语义，没有跑。副本
`/tmp/sinkexp` 实测（同样只改副本，实验后已逐文件恢复并与仓库比对）：

- 在 `AddNode` 里记一次 `dupNames`、在 `Validate` 末位加 `rejectDuplicateNodeNames`
  （文案 `graph: node name "b" is declared more than once`）后，
  `go -C /tmp/sinkexp test ./pkg/hno/graph -count=1` → **exit 0、0 个 Test 判红**
  （`receipt:330df6ad-0339-400c-a83e-15363007aeed`）；
- 同一条命令加 `-race` → **exit 0**（`receipt:14aa3bd0-d4ed-497e-9d9f-a2e04406c4f3`）；
- 那个子用例覆盖的是**一次 in-flight Run 的捕获隔离**：`Run` 已经越过 `Validate` 之后，另一个 goroutine
  才用同名 `AddNode` 覆盖 `b`，断言的是「本次输出来源不变」。它与「一张新图重复注册同名节点应在构建期被拒」
  不是同一条命题，两者可以共存。
- 爆炸半径也量过：`grep -rn "AddNode(" --include=*.go` 在 `pkg/hno/graph` 之外**零命中**，
  引擎尚未接进 workflow/agent，因此该检查不会打破任何仓库内既有调用方。

结论：第 2 项**不需要票面裁决**，它是一片正常的 `behavior-delta` —— 今天 `AddNode("a")` 两次后
`Validate() = <nil>`（静默覆盖），落地后必须是可定位的构建期拒绝，红能真实复现。
仍需在实现前想清楚的一条设计后果（不是对撞，是取舍）：按名字记账会让「跨 Run 故意重注册同名节点来替换实现」
这种用法从此在 `Validate` 上不可通过；副本实测没有任何既有测试依赖它，但这条自由度的取舍应在契约里写明。

### 16.5 过程如实登记（本文件的自纠）

`design-tests` 阶段第一版契约（`v3-test-scope-p1-graph-slice20.json`）在 M2 里写了
`receipt:efa1ac82-…` 与 `receipt:b917dd0a-…`，而 `.rex-harness/receipts/` 对这两个 id `grep -rn` **零命中** ——
是未实际取回执就写下的引用。现已在副本上重跑实验、取真实回执替换（16.1 表中三条），
并把第一版对冲突根因的偏窄描述（写成「条件/兜底分支终点」）改为本节 16.2 的实测结论。
实验结束时副本逐文件恢复并与仓库比对：两边 `git hash-object --no-filters pkg/hno/graph/*.go` 的 md5
同为 `1250071191a7ddb896141a43bdeb3319`，副本复跑 exit 0；仓库工作树未被实验改动。
本节的写回是**纯文档改动**：`graph.go 4cb363b7…`/`scheduler.go c878a6c7…`/`graph_test.go 624f1962…` 逐字节未变，
写回后在最终字节上复测本片绑定场景命令
`go test ./pkg/hno/graph -run TestP1R19_ -count=1 -race -v` → exit 0（`receipt:72de44ae-4bf5-4761-80ac-c0f3978eb1e6`）。
独立性披露沿用：本轮仍是作者自查，未派发独立审查子代理（P1 的独立审查派发此前已被负责人拒绝）。

## 17. 切片 21 写回（B6 第 2 项的三要素、牙齿归属、观察上限，兼一项事故披露）

工作项 `work-p1-graph-r21-duplicate-node-name` / activation `2d96c915-5fc4-48c2-baa6-b27b7c27b29e`。
契约：`docs/design/v3-test-scope-p1-graph-slice21.json`（D1–D5）。
实现字节：`pkg/hno/graph/graph.go`（`4cb363b7…` → `b7790b9f…`，净增 +34 行）；
`scheduler.go c878a6c7…` 与全部 10 份既有测试文件逐字节未动（契约判据 3，逐文件
`git hash-object --no-filters` 复核一致）。RED `receipt:5964bbf8…`（exit 1，D1/D2/D5 判红、
D3/D4 反向对照绿）→ GREEN `receipt:aaea7171…`（exit 0，5 个 Test 全 `--- PASS`，`-race`）；
事件后复核 `receipt:bd24a87d…`（exit 0，字节与 GREEN 相同）。绑定场景命令逐字：
`go test ./pkg/hno/graph -run TestP1R21_ -count=1 -race -v`。

### 17.1 判据三要素与实现落点

- **D1/D2/D5（正向拒绝）**：`AddNode` 命中已有名字时记 `dupNames`（声明序账本），
  `Validate` 末位 `rejectDuplicateNodeNames` 点名账本首个，文案
  `graph: node name "x" is declared more than once`；`Run` 第一行就是 `Validate`，错误原样透传、
  `res==nil`，不跑半张图。map 覆盖语义保留：in-flight Run 的捕获隔离（`graph_test.go:953`）
  依赖它，且该子用例在最终字节上照旧通过。
- **D3/D4（反向对照）**：全程不同名的合法图 `Validate()==nil` 且 `Run` 照常；`Run` 之后注册
  **新名字**的节点对下一次 `Run` 可见 —— 检查没有退化成「不许注册」或「终身冻结」。

### 17.2 牙齿归属（`scripts/mutation/p1r21-duplicate-node.mjs`，仓库内可复现）

| 变异 | 形态 | 结果 |
|---|---|---|
| m1 | 检查整段失效（末位改回环检查） | exitA=1，杀红 D1/D2/D5 |
| m2 | 名字槽位换成空串（文案不点名） | exitA=1，杀红 D1/D5 |
| m3 | 记账退化成「每次注册都记」 | exitA=1，杀红 D3/D4/D5；命令 B 另有 9 个 `TestP1G_` 判红 |
| m4 | 检查挪到入口检查之前（次序变异） | **全绿 —— 等价变异，登记为观察上限**（见 17.3） |

m2 的第一版锚会把 Go 字符串字面量顶破（无格式指令的常量文案），`go test` 内置的 vet printf
检查先于测试拦下它 —— 那不是测试杀红，已按「保持 `%q` 指令、传空名」重做。

### 17.3 观察上限（登记，不用断言去堵）

1. **次序不被 D1–D5 钉住**（m4 全绿）：「末位」是 `Validate` 文档固定的设计声明，不是测试事实；
   D1–D5 的图除重复名外全部成立，检查放前放后它们都绿。要让次序成为判据需要一张
   「既重复名又有其他非法声明」的图并断言报的是重复名 —— 本片契约没有这一行，不追加。
2. 多个重复名时点哪一个、一次报几个：账本是声明序，公共面只担保「点名的那格确实重复」。
3. 「后注册者赢」的旧形状（M1 形2 的 `Output() = E2`）在落地后不可达，无行能同时观察它与拒绝它。

### 17.4 过程如实登记（两件事）

1. **首次 RED 踩中夹具缺陷**：D1 第一版多注册了一个未接线的 `out` 节点，既有的
   `unreachable node "out"` 文案替重复名断言顶了名（`receipt:90d305b4…` 显示该 Test 假绿）。
   已按契约 D1 行指定的 M1 形2 图修正夹具后重取 RED（`receipt:5964bbf8…`）。
   与 §6 的教训同源：**断言被谁顶名，要靠看失败文案而不是靠推断。**
2. **一次 `git checkout` 事故与逐字节恢复**：变异阶段中途，`git checkout pkg/hno/graph/graph.go`
   把本文件（含切片 2–21 全部未提交字节）回退到了 HEAD 存根（`02122d9d…`）。恢复来源是
   切片 20 实验留下的副本 `/tmp/sinkexp/pkg/hno/graph/graph.go`（`git hash-object` 与
   切片 20 收口基线 `4cb363b7…` 全同），恢复基线后重放本片三处 GREEN 编辑，
   `git hash-object` 复核 `b7790b9f…` 与事故前 GREEN 字节**逐字节相同**，绑定命令复跑
   exit 0（`receipt:bd24a87d…`）。披露两点：其一，切片 2–21 的未提交状态仍无提交授权
   （`S18-SPEC-2` 沿用），本事故把「单拷贝工作树」的脆弱性从口头风险变成了实测事故；
   其二，这次是 `/tmp` 副本救回了仓库 —— `S17-STD-5` 的教训（实验fixture不入库的代价）反过来了：
   **仓库没有备份时，入库的实验副本就是唯一备份**。
独立性披露沿用：本轮仍是作者自查，未派发独立审查子代理（P1 的独立审查派发此前已被负责人拒绝）。
