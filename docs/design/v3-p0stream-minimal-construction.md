# 最小构造评估 — `work-p0-stream-loop-consolidation`

- **workflowActivationId**: `f166216c-061d-47ec-b149-b78c3e02d507`
- **Command**: `rex-minimal-construction` / `stageId: minimize`
- **基线回执**: `receipt:039e7fdb-7673-4731-b31e-d0838e82374d`（exit 0）
- **基线断言**: `pkg/hno/agent` + `pkg/hno/runner` 全绿

---

## 0. 决定性事实（先摆事实，再走阶梯）

**`pkg/hno/runner` 完全没有流式能力。**

```
$ grep -rn "InvokeStream\|Stream" pkg/hno/runner/*.go
pkg/hno/runner/runner_test.go:36:  func (m *mockModel) InvokeStream(...)   ← 仅测试桩
```

`runner.Run` 的状态机在 `StateAwaitModel` 里只调 `r.model.Invoke(ctx, req)`（同步）。而 `RunStream` 的本质是 `streamOnce` 调 `a.Model.InvokeStream(...)` + `AggregateResponseStream` 做增量聚合（`stream.go:229`、`stream_aggregator.go` 44 行）。

**因此交付票的字面判据「RunStream 内部改走 `pkg/hno/runner`」是一个范畴错误** —— 它把「共用循环策略」和「共用传输方式」混为一谈，而这两者在本仓库里**不能**合一：走 runner 就必然退化成 `Invoke`，那 `RunStream` 也不再是流式了。

该判据是 P0b 审查阶段（`SPEC-1` 处置）由我写入的，当时并不知道 runner 无流式支持。这是我的规格缺陷，本阶段予以修正。

## 1. 复用阶梯逐级评估

### 第 1 级：需求中的偶然复杂度可否删除？

**可以，且是本次最大的发现。**

原判据把三件事捆在一起：①共用循环策略 ②共用传输方式 ③流式获得三项能力。实际用户可观察目标只有 ③。

- ①是**真需求**：截断/stop 的计算逻辑不该有两份，否则必然漂移（P0b 已为此付出 4 轮修复）。
- ②是**伪需求**：sync 与 stream 的传输本就应当不同，流式存在的意义就是 `InvokeStream`。强行统一会直接摧毁 `RunStream`。
- ③是**真需求**。

**删除 ②，保留 ①③。**

### 第 2 级：仓库已有代码能否复用？

**能，且这正是可复用的部分。** `runner` 里已实现、值得复用的是**每轮的循环控制决策**（`runner.go:230-256` 截断、`:294-301` 终止判定）与其常量（`stop.go:10,13,16`）。

但它目前**内嵌在 `runner.Run` 的状态机里，无法被流式路径直接调用**。所以复用方式不是「调用 runner」，而是「把该决策抽成可被两侧调用的纯单元」。

### 第 3 级：语言 / 标准库能否解决？

`context` + `sync` 已在用。`streamOnce` 的 goroutine + channel 管道无标准库替代。**不适用。**

### 第 4 级：已安装依赖能否解决且不扩大耦合？

无相关依赖。**不适用。**

### 第 5 级：局部表达式能否保持可读？

三项能力在流式路径上的接线都是薄壳：一次截断判定、一个 `stopLoop` 标志透传、一个终止原因回填。**可以。**

### 第 6 级：都不成立时的最小新构造

需要在 runner 侧新增**一个纯决策函数**（不触碰其状态机），见下。

## 2. 三个候选方案

| | A. 给 runner 加流式 | **B. 抽共享循环策略** | C. 在 stream.go 就地实现 |
|---|---|---|---|
| 做法 | runner 状态机新增流式分支 + 事件/sequence 管道，`RunStream` 整段改走 `runner.Run` | 把 runner 的「本轮跑几个 / 谁被跳过 / 是否终止」抽成纯函数；sync 走 `runner.Run`，stream 保留 `InvokeStream` 传输但调用同一纯函数 | 在 `stream.go` 的循环里直接写截断/stop 判定 |
| 消除第二份循环 | ✅ 是 | ⚠️ 策略同源，传输各留一份 | ❌ 否 |
| 改动面 | runner 核心状态机 + 11 个既有 runner 测试需补流式覆盖 + 新增事件管道 | 新增 1 个小纯函数 + stream.go 少量接线；**runner 语义零变化** | stream.go 局部 |
| 风险 | **高**（改的是已被 11 个测试证明的核心状态机） | **低** | 低 |
| 满足「sync/stream 消息序列一致」 | 天然满足 | 由**对照测试**钉住 | 靠人工保证，最易漂移 |
| 新增能力是否可用 | ✅ | ✅ | ✅ |

