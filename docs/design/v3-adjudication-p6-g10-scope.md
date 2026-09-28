# P6 / G10 范围摸底材料 — workflow 迁移到图引擎的第一片该切多大

> 决策点登记：`docs/design/v3-platform.md` §2.1 G10 行（:88，估 ~1,200 LOC）、§3.1 包边界（:106–111）、
> §10 交付表 P6 行（:582）、§11 BC-3（:598）、§2.2 排除项（:96）、§13 风险行（:624）。
> 本文只登记决策点原文、今日实测与各选项的后果，**裁决权在负责人**：不代作决定、不改任何接口、执行器或测试。
> **本文不建契约**——G10 切片契约以本文的取舍落裁为前置。
> 本文件的写入是本轮唯一的仓库改动；实验全程在 `/tmp/g10probe`（`replace` 指向仓库、只读仓库字节），
> 仓库工作树除本文外逐字节未动。**明确未做**：两套内核的逐字段差分（上一轮正是死在这里）——差分属于实现阶段的验证手段，
> 不是本片的前置，本文不把它列为任何选项的开工条件。

## 1. 决策点原文与实测

### 1.1 决策点原文

**其一，母约 §2.1 G10 行**（`v3-platform.md:88`）：

> | G10 | workflow 迁移到图，公共 API 不变 | — | ~1,200 |

**其二，母约 §3.1 包边界**（:108，另 :111）：

> `pkg/hno/workflow/`  G10 内部改用 graph 编译，公共 API 不变
> **依赖方向单向**：`workflow → graph`，`graph` 不反向依赖任何上层包。

**其三，母约 §10 交付表 P6 行**（:582）：

> | **P6** | G10 workflow 迁移 | P1–P4 | 线性 []Step 编译为链式图 | **现有用户零改动**；`make test` 绿 |

三条验收即：①线性 `[]Step` 编译为链式图；②现有用户零改动；③`make test` 绿。

**配套条款**：§11 BC-3（:598）——「`pkg/hno/workflow` 内部实现换引擎｜性能特征变化｜**公共 API 不变**；
`Node.Execute(ctx, *ExecutionContext)` 签名保留」；§2.2（:96）——「agent 泛型化 `Agent[D,O]`（破坏性大改，**G10 稳定后单独做**）」；
§13（:624）——「图引擎做成『第二个 workflow』｜中｜验收标准必须包含并行/汇聚/条件路由/跨节点环——**静态线性能表达的不算 v1 达标**」
（该行管的是 P1/G2 的达标口径，不是 P6 的，但它是 OPT-γ 的唯一母约依据缺口，见 §2）。

**行号时效披露**：以上行号以 2026-09-28 取证时的 `v3-platform.md` 为准；母约正被各片持续写回，后续读者若见漂移以章节号为准。

### 1.2 基线（今日实测，本轮唯一测试命令面）

| 命令 | 观察 | 回执 |
|---|---|---|
| `go test ./pkg/hno/workflow/... -count=1 -v` | exit 0；**131 个 PASS（含子测试）、0 FAIL、0 SKIP**；包级 `ok github.com/rexleimo/agno-go/pkg/hno/workflow 1.072s`；109 个 `Test` 函数 | `receipt:312de106-ff8e-4006-8918-8b0720d2d824` |
| `go build ./cmd/examples/workflow_demo ./cmd/examples/workflow_history` | exit 0 | `receipt:d52c41ef-2953-4e31-a569-5bb0686e4203` |
| `git hash-object --no-filters pkg/hno/workflow/*.go` | exit 0，34 个字节锚（§1.6） | `receipt:61e7faf5-4b46-4db5-b791-9d15cb887ae0` |

**规模复核（对母约 ~1,200 的第一条张力）**：`pkg/hno/workflow` = 非测试 20 文件 / **3,263 行**，测试 13 文件 / **5,373 行**；
导出面（`grep -E '^(func\|type\|var\|const) [A-Z]'` 与导出方法，均排除 `*_test.go`）= **61 顶层 + 85 方法 = 146**（母约系列口径 153，差的是 const 块成员，如 5 个 `NodeType*`）。
`pkg/hno/graph` 非测试导出面 = **42 顶层 + 19 方法 = 61**（母约系列口径 64，同因）。
→ 母约给 G10 的 ~1,200 既不是「整个 workflow 换内核」（3,263 行的包 + 5,373 行测试），也不止「一条链换个跑法」（本文 §1.5 实测线性适配器净 78 行）。**它没有指明替换哪条路径——这才是本片要裁的东西。**

**耦合实测（第二条张力，逐条 grep 担保）**：

