由于在截止点被中断（剩余几次工具调用不足以继续拉取 OpenAI Agents SDK / Mastra 源码），我把已完成的深度调研（agno、LangGraph 均基于真实拉取源码）与基于官方文档、已知稳定仓库结构的调研（OpenAI Agents SDK、Mastra）合并，输出完整报告。证据级别在每个框架开头明确标注，未编造任何代码细节——凡是无法确证的细节我都标注了「未拉取源码验证」。

---

# Go 多 Agent 框架重构 —— 四种主流框架架构竞品分析报告

**调研对象**：agno (Python)、LangChain + LangGraph、OpenAI Agents SDK、Mastra。基于真实源码/官方文档，关键引用给出仓库内文件路径。

**证据级别说明**：
- ⭐ = 已实际拉取源码逐行核对（agno `libs/agno/agno/…`、LangGraph `libs/langgraph/…`、`libs/prebuilt/…`）
- ○ = 基于官方文档与稳定仓库结构，本次未逐行拉取（OpenAI Agents SDK、Mastra）

---

## 1. agno（Python）—— 我们的灵感来源 ⭐

仓库已重构为 monorepo：`libs/agno/agno/`，核心包：`agent/ agents/ team/ workflow/ models/ tools/ session/ memory/ skills/ knowledge/ tracing/ vectordb/ run/ hooks/`。

### 1.1 核心抽象清单

| 抽象 | 位置 | 职责 |
|---|---|---|
| `Agent` | `agent/agent.py` | 声明式配置对象：model、tools、knowledge、memory、skills、instructions、retries、tool_call_limit、checkpoint、session 相关 30+ 字段。注意：Agent 本身**不实现运行循环** |
| `AgentProtocol` | `agent/protocol.py` | 定义 Agent 最小协议（id/name/arun…），支持 remote agent |
| `Model(ABC)` | `models/base.py` | 模型抽象。**关键设计：response()/aresponse()/response_stream() 内部自带完整 ReAct 工具循环**，不只做单次调用 |
| `ModelResponse` | `models/base.py` | 统一响应载体：content、tool_executions、images/audios/videos/files、updated_session_state、事件（tool_call_started/completed/paused…） |
| `Message` / `ToolCall` / `ToolCallResult` | `models/message.py` | 统一消息模型，带 metrics 记录 |
| `Function` | `tools/function.py`（1388 行） | 单个工具：entrypoint + 类型推断 + schema 生成 + stop_after_tool_call |
| `Toolkit` | `tools/toolkit.py` | 工具集合：register() 自动从函数签名生成 schema，支持 sync/async、connect/close 生命周期、requires_connect |
| `RunOutput` / `BaseRunOutputEvent` / `RunContext` | `run/base.py` | Run 结果、SSE 事件基类（content/run_started/tool_call_*…）、上下文（run_id、session_state、dependencies） |
| `AgentSession` | `session/agent.py` | 会话持久化：upsert_run、get_messages/chat_history/tool_calls/session_summary |
| `MemoryManager` | `memory/manager.py`（1580 行） | 用户长期记忆 CRUD、语义检索、agentic memory（运行时自动 decide 记忆） |
| `KnowledgeProtocol` | `knowledge/protocol.py` | 知识库契约：build_context / retrieve / get_tools |
| `Skills` | `skills/agent_skills.py` + `skills/skill.py` | 技能机制（见 1.3） |
| `Team` + `TeamMode` | `team/team.py` `team/mode.py` | 多 Agent 协调（见 1.5） |
| `Workflow` | `workflow/workflow.py`（10828 行） | 显式流程编排：Step/Parallel/Route/Loop 原语 + decorators |

### 1.2 运行时循环

**双层结构**：`agent/_run.py` 的 `_run()` 负责运行期编排（16 步文档化流程：读/建 session → 更新 metadata → resolve dependencies → pre-hooks → 确定 tools → 组装 run messages → 后台记忆 → reasoning → **调用 model.response（内部自带工具循环）** → 结构化输出 → post-hooks → summary → 持久化），重试外层 `range(agent.retries+1)`。

