# AgentOS Server API Reference

## NewServer

Create HTTP server.

**Signature:**
```go
func NewServer(config *Config) (*Server, error)

type Config struct {
    Address        string           // Server address (default: :8080)
    Prefix         string           // API prefix (default: /api/v1)
    SessionStorage session.Storage  // Session storage (default: memory)
    Logger         *slog.Logger     // Logger (default: slog.Default())
    Debug          bool             // Debug mode (default: false)
    AllowOrigins   []string         // CORS origins
    AllowMethods   []string         // CORS methods
    AllowHeaders   []string         // CORS headers
    RequestTimeout time.Duration    // Request timeout (default: 30s)
    MaxRequestSize int64            // Max request size (default: 10MB)

    // Knowledge API (optional)
    VectorDBConfig *VectorDBConfig  // Vector database configuration (e.g., chromadb)
    EmbeddingConfig *EmbeddingConfig // Embedding model configuration (e.g., OpenAI)
    KnowledgeAPI   *KnowledgeAPIOptions // Toggle knowledge endpoints

    // Session summaries (optional)
    SummaryManager *session.SummaryManager // Configure sync/async summaries
}

type VectorDBConfig struct {
    Type           string // e.g., "chromadb"
    BaseURL        string // Vector DB endpoint
    CollectionName string // Default collection
    Database       string // Optional database name
    Tenant         string // Optional tenant name
}

type EmbeddingConfig struct {
    Provider string // e.g., "openai"
    APIKey   string
    Model    string // e.g., "text-embedding-3-small"
    BaseURL  string // e.g., "https://api.openai.com/v1"
}
```

**Example:**
```go
server, err := agentos.NewServer(&agentos.Config{
    Address: ":8080",
    Debug:   true,
    RequestTimeout: 60 * time.Second,
})
```

## Server.RegisterAgent

Register an agent.

**Signature:**
```go
func (s *Server) RegisterAgent(agentID string, ag *agent.Agent) error
```

**Example:**
```go
err := server.RegisterAgent("assistant", myAgent)
```

## Server.Start / Shutdown

Start and stop server.

**Signatures:**
```go
func (s *Server) Start() error
func (s *Server) Shutdown(ctx context.Context) error
```

**Example:**
```go
go func() {
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}()

// Graceful shutdown
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
server.Shutdown(ctx)
```

## API Endpoints

See [OpenAPI Specification](../../pkg/agentos/openapi.yaml) for complete API documentation.

**Core Endpoints:**
- `GET /health` - Health check
- `POST /api/v1/sessions` - Create session
- `GET /api/v1/sessions/{id}` - Get session
- `PUT /api/v1/sessions/{id}` - Update session
- `DELETE /api/v1/sessions/{id}` - Delete session
- `GET /api/v1/sessions` - List sessions
- `POST /api/v1/sessions/{id}/reuse` - Attach session to another agent/team/workflow
- `GET /api/v1/sessions/{id}/summary` - Retrieve stored summary (404 if not ready)
- `POST /api/v1/sessions/{id}/summary?async=true|false` - Generate session summary (sync or async)
- `GET /api/v1/sessions/{id}/history` - Fetch conversation history (`num_messages`, `stream_events`)
- `GET /api/v1/agents` - List agents
- `POST /api/v1/agents/{id}/run` - Run agent

### Session Summaries & Reuse (v1.2.6)

Configure summaries by supplying a `session.SummaryManager`:

```go
// import (
//     "github.com/rexleimo/agno-go/pkg/hno/models/openai"
//     "github.com/rexleimo/agno-go/pkg/hno/session"
// )
summaryModel, _ := openai.New("gpt-4o-mini", openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})

summary := session.NewSummaryManager(
    session.WithSummaryModel(summaryModel),
    session.WithSummaryTimeout(45*time.Second),
)

server, err := agentos.NewServer(&agentos.Config{
    Address:        ":8080",
    SummaryManager: summary,
})
```

