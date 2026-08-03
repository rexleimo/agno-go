# Architecture

Agno-Go follows a clean, modular architecture designed for simplicity, efficiency, and extensibility.

## Core Philosophy

**Simple, Efficient, Scalable**

## Overall Architecture

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

## Core Interfaces

### 1. Model Interface

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

### 2. Toolkit Interface

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

### 3. Memory Interface

```go
type Memory interface {
    Add(message types.Message) error
    GetMessages() []types.Message
    Clear() error
}
```

## Component Details

### Agent

**File**: `pkg/hno/agent/agent.go`

Autonomous AI entity that:
- Uses LLM for reasoning
- Can call tools
- Maintains conversation memory
- Validates inputs/outputs with hooks

**Key Methods**:
```go
New(config Config) (*Agent, error)
Run(ctx context.Context, input string) (*RunOutput, error)
ClearMemory()
```

### Team

**File**: `pkg/hno/team/team.go`

Multi-agent collaboration with 4 coordination modes:

1. **Sequential** - Agents work one after another
2. **Parallel** - All agents work simultaneously
3. **LeaderFollower** - Leader delegates to followers
4. **Consensus** - Agents discuss until agreement

### Workflow

**File**: `pkg/hno/workflow/workflow.go`

Step-based orchestration with 5 primitives:

1. **Step** - Execute agent or function
2. **Condition** - Branch based on context
3. **Loop** - Iterate with exit condition
4. **Parallel** - Execute steps concurrently
5. **Router** - Dynamic routing

### Models

**Directory**: `pkg/hno/models/`

LLM provider implementations:
- `openai/` - OpenAI GPT models
- `anthropic/` - Anthropic Claude models
- `ollama/` - Ollama local models
- `deepseek/`, `gemini/`, `modelscope/` - Other providers

### Tools

**Directory**: `pkg/hno/tools/`

Extensible toolkit system:
- `calculator/` - Math operations
- `http/` - HTTP requests
- `file/` - File operations
- `search/` - Web search

## AgentOS Production Server

**Directory**: `pkg/agentos/`

Production-ready HTTP server with:

- RESTful API endpoints
- Session management
- Agent registry
- Health monitoring
- CORS support
- Request timeout handling

**Architecture**:
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

## Design Patterns

### 1. Interface-Based Design

All core components use interfaces for flexibility:

```go
type Model interface { /* ... */ }
type Toolkit interface { /* ... */ }
type Memory interface { /* ... */ }
```

### 2. Composition Over Inheritance

Agents compose models, tools, and memory:

```go
type Agent struct {
    Model    Model
    Toolkits []Toolkit
    Memory   Memory
    // ...
}
```

### 3. Context Propagation

All operations accept `context.Context` for cancellation and timeouts:

```go
func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error)
```

### 4. Error Wrapping

Consistent error handling with wrapped errors:

```go
if err != nil {
    return nil, fmt.Errorf("failed to run agent: %w", err)
}
```

## Performance Optimizations

### 1. Low Allocation Count

- Minimal heap allocations (8-9 per agent)
- Pre-allocated slices
- String interning where appropriate

### 2. Efficient Memory Layout

```go
type Agent struct {
    ID           string   // 16B
    Name         string   // 16B
    Model        Model    // 16B (interface)
    // Total: ~112B struct + heap allocations
}
```

### 3. Goroutine Safety

- No global state
- Thread-safe by design
- Lock-free where possible

## Concurrency Model

### Agent Concurrency

```go
// Safe to create multiple agents concurrently
for i := 0; i < 100; i++ {
    go func() {
        ag, _ := agent.New(config)
        output, _ := ag.Run(ctx, input)
    }()
}
```

### Team Parallel Mode

```go
// Agents run in parallel goroutines
team := team.New(team.Config{
    Mode: team.ModeParallel,
    Agents: agents,
})
```

### Workflow Parallel Step

```go
// Steps execute concurrently
workflow.NewParallel("tasks", []Primitive{
    step1, step2, step3,
})
```

## Extensibility Points

### 1. Custom Models

Implement `Model` interface:

```go
type MyModel struct{}

func (m *MyModel) Invoke(ctx context.Context, req *InvokeRequest) (*ModelResponse, error) {
    // Custom implementation
}
```

### 2. Custom Tools

Extend `BaseToolkit`:

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

### 3. Custom Memory

Implement `Memory` interface:

```go
type MyMemory struct{}

func (m *MyMemory) Add(msg types.Message) error {
    // Custom storage
}
```

## Testing Strategy

### Unit Tests

- Each package has `*_test.go` files
- Mock implementations for interfaces
- Table-driven tests

### Integration Tests

- End-to-end workflow tests
- Multi-agent scenarios
- Real API integration tests

### Benchmark Tests

- Performance benchmarks in `*_bench_test.go`
- Memory allocation tracking
- Concurrency stress tests

## Dependencies

### Core Dependencies

- **Go Standard Library** - Most functionality
- **No heavy frameworks** - Lightweight design

### Optional Dependencies

- LLM provider SDKs (OpenAI, Anthropic, etc.)
- Vector database clients (ChromaDB)
- HTTP client libraries

## Future Architecture

### Planned Enhancements

1. **Streaming Support** - Real-time response streaming
2. **Plugin System** - Dynamic tool loading
3. **Distributed Agents** - Multi-node deployment
4. **Advanced Memory** - Persistent storage, vector memory

## Best Practices

### 1. Use Interfaces

```go
var model models.Model = openai.New(...)
```

### 2. Handle Errors

```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### 3. Use Contexts

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### 4. Keep It Simple

Follow the KISS principle - don't over-engineer.

## References

- [Performance Benchmarks](/advanced/performance)
- [Deployment Guide](/advanced/deployment)
- [API Reference](/api/)
- [Source Code](https://github.com/rexleimo/agno-Go)
