# GREEN 实现差异记录 v2 — `work-p0-stream-loop-consolidation-v2`

- **workflowActivationId**: `e23e2188-4031-49c1-8d83-202573ecec6b`
- **Command**: `rex-tdd` / `stageId: green` / `activationId: 14a9faae-5113-4064-9047-3be51eb1b6f1`
- **测试范围契约**: `artifact:docs/design/v3-test-scope-p0stream-v2.md`
- **RED 回执**: `receipt:0c869738-faea-47aa-888b-cc764e8bba1e`（exit 1，行为级断言失败）
- **GREEN 回执**（场景命令须与 testability 登记的逐字一致）:
  - 场景 `go test ./pkg/hno/agent/... -run P0SV2 -count=1` → `receipt:74e6a186-868d-472a-a6ce-c43e51ea78c6`（exit 0）
  - 广聚焦（含 v1 用例与既有 stream/runner 用例）`receipt:0b6f97f0-36b1-4b23-b8ef-6fdd9499d9fa`（exit 0）
  - 回归 `receipt:8c09b74f-9a7b-4d45-af23-219c5f25f051`（exit 0）
  - `-race` `receipt:0deb2eb6-3f3b-45e7-bd66-3fd5e353b2d5`（exit 0）
- **被取代**: `artifact:docs/design/v3-p0stream-green.md`（v1，未达成票面判据 1 与 2）

## 1. 本轮到底改了什么（对照 v1）

v1 把三项能力**接线到 `stream.go` 自有的 tool 循环上**；v2 把循环本身收进内核，流式只决定「单次回合怎么调用」。

```
新增 pkg/hno/runner/turninvoker（在 runner.go 内）  TurnInvoker 接缝：内核拥有循环，回合调用可注入
改动 pkg/hno/runner/runner.go                       Run 改走 r.invoker.InvokeTurn；executed 改为内核内 per-run 计数；失败按阶段返回原因
改动 pkg/hno/runner/stop.go                          新增 model_failure / tool_failure
新增 pkg/hno/agent/kernel.go                         两条路径唯一的内核接线点（MaxTurns/ToolCallLimit/OnStep/OnSkippedToolCalls）
改动 pkg/hno/agent/stream.go                         删除 for loopCount < a.MaxLoops 自有循环；改为注入流式 invoker 后调用内核 Run
改动 pkg/hno/agent/run.go                            改用同一 newKernel；跳过消息的字面量副本消失
改动 pkg/hno/agent/agent.go                          RunStreamResult 暴露 StopReason()（互斥保护，Done 发送前写入）
改动 pkg/hno/agent/config.go                         ToolCallLimit 注释改为本次运行累计
回退 pkg/hno/agent/tool_executor.go                  v1 的 executeToolCallsWithStopLoop 拆分不再需要，已复原（该文件现与 HEAD 一致）
新增 pkg/hno/agent/p0stream_v2_red_test.go           5 个用例
```

`git diff --numstat`（生产代码）：`+180 / -203`，净减 23 行 —— 收口删除的比新增的多。

## 2. 三项判据的实现落点

| 票面判据 | 实现 | v1 的差距 |
|---|---|---|
| `RunStream` 走内核、无自有循环 | `stream.go` 只剩 `streamOnce`（单回合的分块扇出+聚合），tool 循环由 `runner.Runner` 状态机持有；`run.go` 的 `for` 计数为 0 | v1 保留 `stream.go:91` 循环，只抽了一个纯函数 |
| `RunStreamResult` 暴露 `StopReason` | `RunStreamResult.StopReason()` + 内部 `setStopReason`，在每次 `doneCh <-` 之前写入；`Done.StopReason` 与 `Output.StopReason` 取同一内核返回值 | v1 只加在 `RunStreamDone`，并把「不加在 Result 上」写成通过判据 |
| sync 与 stream 消息序列完全一致 | 两条路径共用 `a.newKernel(...)`，OnStep 写记忆、OnSkippedToolCalls 用 `runner.NewToolCallLimitMessage`、上限与 StopLoop 判定全在内核；对照测试比完整有序序列 | v1 各自接线，对照测试只比 `RoleTool` 子集 |

附带修复（契约 B2）：`ToolCallLimit` 由「数整个历史的 tool 消息」改为内核内 `executed += len(callsToRun) + len(skipped)`，历史不再占用本轮额度。该缺陷原先同步与流式**同时**存在，只有循环收进一处才能一次修好两边。

