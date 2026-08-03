# 아키텍처

HNO는 단순성, 효율성, 확장성을 위해 설계된 깨끗하고 모듈화된 아키텍처를 따릅니다.

## 핵심 철학

**단순하고, 효율적이며, 확장 가능하게**

## 전체 아키텍처

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

## 핵심 인터페이스

### 1. Model 인터페이스

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

### 2. Toolkit 인터페이스

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

### 3. Memory 인터페이스

```go
type Memory interface {
    Add(message types.Message) error
    GetMessages() []types.Message
    Clear() error
}
```

## 구성 요소 상세

### Agent

**파일**: `pkg/hno/agent/agent.go`

다음을 수행하는 자율적인 AI 엔티티:
- 추론을 위해 LLM 사용
- 도구 호출 가능
- 대화 메모리 유지
- 훅을 통한 입출력 검증

**주요 메서드**:
```go
New(config Config) (*Agent, error)
Run(ctx context.Context, input string) (*RunOutput, error)
ClearMemory()
```

### Team

**파일**: `pkg/hno/team/team.go`

4가지 조정 모드를 가진 다중 에이전트 협업:

1. **Sequential** - 에이전트가 차례대로 작업
2. **Parallel** - 모든 에이전트가 동시에 작업
3. **LeaderFollower** - 리더가 팔로워에게 작업 위임
4. **Consensus** - 에이전트들이 합의에 도달할 때까지 토론

### Workflow

**파일**: `pkg/hno/workflow/workflow.go`

5가지 프리미티브를 가진 단계 기반 오케스트레이션:

1. **Step** - 에이전트 또는 함수 실행
2. **Condition** - 컨텍스트 기반 분기
3. **Loop** - 종료 조건이 있는 반복
4. **Parallel** - 단계의 동시 실행
5. **Router** - 동적 라우팅

### Models

**디렉토리**: `pkg/hno/models/`

LLM 제공자 구현:
- `openai/` - OpenAI GPT 모델
- `anthropic/` - Anthropic Claude 모델
- `ollama/` - Ollama 로컬 모델
- `deepseek/`, `gemini/`, `modelscope/` - 기타 제공자

### Tools

**디렉토리**: `pkg/hno/tools/`

확장 가능한 툴킷 시스템:
- `calculator/` - 수학 연산
- `http/` - HTTP 요청
- `file/` - 파일 작업
- `search/` - 웹 검색

## AgentOS 프로덕션 서버

**디렉토리**: `pkg/agentos/`

다음을 포함하는 프로덕션용 HTTP 서버:

- RESTful API 엔드포인트
- 세션 관리
- 에이전트 레지스트리
- 상태 모니터링
- CORS 지원
- 요청 타임아웃 처리

**아키텍처**:
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

## 디자인 패턴

### 1. 인터페이스 기반 디자인

모든 핵심 구성 요소는 유연성을 위해 인터페이스를 사용:

```go
type Model interface { /* ... */ }
type Toolkit interface { /* ... */ }
type Memory interface { /* ... */ }
```

### 2. 상속보다 조합

에이전트는 모델, 도구, 메모리를 조합:

```go
type Agent struct {
    Model    Model
    Toolkits []Toolkit
    Memory   Memory
    // ...
}
```

### 3. 컨텍스트 전파

모든 작업은 취소 및 타임아웃을 위해 `context.Context`를 받음:

```go
func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error)
```

### 4. 에러 래핑

래핑된 에러를 통한 일관된 에러 처리:

```go
if err != nil {
    return nil, fmt.Errorf("failed to run agent: %w", err)
}
```

## 성능 최적화

### 1. 낮은 할당 횟수

- 최소 힙 할당 (에이전트당 8-9회)
- 사전 할당된 슬라이스
- 적절한 문자열 인터닝

### 2. 효율적인 메모리 레이아웃

```go
type Agent struct {
    ID           string   // 16B
    Name         string   // 16B
    Model        Model    // 16B (interface)
    // Total: ~112B struct + heap allocations
}
```

### 3. 고루틴 안전성

- 전역 상태 없음
- 설계상 스레드 안전
- 가능한 한 락 프리

## 동시성 모델

### Agent 동시성

```go
// Safe to create multiple agents concurrently
for i := 0; i < 100; i++ {
    go func() {
        ag, _ := agent.New(config)
        output, _ := ag.Run(ctx, input)
    }()
}
```

### Team Parallel 모드

```go
// Agents run in parallel goroutines
team := team.New(team.Config{
    Mode: team.ModeParallel,
    Agents: agents,
})
```

### Workflow Parallel 단계

```go
// Steps execute concurrently
workflow.NewParallel("tasks", []Primitive{
    step1, step2, step3,
})
```

## 확장 지점

### 1. 커스텀 모델

`Model` 인터페이스 구현:

```go
type MyModel struct{}

func (m *MyModel) Invoke(ctx context.Context, req *InvokeRequest) (*ModelResponse, error) {
    // Custom implementation
}
```

### 2. 커스텀 도구

`BaseToolkit` 확장:

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

### 3. 커스텀 메모리

`Memory` 인터페이스 구현:

```go
type MyMemory struct{}

func (m *MyMemory) Add(msg types.Message) error {
    // Custom storage
}
```

## 테스트 전략

### 단위 테스트

- 각 패키지에 `*_test.go` 파일
- 인터페이스용 모의 구현
- 테이블 기반 테스트

### 통합 테스트

- 엔드투엔드 워크플로우 테스트
- 다중 에이전트 시나리오
- 실제 API 통합 테스트

### 벤치마크 테스트

- `*_bench_test.go`의 성능 벤치마크
- 메모리 할당 추적
- 동시성 스트레스 테스트

## 의존성

### 핵심 의존성

- **Go 표준 라이브러리** - 대부분의 기능
- **무거운 프레임워크 없음** - 경량 디자인

### 선택적 의존성

- LLM 제공자 SDK (OpenAI, Anthropic 등)
- 벡터 데이터베이스 클라이언트 (ChromaDB)
- HTTP 클라이언트 라이브러리

## 향후 아키텍처

### 계획된 개선 사항

1. **스트리밍 지원** - 실시간 응답 스트리밍
2. **플러그인 시스템** - 동적 도구 로딩
3. **분산 에이전트** - 다중 노드 배포
4. **고급 메모리** - 영구 저장소, 벡터 메모리

## 모범 사례

### 1. 인터페이스 사용

```go
var model models.Model = openai.New(...)
```

### 2. 에러 처리

```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### 3. 컨텍스트 사용

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### 4. 단순하게 유지

KISS 원칙을 따르세요 - 과도한 엔지니어링 지양.

## 참고 자료

- [성능 벤치마크](/advanced/performance)
- [배포 가이드](/advanced/deployment)
- [API 참조](/api/)
- [소스 코드](https://github.com/rexleimo/HNO)
