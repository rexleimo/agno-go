# Agent Skill 机制与运维层业界标准调研报告

> 调研目的：为 agno-go（Go 多 Agent 框架）的框架级本地 Skill 机制（SKILL.md + 资源文件、渐进式披露）与运维层（OTel 追踪、评估、生产可靠性）设计提供业界依据。
> 调研日期：2026-08-03。所有引用均为官方文档/规范原文（含 URL），未编造。OTel GenAI 语义约定状态为 Development（未冻结）。

---

## 1. Claude Agent Skills 规范（Anthropic 官方）

### 1.1 概述与演进

- **开放标准**：Agent Skills 已于 2025-12-18 由 Anthropic 发布为**开放标准**，见 https://agentskills.io/specification （跨平台可移植，不锁定单一厂商）。
- **官方实现仓库**：https://github.com/anthropics/skills —— 内含 spec 目录、模板（template/SKILL.md）、17 个示例技能（algorithmic-art、docx/pdf/pptx/xlsx 文档技能、mcp-builder、webapp-testing 等），以及 `skill-creator`（让 Claude 帮你写技能的技能）。
- **工程博客**（权威解释渐进式披露设计动机）：https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills
- 支持面：Claude.ai、Claude Code、Claude Agent SDK、Claude Developer Platform（API）均支持。

### 1.2 目录结构（规范原文）

```
skill-name/
├── SKILL.md          # Required: metadata + instructions
├── scripts/          # Optional: executable code
├── references/       # Optional: documentation
├── assets/           # Optional: templates, resources
└── ...               # Any additional files or directories
```

- 一个技能 = 一个目录，最少包含 `SKILL.md`。
- `references/` 建议文件：`REFERENCE.md`、`FORMS.md`、领域文件（finance.md 等），**按需加载，文件越小越省上下文**。
- 文件引用：SKILL.md 内用**相对路径**引用同技能目录下文件（如 `scripts/extract.py`），建议**只保持一层深**，避免深嵌套引用链。

### 1.3 SKILL.md frontmatter 字段（规范字段表）

| 字段 | 必填 | 约束（原文） |
|---|---|---|
| `name` | 是 | ≤64 字符；仅小写字母、数字、连字符；不能以连字符开头/结尾；不能有连续连字符 `--`；**必须与父目录名一致** |
| `description` | 是 | ≤1024 字符，非空；说明技能做什么 + 何时用；应包含帮助 agent 识别任务的**关键词** |
| `license` | 否 | 许可证名或指向技能内 LICENSE 文件的引用 |
| `compatibility` | 否 | ≤500 字符；环境要求（目标产品、系统包、网络访问等） |
| `metadata` | 否 | 任意 string→string 键值对（作者、版本等），键名建议唯一避免冲突 |
| `allowed-tools` | 否 | 空格分隔的预授权工具串（实验性，各实现支持度不一），如 `Bash(git:*) Bash(jq:*) Read` |

最小示例：
```markdown
---
name: skill-name
description: A description of what this skill does and when to use it.
---
```

### 1.4 渐进式披露（Progressive Disclosure）—— 核心设计原则

规范明确三级加载（token 预算原文）：

1. **Metadata（~100 tokens）**：启动时所有已安装技能的 `name` + `description` 预载入 system prompt（即"目录"）。博客原文："At startup, the agent pre-loads the name and description of every installed skill into its system prompt."
2. **Instructions（<5000 tokens 建议；SKILL.md 建议 <500 行）**：技能被激活时，完整 SKILL.md body 载入上下文。
3. **Resources（按需）**：scripts/、references/、assets/ 中的文件仅在需要时读取（agent 自行决定导航读取）。

正文（body）无格式限制，规范建议包含：分步指令、输入/输出示例、常见边界情况。注意"agent 一旦决定激活技能就会加载整个文件"，所以长内容应拆到 references/。

### 1.5 验证工具

- 规范提到 `skills-ref` 参考库：`skills-ref validate ./my-skill` 校验 frontmatter 合法性与命名约定（仓库地址未在规范页给出，未验证到 GitHub 地址，勿臆造）。

### 1.6 Claude Code 中的加载与执行机制

来源：https://code.claude.com/docs/en/skills

**发现与存放位置**（多层，优先级 enterprise > personal > project > bundled）：
- Enterprise：managed settings（组织级）
- Personal：`~/.claude/skills/<skill-name>/SKILL.md`（所有项目可用）
- Project：`.claude/skills/<skill-name>/SKILL.md`（当前项目）
- Plugin：`<plugin>/skills/<skill-name>/SKILL.md`（命名空间 `plugin-name:skill-name`）
- 嵌套目录：子目录 `.claude/skills/` 在 Claude 读写该子目录文件时才加载，重名技能用目录限定名 `apps/web:deploy`。
- 热更新：监控技能目录，会话内增删改即时生效；`--add-dir` 目录内的 `.claude/skills/` 也会加载。
- 兼容：`.claude/commands/deploy.md` 与 `.claude/skills/deploy/SKILL.md` 等价（自定义命令已并入技能体系）。

