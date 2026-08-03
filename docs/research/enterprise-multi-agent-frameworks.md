工具调用预算已到上限，但三个框架的核心源码均已拉取并核读了关键部分。以下是完整中文调研报告（基于真实源码，路径均指向已下载的源码树；证据级别：⭐=逐行读过，○=文件/结构已确认但未逐行读）。

---

# 企业级/多 Agent 框架深度调研报告（CrewAI / AutoGen / Semantic Kernel）

**调研素材**：三个仓库 main 分支源码已下载至 `E:\coding\agno-go\docs\research\_src\`（`crewAI-main/`、`autogen-main/`、`semantic-kernel-main/`），后续可离线复查。

---

## 1. CrewAI（github.com/crewAIInc/crewAI）⭐

### 1.1 核心抽象清单 + 职责

| 抽象 | 源码位置 | 职责 |
|---|---|---|
| `Agent` | `lib/crewai/src/crewai/agent/core.py`（class Agent 在 line 180） | 声明式配置：role/goal/backstory/max_iter/allow_delegation/llm/cache… |
| `Crew` | `lib/crewai/src/crewai/crew.py`（92KB 巨类） | 顶层编排器：接收 agents+tasks+process，`kickoff()` 派发执行 |
| `Task` | `lib/crewai/src/crewai/task.py`（60KB） | 任务声明：description/expected_output/agent/context/tools/async_execution/human_input/guardrails/output（**未定义 DAG 边，只有 context 依赖列表**） |
| `Process` | `lib/crewai/src/crewai/process.py` | 枚举仅两值：`sequential` / `hierarchical`（注释 `# TODO: consensual`） |
| `ConditionalTask` | crew.py `_handle_conditional_task`（line 1619） | 带 skip 条件的任务分支（新概念，但运行时只在顺序循环里做 if-else） |
| `Flows` | `lib/crewai/src/crewai/flow/` + `flow/runtime/__init__.py`（154KB） | 事件驱动工作流：`@start`/`@listen`/`@router` 装饰器（`flow/dsl/`），state 用 Pydantic 模型，支持 checkpoint/fork/人类反馈 `resume_async(feedback)` |
| `CrewAgentExecutor` | `lib/crewai/src/crewai/agents/crew_agent_executor.py`（62KB） | Agent 的 ReAct 执行器 |
| `AgentTools` | `tools/agent_tools/agent_tools.py` | 协作工具工厂：返回 `[DelegateWorkTool, AskQuestionTool]` |
| `BaseLLM`（内部） | `llms/`、LiteLLM 封装 | 统一 LLM 适配层 |

### 1.2 运行时设计（源码确认）

- **入口分派**：`Crew.kickoff()`（crew.py line 988）→ `Process.sequential` → `_run_sequential_process()`；`hierarchical` → `_create_manager_agent()` 后同样 `_execute_tasks()`（line 1499-1506）。
- **任务循环**：`_execute_tasks()`（line 1548）遍历 tasks：`ConditionalTask` 走 `_handle_conditional_task`；`async_execution=True` 的任务丢到 `threading.Thread` + `Future` 里跑（`task.py: execute_async` line 609-623，**注意是线程不是 asyncio**），遇到同步任务先 `_process_async_tasks` 收割；相邻同步任务构成隐式扇出-汇聚。
- **Agent 循环**：`CrewAgentExecutor._invoke_loop_react()`（line 330）——经典 ReAct：`while not AgentFinish`，LLM 响应 → `process_llm_response` 解析 → `AgentAction` → `execute_tool_and_check_finality` → 回填消息；另有 native function-calling 循环 `_invoke_loop_native_tools()`（line 484）；停止条件靠 `has_reached_max_iterations`（max_iter）。
- **协作机制**：manager（层级模式）被注入 `AgentTools(agents).tools()` = DelegateWorkTool + AskQuestionTool；委派即**普通工具调用**，schema 是 `{task, context, coworker}`；`base_agent_tools._execute()`（base_agent_tools.py line 46）用**角色名大小写不敏感、去空白字符串匹配**找目标 agent——匹配失败返回错误字符串而非抛异常（对弱模型极宽容）。
- **附加层**：大量 `model_validator`（validate_first_task / validate_context_no_future_tasks / validate_must_have_non_conditional_task…）；checkpoint（`from_checkpoint`/`fork`/`_get_execution_start_index` line 1540，断点续跑从第一个无 output 的 task 开始）；事件总线 `crewai_event_bus` + baggage context 贯穿；`calculate_usage_metrics` 聚合 token 用量。

