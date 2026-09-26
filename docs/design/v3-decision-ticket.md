# Decision Ticket — agno-go v3.0 范围与关键决策

- **work item**: `agno-go-v3-platform`
- **activationId**: `120d6afe-afd3-4b52-9b4b-ffd313aaf63e`
- **状态**: 已确认（用户确认开发进度，`docs/design/v3-platform.md` 已通过评审）
- **证据**: `artifact:docs/design/v3-baseline-evidence.md`

本 Ticket 是 `artifact:decision-ticket:agno-go-v3-scope` 的实体文件。Delivery Ticket 必须引用它，不得由它改写而成。

## D1–D4 裁定

| # | 决策点 | 裁定 | 理由 |
|---|---|---|---|
| **D1** | 会话事件化是否改动 `Session` 对外 JSON？ | **(a) 不改**。事件流作为内部存储，`RunOutput` 保持为派生视图 | 契约层 9 个 Go↔Python fixture 对齐测试（`internal/session/contract`）为硬约束。(a) 使契约层无感，把 fixture 破坏风险移出 v3.0。证据：`artifact:docs/design/v3-baseline-evidence.md#7` |
| **D1-b** | D1 的「对外不变」精确边界 | **仅指 HTTP/JSON 序列化后的响应体与 `Session` 的公开 Go 字段语义**。内部表示（`session.State` 的内部布局、事件存储的物理表结构、索引）**允许重构**。唯一判据：`go test ./internal/session/contract/... -count=1` 9 个测试全绿 | 不写清这条，实施时会在「不动 Session 结构」与「为了存事件必须改结构」之间反复扯皮。review 修正项 M1 |
| **D2** | Store 长期记忆是否进 v3.0？ | **(a) 进** | "能跨会话记住用户"是相对 adk（无此概念）与 17-provider 广度之外的产品差异化。范围 1,200 LOC + 全新概念，已接受 |
| **D3** | `graph.Node` 用 `any` 还是纯泛型？ | **(a) `any` 契约 + `Typed[TIn,TOut]` 泛型包装** | 图引擎需异构节点组合，纯泛型会锁死组合性（红线条目 4 针对公共 API 参数，非引擎内部契约）。泛型包装保留编译期安全出口 |
| **D4** | P0 循环收口是否先单独发 minor？ | **(a) 是，先发 minor** | `pkg/hno/runner` 零引用（死代码）+ 4 份循环并存是持续风险，不应压到 major 发布。证据：`artifact:docs/design/v3-baseline-evidence.md#2` |

## 被推翻的前序决策

| 来源 | 原决策 | 推翻理由 |
|---|---|---|
| `docs/design/go-agent-framework-design.md` §4「循环归属」 | 放弃图引擎，理由「LangGraph Pregel 在 Go 维护成本过高」 | 论据有误：LangGraph 引擎内核约 4,400 行（非全仓库 19 万行）；adk `workflow` 包 6,244 行实现 + 13,800 行测试已在生产跑通。证据：`artifact:docs/design/v3-baseline-evidence.md#4` |
| `docs/design/go-agent-framework-design.md` §4「可观测性」 | 放弃"自研事件体系" | **部分推翻**：仅推翻「会话持久化事件化」；**可观测仍强制走 OTel**。两者职责不同、并行不冲突 |

## 范围边界（明确不做）

- ❌ Pregel BSP 超步屏障、checkpoint time-travel、channel 状态归约
- ❌ adk 完整 `EventActions`（ArtifactDelta / TransferToAgent / Escalate / Compaction / RequestedToolConfirmations）
- ❌ compaction / 上下文压缩
- ❌ trigger（Eventarc / PubSub）
- ❌ A2A 增强、auth 凭证抽象
- ❌ 泛型 `Agent[D,O]`（BC 级破坏，留 v3.1+）

## 保留红线

1. 构造器 ≤ 10 参数，跨切面走接口 + Options
2. 循环只有一份，sync/stream 共享内核
3. 单文件 ≤ ~500 行
4. 禁止 `map[string]any` 作为**公共 API** 参数（引擎内部契约例外，见 D3）
5. 函数签名 ≤ 5 参数
6. 编排器共享内核，禁止复制式复用
7. 无隐式 LLM 调用
8. 配置（构造时）与运行期输入（RunOptions）严格分离

## 5. 范围事实核查与 P0 拆分（test-design 阶段发现）

核实命令：
```bash
grep -rn "ToolCallLimit\|StopLoop\|HITLBlocked" pkg/hno/ --include='*.go'
```
结果仅命中 `pkg/hno/runner/{runner.go,stop.go,runner_test.go}`。

