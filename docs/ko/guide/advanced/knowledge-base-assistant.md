
# 고급 가이드: 지식 베이스 어시스턴트 (Go 기반)

이 가이드는 **자신의 문서와 검색 인프라** 를 사용하여 질문에 답변하는 어시스턴트를,
Agno-Go 의 Go 프로바이더 클라이언트와 데이터 모델로 구성하는 방법을 설명합니다.
미완성인 HTTP 런타임에 의존하지 않고, Go 코드에만 집중합니다.

## 1. 시나리오 개요

어시스턴트가 답변해야 할 대상 예시는 다음과 같습니다.

- 제품 문서  
- 내부 가이드라인/정책  
- 지식 베이스 기사  

일반적인 패턴은 다음과 같습니다.

1. 오프라인에서 문서를 embedding 하고, 벡터 + 메타데이터를 벡터 스토어나 DB 에 저장  
2. 질문을 받으면 벡터 검색 등으로 관련도가 높은 문서 조각들을 조회  
3. 조회된 문서 조각을 `agent.Message.Content` 에 컨텍스트로 포함해 모델에 질문  

Agno-Go 는 벡터 스토어를 제공하지 않으며, embedding 과 chat 호출만 담당합니다.

## 2. 프로바이더 클라이언트로 문서 embedding 하기

`model.EmbeddingProvider` 를 구현한 어떤 클라이언트든 사용할 수 있습니다.
구체적인 모델 ID 및 지원 여부는 프로바이더 매트릭스와 `.env.example` 을 참고하세요.

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
  "github.com/rexleimo/agno-go/pkg/providers/openai"
)

func embedDoc(ctx context.Context, text string) ([]float64, error) {
  apiKey := os.Getenv("OPENAI_API_KEY")
  if apiKey == "" {
    return nil, fmt.Errorf("OPENAI_API_KEY not set")
  }

  client := openai.New("", apiKey, nil)

  resp, err := client.Embed(ctx, model.EmbeddingRequest{
    Model: agent.ModelConfig{
      Provider: agent.ProviderOpenAI,
      ModelID:  "text-embedding-3-small", // 적절한 embedding 모델 선택
    },
    Input: []string{text},
  })
  if err != nil {
    return nil, err
  }
  if len(resp.Vectors) == 0 {
    return nil, fmt.Errorf("empty embedding response")
  }
  return resp.Vectors[0], nil
}
```

벡터를 어디에 저장할지는 (Postgres, ClickHouse, 전용 벡터 DB 등) 애플리케이션의 선택입니다.

## 3. 컨텍스트를 포함해 질문에 답변하기

질문에 대해 관련 문서 조각들(`[]string`)을 조회했다면, 다음과 같이 프롬프트를 구성해 모델에 전달할 수 있습니다.

```go
func answerWithContext(
  ctx context.Context,
  client model.ChatProvider,
  provider agent.Provider,
  modelID string,
  question string,
  passages []string,
) (string, error) {
  var contextText string
  for _, p := range passages {
    contextText += "- " + p + "\n"
  }

  prompt := fmt.Sprintf(
    "당신은 친절한 어시스턴트입니다.\n\n컨텍스트:\n%s\n질문: %s\n\n반드시 컨텍스트에 있는 정보만 바탕으로 답변하고, 정보가 없으면 '모르겠습니다' 라고 답변하세요.",
    contextText,
    question,
  )

  resp, err := client.Chat(ctx, model.ChatRequest{
    Model: agent.ModelConfig{
      Provider: provider,
      ModelID:  modelID,
    },
    Messages: []agent.Message{
      {Role: agent.RoleUser, Content: prompt},
    },
  })
  if err != nil {
    return "", err
  }
  return resp.Message.Content, nil
}
```

여기서 `client` 는 OpenAI, Gemini, Groq 등 어느 `ChatProvider` 여도 상관없으며,
해당 프로바이더의 환경 변수가 `.env` 에 설정되어 있어야 합니다.

## 4. 전체 구조 정리

완전한 지식 베이스 어시스턴트는 보통 다음 세 부분으로 구성됩니다.

- **인덱서** – 문서를 읽어 `Embed` 를 호출하고, 벡터 + 메타데이터를 저장  
- **리트리버** – 질문에 대해 관련도가 높은 문서 조각을 검색하여 반환  
- **앱서** – 위 패턴대로 컨텍스트를 프롬프트에 포함하고 `Chat` 을 호출해 답변 생성  

이 가운데 Agno-Go 의 책임은 다음과 같습니다.

- 일관된 `ChatRequest` / `EmbeddingRequest` 구조 제공  
- 공통 인터페이스를 구현한 프로바이더 클라이언트 제공  
- 기본적인 에러 처리 및 프로바이더 상태 표현을 통일  

스토리지, 인덱싱, 랭킹 등의 세부 설계는 애플리케이션 영역입니다.

## 5. 다른 문서와의 관계

- [프로바이더 매트릭스](../providers/matrix) 를 사용해 긴 컨텍스트에 적합한 프로바이더와
  모델을 선택하세요.  
- [구성 및 보안](../config-and-security) 문서를 참고해 `OPENAI_API_KEY`, `GEMINI_API_KEY`
  등 필요한 환경 변수를 설정하세요.  
- 스펙에 정의된 HTTP 런타임은 여기서 설명한 개념(컨텍스트 + 채팅)을 그대로 반영하고 있지만,
  구현이 안정되기 전까지는 “데이터 구조 참고용” 으로만 사용하고 바로 복사해서 쓸 수 있는
  구현으로 보지 않는 것을 권장합니다.  

