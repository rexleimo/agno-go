# 切片 33 GREEN（P6/G10 OPT-β：workflow 控制流全换到图内核）

工作项 `work-p6-g10-opt-beta-migration` / 契约 `docs/design/v3-test-scope-p6-g10-migration.json`（D1–D14）。

## 1. GREEN 判据与实测

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| D1–D14 全绿（契约绑定命令，逐字；14/14 PASS） | `go test ./pkg/hno/workflow/... -run TestP6G10_ -count=1 -race -v` | 0 | `receipt:79b20f4e-178a-477e-810d-bb809d2cc35d` |
| 整包 plain+race（既有测试零改动全绿 = 零改动担保） | `go test ./pkg/hno/workflow/... -count=1 -race` | 0 | `receipt:2b0c0e60-1a03-4ccb-a94f-b933cdd8a692` |
| 邻近面（agent/session-contract） | `go test ./pkg/hno/agent/... ./internal/session/contract/... -count=1` | 0 | `receipt:afce9ba5-da62-4fb5-b55c-49dd21d1f66a` |
| 子代理 RED（编译路径未接管时） | 同绑定命令 | 1 | `receipt:b5f7aad2-…` |

## 2. 实现差异

| 文件 | 变更 |
|---|---|
| `pkg/hno/workflow/graph_compiler.go`（新增，694 行） | []Step/Condition/Router/Loop/Parallel → 图声明的编译器：线性主干无条件边、Condition/Router 条件边（互补谓词）、Parallel 分支头克隆 + AddJoin 全前驱汇聚 + mergeParallelResults 逐字复刻、Loop 条件环 + MaxIteration 守卫、execLedger 执行序台账（事件/键集从最终 EC 反查）、Router 未命中显式复刻 `router %s: route '%s' not found`、编译期守卫（nil 主干/自引用环 → 清晰错误，替代旧 panic/栈溢出） |
| `pkg/hno/workflow/executor.go` | 执行路径切到编译图（公共 API 零改动；会话/历史/持久化/取消/metrics 留 workflow 侧，按 OPT-β 裁决） |
| `pkg/hno/workflow/p6g10_migration_test.go`（新增，894 行） | D1–D14 等价测试 |
| `pkg/hno/graph` | **零改动**（契约 explicitNonGoals 第 1 条兑现：别名隔离的解不需要动引擎——克隆发生在 workflow 侧分支头） |

别名硬阻塞的解（D8）：编译器扇出只交不可变信封；Parallel 是唯一并发扇出，分支头
`cloneBranchEC` 后才 Execute（WorkflowHistory/HistoryContext 不拷贝=旧代码被钉住的保真）；
两分支并发写共享上下文的形状经编译器不可表达。

## 3. 门禁

- 结构判据：导出面 146 不变（零新增公共符号）；**13 个既有测试文件 git-blob 逐字节不变且全绿**（D12）；
  `pkg/hno/graph` 字节零改动；`make test` 口径的全包 -race 绿（`receipt:2b0c0e60…`）。
- 防碰巧绿：TestP6G10_ `-count=10 -race` exit 0（`receipt:49b8185e-…`，子代理轮）+ 最终字节
  `-race` 重跑（`receipt:79b20f4e…`）。
- 两个示例（cmd/examples）未改编译过程（D12）。

## 4. 每行由哪段实现担保

D1 线性穿线（无条件边链）｜D2 Resume=startIdx 编译起点｜D3 错误聚合逐字节复刻｜D4 Condition 条件边+互补谓词｜D5 Router 映射+未命中显式报错｜D6 Loop 条件环+MaxIteration｜D7 Parallel 克隆+汇聚逐字复刻｜D8 别名隔离（分支头克隆）｜D9 事件/账本台账反查｜D10 取消原路径｜D11 结构上界预算+合成名消重｜D12 零改动担保｜D13 嵌套组合等价｜D14 编译期守卫。
