# D1 裁决材料 — G7 会话事件化是否改动 `Session` 对外 JSON

> 决策点登记：`docs/design/v3-platform.md` §9.3（契约层影响 ⚠️）与 §12 D1（另涉 §10 P5 行、§11 BC-4、§13 风险登记行）
> 本材料只登记决策点原文、今日实测与各选项后果；**裁决权在负责人**，本文不代作决定，也不因本文改动任何 fixture、判据或存储代码。
> G7（P5 阶段）的切片契约以「P0 + 决策 D1」为前置（母约 §10 :532）：**负责人给出裁决结论前，G7 契约不得起草**。
> 本文件的写入是本轮唯一的仓库改动；实验全程在 `/tmp/d1probe`，仓库工作树除本文外逐字节未动。

## 1. 决策点原文与实测

### 1.1 决策点原文

**其一，母约 §9.3**（`docs/design/v3-platform.md:512–519`）：

> `internal/session/contract` 有 9 个 Go↔Python fixture 对齐测试。**事件化会改变 session 存储形态**。
> **应对**：
> - v1：**不改 `Session` 的对外 JSON 结构**，事件流作为**新增的内部存储 + 派生视图**
> - 若必须改，需同步更新 `internal/session/contract` 的 fixture，并评估是否破坏 agno-python 互操作
> - **这是一个需要显式拍板的决策点（§12 D1）**

**其二，母约 §12 D1**（`docs/design/v3-platform.md:559`）：

> | **D1** | 会话事件化是否改动 `Session` 对外 JSON？ | (a) 不改，事件为内部存储+派生视图（安全，但契约层无感）<br>(b) 改，同步更新 Python fixture（暴露能力，但可能破坏互操作） | **(a) 先做 a**，b 作为独立任务 |

**其三，配套登记**：§11 BC-4（:550）「`session.Session` 新增事件流 —— 若改 JSON 结构则影响契约层；见决策 D1；v1 默认不改对外结构」；§13 风险行（:573）「契约层被事件化破坏（中）—— D1 决策；P5 阶段契约测试必须全绿才可合入」。

本材料的「对外 JSON」按 BC-4 的措辞取全称：既指 `pkg/hno/session.Session` 结构体（`pkg/hno/session/session.go:11–42`，json tag：`session_id`/`agent_id`/`team_id`/`workflow_id`/`user_id`/`name`/`metadata`/`state`/`agent_data`/`runs`/`summary`/`created_at`/`updated_at`），也指 `internal/session`（AgentOS Go↔Python 对齐层）经 HTTP 产出的会话 JSON —— 后者是 9 个 fixture 真正钉住的面（§1.3）。

### 1.2 互操作基线（今日实测，含一条诚实读数）

| 命令 | 观察 | 回执 |
|---|---|---|
| `go test ./internal/session/contract/... -count=1` | exit 0 | `receipt:6738b4c9-7bda-457a-878b-7ea10b050735` |
| 同上 + `-v` | exit 0，且 **9/9 个测试 SKIP** | `receipt:a6001600-5c46-4268-b0fb-f5776528b662` |

基线的关键事实不是「绿」，而是**为什么绿**：fixture 不在仓库里。加载器用 `runtime.Caller(0)` 解析
`../../../../contract-fixtures`（`internal/session/contract/contract_test.go:368–379`，skip 落在 ：375），
本机解析为 `/Users/rex/codes/contract-fixtures` —— 该目录不存在；`git log --all -- contract-fixtures`
零提交，`.gitignore` 也不含该条目（仓库历史里从未有过）。所以今天的「契约测试绿」是 **skip 绿**：
`exit 0` 不等于对齐已被验证。这与 `docs/design/v3-baseline-evidence.md:33`（契约测试数 9）、`:122`
（「无 fixture 时 skip」）的登记相符。

对本案的直接后果：**OPT-2 所说的「同步更新 fixture」在本仓库内没有可单方面编辑的对象** —— fixture 由
agno-python 侧提供/再生成，同步动作必然是一次跨仓库协作；在 fixture 恢复供给之前，任何 v2 改动都发生在
没有护栏的桥上。

### 1.3 fixture 清单（9 个测试 → 9 个文件，各自钉住什么）

9 个测试逐一以 **HTTP 响应 JSON 与 fixture 的全等**（`assertJSONEqual` → `reflect.DeepEqual`，
`contract_test.go:381–388`）钉住对外形状：

