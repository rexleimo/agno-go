# 切片 22 RED 观察（母约 §4 G3：节点策略四合一）

工作项 `work-p2-graph-g3-node-policies` / 契约 `docs/design/v3-test-scope-p1-graph-slice22.json`（D1–D14）。
测试文件：`pkg/hno/graph/p2g3_policies_test.go`（`package graph_test`，只用导出 API）。

## 1. redProtocol 两段制的第一段

本片新增的是公共面符号（不是既有符号上的新行为），按契约 redProtocol：第一段只落 API 面
（`policy.go` 的四个 Config + `NodeOption` + `WithRetry/WithTimeout/WithCache/WithTrace`，
`graph.go` 的变参 `AddNode` 与 policies 表，`scheduler.go` 的 plan 携带）——语义不接线。
绑定命令可编译，行为行按设计期 M1 实测的形状真实判红。

本片绑定场景命令逐字（所有 receipt 都是这一条命令、直接取，无 `sh -c` 包装）：

```bash
go test ./pkg/hno/graph -run TestP2G3_ -count=1 -race -v
```

## 2. RED 实测（receipt:d8269809-2eb9-406b-8662-cc18c72f112b，exit 1）

**判红 10 行**（全部落在「策略缺位」的公共面形状上，零编译失败、零环境失败）：

| 行 | 失败文案（摘） |
|---|---|
| D1 | 节点执行了 1 次，want 恰好 3 次（今天失败即停 calls=1，重试未生效） |
| D3 | 节点执行了 1 次，want 恰好 2 次（上限钳制失效或 off-by-one） |
| D5 | 节点执行了 1 次，want 3（重试没有生效） |
| D6 | Run 交出 no deadline reached the node，want 可归因到 context.DeadlineExceeded |
| D8 | 节点执行了 1 次，want 恰好 2（PerAttempt 期限耗尽一次重试一次） |
| D9 | 节点执行了 2 次，want 1（同键第二次 Run 应命中缓存；今天每次都真跑） |
| D10 | 同键 a 的第三次 Run 又真执行了（calls=3），want 命中缓存保持 2 |
| D11 | 成功后同键 Run 又真执行了（calls=4），want 命中缓存 |
| D12 | 收到 0 条事件，want 恰 1（今天导出面上没有任何 sink） |
| D13 | 收到 0 条事件，want 1（脱敏不得把事件本身弄丢） |

**天然绿 4 行**（反向对照，按契约预期在实现前后都成立）：D2（零值不重试）、
D4（ShouldRetry 门槛与 nil 语义——今天根本没有重试，calls 恒 1）、D7（无期限声明时 ctx 同形）、
D14（trace 关闭零事件）。

第一段字节上全包测试照常 `ok`（既有 10 份测试文件零改动，M2 设计期已预演源兼容）。

## 3. 承接

第二段（接线 `runNode` 包络与消费者侧 trace 发射）与 GREEN 见
`docs/design/v3-p2-graph-g3-green.md`；变异矩阵与契约对账见 `v3-p2-graph-g3-refactor.md`。
