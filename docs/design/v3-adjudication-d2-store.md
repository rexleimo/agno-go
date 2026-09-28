# D2 裁决材料 — G8 Store 长期记忆是否进 v3.0

> 决策点登记：`docs/design/v3-platform.md` §8（:425–459，警示行 :427）与 §12 D2（:566，另涉 §2.1 G8 行 :86、§10 P8 行 :541）。
> 本材料只登记决策点原文、今日实测与各选项后果；**裁决权在负责人**，本文不代作决定，也不因本文改动任何接口、后端或测试。
> P8（G8 Store）的切片契约以「决策 D2」为前置（母约 §10 :541）：**负责人给出裁决结论前，G8 契约不得起草**。
> 本文件的写入是本轮唯一的仓库改动；实验全程在 `/tmp/d2probe`，仓库工作树除本文外逐字节未动。

## 1. 决策点原文与实测

### 1.1 决策点原文

**其一，母约 §8**（`docs/design/v3-platform.md:425–459`）。警示行（:427）：

> ⚠️ **此项引入全新概念，是 v3 中范围最大、破坏性最强的一块。建议单独决策（见 §12 决策点 D2）。**

接口草图（:429–449，逐字）：`Item{Key string; Namespace []string; Value []byte; CreatedAt, UpdatedAt time.Time}`；
`Store` 五方法 `Get / Put / Delete / List(ctx, ns) / Search(ctx, ns, query, k)`，其中 `Search` 行内注「需 Embedder」。
分工表（:451–457）：`memory`＝对话历史（进程内）；`knowledge + vectordb`＝RAG 文档库，「knowledge 是**非结构化文档**；Store 是**结构化用户事实**」；
`session.State`＝会话级 KV。后端（:459）：「v1 提供内存实现 + 复用现有 `session/db` 的 5 种后端之一（建议 Postgres）」。

**其二，母约 §12 D2**（`v3-platform.md:566`）：

> | **D2** | G8 Store 是否进 v3.0？ | (a) 进（+1,200 LOC，全新概念，范围最大）<br>(b) 延到 v3.1 | **(a) 进** —— 「能记住用户」是产品差异化，但需先定与 knowledge 的边界 |

**其三，配套登记**：§2.1 G8 行（:86）「`Store` 长期记忆（层级 namespace + 向量），参考 LangGraph，~1,200 LOC」；
§10 P8 行（:541）「G8 Store ｜ 前置 决策 D2 ｜ 长期记忆 ｜ 验收：与 knowledge/vectordb 分工清晰」。

**与 §2.2「不做」清单的对账**：§2.2（:91–97）排除的是 compaction/trigger/A2A/泛型化/Pregel-BSP，**Store 不在其中**——v3.0 做 Store 与「不做」清单无字面冲突，D2 是纯取舍题，不是自洽性修错题（与 D1/R19-Q2 的性质不同）。

**行号时效披露**：本文引用的母约行号以 2026-09-28 取证时的 `v3-platform.md` 为准（:425 §8 标题、:427 警示行、
:429–449 草图、:451–457 分工表、:459 后端行、:541 P8 行、:566 D2 行）；母约正被 v3 各片持续写回，后续读者若见漂移，以章节号（§8/§10/§12）为准。

### 1.2 基线（今日实测）

仓库内测试仅在四个相关包上执行（本轮唯一测试命令面；**未触碰 `pkg/hno/graph`**）：

| 命令 | 观察 | 回执 |
|---|---|---|
| `go test ./pkg/hno/knowledge/... ./pkg/hno/vectordb/... ./pkg/hno/memory/... ./pkg/hno/embeddings/... -count=1` | exit 0 | `receipt:9f3a00ce-1c4f-40e8-98fa-64988123b380` |
| 同上 + `-v` | exit 0，30 个测试通过（knowledge 9、chromadb 4、memory 9、openai 5、sentencetransformer 1、vllm 2），0 失败 | `receipt:94c70fdf-a43c-4291-9a6f-ed1b907099ce` |
| 同上 + `-cover` | exit 0，总覆盖 48.3% | `receipt:ea4ca0db-6425-46ab-bd3e-62642091eca6` |

