# AgentOS Server API 레퍼런스

## NewServer

HTTP 서버를 생성합니다.

**함수 시그니처:**
```go
func NewServer(config *Config) (*Server, error)

type Config struct {
    Address        string           // 서버 주소 (기본값: :8080)
    Prefix         string           // Optional mount prefix; empty exposes /api/v1
    SessionStorage session.Storage  // 세션 저장소 (기본값: memory)
    Logger         *slog.Logger     // 로거 (기본값: slog.Default())
    Debug          bool             // 디버그 모드 (기본값: false)
    AllowOrigins   []string         // CORS origins
    AllowMethods   []string         // CORS methods
    AllowHeaders   []string         // CORS headers
    RequestTimeout time.Duration    // 요청 타임아웃 (기본값: 30s)
    MaxRequestSize int64            // 최대 요청 크기 (기본값: 10MB)

    // 지식 API (선택) / Knowledge API (optional)
    VectorDBConfig  *VectorDBConfig  // 벡터 DB 구성 (예: chromadb)
    EmbeddingConfig *EmbeddingConfig // 임베딩 모델 구성 (예: OpenAI)
    KnowledgeAPI    *KnowledgeAPIOptions // 지식 엔드포인트 온/오프

    // 세션 요약 (선택) / Session summaries (optional)
    SummaryManager *session.SummaryManager // 동기/비동기 요약을 구성
}

type VectorDBConfig struct {
    Type           string // 예: "chromadb"
    BaseURL        string // 벡터 DB 엔드포인트
    CollectionName string // 기본 컬렉션
    Database       string // 선택 데이터베이스
    Tenant         string // 선택 테넌트
}

type EmbeddingConfig struct {
    Provider string // 예: "openai"
    APIKey   string
    Model    string // 예: "text-embedding-3-small"
    BaseURL  string // 예: "https://api.openai.com/v1"
}
```

**예제:**
```go
server, err := agentos.NewServer(&agentos.Config{
    Address: ":8080",
    Debug:   true,
    RequestTimeout: 60 * time.Second,
})
```

## Server.RegisterAgent

에이전트를 등록합니다.

**함수 시그니처:**
```go
func (s *Server) RegisterAgent(agentID string, ag *agent.Agent) error
```

**예제:**
```go
err := server.RegisterAgent("assistant", myAgent)
```

## Server.Start / Shutdown

서버를 시작하고 중지합니다.

**함수 시그니처:**
```go
func (s *Server) Start() error
func (s *Server) Shutdown(ctx context.Context) error
```

**예제:**
```go
go func() {
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}()

// 우아한 종료
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
server.Shutdown(ctx)
```

## API 엔드포인트

완전한 API 문서는 [OpenAPI 명세](../../pkg/agentos/openapi.yaml)를 참고하세요.

**핵심 엔드포인트:**
- `GET /health` - 헬스 체크
- `POST /api/v1/sessions` - 세션 생성
- `GET /api/v1/sessions/{id}` - 세션 조회
- `PUT /api/v1/sessions/{id}` - 세션 업데이트
- `DELETE /api/v1/sessions/{id}` - 세션 삭제
- `GET /api/v1/sessions` - 세션 목록
- `POST /api/v1/sessions/{id}/reuse` - 세션 공유
- `GET /api/v1/sessions/{id}/summary` - 세션 요약 조회 (준비 중이면 404)
- `POST /api/v1/sessions/{id}/summary?async=true|false` - 동기/비동기 요약 생성
- `GET /api/v1/sessions/{id}/history` - 히스토리 조회 (`num_messages`, `stream_events` 지원)
- `GET /api/v1/agents` - 에이전트 목록
- `POST /api/v1/agents/{id}/run` - 에이전트 실행

### 세션 요약과 재사용 (v1.2.6)

`session.SummaryManager` 를 설정하면 동기/비동기 요약을 사용할 수 있습니다:

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
  - `async=true` 백그라운드 작업을 예약하고 `202 Accepted` 를 반환합니다.
  - `async=false` 즉시 실행하여 생성된 요약을 반환합니다.
