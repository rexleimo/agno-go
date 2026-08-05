# 进阶主题

深入了解 HNO 的高级概念、性能优化、部署策略和测试最佳实践。

## 概览

本节涵盖了面向开发者的进阶主题:

- 🏗️ **理解架构** - 学习核心设计原则和模式
- ⚡ **测量性能** - 在目标环境复现仓库中的 Go benchmark
- 🚀 **部署到生产环境** - 生产部署最佳实践
- 🧪 **有效测试** - 全面的测试策略和工具

## 核心主题

### [架构](/zh/advanced/architecture)

了解 HNO 的模块化架构和设计理念:

- 核心接口 (Model, Toolkit, Memory)
- 抽象模式 (Agent, Team, Workflow)
- Go 并发模型集成
- 错误处理策略
- 包组织结构

**关键概念**: 清晰架构、依赖注入、接口设计

### [性能](/zh/advanced/performance)

理解已测量的性能特征和 benchmark 方法:

- 带环境信息的可复现 Agent 创建 benchmark
- 并发和并行
- 基准测试工具和方法
- 明确列出尚未测量的内容

**关键指标**: `ns/op`、`B/op`、分配次数以及特定工作负载下的吞吐量

### [部署](/zh/advanced/deployment)

生产部署策略和最佳实践:

- AgentOS HTTP 服务器设置
- 容器部署 (Docker, Kubernetes)
- 配置管理
- 监控和可观测性
- 扩展策略
- 安全考虑

**关键技术**: Docker, Kubernetes, Prometheus, 分布式追踪

### [测试](/zh/advanced/testing)

多智能体系统的全面测试方法:

- 单元测试模式
- 使用 Mock 的集成测试
- 性能基准测试
- 测试覆盖率要求 (>70%)
- CI/CD 集成
- 测试工具和实用程序

**关键工具**: Go testing, testify, benchmarking, 覆盖率报告

## 快速链接

### 性能基准

```bash
# 运行可复现的 Agent 创建快照
go test -run='^$' -bench='BenchmarkAgentCreation' -benchmem -count=10 ./pkg/hno/agent/

# 生成 CPU profile
go test -bench=. -cpuprofile=cpu.out ./pkg/hno/agent/
```

[查看详细性能指标 →](/zh/advanced/performance)

### 生产部署

```bash
# 构建 AgentOS 服务器
cd pkg/agentos && go build -o agentos

# 使用 Docker 运行
docker build -t agno-go-agentos .
docker run -p 8080:8080 -e OPENAI_API_KEY=$OPENAI_API_KEY agno-go-agentos
```

[查看部署指南 →](/zh/advanced/deployment)

### 向量索引

```bash
# 创建或删除向量集合（默认 Chroma）
go run ./cmd/vectordb_migrate --action up --provider chroma --collection mycol \
  --chroma-url http://localhost:8000 --distance cosine

# Redis Provider（可选，需 -tags redis）
go run -tags redis ./cmd/vectordb_migrate --action up --provider redis \
  --collection mycol --chroma-url localhost:6379
```

[查看向量索引 →](/zh/advanced/vector-indexing)

### 测试覆盖率

各包的当前测试覆盖率:

| 包 | 覆盖率 | 状态 |
|---------|----------|--------|
| types | 100.0% | ✅ 优秀 |
| memory | 93.1% | ✅ 优秀 |
| team | 92.3% | ✅ 优秀 |
| toolkit | 91.7% | ✅ 优秀 |
| workflow | 80.4% | ✅ 良好 |
| agent | 74.7% | ✅ 良好 |

[查看测试指南 →](/zh/advanced/testing)

## 设计原则

### KISS (Keep It Simple, Stupid)

HNO 拥抱简单性:

- **专注范围**: 3 个 LLM 提供商 (OpenAI, Anthropic, Ollama) 而不是 8+
- **核心工具**: 5 个核心工具而不是 15+
- **清晰抽象**: Agent, Team, Workflow
- **最小依赖**: 优先使用标准库

### 性能优先

Go 的并发模型使得:

- 原生 goroutine 支持并行执行
- 无 GIL (全局解释器锁) 限制
- 高效的内存管理
- 编译时优化

### 生产就绪

为实际部署而构建:

- 全面的错误处理
- 上下文感知的取消
- 结构化日志
- OpenTelemetry 集成
- 健康检查和指标

## 贡献

有兴趣为 HNO 做贡献? 查看:

- [架构文档](/zh/advanced/architecture) - 理解代码库
- [测试指南](/zh/advanced/testing) - 学习测试标准
- [GitHub 仓库](https://github.com/rexleimo/HNO) - 提交 PR
- [开发指南](https://github.com/rexleimo/HNO/blob/main/CLAUDE.md) - 开发环境设置

## 其他资源

### 文档

- [Go 包文档](https://pkg.go.dev/github.com/rexleimo/HNO)
- [Python HNO 框架](https://github.com/agno-agi/agno) (灵感来源)
- [VitePress 文档源码](https://github.com/rexleimo/HNO/tree/main/website)

### 社区

- [GitHub Issues](https://github.com/rexleimo/HNO/issues)
- [GitHub Discussions](https://github.com/rexleimo/HNO/discussions)
- [发布说明](/zh/release-notes)

## 下一步

1. 📖 从 [架构](/zh/advanced/architecture) 开始理解核心设计
2. ⚡ 学习 [性能](/zh/advanced/performance) 优化技术
3. 🚀 查看生产环境的 [部署](/zh/advanced/deployment) 策略
4. 🧪 掌握 [测试](/zh/advanced/testing) 最佳实践

---

**注意**: 本节假设您已熟悉 HNO 的基本概念。如果您是新手,请从 [指南](/zh/guide/) 部分开始。