| # | 测试（`contract_test.go`） | fixture 文件 | 钉住的 JSON 字段 |
|---|---|---|---|
| 1 | `TestListSessionsMatchesFixture` :25 | `get_sessions_agent.json` | `data[]`：`session_id`/`session_name`/`session_state`/`created_at`/`updated_at`；`meta`：`page`/`limit`/`total_count`/`total_pages`（测试侧锚 ：503–517） |
| 2 | `TestGetSessionDetailMatchesFixture` :48 | `get_session_detail_agent.json` | `user_id`/`session_id`/`session_name`/`session_state`/`session_summary`/`agent_id`/`agent_data`/`metrics`/`metadata`/`chat_history`/`created_at`/`updated_at`（测试侧锚 ：519–532） |
| 3 | `TestCreateSessionMatchesFixture` :73 | `create_session_agent.json` | `user_id`/`session_id`/`session_name`/`session_state`/`metadata`/`agent_id`/`created_at`/`updated_at`（测试侧锚 ：534–543） |
| 4 | `TestRenameSessionMatchesFixture` :105 | `rename_session_agent.json` | 同 detail 形状（复用 `detailFixture`） |
| 5 | `TestGetSessionRunsMatchesFixture` :131 | `get_session_runs.json` | **不透明 run map 数组的整体全等**（:150–153，数组里每个键都参与 DeepEqual） |
| 6 | `TestListSessionsInvalidTypeMatchesErrorFixture` :156 | `get_sessions_invalid_type_error.json` | 400 错误体全等 |
| 7 | `TestGetSessionDetailNotFoundMatchesErrorFixture` :178 | `get_session_not_found_error.json` | 404 错误体全等 |
| 8 | `TestListSessionsDatabaseRequiredMatchesErrorFixture` :203 | `list_sessions_database_required_error.json` | 400 错误体全等 |
| 9 | `TestListSessionsDatabaseNotFoundMatchesErrorFixture` :236 | `list_sessions_database_not_found_error.json` | 404 错误体全等 |

测试侧最小锚示例（`detailFixture`，`contract_test.go:519–532` —— fixture 缺席时它就是形状的仓库内唯一定义）：

```go
type detailFixture struct {
	UserID         string         `json:"user_id"`
	SessionID      string         `json:"session_id"`
	SessionName    string         `json:"session_name"`
	SessionState   map[string]any `json:"session_state"`
	SessionSummary map[string]any `json:"session_summary"`
	AgentID        string         `json:"agent_id"`
	AgentData      map[string]any `json:"agent_data"`
	Metrics        map[string]any `json:"metrics"`
	Metadata       map[string]any `json:"metadata"`
	ChatHistory    []any          `json:"chat_history"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}