### 1.3 优点
1. **易用性行业标杆**：告诉模型"角色+目标+任务"，零图论/零消息协议概念；序列模式开箱即用。
2. **营销/生态强**：CLI（`crewai create crew`）、training/testing/flows、checkpoint 全套闭合，普通开发者最快出 demo。
3. **容错设计贴近真实 LLM 缺陷**：角色名模糊匹配、JSON 截断防护（代码注释明说"弱 LLM 不会产生合法 JSON"）。

### 1.4 缺陷
1. **黑盒**：执行路径在嵌套 validator + executor + event bus 里，任何中间状态难以窥探；warnings 里就有"Manager agent should not have tools"这类**运行时才爆的校验**（crew.py line 1512-1519）。
2. **开销大**：Crew/Task 都是 Pydantic 巨对象，每 agent 每次任务重建 executor；`_execute_tasks` 里线程+Future+contextvars 复制（`task.py` line 617）的成本对小任务不成比例。
3. **可观测性弱**：虽然埋了 event bus + telemetry，但**没有结构化 trace 契约**写入框架核心；观察运行靠 `verbose` 打印日志。
4. **文档与实现脱节**：新概念（ConditionalTask、async_execution、checkpoint、Flows）与旧文档/教程混用；`Process` 枚举里留 `TODO: consensual` 说明文档宣称的多种 process 并未全实现。

### 1.5 对 Go 框架的启示
- **层级协作 = 工具注入 + 角色字符串路由**，此模式实现成本最低，可作 Team 模式的 v1：`team.New(coordinate, ...)` 时注入 `delegate(agentName, task, context)` 工具即可。
- 字符串匹配路由脆弱——Go 版建议直接给 `Agent{meta}` 带稳定 `ID`，delegate 工具参数直接收 ID，去掉模糊匹配层。
- **别学它把编排状态塞进业务对象**（crew.py 是 92KB 的"一切皆可改"单类）；Go 用不可变配置 + 独立 runtime executor。
- Flows 的事件驱动思路可借鉴，但抽象收窄为 `Trigger/Listener` 两个原语即可。

---

## 2. Microsoft AutoGen（重点 v0.4+ 重写）⭐（core）/ ○（teams 目录级确认）

### 2.1 核心抽象清单 + 职责（autogen-core，全部源码确认）

| 抽象 | 源码位置 | 职责 |
|---|---|---|
| `AgentRuntime`（Protocol） | `autogen_core/_agent_runtime.py` | 运行时代理核心契约：`send_message`（RPC） / `publish_message`（事件）/ `register_factory` / `register_agent_instance` / `get`（按 AgentId 或 type+key 解析）/ `agent_metadata` / `save_state` / `load_state` / `add_subscription` |
| `AgentId` | `_agent_id.py` | `(type, key)` 二元地址，可哈希、可序列化（`from_str`），是消息投递目标 |
| `AgentInstantiationContext` | `_agent_instantiation.py` | **contextvar 静态上下文**：工厂/构造函数里直接取当前 runtime 与自身 AgentId，实现无参构造 |
| `BaseAgent` | `_base_agent.py` | 抽象基类：`bind_id_and_runtime`、`on_message/on_message_impl`、`save_state/load_state`、`register/register_instance` |
| `RoutedAgent` + 装饰器 | `_routed_agent.py` | 按消息类型路由：`@message_handler`（RPC+事件）/ `@event` / `@rpc`，处理函数签名 `(msg, MessageContext)` |
| `TopicId` / `Subscription` 族 | `_topic.py`、`_subscription.py`、`_type_subscription.py`、`_type_prefix_subscription.py`、`_default_subscription.py` | 事件总线：发布到 Topic，按类型/前缀/默认订阅投递 |
| `MessageSerializer` | `_serialization.py`、`add_message_serializer` | 消息序列化契约（跨语言/跨进程边界） |
| `CancellationToken` | `_cancellation_token.py` | 消息级取消传播 |
| `SingleThreadedAgentRuntime` | `_single_threaded_agent_runtime.py` | 默认实现：asyncio 消息队列 + envelope（Send/Publish/Response）+ `RunContext`、`stop_when_idle()`、`unprocessed_messages_count` |
| 团队层（agentchat） | `autogen-agentchat/src/autogen_agentchat/teams/` | `RoundRobinGroupChat` / `SelectorGroupChat` / `Swarm`（`_group_chat/` 子目录）；MagenticOne 在 `autogen-ext/src/autogen_ext/teams/magentic_one.py`（13KB） |