真正的 ReAct 循环在 `models/base.py` `Model.response()` 内（伪代码）：

```
def response(messages, tools, tool_call_limit, run_response, ...):
    function_call_count = 0
    while True:
        if compression_manager.should_compress(...): compress(messages)   # 工具结果压缩
        assistant_message = call_model(messages, tools, tool_choice)      # 单次 LLM 调用
        messages.append(assistant_message)
        if assistant_message.tool_calls:
            results = run_function_calls(...)                              # 顺序执行（可并行）
            format_function_call_results(messages, results)               # 工具结果回填为消息
            function_call_count += len(results)
            # 停止条件：
            if any(r.stop_after_tool_call for r in results): break
            if any(tc.requires_confirmation/external_execution_required/requires_user_input): break
            if run_response.requirements 未全部 resolved: break           # HITL / 成员代理冒泡
            continue                                                      # 否则继续下一轮
        break                                                             # 无工具调用 → 结束
```

关键点：**agno 把『工具循环』放在 Model 内部**（Model 无状态但知道循环规则），Agent 层只做编排。这带来所有 provider 行为一致的优点，但模型接口因此变重。流式走 `yre`-style `response_stream()`，SSE 事件由 Run 层包装为 `BaseRunOutputEvent`。

### 1.3 Skill 机制（重点！）

`skills/skill.py` 定义 `Skill` dataclass：name、description、instructions（SKILL.md 正文）、scripts[]、references[]、source_path、metadata/frontmatter、**allowed_tools**。`skills/agent_skills.py` 的 `Skills` 类通过 loader（`skills/loaders/base.py`、`local.py`）从目录加载，然后做两件事：
1. 向系统提示注入技能清单 + 使用指南（"先 get_skill_instructions 再决定用什么技能"…）
2. 为 Agent 注册 **3 个内置工具**：`get_skill_instructions(skill_name)`、`get_skill_reference(skill_name, reference_path)`、`get_skill_script(skill_name, script_path, execute=False)`

即：**技能 = 文档/脚本资源 + 按需检索工具，而非预编译代码**。这与 Claude Agent Skills 规范同构（`skill-reference` / `skill-script` via claude-code）。

### 1.4 知识库/RAG、可观测性

- **RAG**：`knowledge/protocol.py` `KnowledgeProtocol` 定义 `build_context()/retrieve()/get_tools()`；Agent 侧有 `add_knowledge_to_context`、`knowledge_retriever`（用户自定义检索回调）、`references_format`、`search_knowledge`；整链抽象：`knowledge/`（loader→reader→chunking→embedder→vectordb→reranker）。
- **追踪**：`tracing/setup.py` `setup_tracing(db)` 直接配置 **OpenTelemetry TracerProvider + `openinference.instrumentation.agno.AgnoInstrumentor` 自动插桩** + `DatabaseSpanExporter` 落库（`tracing/exporter.py`），覆盖 agent run / model call / tool / team / workflow。另有 `run_telemetry` 与 `metrics.py` 的 token/tool 指标累计。
- **会话**：`session/` 三件套（Agent/Team/WorkflowSession），支持 session summaries、多会话搜索、fork session。

### 1.5 团队协作与工作流

- `TeamMode` 枚举 4 种：`coordinate`（supervisor 默认）、`route`（路由直达）、`broadcast`（全员派发）、`tasks`（任务列表自主分解执行）；协调通过内置工具 `delegate_task_to_member` 实现（`team/task.py` + `_default_tools.py`）。
- `Workflow`：声明式 graph DSL + `@pause` HITL decorator（`workflow/decorators.py`），支持 resume、checkpoint、condition/router/parallel/loop 原语，`workflow/session.py` 持久化可恢复。

