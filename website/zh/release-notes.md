---
title: 版本发布说明
description: Agno-Go 版本历史和发布说明
outline: deep
---

# 版本发布说明

## Version 1.2.9 (2025-11-14)

### ✨ 亮点
- **EvoLink 媒体 Provider**：在 `pkg/hno/providers/evolink` 与 `pkg/hno/models/evolink/*` 下提供文本、图片、视频模型，支持异步任务轮询与类型安全参数。
- **EvoLink 媒体智能体示例**：新增 `website/examples/evolink-media-agents.md` 与 `website/zh/examples/evolink-media-agents.md`，演示如何在工作流中串联文本 → 图片 → 视频。
- **知识上传分块（chunking）**：`POST /api/v1/knowledge/content` 支持 `chunk_size` 与 `chunk_overlap`，可用于 JSON、`text/plain`（query 参数）与 multipart 表单；每个分块会记录 `chunk_size`、`chunk_overlap` 与 `chunker_type` 元数据，对齐 Python AgentOS。
- **AgentOS HTTP 提示**：文档补充如何自定义健康检查路径、使用 `/openapi.yaml` 与 `/docs`，以及在重挂载路由后调用 `server.Resync()` 的最佳实践。

### 📚 文档
- 更新 `website/api/agentos.md` 及本地化版本，在 Knowledge 章节中补充分块参数与示例，并在最佳实践中加入 AgentOS HTTP 提示。
- EvoLink 示例页集中说明所需环境变量、模型表与合规注意事项（HTTPS 回调、审核策略等）。

### ✅ 兼容性
- 增量版本，无破坏性变更；`chunk_size` / `chunk_overlap` 为可选参数，缺省行为与之前一致。

## Version 1.2.8 (2025-11-10)

### ✨ 亮点
- Run Context 贯穿执行（hooks → tools → telemetry），流式事件包含 `run_context_id` 便于链路关联。
- 会话状态持久化 `AGUI` 子态，`GET /sessions/{id}` 可返回 UI 状态。
- 向量检索：
  - 可插拔 VectorDB 提供方（Chroma 默认；Redis 可选，不做强依赖）。
  - VectorDB 迁移 CLI（`migrate up/down`）支持幂等创建/回滚。
- Embeddings：VLLM 提供方（本地/远端）遵循通用接口。
- MCPTools：新增可选 `tool_name_prefix`，对注册工具名进行前缀化。

### 🔧 改进
- 移除 Redis 作为默认向量库依赖；启用时按配置注册，未启用对现有用户零影响。
- 团队模型继承仅下沉主模型；辅助参数需在 Agent 端显式启用。

### 🐛 修复
- 模型响应正确绑定到运行步骤，修复历史中“未绑定/零值”。
- 团队工具判定与 OS Schema 对齐，保留成员工具集合。
- 异步 DB 知识过滤尊重复合谓词与超时上下文，不泄露 goroutine。
- 工具包导入缺失模块时返回结构化错误而非 panic。
- AgentOS 错误响应包络标准化，契约测试更稳定。

### 🧪 测试
- 覆盖 Run Context、AGUI 持久化、团队主模型继承、MCP 前缀、VLLM 嵌入。
- 可选依赖（Redis）用例按环境开关执行，默认跳过。

### ✅ 兼容性
- 增量更新；公共 API 保持不变；可选能力默认关闭。

## Version 1.2.7 (2025-11-03)

### ✨ 亮点
- Go 原生会话服务，完整复刻 Python AgentOS 的 `/sessions` API，提供基于 Postgres 的 CRUD、Chi 路由和健康检查（[指南](/zh/guide/session-service)）。
- 全环境部署资产：专用 Dockerfile、集成 Postgres 的 Docker Compose 栈，以及面向 Kubernetes 的 Helm Chart。
- 更新文档与 `test-session-api.sh` 脚本，可在本地或 CI 环境校验端点。

### 🔧 改进
- Postgres 存储实现采用强类型 DTO 与事务安全操作，与现有 AgentOS 架构保持一致。
- 新增配置指南，涵盖 DSN 绑定、环境变量与工作流脚本，帮助上线 Go 会话运行时。

### 🧪 测试
- 合同测试套件对比 Go 与 Python 的响应，并补充专门的 Postgres 存储测试。

### ✅ 兼容性
- 增量更新；Go 会话运行时为可选组件，可与 Python 服务并行运行。

