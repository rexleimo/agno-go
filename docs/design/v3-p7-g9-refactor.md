# 切片 25 REFACTOR（母约 G9：观测接线）

工作项 `work-p7-g9-observability` / 契约 `docs/design/v3-test-scope-p7-g9-observability.json`（D1–D13）。

## 1. REFACTOR 结论：无独立重构动作，最终字节 = GREEN 字节

GREEN 交付直接采用了 M2 设计期副本预演过的形状（缝集中在 resilience.go 一个文件、
调用点只换一行、钳制独立成私有 helper），没有可合并的重复；REFACTOR 阶段零代码改动。
最终字节：`runner.go 18fe1f80…`、`resilience.go 2d6c944c…`、`p7g9_resilience_test.go 1c25b730…`。
最终门禁复测：build/vet/gofmt 干净，两包 `-count=1` 与 `-race` `ok`
（`receipt:afbe63e4…` / `receipt:c17f4bd4…`）。

一处对 M2 预演副本的有意偏离（如实登记）：`/tmp/g9stage1` 里的缝曾把 breaker-open 预包成
`fmt.Errorf("model invoke: %w", observability.ErrOpen)`，与 Run 的唯一 wrap 点叠成
「runner: model invoke: model invoke: circuit breaker open」——M2 已把它定为缺陷并反哺进契约
designConsequence 第 5 条（缝内裸哨兵）。本片落地的 `resilience.go` 直接取修正后的最终形态，
第一段（RED）与第二段（GREEN）之间该文件零改动，两段差异恰好只有调用点一行。

## 2. 完整变异矩阵（`scripts/mutation/p7g9-observability.mjs`，仓库内可复现）

命令 A = 绑定场景命令（`TestP7G9_`，`-race -v`），命令 B = runner 包全部既有判据
（`-skip TestP7G9_`，16 个既有测试）。基线：`runner.go 18fe1f80…`、`resilience.go 2d6c944c…`；
每条变异后 `restored=true`、`finalCheck identical=true`。判红只认 `--- FAIL:` Test 名。

| 变异 | 形态 | 命令 A | 命令 B | 判定 |
|---|---|---|---|---|
| m1 | 钳制摘掉（只留 nil 守卫）—— MaxAttempts=0 透传，零迭代静默成功复现 | exit 1，杀红 D4 | exit 0 | **杀红** |
| m2 | 熔断记账挪进尝试闭包（每尝试记一次，一回合烧光阈值） | exit 1，杀红 D8 | exit 0 | **杀红** |
| m3 | 门禁摘掉 —— open 后照打模型 | exit 1，杀红 D5/D7 | exit 0 | **杀红** |
| m4 | 成功/失败记账取反 | exit 1，杀红 D5/D7 | exit 0 | **杀红** |
| m5 | chat span 挪出尝试闭包（每回合一条） | exit 1，杀红 D10 | exit 0 | **杀红** |
| m6 | 重试循环摘掉（attempts 恒 1，首败即停） | exit 1，杀红 D1/D3/D8/D10/D12/D13 | exit 0 | **杀红** |
| m7 | wrap 点重复（缝内预包 runner: model invoke: 前缀） | exit 1，杀红 D3（前缀计数恰 2） | exit 0 | **杀红** |
| m8 | 只给 New 默认 Invoker 回退接线（调用点直调）—— 自定义 Invoker/流式路径无韧性 | exit 1，杀红 D12 | exit 0 | **杀红** |
| m9 | 回合预算被缝内尝试消耗（尝试数被 MaxTurns 钳制） | exit 1，杀红 D13 | exit 0 | **杀红** |
| e1 | 单次快路径摘掉（恒走 observability.Retry） | exit 0 全绿 | exit 0 | **等价变异 —— 登记为保守写法，非语义承载** |
| e2 | 仅 breaker-open 返回处预包 model invoke: 前缀（M2 缺陷的历史形态） | exit 0 全绿 | exit 0 | **等价变异 —— 登记为观察上限（S25-SPEC-1）** |

