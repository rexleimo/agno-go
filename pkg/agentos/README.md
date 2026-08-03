# AgentOS - Production-Ready Agent Server

AgentOS is a production-ready HTTP server for managing and running AI agents built with Agno-Go.

## Features

- 🚀 **RESTful API** - Clean, intuitive API design
- 💬 **Session Management** - Multi-turn conversation support
- 🤖 **Agent Registry** - Dynamic agent registration and execution
- 🔧 **Tool Support** - Agents can use tools for extended capabilities
- 💾 **Memory Management** - Automatic conversation history management
- 📊 **Structured Logging** - Built-in request logging with slog
- ⚡ **High Performance** - Built on Gin framework for speed
- 🔒 **Thread-Safe** - Concurrent request handling

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-go/pkg/agentos"
)

func main() {
    // Create an OpenAI model
    model, err := openai.New("gpt-4", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create an agent
    ag, err := agent.New(agent.Config{
        Name:         "Assistant",
        Model:        model,
        Instructions: "You are a helpful assistant.",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create AgentOS server
    server, err := agentos.NewServer(&agentos.Config{
        Address: ":8080",
        Debug:   true,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Register the agent
    if err := server.RegisterAgent("assistant", ag); err != nil {
        log.Fatal(err)
    }

    // Start server in a goroutine
    go func() {
        if err := server.Start(); err != nil {
            log.Printf("Server error: %v", err)
        }
    }()

    log.Println("AgentOS server started on :8080")
    log.Println("Try: curl -X POST http://localhost:8080/api/v1/agents/assistant/run -H 'Content-Type: application/json' -d '{\"input\":\"Hello!\"}'")

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Printf("Server shutdown error: %v", err)
    }
    log.Println("Server stopped")
}
```

## API Documentation

Full OpenAPI 3.0 specification is available in [openapi.yaml](openapi.yaml).

### Streaming Events

AgentOS supports Server-Sent Events (SSE) for monitoring agent execution in real time via:

```
POST /api/v1/agents/{agent_id}/run/stream
```

Query parameter `types` can be used to filter event categories (e.g. `types=token,complete,reasoning`).  
Each event is delivered in the following structure:

- `run_start`: Input payload and session metadata.
- `reasoning`: Structured reasoning segments (`content`, `token_count`, `redacted_content`) produced by supported reasoning models.
- `token`: Individual streaming tokens for response generation.
- `tool_call`: Tool invocation details (name, arguments, result).
- `complete`: Final response content, elapsed duration, aggregated token usage (including reasoning token estimates).
- `error`: Rich error payload when execution fails.

📦 需要将事件转发到 Logfire 或其他可观测性平台？请参考 [`cmd/examples/logfire_observability`](../../cmd/examples/logfire_observability) 示例（使用 `go run -tags logfire .`）以及文档 [`docs/release/logfire_observability.md`](../../docs/release/logfire_observability.md)。

Refer to `pkg/agentos/events.go` for the latest schema definitions.

### Endpoints

#### Health Check

```bash
GET /health
```

Returns server health status.

**Response:**
```json
{
  "status": "healthy",
  "service": "agentos",
  "time": 1704067200
}
```

#### Sessions

**Create Session**
```bash
POST /api/v1/sessions
Content-Type: application/json

{
  "agent_id": "assistant",
  "user_id": "user-123",
  "name": "Customer Support"
}
```

**Get Session**
```bash
GET /api/v1/sessions/{session_id}
```

**Update Session**
```bash
PUT /api/v1/sessions/{session_id}
Content-Type: application/json

{
  "name": "Updated Name",
  "metadata": {"key": "value"}
}
```

**Delete Session**
```bash
DELETE /api/v1/sessions/{session_id}
```

**List Sessions**
```bash
GET /api/v1/sessions?agent_id=assistant&user_id=user-123
```

#### Agents

**List Agents**
```bash
GET /api/v1/agents
```

**Response:**
```json
{
  "agents": [
    {
      "id": "assistant",
      "name": "Assistant"
    }
  ],
  "count": 1
}
```

**Run Agent**
```bash
POST /api/v1/agents/{agent_id}/run
Content-Type: application/json

{
  "input": "What is the weather in SF?",
  "session_id": "optional-session-id",
  "media": [
    {
      "type": "image",
      "url": "https://example.com/photo.png"
    }
  ]
}
```

**Response:**
```json
{
  "content": "I don't have access to real-time weather data...",
  "session_id": "optional-session-id",
  "metadata": {
    "agent_id": "assistant",
    "media": [
      {
        "type": "image",
        "url": "https://example.com/photo.png"
      }
    ]
  }
}
```

> **Media support**: when `media` attachments are supplied, AgentOS validates the payload and keeps the attachment metadata alongside the run so downstream consumers can render or audit the original assets. Pure-media requests (no `input` text) are accepted as long as at least one attachment is present.

## Configuration

### Server Config

```go
type Config struct {
    // Server address (default: :8080)
    Address string

    // Session storage (default: memory storage)
    SessionStorage session.Storage

    // Logger (default: slog.Default())
    Logger *slog.Logger

    // Enable debug mode
    Debug bool

    // CORS settings
    AllowOrigins []string
    AllowMethods []string
    AllowHeaders []string

    // Request timeout (default: 30s)
    RequestTimeout time.Duration

    // Max request size in bytes (default: 10MB)
    MaxRequestSize int64
}
```

### Example with Custom Config

```go
server, err := agentos.NewServer(&agentos.Config{
    Address: ":9090",
    Debug:   true,
    AllowOrigins: []string{"https://example.com"},
    RequestTimeout: 60 * time.Second,
    Logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
})
```

## Advanced Usage

### With Multiple Agents

```go
// Register multiple agents
server.RegisterAgent("customer-support", customerSupportAgent)
server.RegisterAgent("sales-assistant", salesAgent)
server.RegisterAgent("technical-support", techSupportAgent)

// Clients can now call:
// POST /api/v1/agents/customer-support/run
// POST /api/v1/agents/sales-assistant/run
// POST /api/v1/agents/technical-support/run
```

### With Tool-Enabled Agents

```go
import (
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-go/pkg/hno/tools/http"
)

// Create agent with tools
ag, err := agent.New(agent.Config{
    Name:  "ToolAgent",
    Model: model,
    Toolkits: []toolkit.Toolkit{
        calculator.New(),
        http.New(),
    },
    Instructions: "You can perform calculations and make HTTP requests.",
})

server.RegisterAgent("tool-agent", ag)
```

### With Custom Session Storage

```go
import "github.com/rexleimo/agno-go/pkg/hno/session"

// Use custom storage (e.g., PostgreSQL, Redis)
storage := session.NewPostgresStorage(connString)

server, err := agentos.NewServer(&agentos.Config{
    SessionStorage: storage,
})
```

## Testing

The package includes comprehensive tests:

```bash
# Run all tests
go test ./pkg/agentos/ -v

# Run with coverage
go test ./pkg/agentos/ -cover

# Generate coverage report
go test ./pkg/agentos/ -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Current test coverage: 65.0%**

- 16 Registry tests (100% coverage)
- 13 Server/API tests
- Session management tests
- Agent execution tests

## Architecture

```
┌─────────────────────────────────────────┐
│         HTTP Client (curl/SDK)          │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│          Gin HTTP Server                │
│  ┌─────────────────────────────────┐   │
│  │      Middleware Stack           │   │
│  │  - Logger                        │   │
│  │  - CORS                          │   │
│  │  - Timeout                       │   │
│  │  - Recovery                      │   │
│  └─────────────────────────────────┘   │
└────────────────┬────────────────────────┘
                 │
    ┌────────────┼────────────┐
    │            │            │
    ▼            ▼            ▼
┌────────┐  ┌─────────┐  ┌──────────┐
│Session │  │ Agent   │  │  Health  │
│Handler │  │ Handler │  │  Handler │
└───┬────┘  └────┬────┘  └──────────┘
    │            │
    ▼            ▼
┌─────────┐  ┌──────────┐
│ Session │  │  Agent   │
│ Storage │  │ Registry │
└─────────┘  └────┬─────┘
                   │
                   ▼
              ┌─────────┐
              │  Agent  │
              │ (Agno)  │
              └────┬────┘
                   │
        ┌──────────┼──────────┐
        ▼          ▼          ▼
    ┌───────┐  ┌──────┐  ┌────────┐
    │ Model │  │Tools │  │ Memory │
    └───────┘  └──────┘  └────────┘
```

## Error Handling

AgentOS uses structured error responses:

```json
{
  "error": "agent not found",
  "message": "agent with ID 'non-existent' not found",
  "code": "AGENT_NOT_FOUND"
}
```

### Error Codes

- `INVALID_REQUEST` - Invalid request format or missing required fields
- `AGENT_NOT_FOUND` - Requested agent does not exist in registry
- `SESSION_NOT_FOUND` - Requested session does not exist
- `EXECUTION_ERROR` - Agent execution failed

## Performance

AgentOS is built for high-performance agent serving:

- **Concurrent Requests**: Thread-safe agent registry with RWMutex
- **Low Latency**: Gin framework for fast routing
- **Memory Efficient**: Reuses agent instances across requests
- **Graceful Shutdown**: Properly closes resources on shutdown

## Deployment

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o agentos cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/agentos .
EXPOSE 8080
CMD ["./agentos"]
```

### Docker Compose

See [docker-compose.yml](../../docker-compose.yml) for full setup.

### Environment Variables

```bash
# Required
OPENAI_API_KEY=sk-...

# Optional
AGENTOS_ADDRESS=:8080
AGENTOS_DEBUG=false
ANTHROPIC_API_KEY=sk-ant-...
```

## Logging

AgentOS uses structured logging with `log/slog`:

```
2025/10/02 00:15:15 INFO request method=POST path=/api/v1/sessions status=201 duration=174.875µs ip=192.168.1.1
2025/10/02 00:15:15 INFO session created session_id=c1110224-9676-41eb-842a-611a205c28c3 agent_id=assistant
2025/10/02 00:15:16 INFO agent run requested agent_id=assistant input="Hello!" session_id=c1110224...
2025/10/02 00:15:17 INFO agent run completed agent_id=assistant content_length=142
```

## Best Practices

1. **Use Sessions for Multi-Turn Conversations**
   - Create a session once, reuse for multiple agent runs
   - Sessions automatically manage conversation history

2. **Register Agents at Startup**
   - Register all agents before starting the server
   - Agent registry is thread-safe but best to register early

3. **Handle Errors Gracefully**
   - Check response status codes
   - Use error codes to determine appropriate action

4. **Set Appropriate Timeouts**
   - Default is 30s, adjust based on your agent complexity
   - LLM calls can take time, especially with tools

5. **Monitor Memory Usage**
   - Sessions store conversation history
   - Consider implementing session cleanup for long-running servers

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./pkg/agentos/`
2. Code is formatted: `make fmt`
3. Coverage is maintained or improved

## License

MIT License - See [LICENSE](../../LICENSE) for details.

## Related Projects

- [Agno-Go](https://github.com/rexleimo/agno-go) - Core agent framework
- [Agno](https://github.com/agno-agi/agno) - Python implementation

## Support

- 📚 [Documentation](https://docs.agno.com)
- 💬 [Discussions](https://github.com/rexleimo/agno-go/discussions)
- 🐛 [Issues](https://github.com/rexleimo/agno-go/issues)