诚实读数两条：① Go 1.26 的 `-v` 输出是逐包一行「PASS: 测试名清单」，SKIP 不出现在该行——chromadb 的 6 个
服务依赖用例（`chromadb_test.go:113/147/227/295/366` `t.Skip("Requires running ChromaDB instance")`）与
openai 的 `TestEmbedIntegration`（`openai_test.go:282`，`OPENAI_API_KEY` 未设即跳）**实际是 skip**，这与
chromadb 11.8% 的覆盖率（§1.3）相符；② `vectordb/redisdb` 整包挂在 `//go:build redis` 标签后
（`redisdb/redis.go:1`），上述基线**根本没编译它**，其冒烟测试还要 `TEST_REDIS_VECTORDB=1` 门控。

### 1.3 四包结构清单（go doc / wc / coverprofile 实测）

| 包 | 导出面（`go doc`） | 非测试 LOC（测试） | 语句覆盖 | 仓库内消费方（grep，不含包内自引用） |
|---|---|---|---|---|
| `pkg/hno/knowledge` | 16 项：`Document`/`Chunk`/`Loader`/`Chunker` 接口 + 5 具体类型 | 529（303） | 81.7%（175 语句） | **1**：`cmd/examples/rag_demo/main.go` |
| `pkg/hno/vectordb` | 7 项：`VectorDB`（10 方法）、`EmbeddingFunction`（2 方法）、`Document`/`SearchResult`/`DistanceFunction`/`CollectionMetadata`；`base.go` 为纯接口（0 可执行语句） | 87 base + chromadb 706 + redisdb 271（build-tag redis）＝1064（581） | base 0 语句；chromadb 11.8%（297 语句） | **8**：`rag_demo`、`internal/vectordb/migrate`×4（含测试）、`pkg/agentos/{server,knowledge_handlers}`×2（含测试） |
| `pkg/hno/memory` | 3 项：`Memory` 接口（4 方法）、`InMemory`、`NewInMemory` | 168（239） | 93.8%（48 语句） | **4**（全部在 `pkg/hno/agent`：`agent.go`、`config.go`、两个测试） |
| `pkg/hno/embeddings/*` | openai 3 项 / sentencetransformer 4 项 / vllm 3 项，三个 provider 均实现 `vectordb.EmbeddingFunction` | 414（461） | openai 71.4%、st 47.1%、vllm 75.6% | **2**：`rag_demo`、`pkg/agentos/server.go`（均只 import `openai` 子包） |

**不存在的东西同样重要**：`knowledge` 包**没有任何存储与检索**——它是 `Load() ([]Document, error)`
（`document.go:29`）与 `Chunk(doc Document) ([]Chunk, error)`（`chunker.go:13`）的文档摄取面。
仓库里现存的「RAG 检索服务」不在 `pkg/hno`，而在 `pkg/agentos.KnowledgeService`
（`knowledge_handlers.go:97` `NewKnowledgeService(vdb vectordb.VectorDB, embFunc vectordb.EmbeddingFunction, config KnowledgeServiceConfig)`）。

### 1.4 重叠实测：Store 草图对四包逐一对账（含探针）

探针在 `/tmp/d2probe`（`replace` 指向仓库、只读仓库类型）：把 §8 草图**逐字**搬为本地 `Item`/`Store`，
架在仓库真实的 `vectordb.VectorDB` 接口上跑通五方法，再对 `memory.Memory` 做实现性对撞。
`node /tmp/rex-receipt.mjs -- go -C /tmp/d2probe run ./cmd/probe` → **`receipt:54cd3a04-2acc-4d24-8410-da5c81cb6963`**（exit 0）。

