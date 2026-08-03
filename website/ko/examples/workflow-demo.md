# Workflow 엔진 예제

## 개요

이 예제는 5가지 원시 빌딩 블록(Step, Condition, Loop, Parallel, Router)을 사용한 HNO의 강력한 워크플로우 엔진을 보여줍니다. Workflow는 결정론적이고 제어된 실행 흐름을 제공하여 정밀한 제어와 관찰 가능성이 필요한 복잡한 다단계 프로세스에 완벽합니다.

## 학습 내용

- 5가지 원시 타입으로 워크플로우를 구축하는 방법
- 순차, 조건부, 병렬 실행 패턴
- 사용자 정의 종료 조건으로 루프를 만드는 방법
- 실행을 동적으로 라우팅하는 방법
- 원시 타입을 결합하여 복잡한 워크플로우를 만드는 방법

## 사전 요구 사항

- Go 1.21 이상
- OpenAI API 키

## 설정

```bash
export OPENAI_API_KEY=sk-your-api-key-here
cd cmd/examples/workflow_demo
```

## Workflow 원시 타입

### 1. Step - 기본 실행 단위

Agent 또는 사용자 정의 함수를 실행합니다.

```go
step, _ := workflow.NewStep(workflow.StepConfig{
	ID:    "research",
	Agent: researchAgent,
})
```

### 2. Condition - 분기 로직

조건에 따라 다른 경로로 라우팅합니다.

```go
condition, _ := workflow.NewCondition(workflow.ConditionConfig{
	ID: "sentiment-check",
	Condition: func(ctx *workflow.ExecutionContext) bool {
		return strings.Contains(ctx.Output, "positive")
	},
	TrueNode:  positiveStep,
	FalseNode: negativeStep,
})
```

### 3. Loop - 반복 실행

조건이 충족될 때까지 단계를 반복합니다.

```go
loop, _ := workflow.NewLoop(workflow.LoopConfig{
	ID:   "refinement",
	Body: refineStep,
	Condition: func(ctx *workflow.ExecutionContext, iteration int) bool {
		return iteration < 3  // 3번 실행
	},
})
```

### 4. Parallel - 동시 실행

여러 단계를 동시에 실행합니다.

```go
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
	ID:    "analysis",
	Nodes: []workflow.Node{techStep, bizStep, ethicsStep},
})
```

### 5. Router - 동적 라우팅

런타임 로직에 따라 다른 단계로 라우팅합니다.

```go
router, _ := workflow.NewRouter(workflow.RouterConfig{
	ID: "task-router",
	Router: func(ctx *workflow.ExecutionContext) string {
		if strings.Contains(ctx.Output, "calculation") {
			return "calc"
		}
		return "general"
	},
	Routes: map[string]workflow.Node{
		"calc":    calcStep,
		"general": generalStep,
	},
})
```

## 전체 예제

### 데모 1: Sequential Workflow

기본 파이프라인: 연구 → 분석 → 작성

```go
func runSequentialWorkflow(ctx context.Context, apiKey string) {
	// Create agents for pipeline
	researcher := createAgent("researcher", apiKey,
		"You are a researcher. Gather facts about the topic.")
	analyzer := createAgent("analyzer", apiKey,
		"You are an analyst. Analyze the facts and draw conclusions.")
	writer := createAgent("writer", apiKey,
		"You are a writer. Write a concise summary.")

	// Create steps
	step1, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "research",
		Agent: researcher,
	})

	step2, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "analyze",
		Agent: analyzer,
	})

	step3, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "write",
		Agent: writer,
	})

	// Create workflow
	wf, _ := workflow.New(workflow.Config{
		Name:  "Content Pipeline",
		Steps: []workflow.Node{step1, step2, step3},
	})

	result, _ := wf.Run(ctx, "The impact of renewable energy on climate change")
	fmt.Printf("Final Output: %s\n", result.Output)
}
```

**흐름:**
```
Input → Researcher → Analyzer → Writer → Output
```

### 데모 2: Conditional Workflow

분기 로직을 사용한 감정 분석