- `pkg/hno/workflow` 内 **零处** import `hno/graph`（`agno-go/pkg/hno/graph` 的全部命中：`internal/hitlbridge/bridge.go:20`、`internal/hitlbridge/p5g7_wiring_test.go:11`、以及 `pkg/hno/graph/*_test.go` 自身）。
- `pkg/hno/workflow` 的仓库内消费者只有两个示例：`cmd/examples/workflow_demo/main.go:14`、`cmd/examples/workflow_history/main.go:11`（对外部而言是包外用户）。
- workflow 的线性执行体今天跑在 **`run.Loop`** 上（`executor.go:31`），即母约 §0.1 收口后剩下的两份循环之一；G10 换内核等于把 workflow 从 `run.Loop` 搬到 `graph`。

### 1.3 M1：执行路径盘点（读码 + file:line）

**先纠一个直觉**：`[]Step` **不是**由 `loop.go/parallel.go/router.go/condition.go` 驱动的。唯一的线性执行器是 `executeSteps`
（`executor.go:27`，内部 `run.Loop` + `Next` 按 `startIdx+len(history)` 取下标，`executor.go:32-47`），
**全仓只被调用一次**：`workflow.go:222`。复合节点各自在自己的 `Execute` 里递归下沉，不经执行器：

| 文件:行 | 角色 | 做什么 |
|---|---|---|
| `executor.go:27` | **唯一的线性内核** | 按位置跑 `steps[startIdx:]`；`current = result` 替换指针（:43）；失败时 `types.NewError(ErrCodeUnknown, "step %s failed", err)`，其中 step ID 取 `lastStepIDOf(history)`＝**最后一个成功步**（:55/:71）；事件按步聚合 `collectStepEvents`（:80） |
| `step.go:84` | 叶子 | 跑 agent，历史注入判定 `shouldAddHistory`（:158） |
| `generic_step.go:76` | 叶子（泛型） | `StepFunc[In,Out]` + JSON 编解码 |
| `condition.go:53` | 控制流（二分支） | 谓词读 `*ExecutionContext`（:9），把判定写进 `condition_<id>_result`（:57），只跑 True/False 之一；nil 分支＝no-op（:70） |
| `router.go:54` | 控制流（多选一） | `RouterFunc` 返回 label 字符串，写 `router_<id>_selected`（:58）；**label 未命中即报错**（:63）；nil 节点＝no-op |
| `loop.go:61` | 控制流（计数循环） | 自带 `MaxIteration`（默认 10，:47-49），谓词 `Condition(execCtx, iteration)`（:66），写 `loop_<id>_iterations`（:81），错误文案 `loop %s iteration %d failed`（:73） |
| `parallel.go:46` | 控制流（扇出+汇聚） | **自己解决共享可变状态**：每支克隆 `SessionState`（:54-61）、每支新建 `ExecutionContext` 并复制 `Data`/`Metadata`（:70-90）；错误只报 `errors[0]` 并包成 `parallel execution failed: %w`（:113）；汇聚用 `MergeParallelSessionStates`（:132）+ 前缀键 `parallel_<id>_branch_<i>_*`（:140-145）；`Output` 取**最后一支**（:152） |

**线程化状态**：`*ExecutionContext`（`execution_context.go:9-45`）＝ `Input`/`Output` string + `Data` map + `Metadata` map +
`SessionState *SessionState` + `SessionID`/`UserID` + `WorkflowHistory []HistoryEntry` + `HistoryContext`。
传递方式是**指针替换**，不是值拷贝（`executor.go:30/43`；节点可原地改也可返回新对象，两条都合法）。

**横切（不属于控制流，换内核时必须原样留在 workflow）**：run context 装配（`workflow.go:129-147`）、历史加载
（`workflow.go:284` + `memory_storage.go`/`storage.go`）、`WorkflowRun` 生命周期与落库（`run.go:101-145`、`persistence.go:10`）、
取消记录（`workflow.go:230-246` 判 `ctx.Err()` 后 `ApplyCancellation` + `saveCancellation`，`session.go:28` `CancellationRecord`）、
metrics（`metrics.go:12`、`persistence.go:43/57`）、事件聚合（`workflow.go:363`）、resume（`run_options.go:36` + `workflow.go:205-218`）、
历史注入 agent 指令（`history_injection.go:13`，该文件 import `pkg/hno/agent`——**workflow 已经依赖 agent**，图侧不许依赖，所以这块只能留在 workflow）。

**能力 → 是否控制流 → 图侧有无对应物**

