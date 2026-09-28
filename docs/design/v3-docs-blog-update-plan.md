# v3 迭代文档与 Blog 更新计划

记录时间：2026-09-28。性质：文档/blog 侧的执行计划（第三轮并行派发产物之一）。
范围：`website/`（VitePress 站点）与 `website/blog/` 的增量更新；本文件只做规划，
不携带内容正稿。所有素材以已落地字节与在册证据为准。

素材来源（只读引用，不改动）：
- 母约：`docs/design/v3-platform.md`（§3 图引擎、§4 策略、§5 G4 协议、§7 G6 Durability、§9.3 D1、§12 交付表）
- 实测数字：`docs/design/v3-p1-graph-status.md`（§1 门禁实测、§2 B 行背书、§4 各切片交付登记）
- 证据链：`v3-red-observation-p2g3.md` / `v3-p2-graph-g3-green.md`、`v3-red-observation-p3g6.md` / `v3-p3-graph-g6-green.md`、`v3-red-observation-p4g4.md` / `v3-p4-run-g4-green.md`
- 裁决材料：`v3-adjudication-r19-q2-send-join.md`（R19-Q2 = OPT-C，G5 解锁）、`v3-adjudication-d1-session-events.md`（D1 = OPT-1，G7 解锁）
- 公开 API 面：`pkg/hno/graph`（go doc，导出符号恰 41）、`pkg/hno/run`（导出符号恰 77）、`Agent.RunStreamMode`

---

## §1 现状盘点

先说结论：**v3 图引擎这条线在网站上目前是零覆盖。** `grep -rln "pkg/hno/graph" website/`
零命中；`website/guide/workflow.md` 与 `website/api/workflow.md` 描述的是 v1
`pkg/hno/workflow`（Step/Condition/Loop/Parallel/Router 五原语），与 v3 零锁图引擎
（`pkg/hno/graph`）是两套 API。G10（workflow 内部迁移到图、公共 API 不变）落地前，
两套文档必须并存、各自指向真实包路径，不得混写。

| 能力（已落地） | 网站板块 | 现状 | 缺口 |
|---|---|---|---|
| 零锁图引擎（DAG/并行/条件/兜底/Join、构建期校验、步数阀、panic 屏障、取消归一、Typed 节点、max-concurrency FIFO、join 屏障） | guide / api / advanced | **零覆盖**。guide/workflow.md 只讲 v1 workflow；无任何页面提及 `pkg/hno/graph` | 缺一篇图引擎 guide（核心概念）、一篇 api/graph.md（签名级 API 参考）；advanced/architecture.md 未反映单消费者零锁模型 |
| 节点策略四合一（`WithRetry/WithTimeout/WithCache/WithTrace` 变参 `AddNode`；guard 钳制零值；缓存 fail-closed 只缓存成功；trace 消费侧 Hook） | guide / api | **零覆盖**。`AddNode(n, opts...)` 的变参形态无任何文档 | 缺策略指南一节或独立页：四策略的零值语义（`MaxAttempts<1` 即 1 次、`Timeout<=0` 即不限时、「不重试/不限时/不缓存」零值直给）、CacheStore 适配缝、NodeEvent Hook 契约 |
| Durability 三档（Sync/Async/Exit）+ Checkpointer/Checkpoint（消费侧时机、零锁；resume/replay 挂起至 G7） | guide / advanced | **零覆盖**。advanced/observability.md 讲的是 OTel/Logfire，与 durability 无关 | 缺 durability 文档：三档时序表、fail-closed（sink 失败可 `errors.Is` 归因）、Checkpoint{Seq,Node,Output} 契约、以及「恢复/重放未交付」的明示 |
| 流式协议（`run.StreamMode` 七模式、六新事件 `node_started/node_completed/task_error/checkpoint/state_update/custom` 规范 JSON、`RunStreamMode` 选择器） | guide / api | **部分覆盖**。既有流式内容只覆盖 `RunStream`（StreamMessages 活的旧形态）；切片 23 的协议层无文档 | 缺协议参考：七模式清单与序数、六事件 wire 名称、`ErrUnsupportedStreamMode` fail-closed 语义（未接线模式启动前报错、绝不静默降级零事件流）；`RunStream` = 纯包装需一句话讲清两者关系 |
| HITL Interrupt/Resume（G7，切片 27 契约起草中） | guide | **无页面**（正确状态：未落地不写） | 落地后才建 guide/human-in-the-loop.md；现在最多在 roadmap 处留一行「规划中」并链到母约 §9 |
| G5 Send + AddJoinSend（切片 26 契约起草中） | guide | **无页面** | 同上，落地后并入图引擎 guide 的「动态扇出」一节 |
| G8 Store 长期记忆（D2 材料待定） | guide / api | **无页面**；knowledge.md 与 memory.md 已覆盖相邻能力 | 落地后再写，且需与 knowledge/vectordb 划界 |
| 性能/实测叙事 | advanced / blog | blog/ai-agent-runtime-benchmark.md 已有三框架矩阵（HNO c=1 均值 1.583ms / 631.71 RPS / 12.2MB RSS；c=8 1.859ms / 4,186.08 RPS；c=32 6.703ms / 3,627.35 RPS / 16.7MB）；advanced/performance.md、system-overhead.md 在册 | 图引擎自身的零锁证据（`sync.`=0、变异矩阵杀红数、-race）散落在 docs/design，未转成面向用户的叙事 |

