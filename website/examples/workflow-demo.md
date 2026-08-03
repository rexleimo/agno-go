# Workflow Engine Example

## Overview

This example demonstrates HNO's powerful workflow engine with 5 primitive building blocks: Step, Condition, Loop, Parallel, and Router. Workflows provide deterministic, controlled execution flow - perfect for complex multi-step processes that require precise control and observability.

## What You'll Learn

- How to build workflows with 5 primitive types
- Sequential, conditional, and parallel execution patterns
- How to create loops with custom exit conditions
- How to route execution dynamically
- How to combine primitives into complex workflows

## Prerequisites

- Go 1.21 or higher
- OpenAI API key

## Setup

```bash
export OPENAI_API_KEY=sk-your-api-key-here
cd cmd/examples/workflow_demo
```

## Workflow Primitives

### 1. Step - Basic Execution Unit

Executes an agent or custom function.

```go
step, _ := workflow.NewStep(workflow.StepConfig{
	ID:    "research",
	Agent: researchAgent,
})
```

### 2. Condition - Branching Logic

Routes to different paths based on a condition.

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

### 3. Loop - Iterative Execution

Repeats a step until a condition is met.

```go
loop, _ := workflow.NewLoop(workflow.LoopConfig{
	ID:   "refinement",
	Body: refineStep,
	Condition: func(ctx *workflow.ExecutionContext, iteration int) bool {
		return iteration < 3  // Run 3 times
	},
})
```

### 4. Parallel - Concurrent Execution

Runs multiple steps simultaneously.

```go
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
	ID:    "analysis",
	Nodes: []workflow.Node{techStep, bizStep, ethicsStep},
})
```

### 5. Router - Dynamic Routing

Routes to different steps based on runtime logic.

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

## Complete Examples

### Demo 1: Sequential Workflow

Basic pipeline: Research → Analyze → Write

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

**Flow:**
```
Input → Researcher → Analyzer → Writer → Output
```

### Demo 2: Conditional Workflow

Sentiment analysis with branching logic.

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

**Flow:**
```
Input → Classify → [Positive?] → Positive Handler
                   ↓ [Negative]
                   → Negative Handler
```

### Demo 3: Loop Workflow

Iterative text refinement.

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

**Flow:**
```
Input → [Refine] → [Iteration < 3?] → [Yes] → Refine again
          ↑                           ↓ [No]
          └─────────────────────────── Output
```

### Demo 4: Parallel Workflow

Multi-perspective analysis running concurrently.

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

**Flow:**
```
Input → [Tech Agent]    → Combine
      → [Biz Agent]     → Results
      → [Ethics Agent]  → Output
      (All run simultaneously)
```

### Demo 5: Complex Workflow with Router

Dynamic routing based on task type.

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

**Flow:**
```
Input → Router → [Type?] → Calc Agent    (if calculation)
                         → General Agent  (if general)
```

## Running the Example

```bash
go run main.go
```

## Expected Output

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

## Execution Context

Access workflow state and results:

```go
result, _ := wf.Run(ctx, input)

// Main output
fmt.Println(result.Output)

// Get specific values from context
value, exists := result.Get("step_id_output")

// Check execution success
if result.Error != nil {
	log.Printf("Workflow error: %v", result.Error)
}
```

### Context Keys

Workflows store data with predictable keys:

- **Step output**: `step_{step-id}_output`
- **Loop iterations**: `loop_{loop-id}_iterations`
- **Parallel branches**: `parallel_{parallel-id}_branch_{index}_output`
- **Condition result**: `condition_{condition-id}_result`

## Workflow vs Team

| Feature | Workflow | Team |
|---------|----------|------|
| **Control** | High - explicit flow | Low - agents self-organize |
| **Flexibility** | Low - predefined paths | High - dynamic collaboration |
| **Observability** | High - every step tracked | Moderate - agent outputs |
| **Use Case** | Deterministic processes | Creative collaboration |
| **Debugging** | Easy - step by step | Harder - emergent behavior |