| workflow 能力 | 控制流？ | 图侧对应物 | 实测落差 |
|---|---|---|---|
| 线性 `[]Step` | 是 | `AddEdge` 链 + `SetEntry`/`SetOutput`（`graph.go:227/270/281`） | **无落差**（§1.5 [P2]/[P3]） |
| Resume from step | 是（入口选择） | `SetEntry(steps[k])` | 无落差（[P3] 两侧同输出） |
| Condition（互斥二分支） | 是 | 只有 `AddConditional(from,to,Predicate)`，**命中几条激活几条**（`graph.go:231-237`、`scheduler.go:398-419`） | 无「互斥分支」原语；谓词读的是**前驱输出 `any`**（`graph.go:47`），不是 `*ExecutionContext` → 要插一个「判定节点」把 bool 放进输出 |
| Router（按 label 选一） | 是 | N 条条件边 + 1 条兜底边可仿 | 语义不等价：today **未命中即 error**（`router.go:63`），图侧未命中走 `AddDefault` 且不报错（`scheduler.go:411-418`） |
| Parallel（扇出+汇聚） | 是 | 扇出 `AddEdge` × N + `AddJoin`（`graph.go:257`） | join 输入是 `map[string]any`（前驱名键控，`graph.go:245-262`、`scheduler.go:372-387`），**不是** ExecutionContext；且扇出把**同一个可变指针**交给并发 goroutine → 实测 DATA RACE（[P6]） |
| Loop（计数循环） | 是 | 条件环可表达（`rejectUnconditionalCycles` `graph.go:475`） | 图侧安全阀是**全局** `WithStepLimit`（默认 1000，`graph.go:87/54`），不是 per-node 上限；`loop_<id>_iterations` 记账与 iteration 错误文案要适配器自数环 |
| 错误聚合与文案 | 否 | 引擎交节点原始错误 | 见 [P4]：文案不逐字节相同，且 today 报的是**最后一个成功步** |
| 事件时点 | 否 | 图侧只有 `WithTrace`/`NodeEvent`（`policy.go:59-66`） | today 事件在步返回后从 `ExecutionContext` 反查（`executor.go:80`、`workflow.go:363`）；图侧 trace 在消费者侧发射（`scheduler.go:352`）→ 时点/顺序要重新对齐 |
| 会话/历史/持久化/取消/metrics | 否 | **无对应物，也不该有**（graph 不依赖 session/agent） | 留 workflow，任何 OPT 都一样 |

### 1.4 M2：形状对撞（本片核心判据，只量不实现）

两边签名：

- `graph.Node`＝`Name() string` + `Run(ctx context.Context, in any) (any, error)`（`graph.go:24-30`），便捷构造 `NodeFunc`（:42）、`Typed[TIn,TOut]`（:622）。
- `workflow.Node`＝`Execute(ctx context.Context, input *ExecutionContext) (*ExecutionContext, error)` + `GetID()` + `GetType()`（`workflow.go:32-36`）。

**「把 ExecutionContext 塞进图状态穿过」是否可行**：**线性链可行且无落差**（探针 [P2]：链上每节点 `in` 断言回 `*ExecutionContext`、
返回同一个指针，`Run` 后 `Output` 与旧核逐字符相同；[P2b]：适配器交回的就是 seed 对象本体——引用穿透，不是拷贝）。
代价与断裂点有三条，全部实测：

1. **并发暴露（OPT-β 的硬阻塞）**：`successors` 对每条命中的边交出的是**同一个 `out`**（`scheduler.go:406`），`spawn` 给每个激活各起一个 goroutine（:261-271）。
   探针 [P6] 用 `AddEdge("e","a")`+`AddEdge("e","b")` 扇出，两支都往 `ec.Data` 写：`go run -race` **实测 WARNING: DATA RACE ×2**（一写一读），
   进程以 `exit status 66` 收场，且两支对 `Output` 的追加顺序在两次运行间翻转（`"|e|b|a"` vs `"|e|a|b"`）——**这是累加器语义，不是结果语义**。
   今天的 `Parallel` 恰恰是靠**自己克隆分支上下文**（`parallel.go:54-90`）绕开这件事的；把扇出交给图就丢掉了那层保护。
2. **形状断裂**：join 激活交出的是累加器 map **本体**（`scheduler.go:384` `activation{name: target, in: collected}`，无克隆）——
   正是 `S19-STD-1` 登记的那条教训：「激活拿到的是累加器 map 本体」（`docs/design/v3-adjudication-r19-q2-send-join.md:89`，
   另 :117「快照修法（`maps.Clone`/重建 map）从可选变成前置」；`v3-p5-graph-g7-refactor.md:84` 记录 P5 已用防御性拷贝从源头不制造别名）。
   探针 [P6] 实测：join 节点收到 `keys=2` 的 `map[string]any`，`m["a"]` 断言回 `*ExecutionContext` 成立且 `sameObjectAsSeed=true`。
   → 链上穿过没问题，**任何一个汇聚点都要把 map 重新装配成 ExecutionContext**，而装配语义（谁覆盖谁、`Data` 前缀、`Output` 取哪一支、SessionState 三方合并）
   今天住在 `parallel.go:127-153`，图侧没有对应物——它只能写进 workflow 的适配器，且**必须逐条钉死**，否则就是新的公共语义。
