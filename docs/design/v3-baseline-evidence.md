# v3.0 基线证据（实测，非估计）

> 所有数字来自本仓库与参考仓库的源码实测命令。采集时间：本次规划周期。
> 用途：作为 `v3-delivery-ticket.md` 中 `evidenceRefs` 的可核验依据。

## 0. 溯源（review 修正后补充）

本文所有结论对应以下提交，重新核验时须比对 commit：

| 仓库 | HEAD | 日期 | 位置 |
|---|---|---|---|
| agno-go（被测） | `17e86c5` | — | `/Users/rex/codes/agno-go` |
| adk-go（参考） | `12f7cab` | 2026-09-25 | `/tmp/fwcmp/adk-go` |
| langgraph（参考） | `7daa3ab` | 2026-09-23 | `/tmp/fwcmp/langgraph` |

⚠️ **参考仓库位于 `/tmp`，重启后可能消失**。若核实不到本文 §4/§6 的参考仓库数字，请重新拉取并比对上述 commit：

```bash
git clone --depth 1 https://github.com/google/adk-go.git     # 期望 HEAD = 12f7cab
git clone --depth 1 https://github.com/langchain-ai/langgraph.git  # 期望 HEAD = 7daa3ab
```

本仓库（§1/§2/§3/§7）的数字不依赖 `/tmp`，随时可复现。

## 1. 本仓库基线

| 指标 | 实测值 | 采集命令 |
|---|---|---|
| 当前版本 | 1.2.9 | `head CHANGELOG.md` |
| 非测试 LOC | 43,684 | `find . -name '*.go' -not -name '*_test.go' -not -path './website/*' \| xargs cat \| wc -l` |
| 测试 LOC | 34,554 | `find . -name '*_test.go' -not -path './website/*' \| xargs cat \| wc -l` |
| 模型 provider 数 | 17 | `ls -d pkg/hno/models/*/ \| wc -l` |
| 契约测试数 | 9 | `grep -c '^func Test' internal/session/contract/contract_test.go` |
| `runner` 包外部引用数 | **0** | `grep -rn 'hno/runner' --include='*.go' . \| grep -v '^./pkg/hno/runner/' \| wc -l` |
| `pkg/hno/runner` 测试包名 | `package runner`（同包测试） | `grep -h '^package' pkg/hno/runner/*_test.go \| sort -u` |
| `pkg/hno/skills` | **398** 非测试 / **684** 含测试 | `find pkg/hno/skills -name '*.go' -not -name '*_test.go' \| xargs cat \| wc -l` |
| `pkg/hno/graph` 是否存在 | **不存在**（v3 新建） | `ls pkg/hno/graph` |

> **口径说明**：本文所有 LOC 数字除明确标注「含测试」外，**一律为非测试（不含 `_test.go`）**。
> `pkg/hno/skills` 的 684 是含测试口径，直接 `wc -l pkg/hno/skills/*.go` 即可得到；两者都对，口径不同。

## 2. 循环实现重复（红线 2 违规证据）

| # | 位置 | 形式 | 归属 |
|---|---|---|---|
| 1 | `pkg/hno/agent/run.go:60` | `for loopCount < a.MaxLoops {` | agent 同步 tool 循环 |
| 2 | `pkg/hno/agent/stream.go:259` | `for {` | agent 流式 tool 循环 |
| 3 | `pkg/hno/runner/runner.go:180` | `for {` | 被设计文档定义为唯一真相源，**实际零引用（死代码）** |
| 4 | `pkg/hno/run/loop.go` | `run.Loop` | **仅** `pkg/hno/team/team.go:300` 被实际构造 |

采集：
```bash
grep -n "for loopCount" pkg/hno/agent/run.go          # → 60
grep -n "for {" pkg/hno/agent/stream.go | head -1      # → 259
grep -n "for {" pkg/hno/runner/runner.go | head -1     # → 180
grep -n "loop := &run.Loop{" pkg/hno/team/team.go     # → 300（实际构造点）
```

> **精度说明（review 修正）**：`grep -n "run.Loop" pkg/hno/team/team.go | head -1` 会返回 **295**，
> 那是一行**注释**（`// Adapt the team to the shared run.Loop kernel`），不是调用点。
> 真实构造点是 **300**（`loop := &run.Loop{`）。核实时请用上面那条精确命令。

**结论**：4 份循环中，2 份在 agent 内部（run/stream），1 份是死代码，1 份只服务 team。workflow 另有 `pkg/hno/workflow/executor.go` 自行驱动 `run.Loop`。

## 3. 里程碑真实进度（对照 v2 设计文档 §5）

