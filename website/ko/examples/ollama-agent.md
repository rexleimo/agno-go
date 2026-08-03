# Ollama Agent 예제

## 개요

이 예제는 Ollama를 통해 HNO에서 로컬 LLM을 사용하는 방법을 보여줍니다. Ollama를 사용하면 강력한 언어 모델을 로컬 컴퓨터에서 실행할 수 있어 프라이버시, 비용 절감, 오프라인 기능을 제공합니다. 개발, 테스트 및 프라이버시가 중요한 애플리케이션에 완벽합니다.

## 학습 내용

- HNO에 Ollama를 통합하는 방법
- 로컬 LLM으로 Agent를 실행하는 방법
- 로컬 모델과 도구 호출을 사용하는 방법
- 로컬 모델의 장점과 한계

## 사전 요구 사항

- Go 1.21 이상
- Ollama 설치 ([ollama.ai](https://ollama.ai))
- 로컬 모델 다운로드 (예: llama2, mistral, codellama)

## Ollama 설정

### 1. Ollama 설치

**macOS/Linux:**
```bash
curl -fsSL https://ollama.ai/install.sh | sh
```

**Windows:**
[ollama.ai/download](https://ollama.ai/download)에서 다운로드

### 2. 모델 다운로드

```bash
# Llama 2 다운로드 (7B 파라미터, ~4GB)
ollama pull llama2

# 또는 다른 모델 시도:
ollama pull mistral      # Mistral 7B
ollama pull codellama    # 코드 특화
ollama pull llama2:13b   # 더 크고 강력함
```

### 3. Ollama 서버 시작

```bash
ollama serve
```

서버는 기본적으로 `http://localhost:11434`에서 실행됩니다.

### 4. 설치 확인

```bash
# 모델 테스트
ollama run llama2 "Hello, how are you?"
```

## 전체 코드

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/ollama"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
	// Create Ollama model (uses local Ollama instance)
	// Make sure Ollama is running: ollama serve
	model, err := ollama.New("llama2", ollama.Config{
		BaseURL:     "http://localhost:11434",
		Temperature: 0.7,
		MaxTokens:   2000,
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Create calculator toolkit
	calc := calculator.New()

	// Create agent with Ollama
	ag, err := agent.New(agent.Config{
		Name:         "Ollama Assistant",
		Model:        model,
		Toolkits:     []toolkit.Toolkit{calc},
		Instructions: "You are a helpful AI assistant running on Ollama. You can use calculator tools to help with math. Be concise and friendly.",
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
	output, err = ag.Run(ctx, "What is 456 multiplied by 789?")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	// Example 3: Complex calculation
	fmt.Println("=== Example 3: Complex Calculation ===")
	output, err = ag.Run(ctx, "Calculate: (100 + 50) * 2 - 75")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", output.Content)

	fmt.Println("✅ All examples completed successfully!")
}
```

## 코드 설명

### 1. Ollama 모델 구성

```go
model, err := ollama.New("llama2", ollama.Config{
	BaseURL:     "http://localhost:11434",
	Temperature: 0.7,
	MaxTokens:   2000,
})
```

**구성 옵션:**
- **Model Name**: 다운로드한 모델과 일치해야 함 (예: "llama2", "mistral")
- **BaseURL**: Ollama 서버 주소 (기본값: `http://localhost:11434`)
- **Temperature**: 0.0 (결정론적) ~ 2.0 (매우 창의적)
- **MaxTokens**: 최대 응답 길이

### 2. API 키 불필요

OpenAI나 Anthropic과 달리 Ollama는 로컬에서 실행됩니다:
- ✅ API 키 불필요
- ✅ 사용 비용 없음
- ✅ 완전한 프라이버시
- ✅ 오프라인 작동

### 3. 도구 지원

로컬 모델도 클라우드 모델처럼 도구를 사용할 수 있습니다:
```go
Toolkits: []toolkit.Toolkit{calc}
```

Agent는 필요할 때 계산기 함수를 호출합니다.

## 예제 실행

### 단계 1: Ollama 시작
```bash
# 터미널 1
ollama serve
```

### 단계 2: 예제 실행
```bash
# 터미널 2
cd cmd/examples/ollama_agent
go run main.go
```

## 예상 출력

```
=== Example 1: Simple Conversation ===
Agent: I'm a helpful AI assistant running on Ollama, here to assist you with questions and tasks.

=== Example 2: Calculator Tool Usage ===
Agent: Let me calculate that for you. 456 multiplied by 789 equals 359,784.

=== Example 3: Complex Calculation ===
Agent: Let me solve this step by step:
- First: 100 + 50 = 150
- Then: 150 * 2 = 300
- Finally: 300 - 75 = 225

The answer is 225.

✅ All examples completed successfully!
```

## 사용 가능한 모델

### 범용

| 모델 | 크기 | RAM | 설명 |
|-------|------|-----|-------------|
| llama2 | 7B | 8GB | Meta의 Llama 2, 범용 |
| llama2:13b | 13B | 16GB | 더 크고 강력한 버전 |
| mistral | 7B | 8GB | Mistral AI, 뛰어난 품질 |
| mixtral | 47B | 32GB | 전문가 혼합, 매우 강력함 |

### 전문화

| 모델 | 사용 사례 |
|-------|----------|
| codellama | 코드 생성 및 분석 |
| llama2-uncensored | 더 적은 콘텐츠 제한 |
| orca-mini | 더 작고 빠름 (3B) |
| vicuna | 대화 및 채팅 |

### 사용 가능한 모델 목록
```bash
ollama list
```

### 특정 모델 다운로드
```bash
ollama pull mistral
ollama pull codellama:13b
```

## 구성 예제

### 속도 우선 (작은 모델)
```go
ollama.Config{
	Model:       "orca-mini",
	Temperature: 0.5,
	MaxTokens:   500,
}
```

### 품질 우선 (큰 모델)
```go
ollama.Config{
	Model:       "mixtral",
	Temperature: 0.7,
	MaxTokens:   3000,
}
```

### 코드 작업용
```go
ollama.Config{
	Model:       "codellama",
	Temperature: 0.3,  // 코드에 더 결정론적
	MaxTokens:   2000,
}
```

### 커스텀 Ollama 서버
```go
ollama.Config{
	BaseURL:     "http://192.168.1.100:11434",  // 원격 Ollama
	Model:       "llama2",
	Temperature: 0.7,
}
```

## 성능 고려 사항

### 속도 요인

1. **모델 크기**: 작은 모델(7B)이 큰 모델(70B)보다 빠름
2. **하드웨어**: GPU가 추론을 크게 가속화
3. **컨텍스트 길이**: 긴 대화는 응답 속도를 늦춤

### 일반적인 응답 시간

| 모델 | 하드웨어 | 속도 |
|-------|----------|-------|
| llama2 (7B) | Mac M1 | ~1-2초 |
| mistral (7B) | Mac M1 | ~1-2초 |
| mixtral (47B) | Mac M1 | ~5-10초 |
| llama2 (13B) | NVIDIA 3090 | ~0.5-1초 |

## 로컬 모델의 장점

### ✅ 장점

1. **프라이버시**: 데이터가 컴퓨터를 떠나지 않음
2. **비용**: API 비용 없음, 무제한 사용
3. **오프라인**: 인터넷 없이 작동
4. **제어**: 모델과 데이터에 대한 완전한 제어
5. **커스터마이징**: 특정 작업을 위해 모델 미세 조정

### ⚠️ 한계

1. **품질**: 일반적으로 GPT-4나 Claude Opus보다 낮음
2. **속도**: 클라우드 API보다 느림 (고성능 GPU가 아닌 경우)
3. **리소스**: RAM/VRAM 필요 (4-16GB+)
4. **유지 관리**: 모델과 업데이트 관리 필요

## 모범 사례

### 1. 올바른 모델 선택

```bash
# 개발/테스트용
ollama pull orca-mini  # 빠름, 3B 파라미터

# 프로덕션용
ollama pull mistral    # 속도/품질의 좋은 균형

# 복잡한 작업용
ollama pull mixtral    # 고품질, 더 많은 리소스 필요
```

### 2. 지침 최적화

로컬 모델은 간결하고 명확한 지침에서 이점을 얻습니다:

```go
// ✅ 좋음
Instructions: "You are a math assistant. Use calculator tools for calculations. Be concise."

// ❌ 너무 장황함
Instructions: "You are an extremely sophisticated mathematical assistant with deep knowledge..."
```

### 3. 리소스 사용 모니터링

```bash
# Ollama 상태 확인
ollama ps

# 모델 정보 보기
ollama show llama2
```

### 4. 오류를 우아하게 처리

```go
output, err := ag.Run(ctx, userQuery)
if err != nil {
	// Ollama가 다운되었을 수 있음
	log.Printf("Ollama error: %v. Is the server running?", err)
	// 클라우드 모델로 대체하거나 오류 반환
}
```

## 통합 패턴

### 하이브리드 접근 방식

개발에는 Ollama, 프로덕션에는 클라우드 사용:

```go
var model models.Model

if os.Getenv("ENV") == "production" {
	model, _ = openai.New("gpt-4o-mini", openai.Config{...})
} else {
	model, _ = ollama.New("llama2", ollama.Config{...})
}
```

### 프라이버시 우선 애플리케이션

```go
// 민감한 데이터에 Ollama 사용
sensitiveAgent, _ := agent.New(agent.Config{
	Model: ollamaModel,
	Instructions: "Handle user PII securely...",
})
```

## 문제 해결

### 오류: "connection refused"
```bash
# Ollama가 실행 중인지 확인
ollama serve

# 또는 프로세스 확인
ps aux | grep ollama
```

### 오류: "model not found"
```bash
# 먼저 모델 다운로드
ollama pull llama2

# 사용 가능한지 확인
ollama list
```

### 느린 응답
```bash
# 더 작은 모델 시도
ollama pull orca-mini

# 또는 하드웨어 가속 확인
ollama show llama2 | grep -i gpu
```

### 메모리 부족
```bash
# 더 작은 모델 사용
ollama pull orca-mini  # 7B 대신 3B

# 또는 스왑 공간 늘리기 (Linux)
# 또는 다른 애플리케이션 종료
```

## 다음 단계

- [OpenAI Agent](./simple-agent.md) 및 [Claude Agent](./claude-agent.md)와 비교
- [멀티 Agent Team](./team-demo.md)에서 로컬 모델 사용
- 로컬 임베딩으로 [프라이버시 보호 RAG](./rag-demo.md) 구축
- 로컬 모델로 [Workflow](./workflow-demo.md) 탐색

## 추가 리소스

- [Ollama 문서](https://github.com/ollama/ollama/blob/main/README.md)
- [Ollama 모델 라이브러리](https://ollama.ai/library)
- [하드웨어 요구 사항](https://github.com/ollama/ollama/blob/main/docs/gpu.md)
- [모델 비교](https://ollama.ai/blog/model-comparison)
