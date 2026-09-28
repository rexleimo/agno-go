# 切片 26 REFACTOR（母约 §6 G5：动态扇出 Send + AddJoinSend）

工作项 `work-p3-graph-g5-send` / 契约 `docs/design/v3-test-scope-p1-graph-slice26.json`（D1–D9）。

## 1. REFACTOR 结论：无独立重构动作，最终字节 = GREEN 字节

GREEN 交付的实现承自契约 M2 预演副本（/tmp/g5exp）的形状（声明集中在 send.go、接线集中在
scheduler 消费者侧：`joinSendTables` 建表 / `absorbSends` 记账 / `joinSendReady` 归零激活），
没有可合并的重复；REFACTOR 阶段零代码改动。与预演副本的字节差恰一处：`send.go` 的 `AddJoinSend`
文档注释行 `//（R19-Q2 裁决…` 被 gofmt 判为需在 `//` 后补空格（doc 注释规范化）——预演副本自身
不是 gofmt 干净字节，本片以门禁为准修正（`gofmt -l pkg/hno/graph` 无输出）。
最终字节：`send.go dcd04661…`、`graph.go 7d7bb8d8…`、`scheduler.go fcd552c5…`、
`p3g5_send_test.go c003a537…`。最终门禁复测：build/vet/gofmt 干净，全包 `-count=1` 与
`-race` `ok`（`receipt:4a0865d4…` / `receipt:3c8f5315…`），绑定命令收尾复跑
`receipt:b5f24ece…`。

结构代价对账（契约 baselineBytes.counters）：导出符号 41 → **45**（+类型 `Send`/`Sender`
+ `func SenderFunc` + 方法 `(g *Graph) AddJoinSend`，与 expectedFinalExportedSymbols 一致）；
非测试 LOC 1219 → **1435**（graph +11、scheduler +118、send +87，恰为契约 expectedFinalLoc，
判据 ≤1500）；`sync.` 计数 **0** → **0**（pending-Send 记账与屏障全在唯一消费者手里，
零锁模型未破）。

## 2. 完整变异矩阵（`scripts/mutation/p3g5-send.mjs`，仓库内可复现）

命令 A = 绑定场景命令（`TestP3G5_`，`-race -v`）；命令 B = 先序既有族
（`-run 'TestP1G_|TestP1R1[3-9]'`，69 个 Test，`-race -v`）。基线：
`scheduler.go fcd552c5…`、`graph.go 7d7bb8d8…`；每条变异后 `restored=true`、
`finalCheck identical=true`。

| 变异 | 形态 | 命令 A | 命令 B | 判定 |
|---|---|---|---|---|
| m1 | 生产者 Sender 断言摘掉（每次激活都走 Node.Run，sends 被 Run 视图丢弃） | exit 1，杀红 D1/D3/D4/D7/D9 | 仅 TestP1R17_Cancellation 判红（环境性，§2.1） | **杀红** |
| m2 | 屏障凑齐判据摘掉（计数归零不再触发 joinSendReady，永不激活 target） | exit 1，杀红 D1/D3/D9 | 仅 TestP1R17_Cancellation 判红（环境性，§2.1） | **杀红** |
| m3 | AddJoinSend 可达性贡献摘掉（source 不再播种为可达） | exit 1，杀红 D1/D2/D3/D4/D5/D7/D9 | exit 0 | **杀红** |
| m4 | Send 未注册检查摘掉并静默丢弃（契约 forbiddenShortcuts 形状：整体校验移除 + 派发跳过未注册目标） | exit 1，杀红 D7 | exit 0 | **杀红** |
| m5 | 零派发改判激活（凑齐判据放宽为「非 Send 完成或归零都激活」——空聚合也激活一次） | exit 1，杀红 D1/D2/D3/D4/D8/D9 | 仅 TestP1R17_Cancellation 判红（环境性，§2.1） | **杀红** |
| m6 | 多波守卫复位摘掉（计数 0→1 不复位 sendFired，第二波被「本波已激活」跳过） | exit 1，杀红 D9 | exit 0 | **杀红** |
| e1 | joinSendReady 的延迟复位改为就地复位 | exit 0 全绿 | exit 0 | **等价变异 —— 登记为观察上限** |

契约 completionCriteria 第 2 条点名的五种形态逐条覆盖：生产者断言摘掉（m1）、屏障凑齐判据
摘掉（m2）、可达性贡献摘掉（m3）、Send 未注册检查摘掉（m4）、零派发改判激活（m5）。m6 另钉
D9 的多波牙齿；D6/D8 是反向锚（对「混用两类声明判据 / 引入默认行为改变」判红），其牙齿分别
由既有 TestP1R19（D6）与既有 12 份测试文件零改动全过（D8）担保，无本片专属杀红变异属预期。
全部杀红变异的 B 命令（先序 69 判据）除 §2.1 的环境项外保持 exit 0——变异只伤本片行为，
不扰既有片。

m3 的杀红面读法：可达性上声明边本身早已入邻接表（D8 的 plain→reduce 因此不依赖播种），
摘掉的是「source 自身被视为可达」那半边——所以判红的恰是 map-reduce 形状（sum 只被运行期
Send 喂养）的那 7 行，D8 不在其列，与实现分工一致。

### 2.1 B 命令上 TestP1R17 的环境性复发（S24-STD-2，如实登记）

m1/m2/m5 三条变异的 B 命令各出现一次 `TestP1R17_CancellationWinsOverStepLimitDuringDispatch`
判红（m3/m4/m6 的 B 干净），失败点即该夹具自己的前提自证（「取消之前 Run 已经交出——派发
窗口没建立」，切片 24 refactor 文档 §2.2 已定位的同一负载敏感竞态）。变异矩阵连跑 14 轮
`-race -v` 属重负载场景。处置按承接指令的 S24-STD-2 口径：