## Version 1.2.6 (2025-10-31)

### ✨ 新增亮点
- 会话对齐：新增会话复用接口、同步/异步摘要（`GET/POST /sessions/{id}/summary`）、历史分页参数（`num_messages`、`stream_events`）、运行元数据（缓存命中、取消原因、时间戳）。
- 响应缓存：为 Agent/Team 提供内存 LRU 缓存与可配置的摘要管理器默认值。
- 媒体附件流水线：Agent/Team/Workflow 全面支持媒体附件，提供校验工具与 `WithMediaPayload` 运行选项。
- 存储适配器：新增 MongoDB 与 SQLite 会话存储，实现与 Postgres 一致的 JSON 协议。
- 工具包扩展：Tavily Reader/Search、Claude Agent Skills、Gmail 标记已读、Jira 工时、ElevenLabs 语音、强化文件工具。
- 文化知识管理器：异步维护组织知识，支持标签筛选。

### 🔧 改进
- 工作流引擎持久化取消原因、支持 resume-from 检查点，并能处理仅媒体负载。
- AgentOS 会话 API 开放摘要端点、复用语义与带 SSE 开关的历史分页。
- MCP 客户端缓存能力清单并转发媒体附件，降低调用延迟。

### 🧪 测试
- 新增缓存层、摘要管理器、存储驱动、工作流恢复路径与新工具包的覆盖。

### ✅ 兼容性
- 增量变更，保持向后兼容。

## Version 1.2.5 (2025-10-20)

### ✨ 新增
- 模型提供商：Cohere、Together、OpenRouter、LM Studio、Vercel、Portkey、InternLM、SambaNova（支持同步/流式与函数调用）
- 核心模块：评估系统（场景评测、指标汇总、多模型对比）、媒体处理（图片元数据；音/视频占位）、调试工具（请求/响应转储）、云部署占位（NoopDeployer）
- 集成与生态：集成注册表（注册/列表/健康检查）、通用 utils（JSONPretty、Retry）

### 🔧 修复
- Airflow 工具返回结构对齐 Airflow REST API v2：`total_entries`、`dag_run_id`、`logical_date`
- 站点图片缺失：将首页 hero 图片从 `/logo.svg` 更换为 `/logo.png`

### 🧪 测试
- 为新增模型与模块补充了聚焦单测；保留现有基准测试

### ✅ 兼容性
- 完全向后兼容（增量变更）

## Version 1.2.1 (2025-10-15)

### 🧭 文档重组

- 明确区分：
  - `website/` → 已实现的对外文档（VitePress 网站）
  - `docs/` → 设计草案、迁移计划、任务与开发者/内部文档
- 新增 `docs/README.md` 说明策略与入口
- 新增 `CONTRIBUTING.md` 方便贡献者上手

### 🔗 链接修复

- README、CLAUDE、CHANGELOG 与发布说明链接统一指向 `website/advanced/*` 与 `website/guide/*`
- 移除指向 `docs/` 下重复实现文档的旧链接

### 🌐 网站更新

- API：在 AgentOS 页面补充知识库 API（/api/agentos）
- 确认 Workflow History、Performance 页面为规范引用

### ✅ 行为变更

- 无（仅文档与结构调整）

### ✨ 本次新增（已实现）

- A2A 流式端点事件类型过滤（SSE）
  - `POST /api/v1/agents/:id/run/stream?types=token,complete`
  - 仅输出所请求的事件类型；标准 SSE 格式；支持 Context 取消
- AgentOS 内容抽取中间件
  - 将 JSON/Form 中的 `content/metadata/user_id/session_id` 注入上下文
  - 支持 `MaxRequestSize` 请求大小保护与路径跳过
- Google Sheets 工具（服务账号）
  - 提供 `read_range`、`write_range`、`append_rows`，支持 JSON/文件凭证
- 最小化知识入库端点
  - `POST /api/v1/knowledge/content` 支持 `text/plain` 与 `application/json`