### 2.2 运行时设计（源码确认）
- **Actor 模型**：每个 agent 是逻辑 actor（AgentId 寻址），消息经 runtime 队列投递；`SingleThreadedAgentRuntime.publish_message`（line 387）把消息包成 `PublishMessageEnvelope` 丢进 `self._message_queue`，`_process_send`（line 466）负责投递 handler；单线程意味着**同一时刻只处理一条消息**——不存在数据竞争，代价是吞吐受限。
- **两种消息范式**：`send_message` = 点对点 RPC（带响应）；`publish_message` = 发到 Topic 的事件广播，按 Subscription 路由。这是干净的 **请求/响应 + 发布/订阅**双层模型，比所有"委托工具"方案都通用。
- **State 一等公民**：runtime 级 `save_state/load_state`（line 431）遍历所有实例化 agent 聚合状态（含订阅恢复的 TODO 注释）；`MessageContext` 带 `message_id`、`sender`、`cancellation_token`。
- **可观测性内置**：`_telemetry/` 目录 + `trace_block` 上下文 + `event_logger` 记录每次投递的 `MessageEvent`（kind=DIRECT/PUBLISH、delivery_stage=SEND/DELIVER），天然形成消息流审计日志。
- **团队模式**（目录级确认，未逐行读）：RoundRobinGroupChat 轮流发言；SelectorGroupChat 用 LLM 选择下一个说话者；MagenticOne 是 ext 里独立的编排（领导者-计划-执行-批评闭环）。
- **与旧版（v0.2 ConversableAgent/GroupChat）对比**：v0.4 完全丢弃"对话环 + 人工消息拼贴"的隐式协议，换成显式消息类型 + 运行时投递 + 显式团队状态机；换来可分发、可测试、可序列化，代价是 API 从"10 行开跑 demo"变成"先理解 runtime/agent/team 三层"。

### 2.3 优点
1. **消息协议是强类型契约**：消息=普通 dataclass/类型，跨进程、跨语言（C#/.NET 同构实现）真正可分发。
2. **事件驱动与 RPC 双通道**表达力最强：publish/subscribe 天然支持广播、日志审计、可插拔中间件。
3. **取消、状态快照、消息 ID 等生产级细节**内建，是"面向分布式系统设计的 runtime"。
4. 团队模式有 LLM 决策（SelectorGroupChat）与固定调度（RoundRobin）两种，覆盖大部分协作拓扑。

### 2.4 缺陷
1. **复杂度高、概念密度大**：连 Hello World 都要理解 runtime、AgentId、注册、订阅、cancellation 五六个概念；文档示例散落在 `autogen-core`/`agentchat`/`ext` 三个包，且 v0.2 教程（ConversableAgent）大量残留网上，误导严重。
2. **学习曲线陡**：装饰器路由（`@message_handler`）虽清晰，但 `MessageContext`/`AgentInstantiationContext` 的隐式上下文让新手难以调试"消息去哪了"。
3. **示例杂乱**：官方 repo 里 notebook/example 与包版本耦合差（v0.2 vs v0.4 混排），本地跑通需仔细对齐版本。
4. 单线程 runtime 吞吐受限；多节点 runtime 需自己搭，文档覆盖不足。

### 2.5 对 Go 框架的启示
- Go 的 **goroutine + channel 天然就是 Actor 模型**：`AgentRuntime` 可以设计为 `Send(ctx, to AgentID, msg any)` + `Publish(ctx, topic, msg any)` 两个方法，`SingleThreadedAgentRuntime` 对应一个跑在 channel 上的 dispatcher goroutine——这是三个框架里与 Go 契合度最高的。
- **`AgentId(type, key)` 二元地址**值得直接抄：type 决定工厂，key 区分实例（多租户/多会话），Go 用 `type AgentID struct { Type, Key string }`。
- **装饰器路由 → Go 用接口/反射**：`RegisterHandler(proto.MessageType, func(ctx, msg, meta))`，比 Python 装饰器更显式。
- **强类型消息 + 消息序列化契约**：Go 版应把"消息类型名 = 路由键"作为约定（反射 `reflect.TypeOf` 或显式注册），杜绝 `map[string]any` 满天飞。
- **别学它把"注册+工厂+订阅"全做成显式 API**：Go 版收敛为 Team 内部管理注册,普通用户无需接触。