导航与注册现状（`.vitepress/config.mjs`）：
- en 根 locale 与 zh 各有完整 nav + sidebar；**ja/ko 没有 Blog nav、没有 blog 目录**。
- zh 站 guide/api/examples/blog 与 en 全量镜像；ja/ko 侧栏已经落后（缺 sandboxed-file-io、code-execution-sandbox 等新页）。
- nav 右上角版本下拉仍写 **v2.0.0**，链接 CHANGELOG —— v3 发布时需要一并更新（登记进 §4）。
- blog/index.md 用「`## Articles` 下每篇一小节（链接 + 摘要 + Category/Tags）」的方式注册文章；侧栏再各加一条。

---

## §2 文档更新清单

### 纪律（先立规矩）

1. **不写未落地的能力。** 落地一片、写一片（fail-closed 文档纪律）：文档只描述能跑的字节；
   未交付行为（resume/replay、六个待接线 StreamMode 的运行期语义、Send、HITL、Store）
   一律不写「如何使用」，至多在页面末尾「Roadmap」一节一句话 + 链母约对应节。
2. **fail-closed 行为要写成一等公民。** `RunStreamMode` 对未接线模式启动前报错、缓存只缓存成功结果、
   Checkpointer 失败 `Run` 不交出结论、`WithStepLimit(0)` 被 Validate 拒绝 ——
   这类「不静默降级」的语义正是本文档线与竞品文档的差异点，每页至少有一处明示。
3. **代码示例只用真实导出 API**（本计划 §2 各页已给骨架），示例须过 `go vet`；
   引用数字必须能在 docs/design 证据链里找到出处，不造数。
4. **en 先行，zh 跟进，ja/ko 延后**（详见 §5）。zh 页不是机翻腔，按 zh/blog 既有行文习惯重写。

### Phase P-now（素材已冻结，可与切片 25–27 并行执行）

- [ ] **D1 `website/guide/graph-engine.md`**（新建，en，约 250–320 行）
  核心概念页：单消费者零锁模型（生产者发 channel、消费者独占可变状态，`grep -o 'sync\.' graph.go scheduler.go | wc -l` = 0）；
  builder API 走真实签名：
  ```go
  g := graph.New(
      graph.WithStepLimit(500),      // 默认 1000；0/负数 Validate 连声明一起拒
      graph.WithMaxConcurrency(8),   // 0/负数 = 不限；打满进 FIFO 等待队列
  )
  g.AddNode(graph.NodeFunc("fetch", fn))           // 或 graph.Typed[TIn, TOut]("name", fn)
  g.AddEdge("fetch", "process")
  g.AddConditional("route", "big", pred)           // Predicate func(out any) bool
  g.AddDefault("route", "small")
  g.AddJoin([]string{"a", "b"}, "merge")           // 前驱 <2 构建期点名拒绝
  g.SetEntry("fetch")
  g.SetOutput("merge")
  err := g.Validate()                               // 悬空边/重复名/不可达/环/join 不足
  res, err := g.Run(ctx, input)                     // 步数阀 ErrStepLimitExceeded、panic 屏障、取消归一
  ```
  含 Warnings() 与 Validate() 的分工、构建期 vs 运行期职责表。
