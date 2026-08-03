# Agent

**Agent**는 도구를 사용하고, 대화 컨텍스트를 유지하며, 독립적으로 작업을 실행할 수 있는 자율 AI 개체입니다.

## 개요

```go
import "github.com/rexleimo/HNO/pkg/hno/agent"

agent, err := agent.New(agent.Config{
    Name:         "My Agent",
    Model:        model,
    Toolkits:     []toolkit.Toolkit{calculator.New()},
    Instructions: "You are a helpful assistant",
    MaxLoops:     10,
})

output, err := agent.Run(context.Background(), "What is 2+2?")
```

## 구성

### Config 구조체

```go
type Config struct {
    Name         string            // Agent 이름
    Model        models.Model      // LLM 모델
    Toolkits     []toolkit.Toolkit // 사용 가능한 도구
    Memory       memory.Memory     // 대화 메모리
    Instructions string            // 시스템 지침
    MaxLoops     int               // 최대 도구 호출 루프 (기본값: 10)
    PreHooks     []hooks.Hook      // 실행 전 훅
    PostHooks    []hooks.Hook      // 실행 후 훅
}
```

### 매개변수

- **Name** (필수): 사람이 읽을 수 있는 에이전트 식별자
- **Model** (필수): LLM 모델 인스턴스 (OpenAI, Claude 등)
- **Toolkits** (선택): 사용 가능한 도구 목록
- **Memory** (선택): 기본값은 100개 메시지 제한이 있는 인메모리 스토리지
- **Instructions** (선택): 시스템 프롬프트/페르소나
- **MaxLoops** (선택): 무한 도구 호출 루프 방지 (기본값: 10)
- **PreHooks** (선택): 실행 전 검증 훅
- **PostHooks** (선택): 실행 후 검증 훅

## 기본 사용법

### 간단한 Agent

```go
package main

import (
    "context"
    "fmt"
    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    ag, _ := agent.New(agent.Config{
        Name:         "Assistant",
        Model:        model,
        Instructions: "You are a helpful assistant",
    })

    output, _ := ag.Run(context.Background(), "Hello!")
    fmt.Println(output.Content)
}
```

### 도구를 사용하는 Agent

```go
import (
    "github.com/rexleimo/HNO/pkg/hno/tools/calculator"
    "github.com/rexleimo/HNO/pkg/hno/tools/http"
)

ag, _ := agent.New(agent.Config{
    Name:  "Smart Assistant",
    Model: model,
    Toolkits: []toolkit.Toolkit{
        calculator.New(),
        http.New(),
    },
    Instructions: "You can do math and make HTTP requests",
})

output, _ := ag.Run(ctx, "Calculate 15 * 23 and fetch https://api.github.com")
```

## 고급 기능

### 커스텀 메모리

```go
import "github.com/rexleimo/HNO/pkg/hno/memory"

// 커스텀 제한이 있는 메모리 생성
mem := memory.NewInMemory(50) // 최근 50개 메시지 유지

ag, _ := agent.New(agent.Config{
    Memory: mem,
    // ... 기타 구성
})
```

### 훅 및 가드레일

훅으로 입력 및 출력 검증:

```go
import "github.com/rexleimo/HNO/pkg/hno/guardrails"

// 내장 프롬프트 인젝션 가드
promptGuard := guardrails.NewPromptInjectionGuardrail()

// 커스텀 검증 훅
customHook := func(ctx context.Context, input *hooks.HookInput) error {
    if len(input.Input) > 1000 {
        return fmt.Errorf("input too long")
    }
    return nil
}

ag, _ := agent.New(agent.Config{
    PreHooks:  []hooks.Hook{customHook, promptGuard},
    PostHooks: []hooks.Hook{outputValidator},
    // ... 기타 구성
})
```

### Context 및 타임아웃

```go
import "time"

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

output, err := ag.Run(ctx, "Complex task...")
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        fmt.Println("Timeout!")
    }
}
```

### 응답 캐시 (v1.2.6)