企业验收步骤请参考 [`docs/ENTERPRISE_MIGRATION_PLAN.md`](https://github.com/rexleimo/agno-Go/blob/main/docs/ENTERPRISE_MIGRATION_PLAN.md)。

## Version 1.1.0 (2025-10-08)

### 🎉 重点功能

本版本为生产级多智能体系统带来强大的新功能：

- **A2A 接口** - 标准化的 Agent 间通信协议
- **Session State 管理** - 工作流步骤间的持久化状态
- **多租户支持** - 单个 Agent 实例服务多个用户
- **模型超时配置** - LLM 调用的细粒度超时控制

---

### ✨ 新功能

#### A2A (Agent-to-Agent) 接口

基于 JSON-RPC 2.0 的标准化 Agent 间通信协议。

**核心特性：**
- REST API 端点（`/a2a/message/send`、`/a2a/message/stream`）
- 多媒体支持（文本、图片、文件、JSON 数据）
- Server-Sent Events (SSE) 流式传输
- 与 Python Agno A2A 实现兼容

**快速示例：**
```go
import "github.com/rexleimo/agno-go/pkg/agentos/a2a"

// 创建 A2A 接口
a2a := a2a.New(a2a.Config{
    Agents: []a2a.Entity{myAgent},
    Prefix: "/a2a",
})

// 注册路由 (Gin)
router := gin.Default()
a2a.RegisterRoutes(router)
```

📚 **了解更多：** [A2A 接口文档](/zh/api/a2a)

---

#### Workflow Session State 管理

线程安全的会话管理，用于在工作流步骤间维护状态。

**核心特性：**
- 跨步骤持久化状态存储
- 使用 `sync.RWMutex` 的线程安全
- 并行分支隔离的深拷贝
- 智能合并策略防止数据丢失
- 修复 Python Agno v2.1.2 的竞态条件

**快速示例：**
```go
// 创建带会话信息的上下文
execCtx := workflow.NewExecutionContextWithSession(
    "input",
    "session-123",  // Session ID
    "user-a",       // User ID
)

// 访问会话状态
execCtx.SetSessionState("key", "value")
value, _ := execCtx.GetSessionState("key")
```

📚 **了解更多：** [Session State 文档](/zh/guide/session-state)

---

#### 多租户支持

用单个 Agent 实例服务多个用户，同时确保数据完全隔离。

**核心特性：**
- 用户隔离的对话历史
- Memory 接口支持可选的 `userID` 参数
- 向后兼容现有代码
- 线程安全的并发操作
- 用于清理的 `ClearAll()` 方法

**快速示例：**
```go
// 创建多租户 Agent
agent, _ := agent.New(&agent.Config{
    Name:   "customer-service",
    Model:  model,
    Memory: memory.NewInMemory(100),
})

// User A 的对话
agent.UserID = "user-a"
output, _ := agent.Run(ctx, "我叫 Alice")

// User B 的对话
agent.UserID = "user-b"
output, _ := agent.Run(ctx, "我叫 Bob")
```

📚 **了解更多：** [多租户文档](/zh/advanced/multi-tenant)

---

#### 模型超时配置

为 LLM 调用配置请求超时，提供细粒度控制。

**核心特性：**
- 默认值：60 秒
- 范围：1 秒到 10 分钟
- 支持的模型：OpenAI、Anthropic Claude
- 上下文感知的超时处理

**快速示例：**
```go
// OpenAI 自定义超时
model, _ := openai.New("gpt-4", openai.Config{
    APIKey:  apiKey,
    Timeout: 30 * time.Second,
})

// Claude 自定义超时
claude, _ := anthropic.New("claude-3-opus", anthropic.Config{
    APIKey:  apiKey,
    Timeout: 45 * time.Second,
})
```

📚 **了解更多：** [模型配置](/zh/guide/models#timeout-配置)

---

### 🐛 Bug 修复

- **Workflow 竞态条件** - 修复并行步骤执行的数据竞争
  - Python Agno v2.1.2 有共享的 `session_state` 字典导致覆盖
  - Go 实现为每个分支使用独立的 SessionState 克隆
  - 智能合并策略防止并发执行中的数据丢失

---

### 📚 文档

所有新功能都包含完整的双语文档（English/中文）：

- [A2A 接口指南](/zh/api/a2a) - 完整协议规范
- [Session State 指南](/zh/guide/session-state) - 工作流状态管理
- [多租户指南](/zh/advanced/multi-tenant) - 数据隔离模式
- [模型配置](/zh/guide/models#timeout-配置) - 超时设置

---

### 🧪 测试

**新测试套件：**
- `session_state_test.go` - 543 行会话状态测试
- `memory_test.go` - 多租户内存测试（4 个新测试用例）
- `agent_test.go` - 多租户 Agent 测试
- `openai_test.go` - 超时配置测试
- `anthropic_test.go` - 超时配置测试

**测试结果：**
- ✅ 所有测试通过 `-race` 检测器
- ✅ Workflow 覆盖率：79.4%
- ✅ Memory 覆盖率：93.1%
- ✅ Agent 覆盖率：74.7%

---

### 📊 性能

**无性能回归** - 所有基准测试保持一致：
- Agent 实例化：~180ns/op（比 Python 快 16 倍）
- 内存占用：~1.2KB/agent
- 线程安全的并发操作

---

### ⚠️ 破坏性变更

**无。** 此版本与 v1.0.x 完全向后兼容。

---

### 🔄 迁移指南

**无需迁移** - 所有新功能都是附加的且向后兼容。

**可选增强：**

1. **启用多租户支持：**
   ```go
   // 在 Agent 配置中添加 UserID
   agent := agent.New(agent.Config{
       UserID: "user-123",  // 新增
       Memory: memory.NewInMemory(100),
   })
   ```

2. **在 Workflow 中使用 Session State：**
   ```go
   // 创建带会话的上下文
   ctx := workflow.NewExecutionContextWithSession(
       "input",
       "session-id",
       "user-id",
   )
   ```

3. **配置模型超时：**
   ```go
   // 在模型配置中添加 Timeout
   model, _ := openai.New("gpt-4", openai.Config{
       APIKey:  apiKey,
       Timeout: 30 * time.Second,  // 新增
   })
   ```

---

### 📦 安装

```bash
go get github.com/rexleimo/agno-go@v1.1.0
```

---

### 🔗 链接

- **GitHub Release:** [v1.1.0](https://github.com/rexleimo/agno-go/releases/tag/v1.1.0)
- **完整变更日志：** [CHANGELOG.md](https://github.com/rexleimo/agno-go/blob/main/CHANGELOG.md)
- **文档：** [https://agno-go.dev](https://agno-go.dev)

---

## Version 1.0.3 (2025-10-06)

### 🧪 改进

#### 测试与质量
- **增强 JSON 序列化测试** - utils/serialize 包达到 100% 测试覆盖率
- **性能基准测试** - 与 Python Agno 性能测试模式对齐
- **全面文档** - 添加双语包文档

#### 性能
- **ToJSON**: ~600ns/op, 760B/op, 15 allocs/op
- **ConvertValue**: ~180ns/op, 392B/op, 5 allocs/op
- **Agent Creation**: ~180ns/op（比 Python 快 16 倍）

---

## Version 1.0.2 (2025-10-05)

### ✨ 新增

#### GLM (智谱AI) 提供商
- 完整集成智谱 AI 的 GLM 模型
- 支持 GLM-4、GLM-4V（视觉）、GLM-3-Turbo
- 自定义 JWT 认证（HMAC-SHA256）
- 同步和流式 API 调用
- 工具/函数调用支持
- 测试覆盖率：57.2%

**快速示例：**
```go
model, _ := glm.New("glm-4", glm.Config{
    APIKey:      os.Getenv("ZHIPUAI_API_KEY"),
    Temperature: 0.7,
})
```

---

## Version 1.0.0 (2025-10-02)

### 🎉 初始版本

Agno-Go v1.0 是 Agno 多智能体框架的高性能 Go 实现。

#### 核心功能
- **Agent** - 带工具支持的单个自主 Agent（74.7% 覆盖率）
- **Team** - 4 种模式的多 Agent 协作（92.3% 覆盖率）
- **Workflow** - 5 种原语的基于步骤的编排（80.4% 覆盖率）

#### LLM 提供商
- OpenAI（GPT-4、GPT-3.5、GPT-4 Turbo）
- Anthropic（Claude 3.5 Sonnet、Claude 3 Opus/Sonnet/Haiku）
- Ollama（本地模型）

#### 工具与存储
- Calculator、HTTP、File 工具
- 内存对话存储（93.1% 覆盖率）
- 会话管理
- ChromaDB 向量数据库

#### 性能
- Agent 创建：~180ns/op（比 Python 快 16 倍）
- 内存占用：~1.2KB/agent
- 测试覆盖率：平均 80.8%

---

## 之前的版本

完整版本历史请参见 [CHANGELOG.md](https://github.com/rexleimo/agno-go/blob/main/CHANGELOG.md)。

---

**最后更新：** 2025-10-08
