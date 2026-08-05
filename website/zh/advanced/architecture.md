# 架构 / Architecture

HNO 遵循简洁、模块化的架构设计,专注于简单性、效率和可扩展性。

## 核心理念 / Core Philosophy

**简单、高效、可扩展 / Simple, Efficient, Scalable**

## 整体架构 / Overall Architecture

```
┌─────────────────────────────────────────┐
│          Application Layer              │
│  (CLI Tools, Web API, Custom Apps)      │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│         Core Abstractions               │
│  ┌─────────┐  ┌──────┐  ┌──────────┐   │
│  │  Agent  │  │ Team │  │ Workflow │   │
│  └─────────┘  └──────┘  └──────────┘   │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│        Foundation Layer                  │
│  ┌────────┐ ┌───────┐ ┌──────┐         │
│  │ Models │ │ Tools │ │Memory│ ...     │
│  └────────┘ └───────┘ └──────┘         │
└─────────────────────────────────────────┘
```

## 核心接口 / Core Interfaces

### 1. Model 接口 / Model Interface

```go
type Model interface {
    // Synchronous invocation
    Invoke(ctx context.Context, req *InvokeRequest) (*ModelResponse, error)

    // Streaming invocation
    InvokeStream(ctx context.Context, req *InvokeRequest) (<-chan ResponseChunk, error)

    // Metadata
    GetProvider() string
    GetID() string
}
```

### 2. Toolkit 接口 / Toolkit Interface

```go
type Toolkit interface {
    Name() string
    Functions() map[string]*Function
}

type Function struct {
    Name        string
    Description string
    Parameters  map[string]Parameter
    Handler     func(context.Context, map[string]interface{}) (interface{}, error)
}
```

### 3. Memory 接口 / Memory Interface

```go
type Memory interface {
    Add(message types.Message) error
    GetMessages() []types.Message
    Clear() error
}
```

## 组件详情 / Component Details

### Agent

**文件 / File**: `pkg/hno/agent/agent.go`

自主 AI 实体,具备以下能力:
- 使用 LLM 进行推理
- 调用工具
- 维护对话记忆
- 通过钩子验证输入/输出

**关键方法 / Key Methods**:
```go
New(config Config) (*Agent, error)
Run(ctx context.Context, input string) (*RunOutput, error)
ClearMemory()
```

### Team

**文件 / File**: `pkg/hno/team/team.go`

多智能体协作,支持 4 种协作模式:

1. **Sequential** - 智能体按顺序工作
2. **Parallel** - 所有智能体同时工作
3. **LeaderFollower** - 领导者委派任务给跟随者
4. **Consensus** - 智能体讨论直到达成一致

### Workflow

**文件 / File**: `pkg/hno/workflow/workflow.go`

基于步骤的编排,支持 5 种原语:

1. **Step** - 执行智能体或函数
2. **Condition** - 基于上下文的分支
3. **Loop** - 带退出条件的迭代
4. **Parallel** - 并发执行步骤
5. **Router** - 动态路由

### Models

**目录 / Directory**: `pkg/hno/models/`

LLM 提供商实现:
- `openai/` - OpenAI GPT 模型
- `anthropic/` - Anthropic Claude 模型
- `ollama/` - Ollama 本地模型
- `deepseek/`, `gemini/`, `modelscope/` - 其他提供商

### Tools

**目录 / Directory**: `pkg/hno/tools/`

可扩展的工具系统:
- `calculator/` - 数学运算
- `http/` - HTTP 请求
- `file/` - 文件操作
- `search/` - 网络搜索

## AgentOS 生产服务器 / AgentOS Production Server

**目录 / Directory**: `pkg/agentos/`

HTTP 服务器能力包括:

- RESTful API 端点
- 会话管理
- Agent 注册表
- 健康监控
- CORS 支持
- 请求超时处理

**架构 / Architecture**:
```
┌─────────────────────┐
│   HTTP Handlers     │
│  (API Endpoints)    │
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│  Agent Registry     │
│  (Thread-safe map)  │
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│ Session Manager     │
│  (In-memory store)  │
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│  Agent Instances    │
│  (Runtime agents)   │
└─────────────────────┘
```

## 设计模式 / Design Patterns

### 1. 基于接口的设计 / Interface-Based Design

所有核心组件使用接口以实现灵活性:

```go
type Model interface { /* ... */ }
type Toolkit interface { /* ... */ }
type Memory interface { /* ... */ }
```

