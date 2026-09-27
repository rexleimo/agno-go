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
| B2 | 并行扇出 + Join 汇聚屏障 | `entry→{a,b}→join`：join 恰好执行一次；`join` 的输入含 a、b 两者的输出（按前驱名键控）；a、b 均先于 join 完成 |
| B3 | 条件路由 + Default 兜底 | 谓词命中 → 该目标执行且 `Completed()` 不含兜底目标；谓词皆不命中 → 仅兜底目标执行；两条都不命中的非兜底节点不得执行 |
| B4 | 合法环收敛 + stepLimit 安全阀 | 条件边计数环在 N 次后走 false 分支收敛，`Run` 返回 nil error；把 `WithStepLimit` 设为小于所需步数 → `errors.Is(err, graph.ErrStepLimitExceeded)` 且 `Output()` 不可用 |
| B5 | 无条件边成环 = 构建期错误 | `AddEdge` 造出 A→B→A：`Validate()` 返回非 nil，且该 error 可判定为「无条件环」类；`Run` 不得启动它 |
| B6 | 其余 7 项构建期校验 | 悬空边、重复节点名、无入口/入口不存在、不可达节点、无出口（非输出节点无出边）、Join 前驱数 < 2 → `Validate()` 非 nil；合法图 → `Validate()` 为 nil |
| B7 | stepLimit 缺失告警（判据 8） | 含**合法条件环**且未显式 `WithStepLimit` 的图：`Validate()` 为 nil、`Run` 正常、`Warnings()` 含该告警；显式设置后 `Warnings()` 不含它 |
| B8 | 并发上限 + FIFO 等待队列 | `WithMaxConcurrency(1)` 且 a、b 并行：a、b 的执行区间**不重叠**（用节点内 gate 同步，不用 sleep 定序）；激活顺序为声明顺序，即第二个节点观测到的前序完成集合包含第一个 |
| B9 | 取消归一化 | 节点阻塞期间 cancel `ctx`：`Run` 返回错误且取消语义为 `context.Canceled`（`errors.Is`），不得伪装成节点业务错误 |
| B10 | panic 转错误 | 节点内 `panic("boom")`：`Run` 返回非 nil error，进程不崩，测试不中断 |
| B11 | 泛型包装 | `graph.Typed` 包装的节点接到 `TIn`、返回 `TOut`；对错误类型输入返回可判定错误而非 panic |
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
