# AgentOS Server API 参考 / AgentOS Server API Reference

## NewServer

创建 HTTP 服务器。/ Create HTTP server.

**签名 / Signature:**
```go
func NewServer(config *Config) (*Server, error)

type Config struct {
    Address        string           // 服务器地址 (默认: :8080) / Server address (default: :8080)
    Prefix         string           // API 前缀 (默认: /api/v1) / API prefix (default: /api/v1)
    SessionStorage session.Storage  // 会话存储 (默认: memory) / Session storage (default: memory)
    Logger         *slog.Logger     // 日志记录器 (默认: slog.Default()) / Logger (default: slog.Default())
    Debug          bool             // 调试模式 (默认: false) / Debug mode (default: false)
    AllowOrigins   []string         // CORS 源 / CORS origins
    AllowMethods   []string         // CORS 方法 / CORS methods
    AllowHeaders   []string         // CORS 头 / CORS headers
    RequestTimeout time.Duration    // 请求超时 (默认: 30s) / Request timeout (default: 30s)
    MaxRequestSize int64            // 最大请求大小 (默认: 10MB) / Max request size (default: 10MB)

    // 知识库 API（可选）/ Knowledge API (optional)
    VectorDBConfig  *VectorDBConfig  // 向量数据库配置（如 chromadb）/ Vector DB configuration
    EmbeddingConfig *EmbeddingConfig // 嵌入模型配置（如 OpenAI）/ Embedding model configuration
    KnowledgeAPI    *KnowledgeAPIOptions // 知识端点开关 / Toggle knowledge endpoints

    // 会话摘要（可选）/ Session summaries (optional)
    SummaryManager *session.SummaryManager // 配置同步/异步摘要 / Configure sync/async summaries
}
 
type VectorDBConfig struct {
    Type           string // 例如 "chromadb" / e.g., "chromadb"
    BaseURL        string // 向量数据库地址 / Vector DB endpoint
    CollectionName string // 默认集合名 / Default collection
    Database       string // 可选数据库名 / Optional database
    Tenant         string // 可选租户名 / Optional tenant
}

type EmbeddingConfig struct {
    Provider string // 例如 "openai" / e.g., "openai"
    APIKey   string
    Model    string // 例如 "text-embedding-3-small"
    BaseURL  string // 例如 "https://api.openai.com/v1"
}
```

**示例 / Example:**
```go
server, err := agentos.NewServer(&agentos.Config{
    Address: ":8080",
    Debug:   true,
    RequestTimeout: 60 * time.Second,
})
```

## Server.RegisterAgent

注册一个智能体。/ Register an agent.

**签名 / Signature:**
```go
func (s *Server) RegisterAgent(agentID string, ag *agent.Agent) error
```

**示例 / Example:**
```go
err := server.RegisterAgent("assistant", myAgent)
```

## Server.Start / Shutdown

启动和停止服务器。/ Start and stop server.

**签名 / Signatures:**
```go
func (s *Server) Start() error
func (s *Server) Shutdown(ctx context.Context) error
```

**示例 / Example:**
```go
go func() {
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}()

// 优雅关闭 / Graceful shutdown
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
server.Shutdown(ctx)
```

## API 端点 / API Endpoints

查看完整的 API 文档请参阅 [OpenAPI 规范](../../pkg/agentos/openapi.yaml)。/ See [OpenAPI Specification](../../pkg/agentos/openapi.yaml) for complete API documentation.

**核心端点 / Core Endpoints:**
- `GET /health` - 健康检查 / Health check
- `POST /api/v1/sessions` - 创建会话 / Create session
- `GET /api/v1/sessions/{id}` - 获取会话 / Get session
- `PUT /api/v1/sessions/{id}` - 更新会话 / Update session
- `DELETE /api/v1/sessions/{id}` - 删除会话 / Delete session
- `GET /api/v1/sessions` - 列出会话 / List sessions
- `POST /api/v1/sessions/{id}/reuse` - 共享会话 / Reuse session across agents
- `GET /api/v1/sessions/{id}/summary` - 获取会话摘要（未准备好返回 404）/ Retrieve session summary (404 until ready)
- `POST /api/v1/sessions/{id}/summary?async=true|false` - 生成同步/异步摘要 / Generate sync or async summary
- `GET /api/v1/sessions/{id}/history` - 获取历史记录（`num_messages`、`stream_events`）/ Fetch history with filters
- `GET /api/v1/agents` - 列出智能体 / List agents
- `POST /api/v1/agents/{id}/run` - 运行智能体 / Run agent

### 会话摘要与复用 (v1.2.6) / Session Summaries & Reuse (v1.2.6)

配置 `session.SummaryManager` 以启用同步/异步摘要：/ Configure `session.SummaryManager` to enable sync/async summaries:

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
  - `async=true` 安排后台任务, 返回 `202 Accepted` / schedules a background job and returns `202 Accepted`
  - `async=false` 同步执行并返回摘要 / runs synchronously and returns the summary