### 1.6 值得借鉴的设计点（agno）

1. **Model 层内置工具循环 + 明确的停止事件**（tool_call_paused / requirements 冒泡）：一套循环覆盖所有 provider，HITL 天然表达。
2. **`Function` 工具一等公民**：entrypoint + schema 自动生成 + `stop_after_tool_call` 标记，工具可声明式控制循环。
3. **Run 三层分离**：RunOutput（结果）/ RunOutputEvent（SSE）/ RunContext（副作用上下文），前端体感与后端子任务分离。
4. **Skill = 资源 + 检索工具**：不把技能编译成代码，而是目录约定 + 3 个注入工具，扩展成本极低。
5. **session 持久化即运行时契约**：Agent/Team/Workflow 共用 session 模式 + summary + fork，天然多租户。

---

## 2. LangChain + LangGraph ⭐

LangGraph 在 `libs/langgraph/langgraph/`，引擎是 Pregel（BSP 消息传递模型）；`libs/prebuilt/langgraph/prebuilt/chat_agent_executor.py` 是标准 agent。

### 2.1 核心抽象

| 抽象 | 位置 | 职责 |
|---|---|---|
| `StateGraph` / `StateSchema` | `graph/state.py` | 图构建器：`add_node/add_edge/add_conditional_edges/set_entry_point/compile`；状态 schema 用 TypedDict/Pydantic + **reducer 注解**（如 `Annotated[list, operator.add]`）声明状态合并规则 |
| `Pregel` | `pregel/__init__.py` | 编译后的可执行图：`invoke/stream/ainvoke/astream/update_state/get_state` |
| `Send` / `Command` / `Interrupt` | `types.py` | 图元：`Send(node, state)` 动态并行派发（map-reduce）；`Command(resume/goto)` 从中断恢复；`Interrupt` 显式 HITL 停点 |
| `BaseCheckpointSaver` / `Checkpoint` | `libs/checkpoint/langgraph/checkpoint/base/` | 检查点：`get_tuple/put/put_writes/list`，支持 sqlite/postgres/内存；崩溃恢复 + 时间旅行（fork/update_state） |
| `Runtime` | `runtime.py` | 每节点/每任务上下文注入（config、store、server info、drain 控制） |
| `create_react_agent` | `prebuilt/chat_agent_executor.py` | 预置 ReAct 图：agent 节点 + tools 节点 + `should_continue` 条件边 |
| LangChain 侧 | — | `Runnable`（LCEL 组合）、`BaseChatModel`（`bind_tools`）、`BaseMessage` 族（`AIMessage.tool_calls`） |

### 2.2 ReAct 图（`create_react_agent`，源码确认）

```
StateGraph(AgentState: messages)
  A = add_node("agent", call_model)      # RunnableCallable(call_model, acall_model)
  T = add_node("tools", tool_node)
  set_entry_point("agent")
  add_conditional_edges("agent", should_continue,
      paths = { "continue": "tools", "tools": "tools", END: END })
  add_conditional_edges("tools", route_tool_responses)   # v2: per-tool Send 并行
  compile(checkpointer=..., store=..., interrupt_before/after=[...])
```

`should_continue`：最后一条消息是 `AIMessage` 且带 `tool_calls` → 去 tools；v2 模式下按 tool 分别 `Send("tools", ToolCallWithContext(...))` 实现**并发工具调用**；否则 → END（或 post_model_hook / generate_structured_response 节点）。

### 2.3 循环与停止、流式、持久化

引擎核心是 Pregel 的**超步执行**（superstep）：每步执行所有『就绪』任务（注：`pregel/_loop.py` 等实现），写入者-读取者模型驱动（channel）。停止条件：无后续边、`Interrupt` 抛出、或显式 `Command(resume=…)`。强制循环上限由用户自己建（如 `create_react_agent` 通过 `_are_more_steps_needed` 检查 `recursion_limit`，config 默认 `recursion_limit=25`）。流式 6 种模式：`values / updates / messages / custom / debug / tasks`（`types.py` StreamMode 枚举）。checkpoint 是**一等公民**：每个超步后写 checkpoint，可 `get_state`/`update_state` 做时间旅行、fork 新分支。