```go
func runConditionalWorkflow(ctx context.Context, apiKey string) {
	classifier := createAgent("classifier", apiKey,
		"Classify the sentiment as positive or negative. Respond with just 'positive' or 'negative'.")
	positiveHandler := createAgent("positive", apiKey,
		"You handle positive feedback. Thank the user warmly.")
	negativeHandler := createAgent("negative", apiKey,
		"You handle negative feedback. Apologize and offer help.")

	// Classification step
	classifyStep, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "classify",
		Agent: classifier,
	})

	// Positive branch
	positiveStep, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "positive-response",
		Agent: positiveHandler,
	})

	// Negative branch
	negativeStep, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "negative-response",
		Agent: negativeHandler,
	})

	// Conditional node
	condition, _ := workflow.NewCondition(workflow.ConditionConfig{
		ID: "sentiment-branch",
		Condition: func(ctx *workflow.ExecutionContext) bool {
			return strings.Contains(strings.ToLower(ctx.Output), "positive")
		},
		TrueNode:  positiveStep,
		FalseNode: negativeStep,
	})

	// Create workflow
	wf, _ := workflow.New(workflow.Config{
		Name:  "Sentiment Handler",
		Steps: []workflow.Node{classifyStep, condition},
	})

	result, _ := wf.Run(ctx, "Your product is amazing! I love it!")
	fmt.Printf("Response: %s\n", result.Output)
}
```

**흐름:**
```
Input → Classify → [Positive?] → Positive Handler
                   ↓ [Negative]
                   → Negative Handler
```

### 데모 3: Loop Workflow

반복적인 텍스트 개선

```go
func runLoopWorkflow(ctx context.Context, apiKey string) {
	refiner := createAgent("refiner", apiKey,
		"Refine and improve the given text. Make it more concise.")

	// Loop body
	refineStep, _ := workflow.NewStep(workflow.StepConfig{
		ID:    "refine",
		Agent: refiner,
	})

	// Loop 3 times for iterative refinement
	loop, _ := workflow.NewLoop(workflow.LoopConfig{
		ID:   "refinement-loop",
		Body: refineStep,
		Condition: func(ctx *workflow.ExecutionContext, iteration int) bool {
			return iteration < 3
		},
	})

	wf, _ := workflow.New(workflow.Config{
		Name:  "Iterative Refinement",
		Steps: []workflow.Node{loop},
	})

	result, _ := wf.Run(ctx, "AI is a technology that enables machines...")

	iterations, _ := result.Get("loop_refinement-loop_iterations")
	fmt.Printf("Refined after %v iterations: %s\n", iterations, result.Output)
}
```

**흐름:**
```
Input → [Refine] → [Iteration < 3?] → [Yes] → Refine again
          ↑                           ↓ [No]
          └─────────────────────────── Output
```

### 데모 4: Parallel Workflow

동시에 실행되는 다각적 분석

```go
func runParallelWorkflow(ctx context.Context, apiKey string) {
	techAgent := createAgent("tech", apiKey, "Analyze technical aspects in 1-2 sentences.")
	bizAgent := createAgent("biz", apiKey, "Analyze business aspects in 1-2 sentences.")
	ethicsAgent := createAgent("ethics", apiKey, "Analyze ethical aspects in 1-2 sentences.")

	techStep, _ := workflow.NewStep(workflow.StepConfig{ID: "tech-analysis", Agent: techAgent})
	bizStep, _ := workflow.NewStep(workflow.StepConfig{ID: "biz-analysis", Agent: bizAgent})
	ethicsStep, _ := workflow.NewStep(workflow.StepConfig{ID: "ethics-analysis", Agent: ethicsAgent})

	parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
		ID:    "multi-perspective",
		Nodes: []workflow.Node{techStep, bizStep, ethicsStep},
	})

	wf, _ := workflow.New(workflow.Config{
		Name:  "Parallel Analysis",
		Steps: []workflow.Node{parallel},
	})

	result, _ := wf.Run(ctx, "The use of facial recognition technology in public spaces")

	// Access individual results
	tech, _ := result.Get("parallel_multi-perspective_branch_0_output")
	biz, _ := result.Get("parallel_multi-perspective_branch_1_output")
	ethics, _ := result.Get("parallel_multi-perspective_branch_2_output")

	fmt.Printf("Tech: %v\nBusiness: %v\nEthics: %v\n", tech, biz, ethics)
}
```

**흐름:**
```
Input → [Tech Agent]    → Combine
      → [Biz Agent]     → Results
      → [Ethics Agent]  → Output
      (모두 동시에 실행)
```

### 데모 5: Router를 사용한 복잡한 Workflow

작업 유형에 따른 동적 라우팅