| 对象 | 实测结论 | 证据（file:line / 探针读数） |
|---|---|---|
| `vectordb` | **可组合，但每个映射都是损失点**。编译期断言 `var _ Store = (*vectorStore)(nil)` 通过、五方法全链路往返成功（探针 [A1]），但：① `Item.Value []byte` → `Document.Content string`（`base.go:10`）；② 层级 `Namespace []string` 只能压平成 metadata 过滤键（`base.go:43` `Query(ctx, query, limit, filter map[string]interface{})`，而 chromadb **一实例绑一个 collection**：`chromadb.go:34-35` `Config.CollectionName`，namespace 映射不到 collection）；③ `CreatedAt/UpdatedAt` 只能塞进 `Document.Metadata map[string]interface{}`（`base.go:11`）；④ `VectorDB` **没有按过滤键列出的方法**，`Store.List(ns)` 只能借打分通道 `QueryWithEmbedding` + 假向量 + 大 limit 仿出（探针损失点 4） | 探针 [A1]/[A2] |
| `vectordb`（身份） | **跨 namespace 同名 Key 相互覆盖**。`vectordb.Document` 的唯一身份是 `ID`（`base.go:9`，`Add(ctx, documents []Document)` :33 无 upsert 语义区分）；把 `Item.Key` 直接当 `Document.ID`，alice 与 bob 各放一个 `"likes"` 后 `Count=1`（探针 [A4]，本探针第一版正是因此拿到 nil panic，实测非推演）。适配层必须用 `(ns,key)` 合成 ID——这是一条**必须写进契约的语义**，不是实现细节 | 探针 [A4] |
| `knowledge` | **重叠为零**。该包只有 Loader/Chunker 摄取面，无存储、无检索、无 KV；母约 :456 把「knowledge 是非结构化文档」列为与 Store 的边界行，实测这行对的是一个**没有存储能力的包**——真正与 Store 分抢地盘的是 `vectordb`（存储底座）与 `agentos.KnowledgeService`（检索服务面，:97） | §1.3「不存在的东西」 |
| `memory` | **结构性不可互换**。`Memory` 接口（`memory.go:12-32`）四方法 `Add(message *types.Message, userID ...string)` / `GetMessages(userID ...string)` / `Clear` / `Size`：无 `context`、无 `error` 返回、键是单个 userID 字符串而非层级 ns、值是 `*types.Message` 而非任意 KV。探针 [B] 实测 `memory.Memory` 不实现 `Store`（`Implements=false`，编译期即不可能）。母约 :455 边界行（对话历史 vs 用户事实）**与实测相符**，无需修补 | 探针 [B] |
| `embeddings` | **是注入件不是竞争者**。三个 provider 全部实现 `vectordb.EmbeddingFunction`（`base.go:59-65` 的 `Embed/EmbedSingle` 两方法；`sentencetransformer.go:70` 有显式 `var _` 断言）。§8 说 Search「需 Embedder」——仓库已有该抽象，Store **不应另起**一套 | 探针 [C]：方法数 Store=5，VectorDB=10，EmbeddingFunction=2，Memory=4 |
| `session.State`（参照） | `State map[string]interface{}`（`session.go:28`），会话作用域；与 Store 的「跨会话」边界如母约 :457 所述。**另发现命名占位**：`internal/session/store/store.go:1` 已声明 `package store` 且自带一个同名 `Store` 接口（`UpsertSession/...`，:26-32）——§8 草图的 `package store` 落位 `pkg/hno/store` 时将与它同名异义，包名/落位本身是一项待裁决 | grep 实测 |

### 1.5 §8 草图的未指明处（诚实登记：以下各点草图没有给答案）

1. **无构造器/配置面**：接口只有五方法，「需 Embedder」只写在注释里——注入点不存在。探针 [A3]：无 Embedder 时
   `Search` 只能运行期报错 fail-closed，接口形状本身装不进这个依赖。
2. **namespace 语法未定**：空段？深度上限？合法字符？（合成 ID 与压平过滤键的方案都依赖它。）
3. **错误分类学未定**：`Get` 未命中返回什么？（仓库先例：`internal/session/store/store.go:9` `ErrNotFound`。）
4. **`List` 无分页/过滤**：`List(ctx, ns)` 返回全量，大 namespace 下直接不可用。
5. **`Value []byte` 与「向量检索」自相紧张**：Search 要对 Value 做嵌入 → 隐含 Value 是文本；`[]byte` 又暗示二进制。二选一未定。
6. **后端复用的口径**：`session/db` 五后端（postgres 482 / sqlite 487 / mongo 400 / surreal 569 / batch 372 非测试行）是
   session 行状 schema，「复用」只能是模式复用，不是代码复用。