```

要点：#1–#5 钉的是**成功路径的会话形状**（顶层键集合参与全等）；#6–#9 钉的是错误路径（与事件化正交）。
**任何顶层新增键都会让 #1–#4 的全等失败；往 run 载荷里加键会让 #5 失败。**

### 1.4 marshal 边界的结构性实测（「不能变的面」一共有几个、在哪）

代码走查测得，会话 JSON 从存储到对外一共四个边界，**全部是显式逐字段投影，只有一处例外**：

1. **internal/session 的 HTTP wire（fixture 真正钉住的面）**：`buildSessionDetail`
   （`internal/session/service/service.go:407–494`）、`toListItem`（:390–398）、响应结构体
   （:81–101）—— 手工逐字段拼 `map[string]any`/结构体，**不是**对 `dto.SessionRecord` 的整包 marshal。
   新增存储字段不会自动漏到这里；只有改这三个投影点才会改 fixture 面。
2. **internal/session 的持久化**：按列 JSONB（`internal/session/store/postgres/store.go:103–158` 的列清单，
   `encodeJSON` 在 ：528–537）；`runs` 列的类型是**不透明** `[]map[string]any`（`internal/session/dto/types.go:88`），
   存取两侧都不解释内容。
3. **`pkg/hno/session` 的持久化**：postgres/sqlite/batch 全部按列显式 marshal（`pkg/hno/session/db/postgres/storage.go:304–340`
   的 `buildArgs`、sqlite `marshalJSON` :483–489、batch `db/batch/postgres.go:206–210`）；mongo 是字段显式
   mapper（`toMongoSession`，`db/mongo/storage.go:353–372`）。**唯一一处整结构体 marshal 在 surreal**：
   `sessionToPayload`（`pkg/hno/session/db/surreal/storage.go:347–369`，:352 `json.Marshal(copy)`）——
   若 `Session` 结构体新增字段，全仓库只有这一处的持久化字节会**自动**携带它。
4. **AgentOS 的 HTTP 派生投影**：`SessionResponse`（`pkg/agentos/session_handlers.go:35–49`）经
   `sessionToResponse`（:545）显式挑字段，`buildSessionRuns`（:563–588）对 run 只挑
   `run_id`/`status`/起止时间/`cancellation_reason`/`cache_hit` —— 结构体加字段不自动漏出。

**先例（对本案很重要）**：`RunOutput.Events` 已经存在 —— `json:"events,omitempty"`
（`pkg/hno/agent/agent.go:76`）。即 run 级事件**今天已在** `runs` 列的持久化字节里，空值不可见
（`omitempty`），且所有 HTTP 投影都不浮现它。仓库里已有一条被验证的「事件在持久化字节中、对外无感」的共存路径。

**消费者清点（OPT-2 的爆炸半径，grep 实测）**：

- `internal/session` 的仓库内导入方：**2 个**（`cmd/agentos-session/main.go`、`internal/http/router/router.go`）；
- `pkg/hno/session` 的仓库内导入方：**7 个文件**（`cmd/agentos/main.go`、`cmd/examples/surreal_demo/main.go`、
  `pkg/agentos/{server,server_test,ops_test,agent_handlers,session_handlers}.go`）；
- `pkg/hno/session` 之外的**整包** `json.Marshal(session)` 调用点：**0 个**（唯一一处就是上边界的 surreal 包内 ：352）；
- 软消费者：`website/guide/session-service.md`、`website/guide/session-state.md`（含 zh/ja 镜像）文字性描述了
  会话 JSON 形状 —— v2 改形状时文档要跟着改，v1 不用。

### 1.5 v1 形状可行性探针（`/tmp/d1probe`）

探针不拷贝源码，而是 `replace github.com/rexleimo/agno-go => /Users/rex/codes/agno-go` 后**直接 import 真实的
`pkg/hno/session` 与 `pkg/hno/agent` 类型**（仓库只读），事件类型按 §9.1 的七类取了探针本地替身（G7 契约未起草，
`Kind` 用字符串而非 int 枚举；被测的字节命题不依赖这个选择）。命令：
`node /tmp/rex-receipt.mjs -- go -C /tmp/d1probe run ./cmd/probe` → **`receipt:cc4bb798-9033-4412-bf73-9598a0ff2cca`**（exit 0）。输出逐行：

```
== [A] 今日形状 marshal(session) = 290 bytes
== [B] v1 形状（两条事件已入旁路存储）marshal(同一 session 值) = 290 bytes
== [B] bytes.Equal(A,B) = true（事件追加对 pinned 视图零字节影响）
== [B2] 对照：store.Update 本身（与事件无关）就改变字节 = true（UpdatedAt 每次触摸重打）
== [C] 事件流独立持久化 = 247 bytes, 往返后 2 条, kinds=[input model_response]
== [D] 派生视图 deriveRuns(events) = 1 条 run, derived[0].Content="hello", RunID="run-1"（与原 runs 等价）
== [E] v2 对照 marshal(结构体级新增) = 547 bytes, 含 "events": 键 = true（整包字节必然变化）
```

五条读数：**[A]/[B]** 事件进旁路存储后，同一 `Session` 值的 marshal 字节逐位不变 —— v1 的核心命题成立；
**[B2]** 附带实测 `MemoryStorage.Update` 每次触摸都重打 `UpdatedAt`（`pkg/hno/session/memory_storage.go:102`），
即 v1 落地时事件追加**不应**路由过 session 行更新（否则每条事件都无谓改写会话行）；**[C]** 事件流可独立
持久化并往返；**[D]** 事件流可投影回 `runs` 派生视图；**[E]** v2 对照 —— 结构体级新增键必然改变整包字节。

过程披露：本探针第一版曾把事件追加与 `store.Update` 混在同一步，得到 `bytes.Equal = false` 的假阴性
（回执 `receipt:d3a7ca5e-c535-4549-a56c-6b380a10196e`，已废弃）—— 假阴性根因正是 [B2] 的 UpdatedAt 重打，
与事件无关。拆开重测后才有上表。两个回执 id 均为 `/tmp/rex-receipt.mjs` 实际打印。

## 2. 选项

### OPT-1（v1，母约原案 a）内部存储 + 派生视图

- **语义**：`Session` 对外 JSON 结构零改动；G7 事件流住在新增的内部存储（独立事件列/表，或 `MemoryStorage`
  式的旁路 map），`RunOutput` 保持为事件流的派生视图（母约 §9.1 的向后兼容承诺）。
- **实现代价**：中 —— 事件存储本体（一种新容器 + 追加/读取）+ 事件→`Runs` 的投影函数；**不碰** §1.4 的四个
  边界中任何一个。唯一要守的纪律是 [B2] 的：事件追加不经过 `Update`（或接受 updated_at 被重打的语义并写进契约）。
- **对 9 个 fixture 的影响**：**零**。probe [A]/[B] 字节级证明。
- **对 agno-python 互操作的影响**：**零风险**（不改任何对外字节，无需 python 侧核对）。
- **对未来 HITL 的支撑度**：够用但受限 —— `EventInterrupt`/`EventInterruptResponse` 持久化在内部事件存储上，
  跨进程 Resume（§9.2 的 schema 校验/幂等/Rerun-Handoff）只要 Go 侧读写同一存储即可成立；**代价是事件不进对外
  契约**：跨进程的非 Go 消费者（含 agno-python 的 AgentOS 面）要靠派生视图或显式 opt-in 的新端点才能读到事件流
  与 pending Interrupt —— 那将是新的对外面积，仍需走一次契约变更（只是不必现在做）。

### OPT-2（v2，母约原案 b）改 JSON + 同步 fixture

- **语义**：事件成为对外契约一等公民：`Session` 对外 JSON 新增事件字段，HITL Interrupt 的持久化直接落在对外结构上。
- **实现代价**：大 —— `pkg/hno/session/session.go` 结构体 + 五个 db 实现的列/映射（postgres/sqlite/batch 加列、
  mongo 加 mapper 字段、surreal 自动携带但仍要读回映射）+ `internal/session` 侧 `buildSessionDetail` 投影扩展 +
  fixture 全量重生成 + 网站文档同步。
- **对 9 个 fixture 的影响**：#1–#5 的全等**必然失败**（顶层新增键 / run 载荷加键），需要从 agno-python 侧再生成
  并回填；#6–#9 错误体不受影响。即「9 个里动 5 个，且 5 个都不是本仓库能单方面动的」。
- **对 agno-python 互操作的影响**：**破坏风险需要 python 侧核对，本仓库无法单方面担保**。母约 §9.3 自己的措辞
  就是「评估是否破坏」—— 评估动作本身在对方仓库；在 fixture 供给恢复之前（§1.2），连「改了之后契约测试是否还绿」
  都无法在本仓库内验证。
- **对未来 HITL 的支撑度**：最好 —— Interrupt 持久化在对外结构上，任何语言、任何进程都能直接读到 pending
  中断与事件流；这是 v2 唯一实质收益。

### OPT-3（证据支持的第三条路）事件随既有 `RunOutput.Events` 通道

- **语义**：不加顶层字段，也不建独立事件存储 —— G7 事件作为 `RunOutput.Events` 的元素随**既有 `runs` 列**持久化
  （`pkg/hno/agent/agent.go:76` 已存在的 `json:"events,omitempty"`；`internal/session` 的 `runs` 列是不透明
  `[]map[string]any` 透传，`dto/types.go:88`）。
- **证据**：§1.4 的先例 —— run 级事件今天已经这样活在持久化字节里而不破任何投影；两个存储面的 `runs` 列都不解释
  内容；HTTP 投影（`buildSessionRuns`）逐字段挑拣，`events` 不浮现。
- **实现代价**：最小 —— 无新表/新列，事件追加写进当前 run 的 `Events` 再走既有 upsert。
- **对 9 个 fixture 的影响**：**零**（前提写死在契约里：HTTP 响应路径不得把 `events` 加进 `runs` 输出 —— 现状
  已满足）。
- **对互操作的影响**：零（同 OPT-1）；若未来希望 python 侧**写**事件，走不透明通道理论上可行，但那是又一层的
  隐式契约，不应在本案预设。
- **边界（为什么它不是 OPT-1 的完全替代）**：事件粒度被绑到 run 边界 —— 跨 run 的会话级事件只能挂在「当前 run」
  上（Interrupt 发生于 run 中途，可挂当前 run，语义成立但不干净）；独立事件存储（OPT-1 原案）才有会话级完整
  时序的一等表达。另外 `Events` 的元素形状（`run.BaseRunOutputEvent` 接口 + 自定义 JSON）与 §9.1 设想的
  `session.Event` 不是同一类型，复用通道不等于复用类型。

## 3. 推荐与理由

**推荐 OPT-1（v1）**，与母约 §12 的既有倾向（「先做 a，b 作为独立任务」）一致；OPT-3 可作为 OPT-1 落地时
「旁路存储物理形态」的候选（独立事件列/表 vs 复用 runs 通道），这个子选择留给 G7 切片契约起草时定，不属于
D1 本身的裁决面。理由：

1. **fixture 紧密锚定互操作形状，而锚的供给不在本仓库**：9 个测试全是整体全等（§1.3），顶层加键即碎；
   且 §1.2 实测 fixture 目录在仓库与仓库历史上都不存在 —— OPT-2 的真实成本不是「改 9 个文件」，
   而是「一次跨仓库契约变更 + agno-python 侧的破坏风险评估」，两者本仓库都无法单方面完成或担保。
2. **基线是 skip 绿，护栏暂时不在**：在 fixture 恢复供给之前做 v2，改动将处于「改完无法验证对齐」的状态
   （§1.2 的诚实读数）；v1 不需要护栏（字节零变化，probe [A]/[B] 已证）。
3. **仓库已有同形先例**：`RunOutput.Events`（omitempty + 投影不浮现）证明「事件进持久化字节、对外无感」
   在本仓库是已验证的共存方式；v1 只是把同样的纪律升到会话级。
4. **HITL 的需求 v1 承载得住**：跨进程 Resume 的三个语义（§9.2）只要求持久化可靠，不要求事件进对外契约；
   「事件/Interrupt 上浮为对外契约」留给一次显式的 v3.1 契约变更（届时 fixture 供给、python 侧核对、
   `get_session_runs` 全等的放宽策略一并上桌），这正是「b 作为独立任务」的本意。

**权属声明**：裁决权在负责人。本材料只登记决策点与实测，**不擅自改任何 fixture、投影或存储代码**
（本文件的写入是本轮唯一的仓库改动；探针在 `/tmp/d1probe`，不入库）。在负责人给出 D1 结论前，
G7（P5）切片契约不得起草；本文亦不预写任何一方落地后的文案或测试期望。

## 4. 登记

- 本材料对应母约 §12 D1（`docs/design/v3-platform.md:559`）；§9.3（:512–519）的三行「应对」即本文 OPT-1/OPT-2
  的原文出处；§11 BC-4（:550）与 §13 风险行（:573）随裁决结论一并由承接片写回，本文不代改。
- 状态文档 §5 的指针由主代理维护，本文不代管、不代改。
- 回执登记（均为 `/tmp/rex-receipt.mjs` 实际打印）：基线 `receipt:6738b4c9-7bda-457a-878b-7ea10b050735`（exit 0）、
  `receipt:a6001600-5c46-4268-b0fb-f5776528b662`（-v，exit 0，9/9 SKIP）；探针
  `receipt:cc4bb798-9033-4412-bf73-9598a0ff2cca`（exit 0）；废弃 `receipt:d3a7ca5e-c535-4549-a56c-6b380a10196e`
  （第一版假阴性，§1.5 已披露根因）。
- 独立性披露：本文为单代理自查材料（取证、实验、成文同一代理），未派发独立审查；仓库内测试仅在
  `./internal/session/contract/...` 上执行（本轮唯一测试命令面）。

## 5. 裁决记录

**负责人 2026-09-28 选 OPT-1**（v1：不改 `Session` 对外 JSON，事件流为侧车内部存储 + 派生视图；「事件上浮为对外契约」留作 v3.1 显式变更）。母约 §9.3/§12 已同步标注；G7 切片契约据此起草（侧车挂载点与派生视图形态由契约定）。D1 关闭。

落地指针：OPT-1 已由切片 28 实现（`pkg/hno/session/sidecar` + `internal/hitlbridge`），
8 个 marshal 边界锚逐字节不变的实测见 `docs/design/v3-p5-s28-green.md` §3。