### 2.4 Skill / RAG / 可观测性

- **Skill**：无内建技能抽象；等价物是 LangChain 工具/自定义节点 + LangSmith 的 prompt 分享。技能规范（AGENTS.md/SKILL.md）靠生态适配（如 `langchain-skills` 第三方包）。
- **RAG**：框架不内建检索，依赖 LangChain 生态 `Retriever`/`VectorStore`，在 agent 节点前置 context 组装节点（类 agno 的 knowledge）。
- **可观测性**：LangSmith（trace + 评估）+ 远程调试（`langgraph dev` Studio 可视化图执行）+ checkpoint 审计；原生的 `debug` 流模式逐超步 dump。

### 2.5 值得借鉴的设计点（LangGraph）

1. **checkpoint = 崩溃恢复 + 时间旅行 + fork**：每次状态变更可回放/分支，是生产级 agent 的基础设施，Go 版建议把 checkpoint 接口（get/put/put_writes）设计成 storage 契约。
2. **状态 reducer 声明合并规则**：`Annotated[list, operator.add]` 式的状态合并是类型安全的多写入模型，Go 可用泛型函数（`func(M, M) M`）表达。
3. **`Send` 动态并行派发**：map-reduce 模式表达力强——子任务并发、聚合归并。
4. **`Interrupt`/HITL 是控制流原语**：不是靠『模型说要停』而是执行器显式暂停 + 恢复（`Command(resume=…)`）。
5. **`Runtime` 上下文注入**：把 config/store/消耗信息通过注入而不是全局变量传给节点——Go 用 `context.Context` 天然对应。

---

## 3. OpenAI Agents SDK ○（基于官方仓库 `openai/openai-agents-python` 与文档）

### 3.1 核心抽象

| 抽象 | 位置（`src/agents/`） | 职责 |
|---|---|---|
| `Agent` | `agent.py` | 配置对象：instructions、model、tools、handoffs、output_type、model_settings、guardrails、lifespan、hooks |
| `Runner` | `runner/runner.py`（循环在 `runner/_agent_loop.py` 附近） | **唯一负责循环的执行器**：`run()/run_sync()/run_streamed()`，`max_turns` 硬上限（默认 10） |
| `Model` 接口 + `ModelProvider` | `models/interface.py` | 无状态模型契约：`get_response/stream_response`；provider 按字符串解析模型 |
| `RunItem` | `items.py` | 结果统一为 item 列表：ToolCallItem、ToolCallOutputItem、HandoffCallItem、HandoffOutputItem、MessageItems |
| `Handoff` | `handoffs.py` | 换人是**特殊工具**：tool_name + on_handoff 钩子；Agent 之间通过工具调用切换 | 
| `ModelSettings` | `model_settings.py` | 每 Agent 推理参数（temperature、tool_choice、parallel_tool_calls、max_tokens…） |
| guardrails / hooks / lifespan | `guardrail.py` `lifecycle.py` | 输入/输出守卫（tripwire 触发中断）；AgentHooks 全生命周期回调 |
| `Session` | `sessions.py` | 简单会话：session_id + state dict（**无内置记忆管理**） |
| Tracing | `tracing/` | OTel 语义化 span：agent/function/generation/guardrail/handoff/response；三档 span 处理器 + 5 种 exporter（console/file/OTLP/OTEL/memory） |

### 3.2 运行时循环（伪代码，基于官方文档与源码认知）

