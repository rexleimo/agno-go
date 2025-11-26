
# 고급 가이드: 메모리 기반 채팅 (Go 기반)

이 가이드는 기존 프로바이더 클라이언트와 사용자 정의 스토리지를 사용하여,
**자신의 Go 애플리케이션 내부에서** 여러 턴/세션에 걸쳐 메모리를 활용하는 채팅 경험을
구현하는 방법을 설명합니다. 모든 예제는 Go 코드에 기반하며 HTTP 런타임에는 의존하지 않습니다.

## 1. 메모리의 세 가지 층

메모리를 크게 세 단계로 나누어 생각하면 설계가 쉬워집니다.

- **대화 히스토리** – 현재 대화 세션에서의 최근 메시지들  
- **사용자 프로필** – 장기적인 선호/설정(학습 스타일, 언어, 플랜 등)  
- **도메인 지식 레코드** – 지원 티켓, 구매 기록, 중요한 이벤트 등의 사실 데이터  

Agno-Go 는 주로 1단계(대화 히스토리)를 위한 원시 타입을 제공하며,
2·3 단계는 애플리케이션과 백엔드 시스템이 관리합니다.

## 2. Go 에서 대화 히스토리 표현하기

Go 에서는 `[]agent.Message` 로 대화 히스토리를 간단히 표현할 수 있습니다.

```go
var history []agent.Message

history = append(history,
  agent.Message{Role: agent.RoleUser, Content: "저는 짧은 학습 세션을 선호해요."},
)

// 모델의 응답도 히스토리에 추가
history = append(history,
  agent.Message{Role: agent.RoleAssistant, Content: "알겠습니다. 한 번에 30분 이내로 유지할게요."},
)
```

모델을 호출할 때는 이 히스토리 일부 또는 전부를 `ChatRequest.Messages` 에 넣어주면 됩니다.

```go
resp, err := client.Chat(ctx, model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
  },
  Messages: history,
})
```

얼마나 많은 히스토리를 유지할지, 언제 요약할지, 어떤 스토리지에 보관할지는
애플리케이션이 결정해야 합니다.

## 3. 장기 메모리 추가하기

장기 메모리는 보통 외부 스토리지에 저장한 후, 필요할 때 프롬프트에 주입합니다.

```go
type UserProfile struct {
  ID          string
  Preferences string // 자연어 요약 형태
}

func buildPrompt(profile UserProfile, recent []agent.Message) string {
  var buf strings.Builder
  buf.WriteString("당신은 친절한 어시스턴트입니다.\n\n")
  buf.WriteString("【사용자 프로필】\n")
  buf.WriteString(profile.Preferences)
  buf.WriteString("\n\n【최근 대화】\n")
  for _, m := range recent {
    buf.WriteString(string(m.Role))
    buf.WriteString(": ")
    buf.WriteString(m.Content)
    buf.WriteString("\n")
  }
  buf.WriteString("\n위 정보를 바탕으로 사용자의 질문에 답변해 주세요.\n")
  return buf.String()
}
```

생성한 프롬프트를 하나의 `user` 메시지로 전송합니다.

```go
prompt := buildPrompt(profile, recentHistory)

resp, err := client.Chat(ctx, model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
  },
  Messages: []agent.Message{
    {Role: agent.RoleUser, Content: prompt},
  },
})
```

애플리케이션은 다음을 책임져야 합니다.

- DB 에서 `UserProfile` 을 읽고/업데이트하기  
- 언제 요약하고 언제 전체 히스토리를 보존할지 결정하기  
- 민감한 데이터가 보안/컴플라이언스 요구 사항에 맞게 처리되는지 확인하기  

## 4. 스트리밍과 결합하기

메모리 기반 채팅은 스트리밍 출력과도 잘 어울립니다.

```go
req := model.ChatRequest{
  Model: agent.ModelConfig{
    Provider: agent.ProviderOpenAI,
    ModelID:  "gpt-4o-mini",
    Stream:   true,
  },
  Messages: history,
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

대화가 끝난 후 최종 어시스턴트 메시지를 `history` 에 추가하면,
다음 요청에서 최신 히스토리를 그대로 사용할 수 있습니다.

## 5. 스토리지 및 구성

메모리를 많이 사용하는 시나리오에서는 다음을 권장합니다.

- 사용자 프로필과 대화 조각을 영속화하기 위해 DB/캐시(Postgres, Redis, 키-값 스토어 등)를 사용  
- [구성 및 보안](../config-and-security) 문서를 참고하여 어떤 프로바이더를 활성화하고
  API 키를 어떻게 관리할지 결정  
- 가능한 한 많은 생 로그를 그대로 프롬프트에 넣지 말고, 선별/요약된 핵심 정보만 전달  

Agno-Go 는 특정 스토리지를 강제하지 않으며, 메시지와 요청 구조만 정의합니다.

## 6. 다른 문서와의 관계

- [퀵스타트](../quickstart) 는 가장 단순한 “무상태” 호출 플로우를 보여줍니다.  
- 이 가이드는 그 위에 애플리케이션 레벨의 메모리 및 스토리지를 추가한 것입니다.  
- 스펙에 정의된 HTTP 런타임도 세션과 메모리 개념을 다루지만, 구현이 안정될 때까지는
  내부 설계로 간주하고 “바로 복사해서 동작하는 기능” 으로 보지 않는 것을 권장합니다.  

