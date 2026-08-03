---
layout: home

hero:
  name: "Agno-Go"
  text: "高性能多智能体框架"
  tagline: "比 Python 快 16 倍 | 180ns 实例化 | 每个智能体仅 1.2KB 内存"
  image:
    src: /logo.png
    alt: Agno-Go
  actions:
    - theme: brand
      text: 快速开始
      link: /zh/guide/quick-start
    - theme: alt
      text: 在 GitHub 上查看
      link: https://github.com/rexleimo/agno-Go

features:
  - icon: 🚀
    title: 极致性能
    details: 智能体实例化仅需 ~180ns, 每个智能体约 1.2KB 内存, 相比 Python 运行时快 16 倍。

  - icon: 🤖
    title: 生产级 AgentOS
    details: 内置 OpenAPI 3.0、会话存储、健康检查、结构化日志、CORS、请求超时, 并补齐摘要、复用与历史筛选等对等端点。

  - icon: 🪄
    title: 会话对齐
    details: 会话可在 Agent / Team 间共享, 支持同步/异步摘要, 记录缓存命中与取消原因, 并复用 Python 上的 `stream_events` 开关。

  - icon: 🧩
    title: 灵活架构
    details: 自由组合 Agent、Team（4 种协作模式）与 Workflow（5 种控制原语）, 继承默认配置并支持检点恢复与确定性编排。

  - icon: 🔌
    title: 多模型供应商
    details: 开箱支持 OpenAI o-series、Anthropic Claude、Google Gemini、DeepSeek、GLM、ModelScope、Ollama、Cohere、Groq、Together、OpenRouter、LM Studio、Vercel、Portkey、InternLM、SambaNova。

  - icon: 🔧
    title: 可扩展工具
    details: 内置计算器、HTTP、文件、搜索, 并新增 Claude Agent Skills、Tavily Reader/Search、Gmail 标记已读、Jira 工时、ElevenLabs 语音、PPTX 阅读器及 MCP 连接器。

  - icon: 💾
    title: 知识与缓存
    details: 集成 ChromaDB、批量导入工具与摄取助手, 提供响应缓存以去重相同的模型调用。

  - icon: 🛡️
    title: 防护与可观测性
    details: 提供提示注入防护、自定义前后置钩子、媒体校验、SSE 推理流以及 Logfire / OpenTelemetry 链路追踪示例。

  - icon: 📦
    title: 易于部署
    details: 提供单一二进制、Docker、Compose 与 Kubernetes 清单, 配套上线指南可快速落地。
---

## 快速示例

仅需几行代码即可创建带工具的 AI 智能体:

```go
package main

import (
    "context"
    "fmt"
    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
)

func main() {
    // 创建模型
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: "your-api-key",
    })

    // 创建带工具的智能体
    ag, _ := agent.New(agent.Config{
        Name:     "数学助手",
        Model:    model,
        Toolkits: []toolkit.Toolkit{calculator.New()},
    })

    // 运行智能体
    output, _ := ag.Run(context.Background(), "25 * 4 + 15 等于多少?")
    fmt.Println(output.Content) // 输出: 115
}
```

## 性能对比

| 指标 | Python Agno | Agno-Go | 改进 |
|--------|-------------|---------|-------------|
| 智能体创建 | ~3μs | ~180ns | **快 16 倍** |
| 内存/智能体 | ~6.5KB | ~1.2KB | **减少 5.4 倍** |
| 并发性 | GIL 限制 | 原生 goroutine | **无限制** |

## 为什么选择 Agno-Go?
### v1.2.9 有哪些更新 / What's New in v1.2.9

- **EvoLink 媒体智能体** – 在 `pkg/hno/providers/evolink` 与 `pkg/hno/models/evolink/*` 下提供文本、图片、视频模型, 并在 EvoLink 媒体示例页中给出端到端工作流。 / First-class EvoLink provider for text, image, and video with end-to-end examples in the EvoLink Media Agents docs.
- **知识上传分块** – `POST /api/v1/knowledge/content` 支持 `chunk_size`、`chunk_overlap`(JSON、`text/plain` query 参数与 multipart 表单), 并在分块 metadata 中记录这两个值以及 `chunker_type`, 与 Python AgentOS 对齐。 / `POST /api/v1/knowledge/content` now supports `chunk_size` and `chunk_overlap` across JSON, `text/plain` query params, and multipart uploads, and records these values plus `chunker_type` in chunk metadata.
- **AgentOS HTTP 提示** – 文档新增如何自定义健康检查路径、使用 `/openapi.yaml` 与 `/docs`, 以及在变更路由后调用 `server.Resync()` 的最佳实践。 / Docs now explain how to customize health endpoints, rely on `/openapi.yaml` and `/docs`, and when to call `server.Resync()` after router changes.

### 为生产而生

Agno-Go 不仅是一个框架——它是一个完整的生产系统。包含的 **AgentOS** 服务器提供:

- 带 OpenAPI 3.0 规范的 RESTful API
- 多轮对话的会话管理
- 线程安全的智能体注册表
- 健康监控和结构化日志
- CORS 支持和请求超时处理

### KISS 原则

遵循 **Keep It Simple, Stupid** 哲学:

- **3 个核心 LLM 提供商**(而非 45+) - OpenAI、Anthropic、Ollama
- **基础工具**(而非 115+) - 计算器、HTTP、文件、搜索
- **质量优于数量** - 专注于生产就绪的功能

### 开发者体验

- **类型安全**: Go 的强类型在编译时捕获错误
- **快速构建**: Go 的编译速度支持快速迭代
- **易于部署**: 单一二进制文件,无运行时依赖
- **优秀工具**: 内置测试、性能分析和竞态检测

## 5 分钟快速开始

```bash
# 克隆仓库
git clone https://github.com/rexleimo/agno-Go.git
cd agno-Go

# 设置 API 密钥
export OPENAI_API_KEY=sk-your-key-here

# 运行示例
go run cmd/examples/simple_agent/main.go

# 或启动 AgentOS 服务器
docker-compose up -d
curl http://localhost:8080/health
```

## 包含内容

- **核心框架**: Agent、Team(4 种模式)、Workflow(5 种原语)
- **模型**: OpenAI、Anthropic Claude、Ollama、DeepSeek、Gemini、ModelScope
- **工具**: Calculator(75.6%)、HTTP(88.9%)、File(76.2%)、Search(92.1%)
- **RAG**: ChromaDB 集成 + OpenAI 嵌入
- **AgentOS**: 生产级 HTTP 服务器(65.0% 覆盖率)
- **示例**: 6 个涵盖所有功能的实际示例
- **文档**: 完整指南、API 参考、部署说明

## 社区

- **GitHub**: [rexleimo/agno-Go](https://github.com/rexleimo/agno-Go)
- **Issues**: [报告问题和请求功能](https://github.com/rexleimo/agno-Go/issues)
- **Discussions**: [提问和分享想法](https://github.com/rexleimo/agno-Go/discussions)

## 许可证

Agno-Go 基于 [MIT 许可证](https://github.com/rexleimo/agno-Go/blob/main/LICENSE) 发布。

灵感来自 [Agno (Python)](https://github.com/agno-agi/agno) 框架。