3. **观测丢失**：`Result.Completed()` 实测是**字典序** `[a b e j]`（探针 [P6]），不是执行序；引擎失败路径交 `nil` Result（`scheduler.go:113-124` + `consume`）。
   → `lastStepID`、按步事件聚合、取消快照都要适配器自带账本（探针 `ledger` 25 行）。

### 1.5 M3：窄探针（一次、且仅一次）—— `/tmp/g10probe/cmd/linprobe`

3 个等价 step 的线性链，同一批 step 分别跑旧核（`Workflow.Run`）与链式图（`NodeFunc`+`AddEdge`+`Validate`+`Run`）。
`node /tmp/rex-receipt.mjs -- go -C /tmp/g10probe run ./cmd/linprobe` → **`receipt:25204637-5023-49c2-abeb-3c365a2e4604`（exit 0）**；
`GORACE=halt_on_error=0 … go run -race ./cmd/linprobe` → **`receipt:7e8029a8-5ad6-4ce7-8de3-c4f3fb5c10a8`（exit 1，即 DATA RACE 实测）**。

| 读数 | 实测 |
|---|---|
| [P1]/[P2] | 旧核 `"\|s1\|s2\|s3"` ＝ 图核 `"\|s1\|s2\|s3"`，`sameAsOld=true` |
| [P3] resume | 旧核 `WithResumeFrom("s2")` → `"\|s2\|s3"`；图核 `start=1` → `"\|s2\|s3"`（一致） |
| [P4] 失败文案 | 旧核 `"[UNKNOWN] step s1 failed: boom"`；朴素适配器 `"step s1 failed: boom"` → **前缀 `[UNKNOWN] ` 会丢**；且两边都报 **s1（最后一个成功步）而非失败的 s2**——这是今天既有语义，换内核必须复刻这份「别扭」而不是顺手修正 |
| [P5] 同名 step ID | 旧核：两个 `id="dup"` 的 step → `err=<nil>`（合法）；图核：`Validate` → `"graph: node name \"dup\" is declared more than once"`（拒绝）→ 适配器**必须合成唯一节点名**，且合成名不能泄漏进任何对外可读处 |
| [P6] | 见 §1.4 三条 |

**适配器真实行数**：`/tmp/g10probe/cmd/linprobe/adapter.go` = **98 行**（非注释非空行 **78 行**），三段：`ledger`+`uniqueName` 合成 33 行、
`compile`（Step→NodeFunc + 链边 + entry/output + Validate）33 行、`runChain`（错误包装 + lastStepID）21 行；`main.go` 159 行是探针读数，不入库。
入库后的真实代价还要加：`[UNKNOWN]` 前缀复刻、按步事件账本、`resume` 的「step 不存在即 `InvalidInput`」既有分支（`workflow.go:215-217`）、
形状判据（哪些 `NodeType` 允许走图）——**估 120–180 行**，加测试。既非 ~1,200，也非「零」。

**探针边界（如实）**：只覆盖线性链 + 一个扇出/join **形状**演示；未覆盖 loop/parallel/condition/router 的行为等价，
未做两套内核的逐字段差分，未跑全仓测试，**未跑 `pkg/hno/graph` 的任何测试**（兄弟片在途领地，且 `TestP1R17_*` 高负载偶发红，不在此误判）。
上一轮遗留物披露：`/tmp/g10probe/wfexp/`（含 `graphcompile_exp.go` 与 `kernel_diff_exp_test.go`）**本文未运行、未复用其任何结论**；
本文适配器为自写，只借了同一形状。

### 1.6 M4：零改动担保面（可比基线 + 字节锚）

- 可比基线：`go test ./pkg/hno/workflow/... -count=1 -v` → **131 PASS / 0 FAIL / 0 SKIP**（§1.2，`receipt:312de106-…`）。「make test 绿」在本片读作：**这 131 项不改一项地继续绿**。
- 示例编译：`workflow_demo`、`workflow_history` exit 0（`receipt:d52c41ef-…`）——它们是「现有用户」的可执行代理；对外包外用户还有 `CHANGELOG`/`website` 里的示例文案，本片未逐条核对（未测清单）。
- **必须逐字节不改的既有测试锚**（`git hash-object --no-filters`，`receipt:61e7faf5-…`）：

```
b4ce15098d1d321c498845e156e015b92d93f029  execution_context_bench_test.go
0aa74b4816427f8dc252b3c1689d05bbfc8f03f3  execution_context_test.go
387885735f6b4b8772f7f1eec30394b76d9ecbb6  generic_step_test.go
e5edbf227111ba81f9aff210ab07953406ebc0ec  history_injection_test.go
4522343b5f822710e629733d4ff45bb7052dd2c3  memory_storage_test.go
c126b0e230ee83672b9d3faac0a3da55aa3183d7  metrics_test.go
02b847a341b6f76b62bd501541f29074a1991ba1  run_test.go
d79645aab3ff7b231c03a951d685d0595c43205d  session_state_test.go
813ba8c592c045d950a9739148d03668bf41d2e7  session_test.go
2c308979684854c76464f5df61e85dbf27e42569  step_history_test.go
92699523ad3f856ad417171342bf3b7be6a2a0cc  workflow_history_e2e_test.go
ef6951bb70a19e1569ab63ca965dbb90b58c547e  workflow_history_test.go
4a822eb55a3827e0b42810eb1e9bed778156a0ea  workflow_test.go
```