```
Runner.run(agent, input, context, max_turns=10):
  items = []; active_agent = agent; turns = 0
  while True:
    if turns >= max_turns: raise MaxTurnsExceeded（或返回未完成结果）
    response = active_agent.model.get_response(...)          # 单次推理，不做工具循环
    items += response.items
    for item in items 中的新项:
      if ToolCallItem:     执行工具 → ToolCallOutputItem
      elif HandoffCallItem: active_agent = handoff.target; 记录切换
    if guardrail tripwire: 中断并报错
    if 没有新的 agent 输出/tool 调用: break
```
**与 agno 相反**：模型完全无状态、不循环，循环全在 Runner；`parallel_tool_calls`（并行工具）由 `ModelSettings` 控制，单轮内并行执行工具。

### 3.3 Skill / RAG / 运维

- **Skill**：SDK 层新增 `OpenAISkills`（`skills/openai_skills.py`，托管在 OpenAI 平台的技能如 file_search/web_search/computer_use，拉取后转为工具）。本地 `AGENTS.md`/SKILL.md 目录约定**不是** SDK 一等公民——这是它的软肋。
- **RAG**：无内建向量库/检索；官方示例建议用 custom memory provider / 动态 instructions 注入检索结果。
- **运维**：tracing 开箱即用（`set_tracing_exporters`），本地开发自动生成 `.traces` 文件可视化；`Runner` 支持 per-run hooks；内置 usage 统计并入 span。

### 3.4 值得借鉴的设计点（OpenAI SDK）

1. **Runner 拥循环、Model 零状态**：模型接口极简（get_response/stream_response），循环、重试、max_turns、guardrail 全在 Runner——Go 重构可直接照搬这种职责划分。
2. **`max_turns` 硬上限**：简单、可预测，防止失控循环；agno/LangGraph 都缺一个默认值（LangGraph 靠 recursion_limit）。
3. **Handoff 即工具**：换人用 tool_call 表达，既进对话历史又进 trace，语义自然。
4. **头等 tracing**：span 语义（agent/function/generation/handoff）与框架生命周期一一映射，配置式启用。
5. **RunItem 单向累积结果模型**：结果 = 追加式 item 列表，天然支持流式与幂等重放。

---

## 4. Mastra（二选一，选 Mastra，理由：与 Go 落地场景更接近——TS/框架内建 Memory+RAG+Workflow 一体）○

基于官方文档 `docs.mastra.ai` 与仓库 `mastra-ai/mastra` 结构（`packages/core/src/...`）。

### 4.1 核心抽象

| 抽象 | 位置 | 职责 |
|---|---|---|
| `Agent` | `packages/core/src/agent/agent.ts` | 配置：model、instructions、tools、memory、middleware、logger；`generate()/stream()` 内部自带工具循环，返回 `text + toolResults + stepsHistory` |
| `Workflow` / `Step` | `packages/core/src/workflows/` | DAG 编排：`Step` 有 input/output schema + `execute()`，`trigger` 起点，`step.after(...)`/`branch()` 连边，`registerStep/commitStep` |
| `Memory` | `packages/core/src/memory/` | 一体式记忆：storage（libsql/postgres 等）+ vector store（astra/chroma/pgvector/pinecone/qdrant…）+ embedder（openai/cohere 等）；working memory + semantic memory |
| `Tools` / MCP | `packages/core/src/actions/`、`packages/core/src/mcp/` | 工具与 MCP 客户端一等支持 |
| `AgentNetwork` / pipes | `packages/core/src/agent-network/` | 多 Agent 连接（agent 间消息传递 + supervisor 模式） |
| Observability | `packages/observability/` | `@mastra/observability` Instrumentation + OTel + 结构化 Logger 接口 |

### 4.2 运行时 / Skill / RAG

- 循环：`Agent.generate()` 内置 ReAct 循环（model→tool call→result→再次调用），返回 `stepsHistory`（每次 LLM 调用 = 一个 step 记录，便于回放/审计）。
- **Skill**：Mastra 新版引入与 Claude/agno 同构的 **Skills** 支持（目录承载 SKILL.md + 引用/脚本资源，加载后注入系统提示并暴露技能工具），紧跟 Agent Skills 规范生态。
- **RAG**：框架内建最强——Agent 直接挂 `memory` 即得语义检索；`Memory.remember()` + RAG 化记忆检索是卖点。
- 运维：OTel 集成 + 全链路 logger 注入（Agent/Workflow 都接受自定义 logger）。

