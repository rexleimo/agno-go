# 切片 24 REFACTOR（母约 §7 G6：Durability 持久化档位）

工作项 `work-p3-graph-g6-durability` / 契约 `docs/design/v3-test-scope-p1-graph-slice24.json`（D1–D9）。

## 1. REFACTOR 结论：无独立重构动作，最终字节 = GREEN 字节

GREEN 交付的实现直接采用了契约 M2 设计期副本（/tmp/g6exp）预演过的形状（声明集中在
durability.go、接线集中在 scheduler 的消费者侧三段：`beginCommits`/`record`/`finishCommits`），
没有可合并的重复；REFACTOR 阶段零代码改动。
最终字节：`durability.go da57a17e…`、`graph.go 98b8bdad…`、`scheduler.go 4641de52…`、
`p3g6_durability_test.go bdeb8ef9…`。最终门禁复测：build/vet/gofmt 干净，全包 `-count=1` 与
`-race` `ok`（`receipt:d8b78d99…` / `receipt:898ad3f9…`）。

结构代价对账（契约 baselineBytes.counters）：导出符号 35 → **41**（+3 类型 +2 Option +1 const
块，与 M2 预测一致）；非测试 LOC 1085 → **1219**（M2 副本实测 1218，实接线后 +1 行来自
record 调用点注明的「Sync 时序在此」两行注释，判据 ≤1500）；`sync.` 计数 **0** → **0**
（Async 冲刷器只共享 commits 通道，零锁模型未破）。

## 2. 完整变异矩阵（`scripts/mutation/p3g6-durability.mjs`，仓库内可复现）

命令 A = 绑定场景命令（`TestP3G6_`，`-race -v`），命令 B = 全部既有判据
（`-run 'TestP1|TestP2G3'`，`-race -v`）。基线：`scheduler.go 4641de52…`、
`durability.go da57a17e…`；每条变异后 `restored=true`、`finalCheck identical=true`。

| 变异 | 形态 | 命令 A | 命令 B | 判定 |
|---|---|---|---|---|
| m1 | Sync 提交点挪到下一轮完成之后（条目滞后一笔提交、最后一笔永不落） | exit 1，杀红 D2/D3/D4/D5/D6/D9 | exit 0 | **杀红** |
| m2 | 默认档与 Sync 档改判 Exit 积累（Sync 分支失配落 default，退出又不冲） | exit 1，杀红 D2/D3/D6/D7/D9 | exit 0 | **杀红** |
| m3 | Exit 的退出落齐整段摘掉（丢条目） | exit 1，杀红 D4/D6/D7 | exit 0 | **杀红** |
| m4 | Async 的退出冲刷摘掉（close+收敛等待整段移除、冲刷体空转；复合变异，保证丢条目判定确定性） | exit 1，杀红 D5/D6 | exit 0 | **杀红** |
| m5 | sink 错误吞成 nil（Sync 即时失败改交 nil；Async/Exit 的退出上交判据钳死） | exit 1，杀红 D7 | exit 0 | **杀红** |
| m6 | 安全阀路径丢提交流（设了非默认步数预算就整体跳过提交——撞阀清空条目的同形后果） | exit 1，杀红 D9 | exit 0 | **杀红** |
| m7 | Seq 盖章摘掉（序号恒 0，对账破） | exit 1，杀红 D2/D3/D4/D5/D6/D9 | exit 0 | **杀红** |
| m8 | 常量表零值错位（Sync 不再是 iota 零值，默认档落死区） | exit 1，杀红 D1/D2 | exit 0 | **杀红** |
| e1 | 失败路径上 Exit 档已完成条目的交付摘掉 | exit 0 全绿 | exit 0 | **等价变异 —— 登记为观察上限** |
| e2 | Async 档 sink 失败的 runErr 优先钳制摘掉 | exit 0 全绿 | exit 0 | **等价变异 —— 登记为观察上限** |

契约 completionCriteria 第 2 条点名的七种形态逐条覆盖：Sync 提交点时序破坏（m1）、Exit 退出
落齐摘掉（m3）、Async 退出冲刷摘掉（m4）、sink 错误吞成 nil（m5）、默认档改判 Exit（m2）、
安全阀路径清空/丢提交流（m6）、Seq 盖章摘掉（m7）。m8 另行钉 D1 的结构前提；D8 是反向锚
（对「引入默认 sink / nil panic」判红），无杀红变异属预期。全部杀红变异的 B 命令
（切片 1–22 既有判据）保持 exit 0——变异只伤本片行为，不扰既有片。

### 2.1 e1/e2 等价性的读法（观察上限）

- **e1**：D 行没有一行观察「runErr != nil 时 Exit 档照样落齐」（D9 只钉 Sync+撞阀的组合）。
  designConsequence 第 4 条的「已完成节点的条目在失败退出时照常交付」这半边在 Exit 档上是
  未钉语义：现在就工作（实现如此），摘掉不判红。要钉它需要「Exit 档 + 中途节点失败」的新行，
  本片不追加。
- **e2**：D7 只钉 Sync/Exit 两档的 fail-closed；Async 档 sink 失败的语义（冲刷器记首错、
  退出时 runErr 优先上交）无行观察，与契约 observabilityLimit 第 2 条（掩蔽次序未单独立行）
  同源登记。

