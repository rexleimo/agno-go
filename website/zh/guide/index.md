# 什么是 HNO?

**HNO** 是一个使用 Go 语言构建的高性能多智能体系统框架。它继承了 Python Agno 框架的设计理念,利用 Go 的并发模型和性能优势来构建高效、可扩展的 AI Agent 系统。

## 核心特性

### 🚀 极致性能

- **Agent 实例化**: 平均约 180ns (比 Python 版本快 16 倍)
- **内存占用**: 每个 Agent 约 1.2KB (比 Python 少 5.4 倍)
- **原生并发**: 完整支持 Goroutine,无 GIL 限制

### 🤖 生产就绪

HNO 包含 **AgentOS**,一个生产级 HTTP 服务器:

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

内置支持 6 个主流 LLM 提供商:

- **OpenAI** - GPT-4、GPT-3.5 Turbo 等
- **Anthropic** - Claude 3.5 Sonnet、Claude 3 Opus/Sonnet/Haiku
- **Ollama** - 本地模型 (Llama 3、Mistral、CodeLlama 等)
- **DeepSeek** - DeepSeek-V2、DeepSeek-Coder
- **Google Gemini** - Gemini Pro、Flash
- **ModelScope** - Qwen、Yi 模型

### 🔧 可扩展工具

遵循 KISS 原则,提供高质量的基础工具:

- **Calculator** - 基础数学运算 (75.6% 测试覆盖率)
- **HTTP** - 发起 HTTP GET/POST 请求 (88.9% 覆盖率)
- **File Operations** - 带安全控制的读、写、列表、删除 (76.2% 覆盖率)
- **Search** - DuckDuckGo 网页搜索 (92.1% 覆盖率)

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

- **3 个核心 LLM 提供商** (而非 45+)
- **基础工具** (而非 115+)
- **1 个向量数据库** (而非 15+)

这种聚焦的方法确保:
- 更好的代码质量
- 更易于维护
- 生产就绪的特性

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
- **高性能工作流** - 处理数千个请求
- **本地 AI 开发** - 使用 Ollama 实现隐私优先的应用
- **RAG 应用** - 构建基于知识库的 AI 助手

## 质量指标

- **测试覆盖率**: 核心包平均 80.8%
- **测试用例**: 85+ 个测试,100% 通过率
- **文档**: 完整的指南、API 参考、示例
- **生产就绪**: Docker、K8s 清单、部署指南

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

灵感来源于 [Agno (Python)](https://github.com/agno-agi/agno) 框架。
