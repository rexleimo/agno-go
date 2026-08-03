# Workflow 引擎示例

## 概述

本示例演示 HNO 强大的 Workflow 引擎,包含 5 个原语构建块:Step、Condition、Loop、Parallel 和 Router。Workflow 提供确定性、受控的执行流 - 非常适合需要精确控制和可观测性的复杂多步骤流程。

## 你将学到

- 如何使用 5 种原语类型构建 Workflow
- 顺序、条件和并行执行模式
- 如何创建具有自定义退出条件的循环
- 如何动态路由执行
- 如何将原语组合成复杂的 Workflow

## 前置要求

- Go 1.21 或更高版本
- OpenAI API key

## 设置

```bash
export OPENAI_API_KEY=sk-your-api-key-here
cd cmd/examples/workflow_demo
```

## Workflow 原语

### 1. Step - 基本执行单元

执行 Agent 或自定义函数。

```go
step, _ := workflow.NewStep(workflow.StepConfig{
	ID:    "research",
	Agent: researchAgent,
})
```

### 2. Condition - 分支逻辑

基于条件路由到不同路径。

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

### 3. Loop - 迭代执行

重复一个步骤直到满足条件。

```go
loop, _ := workflow.NewLoop(workflow.LoopConfig{
	ID:   "refinement",
	Body: refineStep,
	Condition: func(ctx *workflow.ExecutionContext, iteration int) bool {
		return iteration < 3  // Run 3 times
	},
})
```

### 4. Parallel - 并发执行

同时运行多个步骤。

```go
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
	ID:    "analysis",
	Nodes: []workflow.Node{techStep, bizStep, ethicsStep},
})
```

### 5. Router - 动态路由

基于运行时逻辑路由到不同步骤。

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

## 完整示例

### 演示 1: Sequential Workflow

基本管道: 研究 → 分析 → 写作

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

**流程:**
```
输入 → 研究者 → 分析者 → 写作者 → 输出
```

### 演示 2: Conditional Workflow

具有分支逻辑的情感分析。

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

**流程:**
```
输入 → 分类 → [积极?] → 积极处理器
               ↓ [消极]
               → 消极处理器
```

### 演示 3: Loop Workflow

迭代文本优化。

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

**流程:**
```
输入 → [优化] → [迭代 < 3?] → [是] → 再次优化
        ↑                      ↓ [否]
        └────────────────────── 输出
```

### 演示 4: Parallel Workflow

并发运行的多视角分析。

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

**流程:**
```
输入 → [技术 Agent]   → 合并
     → [商业 Agent]   → 结果
     → [伦理 Agent]   → 输出
     (全部同时运行)
```

### 演示 5: 带 Router 的复杂 Workflow

基于任务类型的动态路由。

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

**流程:**
```
输入 → Router → [类型?] → Calc Agent    (如果是计算)
                        → General Agent (如果是常规)
```

## 运行示例

```bash
go run main.go
```

## 预期输出

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

## 执行上下文

访问 Workflow 状态和结果:

```go
result, _ := wf.Run(ctx, input)

// 主输出
fmt.Println(result.Output)

// 从上下文获取特定值
value, exists := result.Get("step_id_output")

// 检查执行成功
if result.Error != nil {
	log.Printf("Workflow error: %v", result.Error)
}
```

### 上下文键

Workflow 使用可预测的键存储数据:

- **Step 输出**: `step_{step-id}_output`
- **Loop 迭代**: `loop_{loop-id}_iterations`
- **Parallel 分支**: `parallel_{parallel-id}_branch_{index}_output`
- **Condition 结果**: `condition_{condition-id}_result`

## Workflow vs Team

| 特性 | Workflow | Team |
|---------|----------|------|
| **控制** | 高 - 明确的流程 | 低 - Agent 自组织 |
| **灵活性** | 低 - 预定义路径 | 高 - 动态协作 |
| **可观测性** | 高 - 每步都被跟踪 | 中等 - Agent 输出 |
| **用例** | 确定性流程 | 创造性协作 |
| **调试** | 容易 - 逐步 | 较难 - 涌现行为 |

**选择 Workflow 当:**
- 需要对执行的精确控制
- 调试和可观测性至关重要
- 流程有明确定义的步骤
- 存在合规/审计要求

