
# 고급 가이드: 멀티 프로바이더 라우팅 (Go 기반)

이 가이드는 기존 프로바이더 클라이언트와 `internal/model.Router` 를 사용하여
**자신의 Go 서비스 내부에서** 여러 모델 프로바이더 사이를 라우팅하는 방법을 설명합니다.
예제는 모두 Go 코드에 기반하며, 미완성인 HTTP 런타임에 의존하지 않습니다.

## 1. 언제 사용할까

대표적인 사용 사례:

- 일반적인 채팅에는 한 프로바이더를, 지연/비용에 민감한 작업에는 다른 프로바이더를 사용  
- 기본 프로바이더가 장애/레이트 리밋일 때 다른 프로바이더로 자동 페일오버  
- 애플리케이션의 외부 인터페이스를 바꾸지 않고, 새로운 모델을 A/B 테스트  

핵심 아이디어는 **여러 `ChatProvider` 구현을 하나의 API 뒤로 숨기는 것** 입니다.

## 2. 핵심 구성 요소

- `go/pkg/providers/*` – 각 모델 프로바이더의 Go 클라이언트.
  `model.ChatProvider` / `model.EmbeddingProvider` 를 구현합니다.  
- `internal/model.Router` – 등록된 프로바이더로 `ChatRequest` /
  `EmbeddingRequest` 를 라우팅하는 디스패처.  
- `agent.ModelConfig` – 특정 요청에서 사용할 프로바이더/모델을 지정합니다.  

## 3. 예시: OpenAI와 Gemini 를 모두 사용하는 Router

다음 예시는 HTTP 서버 없이 Go 프로세스 안에서만 OpenAI와 Gemini 를 호출하는
라우팅 로직입니다.

```go
package main

import (
  "context"
  "fmt"
  "log"
  "os"
  "time"

  "github.com/rexleimo/agno-go/internal/agent"
  "github.com/rexleimo/agno-go/internal/model"
  "github.com/rexleimo/agno-go/pkg/providers/gemini"
  "github.com/rexleimo/agno-go/pkg/providers/openai"
)

func main() {
  ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
  defer cancel()

  openaiKey := os.Getenv("OPENAI_API_KEY")
  geminiKey := os.Getenv("GEMINI_API_KEY")
  if openaiKey == "" && geminiKey == "" {
    log.Fatal("at least one of OPENAI_API_KEY or GEMINI_API_KEY must be set")
  }

  router := model.NewRouter(
    model.WithMaxConcurrency(16),
    model.WithTimeout(30*time.Second),
  )

  if openaiKey != "" {
    router.RegisterChatProvider(openai.New("", openaiKey, nil))
  }
  if geminiKey != "" {
    router.RegisterChatProvider(gemini.New("", geminiKey, nil))
  }

  // 먼저 OpenAI 를 시도하고, 실패하면 Gemini 로 폴백
  providers := []agent.Provider{agent.ProviderOpenAI, agent.ProviderGemini}

  var lastErr error
  for _, prov := range providers {
    req := model.ChatRequest{
      Model: agent.ModelConfig{
        Provider: prov,
        ModelID:  "gpt-4o-mini", // prov==Gemini 일 때는 적절한 Gemini 모델 ID 로 교체
        Stream:   false,
      },
      Messages: []agent.Message{
        {Role: agent.RoleUser, Content: "내부용 작은 도구에 적합한 저렴하고 빠른 모델을 추천하고 이유를 설명해 주세요."},
      },
    }

    resp, err := router.Chat(ctx, req)
    if err != nil {
      lastErr = err
      log.Printf("provider %s failed: %v", prov, err)
      continue
    }

    fmt.Printf("provider=%s reply=%s\n", prov, resp.Message.Content)
    return
  }

  log.Fatalf("all providers failed, last error: %v", lastErr)
}
```

프로바이더별 모델 ID, 키, 엔드포인트 등은 설정/환경 변수에 숨기고,
애플리케이션 로직은 “어떤 프로바이더를 우선 시도할지”에만 집중할 수 있습니다.

## 4. 스트리밍 버전

같은 Router 를 스트리밍에도 사용할 수 있습니다.

```go
req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   true,
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: "짧은 인사를 스트리밍으로 출력해 주세요."},
  },
}

err := router.Stream(ctx, req, func(ev model.ChatStreamEvent) error {
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

폴백이 필요할 때는 비스트리밍 예제와 마찬가지로 프로바이더별 `ChatRequest` 를 만들어
순차적으로 시도하면 됩니다.

## 5. 구성상의 권장 사항

Go 쪽에서 멀티 프로바이더 라우팅을 구성할 때는:

- API 키와 엔드포인트를 `.env` / `config/default.yaml` 에 두고,
  애플리케이션 코드에 하드코딩하지 않기  
- [프로바이더 매트릭스](../providers/matrix) 를 참고해 어떤 프로바이더/모델을 사용할지 결정하기  
- Router 를 “프로바이더의 구체적인 내용을 아는 유일한 컴포넌트” 로 두고,
  나머지 코드는 `agent.ModelConfig` 와 `model.ChatRequest` 에만 의존하게 하기  

## 6. HTTP 런타임과의 관계

스펙에 나오는 HTTP 런타임(agents / sessions / messages)은 여기서 설명한 개념
(모델 설정, 멀티 프로바이더 라우팅)을 반영하고 있지만, 구현은 아직 안정되지 않았습니다.

따라서 현재로서는:

- 이 문서에서처럼 `go/pkg/providers/*` 와 `internal/model.Router` 를 사용해
  자체 서비스 안에서 라우팅을 구현하고  
- `specs/001-go-agno-rewrite/contracts` 의 HTTP 계약은 데이터 구조 참고용으로만 사용하며,
  이미 운영 중인 외부 API 로 보지 않는 것을  

권장합니다. HTTP 계층이 안정화되면 이 라우팅 패턴은 자연스럽게
공개될 `agents/sessions/messages` 엔드포인트로 이어질 것입니다.

