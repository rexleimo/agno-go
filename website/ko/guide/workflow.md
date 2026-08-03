# Workflow - 단계 기반 오케스트레이션

5가지 강력한 기본 요소로 복잡하고 제어된 다단계 프로세스를 구축하세요.

---

## Workflow란?

**Workflow**는 제어된 AI 에이전트 프로세스를 구축하기 위한 결정론적, 단계 기반 오케스트레이션을 제공합니다. 자율적인 Team과 달리, Workflow는 실행 흐름에 대한 완전한 제어를 제공합니다.

### 주요 기능

- **5가지 기본 요소**: Step, Condition, Loop, Parallel, Router
- **결정론적 실행**: 예측 가능하고 반복 가능한 흐름
- **완전한 제어**: 실행 경로를 직접 결정
- **컨텍스트 공유**: 단계 간 데이터 전달
- **조합 가능**: 복잡한 로직을 위한 기본 요소 중첩

---

## Workflow 생성

### 기본 예제

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
    "github.com/rexleimo/agno-go/pkg/hno/workflow"
    "github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

func main() {
    model, _ := openai.New("gpt-4o-mini", openai.Config{
        APIKey: "your-api-key",
    })

    // 에이전트 생성
    fetchAgent, _ := agent.New(agent.Config{
        Name:  "Fetcher",
        Model: model,
        Instructions: "Fetch data about the topic.",
    })

    processAgent, _ := agent.New(agent.Config{
        Name:  "Processor",
        Model: model,
        Instructions: "Process and analyze the data.",
    })

    // 워크플로우 단계 생성
    step1, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "fetch",
        Agent: fetchAgent,
    })

    step2, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "process",
        Agent: processAgent,
    })

    // 워크플로우 생성
    wf, _ := workflow.New(workflow.Config{
        Name:  "Data Pipeline",
        Steps: []workflow.Node{step1, step2},
    })

    // 워크플로우 실행
    result, _ := wf.Run(context.Background(), "AI trends")
    fmt.Println(result.Output)
}
```

---

## Workflow 기본 요소

### 1. Step

에이전트 또는 커스텀 함수를 실행합니다.

```go
// 에이전트 사용
step, _ := workflow.NewStep(workflow.StepConfig{
    ID:    "analyze",
    Agent: analyzerAgent,
})

// 커스텀 함수 사용
step, _ := workflow.NewStep(workflow.StepConfig{
    ID: "transform",
    Function: func(ctx *workflow.ExecutionContext) (*workflow.StepOutput, error) {
        input := ctx.Input
        // 커스텀 로직
        return &workflow.StepOutput{Output: transformed}, nil
    },
})
```

**사용 사례:**
- 에이전트 실행
- 커스텀 데이터 변환
- API 호출
- 데이터 검증

---

### 2. Condition

컨텍스트 기반 조건 분기.

```go
condition, _ := workflow.NewCondition(workflow.ConditionConfig{
    ID: "check_sentiment",
    Condition: func(ctx *workflow.ExecutionContext) bool {
        result := ctx.GetStepOutput("classify")
        return strings.Contains(result.Output, "positive")
    },
    ThenStep: positiveHandlerStep,
    ElseStep: negativeHandlerStep,
})
```

**사용 사례:**
- 감정 라우팅
- 품질 검사
- 오류 처리
- A/B 테스트 로직

---

### 3. Loop

종료 조건이 있는 반복 실행.

```go
loop, _ := workflow.NewLoop(workflow.LoopConfig{
    ID: "retry",
    Condition: func(ctx *workflow.ExecutionContext) bool {
        result := ctx.GetStepOutput("attempt")
        return result == nil || strings.Contains(result.Output, "error")
    },
    Body:          retryStep,
    MaxIterations: 5,
})
```

**사용 사례:**
- 재시도 로직
- 반복적 개선
- 데이터 처리 루프
- 점진적 개선

---

### 4. Parallel

여러 단계를 동시에 실행합니다.

```go
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
    ID: "gather_data",
    Steps: []workflow.Node{
        sourceStep1,
        sourceStep2,
        sourceStep3,
    },
})
```

**사용 사례:**
- 병렬 데이터 수집
- 다중 소스 집계
- 독립적인 계산
- 성능 최적화

---

### 5. Router

다양한 분기로 동적 라우팅.

```go
router, _ := workflow.NewRouter(workflow.RouterConfig{
    ID: "route_by_type",
    Routes: map[string]workflow.Node{
        "email": emailHandlerStep,
        "chat":  chatHandlerStep,
        "phone": phoneHandlerStep,
    },
    Selector: func(ctx *workflow.ExecutionContext) string {
        if strings.Contains(ctx.Input, "@") {
            return "email"
        }
        return "chat"
    },
})
```

**사용 사례:**
- 입력 타입 라우팅
- 언어 감지
- 우선순위 처리
- 동적 디스패치

---

## 실행 컨텍스트

ExecutionContext는 워크플로우 상태에 대한 액세스를 제공합니다.

### 메서드

```go
type ExecutionContext struct {
    Input string  // 워크플로우 입력
}