**Choose Workflow when:**
- You need precise control over execution
- Debugging and observability are critical
- The process has well-defined steps
- Compliance/audit requirements exist

**Choose Team when:**
- The solution path is unclear
- You want agents to collaborate creatively
- The task benefits from multiple perspectives
- Flexibility is more important than control

## Design Patterns

### 1. Pipeline Pattern
```go
Steps: []workflow.Node{step1, step2, step3}
// Linear flow: A → B → C
```

### 2. Branch Pattern
```go
Condition → TrueNode
         → FalseNode
// If-else logic
```

### 3. Retry Pattern
```go
Loop with condition checking success
// Retry until success or max attempts
```

### 4. Fan-Out Pattern
```go
Parallel → Multiple agents simultaneously
// Distribute work, gather results
```

### 5. State Machine Pattern
```go
Router → Different states based on output
// Dynamic routing based on state
```

## Best Practices

### 1. Design for Observability

```go
// ✅ Good: Clear, descriptive IDs
workflow.StepConfig{
	ID:    "validate-input",
	Agent: validator,
}

// ❌ Bad: Vague IDs
workflow.StepConfig{
	ID:    "step1",
	Agent: validator,
}
```

### 2. Handle Errors Gracefully

```go
result, err := wf.Run(ctx, input)
if err != nil {
	log.Printf("Workflow failed: %v", err)
	// Fallback logic
}

if result.Error != nil {
	log.Printf("Step error: %v", result.Error)
}
```

### 3. Keep Steps Focused

```go
// ✅ Good: Single responsibility
extractStep  := "Extract data from input"
validateStep := "Validate extracted data"
saveStep     := "Save validated data"

// ❌ Bad: Too much in one step
megaStep := "Extract, validate, transform, and save data"
```

### 4. Optimize Parallel Execution

```go
// Use parallel for independent tasks
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
	Nodes: []workflow.Node{
		fetchUserData,
		fetchOrderData,
		fetchInventoryData,
	},
})
```

### 5. Use Context for Shared State

```go
// Steps can access previous outputs
analyzer := createAgent("analyzer", apiKey,
	"Analyze the research data provided in the previous step.")
```

## Advanced Features

### Custom Functions in Steps

```go
step, _ := workflow.NewStep(workflow.StepConfig{
	ID: "custom-processing",
	Function: func(ctx context.Context, input string) (string, error) {
		// Custom logic without an agent
		return processData(input), nil
	},
})
```

### Dynamic Loop Conditions

```go
loop, _ := workflow.NewLoop(workflow.LoopConfig{
	Body: improveStep,
	Condition: func(ctx *workflow.ExecutionContext, iteration int) bool {
		// Exit when quality threshold is met
		quality := assessQuality(ctx.Output)
		return quality < 0.9 && iteration < 10
	},
})
```

### Conditional Routing

```go
router, _ := workflow.NewRouter(workflow.RouterConfig{
	Router: func(ctx *workflow.ExecutionContext) string {
		// Route based on content
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

## Performance Tips

1. **Minimize Steps**: Each step adds latency
2. **Use Parallel**: Independent tasks should run concurrently
3. **Limit Loops**: Set reasonable max iterations
4. **Cache Results**: Store expensive computations in context
5. **Choose Fast Models**: Use gpt-4o-mini for speed-critical steps

## Next Steps

- Compare with [Team Collaboration](./team-demo.md) for different use cases
- Build [RAG Workflows](./rag-demo.md) with retrieval steps
- Combine workflows with different [Model Providers](./claude-agent.md)
- Create [Custom Tools](./simple-agent.md) for workflow steps

## Troubleshooting

**Workflow stuck in loop:**
- Check loop condition logic
- Add max iteration limit
- Log iteration count for debugging

**Parallel steps not running concurrently:**
- Verify they're in Parallel node, not sequential steps
- Check for shared resources/locks

**Context values not accessible:**
- Use correct key format: `{type}_{id}_{field}`
- Check if step ID matches exactly
- Verify step executed successfully

**Router always takes same path:**
- Log router function output
- Check condition logic
- Ensure routes are properly defined