7. **包名/落位**（见 §1.4 `session.State` 行的命名占位发现）。

## 2. 选项

### OPT-a1 进 v3.0（最小核：namespace+KV+向量检索，一个持久后端）

- **语义**：落地 §8 草图收窄版——`Item`/`Store` 接口 + namespace 语法 + `ErrNotFound` 分类 + **注入 `vectordb.EmbeddingFunction`**（不自建 embedder 抽象）+ 内存后端 + **一个**持久后端（建议按母约取 Postgres，模式对齐 `session/db/postgres`）。Search 在持久后端上 v1 可先以「取出+线性打分」实现，pgvector 留作 v3.1 扩展判据。
- **实现代价（实测校准）**：接口+类型+错误 ≈120-150 行；内存后端（含线性扫描 Search）≈150-200；Postgres 后端对齐 session 侧量级收窄到 ≈250-350；测试按仓库配比（memory 包 168/239）≈450-600。**合计 ≈1,000-1,300 行**，与母约 ~1,200 同量级。
- **开工前必须先定的边界**：§1.5 的 1/2/3/4/7 五项 + §3 列出的 explicitNonGoals。
- **破坏性**：**零**——Store 是全新包，今日全仓库零消费方（§1.3）；不接线 agent、不动四包任何一行。母约 :427「破坏性最强」按实测爆炸半径**不成立**，真实风险是概念性的（边界不清长出第二个 RAG 栈），不是集成性的。
- **对 P8 验收**：满足「长期记忆 + 与 knowledge/vectordb 分工清晰」（分工表以实测改写后写进契约）。

### OPT-a2 进 v3.0（母约全量：再加 vectordb 适配层）

- **语义**：在 a1 之上把探针的适配层产品化：Store 后端之一直接架在 `vectordb.VectorDB`（chromadb/redisdb）上，namespace 压平为 metadata 过滤。
- **实现代价**：a1 之上 +适配层与测试 ≈300-400 行，**总计 ≈1,300-1,700**，超出母约估计。
- **开工前必须先定的边界**：除 a1 全部前置外，还必须逐条裁决 §1.4 的四个损失点 + [A4] 合成 ID 语义 + 「一实例一 collection」下的租户隔离方案（过滤键无索引保证，Chroma 侧 metadata filter 性能不在本仓库掌控内）。
- **破坏性**：对存量代码仍为零；但把「Store 架在向量库上」这种**有损映射**固化为公共承诺，v3.1 想改回原生 schema 就是一次真破坏性迁移。
- **对 P8 验收**：满足，且「分工清晰」反而更难写——Store 与 vectordb 的边界从「不同底座」退化为「同一底座两种包装」。

### OPT-b 延到 v3.1

- **语义**：v3.0 不做 G8，P8 整行空转；§8 草图留档。
- **实现代价**：v3.0 为零；v3.1 支付 a1（或 a2）全额。**不存在「迁移成本」**——Store 今天零消费方，延后不产生任何要回改的存量代码。真正的代价是**空窗**：期间跨会话用户事实只有两个不合格的落点——`session.State`（会话作用域，`session.go:28`）或 `memory.InMemory`（进程内、重启即失，agent 默认注入 `config.go:71` `NewInMemory(100)`）。若 AgentOS/示例在空窗期把「用户画像」硬编码到这两处，v3.1 引入 Store 时才产生回改点（今天的硬编码面只有 agent 的 `Memory memory.Memory` 字段，`agent.go:37`/`config.go:23`——它已是接口注入，不构成阻塞）。
- **破坏性**：零（什么都不做）。
- **对 P8 验收**：v3.0 缺席；「能记住用户」的差异化推迟一个大版本。

## 3. 推荐与理由

**推荐 OPT-a1**（进 v3.0，收窄为最小核 + 单一持久后端），与母约 §12 的既有倾向（「(a) 进……需先定与 knowledge 的边界」）同向；
a2 的 vectordb 适配层明确**不进** v3.0。依据：

