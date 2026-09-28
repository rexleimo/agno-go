# 切片 24 RED 观察（母约 §7 G6：Durability 持久化档位）

工作项 `work-p3-graph-g6-durability` / 契约 `docs/design/v3-test-scope-p1-graph-slice24.json`（D1–D9）。
测试文件：`pkg/hno/graph/p3g6_durability_test.go`（`package graph_test`，只用导出 API；
夹具自备内存 Checkpointer，互斥锁属夹具不属引擎）。

## 1. redProtocol 两段制的第一段

本片新增的是公共面符号（不是既有符号上的新行为），按契约 redProtocol：第一段只落 API 面
（`durability.go` 的 `Durability` 三常量 + `Checkpoint` + `Checkpointer` + `WithDurability/WithCheckpointer`，
`graph.go` 的 config 加 `durability/checkpointer` 两字段）——`beginCommits/record/finishCommits`
三段接线不属于第一段，调度器不消费。绑定命令可编译，行为行按契约预判的形状真实判红：
sink 收不到任何条目、sink 错误无从传播、撞阀后条目面为空——全部是行为红，零编译失败。

本片绑定场景命令逐字（所有 receipt 都是这一条命令、直接取，无 `sh -c` 包装）：

```bash
go test ./pkg/hno/graph -run TestP3G6_ -count=1 -race -v
```

## 2. RED 实测（receipt:3d22b67c-1e21-44f4-80fe-c7dbf474ad6f，exit 1）

**判红 7 行**（D2–D7、D9，全部落在「提交流缺位」的公共面形状上）：

| 行 | 失败文案（摘） |
|---|---|
| D2 | 后继节点 b 的节点体采样 = 0, want ≥1（默认档必须像 Sync 一样在下一步开始前落盘） |
| D3 | b 体采样 = 0, want ≥1（Sync 的提交点在 complete 之后、下一轮 dispatch 之前） |
| D4 | sink 收到 0 条条目, want 恰 3 条（运行中零条目的半边天然成立，退出落齐的半边缺失） |
| D5 | sink 收到 0 条条目, want 恰 3 条（返回前落齐不成立） |
| D6 | 条目数 = 0, want len(Completed()) = 3（sync/async/exit 三个子用例同形判红） |
| D7 | sync 档 sink 失败后 Run 的错误 = \<nil\>, want errors.Is 到夹具错误；exit 档同形（sink 根本没被调用，错误无从传播） |
| D9 | 撞阀后 sink 条目 = [], want 恰 [1:a]（D9 的 errors.Is(ErrStepLimitExceeded) 半边通过——安全阀是切片 13 已交付行为；判红只来自提交流半边） |

**天然绿 2 行**（按契约预期）：

- D1（常量序 + 零值即 Sync + 四符号存在）：API 面落地即转绿——契约注明的「本片唯一允许的
  编译红」在第一段落地后不复存在；
- D8（三档无 sink 照常跑完）：反向锚，实现前后都绿。

第一段字节上绑定命令 exit 1 且全部失败落在行为断言上；全包其余测试照常 `ok`
（既有 11 份测试文件零改动，契约 M2 已预演源兼容）。

## 3. 承接

第二段（`beginCommits/record/finishCommits` 三段接线：scheduler 快照两字段、consume 成功路径
complete 之后调 record、Run 尾部 beginCommits/finishCommits）与 GREEN 见
`docs/design/v3-p3-graph-g6-green.md`；变异矩阵与契约对账见 `v3-p3-graph-g6-refactor.md`。
两段之间测试文件零改动（`p3g6_durability_test.go` 在两段间一个字节未动）。
