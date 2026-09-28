# 切片 27 Refactor 与契约对账（母约 §9 G7：图引擎 HITL Interrupt/Resume 核心）

工作项 `work-p5-graph-g7-hitl-core` / 契约 `docs/design/v3-test-scope-p1-graph-slice27.json`。
GREEN 判据见 `docs/design/v3-p5-graph-g7-green.md`；RED 观察见 `docs/design/v3-red-observation-p5g7.md`。

## 1. 字节经济：从 1860 到 1800（如实登记）

契约以副本实测 1562 定了 1800 上限并估算「+Send 后约 1780」。两处测量口径的偏差让首版 GREEN 字节
落在 **1860**（超 60）：

1. 契约引用的 1562 是设计期测量；留存的实验副本 `/tmp/g7exp` 今日实测非测试 LOC **1644**（hitl 325 +
   wiring 100 + 基线 1219）。同一缝落在切片 26 收口字节（基线 1435 = 1219 + Send 216）上，理论值即
   1435 + 425 = 1860——Send 缝与 HITL wiring **没有可重叠的行**（`newScheduler` 必须补 send 三表），
   「约 1780」的估算少计了这一点。
2. 处置：不申请二次放宽（放宽是编排方的登记权），做了一轮**纯注释/空行的字节经济**，行为字节零改动：
   - 文档压缩到承载语义的最小行（每条保留事实不变：三语义、七条 designConsequence 的落点、
     fail-closed 边界全部在），删除的只有重复句与换行；
   - `maps.Clone/slices.Clone/slices.Sorted(maps.Keys(…))` 替换手写拷贝/排序循环（Go 1.24）；
   - 修复了实验副本里 `Resume` 对 `beginCommits` 的**双重调用**（第二调用会在 Async 档重建 commits
     通道、泄漏第一只冲刷 goroutine——十五个观察面未覆盖 Async+Handoff 组合，属副本缺陷非语义）；
   - 删除实验副本里 `scheduler.responses` 死字段（只写不读；响应集经 ctx 传递，该字段无任何读者）。
   过程中 GREEN 先后取自 `receipt:f259eec2-57d3-4075-9faf-2b7f0c160839`（1860 字节）与最终
   `receipt:ad6c744c-e211-442f-8232-f088ebf3d295`（1800 字节）；测试文件两次均零改动。
   最终结构判据：导出符号 **60**、非测试 LOC **1800**（policy 138 + graph 639 + durability 146 +
   scheduler 499 + send 87 + hitl 291）、`sync.` **0**。

对实验副本（/tmp/g7exp）的第三处有意偏离：`Resume` 的校验次序从「缺失 → 未知 → schema 三遍扫」并为
「逐中断缺失+schema 一遍扫 → 未知一遍」。仅混合故障场景的错误优先级不同（如既有未知 ID 又有类型错的
响应集，现在先报 schema 错），无任何 D 行钉该次序；恰好覆盖与挂起保留两条 fail-closed 语义不变。

## 2. 变异矩阵（契约判据 2：每行牙齿由变异证明）

执行体 `scripts/mutation/p5g7-hitl.mjs`（p3g6 骨架：锚唯一性预检、逐字节恢复自检、判红只认 `-v`
Test 名）。A = 绑定命令（`TestP5G7_`，-race -v）；B = 既有判据
（`TestP1G_|TestP1R1[3-9]|TestP2G3_|TestP3G[56]_`，-race -v）。基线 blob：`hitl.go 75756a14…`、
`scheduler.go dfd6c1ba…`；`finalCheck identical=true`（逐字节恢复复核）。

| id | 变异 | A exit | A 判红（TestP5G7_） | B exit |
|---|---|---|---|---|
| m1 | 挂起捕获摘掉（中断退化为普通错误） | 1 | D1–D13/D15 全族（无任何挂起面，同 RED 形状） | 1* |
| m2 | schema 校验摘掉（validateResponse 恒放行） | 1 | D2 | 0 |
| m3 | 幂等判据摘掉（Resume 不消费挂起账，重复恢复重跑图） | 1 | D5 | 0 |
| m4 | Handoff 档错走 Rerun（交接分支失配落 default） | 1 | D4（callsB 偷跑） | 0 |
| m5 | EntryInterrupt 写入摘掉 | 1 | D7 + D13（sink 失败半边随之失据） | 0 |
| m6 | 重试不吞中断的判据反转（中断可重试、普通错误短路） | 1 | D8（calls==5） | 1（P2G3 重试族随反转语义判红——B 侧额外信号，与契约预期同向） |
| m7 | 预算累计改每次重置（Resume 从 0 计步） | 1 | D12（calls==2 半边） | 0 |
| e1 | （预期全绿）unknown-ID 错误也包 `ErrInvalidResponse` | 0 | — | 1* |
| e2 | （预期全绿）Suspension.Error 文案不再列 ID 清单 | 0 | — | 1* |

