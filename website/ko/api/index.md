# API 레퍼런스

Agno-Go v1.0의 완전한 API 레퍼런스입니다.

## 핵심 모듈

- [Agent](/api/agent) - 자율 AI 에이전트
- [Team](/api/team) - 다중 에이전트 협업
- [Workflow](/api/workflow) - 단계 기반 오케스트레이션
- [Models](/api/models) - LLM 제공자 통합
- [Tools](/api/tools) - 내장 및 커스텀 도구
- [Memory](/api/memory) - 대화 기록 관리
- [Types](/api/types) - 핵심 타입 및 에러
- [AgentOS Server](/api/agentos) - 프로덕션 HTTP 서버

## 빠른 링크

### Agent

```go
import "github.com/rexleimo/agno-Go/pkg/hno/agent"

agent.New(config) (*Agent, error)
agent.Run(ctx, input) (*RunOutput, error)
agent.ClearMemory()
```

[전체 Agent API →](/api/agent)

### Team

```go
import "github.com/rexleimo/agno-Go/pkg/hno/team"

team.New(config) (*Team, error)
team.Run(ctx, input) (*RunOutput, error)

// 모드: Sequential, Parallel, LeaderFollower, Consensus
```

[전체 Team API →](/api/team)

### Workflow

```go
import "github.com/rexleimo/agno-Go/pkg/hno/workflow"

workflow.New(config) (*Workflow, error)
workflow.Run(ctx, input) (*RunOutput, error)

// 원시 연산: Step, Condition, Loop, Parallel, Router
```

[전체 Workflow API →](/api/workflow)

### Models

```go
import (
    "github.com/rexleimo/agno-Go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-Go/pkg/hno/models/anthropic"
    "github.com/rexleimo/agno-Go/pkg/hno/models/ollama"
)

openai.New(modelID, config) (*OpenAI, error)
anthropic.New(modelID, config) (*Anthropic, error)
ollama.New(modelID, config) (*Ollama, error)
```

[전체 Models API →](/api/models)

### Tools

```go
import (
    "github.com/rexleimo/agno-Go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-Go/pkg/hno/tools/http"
    "github.com/rexleimo/agno-Go/pkg/hno/tools/file"
)

calculator.New() *Calculator
http.New(config) *HTTP
file.New(config) *File
```

[전체 Tools API →](/api/tools)

## 일반적인 패턴

### 에러 처리

```go
import "github.com/rexleimo/agno-Go/pkg/hno/types"

output, err := agent.Run(ctx, input)
if err != nil {
    switch {
    case errors.Is(err, types.ErrInvalidInput):
        // 잘못된 입력 처리
    case errors.Is(err, types.ErrRateLimit):
        // 속도 제한 처리
    default:
        // 기타 에러 처리
    }
}
```

### Context 관리

```go
import (
    "context"
    "time"
)

// 타임아웃 설정
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

output, err := agent.Run(ctx, input)
```

### 동시 에이전트

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

## 타입 정의

### 핵심 타입

```go
// 메시지 타입
type Message struct {
    Role    MessageRole
    Content string
    Name    string
}

// 실행 출력
type RunOutput struct {
    Content  string
    Messages []Message
    Metadata map[string]interface{}
}

// 모델 응답
type ModelResponse struct {
    Content    string
    ToolCalls  []ToolCall
    FinishReason string
}
```

[전체 Types 레퍼런스 →](/api/types)

## AgentOS Server API

프로덕션 배포를 위한 REST API 엔드포인트:

```bash
# 헬스 체크
GET /health

# 에이전트 목록
GET /api/v1/agents

# 에이전트 실행
POST /api/v1/agents/{agent_id}/run

# 세션 생성
POST /api/v1/sessions

# 세션 조회
GET /api/v1/sessions/{session_id}

# 세션 공유 (에이전트/팀 간)
POST /api/v1/sessions/{session_id}/reuse

# 세션 요약 생성 (동기/비동기)
POST /api/v1/sessions/{session_id}/summary?async=true|false

# 세션 요약 스냅샷 조회
GET /api/v1/sessions/{session_id}/summary

# 히스토리 조회 (num_messages, stream_events 필터)
GET /api/v1/sessions/{session_id}/history
```

[전체 AgentOS API →](/api/agentos)

## OpenAPI 명세

완전한 OpenAPI 3.0 명세를 확인하세요:

- [OpenAPI YAML](https://github.com/rexleimo/agno-Go/blob/main/pkg/agentos/openapi.yaml)
- [Swagger UI](https://github.com/rexleimo/agno-Go/tree/main/pkg/agentos#api-documentation)

## 예제

저장소의 실제 작동 예제를 참고하세요:

- [Simple Agent](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/simple_agent)
- [Team Demo](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/team_demo)
- [Workflow Demo](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/workflow_demo)
- [RAG Demo](https://github.com/rexleimo/agno-Go/tree/main/cmd/examples/rag_demo)

## 패키지 문서

전체 Go 패키지 문서는 pkg.go.dev에서 확인할 수 있습니다:

[pkg.go.dev/github.com/rexleimo/agno-Go](https://pkg.go.dev/github.com/rexleimo/agno-Go)
