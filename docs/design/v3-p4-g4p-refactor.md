# 切片 34 REFACTOR（P4 第 3 片，母约 G4：剩余五模式生产者接线）

工作项 `work-p4-g4-producers` / 契约 `docs/design/v3-test-scope-p4-g4-producers.json`（D1–D9）。
RED 见 `v3-red-observation-p4g4p.md`，GREEN 见 `v3-p4-g4p-green.md`。

## 1. 变异矩阵（scripts/mutation/p4g4-producers.mjs，12 条 = 11 杀红 + 1 等价）

绑定命令（在整份仓库副本上逐字复跑，注入不落工作树）：
`go test ./pkg/hno/agent -run TestP4G4P_ -count=1 -race -v`。

| 变异 | 注入 | 接手行（实测判红） | 声明 | 判定 |
|---|---|---|---|---|
| m-table | 模式表漏加 StreamValues | D1, D4, D5 | 同 | ok |
| m-degrade | 归一化静默跳过未知模式 | D2 | 同 | ok |
| m-turn-gate | 逐回合发射摘掉 Updates 族门 | D4, D5, D6, D7, D8 | 同 | ok |
| m-per-chunk | 发射点挪进分块循环（turnObserver 摘除） | D3, D4, D5, D6 | 同 | ok |
| m-debug-union | 摘掉 Debug 并集分支 | D6 | 同 | ok |
| m-debug-messages | Debug 暗含 Messages | D6 | 同 | ok |
| m-custom-gate | writer 恒安装 + 包装器恒挂载（两处同摘） | D7 | 同 | ok（首版只摘安装门，实测 NO_TEETH——见 §2） |
| m-custom-payload | subtype/data 互调 | D7 | 同 | ok |
| m-values-keys | Values 载荷退化成 Updates 键形 | D4, D5 | 同 | ok |
| m-turn-count | 回合序数不递增 | D3, D4, D5, D6 | 同 | ok |
| m-label | checkpoint label 改裸序数 | D5, D6 | 同 | ok |
| m-ledger-order | emit 内记账与上通道换序 | （无——全绿） | 等价变异，不充当牙齿 | ok（等价登记成立） |

行覆盖核对：D1←m-table；D2←m-degrade；D3←m-per-chunk/m-turn-count；D4←m-turn-gate/m-per-chunk/
m-values-keys/m-turn-count/m-table；D5←m-turn-gate/m-per-chunk/m-values-keys/m-turn-count/m-label；
D6←m-turn-gate/m-per-chunk/m-debug-union/m-debug-messages/m-turn-count/m-label；
D7←m-custom-gate/m-custom-payload；D8←m-turn-gate——**每行至少被一个变异杀死**，无未声明杀、
无归属泄漏（census 的逐条 PASS/FAIL 名单见 /tmp/p4g4p-matrix/raw 转录）。

## 2. 首版 m-custom-gate 的 NO_TEETH 与修正（如实登记）

首版变异只摘 Execute 里的安装门（`if e.emitter.enabled(run.StreamCustom)` → 恒安装），实测
注入后全绿。根因：D7 的负向子用例走 **Messages-only** 运行，而 Messages-only 在
`streamExecutor` 处根本不挂包装器（Tasks/Custom 两门全关 → nil executor）——Execute 里的
安装门从未被执行。「writer 缺席」是**两道构造**合起来的性质：executor 选择（哪条运行挂包装器）
+ 安装门（挂上的包装器装不装 writer）。修正后的变异两处同摘，Messages-only 的 handler 写入从
「报错」退成「writer 在、族门关 ⇒ 静默吞」，D7 判红。教训：fail-closed 的牙齿要数清楚**全部**
构造点，只变异最后一个门会让前面的短路把变异整条短路掉。

## 3. 等价变异登记（m-ledger-order）

`emit` 先记账后上通道；换序成「上通道成功后再记账」在全部 8 行上不可分辨——夹具一律**先排空
通道再读 `done.Output.Events`**，两个观察面的读取之间存在 happens-after，账面/通道的先后在
观测上消失。这是契约 notes 预登记的等价形状，实测确认成立。它不是缺陷：emit 内的先后只有
「并发消费者边收边读账本」才可分辨，那不在本片公共语义内（账本在 Done 之后才承诺完整）。

## 4. 收口门禁（全部在最终字节上重取回执）

| 门禁 | 结果 | receipt |
|---|---|---|
| 变异矩阵全量（control 绿 + 12 条按声明） | exit 0（违规 0） | 本轮 receipt（见契约 evidenceRefs） |
| 绑定命令 `-count=1 -race -v` | 8 `--- PASS` | `receipt:4082e7da-164d-4c67-b25a-99b510f4c327` |
| `-count=30 -race` / `-count=5 -race` | exit 0 | `receipt:029f1102…` / `receipt:c6326052…` |
| build/vet/gofmt/邻近回归 | 全清 | `receipt:f9d72be6…` / `receipt:03ae0b3a…` |
| 锚 41 条 + graph md5 前后同值 | 全同 | `receipt:f7885f4b-e29a-4a01-85ef-242ace35e6d6` |

## 5. 重构保持的事实

- `streamEmitter` 仍是「族门 + 账本 + 序列号」的唯一出口；逐回合族计数（turn/contentSoFar）
  与 content 序列号同协程推进，零新增 goroutine、Messages/Tasks 路径零锁。
- Custom 是唯一带互斥锁的发射路径（handler 协程发起），相位论证（工具批次期间内核协程停在
  `wg.Wait`）写进 `customWriter` 的双语注释。
- `run_completed` 记账事实未动：无条件进账本、永不上通道（D3/D5/D7 的账本断言沿用切片 31 形状）。
