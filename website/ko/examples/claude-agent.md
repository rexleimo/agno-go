# Claude Agent 예제

## 개요

이 예제는 HNO에서 Anthropic의 Claude 모델을 사용하는 방법을 보여줍니다. Claude는 사려 깊고 상세한 응답과 강력한 추론 능력으로 유명합니다. 이 예제는 간단한 대화, 계산기 도구 사용, 복잡한 계산, 수학적 추론을 포함한 여러 사용 사례를 보여줍니다.

## 학습 내용

- HNO에 Anthropic Claude를 통합하는 방법
- Claude 모델(Opus, Sonnet, Haiku)을 구성하는 방법
- 도구 호출 기능과 함께 Claude를 사용하는 방법
- Claude 지침의 모범 사례

## 사전 요구 사항

- Go 1.21 이상
- Anthropic API 키 ([console.anthropic.com](https://console.anthropic.com)에서 발급)

## 설정

1. Anthropic API 키 설정:
```bash
export ANTHROPIC_API_KEY=sk-ant-your-api-key-here
```

2. 예제 디렉토리로 이동:
```bash
cd cmd/examples/claude_agent
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
	"github.com/rexleimo/agno-go/pkg/hno/models/anthropic"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	// Create Anthropic Claude model
	model, err := anthropic.New("claude-3-opus-20240229", anthropic.Config{
		APIKey:      apiKey,
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create calculator toolkit
	calc := calculator.New()

	// Create agent with Claude
	ag, err := agent.New(agent.Config{
		Name:         "Claude Assistant",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calc},
		Instructions: "You are Claude, a helpful AI assistant created by Anthropic. Use the calculator tools to help users with mathematical calculations. Be precise and explain your reasoning.",
		MaxLoops:     10,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Example 1: Simple conversation
	fmt.Println("=== Example 1: Simple Conversation ===")
	ctx := context.Background()
	output, err := ag.Run(ctx, "Introduce yourself in one sentence.")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 2: Using calculator tools
	fmt.Println("=== Example 2: Calculator Tool Usage ===")
	output, err = ag.Run(ctx, "What is 156 multiplied by 23, then subtract 100?")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 3: Complex calculation
	fmt.Println("=== Example 3: Complex Calculation ===")
	output, err = ag.Run(ctx, "Calculate the following: (45 + 67) * 3 - 89. Show your work step by step.")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 4: Mathematical reasoning
	fmt.Println("=== Example 4: Mathematical Reasoning ===")
	output, err = ag.Run(ctx, "If I have $500 and spend $123, then earn $250, how much money do I have? Use the calculator to verify.")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	fmt.Println("✅ All examples completed successfully!")
}
```

## 코드 설명

### 1. Claude 모델 구성

```go
model, err := anthropic.New("claude-3-opus-20240229", anthropic.Config{
	APIKey:      apiKey,
	Temperature: 0.7,
	MaxTokens:   2000,
})
```

**사용 가능한 Claude 모델:**
- `claude-3-opus-20240229` - 가장 강력, 복잡한 작업에 최적
- `claude-3-sonnet-20240229` - 성능과 속도의 균형
- `claude-3-haiku-20240307` - 가장 빠름, 간단한 작업에 최적

**구성 옵션:**
- `Temperature: 0.7` - 창의성 균형 (0.0 = 결정론적, 1.0 = 창의적)
- `MaxTokens: 2000` - 최대 응답 길이

### 2. Claude 전용 지침

```go
Instructions: "You are Claude, a helpful AI assistant created by Anthropic.
Use the calculator tools to help users with mathematical calculations.
Be precise and explain your reasoning."
```

Claude는 다음에 잘 반응합니다:
- 명확한 정체성과 목적
- 도구 사용에 대한 명시적 지침
- 추론 및 설명 강조

### 3. 예제 시나리오

#### 예제 1: 간단한 대화
도구 없이 기본 대화 능력을 테스트합니다.

#### 예제 2: Calculator 도구 사용
```
쿼리: "What is 156 multiplied by 23, then subtract 100?"
예상 흐름:
1. multiply(156, 23) → 3588
2. subtract(3588, 100) → 3488
```

#### 예제 3: 복잡한 계산
```
쿼리: "Calculate: (45 + 67) * 3 - 89. Show your work step by step."
예상 흐름:
1. add(45, 67) → 112
2. multiply(112, 3) → 336
3. subtract(336, 89) → 247
Claude는 각 단계를 설명합니다
```

#### 예제 4: 수학적 추론
Claude의 다음 능력을 테스트합니다:
- 응용 문제 분해
- 적절한 도구 선택
- 명확한 설명 제공

## 예제 실행

```bash
# 옵션 1: 직접 실행
go run main.go

# 옵션 2: 빌드 후 실행
go build -o claude_agent
./claude_agent
```

## 예상 출력

```
=== Example 1: Simple Conversation ===
Agent: I'm Claude, an AI assistant created by Anthropic to be helpful, harmless, and honest.

=== Example 2: Calculator Tool Usage ===
Agent: Let me calculate that for you. First, 156 multiplied by 23 equals 3,588. Then, subtracting 100 from 3,588 gives us 3,488.

=== Example 3: Complex Calculation ===
Agent: I'll solve this step by step:
1. First, calculate the parentheses: 45 + 67 = 112
2. Then multiply: 112 * 3 = 336
3. Finally subtract: 336 - 89 = 247

The final answer is 247.

=== Example 4: Mathematical Reasoning ===
Agent: Let me help you track your money:
- Starting amount: $500
- After spending $123: $500 - $123 = $377
- After earning $250: $377 + $250 = $627

You have $627 in total.

✅ All examples completed successfully!
```

## Claude vs OpenAI

### Claude를 사용해야 하는 경우

**최적 용도:**
- 복잡한 추론 작업
- 상세한 설명
- 안전이 중요한 애플리케이션
- 사려 깊고 미묘한 응답

**특성:**
- 더 자세하고 설명적
- 강력한 윤리적 추론
- 복잡한 지침을 따르는 데 뛰어남
- 불확실성을 인정하는 데 더 나음

### OpenAI를 사용해야 하는 경우

**최적 용도:**
- 빠른 응답
- 코드 생성
- 창의적 글쓰기
- 대규모 함수 호출

## 모델 선택 가이드

| 모델 | 속도 | 능력 | 비용 | 사용 사례 |
|-------|-------|------------|------|----------|
| Claude 3 Opus | 느림 | 최고 | 높음 | 복잡한 분석, 연구 |
| Claude 3 Sonnet | 중간 | 높음 | 중간 | 범용, 균형 |
| Claude 3 Haiku | 빠름 | 좋음 | 낮음 | 간단한 작업, 대량 |

## 구성 팁

### 결정론적 출력을 위한 설정
```go
anthropic.Config{
	Temperature: 0.0,
	MaxTokens:   1000,
}
```

### 창의적 작업을 위한 설정
```go
anthropic.Config{
	Temperature: 1.0,
	MaxTokens:   3000,
}
```

### 프로덕션용(균형)
```go
anthropic.Config{
	Temperature: 0.7,
	MaxTokens:   2000,
}
```

## 모범 사례

1. **명확한 지침**: Claude는 상세하고 구조화된 프롬프트에 잘 반응합니다
2. **추론 요청**: Claude에게 "설명" 또는 "작업 과정 표시"를 요청하면 더 나은 결과를 얻습니다
3. **안전성**: Claude는 더 신중함 - 민감한 쿼리를 적절하게 구성하세요
4. **컨텍스트**: Claude는 200K 토큰 컨텍스트 창을 가지고 있음 - 긴 문서에 활용하세요

## 다음 단계

- [OpenAI Simple Agent](./simple-agent.md)와 비교
- [로컬 모델용 Ollama](./ollama-agent.md) 시도
- [멀티 Agent Team](./team-demo.md) 구축
- [Claude와 RAG](./rag-demo.md) 탐색

## 문제 해결

**오류: "ANTHROPIC_API_KEY environment variable is required"**
- API 키 설정: `export ANTHROPIC_API_KEY=sk-ant-...`

**오류: "model not found"**
- 모델 이름이 정확히 일치하는지 확인: `claude-3-opus-20240229`
- API 계층이 모델에 대한 액세스 권한이 있는지 확인

**Opus로 느린 응답**
- 더 빠른 응답을 위해 Sonnet 사용 고려
- 긴 출력이 필요하지 않으면 MaxTokens 줄이기

**속도 제한 오류**
- Anthropic는 계층별로 다른 속도 제한이 있음
- 지수 백오프로 재시도 로직 구현
- 대량 작업에는 Haiku 사용 고려