- `GET /api/v1/sessions/{id}/summary` 返回最新快照（未完成时 404）/ returns the latest snapshot (404 until ready)
- `POST /api/v1/sessions/{id}/reuse` 在 Agent、Team、Workflow 或不同用户之间共享会话 / shares a session across agents, teams, workflows, or user IDs

### 历史筛选与运行元数据 / History Filters & Run Metadata

- `GET /api/v1/sessions/{id}/history?num_messages=20&stream_events=true` 限定最近 N 条记录并与 Python 的 SSE 开关对齐 / trims the history to the last N entries and mirrors Python SSE toggles.
- 会话响应包含运行元数据（`runs[*].status`、时间戳、取消原因、`cache_hit`）以便审计与缓存观测 / session responses now include run metadata (`runs[*].status`, timestamps, cancellation reasons, `cache_hit`) for audit trails and cache observability.

**知识库端点（可选） / Knowledge Endpoints (optional):**
- `POST /api/v1/knowledge/search` — 在知识库中进行向量相似搜索 / Vector similarity search in knowledge base
- `GET  /api/v1/knowledge/config` — 返回可用分块器、向量库与嵌入模型信息 / Available chunkers, VectorDBs, embedding model info
- `POST /api/v1/knowledge/content` — 支持可配置分块的最小入库（`chunk_size` 与 `chunk_overlap`，text/plain 或 application/json）/ Minimal ingestion with configurable chunking (`chunk_size`, `chunk_overlap`).

`POST /api/v1/knowledge/content` 现在支持 `chunk_size` 与 `chunk_overlap`，在 JSON 与 multipart 上传中均可使用。对于 `text/plain` 请求，可以使用查询参数传递；在流式文件上传时，可以通过表单字段（如 `chunk_size=2000&chunk_overlap=250`）传递。两个参数会写入 reader 元数据，以便下游管道了解文档是如何被切分的。 / `POST /api/v1/knowledge/content` now accepts `chunk_size` and `chunk_overlap` in both JSON and multipart uploads. Provide them as query parameters for `text/plain` requests or as form fields (`chunk_size=2000&chunk_overlap=250`) when streaming files. Both values propagate into the reader metadata so downstream pipelines can inspect how documents were segmented.

```bash
curl -X POST http://localhost:8080/api/v1/knowledge/content \
  -F file=@docs/guide.md \
  -F chunk_size=1800 \
  -F chunk_overlap=200 \
  -F metadata='{"source_url":"https://example.com/guide"}'
```

每个存储的分块都会记录 `chunk_size`、`chunk_overlap` 与 `chunker_type`，并与 AgentOS Python 响应保持一致。 / Each stored chunk records `chunk_size`, `chunk_overlap`, and the `chunker_type` used—mirroring the AgentOS Python responses.

示例请求 / Example:
```bash
curl -X POST http://localhost:8080/api/v1/knowledge/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "如何创建 Agent?",
    "limit": 5,
    "filters": {"source": "documentation"}
  }'
```

最小服务器配置（启用知识库 API）/ Minimal server config (enable Knowledge API):
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

可运行示例 / Runnable example: `cmd/examples/knowledge_api/`

## 最佳实践 / Best Practices

### 1. 始终使用 Context / Always Use Contexts

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

output, err := ag.Run(ctx, input)
```

### 2. 正确处理错误 / Handle Errors Properly

```go
output, err := ag.Run(ctx, input)
if err != nil {
    switch {
    case types.IsInvalidInputError(err):
        // 处理无效输入 / Handle invalid input
    case types.IsRateLimitError(err):
        // 使用退避重试 / Retry with backoff
    default:
        // 处理其他错误 / Handle other errors
    }
}
```

### 3. 管理内存 / Manage Memory

```go
// 开始新话题时清除 / Clear when starting new topic
ag.ClearMemory()

// 或使用有限内存 / Or use limited memory
mem := memory.NewInMemory(50)
```

### 4. 设置适当的超时 / Set Appropriate Timeouts

```go
server, _ := agentos.NewServer(&agentos.Config{
    RequestTimeout: 60 * time.Second, // 用于复杂智能体 / For complex agents
})
```

### 5. AgentOS HTTP 提示 / AgentOS HTTP Tips

- 通过 `Config.HealthPath` 或 `server.GetHealthRouter("/health-check").GET("", customHandler)` 覆盖默认的 `GET /health` 路径, 以匹配生产探针 / Override the default `GET /health` path via `Config.HealthPath` or by attaching your own handlers with `server.GetHealthRouter("/health-check").GET("", customHandler)` so it matches production probes.
- `/openapi.yaml` 始终提供当前的 OpenAPI 文档, `/docs` 暴露内置的 Swagger UI。热更新路由或挂载自定义处理器后, 需要调用 `server.Resync()` 以重新挂载文档路由。 / `/openapi.yaml` always serves the current OpenAPI document and `/docs` hosts a self-contained Swagger UI bundle. Call `server.Resync()` after hot-swapping routers to remount the documentation routes.

探针示例 / Sample probes:

```bash
curl http://localhost:8080/health-check
curl http://localhost:8080/openapi.yaml | head -n 5
```
