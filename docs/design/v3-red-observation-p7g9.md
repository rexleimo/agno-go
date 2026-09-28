# 切片 25 RED 观察（母约 G9：观测接线）

工作项 `work-p7-g9-observability` / 契约 `docs/design/v3-test-scope-p7-g9-observability.json`（D1–D13）。
测试文件：`pkg/hno/runner/p7g9_resilience_test.go`（`package runner`，与既有两份测试同包，
夹具自备 scriptedModel 计数器与 otel sdk tracetest exporter）。

## 1. redProtocol 两段制的第一段

按契约 redProtocol：第一段只落 API 面——`runner.go` 的 Config 两个可选字段
（`Retry *observability.RetryConfig`、`Breaker *observability.CircuitBreaker`）+ Runner 四个私有字段
（retry/breaker/provider/modelName）+ `New` 接线，以及新文件 `resilience.go` 的私有缝声明
（`invokeTurn` + `clampAttempts`）——**调用点不动**（`StateAwaitModel` 仍直调
`r.invoker.InvokeTurn`）。绑定命令可编译（build/vet 干净），行为行按设计期 M1 实测的
「重试/熔断/span 缺位」形状真实判红。

本片绑定场景命令逐字（所有 receipt 都是这一条命令、直接取，无 `sh -c` 包装）：

```bash
go test ./pkg/hno/runner -run 'TestP7G9_' -count=1 -race -v
```

## 2. RED 实测（receipt:aa8c6103-3d91-4a2f-ba74-1e9050c90275，exit 1）

**判红 9 行**（全部落在「韧性/span 缺位」的公共面形状上，零编译失败、零环境失败；
文案为 `-v` 逐字摘录）：

| 行 | Test | 失败文案（摘） |
|---|---|---|
| D1 | RetryRecovers | `run err: runner: model invoke: transient`（今天失败即停，重试未生效） |
| D3 | RetryExhaustedSurfacesLastError | `calls = 1, want 2`（耗尽语义缺位） |
| D5 | BreakerOpens | `err = runner: model invoke: boom, want errors.Is ErrOpen` + `calls = 3, want 2`（无门禁、调用数不冻结） |
| D7 | OpenBreakerNotRetried | `err = runner: model invoke: boom, want ErrOpen`（无门禁时重试照烧） |
| D8 | BreakerCountsPostRetryOutcome | `calls = 1, want 3 attempts`（重试缺位导致记账语义无从谈起） |
| D9 | ChatSpanWithUsage | `spans = 0, want 1 chat span`（今天 runner 零 span，M1 结构扫描实证） |
| D10 | SpanPerAttemptWithError | `run: runner: model invoke: transient`（span 与重试同缺） |
| D12 | CustomInvokerAlsoResilient | `err=runner: model invoke: transient reason=model_failure, want success` |
| D13 | RetryDoesNotConsumeTurnBudget | `err=runner: model invoke: transient reason=model_failure, want success`（MaxTurns=1 下无重试） |

**天然绿 4 行**（反向对照与钳制齿，按契约预期在实现前后都成立）：
D2（零值恰 1 次且错误串 `runner: model invoke: boom` 同形——禁用观测时零开销路径的锚）、
D4（MaxAttempts=0 未接线时天然恰 1 次；本行牙齿在变异 m1 上）、
D6（无门禁时恢复天然全通；与 D5 成对，D5 钉「没关门」、本行钉「关了必须能开」）、
D11（默认 noop tracer 下行为同形）。

第一段字节上两包既有测试照常 `ok`（既有 runner_test.go / tool_limit_run_test.go 与
observability 九文件零改动，M2 设计期已预演源兼容）。

## 3. 承接

第二段（`StateAwaitModel` 调用点换 `r.invokeTurn` 缝）只动实现一行，与 GREEN 见
`docs/design/v3-p7-g9-green.md`；变异矩阵与契约对账见 `v3-p7-g9-refactor.md`。
两段之间测试文件零改动（`p7g9_resilience_test.go` 两段逐字节相同）。