### 2. 组合优于继承 / Composition Over Inheritance

Agent 组合了模型、工具和记忆:

```go
type Agent struct {
    Model    Model
    Toolkits []Toolkit
    Memory   Memory
    // ...
}
```

### 3. 上下文传播 / Context Propagation

所有操作接受 `context.Context` 以支持取消和超时:

```go
func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error)
```

### 4. 错误包装 / Error Wrapping

一致的错误处理和错误包装:

```go
if err != nil {
    return nil, fmt.Errorf("failed to run agent: %w", err)
}
```

## 性能优化 / Performance Optimizations

### 1. 低分配次数 / Low Allocation Count

- 最小化堆分配(每个 agent 8-9 次)
- 预分配切片
- 适当的字符串驻留

### 2. 高效的内存布局 / Efficient Memory Layout

```go
type Agent struct {
    ID           string   // 16B
    Name         string   // 16B
    Model        Model    // 16B (interface)
    // Total: ~112B struct + heap allocations
}
```

### 3. Goroutine 安全性 / Goroutine Safety

- 无全局状态
- 设计上线程安全
- 尽可能无锁

## 并发模型 / Concurrency Model

### Agent 并发 / Agent Concurrency

```go
// Safe to create multiple agents concurrently
for i := 0; i < 100; i++ {
    go func() {
        ag, _ := agent.New(config)
        output, _ := ag.Run(ctx, input)
    }()
}
```

### Team 并行模式 / Team Parallel Mode

```go
// Agents run in parallel goroutines
team := team.New(team.Config{
    Mode: team.ModeParallel,
    Agents: agents,
})
```

### Workflow 并行步骤 / Workflow Parallel Step

```go
// Steps execute concurrently
workflow.NewParallel("tasks", []Primitive{
    step1, step2, step3,
})
```

## 扩展点 / Extensibility Points

### 1. 自定义模型 / Custom Models

实现 `Model` 接口:

```go
type MyModel struct{}

func (m *MyModel) Invoke(ctx context.Context, req *InvokeRequest) (*ModelResponse, error) {
    // Custom implementation
}
```

### 2. 自定义工具 / Custom Tools

扩展 `BaseToolkit`:

```go
type MyToolkit struct {
    *toolkit.BaseToolkit
}

func (t *MyToolkit) RegisterFunctions() {
    t.RegisterFunction(&Function{
        Name: "my_function",
        Handler: t.myHandler,
    })
}
```

### 3. 自定义记忆 / Custom Memory

实现 `Memory` 接口:

```go
type MyMemory struct{}

func (m *MyMemory) Add(msg types.Message) error {
    // Custom storage
}
```

## 测试策略 / Testing Strategy

### 单元测试 / Unit Tests

- 每个包都有 `*_test.go` 文件
- 接口的模拟实现
- 表驱动测试

### 集成测试 / Integration Tests

- 端到端工作流测试
- 多智能体场景
- 真实 API 集成测试

### 基准测试 / Benchmark Tests

- `*_bench_test.go` 中的性能基准
- 内存分配跟踪
- 并发压力测试

## 依赖关系 / Dependencies

### 核心依赖 / Core Dependencies

- **Go 标准库 / Go Standard Library** - 大部分功能
- **无重型框架 / No heavy frameworks** - 轻量级设计

### 可选依赖 / Optional Dependencies

- LLM 提供商 SDK (OpenAI, Anthropic 等)
- 向量数据库客户端 (ChromaDB)
- HTTP 客户端库

## 未来架构 / Future Architecture

### 计划的增强功能 / Planned Enhancements

1. **流式支持 / Streaming Support** - 实时响应流
2. **插件系统 / Plugin System** - 动态工具加载
3. **分布式 Agent / Distributed Agents** - 多节点部署
4. **高级记忆 / Advanced Memory** - 持久化存储、向量记忆

## 最佳实践 / Best Practices

### 1. 使用接口 / Use Interfaces

```go
var model models.Model = openai.New(...)
```

### 2. 处理错误 / Handle Errors

```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### 3. 使用上下文 / Use Contexts

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### 4. 保持简单 / Keep It Simple

遵循 KISS 原则 - 不要过度设计。

## 参考资料 / References

- [性能基准 / Performance Benchmarks](/advanced/performance)
- [部署指南 / Deployment Guide](/advanced/deployment)
- [API 参考 / API Reference](/api/)
- [源代码 / Source Code](https://github.com/rexleimo/HNO)
