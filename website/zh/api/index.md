# API 参考 / API Reference

HNO v1.0 的完整 API 参考文档。

## 核心模块 / Core Modules

- [Agent](/api/agent) - 自主式 AI 智能体 / Autonomous AI agents
- [Team](/api/team) - 多智能体协作 / Multi-agent collaboration
- [Workflow](/api/workflow) - 基于步骤的编排 / Step-based orchestration
- [Models](/api/models) - LLM 提供商集成 / LLM provider integrations
- [Tools](/api/tools) - 内置和自定义工具 / Built-in and custom tools
- [Memory](/api/memory) - 对话历史管理 / Conversation history management
- [Types](/api/types) - 核心类型和错误 / Core types and errors
- [AgentOS Server](/api/agentos) - 生产环境 HTTP 服务器 / Production HTTP server

## 快速链接 / Quick Links

### Agent

```go
import "github.com/rexleimo/HNO/pkg/hno/agent"

agent.New(config) (*Agent, error)
agent.Run(ctx, input) (*RunOutput, error)
agent.ClearMemory()
```

[完整 Agent API →](/api/agent)

### Team

```go
import "github.com/rexleimo/HNO/pkg/hno/team"

team.New(config) (*Team, error)
team.Run(ctx, input) (*RunOutput, error)

// 模式 / Modes: Sequential, Parallel, LeaderFollower, Consensus
```

[完整 Team API →](/api/team)

### Workflow

```go
import "github.com/rexleimo/HNO/pkg/hno/workflow"

workflow.New(config) (*Workflow, error)
workflow.Run(ctx, input) (*RunOutput, error)

// 原语 / Primitives: Step, Condition, Loop, Parallel, Router
```

[完整 Workflow API →](/api/workflow)

### Models

```go
import (
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
    "github.com/rexleimo/HNO/pkg/hno/models/anthropic"
    "github.com/rexleimo/HNO/pkg/hno/models/ollama"
)

openai.New(modelID, config) (*OpenAI, error)
anthropic.New(modelID, config) (*Anthropic, error)
ollama.New(modelID, config) (*Ollama, error)
```

[完整 Models API →](/api/models)

### Tools

```go
import (
    "github.com/rexleimo/HNO/pkg/hno/tools/calculator"
    "github.com/rexleimo/HNO/pkg/hno/tools/http"
    "github.com/rexleimo/HNO/pkg/hno/tools/file"
)

calculator.New() *Calculator
http.New(config) *HTTP
file.New(config) *File
```

[完整 Tools API →](/api/tools)

## 常见模式 / Common Patterns

### 错误处理 / Error Handling

```go
import "github.com/rexleimo/HNO/pkg/hno/types"

output, err := agent.Run(ctx, input)
if err != nil {
    switch {
    case errors.Is(err, types.ErrInvalidInput):
        // 处理无效输入 / Handle invalid input
    case errors.Is(err, types.ErrRateLimit):
        // 处理速率限制 / Handle rate limit
    default:
        // 处理其他错误 / Handle other errors
    }
}
```

### Context 管理 / Context Management

```go
import (
    "context"
    "time"
)

// 带超时 / With timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

output, err := agent.Run(ctx, input)
```

### 并发智能体 / Concurrent Agents

```go
var wg sync.WaitGroup

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()

        ag, _ := agent.New(config)
        output, _ := ag.Run(ctx, input)

        fmt.Printf("Agent %d: %s\n", id, output.Content)
    }(i)
}

wg.Wait()
```

## 类型定义 / Type Definitions

### 核心类型 / Core Types

```go
// 消息类型 / Message types
type Message struct {
    Role    MessageRole
    Content string
    Name    string
}

// 运行输出 / Run output
type RunOutput struct {
    Content  string
    Messages []Message
    Metadata map[string]interface{}
}

// 模型响应 / Model response
type ModelResponse struct {
    Content    string
    ToolCalls  []ToolCall
    FinishReason string
}
```

[完整 Types 参考 →](/api/types)

## AgentOS Server API

生产环境部署的 REST API 端点 / REST API endpoints for production deployment:

```bash
# 健康检查 / Health check
GET /health

# 列出智能体 / List agents
GET /api/v1/agents

# 运行智能体 / Run agent
POST /api/v1/agents/{agent_id}/run

# 创建会话 / Create session
POST /api/v1/sessions

# 获取会话 / Get session
GET /api/v1/sessions/{session_id}

# 共享会话（跨智能体/团队）/ Reuse session across agents/teams
POST /api/v1/sessions/{session_id}/reuse

# 生成同步/异步摘要 / Generate session summary (sync or async)
POST /api/v1/sessions/{session_id}/summary?async=true|false

# 获取摘要快照 / Fetch session summary snapshot
GET /api/v1/sessions/{session_id}/summary

# 带筛选的历史查询 (`num_messages`, `stream_events`) / Fetch history with filters
GET /api/v1/sessions/{session_id}/history
```

[完整 AgentOS API →](/api/agentos)

## OpenAPI 规范 / OpenAPI Specification

完整的 OpenAPI 3.0 规范文档 / Complete OpenAPI 3.0 specification available:

- [OpenAPI YAML](https://github.com/rexleimo/HNO/blob/main/pkg/agentos/openapi.yaml)
- [Swagger UI](https://github.com/rexleimo/HNO/tree/main/pkg/agentos#api-documentation)

## 示例 / Examples

查看仓库中的工作示例 / See working examples in the repository:

- [Simple Agent](https://github.com/rexleimo/HNO/tree/main/cmd/examples/simple_agent)
- [Team Demo](https://github.com/rexleimo/HNO/tree/main/cmd/examples/team_demo)
- [Workflow Demo](https://github.com/rexleimo/HNO/tree/main/cmd/examples/workflow_demo)
- [RAG Demo](https://github.com/rexleimo/HNO/tree/main/cmd/examples/rag_demo)

## 包文档 / Package Documentation

完整的 Go 包文档可在 pkg.go.dev 上查看 / Full Go package documentation is available on pkg.go.dev:

[pkg.go.dev/github.com/rexleimo/HNO](https://pkg.go.dev/github.com/rexleimo/HNO)