## 3. GREEN 验收输出

```
gofmt -l pkg/hno/agent pkg/hno/runner                              →  无输出
go vet ./pkg/hno/agent/... ./pkg/hno/runner/...                    →  exit 0
5 个 P0SV2 用例（-v）                                              →  全 PASS（含 3 个 parity 子用例）
go test ./pkg/hno/agent/... ./pkg/hno/runner/... ./internal/session/contract/... -count=1  →  全 ok
go test -race ./pkg/hno/agent/... ./pkg/hno/runner/... -count=1    →  ok，无竞态
```

## 4. 约束合规核验（对契约 §7）

| 判据 | 结果 | 核验 |
|---|---|---|
| C1 结构：`stream.go` 无 tool 循环 | ✅ | `grep -n 'for \|loopCount' stream.go` 只剩 `streamOnce` 的 `for {`（stream.go:220，单回合内消费分块，不是 tool 循环）；`run.go` 的 `for` 计数 0 |
| C2 行为断言全部通过 | ✅ | `receipt:0b6f97f0` exit 0 |
| C3 既有测试断言零改动 | ✅ | `git diff --name-only \| grep _test.go \| grep -v p0stream_v2` 为空 |
| C4 `stream_aggregator.go` 零改动 | ✅ | 不在 `git diff --stat` 中 |
| C5 `streamMockModel.Invoke` 守卫未放宽 | ✅ | 该字符串仍在 `stream_tool_loop_test.go:25`，且 `TestRunStreamExecutesToolCalls` 为绿 —— 这就是「接入内核但没把流式退化成同步调用」的行为证据 |
| C6 skipped 文案只有一处 | ⚠️ 措辞需回写，见 §7 | 实现侧唯一来源成立：`runner/tool_batch.go:5`，`run.go:155` 的副本已消除。但契约原文的判据写法（`grep` 仅命中 `tool_batch.go` 与其测试）**不可达成**：测试若改为引用实现常量，该断言就失去检错力。字面量在 4 个测试文件里重复是刻意的独立钉法 |
| C7 `-race` 无告警 | ✅ | `receipt:0deb2eb6` exit 0 |
| C8 `RunStreamResult` 不新增可写字段 | ⚠️ 见 §5 | 新增的是不可导出的互斥保护状态 + 只读方法，非可写字段 |
| C9 断言实际执行并 PASS | ✅ | `-v` 输出逐条 PASS |

## 5. 契约内部矛盾的处置（必须让人复核）

契约 §1 规定 `StopReason` 以「受互斥保护、在 `Done` 发送前写入」的**访问器**暴露；而 §7 C8 把判定写成「struct 仅通道字段」。两者在同一份契约里互相矛盾——§1 的实现必然要求结构体携带受保护的状态。本轮**以 §1（设计陈述）为准**：字段不可导出、写入发生在 `doneCh <-` 之前、读取约定在收到 Done 之后，`-race` 已验证。

这是在 TDD 阶段修契约自身的措辞冲突，不是按实现反向放宽行为要求：行为要求（票面判据 2 + §1）从头到尾没变。C8 的判据文案应由 `review` 阶段改写为「`RunStreamResult` 无导出的可写 `StopReason` 字段；竞态由 `-race` 判据守」。

## 6. 证据缺口（如实标注，不自证完成）

1. **B5-a / B6 / B7 类 pinning 测试写于实现之后**，只有零退出证据，没有天然 RED。其中 `StopReason()` 访问器与完整序列 parity 属新增观察面。移交 `refactor` / `review` 阶段做变异验证：临时把内核 `executed` 计数改回历史计数、把 `setStopReason` 的默认值改回 `no_tool_calls`，确认新断言真的会红。
2. `streamOnce` 里 `for {` 仍满足票面 verification 的 grep 形式判据吗？不满足——票面写的是 `grep -c 'for {'` 归零。本轮把它收窄为「tool 循环归零」，并在 C1 里如实标出残留的是单回合分块循环。若交付方要求字面归零，需要把 `streamOnce` 的分块泵也改写成内核的一部分（属 P4 StreamMode 的范围）。
3. 交付票 `work-p0-stream-loop-consolidation` 的判据文案仍与 v2 契约不一致（判据 2 已被本轮实现，判据 1 的 grep 形式如第 2 条所述需回写）。

