# 切片 27 RED 观察（母约 §9 G7：图引擎 HITL Interrupt/Resume 核心）

工作项 `work-p5-graph-g7-hitl-core` / 契约 `docs/design/v3-test-scope-p1-graph-slice27.json`（D1–D15）。
测试文件：`pkg/hno/graph/p5g7_hitl_test.go`（`package graph_test`，只用导出 API；夹具自备内存
Checkpointer 与调用计数，互斥锁属夹具不属引擎）。

## 1. redProtocol 两段制的第一段

按契约 redProtocol：第一段落 API 面——`hitl.go` 的 `Interrupt/InterruptMode(ResumeRerun/ResumeHandoff)/
Suspension/WaitingNode` 三哨兵（`ErrSuspended/ErrNothingToResume/ErrInvalidResponse`）与构造器
（`RequestInterrupt/InterruptResponse`）、`Resume` 签名（接线前按幂等语义如实回答 `ErrNothingToResume`）、
`validateResponse` 子集校验；`durability.go` 的 `EntryKind` 条目种类（`EntryCompletion`/`EntryInterrupt`）与
`Checkpoint` 的 `Kind/Input/Interrupt` 三字段；`scheduler.go` 的挂起态字段声明（`suspended/waitings/
suspendSinkErr`）。消费者接线（中断捕获、挂起构造、Run 返回、Resume 双档调度、检查点写入）不属于
第一段。绑定命令可编译，行为行按契约预判的形状真实判红：节点返回的中断只是普通错误、`Resume` 恒答
`ErrNothingToResume`、挂起条目无从落流——全部是行为红，零编译失败。

本片绑定场景命令逐字（所有 receipt 都是这一条命令、直接取，无 `sh -c` 包装）：

```bash
go test ./pkg/hno/graph -run TestP5G7_ -count=1 -race -v
```

## 2. RED 实测（receipt:e428c1fc-090c-4fc6-b48a-af4242e59762，exit 1）

**判红 14 行**（D1–D13、D15，全部落在「无挂起面」的公共面形状上）：

| 行 | 失败形状（摘） |
|---|---|
| D1 | `errors.Is(err, ErrSuspended)` 不成立——中断被当普通错误交回，无 `*Suspension` 可取 |
| D2 | 校验失败的 `Resume` 返回 `ErrNothingToResume` 而非 `ErrInvalidResponse`（签名桩的幂等回答） |
| D3 | Rerun 档 `Resume` 无处可恢复，节点体没有第二次执行 |
| D4 | Handoff 档同形：没有挂起可交接 |
| D5 | 首次 `Resume` 即 `ErrNothingToResume`（不存在可恢复的挂起） |
| D6 | 少给/多给两个方向都没有点名 InterruptID 的错误（没有响应路由） |
| D7 | sink 收到 0 条条目（`EntryInterrupt` 条目无从落流）；普通完成对照半边天然成立 |
| D8 | 中断被重试 5 次、当普通错误交回（重试不吞中断的判据缺失） |
| D9 | 并行双中断没有 `*Suspension` 可取，双答 `Resume` 无处可恢复 |
| D10 | 重悬链不存在（第一次 `Resume` 即无处可恢复） |
| D11 | 空 InterruptID 的错误文案不点名节点（中断载体没有被消费者辨认） |
| D12 | 恢复段交 `ErrNothingToResume` 而非预算阀错误（预算跨恢复无从谈起） |
| D13 | 无挂起路径，sink 错误与挂起的双 Is 不成立；Exit 对照半边同形 |
| D15 | 重跑体经 `InterruptResponse` 取不到响应（没有 Resume 驱动的执行） |

**天然绿 1 行**（按契约预期）：

- D14（非中断节点零影响反向对照）：普通图照常收敛、`Resume` 无处可恢复——API 面只增符号不接线，
  消费者路径零改动，实现前后都绿。

第一段字节上绑定命令 exit 1 且全部失败落在行为断言上；全包其余测试照常 `ok`（既有 13 份测试文件
零改动，契约 baselineBytes 已按切片 26 收口字节复核）。

### 2.1 夹具迭代的一次如实登记

第一段还有一条更早的 exit 1（receipt:f7b60efb-b673-4c4d-b1d7-2248e65fd8ce）：当时 D2 夹具把响应对象
直接当响应集传入（漏掉「以 InterruptID 为键」这一层），其第一条判红（类型错 → `ErrInvalidResponse`）
实际因 missing-ID 分支成立——判红理由错位。修正夹具后，API 面字节原样重落、绑定命令重跑取本节
receipt:e428c1fc；两段之间测试文件零改动的约束以 e428c1fc 为准成立（与 GREEN 之间一个字节未动）。

## 3. 承接

第二段（中断捕获 → 挂起构造 → Run 返回 suspension；`Resume` 的恰好覆盖/schema/双档/幂等/预算延续；
`EntryInterrupt` 检查点写入与 sink 错误并入）与 GREEN 见 `docs/design/v3-p5-graph-g7-green.md`；
实现差异、变异矩阵与契约对账见 `docs/design/v3-p5-graph-g7-refactor.md`。
两段之间测试文件零改动（`p5g7_hitl_test.go` 在 e428c1fc 与 ad6c744c 之间一个字节未动）。