\* m1/e1/e2 的 B 侧 exit 1 均只含 `TestP1R17_CancellationWinsOverStepLimitDuringDispatch` 一条——
即 S24-STD-2 在册的既有夹具负载敏感前提竞态（B 在 A 之后连跑、机载负载窗口内触发），与本片变异
无关：字节恢复后单独重跑该族 exit 0（`receipt:8b1abc0d-619f-43f0-b584-e7afeaa2a7e2`，6/6 PASS）。
本片不改该夹具（allowedTestSeam.modify 为空），处置留待其独立裁决。

等价变异登记（契约判据 2「等价变异登记」）：e1 钉的是 D6 未钉的方向——该错误只要求「点名 ID +
挂起保留」，哨兵归属不在公共面；e2 钉的是挂起错误文案——待答清单的结构化证据在 `errors.As` 取到的
`Suspension.Interrupts`，文案形状属 forbiddenObservations 的「不是证据面」。两者 A 侧全绿为预期，
不算杀红。

## 3. completionCriteria 逐条对账

1. **redProtocol 两段各自留 receipt，两段之间测试文件零改动** ✅：RED
   `receipt:e428c1fc-090c-4fc6-b48a-af4242e59762`（exit 1，14 行行为红 + D14 反向锚绿）、GREEN
   `receipt:ad6c744c-e211-442f-8232-f088ebf3d295`（exit 0）；测试 blob `7587689b…` 两段间一个字节
   未动。夹具迭代的一次中途 exit 1（f7b60efb）已在 RED 文档 §2.1 如实登记。
2. **每行牙齿由变异证明** ✅：m1–m7 覆盖判据 2 点名的七种形态且全部杀红；e1/e2 登记为等价（§2）。
3. **既有全部测试文件零改动** ✅：13 份既有测试 + policy.go + send.go 逐字节复核（hash 清单见
   green 文档 §3），与切片 26 收口基线全同。
4. **门禁全 exit 0** ✅：build/vet/gofmt、`./pkg/hno/graph/...` -count=1（021f711e）、-race
   （23f6b4e3）、邻近 agent/runner/session-contract（60f20a74）。
5. **结构判据** ✅：导出符号恰 **60**；非测试 LOC **1800 ≤ 1800**（压线达成，过程见 §1）；`sync.` **0**。
6. **绑定命令在最终字节重跑取新回执；-race 无 DATA RACE** ✅：ad6c744c（-race -v）。
7. **写回** ✅：母约 §9 引擎核心已交付标注（三语义落点 + Mode 字段补充 + 切片拆分登记）与 §10 交付表
   P5 行（第 1 片 delivered、第 2 片待侧车）已落；状态文档 §4 由编排方收口（本片不碰）。
8. **防碰巧绿** ✅：绑定 `-count=30 -race`（207c88b2）、整包 `-count=5 -race`（0dcf9dd7），各一次
   exit 0（本片未触发 S24-STD-2；变异矩阵运行中的触发见 §2 已登记）。

## 4. carriedItems 与观察上限的承接

- **D1 裁决（OPT-1）**：本片未动 `Session` 对外 JSON；`internal/session` 零触碰（邻近回归全绿）。
  第 2 片（会话侧车 + 派生视图 + 跨进程恢复接线）契约化时引用本契约 carriedItems。
- **R19-Q1**：挂起为显式可恢复态（`*Suspension` 实现 error），与「凑不齐交空结论」的形状判然有别；
  若 R19-Q1 日后裁决为 fail-closed，不改动本片形状。
- **S19-STD-1**：`Resume` 对旧 `Suspension` 的 Values/Completed 做 `maps.Clone/slices.Clone` 防御性
  拷贝，从源头不制造别名；Rerun/Handoff 后 Join 仍走既有 joinReady 键控路径（D9 实测）。
- **S24-SPEC-1（恢复/重放）**：本片交付引擎内恢复 + EntryInterrupt 持久化；跨进程重建入口
  （Graph 重建 + pending 恢复）由第 2 片定型，`observabilityLimit` 第 3 条（Resume 只在同一 Graph
  实例或其等价重建上担保）随之闭合。
- **S18-SPEC-2 / S19-STD-6**：未提交状态与 lint 欠项原样沿用；本片全部字节在 gofmt 与 go vet 通过的
  状态上收口，未对工作树执行任何 git 写操作（无 checkout/restore/stash/commit/push）。