- 非测试文件锚（换内核真正会碰的只有 `executor.go` 与其调用点 `workflow.go`；其余 18 个属于「不该动」面）：

```
a06f140a52dbf2c32f35aadb0fbb687b18d03647  condition.go      6f39729b3377ee6b82f140f0b72faefd69fc765e  events.go
3aa215ef03fc72fb7be8e2765e003ccfb95b818d  execution_context.go  e720fb7555000776a568b1c3471f531955219689  executor.go
d9ecc9e46c204275cee5f341c213a3b341857869  generic_step.go   9e27df65ffb85917f25d33cbf87ca02c776da00d  history.go
3d7b2235e0e3a998b2188946d8300bea0502322e  history_injection.go  ef8619f93b47f6df32fcde2e15446c5a9ccf72f8  loop.go
19a8784bfe821892292682963f835f36297b5bb6  memory_storage.go 28b1feca0d3ca7d66d04910090e20823a4d3d127  metrics.go
01a0c050478c4954b955057d06235e03cbb6d5ff  parallel.go     b0d3ac7784a043634319c008f0550f72172564ad  persistence.go
29906434bc0d2c354c6eb7deafdf51de749fd0c7  router.go       9cfa348f0f7e7b01fb32c6b6d6737145dab1d527  run.go
7f293f7753d664b82e8ff360e1781f3c82762ce4  run_options.go  3c4529f8e4a910bba07dab2778c0572da1a50d71  session.go
64a0a811ebee2c47d5c1754d8c4f41b36737cfc4  session_state.go 0e6d8d69a0291b61f986f1331edc9508c0e9297c  step.go
423618e93d74c70dd549b03eaffb3d6cd8d39506  storage.go      8622280b7485681eb4c4d7b01a95dc4b37b9b5f5  workflow.go
```

## 2. 选项

每条都要回答：怎么担保「现有用户零改动」／「make test 绿」要动多少既有测试字节／是否要给 `pkg/hno/graph` 补公共面（补面＝动导出数字，本系列硬门禁）。

### OPT-α 只换线性链

- **语义**：`Steps` 全部是叶子（`Step`/`GenericStep`）时，`executeSteps` 走「链式图」；只要出现 Condition/Loop/Parallel/Router，**整条**走今天的 `run.Loop` 路径。横切面（历史/会话/持久化/取消/metrics/事件）一行不改。
- **实现代价（实测校准）**：核心 78 行 → 入库 **≈120–180 行**（§1.5）+ 新测试（按仓库配比 1:1.6，约 200–300 行）。**远低于母约 ~1,200**。
- **开工前置（必须先写死，否则「零改动」是口号）**：①形状判据本身是可观察行为，要进契约；②错误串逐字节复刻（[P4] 的 `[UNKNOWN] step <最后一个成功步> failed:`）；
  ③`lastStepID`/按步事件账本（[P6] Completed() 是字典序）；④同名 step ID 合成唯一节点名且不外泄（[P5]）。
- **破坏性**：签名零改动（BC-3 满足，`Node.Execute` 原样）。既有 13 个测试文件（§1.6）**目标逐字节不改**；
  诚实标注：**本文没有执行「换内核后」的 workflow 测试面**，所以「不改一个字节」目前是判据而非实测（未测清单第 1 条）；
  已实测的是「会对撞的形状」有哪四处（[P4]/[P5]/[P6]/Completed 序），它们都能被上面四条前置钉住。
- **需要给 graph 补公共面吗**：**不需要**——探针只用到 `NodeFunc`/`AddEdge`/`SetEntry`/`SetOutput`/`Validate`/`Run`/`Result`，全在既有 61 个导出声明内。导出数字零变化。
- **对验收**：①「线性 []Step 编译为链式图」＝字面达成；②「现有用户零改动」＝可担保（判据①～④）；③「make test 绿」＝代价最低。
  **代价**：双内核并存——同一包里一半路径走 `run.Loop`、一半走 `graph`，错误聚合/事件时点/取消语义**天然不一致**（除非判据①～④逐条复刻并被测试钉住）；且 §13（:624）「静态线性能表达的不算 v1 达标」虽管的是 G2，读作 G10 时它没被满足。

### OPT-β 控制流全换（线性+条件+并行+loop 走图；会话/历史/持久化留 workflow）

