# TS 生态两大框架 + Go 生态盘点 —— agno-go 重构调研报告

> 调研日期：2026-08-03。调研方法：codeload tarball 全量下载解压 + 源码逐行读取（源码缓存保留在 `docs/research/_src/repos/`）。证据级别：⭐=已读源码文件:行号；○=官方文档/仓库结构；✗=无法验证（如实标注）。
> 仓库现状：vercel/ai 25.9k★（活跃）；genkit-ai/genkit 6.3k★（原 firebase/genkit 改名，**含官方 Go 实现**）；langchaingo 9.5k★（最后 push 2026-01，近 7 个月停滞）；openai-go 3.3k★（活跃）。

---

## 1. Vercel AI SDK（github.com/vercel/ai）⭐

### 1.1 核心抽象（文件路径均指 `packages/` 下）

| 抽象 | 文件 | 要点 |
|---|---|---|
| LanguageModelV4 | `provider/src/language-model/v4/language-model-v4.ts`（61 行） | **极简**：仅 `specificationVersion/provider/modelId/supportedUrls` + `doGenerate()/doStream()` 两方法；同目录 v2/v3/v4 规格并存 |
| tool | `provider-utils/src/types/tool.ts` | `inputSchema`(必需) + `execute`(可选，无则声明式/客户端执行) + `outputSchema/contextSchema/needsApproval/providerOptions/metadata`。工具=数据+可选函数 |
| generateText | `ai/src/generate-text/generate-text.ts`（1909 行） | 顶层纯函数；默认 `stopWhen=isStepCount(1)`（单轮）；核心 do/while 循环（L1429-1439）：客户端工具调用全部执行或被拒 + pending deferred 清空 + 停止条件未满足 → 继续 next step；每 step 产出 StepResult（messages/usage/finishReason），聚合为 GenerateTextResult |
| streamText | `ai/src/generate-text/stream-text.ts`（94.9K） | 同一循环流式版；`textStream/fullStream`（TextStreamPart 异步迭代器）；`toUIMessageStream()/toUIMessageStreamResponse()` 把流直接变 UI 消息流/HTTP Response |
| Agent（v5） | `ai/src/agent/tool-loop-agent.ts`（323 行）+ `agent.ts` | **Agent 只是 generateText/streamText 的配置壳**：默认 `stopWhen=isStepCount(20)`、prepareCall 注入 runtimeContext、mergeCallbacks——无自有运行循环，随时可退回裸函数 |
| React 集成 | `react/src/use-chat.ts`（200 行） | 薄壳包 `Chat` 类 + useSyncExternalStore 订阅 messages/status/error；sendMessage/stop/regenerate/resumeStream/addToolOutput |

分层：`ai`（编排层）→ `@ai-sdk/provider-utils`（工具）→ `@ai-sdk/provider`（协议）；`packages/<provider>` 独立实现，`packages/<framework>` 官方 UI 集成（react/vue/svelte/angular/rsc）。

### 1.2 为什么 DX 公认最好

- **极薄模型接口**：接新 provider 只需实现 doGenerate/doStream，SDK 包办循环/重试/结构化/流协议——对比 agno Python 把 ReAct 循环塞进 Model（144K 文件），Vercel 的接口层只有协议没有"框架感"。
- **函数组合而非类继承**：一切都是可组合的纯函数（generateText/streamText/generateObject），Agent 是配置壳而非基类。
- **流式一等公民**：TextStreamPart 异步迭代器、UI 消息流转换器内置，前端集成零胶水。
- **工具 = 数据 + 可选函数**：`inputSchema` 声明式 schema，`execute` 可选（无 execute 的工具由客户端/声明式执行，天然支持 HITL 审批）。

### 1.3 优点

1. 薄、可组合、无框架感；2. 流式与 UI 集成标杆；3. 多 provider 极简接入；4. v5 gateway（模型路由/限流）内置。

### 1.4 缺陷

1. 偏 UI 应用层：无会话管理、无记忆、无多 Agent 编排（Agent 只是循环壳）；2. 工具 schema 是手写 TS 类型（无运行时校验闭环）；3. 无内置可观测性（靠第三方 OTel 集成）；4. 无技能/知识机制。

### 1.5 对 Go 框架的启示

- **Model 接口薄到极致**：Go 版 `Provider` 接口只保留 `Invoke/Stream` 两个方法 + 元数据，循环/重试/结构化全部在 Runner —— 这是"模型层最优雅"的实证。
- **Agent 是配置壳**：Agent 本身不含循环逻辑，只是工具+模型+指令的组装；Go 版同样：`Agent` 组合 Runner，而非继承。
- **工具=数据+可选执行**：schema 与 execute 分离，天然支持声明式工具（客户端执行/HITL）。
- **多 provider 用同一协议**：Go 版 models 包内 `Provider` 接口 + 各 provider 适配器，禁止各自为政。

---

## 2. Google Genkit（github.com/genkit-ai/genkit，原 firebase/genkit）⭐（Go 部分）

**重要发现：Genkit 有官方 Go 实现**（`go/` 目录，与 js/ py/ 并列）——对 agno-go 是直接的 Go 生态参照。

### 2.1 核心抽象（Go 实现，`go/core/`）

