# API Reference

Complete API reference for HNO v1.0.

## Core Modules

- [Agent](/api/agent) - Autonomous AI agents
- [Team](/api/team) - Multi-agent collaboration
- [Workflow](/api/workflow) - Step-based orchestration
- [Models](/api/models) - LLM provider integrations
- [Tools](/api/tools) - Built-in and custom tools
- [Memory](/api/memory) - Conversation history management
- [Types](/api/types) - Core types and errors
- [AgentOS Server](/api/agentos) - Production HTTP server

## Quick Links

### Agent

```go
import "github.com/rexleimo/HNO/pkg/hno/agent"

agent.New(config) (*Agent, error)
agent.Run(ctx, input) (*RunOutput, error)
agent.ClearMemory()
```

[Full Agent API →](/api/agent)

### Team

```go
import "github.com/rexleimo/HNO/pkg/hno/team"

team.New(config) (*Team, error)
team.Run(ctx, input) (*RunOutput, error)

// Modes: Sequential, Parallel, LeaderFollower, Consensus
```

[Full Team API →](/api/team)

### Workflow

```go
import "github.com/rexleimo/HNO/pkg/hno/workflow"

workflow.New(config) (*Workflow, error)
workflow.Run(ctx, input) (*RunOutput, error)

// Primitives: Step, Condition, Loop, Parallel, Router
```

[Full Workflow API →](/api/workflow)

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

[Full Models API →](/api/models)

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

[Full Tools API →](/api/tools)

## Common Patterns

### Error Handling

```go
import "github.com/rexleimo/HNO/pkg/hno/types"

output, err := agent.Run(ctx, input)
if err != nil {
    switch {
    case errors.Is(err, types.ErrInvalidInput):
        // Handle invalid input
    case errors.Is(err, types.ErrRateLimit):
        // Handle rate limit
    default:
        // Handle other errors
    }
}
```

### Context Management

```go
import (
    "context"
    "time"
)

// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

output, err := agent.Run(ctx, input)
```

### Concurrent Agents

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

## Type Definitions

### Core Types

```go
// Message types
type Message struct {
    Role    MessageRole
    Content string
    Name    string
}

// Run output
type RunOutput struct {
    Content  string
    Messages []Message
    Metadata map[string]interface{}
}

// Model response
type ModelResponse struct {
    Content    string
    ToolCalls  []ToolCall
    FinishReason string
}
```

[Full Types Reference →](/api/types)

## AgentOS Server API

REST API endpoints for production deployment:

```bash
# Health check
GET /health

# List agents
GET /api/v1/agents

# Run agent
POST /api/v1/agents/{agent_id}/run

# Create session
POST /api/v1/sessions

# Get session
GET /api/v1/sessions/{session_id}

# Reuse session across agents/teams
POST /api/v1/sessions/{session_id}/reuse

# Generate session summary (sync or async)
POST /api/v1/sessions/{session_id}/summary?async=true|false

# Fetch session summary snapshot
GET /api/v1/sessions/{session_id}/summary

# Fetch history with filters (num_messages, stream_events)
GET /api/v1/sessions/{session_id}/history
```

[Full AgentOS API →](/api/agentos)

## OpenAPI Specification

Complete OpenAPI 3.0 specification available:

- [OpenAPI YAML](https://github.com/rexleimo/HNO/blob/main/pkg/agentos/openapi.yaml)
- [Swagger UI](https://github.com/rexleimo/HNO/tree/main/pkg/agentos#api-documentation)

## Examples

See working examples in the repository:

- [Simple Agent](https://github.com/rexleimo/HNO/tree/main/cmd/examples/simple_agent)
- [Team Demo](https://github.com/rexleimo/HNO/tree/main/cmd/examples/team_demo)
- [Workflow Demo](https://github.com/rexleimo/HNO/tree/main/cmd/examples/workflow_demo)
- [RAG Demo](https://github.com/rexleimo/HNO/tree/main/cmd/examples/rag_demo)

## Package Documentation

Full Go package documentation is available on pkg.go.dev:

[pkg.go.dev/github.com/rexleimo/HNO](https://pkg.go.dev/github.com/rexleimo/HNO)