- [ ] **D2 `website/zh/guide/graph-engine.md`**（D1 的中文版，同结构）
- [ ] **D3 `website/guide/node-policies.md`**（新建，en，约 200–260 行；zh 镜像 D4）
  四策略逐节：RetryConfig{MaxAttempts, InitialDelay, BackoffFactor, MaxDelay, Jitter, ShouldRetry}、
  TimeoutConfig{Timeout, PerAttempt}、CacheConfig{KeyFunc, TTL, Store}（CacheStore 接口 `GetAny/SetAny`，
  适配缝留给调用方、graph 不依赖 internal/cache；fail-closed：失败不进缓存）、
  TraceConfig{Enabled, RedactIn, RedactOut, Hook func(NodeEvent)}（NodeEvent{Node, Attempt, Err, In, Out}，
  消费者串行发射、零锁不破；RedactIn/RedactOut 控制脱敏）。
  零值语义单独成表：guard 钳制、不进 Validate。
- [ ] **D5 `website/advanced/graph-durability.md`**（新建，en，约 150–200 行；zh 镜像）
  Durability 三档时序表（Sync：complete 后下一轮 dispatch 前同步落，零值即最安全档；
  Async：后台冲刷 goroutine、与引擎只共享 commits 通道、Run 返回前冲刷完毕；
  Exit：只积累、退出一次落齐）；Checkpoint{Seq, Node, Output}、Seq 消费者盖章从 1 连续递增；
  Checkpointer 失败 fail-closed；**明示 resume/replay 未交付（S24-SPEC-1，挂至 P5/G7）**。
- [ ] **D6 `website/api/graph.md`**（新建，en，约 200 行）
  按 `website/api/workflow.md` 的「Signature → Parameters → Example」体例，逐符号过
  `go doc ./pkg/hno/graph`（导出符号恰 41：Graph/Node/NodeFunc/Typed/Option 族/NodeOption 族/
  Result/NodeEvent/Checkpointer/Checkpoint/Durability/CacheStore/ErrStepLimitExceeded）。
- [ ] **D7 `website/api/run-events.md`**（新建，en，约 180 行）
  七模式清单与序数（StreamValues/StreamUpdates/StreamMessages/StreamTasks/StreamCheckpoints/StreamDebug/StreamCustom）、
  六事件 wire 名称（node_started/node_completed/task_error/checkpoint/state_update/custom）与构造器
  （NewNodeStartedEvent(runID, node, attempt, input) 等）、ErrUnsupportedStreamMode fail-closed 契约、
  `Agent.RunStreamMode(ctx, input, modes...)` 与 `RunStream`（纯包装）的关系。
  明示：今天只有 StreamMessages 有活的 token 流；其余六模式为协议占位，生产者片未接线。
- [ ] **D8 导航注册**：`config.mjs` en 根 locale sidebar `/guide/` 加 Graph Engine、`/advanced/` 加 Graph Durability、
  `/api/` 加 Graph、Run Events；zh locale 同步。（config.mjs 属 website 编辑，走同一批 PR。）
- [ ] **D9 交叉链接修缮**（编辑既有页，小改）：
  `guide/index.md`「Flexible Architecture」段加一行 Graph（与 Workflow 并列、注明两套 API 并存与 G10 关系）；
  `guide/workflow.md` 顶部加一段「与 Graph Engine 的关系」避免读者混淆；
  `advanced/architecture.md` 补零锁单消费者模型小节。

### Phase P-after-26/27（切片 26 Send、切片 27 HITL 落地后触发）

- [ ] **D10 图引擎 guide 增补「动态扇出 Send」一节**：`AddJoinSend(source, target)` 显式汇聚声明
  （R19-Q2 = OPT-C：构建期不经 §3.5 第 6 项、运行期屏障按 source 的 pending Send 归零判凑齐）；
  map-reduce 示例按母约 §6 新声明形态改写。前置：切片 26 合入 + 母约 §6 改写完成。
- [ ] **D11 `website/guide/human-in-the-loop.md`**（新建）：Interrupt/Resume 核心用法、session 侧车存储
  （D1 = OPT-1：v1 侧车存储 + 派生视图，不改 Session 对外 JSON）、重复 Resume 幂等、跨进程恢复示例。
  Durability 页同步撤下「resume/replay 未交付」明示、改为链接本页。前置：切片 27 合入。
