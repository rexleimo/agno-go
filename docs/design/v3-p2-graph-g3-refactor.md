# 切片 22 REFACTOR（母约 §4 G3：节点策略四合一）

工作项 `work-p2-graph-g3-node-policies` / 契约 `docs/design/v3-test-scope-p1-graph-slice22.json`（D1–D14）。

## 1. REFACTOR 结论：无独立重构动作，最终字节 = GREEN 字节

GREEN 交付的实现直接采用了 M2 设计期副本预演过的形状（策略声明集中在 policy.go、执行缝集中在
runNode 包络、发射集中在消费者两处），没有可合并的重复；REFACTOR 阶段零代码改动。
最终字节：`graph.go 72974bf6…`、`scheduler.go 873f4a82…`、`policy.go 6d71fa47…`、
`p2g3_policies_test.go ec390cc7…`。最终门禁复测：build/vet/gofmt 干净，全包 `-count=1` 与
`-race` `ok`（`receipt:b45262e0…` / `receipt:5e54d8c6…`）。

## 2. 完整变异矩阵（`scripts/mutation/p2g3-policies.mjs`，仓库内可复现）

命令 A = 绑定场景命令（`TestP2G3_`，`-race -v`），命令 B = 全部既有判据（`-run TestP1`）。
基线：`scheduler.go 873f4a82…`、`policy.go 6d71fa47…`；每条变异后 `restored=true`、
`finalCheck identical=true`。

| 变异 | 形态 | 命令 A | 命令 B | 判定 |
|---|---|---|---|---|
| m1 | 重试循环整段失效（恒 break） | exit 1，杀红 D1/D3/D5/D8 | exit 0 | **杀红** |
| m2 | ShouldRetry 判据取反 | exit 1，杀红 D1/D3/D4/D5/D8 | exit 0 | **杀红** |
| m3 | 期限钳制整段失效 | exit 1，杀红 D6/D8 | exit 0 | **杀红** |
| m4 | PerAttempt 退化为单次期限 | exit 1，杀红 D8 | exit 0 | **杀红** |
| m5 | 缓存查询整段失效 | exit 1，杀红 D9/D10/D11 | exit 0 | **杀红** |
| m6 | 失败也写缓存 | exit 1，杀红 D11 | exit 0 | **杀红** |
| m7 | 完成事件发射摘掉 | exit 1，杀红 D12/D13 | exit 0 | **杀红** |
| m8 | 脱敏分支摘掉 | exit 1，杀红 D13 | exit 0 | **杀红** |
| e1 | 失败路径的 trace 发射摘掉 | exit 0 全绿 | exit 0 | **等价变异 —— 登记为观察上限** |

脚本自纠一处：m1 第一版锚替换成 `if true {` 会让 `max` 变成未使用变量（编译失败，不是测试杀红），
已改为 `if true || attempt >= max || …` 保持其余判据被引用。

### 2.1 e1 等价性的读法（观察上限）

D1–D14 没有一行观察「失败节点的事件」：D12 只钉完成事件（Node/Out/In），错误分支的发射
（带 Err 字段）是 fail-closed 方向的未钉语义——发射它不判红任何既有行，摘掉它也不判红。
要钉它需要一行新契约（失败事件含 Err 且 Attempt 正确），留给需要它的切片，本片不追加。

## 2.1 一次未复现的裸 FAIL（如实登记，附排查记录）

收口核查时全包 `-count=1`（非 race）出现过一次**裸 `FAIL`**（末行无包名、无耗时，与任何测试失败文案
都不同形）。处置：随后 plain 全包 7 次复跑（含 -v、14×3 轮 P2G3 全 PASS）、`-race` 多轮、
count=30/ 整包 count=5 全部 `ok`，无法复现；D6/D8 两个时序敏感 Test 的判据复核为确定性
（错误类型与 deadline 值递增，不含真实耗时断言）。结论登记为**疑似工具链/环境一次性瞬态，
成因未定位**：不下「必然是环境」的结论，也不为不可复现的形状追加断言；若再现，第一动作是
`-v` 全量捕获 + goroutine 泄漏探针，届时按 rex-test-design 另起一片。

## 3. 契约 completionCriteria 逐条对账

1. redProtocol 两段各自留 receipt：第一段 `receipt:d8269809…`（exit 1，10 红/4 反向绿）→
   第二段 `receipt:b5afcc0f…`（exit 0，14 绿）；两段之间测试文件零改动。✅
2. 牙齿由变异证明：m1–m8 杀红（判据 2 点名的八种形态逐条覆盖）、e1 等价登记。✅
3. 既有 10 份测试文件零改动，逐文件 hash-object 复核一致（green 文档 §3）。✅
4. 门禁全 exit 0：build / test / -race / vet / gofmt + 邻近回归（green 文档 §3）。✅
5. 结构判据：`sync.`=0、LOC 1085≤1500、导出符号恰 35。✅
6. 绑定命令在最终字节上重跑并取新回执；`-race` 无 DATA RACE。✅
7. 写回完成：母约 §4（TraceConfig 补 Hook + 四策略标注已交付 + 交付表 P2 行）、
   状态文档 §4（P2 第 1 片交付记录与携带项去向）。✅
8. 防碰巧绿：`-count=30 -race`（`receipt:4d26a57d…`）与整包 `-count=5 -race`
   （`receipt:5bcf938b…`）均 exit 0。✅

## 4. 审查

review 结论见 `docs/design/v3-p2-graph-g3-review-verdict.json`（作者自查，独立性披露同前序切片）。
