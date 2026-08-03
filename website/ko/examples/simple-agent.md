# Simple Agent 예제

## 개요

이 예제는 Agno-Go를 사용하여 도구 호출 기능을 갖춘 간단한 AI Agent를 만드는 기본 사용법을 보여줍니다. Agent는 OpenAI의 GPT-4o-mini 모델을 사용하며 계산기 툴킷을 장착하여 수학 연산을 수행합니다.

## 학습 내용

- OpenAI 모델을 생성하고 구성하는 방법
- 도구를 사용하여 Agent를 설정하는 방법
- 사용자 쿼리로 Agent를 실행하는 방법
- 실행 메타데이터(루프 수, 토큰 사용량)에 액세스하는 방법

## 사전 요구 사항

- Go 1.21 이상
- OpenAI API 키

## 설정

1. OpenAI API 키 설정:
```bash
export OPENAI_API_KEY=sk-your-api-key-here
```

2. 예제 디렉토리로 이동:
```bash
cd cmd/examples/simple_agent
```

## 전체 코드

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create OpenAI model
	model, err := openai.New("gpt-4o-mini", openai.Config{
		APIKey:      apiKey,
		Temperature: 0.7,
		MaxTokens:   1000,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create calculator toolkit
	calc := calculator.New()

	// Create agent
	ag, err := agent.New(agent.Config{
		Name:         "Math Assistant",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calc},
		Instructions: "You are a helpful math assistant. Use the calculator tools to help users with mathematical calculations.",
		MaxLoops:     10,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Run agent
	ctx := context.Background()
	output, err := ag.Run(ctx, "What is 25 multiplied by 4, then add 15?")
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}

	// Print result
	fmt.Println("Agent Response:")
	fmt.Println(output.Content)
	fmt.Println("\nMetadata:")
	fmt.Printf("Loops: %v\n", output.Metadata["loops"])
	fmt.Printf("Usage: %+v\n", output.Metadata["usage"])
}
```

## 코드 설명

### 1. 모델 구성

```go
model, err := openai.New("gpt-4o-mini", openai.Config{
	APIKey:      apiKey,
	Temperature: 0.7,
	MaxTokens:   1000,
})
```

- GPT-4o-mini를 사용하여 OpenAI 모델 인스턴스 생성
- `Temperature: 0.7`은 창의성과 일관성의 균형을 제공
- `MaxTokens: 1000`은 응답 길이 제한

### 2. Calculator 툴킷

```go
calc := calculator.New()
```

Calculator 툴킷은 네 가지 함수를 제공합니다:
- `add` - 두 숫자의 덧셈
- `subtract` - 두 숫자의 뺄셈
- `multiply` - 두 숫자의 곱셈
- `divide` - 두 숫자의 나눗셈

### 3. Agent 구성

```go
ag, err := agent.New(agent.Config{
	Name:         "Math Assistant",
	Model:        model,
	Toolkits:     []toolkit.Toolkit{calc},
	Instructions: "You are a helpful math assistant...",
	MaxLoops:     10,
})
```

- `Name` - Agent 식별자
- `Model` - 추론에 사용할 LLM
- `Toolkits` - Agent가 사용할 수 있는 도구 컬렉션 배열
- `Instructions` - Agent 동작을 정의하는 시스템 프롬프트
- `MaxLoops` - 최대 도구 호출 반복 횟수 (무한 루프 방지)

### 4. Agent 실행

```go
output, err := ag.Run(ctx, "What is 25 multiplied by 4, then add 15?")
```

Agent는 다음과 같이 동작합니다:
1. 사용자 쿼리 분석
2. 계산기 도구 사용 필요성 판단
3. `multiply(25, 4)` 호출하여 100 획득
4. `add(100, 15)` 호출하여 115 획득
5. 자연어 응답 반환

## 예제 실행

```bash
# 옵션 1: 직접 실행
go run main.go

# 옵션 2: 빌드 후 실행
go build -o simple_agent
./simple_agent
```

## 예상 출력

```
Agent Response:
The result of 25 multiplied by 4 is 100, and when you add 15 to that, you get 115.

Metadata:
Loops: 2
Usage: map[completion_tokens:45 prompt_tokens:234 total_tokens:279]
```

## 주요 개념

### 도구 호출 루프

`MaxLoops` 매개변수는 Agent가 도구를 호출할 수 있는 횟수를 제어합니다:

1. **Loop 1**: Agent가 `multiply(25, 4)` 호출 → 결과 수신: 100
2. **Loop 2**: Agent가 `add(100, 15)` 호출 → 결과 수신: 115
3. **최종**: Agent가 자연어 응답 생성

각 루프는 도구 호출 및 결과 처리의 한 라운드를 나타냅니다.

### 메타데이터

`output.Metadata`에는 유용한 실행 정보가 포함되어 있습니다:
- `loops` - 수행된 도구 호출 반복 횟수
- `usage` - 토큰 소비량 (프롬프트, 완료, 전체)

## 다음 단계

- [Claude Agent 예제](./claude-agent.md)에서 Anthropic 통합 살펴보기
- [Team 협업](./team-demo.md)으로 여러 Agent 사용하기
- [Workflow 엔진](./workflow-demo.md)으로 복잡한 프로세스 시도하기
- [RAG 애플리케이션](./rag-demo.md)으로 지식 검색 구축하기

## 문제 해결

**오류: "OPENAI_API_KEY environment variable is required"**
- API 키를 내보냈는지 확인: `export OPENAI_API_KEY=sk-...`

**오류: "model not found"**
- GPT-4o-mini 모델에 대한 액세스 권한이 있는지 확인
- 대안으로 "gpt-3.5-turbo" 사용 시도

**오류: "max loops exceeded"**
- Agent가 MaxLoops 제한(10)에 도달
- `MaxLoops`를 늘리거나 쿼리를 단순화
