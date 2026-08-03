# 빠른 시작

5분 이내에 Agno-Go를 시작하세요!

## 전제 조건

- Go 1.21 이상
- OpenAI API 키 (또는 Anthropic/Ollama)
- AI 에이전트에 대한 기본 이해

## 설치

### 옵션 1: Go Get 사용

```bash
go get github.com/rexleimo/agno-Go
```

### 옵션 2: 리포지토리 복제

```bash
git clone https://github.com/rexleimo/agno-Go.git
cd agno-Go
go mod download
```

## 첫 번째 Agent

### 1. 간단한 Agent (도구 없음)

`main.go` 파일을 생성하세요:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-Go/pkg/hno/agent"
    "github.com/rexleimo/agno-Go/pkg/hno/models/openai"
)

func main() {
    // 환경에서 API 키 가져오기
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY environment variable is required")
    }

    // OpenAI 모델 생성
    model, err := openai.New("gpt-4o-mini", openai.Config{
        APIKey: apiKey,
    })
    if err != nil {
        log.Fatalf("Failed to create model: %v", err)
    }

    // 에이전트 생성
    ag, err := agent.New(agent.Config{
        Name:         "Assistant",
        Model:        model,
        Instructions: "You are a helpful assistant.",
    })
    if err != nil {
        log.Fatalf("Failed to create agent: %v", err)
    }

    // 에이전트 실행
    output, err := ag.Run(context.Background(), "What is the capital of France?")
    if err != nil {
        log.Fatalf("Agent run failed: %v", err)
    }

    fmt.Println("Agent:", output.Content)
}
```

**실행:**

```bash
export OPENAI_API_KEY=sk-your-key-here
go run main.go
```

**예상 출력:**

```
Agent: The capital of France is Paris.
```

### 2. 도구를 사용하는 Agent

계산기 도구 추가:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-Go/pkg/hno/agent"
    "github.com/rexleimo/agno-Go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-Go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-Go/pkg/hno/tools/toolkit"
)

func main() {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY required")
    }

    // 모델 생성
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: apiKey,
    })

    // 도구를 사용하는 에이전트 생성
    ag, _ := agent.New(agent.Config{
        Name:  "Calculator Agent",
        Model: model,
        Toolkits: []toolkit.Toolkit{
            calculator.New(),
        },
        Instructions: "You are a math assistant. Use the calculator tools for calculations.",
    })

    // 수학 질문하기
    output, _ := ag.Run(context.Background(), "What is 123 * 456 + 789?")

    fmt.Println("Question: What is 123 * 456 + 789?")
    fmt.Println("Agent:", output.Content)
}
```

**실행:**

```bash
go run main.go
```

**예상 출력:**

```
Question: What is 123 * 456 + 789?
Agent: The result is 56,877
```

### 3. 다중 턴 대화

대화를 위한 메모리 추가:

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/rexleimo/agno-Go/pkg/hno/agent"
    "github.com/rexleimo/agno-Go/pkg/hno/models/openai"
)

func main() {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Fatal("OPENAI_API_KEY required")
    }

    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: apiKey,
    })

    ag, _ := agent.New(agent.Config{
        Name:         "Chat Assistant",
        Model:        model,
        Instructions: "You are a friendly chatbot. Remember context from previous messages.",
    })

    fmt.Println("Chat Assistant (type 'quit' to exit)")
    fmt.Println("=====================================")

    scanner := bufio.NewScanner(os.Stdin)
    for {
        fmt.Print("\nYou: ")
        if !scanner.Scan() {
            break
        }

        input := strings.TrimSpace(scanner.Text())
        if input == "quit" || input == "exit" {
            fmt.Println("Goodbye!")
            break
        }

        if input == "" {
            continue
        }

        // 에이전트 실행 (메모리는 자동으로 유지됨)
        output, err := ag.Run(context.Background(), input)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }

        fmt.Printf("Agent: %s\n", output.Content)
    }
}
```

**대화 예시:**

```
You: My name is Alice
Agent: Nice to meet you, Alice! How can I help you today?

You: What's my name?
Agent: Your name is Alice!

