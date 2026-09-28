# 切片 29 GREEN（P8 第 1 片：G8 Store 最小核，D2=OPT-a1 落地）

工作项 `work-p8-store` / 契约 `docs/design/v3-test-scope-p8-g8-store.json`（D1–D13）。
RED 观察与锚点再基线披露见 `docs/design/v3-red-observation-p8g8.md`。本片绑定场景命令逐字：

```bash
go test ./pkg/hno/store/... -run TestP8G8_ -count=1 -race -v
```

## 1. GREEN 判据与实测（最终字节）

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| D1–D13 全绿（契约绑定命令，逐字；22 个 `TestP8G8_*`，**0 skip**） | 同上 | 0 | `receipt:42224917-0ec1-49ea-a827-745dbfd2cd4b` |
| 防碰巧绿：绑定 `-count=30 -race` | ok | 0 | `receipt:40613f40-307c-4e71-96a0-060d14ff78d5` |
| 邻近包基线零改动（knowledge/vectordb/memory/embeddings/session-db） | `go test … -count=1` | 0 | `receipt:1c7b2fb5-f13c-4726-8cba-eb18b5547a4a` |
| `go test ./pkg/agentos/... -count=1`（透明构建 agent 包，不跑其测试） | ok | 0 | `receipt:a94c3726-fd25-4e82-9ccc-6660111bd913` |
| 覆盖率（无 skip 撑数）：store **91.1%** / postgres **85.0%**（判据 ≥75%） | `go test ./pkg/hno/store/... -count=1 -cover` | 0 | 非回执记录 |

## 2. 实现清单

| 文件 | 状态 | 内容 |
|---|---|---|
| `pkg/hno/store/item.go` | 新增（37 行） | `Item` 恰五字段 + `SearchHit`（ADJ-5=B）+ `Store` 七方法（草图五方法签名逐字 + `SearchScored`/`PutMany`）+ `var _ Store` 编译期断言 |
| `pkg/hno/store/errors.go` | 新增（14 行） | 7 哨兵：`ErrNotFound`/`ErrInvalidNamespace`/`ErrInvalidKey`/`ErrInvalidItem`/`ErrInvalidSearch`/`ErrNoEmbedder`/`ErrDimensionMismatch` |
| `pkg/hno/store/namespace.go` | 新增（82 行） | ADJ-2=A 段规则（非空/UTF-8/禁 U+001F/≤128/深度≤16，根层合法）+ `Composite` 长度前缀无碰撞规范编码 + `ValidateNamespace`/`ValidateKey` |
| `pkg/hno/store/memory.go` | 新增（235 行） | `MemoryStore`：写入期嵌入、upsert（CreatedAt 写一次/UpdatedAt 刷新）、Delete 未命中上报、List 全量按 Key 升序、`SearchScored`（降序/top-k/min(k,命中数)/k≤0 报错/确定性序/非 UTF-8 剔除/维度不一致上交/只读）、`PutMany` 整批一次 Embed + 整批原子 + 错误点名条目 |
| `pkg/hno/store/postgres/postgres.go` | 新增（358 行） | database/sql 注入式持久后端：标识符白名单 + 带引号限定名、显式列 + `$n` 占位、ON CONFLICT upsert（created_at COALESCE/updated_at EXCLUDED）、`sql.ErrNoRows`→`ErrNotFound`、RowsAffected==0→`ErrNotFound`、Search 恰一次 SELECT + 应用层线性打分、`PutMany` 单事务回滚、`Close` |
| `pkg/hno/store/store_test.go` | 新增（619 行，外部包） | D1–D13 测试：形状反射锚（七方法签名串逐字）、注入与 nil fail-closed、错误分类学互斥、(ns,key) 身份与编码碰撞、字节恒等往返、打分与 k 语义、调用数判据（200+10=210，行数二倍增量 0）、SearchScored 等价对、PutMany 批量语义、List 全量确定序、D11 导出符号 AST 白名单 |
| `pkg/hno/store/internal_shapes_test.go` | 新增（64 行，同包夹具） | stepClock 步进时钟夹具 + D5 时间戳确定性测试（禁 sleep、禁墙钟断言；外部包无法注入未导出时钟，落位理由登记于 refactor §4） |
| `pkg/hno/store/postgres/postgres_test.go` | 新增（308 行，go-sqlmock） | 五方法往返、未命中翻译双路（ErrNoRows / RowsAffected==0）、后端故障不伪装、标识符注入拒绝、单查询线性打分 + 维度不一致、nil 嵌入件 fail-closed、PutMany 整批原子（失败不触库）、List 全量 |
| `scripts/migrations/003_agentos_store.sql` + `deploy/helm/agno-agentos/files/003_agentos_store.sql` | 新增（DDL 双份） | `(namespace,key)` 复合主键 + `embedding BYTEA` 物理列 + namespace 索引；不在代码里跑迁移（既有约定） |