// 이전 단계의 출력 가져오기
func (ctx *ExecutionContext) GetStepOutput(stepID string) *StepOutput

// 커스텀 데이터 저장
func (ctx *ExecutionContext) SetData(key string, value interface{})

// 커스텀 데이터 검색
func (ctx *ExecutionContext) GetData(key string) interface{}
```

### 예제

```go
step := workflow.NewStep(workflow.StepConfig{
    ID: "use_context",
    Function: func(ctx *workflow.ExecutionContext) (*workflow.StepOutput, error) {
        // 이전 단계 액세스
        previous := ctx.GetStepOutput("classify")

        // 공유 데이터 액세스
        userData := ctx.GetData("user_info")

        // 처리 및 반환
        result := processData(previous.Output, userData)
        return &workflow.StepOutput{Output: result}, nil
    },
})
```

---

## 완전한 예제

### 조건부 워크플로우

감정 기반 라우팅.

```go
func buildSentimentWorkflow(apiKey string) *workflow.Workflow {
    model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

    classifier, _ := agent.New(agent.Config{
        Name:  "Classifier",
        Model: model,
        Instructions: "Classify sentiment as 'positive' or 'negative'.",
    })

    positiveHandler, _ := agent.New(agent.Config{
        Name:  "PositiveHandler",
        Model: model,
        Instructions: "Thank the user warmly.",
    })

    negativeHandler, _ := agent.New(agent.Config{
        Name:  "NegativeHandler",
        Model: model,
        Instructions: "Apologize and offer help.",
    })

    classifyStep, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "classify",
        Agent: classifier,
    })

    positiveStep, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "positive",
        Agent: positiveHandler,
    })

    negativeStep, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "negative",
        Agent: negativeHandler,
    })

    condition, _ := workflow.NewCondition(workflow.ConditionConfig{
        ID: "route",
        Condition: func(ctx *workflow.ExecutionContext) bool {
            result := ctx.GetStepOutput("classify")
            return strings.Contains(result.Output, "positive")
        },
        ThenStep: positiveStep,
        ElseStep: negativeStep,
    })

    wf, _ := workflow.New(workflow.Config{
        Name:  "Sentiment Router",
        Steps: []workflow.Node{classifyStep, condition},
    })

    return wf
}
```

### 루프 워크플로우

품질 개선을 위한 재시도.

```go
func buildRetryWorkflow(apiKey string) *workflow.Workflow {
    model, _ := openai.New("gpt-4o-mini", openai.Config{APIKey: apiKey})

    generator, _ := agent.New(agent.Config{
        Name:  "Generator",
        Model: model,
        Instructions: "Generate creative content.",
    })

    validator, _ := agent.New(agent.Config{
        Name:  "Validator",
        Model: model,
        Instructions: "Check if content meets quality standards. Return 'pass' or 'fail'.",
    })

    generateStep, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "generate",
        Agent: generator,
    })

    validateStep, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "validate",
        Agent: validator,
    })

    loop, _ := workflow.NewLoop(workflow.LoopConfig{
        ID: "improve",
        Condition: func(ctx *workflow.ExecutionContext) bool {
            result := ctx.GetStepOutput("validate")
            return result != nil && strings.Contains(result.Output, "fail")
        },
        Body:          generateStep,
        MaxIterations: 3,
    })

    wf, _ := workflow.New(workflow.Config{
        Name:  "Quality Loop",
        Steps: []workflow.Node{generateStep, validateStep, loop},
    })

    return wf
}
```

---

## 모범 사례

### 1. 명확한 단계 ID

디버깅을 위해 설명적인 단계 ID 사용:

```go
// 좋음 ✅
ID: "fetch_user_data"