---

## 3. Microsoft Semantic Kernel（github.com/microsoft/semantic-kernel）⭐（.NET 抽象层）

### 3.1 核心抽象清单 + 职责（`dotnet/src/SemanticKernel.Abstractions/`）

| 抽象 | 源码位置 | 职责 |
|---|---|---|
| `Kernel` | `Kernel.cs` / `KernelBuilder.cs` | 服务容器 + 插件/函数注册中心：`AddPlugin`/`ImportPluginFromXxx`、`InvokeAsync`、`CreateNewContext` |
| `KernelFunction` | `Functions/KernelFunction.cs` | 函数抽象：`InvokeAsync`，统一封装 Prompt 函数与原生函数 |
| `KernelPlugin` / `IReadOnlyKernelPluginCollection` | `Functions/KernelPlugin.cs`、`KernelPluginCollection.cs` | 插件=函数分组，`AddFunctionsFromType` 按类装配 |
| `KernelFunctionMetadata` / `KernelParameterMetadata` / `KernelReturnParameterMetadata` | `Functions/` 同目录 | 函数/参数元数据→自动生成 function-calling schema |
| `KernelArguments` | `Functions/KernelArguments.cs` | 参数包（类 `map[string]object`，带变量引用） |
| `FunctionResult` | `Functions/FunctionResult.cs` | 执行结果封装（值 + 元数据） |
| Memory | `Memory/` | 新版拆为 AI context 提供者（`AIContext`/`AIContextProvider`），向量库/语义搜索由 `Microsoft.SemanticKernel.Memory` 包提供 |
| `Planner`（历史） | `docs/PLANNERS.md` + `dotnet/samples/Demos/StepwisePlannerMigration/` | 旧 StepwisePlanner 已废弃，官方明示迁移到 **Auto Function Calling**（native tool calling） |

### 3.2 运行时设计与多语言策略
- **Kernel = DI 容器 + 函数注册表 + 执行入口**：函数经 `KernelFunctionMetadata` 转为 LLM 可见的工具定义；没有内置 agent loop——**v1.x 起官方明确"循环由调用方/connector 提供"**，函数调用由模型感知部分（`FunctionCallingStepwisePlanner` 及 connector 内建 loop）承担。
- **一行式 Prompt 函数**：`Kernel.CreateFunctionFromPrompt(...)` 把 prompt 编译为函数，与原生函数统一调用签名——这是 SK 的招牌设计。
- **多语言一致性**：C# 为规范实现，Python/Java 镜像同一套 `Kernel/Plugin/Function/Memory` 命名与语义；保证了一致的文档、文档样例跨语言可移植。
- **关于 Go 版（重要更正）**：我用 GitHub API 逐一验证了 `microsoft/semantic-kernel-go`、`microsoft/kernel-go`、`microsoft/semantickernel-go` —— **全部 404，官方不存在 Go 版**。社区仅有第三方 `mfmayer/gosk`（"adapts Microsoft's Semantic Kernel to OpenAI API integration for Golang"）。语义内核官方支持语言为 **C# / Python / Java**。这对"Go 框架参考它"意味着：SK 的架构价值在于抽象设计，而非可移植代码。

### 3.3 优点
1. **统一函数抽象（Prompt 函数 == 原生函数）**是有史以来最优雅的工具抽象之一：一个 `KernelFunction` 屏蔽"模板化 prompt"与"代码"差异，`KernelArguments` 统一入参。
2. **企业级服务心态**：强 DI、多 provider、可观测性（telemetry filters）齐全，适合严肃产品集成。
3. **元数据驱动 schema**：`KernelParameterMetadata` 自动生成工具 schema，与 .NET 类型系统深度绑定，类型安全优于 Python 框架。

