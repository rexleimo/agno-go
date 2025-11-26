# 핵심 기능 및 Go API 개요

이 페이지는 현재 Agno-Go에 구현되어 있는 **Go 레벨 API** 를 정리한 것입니다.
지금 시점에서 안정적으로 사용할 수 있는 공개 인터페이스는 다음 두 가지입니다.

- `go/pkg/providers/*` 아래의 각 프로바이더 클라이언트  
- `internal/agent` 및 `internal/model` 에 정의된 공통 데이터 모델  

스펙에 등장하는 HTTP 런타임(` /agents`, `/sessions`, `/messages` 등)은 아직
개발 중이며, **안정된 공개 API 로 간주하지 않는 것** 을 권장합니다.

## 1. 공통 데이터 타입

모델을 호출할 때 자주 사용하는 타입은 다음과 같습니다.

- `agent.ModelConfig` – 사용할 프로바이더/모델과 기본 옵션  
  (`Provider` 열거형, `ModelID`, `Stream`, `MaxTokens`, `Temperature` 등).  
- `agent.Message` – 하나의 메시지. `Role`(`user` / `assistant` / `system`)과
  `Content`(현재는 일반 텍스트)를 가집니다.  
- `model.ChatRequest` – 채팅 요청:

  ```go
  type ChatRequest struct {
    Model    agent.ModelConfig `json:"model"`
    Messages []agent.Message   `json:"messages"`
    Tools    []agent.ToolCall  `json:"tools,omitempty"`
    Metadata map[string]any    `json:"metadata,omitempty"`
    Stream   bool              `json:"stream,omitempty"`
  }
  ```

- `model.ChatResponse` – 한 번의 응답과 사용량 정보:

  ```go
  type ChatResponse struct {
    Message      agent.Message `json:"message"`
    Usage        agent.Usage   `json:"usage,omitempty"`
    FinishReason string        `json:"finishReason,omitempty"`
  }
  ```

- `model.ChatStreamEvent` / `model.StreamHandler` – 토큰 단위 스트리밍용.  
- `model.EmbeddingRequest` / `model.EmbeddingResponse` – embedding 호출용.  
- `model.ChatProvider` / `model.EmbeddingProvider` – 각 프로바이더 클라이언트가
  구현하는 인터페이스.  

## 2. 프로바이더 클라이언트 (`go/pkg/providers/*`)

각 프로바이더 패키지(OpenAI, Gemini, Groq 등)는 `internal/model` 의 인터페이스를
구현합니다. 예를 들어 OpenAI 클라이언트는:

- `go/pkg/providers/openai` 에 위치  
- `New(endpoint, apiKey string, missingEnv []string) *Client` 를 공개  
- `model.ChatProvider` 및 `model.EmbeddingProvider` 를 구현합니다.  

가장 단순한 비스트리밍 채팅 호출은 다음과 같습니다.

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

client := openai.New("", os.Getenv("OPENAI_API_KEY"), nil)

resp, err := client.Chat(ctx, model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   false,
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: "Agno-Go를 간단히 소개해 주세요."},
  },
})
if err != nil {
  log.Fatalf("chat error: %v", err)
}

fmt.Println("assistant:", resp.Message.Content)
```

스트리밍 출력이 필요하다면 동일한 클라이언트의 `Stream` 메서드와
`model.ChatStreamEvent` 를 사용합니다.

```go
req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   true,
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: "짧은 토큰으로 인사를 출력해 주세요."},
  },
}

err := client.Stream(ctx, req, func(ev model.ChatStreamEvent) error {
  if ev.Type == "token" {
    fmt.Print(ev.Delta)
  }
  if ev.Done {
    fmt.Println()
  }
  return nil
})
if err != nil {
  log.Fatalf("stream error: %v", err)
}
```

Embedding 호출도 비슷하게 `EmbeddingRequest` / `EmbeddingResponse` 를 사용합니다.
자세한 예시는 `go/tests/contract` 와 `go/tests/providers` 를 참고하세요.

## 3. Router: 여러 프로바이더 합성

`internal/model.Router` 는 여러 프로바이더 클라이언트를 하나의 디스패처에 묶어 주는
역할을 합니다.

```go
router := model.NewRouter(
  model.WithMaxConcurrency(16),
  model.WithTimeout(30*time.Second),
)

openAI := openai.New("", os.Getenv("OPENAI_API_KEY"), nil)
router.RegisterChatProvider(openAI)

// Gemini, Groq 등 다른 프로바이더도 같은 방식으로 등록할 수 있습니다.

req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: "Hello from router."},
  },
}

resp, err := router.Chat(ctx, req)
if err != nil {
  log.Fatalf("router chat error: %v", err)
}

fmt.Println("assistant:", resp.Message.Content)
```

Router 는 또한:

- `router.Stream(ctx, req, handler)` – 스트리밍 채팅  
- `router.Embed(ctx, embeddingReq)` – embedding 호출  
- `router.Statuses()` – 각 프로바이더 상태 목록(헬스 체크 용도)  

으로도 활용할 수 있습니다. 내부 구현에서도 이 Router 를 사용하고 있으며, 자신의
서비스 안에서 그대로 재사용할 수 있습니다.

## 4. HTTP 런타임 (설계 메모, 아직 불안정)

`specs/001-vitepress-docs/contracts/docs-site-openapi.yaml` 에는 다음과 같은
HTTP 런타임 설계가 포함되어 있습니다.

- `GET /health` – 헬스 체크 및 프로바이더 상태  
- `POST /agents` – 에이전트 정의 생성  
- `POST /agents/{agentId}/sessions` – 세션 생성  
- `POST /agents/{agentId}/sessions/{sessionId}/messages` – 메시지 전송  

하지만 이 HTTP 인터페이스는 아직 설계/구현이 진행 중입니다.

- Go 런타임 구현이 완전히 안정되지 않음  
- 일부 플로우는 아직 외부에 공개되지 않은 `go/cmd/agno` 의 동작에 의존함  

따라서, 현재는 다음과 같은 사용 방식을 권장합니다.

- `go/pkg/providers/*` 를 통해 각 프로바이더를 직접 호출  
- 여러 프로바이더를 조합하고 싶다면, 자신의 서비스 안에서
  `internal/model.Router` 를 사용  

HTTP 런타임과 계약이 안정되면, 별도의 엔드투엔드 예제를 문서에 추가할 예정입니다.

## 5. 리포지토리에서 참고할 위치

- `go/pkg/providers/*` – 각 프로바이더 클라이언트(OpenAI, Gemini, Groq 등)  
- `go/internal/agent` – 에이전트/모델 설정 타입, 사용량 집계 등  
- `go/internal/model` – 요청/응답 타입, Router, Provider 인터페이스  
- `go/tests/providers` – 프로바이더 클라이언트의 실제 사용 예제  
- `go/tests/contract` – HTTP 모양의 데이터 모델을 검증하는 계약 테스트  

자신의 Go 애플리케이션에 예제를 적용할 때는 위 파일들을 최종적인 기준으로 삼는 것을
추천합니다.