- **语义**：workflow 变成 graph 的**声明前端**，复合节点编译成子图。
- **实现代价（实测外推，别抄 ~1,200）**：线性 78 + 互斥分支装配（互补谓词 + 判定节点，约 80–120）+ Router 语义映射（约 60–100，含未命中报错）+
  Loop 自数环与 `loop_*_iterations` 记账（约 80–120）+ Parallel 的分支克隆与三路合并（今天 `parallel.go` 166 行的逻辑要在适配器侧重建，约 250–350）+
  join 处的 map→ExecutionContext 重装配（约 120–200）≈ **适配器 700–1,100 行**，加测试（1:1.6）**总量 ≈1,800–2,900 行**，**超母约**。
- **开工前置（三条硬事实先裁决，缺一条就会静默产出错误语义）**：
  (i) **图无互斥分支原语**——要么 workflow 侧写互补谓词（谓词成对出现，一漏即两分支同跑），要么给 `pkg/hno/graph` 加 `Branch`/`Router` 公共面＝**动导出数字**（硬门禁，且 graph 是兄弟片在途领地）；
  (ii) **Router 未命中语义不等价**（today 报错 vs 图侧走兜底）；
  (iii) **累加器进并发**：[P6] 实测 DATA RACE，且 join 交出的是累加器 map 本体（`S19-STD-1`）——必须先裁决 `ExecutionContext` 入图是**引用语义**（则任何扇出/汇聚都要在 workflow 侧克隆，等价于把 `parallel.go` 再写一遍）还是**值语义**（则 `Data`/`Metadata`/`SessionState` 的拷贝规则本身变成新公共语义，today 没有）。
- **破坏性**：最高。事件时点、取消快照的取得时刻（`workflow.go:234` 在步返回后）、109 个既有 `Test` 函数中的行为断言都可能漂移；「双内核」消失的同时换来「单内核新语义」。
- **对验收**：①超额达成（不只线性），但母约 P6 行的交付面只要求线性——β 把 P6 变成一次大版本级重构。

### OPT-γ 不换内核（只把声明编译成 graph 描述，用于校验/可视化）

- **语义**：workflow 增加一个「编译为 `*graph.Graph` 并 `Validate()`」的旁路（可暴露图侧的不可达/重复名/无条件环告警），执行仍走 `executeSteps`。
- **实现代价**：≈200–300 行（声明→图映射 + 名称合成 + 校验报告）。
- **破坏性**：**最低**——执行路径一行不改，131 PASS 原样绿；两个示例不改。
- **但要说实话**：`Workflow.Steps` 若加导出方法（如 `BuildGraph()`）就是**动 workflow 导出数字**（146→147），同样过门禁；不加导出面则这层能力只在内部，用户拿不到，
  「迁移」名不副实：§0.2 推翻 v2「不用图引擎」的决定没有被兑现，红线 2「循环只有一份」也没改善——workflow 继续跑在 `run.Loop` 上，仓库仍是两份循环并存。
- **对验收**：①「线性 []Step 编译为链式图」**只在字面上**满足（编译了，没跑）；②③满足。负责人若按字面验收，本片会通过；按动机验收，它不会。

### OPT-δ 推迟（等 agent 泛型化或 G4 生产者片落地后再动）

- **语义**：P6 行 v3.0 空转，G10 留 §3.1 的书面意图。
- **真实代价（这是它唯一但重要的优点）**：**推迟不产生任何回改面**——实测 workflow 对 graph 今天零 import（§1.2），workflow 的消费者只有两个示例；
  没有任何存量代码因为「没换内核」而将来要改。§1.4 的三条形状断裂也**不会因为等待而消失**，但也不需要为它们付任何学费。
- **破坏性**：零（什么都不做）。
- **反向代价**：①§2.2（:96）把 agent 泛型化排在「G10 稳定后」，G10 不动则泛型化永远没有前置；②`run.Loop` 与 `graph` 两份循环继续并存，红线 2 只兑现一半；
  ③关键路径 `P0 → P1 → (P2/P3/P4) → P6` 的终点空着，P1–P4 已交付的引擎能力（策略/ Durability/HITL/Send）在 v3.0 里**没有任何 workflow 侧消费者**——
  仓库内 graph 消费者只有 `internal/hitlbridge`（§1.2），这是 G2 投资的实际回收率。

## 3. 推荐与理由

**推荐 OPT-α（只换线性链），并把「双内核边界」当成第一页契约内容而不是实现细节**。依据：

1. **它是唯一「零补面」的换内核选项**：探针跑通只用既有 61 个 graph 导出声明（§1.5），不动 `pkg/hno/graph` 的任何数字——
   而 OPT-β 的前置 (i) 十有八九要动它，那既是硬门禁、又落在兄弟片在途领地。
