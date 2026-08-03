# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 提供在此代码库工作时的指导。

## 项目概述

**Agno-Go** 是一个高性能的 Go 语言多智能体系统框架,继承自 Python Agno 的设计理念,利用 Go 的并发模型和性能优势构建高效、可扩展的 AI Agent 系统。

**当前状态**: Week 3-4, 70% 完成
**性能**: Agent 实例化 ~180ns, 内存占用 ~1.2KB/agent (比 Python 版本快 16 倍)
**设计原则**: KISS (Keep It Simple, Stupid) - 专注质量而非数量

## 开发环境设置

### 前置要求
- Go 1.21 或更高版本
- 推荐: golangci-lint (代码检查), goimports (格式化)

### 初始化项目

```bash
# 克隆仓库
cd agno-Go

# 下载依赖
go mod download

# 设置 API 密钥 (用于测试)
export OPENAI_API_KEY=your-api-key

# (可选) 安装开发工具
make install-tools  # 安装 golangci-lint 和 goimports
```

## 常用开发命令

### 测试

```bash
# 运行所有测试 (包含竞态检测和覆盖率)
make test

# 运行特定包的测试
go test -v ./pkg/hno/agent/...

# 生成覆盖率报告 (生成 coverage.html)
make coverage

# 运行特定测试用例
go test -v -run TestAgentRun ./pkg/hno/agent/
```

### 代码质量

```bash
# 格式化代码 (运行 gofmt 和 goimports)
make fmt

# 运行代码检查 (需要 golangci-lint)
make lint

# 运行 go vet
make vet
```

### 构建和运行

```bash
# 构建示例程序 (生成到 bin/ 目录)
make build

# 运行示例
./bin/simple_agent
# 或直接运行
go run cmd/examples/simple_agent/main.go
```

### 工具命令

```bash
# 清理构建产物
make clean

# 显示帮助信息
make help
```

## 项目架构

### 核心抽象模式

Agno-Go 遵循两种主要设计模式:

1. **Agent/Team** - 用于自主式多智能体系统,智能体独立运作,最小化人工干预
   - `agent.Agent` - 单个智能体
   - `team.Team` - 多智能体协作 (4 种协作模式)

2. **Workflow** - 用于可控的、基于步骤的流程,完全掌控执行流
   - `workflow.Workflow` - 使用 5 种原语 (Step, Condition, Loop, Parallel, Router)

### 核心模块

**源码根目录**: `pkg/hno/`

#### 1. Agent (pkg/hno/agent/)
- **agent.go** - Agent 结构体和 Run 方法
- **agent_test.go** - 单元测试 (74.7% 覆盖)
- **agent_bench_test.go** - 性能基准测试

**配置选项** (agent.Config):
```go
type Config struct {
    Name         string            // Agent 名称
    Model        models.Model      // LLM 模型
    Toolkits     []toolkit.Toolkit // 工具集
    Memory       memory.Memory     // 对话记忆
    Instructions string            // 系统指令
    MaxLoops     int               // 最大工具调用循环次数
}
```

#### 2. Team (pkg/hno/team/)
多智能体协作,支持 4 种协作模式:

- `ModeSequential` - 顺序执行,智能体逐个工作
- `ModeParallel` - 并行执行,所有智能体同时工作
- `ModeLeaderFollower` - 领导者分配任务给跟随者
- `ModeConsensus` - 智能体讨论直到达成共识

**测试覆盖**: 92.3%

#### 3. Workflow (pkg/hno/workflow/)
基于步骤的工作流引擎,支持 5 种原语:

- **step.go** - 基本工作流步骤 (运行 Agent 或自定义函数)
- **condition.go** - 基于上下文的条件分支
- **loop.go** - 带退出条件的迭代循环
- **parallel.go** - 多步骤并行执行
- **router.go** - 动态路由到不同步骤
- **workflow.go** - 主工作流协调器

**测试覆盖**: 80.4%

#### 4. Models (pkg/hno/models/)
LLM 提供商接口和实现:

- **base.go** - Model 接口 (Invoke/InvokeStream 方法)
- **openai/openai.go** - OpenAI 实现 (GPT-4, GPT-3.5, 等)
- **anthropic/anthropic.go** - Anthropic Claude 实现
- **groq/groq.go** - Groq 超快速推理实现 (LLaMA 3.1, Mixtral, Gemma) ⭐ NEW
- **glm/glm.go** - 智谱AI GLM 实现 (GLM-4, GLM-4V, GLM-3-Turbo)
- **ollama/ollama.go** - Ollama 本地模型实现

