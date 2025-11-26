# 功能规格：Go 版 Agno Agents 重构计划

**功能分支**：`001-agno-agents-refactor`  
**创建时间**： 2025-02-25  
**状态**： 草稿  
**输入**：用户描述：“使用golang语言对agno的agents进行重塑工作，使用chrome-devtools打开他们的官方文档https://docs.agno.com/，然后将所有的左侧菜单都阅读一遍。将这些内容归纳总结后。根据我们的宪章要求对这个框架agno框架进行重构工作计划。”

## 用户场景与测试 *(必填)*

### 用户故事 1 - 平台负责人获取 Go 版同等能力（优先级：P1）

平台负责人希望在现有 Go 代码库中具备 docs.agno.com/basics 覆盖的核心能力（Agents/Teams、Memory/Sessions/State、Knowledge/RAG、Tools/MCP/HITL/Guardrails、Workflows、Multimodal、Evals/Reasoning/Telemetry）的同等行为，以便统一治理、成本与性能监控。

**优先级原因**：是最核心的价值兜底，支撑后续接口、运营与性能目标。

**独立测试**：选取文档左侧导航中代表性的 5 个场景（基础 agent、记忆与会话、RAG 检索、工具调用与 HITL、工作流分支），用 Go 版本串起端到端用例，验证输出、事件与边界处理与 Python 版一致。

**验收场景**：

1. **Given** 官方示例的输入与工具调用链，**When** 在 Go 版运行 agent，**Then** 返回内容、工具调用顺序与会话状态与示例一致且可流式输出。
2. **Given** 含记忆与聊天历史的连续对话，**When** 在 Go 版重复同一脚本，**Then** 会话状态、记忆读写与回滚逻辑与文档描述一致。

---

### 用户故事 2 - 集成工程师快速切换供应商与接口（优先级：P2）

集成工程师需要在 Go 环境里切换或并行使用 Basics 文档列出的模型与向量库集成（OpenAI、Gemini、Groq、GLM4、OpenRouter、SiliconFlow、Cerebras、ModelScope、Ollama 等），并用基础示例/CLI 展示核心能力，无需 Agent OS 接口演示。

**优先级原因**：直接影响商用可落地性与演示成功率。

**独立测试**：为每个供应商配置 `.env` 占位，运行示例脚本/接口 smoke 测试，收集成功率与跳过原因，输出至 artifacts 日志。

**验收场景**：

1. **Given** `.env` 中配置的可用密钥，**When** 运行供应商示例，**Then** 请求成功、延迟与输出格式符合官方示例记录，失败时日志包含跳过原因。
2. **Given** 未配置密钥或本地端点不可达，**When** 运行供应商测试，**Then** 自动跳过并在日志中标注 unreachable/缺密钥。

---

### 用户故事 3 - 质量负责人要求可验证的规范与基准（优先级：P3）

质量负责人需要一套可执行的契约/治具与性能基准，以便验证 Go 重构覆盖文档导航的关键模块（知识、记忆、工具、会话、工作流、多模态），并持续满足 20% p95 延迟改善与 25% 峰值内存下降目标。

**优先级原因**：确保重构不偏离业务价值，并具备可持续的回归能力。

**独立测试**：运行契约测试与供应商基准，比较基线与当前结果，自动生成差异报告与偏离说明。

**验收场景**：

1. **Given** 现有契约治具与基线，**When** 在 Go 版跑通知识、记忆、工具调用与工作流测试，**Then** 通过率 ≥ 95%，差异记录在 `contracts/deviations.md`。
2. **Given** 性能基准脚本，**When** 在相同硬件上运行，**Then** p95 延迟相对 Python 版下降 ≥20%，峰值内存下降 ≥25%，未达标时报告包含瓶颈描述。

---

### 边界场景

- 当模型供应商密钥缺失或端点不可达时，流程自动降级为跳过并记录原因，不影响其他用例执行。
- 当知识库/向量库不可用时，RAG 路径需回退为提示级回答或明确错误提示。
- 多模态输入（音频/图片/视频/文件）尺寸超限或格式不符时，应给出可操作的错误信息并不中断会话状态。
- 会话恢复与记忆持久化失败时，需保持只读输出且记录恢复手段，避免数据污染。

## 需求 *(必填)*

### 功能需求

- **FR-001**：提供 Go 版 Agent/Team 执行引擎，覆盖官方导航中的基础运行、调试、流式/异步、工具调用、HITL 与多模态能力，行为需与 Python 示例一致可回放。
- **FR-002**：实现会话、状态与记忆管理（创建、搜索、共享、持久化、回滚），在连续对话与多用户多会话下保持一致性，并支持日志化的会话摘要。
- **FR-003**：提供知识管理与检索（内容接入、分块、嵌入、多向量库、过滤、RAG/Agentic RAG、团队知识协作），可按文档场景运行并生成基线治具。
- **FR-004**：提供基础层入口（示例/CLI/SDK）覆盖 Basics 文档场景，无需 Agent OS 接口；认证与中间件仅保留支撑基础示例所需的最小能力（仅允许 API Key Header 认证，允许的中间件仅日志/恢复/超时，禁止 Basic/OAuth/JWT 等其他认证与自定义拦截）。验收：示例与测试需覆盖允许/禁止路径。
- **FR-005**：覆盖工具体系（内置工具、MCP、多供应商工具、选择策略、工具调用上限）、人机协同（HITL）、守护（Guardrails 与 PII/Prompt 注入检测）与观测（会话指标、运行指标）。
- **FR-006**：提供模型与向量库供应商矩阵的配置、回退与错误处理（密钥缺失/不可达自动跳过、输出日志），并在 `.env.example` 预置所需变量。
- **FR-007**：建立契约测试、供应商测试与性能基准，生成/更新 fixtures（含差异记录），输出覆盖率与性能报告，确保综合覆盖率 ≥85% 且满足性能改进目标。