캐시를 활성화하면 모델 출력을 재사용하여 결정적인 응답을 받을 수 있습니다:

```go
ag, _ := agent.New(agent.Config{
    Model:       model,
    EnableCache: true,
    CacheTTL:    2 * time.Minute,
})

first, _ := ag.Run(ctx, "Summarise REST vs gRPC")
second, _ := ag.Run(ctx, "Summarise REST vs gRPC")

if cached, _ := second.Metadata["cache_hit"].(bool); cached {
    // Handle cached response
}
```

Redis나 공유 저장소를 사용하려면 `cache.Provider` 를 교체하세요. 기본값은 인메모리 LRU입니다.

## Run 출력

`Run` 메서드는 `*RunOutput`을 반환합니다:

```go
type RunOutput struct {
    Content  string                 // Agent의 응답
    Messages []types.Message        // 전체 메시지 히스토리
    Metadata map[string]interface{} // 추가 데이터
}
```

예제:

```go
output, err := ag.Run(ctx, "Tell me a joke")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Response:", output.Content)
fmt.Println("Messages:", len(output.Messages))
fmt.Println("Metadata:", output.Metadata)
```

## 메모리 관리

### 메모리 지우기

```go
// 모든 대화 히스토리 지우기
ag.ClearMemory()
```

### 메모리 액세스

```go
// 현재 메시지 가져오기
messages := ag.GetMemory().GetMessages()
fmt.Println("History:", len(messages))
```

## 오류 처리

```go
output, err := ag.Run(ctx, input)
if err != nil {
    switch {
    case errors.Is(err, types.ErrInvalidInput):
        // 잘못된 입력 처리
    case errors.Is(err, types.ErrRateLimit):
        // 요율 제한 처리
    case errors.Is(err, context.DeadlineExceeded):
        // 타임아웃 처리
    default:
        // 기타 오류 처리
    }
}
```

## 모범 사례

### 1. 항상 Context 사용

```go
ctx := context.Background()
output, err := ag.Run(ctx, input)
```

### 2. 적절한 MaxLoops 설정

```go
// 간단한 작업용
MaxLoops: 5

// 복잡한 추론용
MaxLoops: 15
```

### 3. 명확한 지침 제공

```go
Instructions: `You are a customer support agent.
- Be polite and professional
- Use tools to look up information
- If unsure, ask for clarification`
```

### 4. 타입 안전 도구 구성 사용

```go
calc := calculator.New()
httpClient := http.New(http.Config{
    Timeout: 10 * time.Second,
})

ag, _ := agent.New(agent.Config{
    Toolkits: []toolkit.Toolkit{calc, httpClient},
})
```

## 성능 고려사항

- **Agent 생성**: 평균 ~180ns
- **메모리 사용량**: 에이전트당 ~1.2KB
- **동시 Agent**: 완전히 스레드 안전, goroutine을 자유롭게 사용

```go
// 동시 에이전트
for i := 0; i < 100; i++ {
    go func(id int) {
        ag, _ := agent.New(config)
        output, _ := ag.Run(ctx, input)
        fmt.Printf("Agent %d: %s\n", id, output.Content)
    }(i)
}
```

## 예제

실제 예제를 확인하세요:

- [Simple Agent](https://github.com/rexleimo/HNO/tree/main/cmd/examples/simple_agent)
- [Claude Agent](https://github.com/rexleimo/HNO/tree/main/cmd/examples/claude_agent)
- [Ollama Agent](https://github.com/rexleimo/HNO/tree/main/cmd/examples/ollama_agent)
- [Agent with Guardrails](https://github.com/rexleimo/HNO/tree/main/cmd/examples/agent_with_guardrails)

## API 레퍼런스

전체 API 문서는 [Agent API Reference](/api/agent)를 참조하세요.

## 다음 단계

- [Team](/guide/team) - 멀티 에이전트 협업
- [Workflow](/guide/workflow) - 단계 기반 오케스트레이션
- [Tools](/guide/tools) - 내장 및 커스텀 도구
- [Models](/guide/models) - LLM 제공업체 구성