- `POST /api/v1/sessions/{id}/summary`
  - `async=true` schedules background jobs; the endpoint returns `202 Accepted`
  - `async=false` runs synchronously and returns the generated summary
- `GET /api/v1/sessions/{id}/summary` returns the latest snapshot (404 until ready)
- `POST /api/v1/sessions/{id}/reuse` shares a session across agents, teams, workflows, or user IDs

### History Filters & Run Metadata

- `GET /api/v1/sessions/{id}/history?num_messages=20&stream_events=true` trims history to the latest N interactions and mirrors Python SSE options.
- Session responses now include run metadata (`runs[*].status`, timestamps, cancellation reasons, `cache_hit`) enabling audit trails and cache observability.

**Knowledge Endpoints (optional):**
- `POST /api/v1/knowledge/search` — Vector similarity search in knowledge base
- `GET  /api/v1/knowledge/config` — Available chunkers, VectorDBs, and embedding model info
- `POST /api/v1/knowledge/content` — Knowledge ingestion with configurable chunking (`chunk_size`, `chunk_overlap`).

`POST /api/v1/knowledge/content` accepts `chunk_size` and `chunk_overlap`
in both JSON and multipart uploads. Provide them as query parameters for
`text/plain` requests or as form fields (`chunk_size=2000&chunk_overlap=250`) when
streaming files. Both values propagate into the reader metadata so downstream
pipelines can inspect how documents were segmented.

```bash
curl -X POST http://localhost:8080/api/v1/knowledge/content \
  -F file=@docs/guide.md \
  -F chunk_size=1800 \
  -F chunk_overlap=200 \
  -F metadata='{"source_url":"https://example.com/guide"}'
```

Each stored chunk records `chunk_size`, `chunk_overlap`, and the
`chunker_type` used—mirroring the AgentOS Python responses.

Example request:
```bash
curl -X POST http://localhost:8080/api/v1/knowledge/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "How to create an Agent?",
    "limit": 5,
    "filters": {"source": "documentation"}
  }'
```

Minimal server config for Knowledge API:
```go
server, err := agentos.NewServer(&agentos.Config{
  Address: ":8080",
  VectorDBConfig: &agentos.VectorDBConfig{
    Type:           "chromadb",
    BaseURL:        os.Getenv("CHROMADB_URL"),
    CollectionName: "agno_knowledge",
  },
  EmbeddingConfig: &agentos.EmbeddingConfig{
    Provider: "openai",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
    Model:    "text-embedding-3-small",
  },
})
```

See runnable example: `cmd/examples/knowledge_api/`.

## Best Practices

### 1. Always Use Contexts

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

output, err := ag.Run(ctx, input)
```

### 2. Handle Errors Properly

```go
output, err := ag.Run(ctx, input)
if err != nil {
    switch {
    case types.IsInvalidInputError(err):
        // Handle invalid input
    case types.IsRateLimitError(err):
        // Retry with backoff
    default:
        // Handle other errors
    }
}
```

### 3. Manage Memory

```go
// Clear when starting new topic
ag.ClearMemory()

// Or use limited memory
mem := memory.NewInMemory(50)
```

### 4. Set Appropriate Timeouts

```go
server, _ := agentos.NewServer(&agentos.Config{
    RequestTimeout: 60 * time.Second, // For complex agents
})
```

### 5. AgentOS HTTP Tips

- Override the default `GET /health` path via `Config.HealthPath` or attach your
  own handlers with `server.GetHealthRouter("/health-check").GET("", customHandler)`.
- `/openapi.yaml` always serves the current OpenAPI document and `/docs` hosts a
  self-contained Swagger UI bundle. Call `server.Resync()` after hot-swapping
  routers to remount the documentation routes.

Sample probes:

```bash
curl http://localhost:8080/health-check
curl http://localhost:8080/openapi.yaml | head -n 5
```