| 抽象 | 文件 | 要点 |
|---|---|---|
| `Flow[In, Out, Stream]` | `core/flow.go` | **泛型流程**：`NewFlow[In, Out](name, fn Func[In, Out])` + `NewStreamingFlow[In, Out, Stream]`；`Run(ctx, name, fn)` 执行并自动埋点 |
| `Action` | `core/action.go`（14.7K） | 可注册的执行单元（Flow 的底层）；`DefineAction` 注册到 Registry |
| Registry | `core/api/registry.go` | 命名注册中心：按 name 查找/调用 action（插件模型） |
| Tracing | `core/tracing/` | **内置 tracing**：Flow/Action 自动创建 span，`Run` 包装的每个 fn 都有 trace；支持 OTLP 导出 |
| Middleware | `core/middleware.go` | 请求中间件（日志/指标/限流挂载点） |
| 错误/状态 | `core/error.go`、`core/status_types.go` | 结构化错误与状态码 |

### 2.2 优点

1. **官方 Go 实现**（罕见）：TS/Go/Python 三语言同构 API，Go 生态里最完整的"框架级"参考；2. **tracing 内置**：Flow 执行自动 span，无需手动埋点；3. 泛型 Flow[In,Out,Stream] 与 Go 风格契合；4. Registry 插件模型清晰。

### 2.3 缺陷

1. 偏"函数流程"而非"agent 循环"：无 ReAct 循环、无工具调用闭环（Flow 是用户定义的函数，模型调用需自己接）；2. 无记忆/会话管理；3. 生态以 Google 系为主（googlegenai 插件）。

### 2.4 对 Go 框架的启示

- **Flow[In, Out, Stream] 泛型**是 Go 表达"可复用流程"的优雅方式——对应我们 workflow 的 Step 泛型化。
- **tracing 内置**：span 自动创建 + OTLP 导出是"可观测性内置"的 Go 实证，比"事后加插件"优雅得多。
- Registry 模式可借鉴为 `pkg/agno/registry`（命名注册 provider/工具/技能）。

---

## 3. Go 生态盘点

### 3.1 langchaingo（github.com/tmc/langchaingo）⭐ 9.5k★，**停滞**

- 架构照搬 Python LangChain：`chains/`（LLMChain、RetrievalQAChain...）、`llms/`、`memory/`、`outputparser/`、`prompts/`、`documentloaders/`、`embeddings/`、`schema/`。
- **缺陷**：① 链式 API 是 Python 概念的直译，Go 用户心智负担重（Go 社区习惯函数组合而非链对象）；② 最后 push 2026-01，近 7 个月停滞；③ 无 Agent 循环（有 tools/ 但无完整 ReAct runner）；④ 大量 `interface{}` 透传。
- **教训**：Go Agent 框架不要照搬 Python 链式 API；停滞的项目说明"Python 概念直译"路线在 Go 走不通。

### 3.2 gollm（github.com/parakeet-nest/gollm）○ 活跃

- 轻量 LLM 封装：`llm/`（openai/anthropic/ollama... provider 适配）+ `config/` + `presets/` + `optimizer/`（prompt 优化）+ `assess/`（评估）。
- 定位是"LLM 客户端库"而非 Agent 框架；无工具循环、无记忆、无多 Agent。

### 3.3 go-agents（github.com/mattn/go-agents）○ 停滞

- 迷你 Agent 示例库（~200 行核心），演示 OpenAI function calling 循环；无框架价值，仅教学。

### 3.4 openai-go（github.com/openai/openai-go）⭐ 3.3k★，活跃

- OpenAI 官方 Go SDK：`client/` + `option/` + `auth/` + `azure/` + `bedrock/` 多端点；类型齐全、代码生成维护。
- **对 agno-go 的价值**：官方 SDK 成熟，我们 models/openai 适配层可直接基于它（而非自写 HTTP 客户端）；其 option 模式（`option.WithAPIKey`）是 Go 配置的标杆。

### 3.5 Semantic Kernel Go 版：**官方不存在**（✗ 已验证 404）

- `microsoft/semantic-kernel-go`、`microsoft/kernel-go`、`microsoft/semantickernel-go` 全部 404；官方支持 C#/Python/Java。社区仅有第三方 `mfmayer/gosk`。
- 意义：Go 生态缺少"企业级框架"标杆，agno-go 有空白市场机会。

### 3.6 Go 生态结论

- **空白即机会**：Go 生态没有"优雅的 Agent 框架"（langchaingo 停滞且概念过时、gollm 只是客户端、无官方 SK Go 版）——agno-go v2 定位"Go 第一个优雅的多 Agent 框架"有真实市场空间。
- **可借鉴**：openai-go 的 option 模式、Genkit Go 的 Flow 泛型 + 内置 tracing、Vercel 的薄 Provider 接口。
- **必须避免**：langchaingo 的"Python 概念直译"、chains 链式 API、interface{} 透传。

---

## 4. 对 Go 版优雅 Agent 框架的设计建议（整合）

1. **Provider 接口薄到极致**（Vercel 实证）：`Invoke(ctx, req) / Stream(ctx, req)` + 元数据，循环在 Runner。
2. **Agent 是配置壳 + 组合**（Vercel v5 实证）：Agent 不含循环，只组装 Provider + Tools + Memory + Skills + Instructions。
3. **泛型贯穿**（Genkit Go 实证）：`Flow[In, Out, Stream]` 式泛型用于 workflow step；`RunContext[D]` 式依赖注入用于工具。
4. **tracing 内置**（Genkit Go 实证）：span 自动创建，默认 no-op，OTLP 一行导出。
5. **Option 模式**（openai-go 实证）：`WithXxx` 函数式选项做配置，禁 115 参数构造器。
6. **工具 = 数据 + 可选执行**（Vercel 实证）：schema 声明 + execute 可选，支持 HITL 审批。
7. **Go 特有的优雅点**：goroutine 并发工具调用（天然 Actor，AutoGen 的 AgentID 可 Go 化）；静态类型工具定义（struct tag → JSON Schema）；单二进制 embed 技能（FS loader）。