**选择 Team 当:**
- 解决路径不明确
- 希望 Agent 创造性协作
- 任务受益于多个视角
- 灵活性比控制更重要

## 设计模式

### 1. 管道模式
```go
Steps: []workflow.Node{step1, step2, step3}
// 线性流: A → B → C
```

### 2. 分支模式
```go
Condition → TrueNode
         → FalseNode
// If-else 逻辑
```

### 3. 重试模式
```go
Loop with condition checking success
// 重试直到成功或达到最大尝试次数
```

### 4. 扇出模式
```go
Parallel → Multiple agents simultaneously
// 分发工作,收集结果
```

### 5. 状态机模式
```go
Router → Different states based on output
// 基于状态的动态路由
```

## 最佳实践

### 1. 为可观测性设计

```go
// ✅ 好: 清晰、描述性的 ID
workflow.StepConfig{
	ID:    "validate-input",
	Agent: validator,
}

// ❌ 坏: 模糊的 ID
workflow.StepConfig{
	ID:    "step1",
	Agent: validator,
}
```

### 2. 优雅地处理错误

```go
result, err := wf.Run(ctx, input)
if err != nil {
	log.Printf("Workflow failed: %v", err)
	// 回退逻辑
}

if result.Error != nil {
	log.Printf("Step error: %v", result.Error)
}
```

### 3. 保持步骤专注

```go
// ✅ 好: 单一职责
extractStep  := "Extract data from input"
validateStep := "Validate extracted data"
saveStep     := "Save validated data"

// ❌ 坏: 一个步骤做太多事
megaStep := "Extract, validate, transform, and save data"
```

### 4. 优化并行执行

```go
// 为独立任务使用 parallel
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
	Nodes: []workflow.Node{
		fetchUserData,
		fetchOrderData,
		fetchInventoryData,
	},
})
```

### 5. 使用上下文共享状态

```go
// 步骤可以访问之前的输出
analyzer := createAgent("analyzer", apiKey,
	"Analyze the research data provided in the previous step.")
```

## 高级特性

### 步骤中的自定义函数

```go
step, _ := workflow.NewStep(workflow.StepConfig{
	ID: "custom-processing",
	Function: func(ctx context.Context, input string) (string, error) {
		// 不使用 Agent 的自定义逻辑
		return processData(input), nil
	},
})
```

### 动态循环条件

```go
loop, _ := workflow.NewLoop(workflow.LoopConfig{
	Body: improveStep,
	Condition: func(ctx *workflow.ExecutionContext, iteration int) bool {
		// 达到质量阈值时退出
		quality := assessQuality(ctx.Output)
		return quality < 0.9 && iteration < 10
	},
})
```

### 条件路由

```go
router, _ := workflow.NewRouter(workflow.RouterConfig{
	Router: func(ctx *workflow.ExecutionContext) string {
		// 基于内容路由
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

## 性能技巧

1. **最小化步骤**: 每个步骤都会增加延迟
2. **使用 Parallel**: 独立任务应并发运行
3. **限制循环**: 设置合理的最大迭代次数
4. **缓存结果**: 在上下文中存储昂贵的计算
5. **选择快速模型**: 对速度关键的步骤使用 gpt-4o-mini

## 下一步

- 与 [Team 协作](./team-demo.md) 比较不同用例
- 使用检索步骤构建 [RAG Workflow](./rag-demo.md)
- 将 Workflow 与不同的 [模型提供商](./claude-agent.md) 结合
- 为 Workflow 步骤创建 [自定义工具](./simple-agent.md)

## 故障排除

**Workflow 卡在循环中:**
- 检查循环条件逻辑
- 添加最大迭代限制
- 记录迭代计数以进行调试

**Parallel 步骤未并发运行:**
- 验证它们在 Parallel 节点中,而不是顺序步骤
- 检查共享资源/锁

**上下文值不可访问:**
- 使用正确的键格式: `{type}_{id}_{field}`
- 检查步骤 ID 是否完全匹配
- 验证步骤是否成功执行

**Router 总是走同一条路径:**
- 记录 Router 函数输出
- 检查条件逻辑
- 确保路由正确定义