- `GET /api/v1/sessions/{id}/summary` 는 최신 스냅샷을 반환하며, 준비 전에는 404 를 반환합니다.
- `POST /api/v1/sessions/{id}/reuse` 는 에이전트, 팀, 워크플로, 사용자 간에 세션을 공유합니다.

### 히스토리 필터와 실행 메타데이터

- `GET /api/v1/sessions/{id}/history?num_messages=20&stream_events=true` 는 최근 N 개만 반환하며 Python 런타임의 SSE 토글과 동일하게 작동합니다.
- 세션 응답에는 실행 메타데이터(`runs[*].status`, 타임스탬프, 취소 사유, `cache_hit`)가 포함되어 감사와 캐시 모니터링을 지원합니다.

**지식 엔드포인트 (선택) / Knowledge Endpoints (optional):**
- `POST /api/v1/knowledge/search` — 지식 베이스에서 벡터 유사도 검색 / Vector similarity search
- `GET  /api/v1/knowledge/config` — 사용 가능한 청커, VectorDB, 임베딩 모델 정보 / Available chunkers, VectorDBs, embedding model
- `POST /api/v1/knowledge/content` — 청크 크기를 설정할 수 있는 최소 적재 (`chunk_size`, `chunk_overlap`, text/plain 또는 application/json) / Minimal ingestion with configurable chunking (`chunk_size`, `chunk_overlap`).

`POST /api/v1/knowledge/content` 는 JSON 과 multipart 업로드 모두에서 `chunk_size` 와 `chunk_overlap` 를 지원합니다. `text/plain` 요청에서는 쿼리 파라미터로 전달하고, 파일 스트리밍 시에는 폼 필드(예: `chunk_size=2000&chunk_overlap=250`)로 전달합니다. 이 값들은 reader 메타데이터에 기록되어 후속 파이프라인이 문서가 어떻게 분할되었는지 확인할 수 있습니다. / `POST /api/v1/knowledge/content` accepts `chunk_size` and `chunk_overlap` in both JSON and multipart uploads. Provide them as query parameters for `text/plain` requests or as form fields (`chunk_size=2000&chunk_overlap=250`) when streaming files. Both values propagate into the reader metadata so downstream pipelines can inspect how documents were segmented.

```bash
curl -X POST http://localhost:8080/api/v1/knowledge/content \
  -F file=@docs/guide.md \
  -F chunk_size=1800 \
  -F chunk_overlap=200 \
  -F metadata='{"source_url":"https://example.com/guide"}'
```

각 청크에는 `chunk_size`, `chunk_overlap`, 사용된 `chunker_type` 이 기록되며 Python AgentOS 응답과 동일하게 동작합니다. / Each stored chunk records `chunk_size`, `chunk_overlap`, and the `chunker_type` used—mirroring the AgentOS Python responses.

요청 예시 / Example:
```bash
curl -X POST http://localhost:8080/api/v1/knowledge/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "에이전트 생성 방법?",
    "limit": 5,
    "filters": {"source": "documentation"}
  }'
```

최소 서버 구성 (지식 API 활성화) / Minimal server config (enable Knowledge API):
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

실행 가능한 예시 / Runnable example: `cmd/examples/knowledge_api/`

## 모범 사례

### 1. 항상 Context 사용하기

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

output, err := ag.Run(ctx, input)
```

### 2. 적절한 에러 처리

```go
output, err := ag.Run(ctx, input)
if err != nil {
    switch {
    case types.IsInvalidInputError(err):
        // 잘못된 입력 처리
    case types.IsRateLimitError(err):
        // 백오프와 함께 재시도
    default:
        // 기타 에러 처리
    }
}
```

### 3. 메모리 관리

```go
// 새 주제를 시작할 때 초기화
ag.ClearMemory()

// 또는 제한된 메모리 사용
mem := memory.NewInMemory(50)
```

### 4. 적절한 타임아웃 설정

```go
server, _ := agentos.NewServer(&agentos.Config{
    RequestTimeout: 60 * time.Second, // 복잡한 에이전트용
})
```