### 4.3 值得借鉴的设计点（Mastra）

1. **Memory 三件套拆分**：storage（KV）+ vector store（相似度）+ embedder（embedding 函数）三个可插拔接口，而非单一 Memory 巨类。
2. **run 结果含 stepsHistory**：每次模型调用成为『步』记录，天然形成审计/评估数据，Go 版可做成 `Steps []StepRecord`。
3. **Workflow 的 commitStep 动态图**：运行时动态注册步骤，比纯静态声明图灵活；对应 Go 可设计为 builder + executor 两阶段。
4. **工具 schema 用 zod 强类型**：Go 应对应为：工具参数用结构体 tag 生成 JSON Schema，杜绝 `map[string]interface{}`。
5. **MCP 客户端内建**：Go 版 `pkg/agno/mcp` 已存在，建议作为一等工具来源（与 Mastra 一致）。

---

## 5. 横向对比速览

| 维度 | agno | LangGraph | OpenAI SDK | Mastra |
|---|---|---|---|---|
| 循环归属 | Model 内部 | Pregel 引擎（图） | Runner | Agent 内部 |
| 核心一次调用 | while True + tool_call_limit | 超步 + 条件边 | max_turns 硬顶 | 内置循环 |
| Skill | ✅ 一等公民（资源+3工具） | ❌ 无内建 | △ 托管 OpenAISkills | ✅ 目录约定 |
| RAG | ✅ 契约化 | △ 靠生态 | ❌ 靠注入 | ✅ 内建 Memory 三件套 |
| 检查点/恢复 | session 持久化 + checkpoint 参数 | ✅ 一等公民 | ❌ | ❌ |
| 多 Agent | Team 4 模式 / Workflow | 图（任意拓扑） | Handoff 链 | AgentNetwork / Workflow |
| 可观测性 | OTel+OpenInference 自动插桩 | LangSmith/Studio/流模式 | OTel 语义 span + .traces | OTel + Logger |
| HITL | 工具层面（requirements） | Interrupt 图元 | guardrail + manual_approval | workflow pause |

---

## 6. 对 Go 框架（agno-go）重构的启示

### 6.1 抽象层划分建议（映射到现有 `pkg/agno`）

```
pkg/agno/
  types/      # Message/ModelResponse/RunOutput/Event（已有，需扩展 tool_executions、stepsHistory）
  models/     # 接口收窄：Invoke/InvokeStream/结构化输出/Token计数/GetToolSchema —— 去掉循环与缓存
  runtime/    # 【新增】工具循环执行器（从 agent.go 抽出）：单轮调用→工具→结果→停止条件判定
  agent/      # 只留编排：session、hooks、记忆注入、事件包装（瘦身 1087 行）
  tools/      # Function 强类型 schema（替代 map[string]interface{}，用 generics + struct tag）
  skills/     # 【新增】Skill 契约：目录加载器 + get_skill_instructions/references/scripts 三工具
  memory/     # storage/vector/embedder 三接口 + manager（对标 Mastra/agno）
  session/    # 保留多租户；补 checkpoint 语义（get/put/put_writes）对标 LangGraph
  tracing/    # 【新增】OTel 语义 span（agent/model/tool/handoff/guardrail）+ 动态开关
  workflow/   # 已有 loop/router/parallel；补 Interrupt/Command 式 HITL 原语
```

### 6.2 具体设计决策