### 关键实体 *(若涉及数据则必填)*

- **Agent / Team**：执行用户请求的主体，负责计划、调用工具、生成回复，并可协作或委派。
- **Session / State / Chat History**：跨轮次保存上下文、状态与对话历史，用于连续对话与回放。
- **MemoryStore**：记忆存储抽象，支持多实现（内存、文件、Bolt/Badger 等），提供读写、搜索与回滚。
- **Knowledge Base / Retriever**：知识内容、分块与索引的集合，支持多向量库与过滤策略。
- **Tool / MCP Server**：可被调用的外部能力集合，含本地工具与 MCP 协议服务器。
- **Model Provider / Embedder**：对接多家模型与嵌入供应商的抽象，支持能力宣告与回退。
- **Workflow / Hook / Middleware**：串联步骤、分支、条件与前后置处理的编排与扩展点。

## 成功标准 *(必填)*

### 可度量结果

- **SC-001**：基于官方 5 个核心示例的端到端回放通过率 ≥95%，失败项均有差异说明与修复计划。
- **SC-002**：在同一硬件上，相对基线 `specs/001-agno-agents-refactor/artifacts/baseline/python-bench.json`（预存的 Python 参考值）p95 端到端延迟下降 ≥20%，峰值内存下降 ≥25%，并在 `artifacts/bench.txt` 写入差异；未达标时说明瓶颈。
- **SC-003**：模型/向量库供应商示例执行成功或被正确跳过的比例 ≥90%，所有跳过原因写入 `artifacts/coverage/providers.log` 并可复跑。
- **SC-004**：综合测试覆盖率 ≥85%，并包含用户故事场景、契约测试、供应商集成与性能基准四类报告。
- **SC-005**：安全与合规：密钥不出现在代码/日志/仓库，异常路径均给出可操作提示，HITL/守护策略在测试中无漏拦截。

## 宪章对齐 *(必填)*

- **纯 Go / 禁止桥接**：所有代理、会话、知识、工具、工作流与接口逻辑以 Go 1.25.1 实现；不调用 Python/cgo/子进程，替换原 Python 能力的 Go 接口置于 `go/internal` 与 `go/pkg` 中，命令入口 `go/cmd/agno`。
- **模型供应商矩阵**：对接 Ollama、Gemini、OpenAI、GLM4、OpenRouter、SiliconFlow、Cerebras、ModelScope、Groq 的 chat/stream/embedding 能力，`.env.example` 列出各自 key/endpoint/model 变量，未配置时自动跳过并记录。
- **契约/治具与基准**：治具位于 `specs/001-agno-agents-refactor/contracts/fixtures`，偏差记录于 `contracts/deviations.md`；契约/供应商/性能基准脚本均不依赖 Python 运行时。
- **自动化与 Make**：扩展/复用 `make fmt lint test providers-test coverage bench gen-fixtures release constitution-check`，CI 复用 `.github/workflows/ci.yml`，确保新增目标在 CI 可跑。
- **测试纪律 + 85% 覆盖率**：新增/迁移 Go 单测、契约测试、供应商集成测试与性能基准，覆盖核心实体与边界场景，保障综合覆盖率 ≥85% 并产出报告。
- **密钥与安全**：所有密钥放置 `.env.example` 占位，使用环境注入与日志脱敏；供应商不可达/缺密钥时以跳过+记录方式处理，避免泄漏。
- **文档同步（VitePress）**：完成实现后，同步更新与本次 Go 改动相关的 docs.agno.com/basics 对应 VitePress 文档章节，内容仅围绕 `./go` 实现的 Basics 五场景与完整的 provider 客户端示例，移除无关部分；采用官方 5 场景分节（前置/环境 → basic/memory/rag/tool+HITL/workflow 运行命令与契约测试 → 回退/安全/FR-004 认证 → 供应商跳过与日志 → 构建命令），提交文档源并通过 VitePress 构建检查。

## Clarifications

### Session 2025-02-25

- Q: 本阶段是否将范围严格限定在 docs.agno.com/basics，暂不覆盖 Agent OS（A2A/AG UI/Slack/WhatsApp）、控制面、中间件、部署模板？ → A: 严格聚焦 Basics，Agent OS/接口层暂不做。
- Q: VitePress 文档更新范围与验收方式？ → A: 仅更新与本次 Go 实现相关的 Basics 章节，提交文档源并通过 VitePress 构建。
- Q: VitePress 文档源路径与构建命令？ → A: 文档源位于 `../rex-agno/agno-Go/docs`（VitePress），使用 `npm run docs:build` 校验构建。
- Q: Basics CLI/SDK 入口允许的认证与中间件范围？ → A: 仅允许 API Key Header 认证，允许的中间件仅日志/恢复/超时，禁止其他认证与自定义拦截。
- Q: PII/Prompt 注入拦截测试结果存放与阻断策略？ → A: 记录在 `artifacts/coverage.txt`，漏拦截或误拦截均阻断 CI，并写明案例以对齐 SC-005。

### Session 2025-11-26

- Q: VitePress Basics 章节采用何种结构以对标 docs.agno.com/basics？ → A: 采用官方 5 场景分节（前置/环境 → basic/memory/rag/tool+HITL/workflow 运行命令与契约测试 → 回退/安全/FR-004 认证 → 供应商跳过与日志 → 构建命令），内容基于 Go 实现与测试命令编写。
- Q: VitePress 文档是否仅保留本次重构的 Basics 场景并补全 provider 客户端示例，移除无关内容？ → A: 是，整篇文档围绕 `./go` 实现的 Basics 五场景与相关回归/构建命令，补全 provider 客户端示例，其余无关内容移除。