- [ ] **D12 `website/api/graph.md` / `api/run-events.md` 增补**对应新导出符号。
- [ ] **D13 G10（workflow 迁移）落地时**：`guide/workflow.md` 加「底层已编译为图」一段（公共 API 不变、现有用户零改动），不重写。

### Phase P-after-G8（D2 材料定稿、G8 落地后触发）

- [ ] **D14 `website/guide/store.md` + `website/api/store.md`**（新建）：层级 namespace + 向量检索；
  必须含与 knowledge/vectordb 的分工边界表（母约 D2 的裁决内容）。前置：G8 合入 + D2 边界定稿。

---

## §3 Blog 计划

风格基准（读完三篇既有文章后定）：
frontmatter 必含 `title/description/date/lastUpdated/author: HNO Team/category/tags/head`(keywords + og:* + article:published_time + canonical)；
正文走「hook → 问题重述 → 设计/协议 → 结果表 → **What this does not prove（诚实的边界）** → 复现路径 → Continue reading」；
evidence-led、不造绝对化结论；日期用 `YYYY-MM-DD`。注册三件套：blog/index.md 的 `## Articles` 小节 +
config.mjs `/blog/` 侧栏条 + zh 镜像。category 沿用既有词表（Security engineering / Benchmarks 这类两词内大写开头）。

### 文章一（现在就能写，素材已齐、已冻结）

- **标题候选**：
  1. 「在 Go 里造一个 LangGraph：一个零锁图引擎的设计与实证」（首推）
  2. 「sync.=0：HNO 图引擎为什么不加一把锁」
  3. 「Build-time or Die：把 DAG 的错误留在 Validate」
- **category**：Go engineering；**tags**：Go, concurrency, graph, LangGraph, DAG, agent framework
- **章节骨架**：
  1. hook：Agent 编排框架的图执行器通常长成什么样，锁为什么是默认选择；
  2. 问题重述：锁保护的是什么？——答案是「可变状态的独占权」，而不是时间片；
  3. 单消费者模型：生产者只向 channel 发送、消费者独占可变状态（致谢 adk-go 先例），
     实证：`grep -o 'sync\.' pkg/hno/graph/graph.go pkg/hno/graph/scheduler.go | wc -l` → **0**，
     且 `-race` 全绿（receipt 在册 `docs/design/v3-p1-graph-status.md` §1）；
  4. 构建期校验清单：悬空边、重复节点名、不可达、环、join 前驱不足 —— 每条给一个 Validate 报错文案样例（可归因文案是刻意设计）；
  5. 运行期护栏：步数阀（默认 1000）、panic 屏障、取消归一、max-concurrency FIFO、join 屏障；
  6. 工程纪律怎么长出信任：变异测试矩阵 —— 策略片 8 条杀红 + e1 等价登记、协议片 m1–m6 全杀红、
     durability 片 m1–m8 全杀红；RED→GREEN 双回执制度（引 `v3-red-observation-*.md`）；
  7. 结果表（可引既有 benchmark）：同一本地 stub 协议下 HNO c=1 均值 **1.583ms / 631.71 RPS / 12.2MB RSS**，
     c=8 **1.859ms / 4,186.08 RPS**，c=32 **6.703ms / 3,627.35 RPS / 16.7MB**
     （对照 LangGraph c=1 7.687ms / 129.68 RPS、Agno 41.632ms / 24.01 RPS；照抄原文的「这证明什么/不证明什么」边界声明）；
  8. What this does not prove：单消费者模型不适用于所有拓扑；benchmark 是冷启动生命周期口径；
     Send/动态扇出当时未交付（写作时如实写「后续文章覆盖」）；
  9. Reproduce it：`go test ./pkg/hno/graph/... -race` + benchmark 复现命令 + 指向 guide/graph-engine.md；
  10. Continue reading：链接文章二、api/graph.md。
- **前置条件**：无新依赖（全部素材已冻结在 docs/design）；**目标读者**：Go 后端工程师、框架作者。
- **预估规模**：300–400 行。

### 文章二（现在就能写）