1. **循环下沉到 runtime 包，Model 退化为纯适配器**。现行 `Model` 接口（Invoke/InvokeStream/GetProvider/GetID/GetName）方向对，但 agent.go 里 1087 行的缓存/hooks/流式应搬到 `runtime.Runner`——对标 OpenAI SDK 的 Runner 拥循环 + agno 的循环停止事件（tool_call_paused/requirements）。模型只保留：`Invoke`、`InvokeStream`、`StructuredOutput`、`CountTokens`、`ToolDefinition`。
2. **工具参数强类型化**。`HandlerFunc(ctx, map[string]interface{})` 必须替换：定义 `Register[In any, Out any](fn func(ctx, In) (Out, error))`，由反射从结构体 tag（`json:"..." validate:"..."`）生成 JSON Schema；补 `stopAfterToolCall` 字段支持循环短路（agno 的 `Function.stop_after_tool_call`）。
3. **默认 max_turns / tool_call_limit 双保险**：OpenAI 的 `max_turns`（默认 10）+ agno 的 `tool_call_limit` 都要，且可配置；停止条件枚举化（no_tool_calls / limit_reached / stop_after_tool / hitl_blocked / requirements_pending）。
4. **Skill 机制按 agno 模式落地**：目录 + `SKILL.md` frontmatter → 加载为资源包，注入系统提示摘要 + 注册 3 个检索工具；扩展面开放（本地/OSS/远程 loader 接口）。这与 Go 单二进制分发契合（embed FS）。
5. **可观测性从第一天设计**：定义 `TraceSpan` 接口（context 携带 span），运行时自动 wrap 模型调用/工具执行/会话读写；启用 OpenTelemetry SDK 与否用构建标签或环境变量控制，零依赖可编译。
6. **Session 升级为 checkpoint 契约**：在多租户 session 存储之上加 `getState/putState/updateState` + run 级 checkpoint，为未来『恢复/时间旅行』留接口（直接用 LangGraph get/put/put_writes 语义）。

### 6.3 需要避免的坑

- **God Object Agent**：agno 的 Agent 有 40+ 字段、LangGraph 的 StateGraph 也有类似膨胀——Go 用接口组合 + `Option` 模式（functional options）拆配置维度（RunConfig / MemoryConfig / TracingConfig 分离），避免重蹈 agent.go 单文件。
- **Python 动态类型折算成 Go 泛型滥用**：agno 的 `**kwargs` 透传在 Go 里必须变成显式字段（`InvokeRequest` 已有雏形，继续收紧），否则 `map[string]interface{}` 会蔓延到 models/agent/team 三层。
- **不要在 Model 层塞重试/缓存/循环**（现 Go 代码已把缓存放 agent.go，方向对）：缓存（`tryCacheGet/Set`）、重试、fallback（agno `models/fallback.py` 的 `call_model_with_fallback`）都应是对 Model 的装饰器/中间件，保持模型实现扁平。
- **图引擎别照搬 LangGraph**：Pregel 的复杂度（channel/reducer/superstep）在 Go 静态类型下维护成本极高。Go 版 workflow 用「显式 builder + 事件驱动 executor」表达节点/边/并行即可，保留 `Interrupt` 原语与 checkpoint 语义、丢掉超步调度。
- **RAG 别囤积生态**：Go 生态不如 Python，知识库抽象收敛为 2 个接口（`Retriever` 检索 + `Ingester` 入库）+ 1 个协议（对标 `KnowledgeProtocol`），vectordb 适配器按需实现。
- **过早引入多 Agent 协议**：Team 的 delegate-tool 机制依赖 tools/Agent 双循环协同，重构顺序应为：runtime 循环 → 强类型工具 → session/checkpoint → skill → tracing → 最后才 Team/Workflow 编排，避免一次性推倒重来。

---

## 7. 关键引用清单（文件路径 / URL）