### 裁定：**B**

理由：

1. **A 的成本被低估了。** 「给 runner 加流式」要动的是 `runner.Run` 的状态机本身，而该状态机是 P0b 刚用 11 个测试钉死的核心。P0b 期间把 `Agent.Run` 接过去就已经引发 5 个既有测试失败 + 1 个 panic；再动状态机的爆炸半径更大。
2. **B 直接命中真正的重复。** 重复的是**策略计算**，不是传输。抽出纯函数后，两侧共用同一份截断/stop 逻辑，且 runner 行为完全不变 —— 11 个测试原封不动。
3. **B 不牺牲判据 5。** 「sync 与 stream 消息序列一致」由对照测试钉住，比「结构上共用一个循环」更直接地验证了用户可观察结果。
4. **C 不可接受。** 那等于把 P0b 花四轮才消灭的第四份逻辑副本再放回来。

## 3. 最小实现清单

| # | 改动 | 性质 | 新增逻辑量 |
|---|---|---|---|
| 1 | `runner` 新增**纯函数**：`DecideToolBatch(executed, limit, calls) (toRun, skipped []types.ToolCall, limitHit bool)` | 抽取 | 小 |
| 2 | `runner.Run` 的截断分支改为调用该纯函数，**语义逐字不变** | 接线 | 极小 |
| 3 | `toolResult` 已带 `stopLoop`；`stream.go` 循环读取 `fn.StopLoop` | 透传 | 极小 |
| 4 | `RunStreamDone` 增加 `StopReason string`（**不加在 `RunStreamResult` 上** —— 它在流开始时就已返回，值那时未知） | 新增字段 | 极小 |
| 5 | `stream.go` 循环使用 #1 的纯函数做截断，回注 skipped tool 消息（走 `types.NewToolMessage`，与 sync 路径同源） | 接线 | 小 |
| 6 | **不修改** `runner` 状态机、既有 runner 测试、既有 stream 测试 | — | 0 |
| 7 | **不改** `stream_aggregator.go` 的聚合语义与事件顺序 | — | 0 |

## 4. 对交付票判据的修正

| 原判据 | 修正为 |
|---|---|
| `Agent.RunStream` 内部改走 `pkg/hno/runner`，不再使用 `stream.go:89` 的 `for loopCount < a.MaxLoops` 自有循环 | `Agent.RunStream` 的**循环控制策略与 `pkg/hno/runner` 同源**（共用截断/stop 决策）；流式传输保留 `InvokeStream`，因 runner 无流式能力 |
| 流式路径同样支持 `ToolCallLimit` 截断… | 不变 |
| 流式路径同样支持 `StopLoop`，且 `RunStreamResult` 暴露 `StopReason` | `RunStreamDone` 暴露 `StopReason`（`RunStreamResult` 在流开始时即返回，值未知） |
| `stream_aggregator` 语义逐字不变 | 不变 |
| sync 与 stream 消息序列完全一致 | 不变，且**必须由对照测试钉住**（不靠结构保证） |

## 5. 预期 RED（供下一阶段 test-design 用）

三项能力在流式路径当前**均不存在**：

- `RunStreamDone.StopReason` 字段不存在
- `Agent.Config.ToolCallLimit` 对流式路径无任何作用
- 工具 `StopLoop` 对流式路径无任何作用

⇒ `behavior-delta` 成立，有诚实 RED。

## 6. 遗留

- 交付票第 1、3 条判据需按 §4 修正（`dependsOn`/`completionCriteria` 文案，不影响依赖图）。
- `v3-platform.md` 仍写着「删除 `pkg/hno/runner`」这条红线，已因 P0b 失效，待回写。