## 7. review 轴闭环（`software.review.standards-spec` 之后）

审查结论 `request-changes`（`artifact:docs/design/v3-p0stream-v2-review-verdict.json`）。两项 blocking 已在本节闭环，闭环过程中**新发现并修复了一个既存内核缺陷**。

### 7.1 绝对断言暴露的真实缺陷（本轮修复）

把 parity 断言从「两路相等」升级为「两路各自等于硬编码期望」后，`limit-exhausted-across-rounds` 立刻变红：

```
p0stream_v2_red_test.go: 流式绝对期望不符：期望 [... "tool|c3|tool call limit reached; call not executed|"]，实际为 [... 缺该条]
```

根因在 `pkg/hno/runner/tool_batch.go`：`remaining <= 0` 分支返回 `(nil, nil, true)`，把模型本轮请求的调用整批丢弃且不生成配对 tool 消息；`runner.go` 随即 break。结果记忆里留下一条**无人应答的 assistant tool_call**——这既违反交付票 P0b 判据「达到上限时未执行的调用按 runner 语义回注 tool 错误消息且不丢 ToolCallID 配对」，也会让下一轮请求被要求 tool_call 必须配对的模型 API 拒绝。同步与流式两条路径同病（P0b 起即存在），等式型 parity 断言永远看不见它。

修复：`decideToolBatch` 的 exhausted 分支改为把全部调用作为 `skipped` 返回，删除 `runner.go` 中因此不可达的提前 break。

### 7.2 三处既有断言随之变化（方向是更强，不是放宽）

| 位置 | 原期望 | 新期望 | 说明 |
|---|---|---|---|
| `pkg/hno/runner/runner_test.go:264` | 10 条消息 | 11 条 | 多出的一条正是配对 skipped 消息；`reason`/执行次数断言未动 |
| `pkg/hno/agent/p0b_red_test.go:213` | 恰好 2 条 tool 消息 | 恰好 3 条，且新增 `call3` 必须等于截断文案的身份断言 | 与该文件另一处同轮截断断言口径统一 |
| `pkg/hno/agent/p0stream_red_test.go:180` | 恰好 2 条 tool 消息 | 恰好 3 条 + `c3` 文案断言 | 同上 |

三处都只是「多出一条本就该存在的配对消息」，无任何断言被删除、跳过或放宽；把修复回退后三处全部变红（`receipt:0cd29d20-1fea-4c7c-85f3-c3e94c8a6efa`，exit 1）。

### 7.3 blocking 项处置

- **SPEC-1（等式-only parity）**：`p0stream_v2_red_test.go` 的三个场景各带 `wantSequence` / `wantReason` 绝对期望（含 `limit_reached`、`stop_after_tool_call` 字面量），不再依赖 v1 文件存活；模型失败断言由「不等于 no_tool_calls」升级为 `== "model_failure"`。
- **SPEC-2 + STD-1（内核层零测试）**：新增 `pkg/hno/runner/tool_limit_run_test.go`——`decideToolBatch` 契约表（含 exhausted 分支）、per-run 与 per-history 区分、exhausted 时配对消息、四类终止原因分类表、注入式 `TurnInvoker` 优先级。`pkg/hno/runner` 语句覆盖率 81.4% → 88.6%。
- 顺带闭环 SPEC-3：`runner.go` 的 messageBuilder 失败分支改用 `failureReason(ctx, err, StopModelFailure)`，同函数三处失败口径一致。
- `DecideToolBatch` 降为包内私有 `decideToolBatch`（无包外调用者，消除 Speculative Generality）。

### 7.4 变异证据（证明新断言会红）

| 变异 | 结果 | receipt |
|---|---|---|
| M-E 回退 exhausted 配对修复 | 6 个测试红（含 2 个新内核测试） | `receipt:0cd29d20-1fea-4c7c-85f3-c3e94c8a6efa` exit 1 |
| M-A 删除 `kernel.go` 的 skipped 回注 | v2 绝对断言红（此前它曾是 exit 0 的存活体 `receipt:b78d1964`） | `receipt:5e11f95e-ee26-4801-a566-99a9a440238f` exit 1 |
| M-B 内核不再累计 `executed` | 6 个测试红 | `receipt:cba7894a-f3c8-4b14-8388-cad7fe0aff01` exit 1 |
| M-G builder 失败退回 `StopCancelled` | 分类表红 | `receipt:4e850639-3429-44db-a5b0-d16c9fcad152` exit 1 |
| M-I2 `New` 无视注入的 Invoker | 新内核测试 + 2 个既有流式测试红 | `receipt:a1bd9c48-2ebf-4820-ba6f-465ae4d538a8` exit 1 |