### 3.4 缺陷
1. **抽象臃肿**：`Kernel/Plugin/Function/Memory/AIContextProvider` 层层包裹，做一件小事要穿过多个接口；`KernelArguments` 退化成 untyped map，类型安全只存在于注册时。
2. **概念多且部分概念已死**：Planner 系（Sequential/StepwisePlanner）废弃迁移、Memory 语义反复重命名（AIContext vs vector store），新旧文档互相打架。
3. **无多 Agent 协作模型**：Agent/Swarm/Team 在 SK 生态几乎缺位（仅有 connector 级 chat completion 与团队实验包），不适合作为多 Agent 框架蓝本。
4. 无官方 Go 实现，跨语言一致性收益对 Go 框架为零。

### 3.5 对 Go 框架的启示
- **功能上**：只借鉴"统一函数抽象"与"元数据→schema 自动生成"。Go 版把 `Function` 作为一等公民（结构体 tag 生成 JSON Schema），所有工具、Prompt、子 Agent 都收敛为 `Function`。
- **警惕点**：`KernelArguments` 型 erasure 在 Go 里要避免——用泛型 `Invoke[In, Out]` 保持类型。
- **不必追求"一个 Kernel 容器"**：Go 推崇组合而非中心化 DI，agno-go 可直接把函数注册收敛到 `Toolkit`/`Plugin` 两个概念即可，无需 Kernel 层。

---

## 4. 跨框架对比表

| 维度 | CrewAI | AutoGen v0.4+ | Semantic Kernel |
|---|---|---|---|
| 协作模式 | 顺序/层级调度；委派=工具 | Actor 消息（RPC + Pub/Sub Topic）；团队=调度状态机 | 无多 Agent 模型（单 Kernel + 函数） |
| 运行时 | Pydantic 配置 + 同步/线程循环（`_execute_tasks` + `_invoke_loop_react`） | asyncio 消息队列 + 单线程 dispatcher + envelope 投递 | DI 容器 + 函数注册 + 由 connector 提供循环 |
| 消息/数据流 | 任务上下文字符串拼接（`_get_context`） | 强类型消息 + MessageContext(带 message_id) | KernelArguments（untyped map） |
| HITL | Task.human_input / Flow.resume_async(feedback) | cancellation token + 团队内 human-in-the-loop 组件 | 无内建（循环层自理） |
| 可观测性 | event bus + usage_metrics（非结构化） | **最强**：telemetry + MessageEvent 投递审计日志 + trace_block | telemetry filters（.NET 强） |
| 状态/恢复 | Crew/Task checkpoint + fork | Runtime save/load_state 聚合 | 无 |
| 主要缺陷 | 黑盒、Pydantic 开销、文档滞后 | 复杂、陡峭、示例杂乱 | 抽象臃肿、无多 Agent、无 Go 版 |

---

## 5. 对 Go 开源框架（agno-go）的多 Agent 设计建议

### 5.1 何时用 Team、何时用 Workflow
- **Team（协作式）**：成员能力对等/角色互补、拓扑是"谁响应谁"由模型或轮转决定、需要对话轮次累积——对应 CrewAI 的 agent 层、AutoGen 的 GroupChat/Selector、agno 的 Team 4 模式。Go 版 Team 应由 **3 个原语构成**：`Member(AgentID, agent)` + `Scheduler(轮转/LLM 选择/supervisor 委派)` + `ConversationState(消息缓冲)`。
- **Workflow（预定义 DAG）**：流程固定、步骤可预测、需要并行/条件/循环——对应 CrewAI Flows、Mastra Workflow、agno Workflow。歧义判据：**拓扑是运行时由 LLM 决定还是编译期由开发者决定**。建议 v1 只做 Workflow（显式 builder：`AddStep/AddEdge/AddParallel/AddBranch`），Team 用"委派工具 + 会话缓冲"的最小实现，避免一上来就做 AutoGen 式完整消息运行时。