```go
func runComplexWorkflow(ctx context.Context, apiKey string) {
	// Router determines task type
	router := createAgent("router", apiKey,
		"Determine if this is a 'calculation' or 'general' task. Respond with just the word.")

	// Calculation route
	calcAgent := createAgent("calculator", apiKey,
		"You perform calculations.", calculator.New())

	// General route
	generalAgent := createAgent("general", apiKey,
		"You handle general questions.")

	// Create steps
	routerStep, _ := workflow.NewStep(workflow.StepConfig{ID: "router", Agent: router})
	calcStep, _ := workflow.NewStep(workflow.StepConfig{ID: "calc-task", Agent: calcAgent})
	generalStep, _ := workflow.NewStep(workflow.StepConfig{ID: "general-task", Agent: generalAgent})

	// Router node
	routerNode, _ := workflow.NewRouter(workflow.RouterConfig{
		ID: "task-router",
		Router: func(ctx *workflow.ExecutionContext) string {
			if strings.Contains(strings.ToLower(ctx.Output), "calculation") {
				return "calc"
			}
			return "general"
		},
		Routes: map[string]workflow.Node{
			"calc":    calcStep,
			"general": generalStep,
		},
	})

	wf, _ := workflow.New(workflow.Config{
		Name:  "Smart Router",
		Steps: []workflow.Node{routerStep, routerNode},
	})

	// Test with calculation
	result1, _ := wf.Run(ctx, "What is 25 * 4 + 100?")
	fmt.Printf("Calculation result: %s\n", result1.Output)

	// Test with general question
	result2, _ := wf.Run(ctx, "What is the capital of France?")
	fmt.Printf("General result: %s\n", result2.Output)
}
```

**흐름:**
```
Input → Router → [Type?] → Calc Agent    (계산인 경우)
                         → General Agent  (일반인 경우)
```

## 예제 실행

```bash
go run main.go
```

## 예상 출력

```
=== Demo 1: Sequential Workflow ===
Final Output: Renewable energy significantly reduces greenhouse gas emissions, helping combat climate change by replacing fossil fuels with clean power sources like solar and wind.

=== Demo 2: Conditional Workflow ===
Response: Thank you so much for your wonderful feedback! We're thrilled that you love our product!

=== Demo 3: Loop Workflow ===
Refined after 3 iterations: AI enables machines to learn, reason, and understand language.

=== Demo 4: Parallel Workflow ===
Tech: Uses computer vision and deep learning for pattern recognition in real-time.
Business: Creates new security markets but raises privacy-related costs.
Ethics: Raises serious concerns about surveillance, consent, and civil liberties.

=== Demo 5: Complex Workflow with Router ===
Calculation result: 25 * 4 + 100 equals 200.
General result: The capital of France is Paris.
```

## 실행 컨텍스트

워크플로우 상태 및 결과 액세스:

```go
result, _ := wf.Run(ctx, input)

// 주요 출력
fmt.Println(result.Output)

// 컨텍스트에서 특정 값 가져오기
value, exists := result.Get("step_id_output")

// 실행 성공 확인
if result.Error != nil {
	log.Printf("Workflow error: %v", result.Error)
}
```

### 컨텍스트 키

Workflow는 예측 가능한 키로 데이터를 저장합니다:

- **Step 출력**: `step_{step-id}_output`
- **Loop 반복**: `loop_{loop-id}_iterations`
- **Parallel 분기**: `parallel_{parallel-id}_branch_{index}_output`
- **Condition 결과**: `condition_{condition-id}_result`

## Workflow vs Team

| 기능 | Workflow | Team |
|---------|----------|------|
| **제어** | 높음 - 명시적 흐름 | 낮음 - Agent 자체 구성 |
| **유연성** | 낮음 - 사전 정의된 경로 | 높음 - 동적 협업 |
| **관찰 가능성** | 높음 - 모든 단계 추적 | 중간 - Agent 출력 |
| **사용 사례** | 결정론적 프로세스 | 창의적 협업 |
| **디버깅** | 쉬움 - 단계별 | 어려움 - 창발적 동작 |

**Workflow를 선택해야 하는 경우:**
- 실행에 대한 정밀한 제어가 필요할 때
- 디버깅과 관찰 가능성이 중요할 때
- 프로세스에 잘 정의된 단계가 있을 때
- 규정 준수/감사 요구 사항이 있을 때

**Team을 선택해야 하는 경우:**
- 솔루션 경로가 불확실할 때
- Agent가 창의적으로 협업하기를 원할 때
- 작업이 여러 관점에서 이익을 얻을 때
- 유연성이 제어보다 중요할 때

## 설계 패턴

### 1. 파이프라인 패턴
```go
Steps: []workflow.Node{step1, step2, step3}
// 선형 흐름: A → B → C
```