契约判据 2 点名的九种形态逐条对上：钳制摘掉=m1（D4 静默成功）、记账挪进闭包=m2（D8 提前
开断）、门禁摘掉=m3（D5 红且 D7 烧尝试）、记账取反=m4（D5/D7 反向）、span 挪出闭包=m5
（D10 粒度变）、重试循环摘掉=m6（D1）、wrap 点重复=m7（D3 计数）、只给默认回退接线=m8
（D12，同时即「流式路径绕过缝」的形态——流式调用方供自定义 Invoker）、turn++ 挪进缝内=m9
（D13）。脚本骨架沿 p2g3（锚唯一性预检、逐字节恢复自检、判红只认 FAIL 名），唯一扩展是
允许单个变异跨 runner.go/resilience.go 两个实现文件（m7/m8 需要），测试文件永远不在变异面内。

### 2.1 两处等价性/牙齿的如实读法（观察上限）

1. **m4 与 D6**：记账取反在 D6（恢复行）上等价——失败被记成 Success 时熔断根本不开，
   「开断后的恢复」无从谈起；D6 的牙齿依赖 D5 先钉住「会开断」。D5+D6 成对观察
   （契约 D6 assertion 自述「本行与 D5 成对」），m4 由 D5 杀，D6 单独不构成杀红。登记为
   成对行的固有观察上限，不追加新行。
2. **e1/e2**：e1——`attempts == 1` 快路径换走 `observability.Retry`（MaxAttempts=1）在 D1–D13 上
   逐语义同形——Retry 对 1 次恰执行 fn 一次并原样交错、无退避等待；快路径只是防意外退避的
   保守写法。e2——仅在 breaker-open 返回处预包 `model invoke: %w`（M2 实测缺陷的历史形态）
   全绿：errors.Is 穿透 `%w` 链仍可达 ErrOpen，D5/D7 钉的是错误身份与调用冻结，不钉前缀
   形状。该双前缀缺陷靠契约 designConsequence 第 5 条的设计裁决修正（缝内裸哨兵），不是被
   某条 D 行钉住——登记为观察上限 S25-SPEC-1（见 review-verdict findings）。两者均全绿登记，
   不算杀红。

## 3. 契约 completionCriteria 逐条对账

1. redProtocol 两段各自留 receipt：第一段 `receipt:aa8c6103…`（exit 1，9 红 / 4 反向绿）→
   第二段 `receipt:669c0c6d…`（exit 0，13 绿）；两段之间测试文件零改动。✅
2. 牙齿由变异证明：m1–m9 杀红（判据 2 点名的九种形态全覆盖）、e1/e2 等价登记。✅
3. 既有测试文件与 observability 九文件零改动：逐文件 hash-object 复核与 baselineBytes 全同
   （green 文档 §3；runner.go 除外——它必须变；state.go/stop.go/tool_batch.go 仍全同）。✅
4. 门禁全 exit 0：build / test / -race / vet / gofmt + 邻近回归 agent、graph、
   internal/session/contract（agent 经 newKernel 构造 runner.Config，加法字段未破坏其编译与
   行为）。✅
5. 结构判据：runner 导出符号恰 23（零新增导出：2 个 Config 字段 + 4 个私有字段 +
   2 个私有函数）；非测试 LOC 582 ≤ 700（M2 副本实测 582，实落地同数）；`resilience.go`
   `sync.` 计数 0。✅
6. 绑定命令在最终字节上重跑并取新回执；`-race` 无 DATA RACE。✅
7. 防碰巧绿：绑定命令 `-count=30 -race`（`receipt:908c02ff…`）与两包整包 `-count=5 -race`
   （`receipt:26bc6f79…`）均 exit 0。✅
8. 完成后回写：母约 §2.1 G9 概念行与 §10 交付表 P7 行标注第 1 片交付并附 span 覆盖状态表
   （model 已接于 runner、tool 已在 agent 既有、run/agent 待第 2 片 / S25-DEFER-1）；
   交付票 criterion 2 解释的确认记录落在 review-verdict 的 completionAudit；
   状态文档（v3-p1-graph-status.md）由编排方落笔，本片不代写。✅

## 4. 审查

review 结论见 `docs/design/v3-p7-g9-review-verdict.json`（派发子代理自查，独立性披露见该文件
independenceDisclosure）。