### 2.2 一次既有夹具的负载敏感 FAIL（如实登记，附排查记录）

整包 `-count=5 -race` 防碰巧绿首跑（`receipt:259b1e27-05f8-488e-a35d-749761894615`，exit 1）
出现 `TestP1R17_CancellationWinsOverStepLimitDuringDispatch` 一次判红，失败点在夹具自己的
前提自证（`p1r17_cancel_normalization_test.go:211`）：「取消之前 Run 已经交出（err=step limit
100 reached…）——派发窗口没建立，本行前提不成立」。处置按契约 carriedItems S24-STD-1：

1. 单测复跑：该 Test `-count=5 -race` 单独跑 5/5 PASS；
2. 整包复跑：`-count=5 -race` 复跑 exit 0（`receipt:21238297…`），另两轮全包 race 复跑亦绿；
3. 归因排查：失败路径在 dispatch 的步数判据与夹具窗口的建立竞态，本片字节对该路径零改动
   （nil sink 下 record/finishCommits 均 nil 早退）；本片绑定命令（含 `-count=30 -race`）
   从未再现；
4. 基线对照实验：把**切片 22 收口基线字节**（graph.go `72974bf6…` / scheduler.go `873f4a82…`，
  11 份既有测试文件，不含本片任何字节）放进隔离副本 /tmp/g6m1，与带缝副本 /tmp/g6wired 在
  同机同负载（6 个 CPU 压载循环模拟并行切片的读跑负载）下交替跑整包 `-count=5 -race` 各 3 轮：
  **基线字节第 2 轮判红 2 条，带缝字节 3 轮全绿**——该夹具的窗口竞态在基线上就在高负载下
  存在，与本片无关。

结论登记为**既有 R17 夹具的负载敏感前提竞态**（S24-STD-1 的同族观察，切片 22 收口时的
不可复现裸 FAIL 与此相邻）：成因是「主测 goroutine 在窗口建立前被饿死、消费者先行撞阀」，
只在机器高负载（多切片并行读跑）时偶发。本片不改既有测试（契约 allowedTestSeam.modify 为空），
处置留待该夹具的独立裁决；不在本片顺手加 flake 容忍。

## 3. 契约 completionCriteria 逐条对账

1. redProtocol 两段各自留 receipt：第一段 `receipt:3d22b67c…`（exit 1，7 红 D2–D7/D9 +
   2 反向绿 D1/D8，零编译失败）→ 第二段 `receipt:04eb9c8c…`（exit 0，9 Test 全绿）；
   两段之间测试文件零改动（`p3g6_durability_test.go` 两段同字节）。✅
2. 牙齿由变异证明：m1–m8 杀红（判据 2 点名的七种形态逐条覆盖 + D1 结构前提）、e1/e2 等价
   登记为观察上限。✅
3. 既有 11 份测试文件与 policy.go 零改动，逐文件 hash-object 复核与契约 baselineBytes 全同
   （green 文档 §3）。✅
4. 门禁全 exit 0：build / test / -race / vet / gofmt + 邻近回归（green 文档 §3）。✅
5. 结构判据：`sync.`=0、LOC 1219≤1500、导出符号恰 41（+1 分解见契约 docScanDecomposition）。✅
6. 绑定命令在最终字节上重跑并取新回执（`receipt:3b49530f…`，变异矩阵恢复并复核哈希之后；
   GREEN 本体 `receipt:04eb9c8c…`）；`-race` 无 DATA RACE。✅
7. 完成后回写：母约 §7（Durability 三档已交付标注 + 缝形状 + StepLimit 已随切片 13 交付）、
   §10 交付表 P3 行（G6 半边交付、G5 待 R19-Q2 裁决）、证据文档四份（red/green/refactor/
   verdict）。状态文档 §4/§5 的回写属主会话（该文件为 orchestrator 所有，本片不碰）。✅（本片范围）
8. 防碰巧绿：绑定命令 `-count=30 -race`（`receipt:dc787347…`）与整包 `-count=5 -race`
   （`receipt:21238297…`）均 exit 0；首跑撞 R17 既有夹具竞态一次，已按 S24-STD-1 处置并
   登记（§2.2）。✅

## 4. 观察限度（契约 observabilityLimit 的本片登记）

- Sync 与 Async 的差别只有 Sync 的正向担保可断（D3 后继体采样）；Async 的 mid-run 可见性
  不作断言（任何一次读到与否都不足证），由 D3 正向行 + D5 落齐行共同夹住，m1/m4 的联合面
  佐证。
- 失败路径与 sink 错误叠加时的掩蔽次序（runErr 优先）未单独立行：实现按 designConsequence
  第 4 条执行（`finishCommits` 只在 runErr==nil 时上交 sink 错误），e2 登记为无行观察。
- Async 冲刷器的调度延迟、提交吞吐、背压行为（慢 sink 让消费者在 handoff 处等）不在公共面
  担保范围，不设行。
- Checkpoint 条目只承诺 Seq/Node/Output 三字段最小集；join 目标 Output 与激活输入同源别名
  （S19-STD-1 延伸），条目对大输出的保真度以「夹具可取回同值」为限（D6 的同值断言口径），
  不钉深拷贝语义。

## 5. 审查

review 结论见 `docs/design/v3-p3-graph-g6-review-verdict.json`（独立性披露见该文件
independenceDisclosure）。