**发现**：当前在线的 `Agent` 只有 `MaxLoops` 一个循环控制概念。`ToolCallLimit` / `StopLoop` / `StopHITLBlocked` / `StopReason` **只存在于死代码 `pkg/hno/runner`**。

**影响**：收口到 runner 天然附带三项当前 Agent 不具备的能力。若在 P0 一并启用，P0 即从「等价重构」变为「新增用户可观察行为」，需 RED 优先 TDD，与 D4「P0 单独发 minor」冲突。

**裁定**：
- **P0 只做严格等价**，不启用 runner 的三项增量能力 → 保持 `behavior-preserving-hardening`，可快速安全发 minor
- 三项能力采纳拆为 **`work-p0b-runner-capability-adoption`**，明确标注为新增用户可观察行为，走 RED 优先 TDD
- 交付票相应从 10 个工作项增至 11 个，`work-p9-release` 增加对 P0b 的依赖

**同时修正了草稿中的一处事实错误**：交付票 P0 的完成标准原写作「为 run.go 现有行为补齐 golden 等价测试：……tool call limit 截断、StopHITLBlocked、StopAfterToolCall……」，误将 runner 独有能力当作 Agent 现有行为。已重写。

**真正的当前缺口**（已核实零覆盖）：`Agent.Run` 的三条错误分类路径——工具执行错误→`ErrCodeToolExecution`、PreHooks 失败→`ErrCodeInputCheck`、PostHooks 失败→`ErrCodeOutputCheck`。

## 6. review 修正记录（2026-08 独立子 Agent 审查）

本决策票与交付票经独立 fresh 子 Agent（`claude -p`，无本会话上下文）审查。逐条处置：

| 审查项 | 处置 | 说明 |
|---|---|---|
| **I1-a** skills LOC 398/684 | **接受并澄清** | 两个数都对，口径不同。已在基线证据加「口径说明」：默认非测试口径 |
| **I1-b** `team.go:295` 是注释非调用 | **接受并修正** | 原采集命令 `grep "run.Loop" \| head -1` 确实返回注释行 295。真实构造点 300，命令已改为精确匹配 |
| **I1-c** workflow builder 声称 grep 空 | **接受并修正** | 原命令 `grep 'AddNode\|AddEdge\|Builder'` 会被 `strings.Builder` 与注释污染。命令已收窄为 `AddNode\|AddEdge`，实测 0；结论（无图式 builder）不变 |
| **I2** 并行分组 2 依据不足 | **接受** | 已在 `work-p7-observability` 增加显式完成标准：只接线不改 runner 公共接口、不依赖图引擎 |
| **I3** 三处标准不可测 | **接受** | LOC 上限、Session 不变、P9 说明 均改为可执行判据 |
| **M1** D1 边界歧义 | **接受** | 新增 **D1-b** 行，区分「对外契约不变」与「内部表示可重构」 |
| **M2** `work-p8-store` 不应抢跑 | **驳回，附理由** | Store 与图引擎无依赖关系（`pkg/hno/store/` 独立包，图引擎不消费它）。强绑 `dependsOn: [work-p1]` 会无理由地阻塞一条独立资产。改为在完成标准中写明**禁止与图引擎耦合** |
| **M3** 缺 BC 影响面量化 | **接受** | 已在 `work-p9-release` 增加必交付的 `v3-breaking-changes.md` |
| **B1** schema 校验未跑通 | **非缺陷** | 审查者因 heredoc 引号问题失败；已提供独立脚本并校验通过（`VALID`，10 work items） |
| **B2** 参考仓库不可访问 | **部分接受** | 仓库实际存在，属审查环境访问问题。但审查暴露了**真实脆弱性**：`/tmp` 是易失的。已在基线证据加 §0 溯源（commit SHA + 重新拉取指引） |


| 风险 | 缓解 | 归属工作项 |
|---|---|---|
| 会话事件化仍触及 session 存储形态 | D1 选 (a) 内部存储 + 派生视图；D1-b 明确「对外契约不变 ≠ 内部表示冻结」；契约层 9 测试为唯一判据 | `work-p5-event-hitl` |
| 图引擎与 agent 循环职责重叠 | 明确定界：图管节点间控制流，节点管节点内循环；节点内 tool loop 不上图为环 | `work-p1-graph-engine` |
| P0 收口时行为漂移（cache/hitl/hooks 交织） | 必须先补行为等价测试（golden cases）再切换 | `work-p0-loop-consolidation` |