// 나쁨 ❌
ID: "step1"
```

### 2. 오류 처리

각 단계에서 오류 처리:

```go
result, err := wf.Run(ctx, input)
if err != nil {
    log.Printf("Workflow failed at step: %v", err)
    return
}
```

### 3. Context 타임아웃

합리적인 타임아웃 설정:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

result, _ := wf.Run(ctx, input)
```

### 4. 루프 제한

무한 루프를 방지하기 위해 항상 MaxIterations 설정:

```go
loop, _ := workflow.NewLoop(workflow.LoopConfig{
    // ...
    MaxIterations: 10,  // 항상 제한 설정
})
```


## 고급 Workflow 기능

### 체크포인트에서 재개

중단된 워크플로우를 마지막으로 성공한 단계에서 다시 시작합니다:

```go
execCtx, err := wf.Run(ctx, input, "session-id",
    workflow.WithResumeFrom("validate-output"),
)
if err != nil {
    log.Fatal(err)
}
```

`EnableHistory` 를 활성화하면 각 실행의 단계 출력과 취소 기록이 저장됩니다. `WithResumeFrom(stepID)` 를 전달하면 완료된 단계를 건너뛰고 지정된 체크포인트부터 이어집니다.

### 취소 정보 영구 저장

취소 이유, 실행 ID, 타임스탬프를 추적하려면 히스토리를 활성화하세요:

```go
store := workflow.NewMemoryStorage(100)
wf, _ := workflow.New(workflow.Config{
    Name:          "retriable-pipeline",
    Steps:         []workflow.Node{firstStep, finalStep},
    EnableHistory: true,
    HistoryStore:  store,
})
```

어떤 단계에서 컨텍스트를 취소하면 최신 히스토리 항목에 `RunStatusCancelled` 와 `CancellationReason` 이 기록됩니다. `store.GetSession(ctx, sessionID)` 로 확인할 수 있습니다.

### 미디어 페이로드

프롬프트에 직접 삽입하지 않고 이미지, 오디오, 파일을 첨부할 수 있습니다:

```go
// import "github.com/rexleimo/agno-go/pkg/hno/media"
attachments := []media.Attachment{
    {Type: "image", URL: "https://example.com/diagram.png"},
    {Type: "file",  ID:  "spec-v1", Name: "spec.pdf"},
}

execCtx, _ := wf.Run(ctx, "review this", "sess-media",
    workflow.WithMediaPayload(attachments),
)

payload, _ := execCtx.GetSessionState("media_payload")
```

워크플로우는 검증된 첨부 파일을 세션 상태(`media_payload`)에 저장하여 이후 단계와 AgentOS 통합에서 활용할 수 있게 합니다.

---

## Workflow vs Team

각각을 사용할 때:

### Workflow 사용 시:
- 결정론적 실행이 필요할 때
- 단계가 특정 순서로 발생해야 할 때
- 세밀한 제어가 필요할 때
- 디버깅 및 테스트가 중요할 때

### Team 사용 시:
- 에이전트가 자율적으로 작업해야 할 때
- 순서가 중요하지 않을 때 (병렬/합의)
- 창발적 행동을 원할 때
- 제어보다 유연성을 원할 때

---

## 성능 팁

### 병렬 실행

독립적인 단계에 Parallel 사용:

```go
// Sequential: 총 3초
steps := []workflow.Node{step1, step2, step3} // 1s + 1s + 1s

// Parallel: 총 1초
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
    Steps: []workflow.Node{step1, step2, step3}, // max(1s, 1s, 1s)
})
```

### 컨텍스트 재사용

컨텍스트 데이터를 재사용하여 불필요한 LLM 호출 방지:

```go
func (ctx *ExecutionContext) {
    // 비용이 많이 드는 계산 캐시
    if ctx.GetData("cached") == nil {
        expensive := computeExpensiveData()
        ctx.SetData("cached", expensive)
    }
}
```

---

## 다음 단계

- 자율 에이전트를 위한 [Team](/guide/team)과 비교
- 워크플로우 에이전트에 [Tools](/guide/tools) 추가
- 다양한 LLM 제공업체를 위한 [Models](/guide/models) 탐색
- 자세한 문서는 [Workflow API Reference](/api/workflow) 확인

---

## 관련 예제

- [Workflow Demo](/examples/workflow-demo) - 완전한 예제
- [Conditional Routing](/examples/workflow-demo#conditional)
- [Retry Loop Pattern](/examples/workflow-demo#loop)
