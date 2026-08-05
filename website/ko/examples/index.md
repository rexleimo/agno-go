# 예제

HNO의 모든 기능을 보여주는 실제 예제입니다.

## 사용 가능한 예제

### 1. Simple Agent

계산기 도구를 사용하는 기본 Agent입니다.

**위치**: `cmd/examples/simple_agent/`

**기능**:
- OpenAI GPT-4o-mini 통합
- Calculator 툴킷
- 기본 대화

**실행**:
```bash
export OPENAI_API_KEY=sk-your-key
go run cmd/examples/simple_agent/main.go
```

[소스 코드 보기](https://github.com/rexleimo/agno-go/tree/main/cmd/examples/simple_agent)

---

### 2. Claude Agent

Anthropic Claude와 도구를 통합한 예제입니다.

**위치**: `cmd/examples/claude_agent/`

**기능**:
- Anthropic Claude 3.5 Sonnet
- HTTP 및 Calculator 도구
- 오류 처리 예제

**실행**:
```bash
export ANTHROPIC_API_KEY=sk-ant-your-key
go run cmd/examples/claude_agent/main.go
```

[소스 코드 보기](https://github.com/rexleimo/agno-go/tree/main/cmd/examples/claude_agent)

---

### 3. Ollama Agent

Ollama를 통한 로컬 모델 지원 예제입니다.

**위치**: `cmd/examples/ollama_agent/`

**기능**:
- 로컬 Llama 3 모델
- 프라이버시 중심 (API 호출 없음)
- 파일 작업 툴킷

**설정**:
```bash
# Ollama 설치
curl -fsSL https://ollama.com/install.sh | sh

# 모델 다운로드
ollama pull llama3

# 예제 실행
go run cmd/examples/ollama_agent/main.go
```

[소스 코드 보기](https://github.com/rexleimo/agno-go/tree/main/cmd/examples/ollama_agent)

---

### 4. Team Demo

다양한 협업 모드를 사용한 멀티 Agent 협업 예제입니다.

**위치**: `cmd/examples/team_demo/`

**기능**:
- 4가지 협업 모드 (Sequential, Parallel, Leader-Follower, Consensus)
- Researcher + Writer 팀
- 실제 워크플로우

**실행**:
```bash
export OPENAI_API_KEY=sk-your-key
go run cmd/examples/team_demo/main.go
```

[소스 코드 보기](https://github.com/rexleimo/agno-go/tree/main/cmd/examples/team_demo)

---

### 5. Workflow Demo

제어 흐름 원시 타입을 사용한 단계 기반 오케스트레이션 예제입니다.

**위치**: `cmd/examples/workflow_demo/`

**기능**:
- 5가지 워크플로우 원시 타입 (Step, Condition, Loop, Parallel, Router)
- 감정 분석 워크플로우
- 조건부 라우팅

**실행**:
```bash
export OPENAI_API_KEY=sk-your-key
go run cmd/examples/workflow_demo/main.go
```

[소스 코드 보기](https://github.com/rexleimo/agno-go/tree/main/cmd/examples/workflow_demo)

---

### 6. RAG Demo

ChromaDB를 사용한 검색 증강 생성(RAG) 예제입니다.

**위치**: `cmd/examples/rag_demo/`

**기능**:
- ChromaDB 벡터 데이터베이스
- OpenAI 임베딩
- 의미론적 검색
- 문서 Q&A

**설정**:
```bash
# ChromaDB 시작 (Docker)
docker run -d -p 8000:8000 chromadb/chroma

# API 키 설정
export OPENAI_API_KEY=sk-your-key

# 예제 실행
go run cmd/examples/rag_demo/main.go
```

[소스 코드 보기](https://github.com/rexleimo/agno-go/tree/main/cmd/examples/rag_demo)

---

## 코드 스니펫

### 여러 도구를 사용하는 Agent

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
    "github.com/rexleimo/agno-go/pkg/hno/tools/http"
    "github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    ag, _ := agent.New(agent.Config{
        Name:  "Smart Assistant",
        Model: model,
        Toolkits: []toolkit.Toolkit{
            calculator.New(),
            http.New(),
        },
        Instructions: "You can do math and make HTTP requests",
    })

    output, _ := ag.Run(context.Background(),
        "Calculate 15 * 23 and fetch https://api.github.com")
    fmt.Println(output.Content)
}
```

### 멀티 Agent Team

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-go/pkg/hno/team"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    researcher, _ := agent.New(agent.Config{
        Name:         "Researcher",
        Model:        model,
        Instructions: "Research and gather information",
    })

    writer, _ := agent.New(agent.Config{
        Name:         "Writer",
        Model:        model,
        Instructions: "Create compelling content",
    })

    tm, _ := team.New(team.Config{
        Name:   "Content Team",
        Agents: []*agent.Agent{researcher, writer},
        Mode:   team.ModeSequential,
    })

    output, _ := tm.Run(context.Background(),
        "Write a short article about Go programming")
    fmt.Println(output.Content)
}
```

### 조건문이 있는 Workflow

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
    "github.com/rexleimo/agno-go/pkg/hno/workflow"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    classifier, _ := agent.New(agent.Config{
        Name:         "Classifier",
        Model:        model,
        Instructions: "Classify sentiment as positive or negative",
    })

    positiveHandler, _ := agent.New(agent.Config{
        Name:         "Positive Handler",
        Model:        model,
        Instructions: "Respond enthusiastically",
    })

    negativeHandler, _ := agent.New(agent.Config{
        Name:         "Negative Handler",
        Model:        model,
        Instructions: "Respond empathetically",
    })

    wf, _ := workflow.New(workflow.Config{
        Name: "Sentiment Workflow",
        Steps: []workflow.Primitive{
            workflow.NewStep("classify", classifier),
            workflow.NewCondition("route",
                func(ctx *workflow.ExecutionContext) bool {
                    result := ctx.GetResult("classify")
                    return strings.Contains(result.Content, "positive")
                },
                workflow.NewStep("positive", positiveHandler),
                workflow.NewStep("negative", negativeHandler),
            ),
        },
    })

    output, _ := wf.Run(context.Background(), "I love this!")
    fmt.Println(output.Content)
}
```

## 더 알아보기

- [빠른 시작](/guide/quick-start) - 5분 안에 시작하기
- [Agent 가이드](/guide/agent) - Agent에 대해 배우기
- [Team 가이드](/guide/team) - 멀티 Agent 협업
- [Workflow 가이드](/guide/workflow) - 오케스트레이션 패턴
- [API 레퍼런스](/api/) - 전체 API 문서

## 예제 기여하기

흥미로운 예제가 있나요? 저장소에 기여해주세요:

1. 저장소 Fork
2. `cmd/examples/your_example/`에 예제 작성
3. 설명과 사용법이 포함된 README.md 추가
4. Pull Request 제출

[기여 가이드라인](https://github.com/rexleimo/agno-go/blob/main/CONTRIBUTING.md)
