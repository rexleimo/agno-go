# 切片 29 RED 观察（P8 第 1 片：G8 Store 最小核）

工作项 `work-p8-store` / 契约 `docs/design/v3-test-scope-p8-g8-store.json`（D1–D13，ADJ-1…ADJ-6 已回填 A,A,A,A,B,B）。
实现由派发子代理执行（用户 2026-09-28 明确允许并行子代理，独立性披露见 review-verdict）。
绑定场景命令逐字：

```bash
go test ./pkg/hno/store/... -run TestP8G8_ -count=1 -race -v
```

## 1. 开工基线（仓库只读复核，未沿用契约设计期回执）

| 项 | 结果 | receipt |
|---|---|---|
| 邻近包基线（knowledge/vectordb/memory/embeddings/session-db）`-count=1` | 全 ok，exit 0 | `receipt:f6fd0c96-e93c-4dc2-b20d-8fd4fa56f279` |
| `go test ./pkg/agentos/... -count=1` | ok，exit 0 | `receipt:f3c96444-15d4-4dbb-84c1-f2e81b819529` |
| `grep -rn "hno/store" pkg/hno/agent pkg/hno/session internal/session pkg/agentos` | 0（零接线今日复核成立） | — |
| 12 个 baselineBytes 锚 `git hash-object --no-filters` | **11/12 与契约逐字同**；`pkg/hno/agent/agent.go` 漂移（见 §4） | 快照存 `/tmp/p8g8-baseline/anchors-start.txt` |

## 2. 两段制 RED（契约 redProtocol）

| 段 | 状态 | receipt |
|---|---|---|
| 第一段 RED（测试文件三件全落、产品零字节） | exit 1——`pkg/hno/store` 不存在，全仓 0 处引用下的**允许类编译红**（新包落地前的引用错） | `receipt:b9d5fc23-db0b-4f9b-ba7b-7306dc1b7826` |
| 第一段完成 / 第二段 RED（公共面 + 内存后端落地；postgres 为构造面骨架，七方法 `not implemented`） | exit 1——内存侧 D1–D7/D11/D12/D13 全绿，postgres 侧 7 个 `TestP8G8_PG_*` **全部行为红**（方法未实现，非编译红；D8/D9 的翻译/单查询/标识符校验牙齿在此形态下可判红） | `receipt:08cefc16-a91b-4339-a1ee-39f651891a81` |
| 第二段 GREEN（postgres 实装 + DDL 双份落） | exit 0 | `receipt:f0ba21c1-c720-4654-83ac-38812794b208`（落地字节）；最终字节复测 `receipt:42224917-0ec1-49ea-a827-745dbfd2cd4b` |

两段之间测试文件零改动（`store_test.go` / `internal_shapes_test.go` / `postgres/postgres_test.go` 在第一段 RED 前一次写就，其后未动）。

## 3. RED 真实性说明

- 编译红仅一段且属契约明示的允许类（新包落地前）；postgres 子包的行为红以「骨架 + not implemented」形态取得，
  避免「postgres 测试包无法编译」淹没行为红——这是 redProtocol「落地后立即转行为红」在第二段的重演。
- 变异矩阵在 GREEN 字节上对每颗牙做了二次判红（14 杀，红形态各不相同，见 refactor 文档 §2），
  与本文件的段红互为印证。

## 4. 开工即登记的基线偏移（不属本片）

契约 `baselineBytes.anchors` 中 `pkg/hno/agent/agent.go` = `9813b86a…`，开工实测 = `54b5dad5…`。
溯源：`9813b86a…` 恰是 `c8142b2^`（契约起草时点）的值；编排方随后的提交 `c8142b2`（StreamTasks producer 接线）
改写了 agent.go——而 `c8142b2` 正是本任务书面声明的工作基线 HEAD。故按契约 completionCriteria 第 3 条登记为
**开工前已发生的再基线**（归因：编排方自己的交付提交，非本片改动；本片对 12 锚零写入，开工/收尾两次快照逐字节全同）。
其余 11 锚（含 `agent/config.go` 的 `memory.NewInMemory(100)` 默认注入面）与契约逐字同；agent.go 的
`Memory memory.Memory` 字段实测仍在（agent.go:36）。§12 D2 行与 §2.1 G8 行的写回由主代理独占，本片未代改。