**Model 接口**:
```go
type Model interface {
    Invoke(ctx context.Context, req *InvokeRequest) (*types.ModelResponse, error)
    InvokeStream(ctx context.Context, req *InvokeRequest) (<-chan types.ResponseChunk, error)
    GetProvider() string
    GetID() string
}
```

#### 5. Tools (pkg/hno/tools/)
工具系统,扩展 Agent 能力:

- **toolkit/toolkit.go** - Toolkit 接口和基础实现
- **calculator/calculator.go** - 基础数学运算 (add, subtract, multiply, divide)
- **http/http.go** - HTTP GET/POST 请求
- **file/file.go** - 文件操作 (读、写、列表、删除,带安全控制)

#### 6. Memory (pkg/hno/memory/)
对话历史管理:

- **memory.go** - 内存存储,支持自动截断
- 可配置消息限制 (默认: 100 条消息)

**测试覆盖**: 93.1%

#### 7. Types (pkg/hno/types/)
核心类型和错误:

- **message.go** - 消息类型 (System, User, Assistant, Tool)
- **response.go** - 模型响应结构
- **errors.go** - 自定义错误类型 (InvalidConfigError, InvalidInputError, 等)

**测试覆盖**: 100% ⭐

### 示例程序

**位置**: `cmd/examples/`