| 里程碑 | 状态 | 证据 |
|---|---|---|
| M1 runner 状态机 | **写了但未接入** | 见 §2 第 3 行 |
| M2 工具类型安全 | 未做 | `grep -rln 'jsonschema\|"reflect"' --include='*.go' pkg/hno/tools/ \| grep -v _test` → 空 |
| M3 Skills | 已做 | `pkg/hno/skills/` **398 LOC**（非测试；含测试 684） |
| M4 observability | 零件在，未接线 | `grep -rn 'observability.Retry\|observability.CircuitBreaker' pkg/ --include='*.go' \| grep -v test` → 仅定义处，无业务调用 |
| M5 team Scheduler | 已做 | `pkg/hno/team/scheduler.go` |
| M5 workflow builder | 未做 | 仍为线性 `[]Step`。核验命令须避开 Go 标准库 `strings.Builder` 与代码注释：<br>`grep -rn 'AddNode\|AddEdge' --include='*.go' pkg/hno/workflow/ \| grep -v _test \| wc -l` → **0** |
| M6 发版 v2.0 | 未做 | 版本停留 1.2.9 |
| `openspec/` 提案目录 | **不存在** | `ls openspec/` → No such file（v2 设计文档 §7 引用了它） |

## 4. 参考仓库实测（借鉴成本依据）

| 指标 | 实测值 | 采集命令 |
|---|---|---|
| adk-go `workflow` 非测试 LOC | 6,244 | `find workflow -name '*.go' -not -name '*_test.go' \| xargs cat \| wc -l` |
| adk-go `workflow` 测试 LOC | 13,800 | `find workflow -name '*_test.go' \| xargs cat \| wc -l` |
| adk-go 图引擎核心 6 文件 | 2,134 | `cat graph.go scheduler.go base_node.go branch.go edgebuilder.go validation.go \| wc -l` |
| adk-go HITL/resume/state（我方短期不做） | 980 | `cat resume.go persistence.go state.go \| wc -l` |
| LangGraph 全仓库 py LOC | 190,919 | `find . -name '*.py' \| xargs wc -l \| tail -1` |
| LangGraph 引擎内核（_loop+_algo+_runner） | ≈ 4,400 | 1,988 + 1,460 + 941 |

**推论**：v2 设计文档 §4「LangGraph Pregel 在 Go 维护成本过高」的原论据有误 —— 该结论把 LangGraph 全仓库规模（19 万行，含 SDK/CLI/多 DB checkpointer）当作引擎复杂度。引擎内核实为约 4,400 行；adk 用 6,244 行 Go（含我不做的 HITL）在生产跑通。**图引擎成本可控。**

## 5. 零锁并发模型（借鉴可行性依据）

来源：`adk-go/workflow/scheduler.go` 头部注释原文：

> Concurrency model: producer-consumer over eventQueue.
> - Producers are the per-node goroutines started by scheduleNode. They only send to eventQueue …; they never read from it and never touch any other scheduler field.
> - The consumer is the single goroutine running scheduler.run … the only mutator of runsByName, runCancels, state.Nodes…
>
> Because producers and the consumer share only eventQueue (a channel — already safe for concurrent use), the consumer-only fields below need no mutex.

**Go 特有优势**：单 goroutine 独占可变状态是 Go 的天然模式，把锁问题在结构上消除。Python 版必须用 `asyncio.Lock`/`asyncio.Queue`。

## 6. LangGraph 独有能力定位（adk 不具备）

| 能力 | 精确位置 | 采集命令 |
|---|---|---|
| `Send` 动态扇出（map-reduce） | `langgraph/types.py:732` | `sed -n '732p' types.py` → `class Send:` |
| Durability 三档 | `langgraph/types.py:98` | → `Durability = Literal["sync", "async", "exit"]` |
| StreamMode 七种 | `langgraph/types.py:131` | → `StreamMode = Literal[` |
| RetryPolicy / CachePolicy / TimeoutPolicy / TracePolicy | `types.py:427 / 530 / 461 / 542` | — |
| Store 长期记忆（层级 ns + 向量） | `libs/checkpoint/langgraph/store/base/__init__.py` | — |
| recursion_limit / GraphRecursionError | `langgraph/errors.py:67` | — |

**注**：adk 的边是静态声明（`AddEdge`），运行时无法决定扇出数量与参数，故 `Send` 无对应实现。

## 7. 破坏性变更影响面实测

| 项 | 实测 | 结论 |
|---|---|---|
| `RunStreamResult` 外部消费方 | `grep -rn 'RunStreamResult' pkg/ cmd/ examples/ \| grep -v test` → 仅 `pkg/hno/agent/` 自身 | 流式 API 变更爆炸半径可控 |
| `output.Events` 消费方 | `pkg/hno/team/inheritance.go:35`、`pkg/hno/workflow/run.go:182`、`pkg/hno/workflow/step.go:129` | 扩展新事件类型为增量兼容 |
| 契约层对齐点 | `internal/session/contract` 9 测试，Go↔Python fixture；`contract_test.go:375` 无 fixture 时 skip | 事件化若改 Session JSON 必须同步 fixture |