1. 字节核验：变异脚本基线与 `finalCheck identical=true` 对 `scheduler.go fcd552c5…`、
   `graph.go 7d7bb8d8…`（= 本片最终字节）逐字节复核，判红非字节漂移；
2. 静置约 30 秒后在最终字节上单跑 B 命令：exit 0、69 Test 全 PASS
   （`receipt:2c648519-9e54-44ca-ab3d-43a13d365a1c`）；
3. 本片对 dispatch 步数判据/取消路径的语义零改动（absorbSends 只在既有 dispatch 循环之外
   追加 pending 项）；本片绑定命令（含 `-count=30 -race`）与整包 `-count=5 -race` 均一次过、
   从未再现。

结论：S24-STD-2 的已知环境复发（仅 TestP1R17 族、仅重负载、静置后干净），非本片引入；
不改既有测试（allowedTestSeam.modify 为空）。

同族复发另有一次落在收尾复测上：全部门禁与写回完成后的最终字节全包 `-count=1`
复跑一次判红同一 Test（同一前提自证文案，`p1r17:211`）；按同口径处置——字节先复核
（`graph.go 7d7bb8d8…`/`scheduler.go fcd552c5…` 与收口字节全同）、静置约 30 秒复跑一次
exit 0（`receipt:c57d2a4b-08af-4a7f-9f1c-e90534bc93ac`）。两次复发均为仅该族、静置即绿，
登记不变。

### 2.2 e1 等价性的读法（观察上限）

joinSendReady 把聚合器复位推迟到「同一次归零触发的全部就绪 target 都取走聚合之后」，是为了
多个 target 共享同一 source 且同波就绪时不被先处理者清空。D1–D9 没有任何一行构造该形状
（D9 的两个声明在 OBS-9 探针里是同 source 双 target，但本片 D 行按契约验收映射只钉单 target
的逐波形状）——就地复位因此不可观察，登记为观察上限；要钉它需要「同 source 双 target 同波」
的新行，本片不追加。

## 3. 契约 completionCriteria 逐条对账

1. redProtocol 两段各自留 receipt：第一段 `receipt:259ee969…`（exit 1，5 红 D1/D3/D4/D7/D9 +
   4 绿 D2/D5/D6/D8——D5 是第一段 API 面的预期绿、D2 是「无派发发生」下的真空绿、D6/D8 反向锚，
   全部行为红、零编译失败）→ 第二段 `receipt:dfab55ac…`（exit 0，9 Test 全绿）；两段之间
   测试文件零改动（`p3g5_send_test.go` 两段同字节 `c003a537…`）。✅
2. 牙齿由变异证明：判据点名的五种形态 m1–m5 逐条杀红、m6 另钉 D9、e1 等价登记为观察上限。✅
3. 既有测试文件零改动：12 份（`graph_test` + `p1r13–p1r21` 8 份 + `p2g3` + `p3g6`，加上本片
   新增共 13 份在册）逐文件 `git hash-object --no-filters` 复核与切片 24 收口字节全同；
   `policy.go`/`durability.go` 与契约 baselineBytes 全同（green 文档 §3）。票面表述「13 份」
   与实数 12 份的差异在此说明：契约 allowedTestSeam.modify 的义务（一份都不许动）已按实数履行。✅
4. 门禁全 exit 0：build / test / -race / vet / gofmt + 邻近回归 agent/runner/session-contract
   （green 文档 §3）。✅
5. 结构判据：导出符号恰 45、非测试 LOC 1435≤1500、`sync.`=0（green 文档 §3）。✅
6. 绑定命令在最终字节上重跑并取新回执（`receipt:b5f24ece…`，变异矩阵恢复并复核哈希之后；
   GREEN 本体 `receipt:dfab55ac…`）；`-race` 无 DATA RACE。✅
7. 写回：母约 §6 示例改写为 SenderFunc+AddJoinSend 可跑通形态并标注已交付（切片 26）、交付表
   P3 行标记全清（G5+G6）；证据文档四份（red/green/refactor/verdict）。状态文档 §4 由编排方
   收口（本片不碰 `v3-p1-graph-status.md`）。✅（本片范围）
8. 防碰巧绿：绑定命令 `-count=30 -race`（`receipt:c85e8a42…`）与整包 `-count=5 -race`
   （`receipt:e487bbfd…`）各一次 exit 0。✅

## 4. 观察限度（契约 observabilityLimit 的本片登记）

- Send 目标注册性只能运行期判（D7 是运行期判据）；构建期对 join-send source 的可达性播种是
  乐观先例（无法反驳「会有 Sender 指名它」），与「可达不看谓词」同一取向。
- 多波之间 target 激活的相对次序由确定性的波完成次序决定（D9 用条件环 + reduce→fan 回边建立
  happens-before，不依赖真实时序）；并行波的全局次序不断言。
- 聚合输入按派发序（slice）是本片钉的形状（D1/D9 断序合法，因为它是 slice 不是 map 迭代）；
  若日后改为按完成序，须回契约改 D1 而非静默。
- 多 source 交错多波（source A 的第 2 波在 source B 的第 1 波在途时开派）的聚合切分语义
  未钉（当前实现按「各 source 计数独立归零 + 同波就绪判据」自然切分）；单 source 多波（D9）
  与多 source 同步波（OBS 形状）已钉。e1 的同波多 target 一致性同属观察上限（§2.2）。
- join-send 激活计一步、其完成照常产检查点条目（`absorbSends` 在 `record` 之后）；本片
  零专属条目种类，durability/policy 字节未动。

## 5. 审查

review 结论见 `docs/design/v3-p3-graph-g5-review-verdict.json`（独立性披露见该文件
independenceDisclosure）。