- **标题候选**：
  1. 「节点策略四合一：Retry/Timeout/Cache/Trace 的一次 variadic 设计」（首推）
  2. 「零值即语义：策略配置里的 guard 钳制」
- **category**：API design；**tags**：Go, API design, retry, cache, timeout, observability
- **章节骨架**：
  1. hook：给图的节点加策略，为什么不该再加一个策略参数爆炸版 Config；
  2. 变参 NodeOption：`AddNode(n, WithRetry(...), WithTimeout(...), WithCache(...), WithTrace(...))`，
     对既有单参调用源零破坏（沿 WithMaxConcurrency 先例）；
  3. 逐策略一节（Retry 借鉴 adk retry.go、Cache/Timeout 借鉴 LangGraph CachePolicy/TimeoutPolicy —— 命名出处照实写）；
  4. 零值即语义表：`MaxAttempts<1` 即 1 次、`Timeout<=0` 即不限时、
     「不重试/不限时/不缓存」都是零值直接给出的语义，引擎不为策略新增构建期拒绝（对比 WithStepLimit 的 0 拒绝，讲清两种口径的取舍）；
  5. fail-closed 的缓存：只缓存成功结果，失败绝不进缓存；
  6. trace 为什么放消费侧：Hook func(NodeEvent) 由消费者 goroutine 串行发射，零锁不破；RedactIn/RedactOut 脱敏；
  7. 变异矩阵怎么抓住策略 bug：策略片 8 条杀红的片段式讲法（讲一两条最有代表性的）；
  8. What this does not prove / 边界：策略在生产者 goroutine 内生效的重入语义、CacheStore 适配留给调用方；
  9. Reproduce + Continue reading：guide/node-policies.md。
- **前置条件**：无（切片 22 已收口，证据链 `v3-p2-graph-g3-green.md`）；**目标读者**：库作者、平台工程师。
- **预估规模**：250–320 行。

### 文章三（切片 27 落地后再写；26 落地可提前作为素材并入文章一增补版）

- **标题候选**：
  1. 「HITL：把人放回回路 —— Interrupt/Resume 的 durability 之约」
  2. 「审批按钮按下去之前：跨进程恢复一个正在跑的图」
- **category**：Agent engineering；**tags**：human-in-the-loop, HITL, checkpoint, resume, agent
- **章节骨架**（落地后按实际 API 校准）：
  1. hook：为什么「暂停等审批」比看起来难（进程可能死掉、图状态在内存、事件序号要可信）；
  2. 三档 Durability 与 Interrupt 的关系（Sync/Async/Exit 各自给 resume 留了什么）；
  3. session 侧车存储：D1 = OPT-1 的取舍叙事 —— v1 侧车 + 派生视图、对外 JSON 逐字节不变；
  4. Interrupt/Resume 核心用法与幂等（重复 Resume）；map-reduce/Send 场景下的恢复；
  5. 实测数字：Seq 消费者盖章、append-only 条目流；变异矩阵 m1–m8；
  6. What this does not prove；7. Reproduce；8. Continue reading：guide/human-in-the-loop.md。
- **前置条件**：**切片 27 合入（硬前置）**，建议等 Durability resume/replay（S24-SPEC-1 随 G7）一并可用；**目标读者**：生产落地团队。
- **预估规模**：280–350 行。

发布节奏建议：文章一、二间隔 1–2 周先后发（互相引流）；文章三随切片 27 的 release note 同周发。

---

## §4 执行清单

顺序即依赖；「并行」= 可与切片 25–27 的进行中工作同时执行（docs 代理不改代码、不改 website 之外的任何文件）。

