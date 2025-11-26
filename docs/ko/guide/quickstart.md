# 퀵스타트: Go 코드에서 Agno-Go 프로바이더 호출하기

이 가이드는 리포지토리에 포함된 **Go 프로바이더 클라이언트(`go/pkg/providers/*`)** 를 사용해서
가장 간단한 채팅 호출을 만드는 방법을 설명합니다. 현재 1차 버전의 목표는 다음과 같습니다.

1. 프로바이더(예: OpenAI) 환경 변수를 설정한다  
2. `go/pkg/providers/openai` 를 사용해 Go 코드에서 호출한다  
3. 응답 메시지를 출력해서 확인한다  

> 주의: AgentOS HTTP 런타임(`/agents`, `/sessions`, `/messages` 등)은 아직 정리 중인 설계입니다.
> 이 퀵스타트는 **HTTP 서버나 curl 예제에 의존하지 않고**, 테스트에서 이미 사용 중인 Go 프로바이더 클라이언트만을 다룹니다.

## 사전 준비

1. Go 1.25.1 설치
2. 프로젝트 루트에서 환경 변수 파일 준비:

```bash
cd <your-project-root>
cp .env.example .env
```

`.env` 에 OpenAI 키를 설정합니다.

```bash
OPENAI_API_KEY=your-openai-key
```

## 최소 예제: OpenAI Chat 호출

아래 코드는 리포지토리의 테스트와 동일한 형태로,

- `internal/agent`  
- `internal/model`  
- `go/pkg/providers/openai`  

를 그대로 재사용합니다.

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

func main() {
  ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
  defer cancel()

  apiKey := os.Getenv("OPENAI_API_KEY")
  if apiKey == "" {
    log.Fatal("OPENAI_API_KEY not set")
  }

  // 기본 OpenAI 퍼블릭 엔드포인트 사용 (.env 의 OPENAI_ENDPOINT 로 프록시를 설정할 수도 있음)
  client := openai.New("", apiKey, nil)

  resp, err := client.Chat(ctx, model.ChatRequest{
    Model: agent.ModelConfig{
      Provider: agent.ProviderOpenAI,
      ModelID:  "gpt-4o-mini",
      Stream:   false,
    },
    Messages: []agent.Message{
      {Role: agent.RoleUser, Content: "Agno-Go를 짧게 소개해 주세요."},
    },
  })
  if err != nil {
    log.Fatalf("chat error: %v", err)
  }

  fmt.Println("assistant:", resp.Message.Content)
}
```

위 코드를 다음 경로에 저장합니다.

```bash
<your-project-root>/examples/openai_quickstart/main.go
```

프로젝트 루트에서 실행합니다.

```bash
cd <your-project-root>
go run ./examples/openai_quickstart
```

모델에 따라 내용은 달라지겠지만, 대략 다음과 같은 출력이 나와야 합니다.

```text
assistant: Agno-Go는 Go로 구현된 AgentOS이며, ...
```

## 다음 단계

- [구성 및 보안](./config-and-security) 문서를 읽고, 각 프로바이더의 키/엔드포인트/런타임 옵션을 어떻게 설정할지 확인하세요  
- [프로바이더 매트릭스](./providers/matrix)에서 각 프로바이더가 지원하는 Chat / Embedding / Streaming 기능과 필요한 환경 변수를 확인하세요  
- AgentOS HTTP 런타임(agents / sessions / messages)은 아직 안정화 중이며, 향후 별도의 문서에서 다룰 예정입니다. 그 전까지는 `go/pkg/providers/*`를 주요 공개 엔트리 포인트로 사용하는 것을 권장합니다  

