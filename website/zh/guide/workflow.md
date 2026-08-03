# Workflow - 基于步骤的编排

使用 5 个强大原语构建复杂、可控的多步骤流程。

---

## 什么是 Workflow?

**Workflow** 提供确定性的、基于步骤的编排,用于构建可控的 AI Agent 流程。与 Team(自主式)不同,Workflow 让您完全控制执行流程。

### 核心特性

- **5 种原语**: Step、Condition、Loop、Parallel、Router
- **确定性执行**: 可预测、可重复的流程
- **完全控制**: 您决定执行路径
- **上下文共享**: 在步骤间传递数据
- **可组合**: 嵌套原语以实现复杂逻辑

---

## 创建 Workflow

### 基础示例

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

    // Create agents
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

    // Create workflow steps
    step1, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "fetch",
        Agent: fetchAgent,
    })

    step2, _ := workflow.NewStep(workflow.StepConfig{
        ID:    "process",
        Agent: processAgent,
    })

    // Create workflow
    wf, _ := workflow.New(workflow.Config{
        Name:  "Data Pipeline",
        Steps: []workflow.Node{step1, step2},
    })

    // Run workflow
    result, _ := wf.Run(context.Background(), "AI trends")
    fmt.Println(result.Output)
}
```

---

## Workflow 原语

### 1. Step

执行 Agent 或自定义函数。

```go
// With agent
step, _ := workflow.NewStep(workflow.StepConfig{
    ID:    "analyze",
    Agent: analyzerAgent,
})

// With custom function
step, _ := workflow.NewStep(workflow.StepConfig{
    ID: "transform",
    Function: func(ctx *workflow.ExecutionContext) (*workflow.StepOutput, error) {
        input := ctx.Input
        // Custom logic
        return &workflow.StepOutput{Output: transformed}, nil
    },
})
```

**使用场景:**
- Agent 执行
- 自定义数据转换
- API 调用
- 数据验证

---

### 2. Condition

基于上下文的条件分支。

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

**使用场景:**
- 情感路由
- 质量检查
- 错误处理
- A/B 测试逻辑

---

### 3. Loop

带退出条件的迭代执行。

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

**使用场景:**
- 重试逻辑
- 迭代改进
- 数据处理循环
- 渐进式改进

---

### 4. Parallel

并发执行多个步骤。

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

**使用场景:**
- 并行数据收集
- 多源聚合
- 独立计算
- 性能优化

---

### 5. Router

动态路由到不同分支。

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

**使用场景:**
- 输入类型路由
- 语言检测
- 优先级处理
- 动态分发

---

## 执行上下文

ExecutionContext 提供对 Workflow 状态的访问。

### 方法

```go
type ExecutionContext struct {
    Input string  // Workflow 输入
}

// Get output from a previous step
func (ctx *ExecutionContext) GetStepOutput(stepID string) *StepOutput

// Store custom data
func (ctx *ExecutionContext) SetData(key string, value interface{})

// Retrieve custom data
func (ctx *ExecutionContext) GetData(key string) interface{}
```

### 示例

```go
step := workflow.NewStep(workflow.StepConfig{
    ID: "use_context",
    Function: func(ctx *workflow.ExecutionContext) (*workflow.StepOutput, error) {
        // Access previous step
        previous := ctx.GetStepOutput("classify")

        // Access shared data
        userData := ctx.GetData("user_info")

        // Process and return
        result := processData(previous.Output, userData)
        return &workflow.StepOutput{Output: result}, nil
    },
})
```

---

## 完整示例

### 条件 Workflow

基于情感的路由。

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

### 循环 Workflow

带质量改进的重试。

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

## 最佳实践

### 1. 清晰的 Step ID

使用描述性的 Step ID 以便调试:

```go
// Good ✅
ID: "fetch_user_data"

// Bad ❌
ID: "step1"
```

### 2. 错误处理

在每个步骤处理错误:

```go
result, err := wf.Run(ctx, input)
if err != nil {
    log.Printf("Workflow failed at step: %v", err)
    return
}
```

### 3. Context 超时

设置合理的超时:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

result, _ := wf.Run(ctx, input)
```

### 4. 循环限制

始终设置 MaxIterations 以防止无限循环:

```go
loop, _ := workflow.NewLoop(workflow.LoopConfig{
    // ...
    MaxIterations: 10,  // Always set a limit
})
```


## 高级 Workflow 功能

### 从检查点恢复

在上次成功步骤处恢复被中断的工作流:

```go
execCtx, err := wf.Run(ctx, input, "session-id",
    workflow.WithResumeFrom("validate-output"),
)
if err != nil {
    log.Fatal(err)
}
```

当配置项 `EnableHistory` 启用时, 每次运行都会持久化步骤输出与取消记录。传入 `WithResumeFrom(stepID)` 会跳过已完成的步骤, 从目标检查点继续执行。

### 取消持久化

启用历史记录以捕获取消原因、运行 ID 与时间戳:

```go
store := workflow.NewMemoryStorage(100)
wf, _ := workflow.New(workflow.Config{
    Name:          "retriable-pipeline",
    Steps:         []workflow.Node{firstStep, finalStep},
    EnableHistory: true,
    HistoryStore:  store,
})
```

若某个步骤取消了上下文, 最新的历史记录会保存 `RunStatusCancelled` 状态以及 `CancellationReason`。可通过 `store.GetSession(ctx, sessionID)` 检查细节。

### 媒体负载

在不影响提示的情况下附加图片、音频或文件:

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

工作流会将校验后的附件存入会话状态 (`media_payload`), 供后续步骤与 AgentOS 集成访问。

---

## Workflow vs Team

何时使用每种:

### 使用 Workflow 当:
- 需要确定性执行
- 步骤必须按特定顺序发生
- 需要细粒度控制
- 调试和测试至关重要

### 使用 Team 当:
- Agent 应该自主工作
- 顺序不重要 (parallel/consensus)
- 想要涌现行为
- 灵活性优于控制

---

## 性能提示

### 并行执行

对独立步骤使用 Parallel:

```go
// Sequential: 3 seconds total
steps := []workflow.Node{step1, step2, step3} // 1s + 1s + 1s

// Parallel: 1 second total
parallel, _ := workflow.NewParallel(workflow.ParallelConfig{
    Steps: []workflow.Node{step1, step2, step3}, // max(1s, 1s, 1s)
})
```

### Context 重用

通过重用上下文数据避免冗余的 LLM 调用:

```go
func (ctx *ExecutionContext) {
    // Cache expensive computation
    if ctx.GetData("cached") == nil {
        expensive := computeExpensiveData()
        ctx.SetData("cached", expensive)
    }
}
```

---

## 下一步

- 与 [Team](/guide/team) 比较自主 Agent
- 向 Workflow Agent 添加 [Tools](/guide/tools)
- 探索 [Models](/guide/models) 的不同 LLM 提供商
- 查看 [Workflow API Reference](/api/workflow) 获取详细文档

---

## 相关示例

- [Workflow Demo](/examples/workflow-demo) - 完整示例
- [Conditional Routing](/examples/workflow-demo#conditional)
- [Retry Loop Pattern](/examples/workflow-demo#loop)