### 2. 분기 패턴
```go
Condition → TrueNode
         → FalseNode
// If-else 로직
```

### 3. 재시도 패턴
```go
Loop with condition checking success
// 성공하거나 최대 시도 횟수까지 재시도
```

### 4. Fan-Out 패턴
```go
Parallel → Multiple agents simultaneously
// 작업 분산, 결과 수집
```

### 5. 상태 머신 패턴
```go
Router → Different states based on output
// 상태 기반 동적 라우팅
```

## 모범 사례

### 1. 관찰 가능성을 위한 설계

```go
// ✅ 좋음: 명확하고 설명적인 ID
workflow.StepConfig{
	ID:    "validate-input",
	Agent: validator,
}

// ❌ 나쁨: 모호한 ID
workflow.StepConfig{
	ID:    "step1",
	Agent: validator,
}
```

### 2. 오류를 우아하게 처리

```go
result, err := wf.Run(ctx, input)
if err != nil {
	log.Printf("Workflow failed: %v", err)
	// 대체 로직
}

if result.Error != nil {
	log.Printf("Step error: %v", result.Error)
}
```

### 3. 단계를 집중적으로 유지

```go
// ✅ 좋음: 단일 책임
extractStep  := "Extract data from input"
validateStep := "Validate extracted data"
saveStep     := "Save validated data"

// ❌ 나쁨: 한 단계에 너무 많음
megaStep := "Extract, validate, transform, and save data"
```

### 4. 병렬 실행 최적화

```go
// 독립적인 작업에 parallel 사용
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
	Nodes: []workflow.Node{
		fetchUserData,
		fetchOrderData,
		fetchInventoryData,
	},
})
```

### 5. 공유 상태에 컨텍스트 사용

```go
// 단계는 이전 출력에 액세스 가능
analyzer := createAgent("analyzer", apiKey,
	"Analyze the research data provided in the previous step.")
```

## 고급 기능

### Step의 사용자 정의 함수

```go
step, _ := workflow.NewStep(workflow.StepConfig{
	ID: "custom-processing",
	Function: func(ctx context.Context, input string) (string, error) {
		// Agent 없이 사용자 정의 로직
		return processData(input), nil
	},
})
```

### 동적 루프 조건

```go
loop, _ := workflow.NewLoop(workflow.LoopConfig{
	Body: improveStep,
	Condition: func(ctx *workflow.ExecutionContext, iteration int) bool {
		// 품질 임계값이 충족되면 종료
		quality := assessQuality(ctx.Output)
		return quality < 0.9 && iteration < 10
	},
})
```

### 조건부 라우팅

```go
router, _ := workflow.NewRouter(workflow.RouterConfig{
	Router: func(ctx *workflow.ExecutionContext) string {
		// 콘텐츠 기반 라우팅
		if containsCode(ctx.Output) {
			return "code-review"
		}
		if containsData(ctx.Output) {
			return "data-analysis"
		}
		return "general"
	},
	Routes: map[string]workflow.Node{...},
})
```

## 성능 팁

1. **단계 최소화**: 각 단계가 지연 시간을 추가
2. **Parallel 사용**: 독립적인 작업은 동시에 실행되어야 함
3. **루프 제한**: 합리적인 최대 반복 횟수 설정
4. **결과 캐싱**: 컨텍스트에 비용이 많이 드는 계산 저장
5. **빠른 모델 선택**: 속도가 중요한 단계에는 gpt-4o-mini 사용

## 다음 단계

- 다양한 사용 사례를 위해 [Team 협업](./team-demo.md)과 비교
- 검색 단계로 [RAG Workflow](./rag-demo.md) 구축
- 다양한 [모델 공급자](./claude-agent.md)와 워크플로우 결합
- 워크플로우 단계용 [사용자 정의 도구](./simple-agent.md) 생성

## 문제 해결

**Workflow가 루프에서 멈춤:**
- 루프 조건 로직 확인
- 최대 반복 제한 추가
- 디버깅을 위해 반복 횟수 로깅

**Parallel 단계가 동시에 실행되지 않음:**
- Parallel 노드에 있는지 확인, 순차 단계가 아님
- 공유 리소스/잠금 확인

**컨텍스트 값에 액세스할 수 없음:**
- 올바른 키 형식 사용: `{type}_{id}_{field}`
- 단계 ID가 정확히 일치하는지 확인
- 단계가 성공적으로 실행되었는지 확인

**Router가 항상 같은 경로를 선택:**
- Router 함수 출력 로깅
- 조건 로직 확인
- 경로가 올바르게 정의되었는지 확인
