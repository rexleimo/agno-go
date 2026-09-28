# 切片 29 REFACTOR（P8 第 1 片：G8 Store 最小核）

工作项 `work-p8-store` / 契约 `docs/design/v3-test-scope-p8-g8-store.json`（D1–D13）。

## 1. REFACTOR 结论

两段落地后 REFACTOR 阶段做了两次收敛（均有门禁复测）：
① **LOC 收敛**：postgres 后端首版 390 行使非测试面达 764（超契约 626–726 窗口），压缩扫描变量块/辅助函数/
文档注释至 **724** 入窗（内存面 368 + postgres 356 口径，逐文件见 green §2 表）；
② **变异修牙**：首轮矩阵 4 条未达杀红（m3/m5/m11 编译失败代替行为红、m10 恰被夹具的键序巧合放行），
修法是把 D6 夹具的键序与语义序刻意错开（`a_runs`/`m_jazz`/`z_coffee`——语义胜者不得同时是键序胜者）
并修正 m3（补 `"errors"` import 摘除）/m5（`prev` 改空白标识符）/m11（不再摘 `"sort"` import，
SearchScored 仍合法使用）/grep 判据的退出码语义（grep 0=命中=杀）。修牙后绑定命令与防碰巧绿在最终字节重取新回执。

最终字节：`item.go 825603fb…` / `namespace.go ffa26fe9…` / `memory.go e8af66e3…` / `postgres.go b5db81f5…`
（变异矩阵基线口径 git-blob-hash；两轮矩阵 finalCheck identical=true）。

## 2. 完整变异矩阵（`scripts/mutation/p8g8-store.mjs`，仓库内可复现）

预检 16 条（m1–m14 + e1/e2）锚唯一性全过；命令 A = 绑定 TestP8G8_（-race -v）、命令 B = 邻近包
（knowledge/vectordb/memory/embeddings/session-db/agentos，全程零误伤）；结构判据（12 锚 hash + 零接线 grep）
每轮 before/after 各跑一次。每条 `restored=true`、`finalCheck identical=true`。

| 变异 | 形态（契约牙齿） | 判红面 | 判定 |
|---|---|---|---|
| m1 | Key 裸用作身份（D4 破，[A4] 事故形态） | 杀红 CompositeIdentity 等 6 条 | 杀红 |
| m2 | namespace 进 '.' 压平（D4 编码碰撞） | 杀红 CompositeIdentity | 杀红 |
| m3 | Get 未命中裸传 sql.ErrNoRows（D3/D8 翻译破） | 杀红 PG_RoundTripAndNotFound | 杀红 |
| m4 | Delete 未命中静默 no-op（D3 破） | 杀红 ErrorTaxonomy | 杀红 |
| m5 | 每次 Put 重打 CreatedAt（D5 破） | 杀红 PutUpsertTimestamps | 杀红 |
| m6 | 读取期逐条重嵌 Value（D7 调用数破） | 杀红 EmbedCallBudgetOnWrite / SearchRankingAndK | 杀红 |
| m7 | 向量塞进 Item 字段（D1/D7 破） | 杀红 ShapeAnchor | 杀红 |
| m8 | nil 注入 Search 静默空集（D2 fail-closed 破） | 杀红 NilEmbedderFailClosed | 杀红 |
| m9 | 表名不走标识符校验裸拼（D8 注入防线破） | 杀红 PG_IdentifierValidation | 杀红 |
| m10 | 相似度恒 0 的非语义打分（D6/D7 破） | 杀红 SearchRankingAndK | 杀红 |
| m11 | List 用 map 迭代序（D6/ADJ-3 确定性破） | 杀红 ListFullAndDeterministic | 杀红 |
| m12 | Delete 零行受影响不翻译（D8 持久侧破） | 杀红 PG_RoundTripAndNotFound | 杀红 |
| m13 | 代码里进 DDL / 预置原生向量列（D8/D9 破） | grep 判据计数 >0（结构牙） | 杀红 |
| m14 | import 存量会话类型（D8/D11 依赖图破） | grep 判据计数 >0（结构牙；go 侧 unused import 编译失败不计杀红） | 杀红 |
| e1 | （预期全绿）包内声明 unexported 死 embedder 接口——D11 白名单只看导出面 | 全绿 | 等价登记 |
| e2 | （预期全绿）List 排序键 Key 升序 → UpdatedAt 降序——契约只断「全量+同输入同序」，排序算法属观察上限 | 全绿 | 等价登记 |