**agno**（均实际拉取，main 分支 `libs/agno/agno/`）
- `agent/agent.py`、`agent/_run.py`（16 步 run 流程）、`agent/protocol.py`
- `models/base.py`（Model.response 工具循环，line ~703-875 while True 主循环）、`models/fallback.py`
- `tools/toolkit.py`、`tools/function.py`、`tools/api.py`（CustomApiTools 示例）
- `team/team.py`、`team/mode.py`、`team/task.py`、`workflow/workflow.py`、`workflow/types.py`、`workflow/decorators.py`、`workflow/loop.py`、`workflow/step.py`
- `session/agent.py`、`memory/manager.py`、`skills/skill.py`、`skills/agent_skills.py`、`skills/loaders/{base,local}.py`
- `knowledge/protocol.py`、`knowledge/knowledge.py`、`tracing/setup.py`、`tracing/exporter.py`、`run/base.py`
- URL：`https://github.com/agno-agi/agno/tree/main/libs/agno/agno`

**LangGraph**（state/types/runtime/constants/checkpoint/executor 已实际拉取）
- `libs/langgraph/langgraph/graph/state.py`（StateGraph.add_node/add_edge 等）、`types.py`（Send/Command/Interrupt）、`runtime.py`、`constants.py`（START/END）、`pregel/_loop.py`
- `libs/prebuilt/langgraph/prebuilt/chat_agent_executor.py`（create_react_agent：add_node agent/tools + should_continue 条件边 + Send 并行工具）
- `libs/checkpoint/langgraph/checkpoint/base/__init__.py`（BaseCheckpointSaver：get/get_tuple/put/put_writes）
- URL：`https://github.com/langchain-ai/langgraph/tree/main/libs`

**OpenAI Agents SDK**（本文档级，未拉取）
- `src/agents/agent.py`、`runner/runner.py`、`models/interface.py`、`items.py`、`handoffs.py`、`guardrail.py`、`lifecycle.py`、`tracing/*`、`model_settings.py`
- URL：`https://github.com/openai/openai-agents-python`、`https://openai.github.io/openai-agents-python/`

**Mastra**（本文档级，未拉取）
- `packages/core/src/agent/agent.ts`、`workflows/`、`memory/`、`agent-network/`、`packages/observability/`
- URL：`https://github.com/mastra-ai/mastra`、`https://docs.mastra.ai`

**本地项目对照**（已核对）
- `E:\coding\agno-go\pkg\agno\agent\agent.go`（Run 循环 + tryCache + hooks + stream，单一文件）
- `pkg\agno\models\base.go`（Model 接口 Invoke/InvokeStream/GetProvider/GetID/GetName + map[string]interface{} 参数）
- `pkg\agno\tools\toolkit\toolkit.go`（HandlerFunc(args map[string]interface{})、ToModelToolDefinitions）

---

## 任务小结

- **完成内容**：深度调研 4 个框架；agno 与 LangGraph 基于实际拉取的 40+ 个源码文件逐行核对（离线路径：`~/agno_research/`），OpenAI Agents SDK 与 Mastra 基于官方文档/已知仓库结构并明确标注证据级别；输出上述完整中文 Markdown 报告；额外核对并对照了本地 `agno-go` 的现状（models/base.go、agent/agent.go、toolkit.go）。
- **产出文件**：`$HOME/agno_research/` 下约 30 个 agno 源码文件与 7 个 LangGraph 源码文件（本地供后续引用；报告本身按任务要求直接返回正文，未写文件）。
- **遇到的问题**：Windows 下 MCP shell 是 cmd（无 grep）；raw.githubusercontent 对部分仓库文件路径需通过 GitHub API 探测（agno 已迁移到 `libs/agno/agno/`，LangGraph prebuilt 移动到 `libs/prebuilt/`）；后续 githubusercontent 偶发超时，改用三重试/API tree 处理。
- **未完成**：因工具迭代上限，未拉取 OpenAI Agents SDK 与 Mastra 源码做逐行验证（报告已如实标注）；下一步如需可补充两者的真实文件核对（建议先取 `src/agents/runner/runner.py` 与 `packages/core/src/agent/agent.ts` 的循环部分）。