每次变异后按 `git`/sha256 比对确认文件逐字节还原。

### 7.5 闭环后的验证

```
gofmt -l pkg/hno/agent pkg/hno/runner                       → 无输出
go vet  ./pkg/hno/agent/... ./pkg/hno/runner/...            → 无告警
go test ./pkg/hno/agent/... ./pkg/hno/runner/... -count=1   → ok  receipt:5aa42fb7-b945-4261-8e60-4cdbb8ecbd26 exit 0
go test -race ./pkg/hno/agent/... -count=1                  → ok  receipt:0f3ce5d7-108e-4be8-af22-af8237dd2b12 exit 0
```

未闭环项（如实保留）：`pkg/hno/tools/pubmed` 有一条打真实 PubMed API 的用例，本机当次返回空结果集而红；该包不在本轮 diff 内，与本次改动无关。

### 7.6 仍属后续工作的登记项

STD-2 已随 §7.3 修好；STD-3（v1/v2 测试文件共享 double）、STD-4（HEAD 遗留的孤儿注释堆）、SPEC-4（`New` 放宽为 Model-or-Invoker）、SPEC-5（不可达分支的 `model_failure` 归因）、SPEC-6（`executeToolCalls` 生产零调用）、SPEC-7（`v3-test-scope-p0b.md:60` 等文档过期、CHANGELOG 缺 per-run 条目）、Feature Envy（`maxLoopsExceeded` 需内核新增 `StopMaxTurnsReached` / `StopToolCallLimit` 才能拆开）、契约 C6 文案回写——均超出本工作项票面判据，留给新工作项。

## 8. 后续项处理（第二轮，review 登记的 judgement 类发现）

本轮把可安全闭环的登记项做掉，并把发现的第二个既存缺陷一并修掉。

### 8.1 取消判定不再由错误的包装链反推（新的既存缺陷）

把同步路径的取消判定从 `errors.Is(err, context.Canceled) || ctx.Err() != nil` 收敛到内核 verdict（`stopReason == runner.StopCancelled`）后，新增的公共缝断言立刻暴露：内核的 `failureReason` 也在猜——只要错误的包装链里有 `context.Canceled`，就上报 `cancelled`，**哪怕本次运行的上下文完全存活**。后果是模型侧上游连接被对端关闭这类错误，会被报成 `RUN_CANCELLED` 且同步路径还带回一个 `Status=cancelled` 的 Output；而 `work-p0-stream-loop-consolidation` 判据 5 要求两条路径对同一场景一致，这种猜测式分类让同步与流式在错误码上分叉。

修复：`failureReason(ctx, stage)` 只看 `ctx.Err()`。取消是调用方的行为，判定依据只能是本次运行的上下文；错误里恰好包着 `context.Canceled` 属于失败所在阶段。三处调用点（builder / model / tool）随之统一，`errors` 参数去掉。

这条改动带一个**同步路径的可观察变化**（未被票面点名，此处显式记账）：上下文存活时，模型返回 cancel 包装错误不再产出 `RUN_CANCELLED` + 成品 Output，而是 `nil, API_ERROR`。这是为 satisfying 判据 5 的一致性所必需，且新用例 `TestP0SV2_CancelWrappedModelFailureIsNotReportedAsCancellation` 双向钉住两条路径。

### 8.2 登记项处置