1. **实测重叠度把风险从「范围最大」降为「可圈定」**：与 knowledge 重叠为零（它没有存储）、与 memory 结构性不可互换
   （探针 [B]）、embeddings 是现成注入件——四包里真正相邻的只有 vectordb 一个，而组合面已被探针跑通且损失点可数（四个 + 一个身份冲突）。母约 :427 的「破坏性最强」按 §1.3 的零消费方实测不成立；**风险不在集成，在边界文案**——这正是可以用契约钉死的东西。
2. **a1 的量级就是母约自己报的量级**：≈1,000-1,300 行（§2 校准）≈ 母约 :86 的 ~1,200。母约报数时并未包含 a2 的
   适配层；选 a2 才是真正的超支（≈1,300-1,700）。
3. **(b) 省的不是钱，是时间**：延后无迁移成本可省（零消费方），换来的只是空窗 + 差异化推迟；而空窗期的替代落点
   （session.State / InMemory）恰恰会把「结构化用户事实」污染进两个不合格的容器，制造 a1 本可避免的回改面。
4. **有损映射不该抢先固化**：[A4] 的 ID 冲突、单 collection 绑定、`List` 无列出通道——在这些问题逐条裁决前把
   Store 架上 vectordb（a2），等于把探针里的权宜之计升格为公共语义。

**必须在 G8 契约 explicitNonGoals 里先写死的边界**（承接片起草契约的第一页）：

- Store **不自建** embedder 抽象，只注入 `vectordb.EmbeddingFunction`（`base.go:59-65`）；
- v3.0 **不做** vectordb/chromadb/redisdb 适配层（§1.4 四损失点 + [A4] 未裁决前不写）；
- **不动** `agent.Memory` 面（`agent.go:37`、`config.go:23/71` 原样，Store 不自动接线 agent）；
- **不动** `knowledge`/`vectordb`/`agentos.KnowledgeService` 的 RAG 职责（检索服务继续住在 agentos）；
- **不迁移** `session.State`/session 存储（会话级 KV 与跨会话 Store 分治）；
- 包名/落位先裁决（`internal/session/store` 已占用 `package store`）；
- namespace 语法、`ErrNotFound`、`(ns,key)` 合成身份、`List` 分页形状、`Value` 文本/二进制取舍——五项未指明处（§1.5）逐条入契约，不得默认。

**权属声明**：裁决权在负责人。本材料只登记决策点与实测，**不擅自改任何接口、后端或测试**（本文件的写入是本轮唯一的
仓库改动；探针在 `/tmp/d2probe`，不入库）。在负责人给出 D2 结论前，P8/G8 切片契约不得起草；本文亦不预写任何一方落地后的文案或测试期望。

## 4. 登记

- 本材料对应母约 §12 D2（`docs/design/v3-platform.md:566`）；§8（:425–459）的草图/分工表/后端行即本文 §1.1 的原文出处；
  §2.1 G8 行（:86）与 §10 P8 行（:541）随裁决结论一并由承接片写回，本文不代改。
- 状态文档的指针由主代理维护，本文不代管、不代改。
- 回执登记（均为 `/tmp/rex-receipt.mjs` 实际打印）：基线 `receipt:9f3a00ce-1c4f-40e8-98fa-64988123b380`（exit 0）、
  `-v` `receipt:94c70fdf-a43c-4291-9a6f-ed1b907099ce`（exit 0，30 测试通过）、`-cover` `receipt:ea4ca0db-6425-46ab-bd3e-62642091eca6`
  （exit 0，总 48.3%）、探针 `receipt:54cd3a04-2acc-4d24-8410-da5c81cb6963`（exit 0）；废弃
  `receipt:dc976245-8ddb-4be2-b4ca-c69b6b3d7c2f`（exit 1，探针第一版把 `SearchResult` 当 `Document` 用的编译错，
  已修）。过程披露：第一版适配器还曾因裸 Key 当 ID 拿到 nil panic（未走回执）——该事故未丢弃，转为 §1.4 的 [A4] 实测发现。
- 独立性披露：本文为单代理自查材料（取证、实验、成文同一代理），未派发独立审查；仓库内测试仅在
  `./pkg/hno/knowledge/... ./pkg/hno/vectordb/... ./pkg/hno/memory/... ./pkg/hno/embeddings/...` 上执行（本轮唯一测试命令面，
  未触碰并发编辑中的 `pkg/hno/graph`）。