- **simple_agent/** - 基础 Agent,使用计算器工具
- **claude_agent/** - Anthropic Claude 集成示例
- **groq_agent/** - Groq 超快速推理示例 (LLaMA 3.1 8B) ⭐ NEW
- **glm_agent/** - 智谱AI GLM 集成示例 (支持中文对话)
- **ollama_agent/** - 本地模型支持示例
- **team_demo/** - 多智能体协作演示
- **workflow_demo/** - 工作流引擎演示

## 性能设计

Agno-Go 利用 Go 的并发模型实现卓越性能:

- **Agent 实例化**: ~180ns 平均 (目标: <1μs, 超越 5 倍)
- **内存占用**: ~1.2KB/agent 平均 (目标: <3KB, 比目标低 60%)
- **原生 Goroutine**: 支持并行执行,无 GIL 限制

**详细性能报告**: 查看 [website/advanced/performance.md](website/advanced/performance.md)

## 添加新组件

### 添加模型提供商

1. 创建目录: `pkg/hno/models/<your_model>/`
2. 实现 `models.Model` 接口 (来自 `models/base.go`):
   - `Invoke(ctx context.Context, req *InvokeRequest) (*ModelResponse, error)`
   - `InvokeStream(ctx context.Context, req *InvokeRequest) (<-chan ResponseChunk, error)`
   - `GetProvider() string` 和 `GetID() string`
3. 参考 `models/openai/openai.go` 作为参考实现
4. 在 `<your_model>_test.go` 中添加单元测试
5. 格式化和验证: `make fmt && make test`

**示例结构**:
```go
type YourModel struct {
    models.BaseModel
    config     Config
    httpClient *http.Client
}

func New(modelID string, config Config) (*YourModel, error) {
    // 初始化逻辑
}

func (m *YourModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
    // 实现逻辑
}
```

### 添加工具

1. 创建目录: `pkg/hno/tools/<your_tool>/`
2. 创建嵌入 `toolkit.BaseToolkit` 的结构体
3. 使用 `RegisterFunction` 注册函数,提供正确的参数定义
4. 参考 `tools/calculator/calculator.go` 或 `tools/http/http.go` 作为示例
5. 在 `<your_tool>_test.go` 中添加单元测试
6. 格式化和验证: `make fmt && make test`

**示例结构**:
```go
type MyToolkit struct {
    *toolkit.BaseToolkit
}

func New() *MyToolkit {
    t := &MyToolkit{
        BaseToolkit: toolkit.NewBaseToolkit("my_tools"),
    }

    t.RegisterFunction(&toolkit.Function{
        Name:        "my_function",
        Description: "执行某个有用的操作",
        Parameters: map[string]toolkit.Parameter{
            "input": {
                Type:        "string",
                Description: "输入参数",
                Required:    true,
            },
        },
        Handler: t.myHandler,
    })

    return t
}

func (t *MyToolkit) myHandler(args map[string]interface{}) (interface{}, error) {
    input := args["input"].(string)
    // 实现逻辑
    return result, nil
}
```

## 代码风格指南

### 函数文档

```go
// New 创建一个新的 Agent,使用给定的配置。
// 如果未提供 Model 或配置无效,返回错误。
func New(config *Config) (*Agent, error) {
    // ...
}
```

### 错误处理

```go
if err != nil {
    return nil, fmt.Errorf("failed to create agent: %w", err)
}
```

### Context 使用

```go
func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // 实现逻辑
    }
}
```

### 提交前检查

1. 运行 `make fmt` 格式化代码
2. 运行 `make test` 确保测试通过
3. 运行 `make lint` (如果已安装 golangci-lint)
4. 确保测试覆盖率保持 (使用 `make coverage` 检查)

## 测试标准

**目标**: 所有核心包 >70% 测试覆盖率

### 当前覆盖率状态

| 包 | 覆盖率 | 状态 |
|---|---|---|
| types | 100.0% | ✅ 优秀 |
| memory | 93.1% | ✅ 优秀 |
| team | 92.3% | ✅ 优秀 |
| toolkit | 91.7% | ✅ 优秀 |
| http | 88.9% | ✅ 良好 |
| workflow | 80.4% | ✅ 良好 |
| file | 76.2% | ✅ 良好 |
| calculator | 75.6% | ✅ 良好 |
| agent | 74.7% | ✅ 良好 |
| groq | 52.4% | 🟡 需要改进 |
| anthropic | 50.9% | 🟡 需要改进 |
| openai | 44.6% | 🟡 需要改进 |
| ollama | 43.8% | 🟡 需要改进 |

### 编写测试

**单元测试示例**:
```go
func TestAgentRun(t *testing.T) {
    model := &MockModel{
        InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
            return &types.ModelResponse{
                Content: "test response",
            }, nil
        },
    }

    agent, err := New(Config{
        Name:  "test-agent",
        Model: model,
    })
    if err != nil {
        t.Fatalf("Failed to create agent: %v", err)
    }

    output, err := agent.Run(context.Background(), "test input")
    if err != nil {
        t.Fatalf("Run failed: %v", err)
    }

    if output.Content != "test response" {
        t.Errorf("Expected 'test response', got '%s'", output.Content)
    }
}
```

**性能基准测试示例**:
```go
func BenchmarkAgentCreation(b *testing.B) {
    model := &MockModel{}

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _, err := New(Config{
            Name:  "benchmark-agent",
            Model: model,
        })
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## 配置

### 环境变量

```bash
# OpenAI
export OPENAI_API_KEY=sk-...

# Groq (超快速推理,获取密钥: https://console.groq.com/keys)
export GROQ_API_KEY=gsk-...

# Anthropic Claude
export ANTHROPIC_API_KEY=sk-ant-...

# 智谱AI GLM (格式: {key_id}.{key_secret})
export ZHIPUAI_API_KEY=your-key-id.your-key-secret

# Ollama (本地运行,默认: http://localhost:11434)
export OLLAMA_BASE_URL=http://localhost:11434
```

## 文档和资源

- **性能基准**: [website/advanced/performance.md](website/advanced/performance.md)
- **架构文档**: [website/advanced/architecture.md](website/advanced/architecture.md)
- **开发指南**: [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)

## KISS 原则应用

我们在项目中应用 KISS (Keep It Simple, Stupid) 原则:

**简化的范围**:
- 3 个核心 LLM (不是 8 个): OpenAI, Anthropic, Ollama
- 5 个基础工具 (不是 15+): Calculator, HTTP, File, Search, (未来扩展)
- 1 个向量数据库 (不是 3 个): ChromaDB (用于验证)

**原因**:
- 更清晰的优先级
- 更好的代码质量
- 更易于维护的项目

## 快速链接

- [GitHub Issues](https://github.com/rexleimo/agno-go/issues)
- [GitHub Discussions](https://github.com/rexleimo/agno-go/discussions)
- [Python Agno 框架](https://github.com/agno-agi/agno) (灵感来源)