You: quit
Goodbye!
```

## AgentOS 사용하기 (HTTP 서버)

### 1. 서버 시작

#### Docker Compose 사용 (권장)

```bash
# 환경 템플릿 복사
cp .env.example .env

# .env를 편집하고 API 키 추가
nano .env  # Add: OPENAI_API_KEY=sk-your-key

# 서버 시작
docker-compose up -d

# 헬스 체크
curl http://localhost:8080/health
```

#### Go 사용 (네이티브)

```bash
# 서버 빌드
go build -o agentos cmd/server/main.go

# 서버 실행
export OPENAI_API_KEY=sk-your-key
./agentos
```

### 2. API 사용

#### 헬스 체크

```bash
curl http://localhost:8080/health
```

**응답:**
```json
{
  "status": "healthy",
  "service": "agentos",
  "time": 1704067200
}
```

#### Agent 실행

```bash
curl -X POST http://localhost:8080/api/v1/agents/assistant/run \
  -H "Content-Type: application/json" \
  -d '{
    "input": "What is 2+2?"
  }'
```

**응답:**
```json
{
  "content": "2 + 2 equals 4.",
  "metadata": {
    "agent_id": "assistant"
  }
}
```

전체 API 문서는 [AgentOS API Reference](/api/agentos)를 참조하세요.

## 다음 단계

### 더 알아보기

- [Core Concepts](/guide/agent) - Agent, Team, Workflow 이해하기
- [Tools Guide](/guide/tools) - 내장 및 커스텀 도구에 대해 알아보기
- [Models Guide](/guide/models) - 다중 모델 지원
- [Advanced Topics](/advanced/) - 아키텍처, 성능, 배포

### 예제 시도하기

모든 예제는 `cmd/examples/` 디렉토리에 있습니다:

```bash
# 계산기를 사용하는 간단한 에이전트
go run cmd/examples/simple_agent/main.go

# Anthropic Claude
go run cmd/examples/claude_agent/main.go

# Ollama를 사용한 로컬 모델
go run cmd/examples/ollama_agent/main.go

# 멀티 에이전트 팀
go run cmd/examples/team_demo/main.go

# 워크플로우 엔진
go run cmd/examples/workflow_demo/main.go

# ChromaDB를 사용한 RAG
go run cmd/examples/rag_demo/main.go
```

각 예제에 대한 자세한 문서는 [Examples](/examples/)를 참조하세요.

## 문제 해결

### 일반적인 문제

**1. "OPENAI_API_KEY not set"**

```bash
export OPENAI_API_KEY=sk-your-key-here
```

**2. "Module not found"**

```bash
go mod download
go mod tidy
```

**3. "Port 8080 already in use"**

`.env` 또는 구성에서 포트 변경:
```bash
AGENTOS_ADDRESS=:9090
```

**4. "Context deadline exceeded"**

타임아웃 증가:
```bash
export REQUEST_TIMEOUT=60
```

### 디버그 로그 가져오기

```bash
export AGENTOS_DEBUG=true
export LOG_LEVEL=debug
```

## 빠른 참조

### 일반적인 임포트

```go
import (
    "github.com/rexleimo/agno-Go/pkg/hno/agent"
    "github.com/rexleimo/agno-Go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-Go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-Go/pkg/hno/team"
    "github.com/rexleimo/agno-Go/pkg/hno/workflow"
    "github.com/rexleimo/agno-Go/pkg/agentos"
)
```

### Agent 생성 템플릿

```go
model, err := openai.New("gpt-4o-mini", openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})

ag, err := agent.New(agent.Config{
    Name:         "Agent Name",
    Model:        model,
    Toolkits:     []toolkit.Toolkit{/* tools */},
    Instructions: "System instructions",
    MaxLoops:     10,
})

output, err := ag.Run(context.Background(), "input")
```

## 다음: 핵심 개념

세 가지 핵심 추상화에 대해 알아보세요:

- [Agent](/guide/agent) - 자율 AI 에이전트
- [Team](/guide/team) - 멀티 에이전트 협업
- [Workflow](/guide/workflow) - 단계 기반 오케스트레이션