## 3. 门禁（契约判据 3–7，全部在最终字节上）

| 门禁 | 结果 |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./pkg/hno/store/...` / `gofmt -l pkg/hno/store` | 干净 |
| 结构判据：新包非测试 LOC **724**（契约窗口 626–726）；单文件 ≤500（最大 memory.go 235 / postgres.go 358） | 达标 |
| 导出面：store `go doc -all` 计数 **17**（≤20，含 7 个接口方法行）；唯一导出标识符 **18**（AST 白名单测试逐字断言）；postgres 唯一导出标识符 **6**（Storage/Config/Option/NewStorage/WithSchema/WithTable，≤8） | 达标 |
| grep 判据：`hno/store` 在 agent/session/internal/agentos 命中 **0**；D8/D11 禁运（vectordb.VectorDB|Document|SearchResult、knowledge、types.Message、session、pgx、BatchWriter）命中 **0**；`vectordb.EmbeddingFunction` 命中 **7**（≥2）；D9 半边（pgvector/CREATE EXTENSION/vector(/<=>/ivfflat/hnsw）在 pkg/hno/store + scripts/migrations + deploy 命中 **0** | 达标 |
| 12 个 baselineBytes 锚开工/收尾两次快照 | **逐字节全同**（agent.go 再基线披露见 red 文档 §4） |
| boundaryModifiedFiles | **0**（存量包零改动） |
| 变异矩阵 | 14 杀 + 2 等价，逐字节恢复（见 refactor 文档 §2） |

## 4. 每行由哪段实现担保

| 行 | 断言的公共事实 | 实现落点 | 测试 |
|---|---|---|---|
| D1 | Item 恰五字段、Store 恰七方法、草图五签名逐字 | item.go 形状 + 反射锚 | ShapeAnchor |
| D2 | 注入仓库现成 `vectordb.EmbeddingFunction`；nil 时四方法照常、Search/SearchScored fail-closed | 构造器注入 + `ErrNoEmbedder` | EmbedderInjectionRoundTrip / NilEmbedderFailClosed |
| D3 | 未命中 `ErrNotFound`、参数三类互斥、后端翻译且故障不伪装 | errors.go + 两后端翻译 | ErrorTaxonomy / PG_BackendErrorNotMasked |
| D4 | (ns,key) 复合身份跨 ns 共存；规范编码无碰撞；空段拒绝；根层合法 | Composite + 校验面 | CompositeIdentity |
| D5 | upsert：行数 1、CreatedAt 写一次、UpdatedAt 刷新、Value 覆盖、失败不留半截行 | upsertLocked / stamp | PutUpsertTimestamps（步进时钟）/ FailedPutLeavesNoPartialRow |
| D6 | 降序、top-k=min(k,命中数)、k≤0 报错、确定性序、Search 只读 | SearchScored | SearchRankingAndK（键序与语义序刻意错开，零分/关键词冒充打分活不过本行） |
| D7 | 查询期恰一次嵌入、调用数与行数无关（200+10=210，二倍行数增量 0）、维度不一致上交 | 写入期物理列 | EmbedCallBudgetOnWrite / DimensionMismatchSurfaces |
| D8 | 模式复用非代码复用：注入 db、标识符校验、upsert 翻译、Close；0 pgx/0 session import | postgres.go 八条清单 | PG_* 七件 |
| D9 | Search 恰一次 SELECT + 应用层打分；无 pgvector/扩展/索引；形状不变量由 D1 反射锚担保 | SearchScored 单查询 | PG_SearchSingleQueryLinearScoring（ExpectQuery 次数硬断言）+ m13 grep 牙 |
| D10 | 12 锚逐字节 + 零接线 + 存量零改动 | 旁路设计 | structural 检查（变异脚本 before/after 两行）+ grep 0 |
| D11 | 导出符号 = 18 项白名单、无 Loader/Chunk/Collection/VectorDB/Message/Session 命名 | item/errors/namespace | ExportedSurface（AST 解析本包） |
| D12 | Search 与 SearchScored 同集合同序只差分数、分数单调不增、非文本剔除可见、错误分类一致 | Search 委托 SearchScored | SearchScoredEquivalence |
| D13 | PutMany 整批一次 Embed（调用数与条数解耦）、整批原子、错误点名 Key | 两后端 PutMany | PutManyBatchSemantics / PG_PutManyAtomic |

## 5. 本阶段未闭环

- 变异矩阵与契约逐条对账见 `docs/design/v3-p8-g8-refactor.md`；偏差登记（agent.go 再基线、测试 LOC 超估计、
  agent 包测试按派发指令不跑）见同文件 §3/§4。
- `golangci-lint` 欠项沿用前序切片；切片 2–29 未提交（S18-SPEC-2 沿用，由编排方收口提交）。