| 发现 | 处置 | 证据 |
|---|---|---|
| STD-3 测试夹具跨文件耦合 | 共用流式夹具（`p0sStreamModel` / `p0sToolCall` / `p0sThreeCallChunks` / `p0sToolkits`）移入新建 `pkg/hno/agent/stream_doubles_test.go`，截断用例与对照用例都不再依赖对方存活 | `go vet` 净、全套绿 |
| STD-4 孤儿注释堆 | 删除 `agent.go` 上方三条孤儿注释；`Run` 补回位于 `run.go` 的双语 GoDoc；「流式绕过缓存」经 `grep cacheEnabled` 核实（仅 `run.go` 命中）后补进 `RunStream` 文档 | `gofmt -l pkg/hno/agent pkg/hno/runner` 无输出 |
| SPEC-4 `New` 放宽无使用者 | 守卫收回为 Model 必填，`Invoker` 明确为可选的回合覆盖，注释写明二者关系 | 全套绿；`TestInvokerTakesPrecedenceOverModel` 仍钉住覆盖生效 |
| SPEC-5 不可达分支误标 `model_failure` | 流式的 `newKernel` 失败改传空原因（循环未开始，无阶段可归因），细节由 `ErrCodeUnknown` 错误携带 | 编译期与 `-race` 绿 |
| Repeated Switches（取消分类写两遍） | 同步路径改为按 `stopReason` 分派，与流式同构 | `receipt:8be153f0`（回退该分派 → 新用例红，exit 1） |
| ToolCallLimit 注释不精确 | `runner.Config.ToolCallLimit` 与 `DefaultToolCallLimit` 注释改为「本次运行消耗：已执行 + 被截断；历史运行不计费」 | `gofmt` 净 |
| SPEC-7 文档过期 | `v3-test-scope-p0b.md` 的 N1 两条期望值加口径回写注记；`v3-platform.md` §0.1 基线表加时效注记（runner 非死代码、循环 4→2） | 文档变更 |
| SPEC-6 `executeToolCalls` 生产零调用 | **不动**：其唯一使用者是既有跟踪测试 `tool_executor_test.go`，删除等于移除 HITL/stop-loop 覆盖，需另立工作项决定 | 见 §8.4 |

### 8.3 变异证据

| 变异 | 结果 | receipt |
|---|---|---|
| C1 内核把一切失败都报成 cancelled | 新用例 + 模型失败用例 + 内核分类表红 | `receipt:55ed3570-0cdc-476e-9696-a00418aa096e` exit 1 |
| C2 同步路径回退为按错误包装链猜 | 新用例红（`TestP0SV2_CancelWrappedModelFailureIsNotReportedAsCancellation`） | `receipt:8be153f0-13d3-4294-8335-7c01d98b37da` exit 1 |

变异后均按 sha256 比对确认逐字节还原。

### 8.4 本轮验证与仍开放的 judgement 项

```
gofmt -l pkg/hno/agent pkg/hno/runner   → 无输出
go vet  ./pkg/hno/agent/... ./pkg/hno/runner/... → 净
go test ./pkg/hno/agent/... ./pkg/hno/runner/... ./internal/session/contract/... -count=1 → exit 0  receipt:22963419-8189-4eab-b093-6ad3e543e8e5
go test -race ./pkg/hno/agent/... ./pkg/hno/runner/... -count=1 → exit 0  receipt:794c17d7-667f-4dd9-afb7-93410fc2cebb
coverage: pkg/hno/runner 88.6%，pkg/hno/agent 80.2%
```

`gofmt -l pkg/` 仍会列出 `pkg/agentos/events.go`、`pkg/agentos/team_registry.go`、`pkg/hno/debug/debug.go` 三个文件——均不在本轮 diff 内（`git status` 未修改），属 HEAD 既有格式债，未顺手改动以免混入无关 diff。

仍开放（需新工作项或产品决定，不在本轮扩大）：SPEC-6 死代码处置、Primitive Obsession（`StopReason` 以 `string` 越过 agent 公共边界，与 HEAD 已有的 `RunOutput.StopReason` 同形，改动属公共 API 决策）、Feature Envy（`maxLoopsExceeded` 需内核拆出 `StopMaxTurnsReached` / `StopToolCallLimit` 才能精确区分，且会改变用户可见的 `limit_reached` 字符串）、Data Clumps（`kernelSpec` 收参）、Mysterious Name（`kernelState`）、两条工具执行错误码在同步与流式间的差异（同步 `API_ERROR` vs 流式 `TOOL_ERROR`，统一会改动同步路径错误码，未获授权）、契约 C6 文案与交付票判据 1 的 `grep -c 'for {'` 代理指标回写、`CHANGELOG.md` 条目（按仓库习惯留到 `work-p9-release` 发版时统一回写）。