契约 completionCriteria 第 2 条点名的 12 颗牙对账：①=m1、②=m2、③=m3/m4、④=m5、⑤=m6、⑥=m7、
⑦=m8（自建 Embedder 接口的半边=e1 等价登记 + D11 AST 白名单测试常驻）、⑧=m9、⑨=m13/m14、
⑩=m10（零分形态）+ m13（pgvector 预置形态）、⑫=m11；⑪（给 Agent 加 Store 字段接线）**不做文件变异**——
pkg/hno/agent 是兄弟片在途领地，写-恢复循环可能踩掉并发编辑；其牙齿由常驻结构检查承担
（12 锚 `git hash-object` before/after 全同 + `grep "hno/store"` 于 agent/session/internal/agentos 全程 0，
矩阵输出的 structural(before/after) 两行即该检查的运行记录）。

## 3. 契约 completionCriteria 逐条对账

1. redProtocol 两段各留 receipt（`b9d5fc23…` 编译红 / `08cefc16…` 行为红），两段之间测试文件零改动。✅
2. 牙齿由变异证明：14 杀红形态各不相同 + e1/e2 等价如实登记；⑪ 以常驻结构检查替代文件变异（理由如上）。✅（带披露）
3. 既有全部测试文件零改动；12 锚开工/收尾两次快照逐字节全同（agent.go 再基线归因编排方提交 c8142b2，
   详 red 文档 §4，非本片改动、不自批）。✅（带披露）
4. 门禁全 exit 0：build/vet/gofmt + 绑定命令 + 邻近包 + agentos。**`go test ./pkg/hno/agent/...` 按派发指令
   不跑**（兄弟片 G9 在途领地）；以 agentos 透明构建 agent 包 + 12 锚 + 零接线 grep 替代。✅（带披露）
5. 覆盖率 91.1%/85.0%（≥75%），绑定命令 skip 数 **0**。✅
6. 结构判据：非测试 LOC 724（626–726）；单文件 ≤500；store 导出 17（≤20）/postgres 唯一导出标识符 6（≤8）；
   boundaryModifiedFiles == 0；D11 grep 判据全命中。**含测试总数 1,715，超出契约估计 1,080–1,330**——
   诚实偏差：该估计由探针 233 测试行（仅 D1–D7 预演）外推；ADJ-5/6 回填新增 D12/D13 两行 + postgres sqlmock
   缝（308 行）+ 14 颗变异牙的可观察面使测试面达 991 行。命令化判据（非测试 LOC）达标；
   测试面不为凑数删行（删任何一行都是拆一颗契约牙）。⚠️ 偏差登记
7. 绑定命令最终字节重取新回执 `42224917…`；-race 无 DATA RACE；`-count=30 -race` 一次 exit 0（`40613f40…`）。✅
8. 写回：母约 §8 增交付块（分工表按 M1 实测改写、Item「带元数据」文案修正、§1.5 定形回填）+ §10 P8 行标交付。
   §12 D2 行补「落地见切片 29」指针与 §2.1 G8 行 LOC 实测回填由主代理独占（本片按派发权限只写 §8/§10）；
   裁决材料 v3-adjudication-d2-store.md 未代改；状态文档由编排方收口。✅（范围按派发指令收窄）

## 4. 过程如实登记

- D5 时间戳测试落同包夹具文件（`internal_shapes_test.go`）：契约 D5 断言要求 `UpdatedAt` 严格递增，
  外部包测试无法注入未导出时钟、用墙钟在 -count=30 下有同刻度风险；stepClock 属契约 allowedTestSeam
  的「同包夹具」项，测试本身只用导出 API + 步进时钟。
- 首轮矩阵 4 条未杀红（m3/m5/m11 编译失败、m10 巧合放行）——全部修在**矩阵与夹具**上而非产品代码，
  产品字节在两轮矩阵间逐位未动（baseline git-blob-hash 两轮一致可证）。
- m14 的 go 侧 BUILD_FAILED（unused import）如实保留展示，判定走 grep 牙；未把它伪装成行为红。
- postgres 与 memory 各持一份 ~12 行的 cosine/校验小助手（未导出重复）：D11 导出白名单已钉死、
  抽公共导出助手会扩面越权，登记为已知最小重复。
- Postgres 后端**已落地**（非内存-only 延期）：database/sql + go-sqlmock 路线，0 skip、-race 可链接，
  与契约 M3 的选型实测一致。

## 5. 审查

review 结论见 `docs/design/v3-p8-g8-review-verdict.json`（作者自查 + 编排方复核；
独立性披露：实现由派发子代理执行，两轴审查同会话承担，缺口如实登记）。