2. **实测把「双内核」的代价变成了可数四条**：[P4] 错误前缀与「最后一个成功步」、[P5] 同名 ID、[P6] Completed() 字典序、join/扇出形状断裂——
   线性链只碰到前两条半（第三条是账本，第四条在纯链上不出现）。**双内核真正不可控的是 β 的 (iii)**，α 不需要裁决引用/值语义就能收口。
3. **量级诚实**：α 实测核心 78 行、入库 120–180 行，与「整个 workflow 换内核（3,263 行 + 5,373 行测试、146 个导出符号）」是两件事，
   母约 ~1,200 既不指 α 也不指 β——**这个偏差要在落裁时改母约，不能靠猜**（母约由主代理独占写回，本文不代改）。
4. **γ/δ 不是「更便宜的 α」**：γ 让 G10 只在字面成立（§0.2 的推翻没兑现，还可能动 workflow 导出数字），
   δ 的唯一优点是零回改面（实测零耦合担保），但它把 P1–P4 的引擎投资全部留在无消费者状态。选它们必须是**主动取舍**，不是省工。

**必须在落裁时先写死、不能留给实现猜的边界**（承接片契约的第一页）：

- **形状判据**：哪些 `NodeType` 组合允许走图（建议：`Steps` 全为 `Step`/`GenericStep`）；混合链**整条**走旧器，不做「按段切换」——按段切换会把双内核塞进同一次 Run。
- **错误聚合逐字节复刻**：`[UNKNOWN] step <ID> failed: <inner>`，且 `<ID>` 是**最后一个成功步**（today 语义，见 [P4]）；修正它属于行为变更，要单独的 BC 条目，不许夹带。
- **`lastStepID` / 按步事件**来源：适配器账本（执行序），不得读 `Result.Completed()`（字典序，实测）。
- **同名 step ID**：图内合成唯一名（`id#2` 类），合成名**不得**出现在任何对外可读处（`WorkflowRun.Messages`、取消快照 `StepID`、日志 `resume_from` 都用原 ID）。
- **取消与 resume 的取得时刻不改**：`workflow.go:215-217`（step 不存在 → `InvalidInput`）、`workflow.go:230-246`（`ctx.Err()` → `CancellationRecord` + `saveCancellation`）在图路径上必须走同一份代码，不在适配器里重写。
- **`ExecutionContext` 引用语义仅适用于「无扇出」的链**：一旦某条链出现任何 fan-out，引用穿透即 DATA RACE（[P6] 实测）——
  这条要写成**显式非目标**，而不是等 β 时才发现。
- **本片不做的事**：不做两内核逐字段差分（属实现阶段验证）；不给 `pkg/hno/graph` 补公共面；不动 `history_injection.go` 的 agent 依赖；不动 §1.6 的 13 个测试文件字节。

**权属声明**：裁决权在负责人。本材料只登记决策点与实测，不擅自改任何接口、执行器或测试（本文写入是本轮唯一仓库改动；探针在 `/tmp/g10probe`，不入库）。
G10 切片契约以本文落裁为前置——**结论给出前不建契约**，也不因本文预写任何一方落地后的文案或测试期望。

## 4. 登记

- 对应母约：§2.1 G10 行（`v3-platform.md:88`）、§3.1（:106–111）、§10 P6 行（:582）、§11 BC-3（:598）、§2.2（:96）、§13（:624）。
  母约与状态文档（含 `v3-p1-graph-status.md` §4l/§6 的回执纪律教训——**本文所有 ID 均为本轮现取现贴，无一条凭印象**）由主代理独占写回，本文不代改。
- 回执登记（全部为 `/tmp/rex-receipt.mjs` 实际打印，文件已抽查存在，5/5 存在于 `.rex-harness/receipts/`）：
  - 探针 `receipt:25204637-5023-49c2-abeb-3c365a2e4604`（`go -C /tmp/g10probe run ./cmd/linprobe`，exit 0）
  - 探针 `-race` `receipt:7e8029a8-5ad6-4ce7-8de3-c4f3fb5c10a8`（exit 1 —— **这就是 [P6] 的 DATA RACE 实测**，`exit status 66`；不是回归、不是环境红）
  - 基线 `receipt:312de106-ff8e-4006-8918-8b0720d2d824`（`go test ./pkg/hno/workflow/... -count=1 -v`，exit 0，131 PASS / 0 FAIL / 0 SKIP）
  - 示例编译 `receipt:d52c41ef-2953-4e31-a569-5bb0686e4203`（exit 0）
  - 字节锚 `receipt:61e7faf5-4b46-4db5-b791-9d15cb887ae0`（`git hash-object --no-filters`，exit 0）
  - 过程失败登记（未静默丢弃）：①探针第一版 `[P2]` 的 `Printf` 动词与参数不匹配、`[P2b]` 断言写成与新建对象比较——**在运行前经走查修正**，
    两条修的都是探针自身，未产生错误读数；②第一次「非测试导出面」统计把 `*_test.go` 计入（195/95），已用排除测试文件的第二次统计替换（61/85），
    第二次未走回执（纯静态 grep，读数已在 §1.2 标注口径与母约 153/64 的差异原因）。