**frontmatter 扩展字段**（Claude Code 在开放标准之上的扩展）：
`name`（显示名）、`description`（推荐）、`when_to_use`（触发场景补充，与 description 合并后截断 1536 字符）、`argument-hint`、`arguments`（命名参数，`$name` 替换）、`disable-model-invocation`（禁止模型自动加载，仅手动 `/name`）、`user-invocable: false`（隐藏于 `/` 菜单，仅模型可用）、`allowed-tools` / `disallowed-tools`（技能激活当轮的工具授权/禁用）、`model`、`effort`、`context: fork`（子代理中运行）、`agent`（fork 用的子代理类型）、`background`、`hooks`、`paths`（glob 限定自动激活的文件范围）、`shell`（bash/powershell）。

**调用与注入机制**：
- 用户 `/skill-name` 直接调用，或模型根据 description 自动加载；技能对模型暴露为 **Skill tool**，可用权限规则 `Skill(name)` / `Skill(name *)` 精确允许/拒绝。
- **上下文生命周期**：渲染后的 SKILL.md 内容以**单条消息**进入对话并**持续到会话结束**（不是每轮重读）；重复调用且内容相同则只追加"已加载"提示；auto-compaction 后重挂最近调用的技能（每个技能前 5000 tokens，合计 25,000 tokens 预算）。
- **动态上下文注入（dynamic context injection）**：`!`command`` 行内语法与 ```! fenced 块 —— 在技能内容送进模型**之前**由 Claude Code 执行 shell 命令并把输出内联，模型只看到结果（预处理而非模型执行）。可禁用（`disableSkillShellExecution` 设置）。
- **字符串替换**：`$ARGUMENTS`、`$ARGUMENTS[N]`/`$N`、`$name`（arguments 声明）、`${CLAUDE_SKILL_DIR}`、`${CLAUDE_PROJECT_DIR}`、`${CLAUDE_SESSION_ID}`、`${CLAUDE_EFFORT}`。
- **子代理执行**：`context: fork` 时技能内容作为子代理的 prompt（agent: Explore/Plan/general-purpose 或自定义 `.claude/agents/` 子代理），子代理无主对话历史，结果回传。
- 其他：`skillOverrides` 设置（on / name-only / user-invocable-only / off）、符号链接支持、`.claude-plugin/plugin.json` 可让技能目录升级为插件（捆绑 agents/hooks/MCP servers）。

### 1.7 Claude API 中的执行机制（远程托管技能）

来源：https://docs.claude.com/en/api/skills-guide（重定向至 https://platform.claude.com/docs/en/build-with-claude/skills-guide）

注意：这是 **Anthropic 托管的远程技能**（与我们要做的本地技能包机制不同，但字段/约束值得对齐）：

- 通过 Messages API 的 **`container` 参数** + **`code_execution` 工具**执行：
  ```json
  container: { "skills": [{"type": "anthropic|custom", "skill_id": "pptx", "version": "latest"}] }
  tools: [{"type": "code_execution_20250825", "name": "code_execution"}]
  ```
- Beta 头：`code-execution-2025-08-25`（必需）、`skills-2025-10-02`（Skills API）、`files-api-2025-04-14`（文件上传下载）。
- 每次请求最多 **8 个技能**；技能在代码执行容器内运行，产出的文件通过响应中的 `file_id` + Files API 下载。
- 多轮对话复用 `container.id`；长任务可能返回 `stop_reason: "pause_turn"`，需把响应原样回传继续。
- 自定义技能上传约束：zip 顶层目录名必须与 frontmatter `name` 匹配（大小写/下划线不敏感，如 `Financial_Skill` 匹配 `financial-skill`）；总大小 <30MB；`name` ≤64 字符、小写+数字+连字符、不含 XML 标签、**不能用保留词 "anthropic"/"claude"**；`description` ≤1024 字符。
- 版本管理：Anthropic 技能用日期版本（`20251013`），自定义技能用 epoch 时间戳（`1759178010641129`），均支持 `latest`。

---

## 2. MCP 与 Skills 的关系和互补性

### 2.1 MCP 是什么

- 规范：https://modelcontextprotocol.io/specification/2025-06-18 ；架构：https://modelcontextprotocol.io/docs/learn/architecture
- 开放协议，JSON-RPC 2.0，三角角色：**Host**（LLM 应用，发起连接）、**Client**（宿主内的连接器）、**Server**（提供上下文与能力的服务）。
- Server 功能：**Resources**（数据/上下文）、**Prompts**（模板化消息与工作流）、**Tools**（模型可执行的函数）；Client 功能：**Sampling**（服务端发起的递归 LLM 调用）、**Roots**（文件系统/URI 边界）、**Elicitation**（向用户要信息）；工具类：进度、取消、错误报告、日志。
- 安全原则（规范原文）：用户同意与控制、数据隐私、工具安全（工具 = 任意代码执行，描述不可信）、LLM sampling 控制。

### 2.2 互补关系（业界共识）

- Anthropic 博客原文："We'll also explore how Skills can complement MCP servers by **teaching agents more complex workflows that involve external tools and software**."
- 定位差异：
  - **MCP = 运行时的能力通道协议**：把"能做什么"（工具/资源/提示模板）标准化地暴露给 agent，解决连接与互操作。
  - **Skills = 静态指令包**：把"怎么做好"（程序性知识、组织上下文、分步流程、脚本与参考文档）封装为可发现、按需加载的资源。
- 互补模式：一个 Skill 可以教 agent 如何编排一组 MCP 工具完成复杂工作流（如 mcp-builder 技能指导如何构建/测试 MCP server）；Claude Code 中插件可同时捆绑 skills 与 MCP servers。
- 对框架而言两者正交：Skills 是**内容/指令层**，MCP 是**工具/连接层**；Skill 机制设计时不应替代 MCP，而应允许技能"引用/编排"MCP 工具（如 frontmatter 的 `allowed-tools` 可放开到 MCP 工具名）。

---

## 3. 主流框架的 Skill 实现对比

| 框架 | 载体格式 | 暴露给 LLM 的方式 | 加载时机 | 权限/控制 |
|---|---|---|---|---|
| **Claude Code** | `SKILL.md`（YAML frontmatter + Markdown）+ scripts/references/assets | **渐进式披露**：目录（name+description）常驻上下文；body 激活时作为单条消息注入；文件按需读取；另有 Skill tool 与 `!`cmd`` 动态注入 | 启动载入目录；激活时载 body；会话内持久 | `disable-model-invocation`、`user-invocable`、`allowed-tools`、权限规则 `Skill(name)`、`skillOverrides`、`context: fork` |
| **Claude API（远程）** | 同上 + 托管 zip 上传 | container 参数把技能放进代码执行沙箱，模型通过 `code_execution` 工具驱动 | 每请求指定（≤8 个），沙箱内多轮复用 | 工作区私有、版本固定、30MB 限制、保留词校验 |
| **OpenAI Agents SDK** | 无技能文件格式；`Agent(instructions=...)` + function tools + MCP tools + handoffs + guardrails | **全量 system prompt 注入**（instructions）+ **工具调用**（tools 全部暴露）；无渐进式披露 | 每轮完整发送（SDK 做上下文管理） | `RunConfig`、guardrails、tool 级校验（Pydantic） |
| **agno** | Python 函数 / `Toolkit` 类 / `MCPTools` / `knowledge`（向量库 RAG）/ `instructions` / Context Providers | 工具 schema 注入 + instructions 注入；knowledge 通过检索（RAG）按需注入上下文 | 工具定义常驻；knowledge 检索时注入 | `tool_call_limit`、`requires_confirmation`（HITL）、`include_tools`/`exclude_tools`、`external_execution` |
| **Cursor Rules** | `.mdc` 文件（frontmatter：`description`/`globs`/`alwaysApply`）；另有 AGENTS.md | **前缀注入**：规则内容追加到模型上下文**起始位置**；四种模式：Always Apply（全注入）/ Apply Intelligently（agent 按 description 判断）/ Apply to Specific Files（globs 匹配时自动附加）/ Apply Manually（@-mention） | Always 每会话；globs 按文件匹配；其余按需 | 团队规则（Team > Project > User 优先级）、强制执行开关；建议 <500 行 |
| **Agent Skills 开放标准** | `SKILL.md` 目录包 | 渐进式披露（见 1.4） | 三级按需 | `allowed-tools`（实验性） |

要点归纳（回答"如何把技能暴露给 LLM"）：
- **system prompt 注入**：OpenAI Agents SDK instructions、Cursor rules（alwaysApply）、agno instructions —— 简单但全量常驻，成本随技能数线性增长。
- **工具调用**：OpenAI function tools、agno tools/toolkits、MCP —— 能力以 JSON schema 暴露，模型自主选择；但 schema 也占上下文，且不适合长指令。
- **渐进式披露**：Claude/Agent Skills 标准 —— 目录常驻（~100 tokens/技能）+ body 按需加载 + 文件按需读取；Cursor 的 description 判断（Apply Intelligently）也是轻量版。**业界趋势是渐进式披露**（Anthropic 博客、Claude Code 文档均强调"skill body loads only when used, long reference material costs almost nothing until you need it"）。

---

## 4. Agent 可观测性：OpenTelemetry GenAI 语义约定与实现

### 4.1 OTel GenAI Semantic Conventions（现状与权威来源）

- **重要变更**：GenAI 语义约定已从 opentelemetry.io 主站迁出，独立仓库：https://github.com/open-telemetry/semantic-conventions-genai （旧页 https://opentelemetry.io/docs/specs/semconv/gen-ai/ 仅剩迁移说明）。当前状态 **Development**（未冻结，属性可能变更）。
- 核心文档：`docs/gen-ai/gen-ai-spans.md`（LLM 调用）、`gen-ai-agent-spans.md`（agent/框架 span）、`gen-ai-events.md`、`gen-ai-metrics.md`、`mcp.md`、`anthropic.md`、`openai.md`；属性注册表：`docs/registry/attributes/gen-ai.md`。

**`gen_ai.operation.name` 预定义值**（MUST 优先用）：`chat`、`text_completion`、`generate_content`、`embeddings`、`retrieval`、`create_agent`、`invoke_agent`、`invoke_workflow`、`plan`、`execute_tool`、`create_memory`、`create_memory_store`、`delete_memory`、`delete_memory_store`、`search_memory`、`upsert_memory`、`update_memory`、`fetch_response`。

**Agent 相关 span 族**（gen-ai-agent-spans.md）：
- `create_agent` span：span name `create_agent {gen_ai.agent.name}`，kind=CLIENT（远程 agent 服务创建）。
- `invoke_agent` **client** span：远程调用（OpenAI Assistants API、AWS Bedrock Agents），kind=CLIENT。
- `invoke_agent` **internal** span：**同进程本地框架 agent（如 LangChain/CrewAI）——我们的 Go 框架应对齐这个**，kind=INTERNAL，span name `invoke_agent {gen_ai.agent.name}`。
- `invoke_workflow` span、`plan` span（规划/任务分解阶段；生成计划的 LLM 调用应为其子 span，工具 span 为同级）。
- `execute_tool` span：kind=INTERNAL，span name `execute_tool {gen_ai.tool.name}`；必填 `gen_ai.tool.name`；推荐 `gen_ai.tool.call.id`、`gen_ai.tool.description`、`gen_ai.tool.type`（function/extension/datastore）；Opt-In `gen_ai.tool.call.arguments`、`gen_ai.tool.call.result`。

**关键属性**（registry）：
- `gen_ai.provider.name`（必填，判别 provider 遥测格式）：well-known 值包括 `openai`、`anthropic`、`aws.bedrock`、`azure.ai.openai`、`azure.ai.inference`、`gcp.vertex_ai`、`gcp.gemini`、`gcp.gen_ai`、`cohere`、`mistral_ai`、`deepseek`、`moonshot_ai`、`x_ai`、`groq`、`perplexity`、`ibm.watsonx.ai`。
- `gen_ai.agent.name` / `gen_ai.agent.id` / `gen_ai.agent.description` / `gen_ai.agent.version`。
- `gen_ai.request.model`、`gen_ai.request.max_tokens`、`temperature`、`top_p`、`frequency_penalty`、`presence_penalty`、`stop_sequences`、`seed`、`choice.count`。
- `gen_ai.response.finish_reasons`。
- **Token 用量**：`gen_ai.usage.input_tokens`、`gen_ai.usage.output_tokens`、`gen_ai.usage.reasoning.output_tokens`、`gen_ai.usage.cache_creation.input_tokens`、`gen_ai.usage.cache_read.input_tokens`（cache 值应包含在 input_tokens 内）。
- `gen_ai.conversation.id`（会话/线程 ID，无则**不要**用随机 UUID 兜底）、`gen_ai.data_source.id`。
- Opt-In（含敏感数据，需采样/脱敏策略）：`gen_ai.system_instructions`（有 JSON schema 约束）、`gen_ai.input.messages`、`gen_ai.output.messages`、`gen_ai.tool.definitions`。
- 采样决策属性应在 span 创建时设置：`gen_ai.agent.name`、`gen_ai.operation.name`、`gen_ai.provider.name`、`gen_ai.request.model`、`server.address/port`。

**Metrics**：`gen_ai.client.operation.duration`（Histogram, s）、`gen_ai.client.operation.time_to_first_chunk`、`gen_ai.client.operation.time_per_output_chunk`。

**MCP semconv**（docs/gen-ai/mcp.md）：`mcp.client` / `mcp.server` span；metrics `mcp.client.operation.duration`、`mcp.server.operation.duration`、`mcp.client.session.duration`、`mcp.server.session.duration`；属性含 `mcp.request.id` 等。

### 4.2 Langfuse（OTel 后端 + 观测平台）

来源：https://langfuse.com/integrations/native/opentelemetry
- **OTLP 端点**：`/api/public/otel`（trace 专用 `/api/public/otel/v1/traces`），Basic Auth（base64(公钥:私钥)），支持 OTLP/HTTP protobuf 与 JSON；v4 需带请求头 `x-langfuse-ingestion-version: 4` 实时入库（否则延迟可达 10 分钟）。
- **v4 SDK 是 OTel 薄封装**：自动把 span 转换为 Langfuse observation（span/generation/event），默认只关注 LLM 相关 span（自身 span、含 `gen_ai.*` 属性的 span、已知 LLM instrumentor 的 span），可用自定义 filter 放开。
- **属性传播要求**：跨 span 过滤/聚合需要把 trace 级属性**传播到每个 span**，推荐 OTel Baggage + BaggageSpanProcessor；命名：`langfuse.user.id`、`langfuse.session.id`、`langfuse.trace.metadata.*`、`langfuse.version`、`langfuse.release`、`langfuse.trace.tags`、`langfuse.trace.name`（注意：baggage 会跨服务传播，勿放敏感信息）。
- **属性映射**：Langfuse 把 OTel GenAI 属性映射到其数据模型，语义约定仍在演进，映射可能不完整（文档明示"attribute mapping"需社区贡献）。
- 支持 OpenLLMetry / OpenLIT / Arize 等 OTel 插桩库直接导出；也支持 OTel Collector 转发（配置示例：receivers.otlp → processors [memory_limiter, batch] → exporters.otlphttp/langfuse）。

### 4.3 OpenLLMetry / Traceloop

来源：https://www.traceloop.com/docs/openllmetry/introduction
- OpenLLMetry = Traceloop 的开源 OTel 自动插桩层，提供 Python / JavaScript / TypeScript / **Go** SDK；自动捕获 LLM 调用、embedding、向量库、agent/workflow/tool 执行，输出标准 OTLP，可接入 Langfuse、Dynatrace、Datadog 等任意 OTel 后端。Go 语言是其显式支持对象（对 agno-go 是现成参考，但本项目已自带 otel SDK，倾向直接手写 instrument）。

### 4.4 OpenAI Agents SDK 内置 tracing（可参考的 span 层级设计）

来源：https://openai.github.io/openai-agents-python/tracing/
- 概念：**Trace**（一次端到端 workflow；属性 workflow_name、trace_id 格式 `trace_<32位字母数字>`、group_id 关联同一会话多 trace、metadata、disabled）+ **Span**（started_at/ended_at、trace_id、parent_id、span_data）。
- 默认 span 层级（自上而下）：`trace()` 包住整个 Runner → `task_span()`（每次 runner 调用）→ `turn_span()`（每次模型轮次）→ `agent_span()`（每个 agent 运行）→ `generation_span()`（每次 LLM 生成）；工具调用 `function_span()`；`guardrail_span()`、`handoff_span()`、语音 `transcription_span()`/`speech_span()`。可用 `RunConfig(tracing={"include_task_and_turn_spans": False})` 压缩层级。
- 生命周期：contextvar 跟踪当前 trace/span（并发安全）；`BatchTraceProcessor` + `BackendSpanExporter` 批量导出；`flush_traces()` 强制立即导出（长驻 worker）；`add_trace_processor()` / `set_trace_processors()` 自定义导出；`OPENAI_AGENTS_DISABLE_TRACING=1` 全局关闭；`trace_include_sensitive_data` 控制是否记录 generation/function 输入输出（敏感数据开关）。
- 生态：Langfuse、Arize Phoenix、W&B、Braintrust、LangSmith、Datadog 等 25+ 厂商实现了自定义 trace processor。

### 4.5 agno AgentOS tracing

来源：https://docs.agno.com/agent-os/tracing/overview
- `tracing=True` 开启；基于 **OpenTelemetry 插桩**，trace 存入 agno 数据库（AgentOS Control Plane 或 API 查看）。
- 记录的 span 类型：agent/team/workflow 运行（输入、输出、状态、时长、子操作）、**model calls**、**tool calls**（工具名、耗时、结果、错误）、team 协调调用、workflow 步骤。
- `setup_tracing(db, batch_processing=True, max_queue_size=2048, max_export_batch_size=512, schedule_delay_millis=3000)` 控制批量导出。

### 4.6 给 agno-go 的 span 设计建议（依据 4.1 规范）

```
invoke_agent {agent.name}        (INTERNAL; gen_ai.operation.name=invoke_agent)
├── plan {agent.name}            (INTERNAL; 规划阶段，可选)
├── execute_tool {tool.name}     (INTERNAL; 每次工具调用, gen_ai.tool.call.id)
│   └── (MCP 调用时嵌套 mcp.client span)
├── chat {model}                 (CLIENT; 每次 LLM 调用, gen_ai.usage.*)
└── retrieval / search_memory    (如启用 knowledge)
```
- 所有 span 创建时即设置 `gen_ai.agent.name`、`gen_ai.operation.name`、`gen_ai.provider.name`、`gen_ai.request.model`（采样决策属性）。
- token 用量从模型响应 Usage 映射到 `gen_ai.usage.*`；成本监控用 cache_read/cache_creation 区分缓存命中。
- 事件（gen-ai-events.md）与错误（`error.type`，well-known `_OTHER`）按规范补全。

---

## 5. Agent 评估：LLM-as-judge、Evals 框架与黄金数据集

### 5.1 LLM-as-judge（业界标准做法）

- 用一个（或多个）LLM 按**准则（criteria）**给被测 agent 输出打分；核心参数化元素（agno 的 AgentAsJudgeEval 具代表性）：
  - `criteria`（评估准则，必填，否则 judge prompt 无约束）
  - `scoring_strategy`：`numeric`（1-10）或 `binary`（pass/fail）；`threshold` 及格线
  - `evaluator_agent`：可自定义 judge 的 instructions/模型（如"严格评审员"）
  - 批量 `cases`（input/output 对）、`on_fail` 回调、结果持久化到 DB、telemetry
- 局限与对策：judge 偏差（自评分、位置偏差、冗长偏好）→ 用不同模型做 judge、多次迭代取均值（agno `num_iterations=3`）。

### 5.2 OpenAI Evals

来源：https://github.com/openai/evals ；Dashboard 版：https://platform.openai.com/docs/guides/evals
- 结构：`evals/registry/data/`（数据集，Git-LFS）+ `evals/registry/evals/*.yaml`（eval 定义：类型、参数、数据集）。
- 类型：**model-graded evals**（模型按 rubric 打分，如 fact/classification 模板）、**basic evals**（确定性/编辑距离/模糊匹配）、**自定义 completion functions**（`docs/completion-fns.md` Completion Function Protocol，支持 prompt chains 与 tool-using agents）。
- 关键文档：`docs/build-eval.md`（构建流程）、`docs/eval-templates.md`、`docs/run-evals.md`；可接 W&B/Snowflake 记录结果。

### 5.3 DeepEval

来源：https://github.com/confident-ai/deepeval
- "pytest 风格的 LLM 单测框架"，指标可基于 **LLM-as-judge**（如 G-Eval、DAG 图式确定性 judge 构建器）或本地统计/NLP 模型。
- **Agentic 指标**（对本项目最相关）：Task Completion（是否达成目标）、Tool Correctness（是否调用正确工具+正确参数）、Goal Accuracy、Step Efficiency（是否多余步骤）、Plan Adherence（是否按计划执行）、Plan Quality、Tool Use、Argument Correctness。
- RAG 指标：Answer Relevancy、Faithfulness、Contextual Precision/Recall/Relevancy、RAGAS。
- 多轮指标：Knowledge Retention、Conversation Completeness、Turn Relevancy、Turn Faithfulness、Role Adherence。

### 5.4 agno 的 eval（与本项目同名，直接对标）

来源：https://docs.agno.com/evals/accuracy/overview 、https://docs.agno.com/evals/agent-as-judge/overview 、https://docs.agno.com/evals/suite/overview
- **AccuracyEval**：`expected_output` 黄金答案 + LLM judge 打分；`num_iterations` 多次取 `avg_score`；`run_with_output()` 只评给定输出不跑 agent；支持 tools/teams/异步 `arun()`；结果入库（SqliteDb/PostgresDb）并经 AgentOS 暴露 `GET /eval-runs`（过滤 `?agent_id=` `?eval_types=accuracy` 等）。
- **AgentAsJudgeEval**：自定义 criteria 的 LLM-as-judge（见 5.1）。
- **Eval Suite**：把多个 `Case`（含 judge_mode/judge_threshold）组成套件在 CI 门禁运行。
- 模式：eval 可以作为 agent 的 **post-hook** 自动在每次运行后评估。

### 5.5 黄金数据集回归测试做法（业界综合）

1. **数据集**：收集真实/代表性 input → expected output（黄金答案）对；按任务类型分层（简单/复杂/边界）；版本化管理（git）。
2. **判分器分层**：
   - 确定性：子串/正则/编辑距离/JSON schema 校验（agno-go 现有 `Scenario.ExpectedContains` 即此层，成本为零）；
   - 语义相似：embedding cosine 阈值；
   - LLM-as-judge：rubric 打分（覆盖确定性判不了的开放性任务）；
   - 工具正确性：校验 tool call 序列（DeepEval Tool Correctness 思路）。
3. **回归门禁**：CI 中对每次改动跑套件，与基线对比（成功率、avg_score、token 成本、延迟），超过阈值即失败；用 `seed`/固定模型版本保证可比性。
4. **结果追踪**：eval runs 持久化 + 时间线对比（agno eval-runs、Langfuse datasets/experiments、OpenAI Evals + W&B）。

---

## 6. 生产级 Agent 运维 Checklist

> 综合业界文档与工程实践；标注出处者为已验证引用，其余为通用工程建议。

**超时（Timeouts）**
- [ ] 单次 LLM 调用超时（上下文 deadline，如 30s-120s 按模型/任务分级）——agno-go 的 claude client 已有 `defaultTimeout=30s` 模式可推广到所有 provider。
- [ ] 整个 run 的总体截止时间（多轮 agent loop 必须外层 deadline，防止死循环；配合 max_turns/`tool_call_limit`）。
- [ ] 流式场景用首包时间（TTFT）与每包间隔超时（对应 OTel `time_to_first_chunk`/`time_per_output_chunk` 指标）。

**重试（Retries）**
- [ ] 仅对幂等/可安全重复的操作重试；指数退避 + 抖动（jitter）。
- [ ] 对 HTTP 429/5xx/网络错误重试；4xx 业务错误不重试。
- [ ] 工具副作用操作：先做幂等键（idempotency key）再允许重试；重试计数与预算纳入 metrics。

**限流（Rate Limiting）**
- [ ] 客户端令牌桶/滑动窗口限流（按 API key、按用户/租户、按模型），防止触发 provider 429 雪崩。
- [ ] 队列与并发上限（控制同时 in-flight 的 LLM 请求与工具执行）。
- [ ] provider 返回 `Retry-After` 时遵守并记录。

**成本控制（Cost）**
- [ ] 每请求 `max_tokens` 上限；reasoning 模型单独预算（`gen_ai.usage.reasoning.output_tokens`）。
- [ ] 按 run/用户/天记录成本（用 `gen_ai.usage.*` + 单价表，Langfuse 有内置 token & cost tracking）。
- [ ] 模型路由：简单任务用小模型，复杂任务才用大模型；长上下文任务用缓存。
- [ ] 上下文预算：渐进式披露技能、压缩（compaction）、控制注入内容体积（Claude Code 文档明示"every line is a recurring token cost"）。

**缓存（Caching）**
- [ ] **Prompt caching**：稳定 system prompt 前缀（技能目录、工具定义）用 provider 缓存（Anthropic `cache_control`；指标 `gen_ai.usage.cache_read.input_tokens` 监控命中率）。
- [ ] 响应缓存：确定性查询（工具结果、检索结果）做语义缓存/TTL 缓存。
- [ ] 技能内容去重：同一技能重复激活不重复注入（Claude Code 的做法），避免上下文膨胀。

**熔断（Circuit Breaker）**
- [ ] 对 provider/工具/MCP server 做熔断（连续错误率/错误数阈值 → 快速失败 → 半开探测）。
- [ ] 降级与 fallback：主模型失败切备模型（Anthropic 官方有 stop-reasons/fallback 机制文档：https://docs.claude.com/en/docs/build-with-claude/stop-reasons-fallback ，以及 fallback credit 概念）；工具失败时给模型可执行的备选路径。

**密钥管理（Secrets）**
- [ ] API key 走环境变量/secret manager，禁止硬编码与进日志（agno-go `.env.example` 模式 + `pkg/agentos` HTTP 服务需注意不回显密钥）。
- [ ] 追踪脱敏：敏感数据默认不落 span（OpenAI Agents SDK `trace_include_sensitive_data=false`、Langfuse masking）；baggage 不携带敏感信息。
- [ ] 技能/插件可信审计：技能可自授权工具（`allowed-tools`），项目技能合入前需审查（Claude Code 文档明确警告"a skill can grant itself broad tool access"）。

**其他**
- [ ] 可观测性：OTLP 导出 + 采样策略 + `error.type` 标准化 + 告警（成功率、TTFT、成本异常、熔断触发）。
- [ ] 审计：会话/输入输出留痕（DB 持久化，agno 的 session 表模式）。
- [ ] 评估门禁：golden 数据集回归进 CI（见第 5 节）。

---

## 7. Go 框架 Skill 机制设计建议（agno-go 落地）

### 7.1 设计目标与原则

- 对齐 **Agent Skills 开放标准**（https://agentskills.io/specification）：本地目录技能包，`SKILL.md` + scripts/references/assets；frontmatter 只认规范字段（可扩展 `metadata`）。
- **渐进式披露**是核心：目录常驻 system prompt，body 按需加载，文件按需读取 —— 保证"技能数增长不线性吃上下文"。
- 与现有代码共存：`pkg/agno/tools/claude/skills.go` 是**远程 Anthropic Skills API 工具**（`invoke_claude_skill`，POST /v1/agent-skills/messages），属于 toolkit 层，**保留不动**；新建本地技能机制为独立包，二者语义不同（远程托管执行 vs 本地指令包）。

### 7.2 包结构与接口

新建 `pkg/agno/skills/`：

```go
// 元数据（渐进式披露第一级，~100 tokens/技能）
type Metadata struct {
    Name          string            // 校验: ^[a-z0-9]+(-[a-z0-9]+)*$，≤64，与目录名一致
    Description   string            // ≤1024，非空，"做什么+何时用"
    License       string            // 可选
    Compatibility string            // 可选 ≤500
    Metadata      map[string]string // 可选
    AllowedTools  []string          // 可选（实验性）
}

// 完整技能（第二、三级）
type Skill struct {
    Metadata
    Dir    string   // 技能根目录
    Body   string   // SKILL.md frontmatter 之后的正文
    Files  []string // 相对路径资源清单（scripts/references/assets/...）
    Size   int64    // 体积上限校验用
}

type Loader interface {
    List(ctx context.Context) ([]Metadata, error)   // 扫描所有技能目录（仅 frontmatter）
    Load(ctx context.Context, name string) (*Skill, error) // 解析并读取 SKILL.md + 文件清单
}

// 内置实现
func NewFSLoader(fsys fs.FS) Loader          // os.DirFS / embed.FS / go:embed 技能包
func NewGitLoader(repo, ref, subdir string)  // 可选：从 git 拉取技能集（配合本地缓存）

type Registry struct { /* 技能名 -> 元数据 + 加载器 */ }
func NewRegistry(loaders ...Loader) (*Registry, error) // 启动时 List() 建立目录
func (r *Registry) Catalog() []Metadata
func (r *Registry) Activate(ctx context.Context, name string) (*Skill, error) // 激活=加载 body（可加 LRU 缓存）
func (r *Registry) Lookup(name string) (Metadata, bool)
```

### 7.3 SKILL.md 解析器

- 用 `gopkg.in/yaml.v3` 解析 frontmatter（首行 `---` 到第二个 `---`），正文为其余 Markdown。
- 校验规则（对齐规范，解析失败即报错并跳过该技能）：
  - name：正则 `^[a-z0-9]+(-[a-z0-9]+)*$`、长度 ≤64、**必须等于目录名**（case-insensitive 允许，参照 Claude API 的 `Financial_Skill`→`financial-skill` 规则）；
  - description 非空且 ≤1024；
  - 保留字拒绝：`anthropic`、`claude`（对齐远程 API 约束，防混淆）；
  - 体积上限：SKILL.md 建议 <500 行 / <20KB，超出提示拆分到 references/（不硬失败，仅警告）；整包上限（如 30MB，对齐 Anthropic）。
- 相对路径安全：`filepath.Clean` 后必须仍位于技能目录内（防 `../../` 越权读文件）。

### 7.4 如何注入系统提示词（渐进式披露接线）

两级注入，注入点放在 Agent 组装 system prompt 的最后阶段：

1. **启动/会话建立时**：`Registry.Catalog()` 渲染为固定格式目录块（每技能一行：`- name: <name> — <description>`），拼进 system prompt。给模型的指令示例：
   > "你具备以下技能（Skills）：[目录]。当任务与某技能相关时，调用 `use_skill` 工具激活它，然后严格遵循其指令。"
2. **激活时**：两种机制可选（建议都支持）：
   - **工具机制（推荐，模型自主可控）**：注册一个内置工具 `use_skill(name)`——handler 里 `Registry.Activate()` 读取 body，把 body 作为**工具结果**返回给模型（等价于 Claude Code 把技能内容作为一条消息注入）。优点：不需要框架猜相关性、模型按 description 决策、天然支持工具权限控制。
   - **框架自动匹配（可选）**：用户显式指定（如 `Agent.WithSkills("pdf")`）或关键词命中 description 时，框架在下一轮把 body 拼进 system 消息（对齐 Claude 的 disable-model-invocation=false 行为）。
3. **引用文件**：body 内保留相对路径（`scripts/x.py`、`references/REFERENCE.md`），模型需要时通过既有的文件读取工具读取 —— 由 Agent 的 tools 提供（对齐规范"Resources loaded on demand"）。

### 7.5 与 toolkit 的关系（关键设计决策）

- **职责分离**：
  - `toolkit` = **可执行函数集合**（JSON schema → 模型调用 → 执行结果），是"能力"；
  - `skills` = **指令 + 资源包**（Markdown 流程、脚本、参考文档），是"知识/流程"；
  - 交叉点：技能 body 中可以指示模型使用某些 toolkit 工具；`allowed-tools` 字段可做激活期的工具授权（对齐 Claude Code 的 turn-scoped grant，本框架实现为：激活技能后，该 turn 内放行指定工具而无需额外审批）。
- 命名建议：`Agent.Skills([]string)` 或 `Agent.SkillsDir(string)`；`skills` 包不 import `toolkit`（保持单向依赖：agent 层组装二者）。
- 远程技能（`tools/claude/skills.go`）作为 toolkit 的一种继续存在，命名上区分 `claude_skills`（远程）与本地 `skills`（本地）。

### 7.6 与 MCP 的关系

- 技能包可声明其依赖的 MCP server（metadata 或独立 `skill.mcp.json`），Agent 装配时按需拉起；`pkg/agno/mcp` 复用。
- 不把技能内容塞进 MCP：MCP 管连接与工具，技能管指令（见第 2 节结论）。

### 7.7 可观测性接线（OTel）

go.mod 已有 `go.opentelemetry.io/otel v1.38.0` + otlptrace/otlptracehttp/sdk，建议：
- 在 agent run 生命周期创建 span：`invoke_agent {name}`（kind=INTERNAL，`gen_ai.operation.name=invoke_agent`、`gen_ai.agent.name`、`gen_ai.provider.name`、`gen_ai.request.model` 在创建时设置）；
- 每次工具调用 `execute_tool {tool.name}`（`gen_ai.tool.call.id`、`gen_ai.tool.name`），MCP 调用再嵌 `mcp.client` span；
- LLM 调用 `chat {model}`（CLIENT），响应后写入 `gen_ai.usage.input_tokens/output_tokens/cache_read.input_tokens`；
- 技能激活打点：可用 `gen_ai.system_instructions`（Opt-In）记录注入的技能 body，或自定义事件（`gen_ai.operation.name=plan` 不适用时用事件/log）；
- 导出：`otlptracehttp` 直连 OTLP Collector / Langfuse `/api/public/otel`（Basic Auth + `x-langfuse-ingestion-version: 4`），实现 20 行内可换后端。注意遵循 semconv-genai 的 Development 状态，属性名用常量集中管理。

### 7.8 评估扩展

- `pkg/agno/eval` 现状：`Scenario{Input, ExpectedContains}`（子串判词）→ 保留为确定性层，新增：
  - `JudgeEval`（LLM-as-judge：criteria、numeric/binary、threshold、judge model、num_iterations、avg_score）——对齐 agno AgentAsJudgeEval；
  - `AccuracyEval`（expected_output + judge）——对齐 agno AccuracyEval；
  - `ToolTraceAssert`（校验 tool call 序列，对齐 DeepEval Tool Correctness）；
  - `Suite`（多个 Case + 门禁阈值，CI 用）+ 结果持久化（已有 sqlite 依赖）与 `/eval-runs` HTTP 端点（`pkg/agentos` 扩展）。
- 黄金数据集放 `testdata/evals/*.yaml`（OpenAI Evals registry 风格），CI 脚本跑 suite 出回归报告。

### 7.9 安全 Checklist（技能机制特有）

- [ ] 目录遍历防护（7.3）；技能体积/文件数上限。
- [ ] shell 执行默认关闭（`!`cmd`` 类动态注入默认禁用，显式开启，对齐 Claude Code `disableSkillShellExecution`）。
- [ ] `allowed-tools` 激活期授权 + 权限模型可被用户策略覆盖（对齐 Claude Code 权限规则 `Skill(name)`）。
- [ ] 技能来源可信审计（项目技能合入前 review，防"技能自授权"）。
- [ ] 密钥不写入技能包，通过 run context 注入（agno 的 RunContext 模式）。

---

## 附：关键 URL 索引

| 主题 | URL |
|---|---|
| Agent Skills 开放规范 | https://agentskills.io/specification |
| Anthropic 工程博客（渐进式披露） | https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills |
| anthropics/skills 官方仓库 | https://github.com/anthropics/skills |
| Claude Code Skills 文档 | https://code.claude.com/docs/en/skills |
| Claude API Skills 指南 | https://docs.claude.com/en/api/skills-guide |
| MCP 规范 2025-06-18 | https://modelcontextprotocol.io/specification/2025-06-18 |
| MCP 架构 | https://modelcontextprotocol.io/docs/learn/architecture |
| OTel GenAI 语义约定仓库 | https://github.com/open-telemetry/semantic-conventions-genai |
| OTel GenAI agent spans | https://github.com/open-telemetry/semantic-conventions-genai/blob/main/docs/gen-ai/gen-ai-agent-spans.md |
| OTel gen_ai 属性注册表 | https://github.com/open-telemetry/semantic-conventions-genai/blob/main/docs/registry/attributes/gen-ai.md |
| Langfuse OTel | https://langfuse.com/integrations/native/opentelemetry |
| OpenLLMetry | https://www.traceloop.com/docs/openllmetry/introduction |
| OpenAI Agents SDK（含 tracing） | https://openai.github.io/openai-agents-python/tracing/ |
| Cursor Rules | https://cursor.com/docs/context/rules |
| OpenAI Evals | https://github.com/openai/evals |
| DeepEval | https://github.com/confident-ai/deepeval |
| agno Evals / Tracing | https://docs.agno.com/evals/agent-as-judge/overview 、https://docs.agno.com/evals/accuracy/overview 、https://docs.agno.com/agent-os/tracing/overview |
