# 切片 35 GREEN（RetryConfig.Jitter 接线）

工作项 `work-p2-graph-jitter-wiring` / 契约 `docs/design/v3-test-scope-p1-graph-slice35.json`（D1–D4）。
本片为小增量片，RED/GREEN/REFACTOR 合并记录（redProtocol 说明：无公共面行为红——切片 22 预登记
「分布不在公共面」，接线行以结构对账+白盒确定性测试收口，不设真实时钟断言）。

## 1. 实测

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| D1–D3（applyJitter 白盒：界限/恒等/可复现，固定种子） | `go test ./pkg/hno/graph -run TestApplyJitter -count=1 -race -v` | 0 | `receipt:604550a8-99ec-4cab-840c-5cf136435a71` |
| D4（结构面：退避包络经 applyJitter，Jitter>0 永不放大上限） | `go test ./pkg/hno/graph -run TestRetryEnvelopeAppliesJitter -count=1 -race -v` | 0 | `receipt:5d853c4c-ebb4-49a3-8357-80960c0dad92` |
| 整包回归 | `go test ./pkg/hno/graph/... -count=1` | 0 | （非 receipt 记录） |

实现：`policy.go` 新增纯函数 `applyJitter(d, jitter, rnd)`（full-jitter，[0, d) 均匀，
rnd 注入可确定性测试）；`scheduler.go` 增 `rnd` 字段（每次 Run 时间种子、消费者协程独占，
零锁口径不变）并在重试退避处调用；`hitl.go` 的 newScheduler 字面量同步初始化。
结构对账（D4）：runNode 调用点存在且传 `pol.retry.Jitter`；变异摘除该调用的行为等价性
按切片 22 预登记为观察上限（分布不在公共面），不作时钟断言（R17 教训）。

## 2. 门禁与结构

build/vet/gofmt 干净；整包 `-count=1` 绿；导出面 64（applyJitter 为包内非导出函数，
+4 来自本轮前的 restore.go 与条目种类累计——逐片可溯）。

## 3. 审查

小增量片，作者自查（编排方=实现者）。verdict：pass。证据回执 2 条全部真实可溯。