| # | 产出物 | 前置 | 预估规模 | 可并行 |
|---|---|---|---|---|
| 1 | D1 en 图引擎 guide（guide/graph-engine.md） | 无 | 250–320 行 | ✅ 并行 |
| 2 | D3 en 策略 guide（guide/node-policies.md） | 无 | 200–260 行 | ✅ 并行 |
| 3 | D5 en durability（advanced/graph-durability.md） | 无 | 150–200 行 | ✅ 并行 |
| 4 | D6 en api/graph.md | 无 | ~200 行 | ✅ 并行 |
| 5 | D7 en api/run-events.md | 无 | ~180 行 | ✅ 并行 |
| 6 | D9 交叉链接修缮（guide/index.md、workflow.md、architecture.md 小改） | 1–5 定稿后做，链接才不断 | 每文件 <20 行 | ✅ 并行（靠后） |
| 7 | Blog 文章一（en + 侧栏/首页注册 + canonical） | 无（素材冻结） | 300–400 行 | ✅ 并行 |
| 8 | Blog 文章二（en + 注册） | 无（素材冻结） | 250–320 行 | ✅ 并行 |
| 9 | D2/D4 zh 镜像（guide 两篇 + durability + blog 两篇 zh 版） | 对应 en 定稿 | ≈en 规模 ×4 | en 定稿后 ✅ 并行 |
| 10 | D8 config.mjs 导航注册（en+zh sidebar、blog 侧栏、zh/blog 侧栏） | 1–5、9 就位 | ~30 行配置 | 收口批 |
| 11 | D10 Send 章节（graph-engine.md 增补 + api/graph.md 增补） | **切片 26 合入** | ~80 行增补 | ❌ 等 26 |
| 12 | D11/D12 HITL guide + API 增补 + durability 页撤「未交付」注 | **切片 27 合入** | 200–260 行 | ❌ 等 27 |
| 13 | Blog 文章三（en + zh + 注册） | 切片 27 合入 | 280–350 行 | ❌ 等 27 |
| 14 | D14 Store guide + api（含 knowledge 分工表） | **G8 合入 + D2 边界定稿** | 2×~180 行 | ❌ 等 G8 |
| 15 | D13 workflow 页「已编译为图」注记 | **G10 合入** | ~15 行 | ❌ 等 G10 |
| 16 | config.mjs nav 版本下拉 v2.0.0 → v3.0.0（含 CHANGELOG 链接核对） | v3 发布流程启动 | ~6 行 | 发布批 |

执行注意：
- 每个网站 PR 过 `npm run docs:build`（website/ 内）再交付；`ignoreDeadLinks: true` 是兜底不是借口，内链尽量手工核对。
- 引用数字一律回指 docs/design 证据文件；blog 里出现回执号时用 `v3-p1-graph-status.md` 在册的真实 id，不虚构。
- 文档代理只写 `website/**` 与本计划文件，代码示例不进 `cmd/examples/`（如后续要配 `cmd/examples/graph_demo`，
  那是代码切片的事，届时 examples 页再加 Location/Run 段，体例对齐 `website/examples/index.md`）。

---

## §5 翻译与导航

现状（实测）：
- **ja/ko 没有 blog 目录、没有 Blog nav**；blog 翻译的既有惯例就是 **en + zh 双语**，ja/ko 不做。
  → 本计划所有 blog 文章：en 正稿 + zh 镜像，**ja/ko 明确 out-of-scope**（deferred，与既有三篇口径一致）。
- ja/ko 的 guide/api 侧栏已落后于 en（缺 sandboxed-file-io、code-execution-sandbox、workflow-history 等），
  说明 guide 级翻译本来就允许滞后。→ 新增 guide/api 页面采用 **en 先行、zh 跟进、ja/ko 不排期**；
  zh 是唯一承诺跟进的镜像（zh 站 guide 全量镜像的现状维持住）。
- 注册清单（每篇 blog 三处 + 新页面两类）：
  1. `website/blog/index.md` 的 `## Articles` 下加小节（链接 + 摘要 + Category/Tags，照既有格式）；
  2. `website/.vitepress/config.mjs` en 根 locale sidebar `'/blog/'` 加条目；
  3. zh 镜像：`website/zh/blog/<slug>.md` + `website/zh/blog/index.md` 小节 + config.mjs `'/zh/blog/'` 条目（canonical 指向 `/zh/blog/<slug>`）；
  4. 新 guide/api/advanced 页面：en 正文 + zh 正文 + config.mjs 对应 locale sidebar 各一条。
- **版本导航**：config.mjs 各 locale nav 的版本下拉均为 `v2.0.0`。v3 发布时统一改 `v3.0.0` 并核对
  CHANGELOG/Release Notes 链接 —— 已登记 §4 #16，属发布批而非文档批。
- RSS（`/rss.xml`）由构建管线产出，blog 新文章无需手工登记（在 blog/index.md 与侧栏登记即可）。
