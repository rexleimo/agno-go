# 切片 33 RED 观察（P6/G10 OPT-β：workflow 控制流全换到图内核，公共 API 不变）

工作项 `work-p6-g10-opt-beta-migration` / 契约 `docs/design/v3-test-scope-p6-g10-migration.json`（D1–D14）。
测试文件：`pkg/hno/workflow/p6g10_migration_test.go`（14 个 TestP6G10_*，外部包）。

## 1. redProtocol 与取消事件披露

实现原派发子代理，在并行扇出收尾阶段被系统取消。取消时已落盘：契约、`graph_compiler.go`、
`executor.go` 接线、14 个等价测试、RED 回执（`receipt:b5f7aad2-…`，exit 1——编译路径尚未接管时
行为差异的真实红）；**未完成**：并行分支头克隆（[P6] 硬阻塞的解）与全部收尾。

主代理接手：两条失败（`ParallelParity` output=zero-out want one-out；
`ParallelConcurrentWriteNoRace` branch writes leaked）诊断为分支头节点漏了 cloneBranchEC——
注释声明「分支头各自克隆私有 EC」但代码是原指针透传，所有分支共享主干 EC（正是 OPT-β 硬阻塞
「并发暴露」的形状）。补 1 行克隆（`&carrier{ec: cloneBranchEC(...)}`）后全绿。

## 2. 绑定命令与实测

```bash
go test ./pkg/hno/workflow/... -run TestP6G10_ -count=1 -race -v
```

| 阶段 | exit | receipt |
|---|---|---|
| 子代理 RED（编译路径未接管） | 1 | `receipt:b5f7aad2-…` |
| 接手时中间态（2 挂：D7 并行两向） | — | 失败文案逐字引用于 §1（未单独收执，字节可溯） |
| 修复后全绿（最终字节，14/14 PASS） | 0 | `receipt:79b20f4e-178a-477e-810d-bb809d2cc35d` |

D8（别名隔离）的红-绿闭环：红=中间态「branch writes leaked into the shared context」；
绿=修复后 ConcurrentWriteNoRace 通过；变异 m7（去掉克隆）精确复现红 —— 牙齿闭环。
