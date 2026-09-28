# 切片 33 REFACTOR（P6/G10 OPT-β：workflow 控制流全换到图内核）

工作项 `work-p6-g10-opt-beta-migration` / 契约 `docs/design/v3-test-scope-p6-g10-migration.json`（D1–D14）。

## 1. REFACTOR 结论

主代理接手后仅补分支头克隆 1 行（D8 硬阻塞的解），实现已符合契约形状；零进一步改动。
最终字节：`graph_compiler.go ff847d51…`（基线口径）、`executor.go ad7b5a96…`（变异矩阵基线口径）。

## 2. 完整变异矩阵（`scripts/mutation/p6g10-migration.mjs`，仓库内可复现，13 条预检全过）

| 变异 | 形态 | 结果 |
|---|---|---|
| m1 | 编译器开关失效形态一 | BUILD_FAILED（不计杀红，登记） |
| m2 | Parallel 汇聚语义破坏 | 杀红 ParallelParity |
| m3 | Loop 映射破坏 | 杀红 BudgetAndDuplicateIDs/LoopParity/NestedComposites |
| m4 | 事件台账破坏 | 杀红 EventsParity（+既有 TestWorkflow_RunStoresEvents） |
| m5 | 恒等节点摘除（主干全线红） | 杀红 7 个 Parity 行 |
| m6 | Router 未命中复刻摘掉 | 杀红 RouterParity |
| m7 | 分支头克隆摘掉（D8 硬阻塞复现） | 杀红 ParallelConcurrentWriteNoRace/ParallelParity |
| m8 | 编译器开关失效形态二 | BUILD_FAILED（不计杀红，登记） |
| m9 | Resume 起点破坏 | 杀红 ResumeParity（+既有 TestWorkflow_RunResumeFromStep） |
| m10 | 预算/合成名破坏 | 杀红 BudgetAndDuplicateIDs |
| m11 | 取消 parity 破坏 | 杀红 CancellationParity（+TestWorkflow_CancellationPersistence） |
| e1/e2 | 等价变异 | 全绿，登记为观察上限 |

全部 `restored=true`、`finalCheck identical=true`。既有测试在 B 侧成对判红 —— 零改动担保的反向验证。
m1/m8 的 BUILD_FAILED 与 p3g5 的 m1 同因（锚替换致未使用变量/缺失符号），登记脚本自纠口径。

## 3. 契约 completionCriteria 逐条对账（6 条）

1. redProtocol 两段：子代理 RED `receipt:b5f7aad2…`（exit 1）+ 接手中间态（D7 两向红，文案引于
   red 文档 §1）→ 全绿 `receipt:79b20f4e…`；两段之间既有测试文件零改动。✅（带取消事件披露）
2. 牙齿由变异证明：m2–m7、m9–m11 覆盖 D2–D6/D8–D11 主干；m1/m8 BUILD_FAILED 登记不计。✅
3. 既有 13 个测试文件逐字节不变且全绿；导出面 146 不变；两示例不改编译。✅（D12）
4. 门禁全 exit 0：build/vet/gofmt + 绑定命令 + 整包 -race + 邻近面。✅
5. 结构判据：pkg/hno/graph 字节零改动（契约 explicitNonGoals 第 1 条）。✅
6. 防碰巧绿：`-count=10 -race`（`receipt:49b8185e…`）+ 最终字节 `-race` 重跑（`receipt:79b20f4e…`）。✅

## 4. 三项开工前置的落地形态（母约 P6 行登记的 (i)/(ii)/(iii)）

(i) S19-STD-1 语义：并行扇出=值语义（分支头克隆私有 EC），线性主干=引用语义穿线（D7/D8 钉）；
(ii) 互斥分支：workflow 侧互补谓词（D4 恰一支执行），未给 graph 加 Branch/Router 公共面；
(iii) Router 未命中：适配器显式复刻 `router %s: route '%s' not found`（D5 逐字节）。
三项前置以实现+测试的形态关闭，P6 行的「不得建切片契约」守卫条款至此解除。

## 5. 审查

review 结论见 `docs/design/v3-p6-g10-review-verdict.json`（作者自查 + 编排方复核，独立性披露同前）。