- **未测清单 / 观察上限（到预算线停笔，未补齐）**：
  1. **未执行「换内核后的 workflow 测试面」**——§2 OPT-α 的「13 个测试文件逐字节不改」目前是判据，不是实测；担保方式是已实测的四处对撞形状（[P4]/[P5]/[P6]/Completed 序）逐条进前置边界，实现片必须以一次 `-count=1` 执行收口。
  2. 未覆盖 Condition/Loop/Router/Parallel 与图的行为等价（母约 M 行第 3 条明令禁止）；Parallel 的克隆/合并语义只做了读码归纳（`parallel.go:46-156`），未逐项执行验证。
  3. 未测事件时点差异：`collectStepEvents`（`executor.go:80`）与图侧 `trace.emit`（`scheduler.go:352`）的发射次序、`TestWorkflow_RunStoresEvents` 的期望是否会被图路径改写——未实测。
  4. 未跑 `pkg/hno/graph` 任何测试（兄弟片在途领地，且 `TestP1R17_*` 偶发红，本片不采信号）。
  5. 未做 `make test` 全仓（含 `pkg/agentos`/契约层）基线——本文只给了 workflow 包的可比基线。
  6. 未核对 `CHANGELOG`/`website`/`docs` 里面向用户的 workflow 文案面（「现有用户」的非代码部分）。
  7. 未量 `-race` 下 workflow 既有并发测试（`TestWorkflowHistory_Concurrency`、`TestSessionState_ThreadSafety`、`TestParallel_SessionStateIsolation`）在图路径上的表现。
- **独立性披露**：本文为**单代理自查材料**（取证、实验、成文同一代理），**未派发任何审查**；
  仓库内测试仅在 `./pkg/hno/workflow/...` 与两个示例的 `go build` 上执行（本轮唯一命令面）；
  工作树自证（本轮结束时）：`git status --porcelain` ＝ ` M yarn.lock`（母约系列既存项）＋ `?? docs/design/v3-adjudication-p6-g10-scope.md`（本文），
  `git diff --name-only` 只有 `yarn.lock`，`.go` 改动数 ＝ 0。派发说明里列为既存的 `docs/design/v3-test-scope-p8-g8-store.json`
  本轮已不再是未跟踪项（16:04 由主代理入库），本文未触碰它。
  上一轮遗留的 `/tmp/g10probe/wfexp/`（含差分测试文件）**未被运行、未被采信**，本文结论全部出自 §1.5 的单次窄探针与读码 file:line。

## 5. 落裁登记（主代理 2026-09-28，负责人裁决）

- **裁决：OPT-β「控制流全换」**——负责人选择本文 §2 的三个选项里实现代价最大的一档，**逆着 §3 的「推荐 OPT-α」**。
  据此写回母约：§2.1 G10 行（成本列由 `~1,200` 改为实测外推 `≈1,800–2,900`，合计行随之变为 `~10,100–11,200`）、
  §10 P6 行（交付列由「线性 []Step 编译为链式图」替换为 β 口径，并把三项开工前置写进验收列）。
- **本文的推荐未被采纳，其依据仍然有效**，因此 β 的开工条件按本文 §2 OPT-β 的前置清单逐条转成阻塞项，而不是当作风险提示：
  1. `(iii)` 累加器进并发是 **β 的硬阻塞**（本文 §1.5 已实测 `[P6]` DATA RACE，`receipt:7e8029a8-5ad6-4ce7-8de3-c4f3fb5c10a8` exit 1）。
     裁决入口是既有欠项 **`S19-STD-1`**（`v3-adjudication-r19-q2-send-join.md:89`：激活拿到的是累加器 map 本体）——
     引用语义 / 值语义二选一，只有负责人能裁。**此项未清 ⇒ P6 不建切片契约。**
  2. `(i)` 图无互斥分支原语：互补谓词（workflow 侧，一漏即两分支同跑）或给 `pkg/hno/graph` 加 `Branch`/`Router` 公共面
     （撞 D7 型「导出面计数」硬门禁，且图目录是兄弟片在途领地）。此项决定 β 的改动落点，属实现片的第一决策点。
  3. `(ii)` Router 未命中语义不等价：β 采「today 报错优先」，适配器必须复刻 `router.go:63` 的错误形状，
     不许借道图侧 `AddDefault` 的静默兜底。
- **本文未做的事一条不改**：§4「未测清单」7 项（尤其第 1 项——α 的「13 个测试文件逐字节不改」至今是判据不是实测，
  β 之下未测面更大）全部转给实现片，实现片不得以「摸底已给 file:line」替代执行证据。