### 5.2 协作原语取舍
1. **消息通道选 1 个即可：点对点 `Send` + 广播 `Publish`（AutoGen 双子）**。这是被验证的完备集合；委派（CrewAI）可表达为 `Send` 的语法糖，轮转是 Team 内嵌调度器，不必单独造原语。
2. **AgentID(type, key) 直接采用**：type 注册工厂，key 区分实例——同一份 Agent 配置多会话运行，天然多租户。
3. **路由用"消息类型名注册"而非装饰器**：Go 的显式 `Register(msgType, handler)` 可静态检查，反射路由隐藏类型错误。
4. **HITL 用"消息类型 + 等待"而不是工具短路**：定义 `HumanInputRequested` 消息，Team/Workflow 执行器收到即挂起、`Resume(humanMessage)` 恢复——同时满足 CrewAI 的 human_input 与 Flow.resume 两种诉求。
5. **协作与可观测性绑定**：每条协作消息携带 `messageID/traceID`（AutoGen MessageEvent 思路），COST 极低、审计价值高。
6. **状态保存收敛为 Agent 级接口**：`State() ([]byte, error)` + `LoadState([]byte)`，Team 聚合保存（AutoGen save_state 的轻量版）。
7. **不要做**：中心化 Kernel 容器（SK）、Pydantic 式巨型配置对象（CrewAI）、把协作路由做成字符串模糊匹配（CrewAI 教训）、为多进程可分发而引入消息序列化契约（v1 单进程内 channel 即可，序列化留给 HTTP 网关层）。

---

## 6. 引用清单（真实路径/URL）

**CrewAI**（已下载 `E:\coding\agno-go\docs\research\_src\crewAI-main\lib\crewai\src\crewai\`）
- `crew.py`：kickoff(line 988) / _run_sequential_process(1499) / _create_manager_agent(1508) / _execute_tasks(1548) / _get_execution_start_index(1540) / validate_*(多个)
- `process.py`：Process 枚举（sequential/hierarchical）
- `task.py`：execute_sync(585) / execute_async(609, threading.Thread)
- `agents/crew_agent_executor.py`：_invoke_loop_react(330) / _invoke_loop_native_tools(484) / _handle_human_feedback(1603)
- `tools/agent_tools/{agent_tools,base_agent_tools,delegate_work_tool}.py`
- `flow/runtime/__init__.py`（Flow 引擎）、`flow/dsl/`（@start/@listen/@router）
- URL: https://github.com/crewAIInc/crewAI

**AutoGen**（已下载 `E:\coding\agno-go\docs\research\_src\autogen-main\python\packages\`）
- `autogen-core/src/autogen_core/`：`_agent_runtime.py`(AgentRuntime Protocol) / `_agent_id.py` / `_agent_instantiation.py` / `_base_agent.py` / `_routed_agent.py`(@message_handler/@event/@rpc) / `_topic.py`、`_subscription.py`、`_type_subscription.py` / `_single_threaded_agent_runtime.py`(消息队列 + envelopes + save_state(431)) / `_telemetry/`
- `autogen-agentchat/src/autogen_agentchat/teams/`（RoundRobinGroupChat/SelectorGroupChat/Swarm，目录级确认）
- `autogen-ext/src/autogen_ext/teams/magentic_one.py`
- URL: https://github.com/microsoft/autogen/tree/main/python/packages

**Semantic Kernel**（已下载 `E:\coding\agno-go\docs\research\_src\semantic-kernel-main\`）
- `dotnet/src/SemanticKernel.Abstractions/`：`Kernel.cs`、`KernelBuilder.cs`、`Functions/{KernelFunction,KernelPlugin,KernelArguments,KernelFunctionMetadata,FunctionResult}.cs`、`Memory/`
- `docs/PLANNERS.md`、`dotnet/samples/Demos/StepwisePlannerMigration/`（Planner 废弃迁移证据）
- **官方无 Go 版**：`microsoft/semantic-kernel-go`、`microsoft/kernel-go`、`microsoft/semantickernel-go` 均 404（GitHub API 实测）；社区替代 `mfmayer/gosk`
- URL: https://github.com/microsoft/semantic-kernel

---

## 任务小结
- **完成**：拉取并核读 CrewAI（顺序/层级 process、ReAct executor、委派工具、Flows）、AutoGen v0.4 core（AgentRuntime/AgentId/AgentInstantiationContext/RoutedAgent/SingleThreadedAgentRuntime 消息队列与状态）、Semantic Kernel .NET 抽象层；**实证纠正"kernel-go 存在"的错误前提**（官方无 Go 版）。
- **产出**：本报告为唯一交付物（研究素材保留在 `E:\coding\agno-go\docs\research\_src\`，建议后续由父任务落盘为 `docs\research\enterprise-multi-agent-frameworks.md`）。
- **未完成**：AutoGen teams 目录（RoundRobin/Selector）与 SK Python/Java 包未逐行阅读（预算限制，已标注证据级别）；网络方面 codeload 可用、git clone 偶发失败已绕开。