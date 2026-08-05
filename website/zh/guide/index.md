# 什么是 HNO?

**HNO** 是一个使用 Go 语言构建的多智能体系统框架。它使用 Go 的并发模型、静态类型、部署方式和标准工具；只有在存在可复现 benchmark 时，文档才发布特定工作负载的性能结论。

## 核心特性

### 🚀 可复现的性能测量

- Agent 创建 benchmark 和完整环境记录在[性能页面](/zh/advanced/performance)。
- benchmark 使用本地 `MockModel`，测量的是框架分配，不是 LLM 延迟或服务吞吐量。
- 没有同工作负载的公平测试，就不发布 Go 与 Python 的倍数加速或内存比例。
- **原生并发**: 应用层可以使用 Go Goroutine。

### 🤖 AgentOS HTTP 服务器

HNO 包含 **AgentOS**,一个可部署的 HTTP 服务器:

- 符合 OpenAPI 3.0 规范的 RESTful API
- 多轮对话的会话管理
- 线程安全的 Agent 注册表
- 健康监控和结构化日志
- CORS 支持和请求超时处理

### 🧩 灵活架构

三种核心抽象适用于不同场景:

1. **Agent** - 具有工具支持和记忆的自主 AI Agent
2. **Team** - 4 种协作模式的多 Agent 协作
   - Sequential(顺序)、Parallel(并行)、Leader-Follower(领导-跟随)、Consensus(共识)
3. **Workflow** - 基于 5 种原语的步骤式编排
   - Step、Condition、Loop、Parallel、Router

### 🔌 多模型支持

内置支持多个 LLM 提供商:

- **OpenAI** - GPT-4、GPT-3.5 Turbo 等
- **Anthropic** - Claude 3.5 Sonnet、Claude 3 Opus/Sonnet/Haiku
- **Ollama** - 本地模型 (Llama 3、Mistral、CodeLlama 等)
- **DeepSeek** - DeepSeek-V2、DeepSeek-Coder
- **Google Gemini** - Gemini Pro、Flash
- **ModelScope** - Qwen、Yi 模型

### 🔧 可扩展工具

遵循 KISS 原则,提供高质量的基础工具:

- **Calculator** - 基础数学运算
- **HTTP** - 发起 HTTP GET/POST 请求
- **File Operations** - 带安全控制的读、写、列表、删除
- **Search** - DuckDuckGo 网页搜索

轻松创建自定义工具 - 查看 [Tools Guide](/guide/tools)。

### 💾 RAG 与知识库

构建具有知识库的智能 Agent:

- **ChromaDB** - 向量数据库集成
- **OpenAI Embeddings** - 支持 text-embedding-3-small/large
- 自动生成嵌入和语义搜索

查看 [RAG Demo](/examples/rag-demo) 获取完整示例。

## 设计理念

### KISS 原则

**Keep It Simple, Stupid** - 专注于质量而非数量:

- 可检查的小型核心
- 基础工具
- 可插拔的存储集成

这种聚焦的方法旨在实现:
- 更好的代码质量
- 更易于维护
- 可部署的服务器能力；是否适合生产取决于具体工作负载

### Go 语言优势

为什么使用 Go 构建多智能体系统?

1. **性能** - 编译型语言,快速执行
2. **并发** - 原生 Goroutine,无 GIL
3. **类型安全** - 在编译时捕获错误
4. **单一二进制** - 易于部署,无运行时依赖
5. **优秀工具** - 内置测试、性能分析、竞态检测

## 使用场景

HNO 非常适合:

- **生产 AI 应用** - 使用 AgentOS HTTP 服务器部署
- **多智能体系统** - 协调多个 AI Agent
- **应用工作流** - 编排多步骤 Agent 任务
- **本地 AI 开发** - 使用 Ollama 实现隐私优先的应用
- **RAG 应用** - 构建基于知识库的 AI 助手

## 质量指标

- **测试状态**: 执行 `go test ./...` 查看当前结果
- **Benchmark 状态**: 查看[性能页面](/zh/advanced/performance)中的可复现快照
- **文档**: 使用 VitePress 构建指南、API 参考和示例
- **部署**: 提供 Docker 和部署材料；是否适合生产取决于具体工作负载

## 下一步

准备开始了吗?

1. [Quick Start](/guide/quick-start) - 5 分钟内构建您的第一个 Agent
2. [Installation](/guide/installation) - 详细的设置说明
3. [Core Concepts](/guide/agent) - 了解 Agent、Team、Workflow

## 快速入口

- 嵌入（Embeddings）：[OpenAI/VLLM 使用](/zh/guide/embeddings)
- 向量索引：[Chroma + Redis（可选）+ 迁移 CLI](/zh/advanced/vector-indexing)

## 社区

- **GitHub**: [rexleimo/HNO](https://github.com/rexleimo/HNO)
- **Issues**: [报告 Bug](https://github.com/rexleimo/HNO/issues)
- **Discussions**: [提问题](https://github.com/rexleimo/HNO/discussions)

## 许可证

HNO 使用 [MIT License](https://github.com/rexleimo/HNO/blob/main/LICENSE) 发布。

灵感来源于 [Agno Python](https://github.com/agno-agi/agno) 项目。HNO 是这个 Go 项目当前使用的名称；仓库没有定义 HNO 的正式全称。
