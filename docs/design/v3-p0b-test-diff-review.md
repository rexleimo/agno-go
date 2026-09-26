# 测试差异审查 — `work-p0b-runner-capability-adoption` / stage `refactor`

- **审查对象**：`pkg/hno/agent/p0b_red_test.go`（本周期唯一新增测试文件，265 行）
- **审查结论**：**部分守住** —— 2 处断言缺口，均已用变异测试**实证**证明
- **审查执行方式**：主 Agent 亲自做对抗性审查。子 Agent 审查派发连续三次因基础设施失败（claude 503 / codex 沙箱 go-build 目录不可写），未采用其结果。

## 1. 既有测试改动核验

```
$ git diff --name-only | grep _test.go
[空]
```

既有测试**零改动**。本周期测试差异 = 1 个新增文件。这条通过。

## 2. 期望值权威出处核验

| 期望值 | 权威出处 | 结论 |
|---|---|---|
| `"no_tool_calls"` | `pkg/hno/runner/stop.go:10` `StopNoToolCalls` | ✅ 取自规格常量，非抄实现 |
| `"limit_reached"` | `pkg/hno/runner/stop.go:13` `StopLimitReached` | ✅ 同上 |
| `"stop_after_tool_call"` | `pkg/hno/runner/stop.go:16` `StopAfterToolCall` | ✅ 同上 |
| `"tool call limit reached; call not executed"` | `pkg/hno/runner/runner.go:284` | ⚠️ **仅存在于实现，不存在于 `runner_test.go`** |

### 2.1 耦合风险（中等）

跳过消息文案**只出现在 `runner.go`，没有任何 runner 测试钉住它**。本测试把它抄了过来作为期望值。

后果：将来若 `runner` 修改该文案（纯措辞调整），本测试会**无故失败**，而 `Agent` 的实际行为并无问题。这是**跨包的字面量耦合**。

不是错误（`runner` 是本工作项指定采纳的内核），但应记录：**该期望值是转录而非独立固定**。

## 3. 发现的缺口（按严重度，均有实证）

### 【严重】缺口 1：StopLoop 场景不验证工具结果被保留

`p0b_red_test.go:207-227` 只断言 `StopReason` 与 `model.index`，**从不检查 `stopTool` 的输出 `"stopped"` 是否写入消息**。

**变异测试实证**：在 `agentToolExecutor.Execute` 注入「`stopLoop` 时不写 `Memory`」的缺陷后 —

```
$ go test ./pkg/hno/agent/... -run 'TestP0B_StopLoop' -count=1
ok  	github.com/rexleimo/agno-go/pkg/hno/agent	1.069s
```

**仍然通过。** 即：一个「终止循环但丢弃工具输出」的实现可以完全通过本测试。

**用户影响**：调用声明 `StopLoop` 的工具后，工具返回值静默丢失。

**注意**：这偏离了测试范围契约 §6 N2 明确要求的「`Messages` 中有该工具的结果消息」—— RED 子 Agent **漏实现了契约中已写明的断言**。

**建议补充断言**：
```go
found := false
for _, m := range output.Messages {
    if m.Role == types.RoleTool && m.ToolCallID == "call1" && m.Content == "stopped" {
        found = true
    }
}
if !found {
    t.Error("stopTool 的结果消息应保留在 output.Messages 中")
}
```

### 【严重】缺口 2：SingleRound 不验证已执行调用真的执行了

`p0b_red_test.go:112-138` 断言 `toolMessages == 3`，并检查「存在一条 skip 文案且其 ID 为 `call3`」。

但它**没有断言 `call1`→`"result1"`、`call2`→`"result2"` 存在**。且 `skippedMessage` 变量在循环中被反复覆盖，只校验**最后一条** skip 的 ID。

**变异测试实证**：把 3 条消息的 Content 全部改成 skip 文案（即**没有任何工具真正执行并记录结果**）后 —

```
$ go test ./pkg/hno/agent/... -run 'TestP0B_ToolCallLimitTruncatesBatch_SingleRound' -count=1
ok  	github.com/rexleimo/agno-go/pkg/hno/agent	0.487s
```

**仍然通过。**

**用户影响**：`ToolCallLimit` 的核心价值是「执行前 N 个、其余如实报告为跳过」。本测试无法区分「正确执行了 2 个」与「一个都没执行」。

**建议补充断言**：
```go
got := map[string]string{}
for _, m := range output.Messages {
    if m.Role == types.RoleTool { got[m.ToolCallID] = m.Content }
}
for id, want := range map[string]string{"call1": "result1", "call2": "result2", "call3": "tool call limit reached; call not executed"} {
    if got[id] != want {
        t.Errorf("call %s: 期望 %q，实际 %q", id, want, got[id])
    }
}
```

### 【中等】缺口 3：MultiRound 不校验消息身份

`p0b_red_test.go:186-196` 只数 `RoleTool` 数量为 2 并断言 `model.index == 3`。

`model.index == 3` 是**强断言**（证明跨轮累计语义确实生效），但没校验这 2 条分别是 `call1`/`call2`。数量对、身份错的情况抓不到。

### 【低】缺口 4：若干交互场景未覆盖

- `StopLoop` 与 `ToolCallLimit` 同时触发时的优先级（`runner.go:294-301` 的 switch 顺序）无测试。
- 多个工具中**仅一个**声明 `StopLoop` 时，其余工具是否仍执行 —— 无测试。实际语义是「全部执行完再停」，这个行为值得钉住。
- 工具未找到 / 参数非法 与 `ToolCallLimit` 的交互无测试。

## 4. 无「为了通过而弱化」的痕迹

| 检查项 | 结果 |
|---|---|
| `t.Skip` | 无 |
| 被注释掉的断言 | 无 |
| 依赖执行顺序 | 无（每个测试自建 fake，无共享状态，无 `t.Parallel`） |
| 恒真断言 | 无 |
| fake 脚本是否足以支撑断言 | 是。`fakeModel` 的 fallback 分支不递增 `index`，但 `StopLoop` 测试的 `responses` 有 2 项，第 2 次调用会递增到 2，故 `index != 1` 确实能抓住「多调一次模型」 |

## 5. 结论与处置

**判定：部分守住用户行为约束。**

测试成功抓住了本周期最难的三个真实缺陷（截断消息配对、assistant 消息写入、MaxLoops 契约）—— 这部分价值是实在的。

但它**没有守住「工具结果不丢失」这一核心承诺**，缺口 1、2 已用变异测试实证。

**处置**：
- 本阶段不修改测试（`refactor` 阶段的目的是保持测试通过 + 审查差异，不是补测试）。
- 缺口 1、2、3 记入待办，应在后续工作项中以**新增测试**方式补齐，不能削弱现有断言。
- 建议并入 `work-p0` 收口项，因为那是一次测试范围的重新界定。
- 缺口 4 属 `work-p5`（HITL 与 stopLoop 优先级）或 `work-p0b` 补充。
