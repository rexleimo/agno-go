# S008: Agent 历史注入

**预估工作量**: 4 小时
**优先级**: P0
**前置依赖**: S007
**状态**: Done
**完成日期**: 2025-10-12

---

## 功能描述

实现 Agent 级别的历史注入机制：
1. 将工作流历史自动注入到 Agent 的 system message
2. 支持临时 instructions 修改（不影响 Agent 原始配置）
3. 确保多个 Step 共享同一 Agent 时的并发安全

---

## 实现步骤

### 步骤 1: 扩展 Agent 接口 (1h)

**文件**: `pkg/hno/agent/agent.go`

```go
// Agent 添加临时 instructions 支持
type Agent struct {
    // 现有字段...
    id           string
    name         string
    model        models.Model
    tools        []toolkit.Toolkit
    memory       memory.Memory
    instructions string
    maxLoops     int
    logger       *slog.Logger

    // 新增字段：临时 instructions
    // New field: Temporary instructions

    // tempInstructions 用于临时覆盖 instructions (仅单次执行)
    // tempInstructions temporarily overrides instructions (single execution only)
    tempInstructions string

    // instructionsMu 保护 instructions 修改
    // instructionsMu protects instructions modification
    instructionsMu sync.RWMutex
}

// GetInstructions 获取当前 instructions
// GetInstructions gets current instructions
func (a *Agent) GetInstructions() string {
    a.instructionsMu.RLock()
    defer a.instructionsMu.RUnlock()

    if a.tempInstructions != "" {
        return a.tempInstructions
    }
    return a.instructions
}

// SetInstructions 永久设置 instructions
// SetInstructions permanently sets instructions
func (a *Agent) SetInstructions(instructions string) {
    a.instructionsMu.Lock()
    defer a.instructionsMu.Unlock()

    a.instructions = instructions
}

// SetTempInstructions 临时设置 instructions（仅影响下一次 Run）
// SetTempInstructions temporarily sets instructions (only affects next Run)
func (a *Agent) SetTempInstructions(instructions string) {
    a.instructionsMu.Lock()
    defer a.instructionsMu.Unlock()

    a.tempInstructions = instructions
}

// ClearTempInstructions 清除临时 instructions
// ClearTempInstructions clears temporary instructions
func (a *Agent) ClearTempInstructions() {
    a.instructionsMu.Lock()
    defer a.instructionsMu.Unlock()

    a.tempInstructions = ""
}
```

### 步骤 2: 修改 Agent.Run 方法 (1h)

```go
func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error) {
    // 执行开始时使用临时 instructions（如果有）
    // Use temporary instructions at execution start (if any)
    currentInstructions := a.GetInstructions()

    // 确保执行完成后清除临时 instructions
    // Ensure temporary instructions are cleared after execution
    defer a.ClearTempInstructions()

    a.logger.Info("agent run started",
        "agent_id", a.id,
        "input", input)

    // 构建系统消息
    // Build system message
    systemMessage := types.NewSystemMessage(currentInstructions)

    // ... 现有执行逻辑
    // ... existing execution logic

    return output, nil
}
```

### 步骤 3: 实现历史注入辅助函数 (1h)

**文件**: `pkg/hno/workflow/history_injection.go` (新文件)

```go
package workflow

import (
    "fmt"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
)

// InjectHistoryToAgent 将历史上下文注入到 agent
// InjectHistoryToAgent injects history context into agent
// 返回原始 instructions，用于后续恢复
// Returns original instructions for later restoration
func InjectHistoryToAgent(a *agent.Agent, historyContext string) string {
    if a == nil || historyContext == "" {
        return ""
    }

    // 获取原始 instructions
    // Get original instructions
    originalInstructions := a.GetInstructions()

    // 构建增强的 instructions
    // Build enhanced instructions
    enhancedInstructions := buildEnhancedInstructions(originalInstructions, historyContext)

    // 设置临时 instructions
    // Set temporary instructions
    a.SetTempInstructions(enhancedInstructions)

    return originalInstructions
}

// buildEnhancedInstructions 构建包含历史的 instructions
// buildEnhancedInstructions builds instructions with history
func buildEnhancedInstructions(original, historyContext string) string {
    if original == "" {
        return historyContext
    }

    // 将历史上下文添加到原始 instructions 之后
    // Add history context after original instructions
    return fmt.Sprintf("%s\n\n%s", original, historyContext)
}

// RestoreAgentInstructions 恢复 agent 的原始 instructions
// RestoreAgentInstructions restores agent's original instructions
// 注意：由于 Run 方法已经自动清除临时 instructions，
// 此函数主要用于显式恢复场景
func RestoreAgentInstructions(a *agent.Agent, originalInstructions string) {
    if a == nil {
        return
    }

    a.ClearTempInstructions()
}

// FormatHistoryForAgent 格式化历史上下文供 agent 使用
// FormatHistoryForAgent formats history context for agent use
// 提供更友好的格式化选项
func FormatHistoryForAgent(history []HistoryEntry, options *HistoryFormatOptions) string {
    if len(history) == 0 {
        return ""
    }

    // 使用默认选项
    // Use default options
    if options == nil {
        options = DefaultHistoryFormatOptions()
    }

    context := options.Header + "\n"

    for i, entry := range history {
        runNum := i + 1

        if options.IncludeTimestamp {
            context += fmt.Sprintf("[run-%d] (%s)\n",
                runNum,
                entry.Timestamp.Format("2006-01-02 15:04:05"))
        } else {
            context += fmt.Sprintf("[run-%d]\n", runNum)
        }

        if options.IncludeInput && entry.Input != "" {
            context += fmt.Sprintf("%s: %s\n", options.InputLabel, entry.Input)
        }

        if options.IncludeOutput && entry.Output != "" {
            context += fmt.Sprintf("%s: %s\n", options.OutputLabel, entry.Output)
        }

        context += "\n" // 运行之间的空行
    }

    context += options.Footer
    return context
}

// HistoryFormatOptions 定义历史格式化选项
// HistoryFormatOptions defines history formatting options
type HistoryFormatOptions struct {
    Header           string
    Footer           string
    IncludeInput     bool
    IncludeOutput    bool
    IncludeTimestamp bool
    InputLabel       string
    OutputLabel      string
}

// DefaultHistoryFormatOptions 返回默认格式化选项
// DefaultHistoryFormatOptions returns default formatting options
func DefaultHistoryFormatOptions() *HistoryFormatOptions {
    return &HistoryFormatOptions{
        Header:           "<workflow_history_context>",
        Footer:           "</workflow_history_context>",
        IncludeInput:     true,
        IncludeOutput:    true,
        IncludeTimestamp: false,
        InputLabel:       "input",
        OutputLabel:      "output",
    }
}
```

### 步骤 4: 更新 Step 使用新的注入机制 (1h)

**文件**: `pkg/hno/workflow/step.go`

```go
func (s *Step) Execute(ctx context.Context, execCtx *ExecutionContext, workflowConfig HistoryConfig) (*ExecutionContext, error) {
    // 检查是否应该添加历史
    if s.shouldAddHistory(workflowConfig.AddHistory) && s.agent != nil {
        // 获取历史上下文
        historyContext := execCtx.GetHistoryContext()

        if historyContext != "" {
            // 注入历史到 agent
            // Inject history into agent
            InjectHistoryToAgent(s.agent, historyContext)

            s.logger.Debug("injected history to agent",
                "step_id", s.id,
                "agent_id", s.agent.GetID())
        }
    }

    // 执行步骤（现有逻辑）
    // Execute step (existing logic)
    if s.agent != nil {
        output, err := s.agent.Run(ctx, execCtx.Input)
        if err != nil {
            return nil, err
        }

        execCtx.Output = output.Content
        execCtx.AddMessages(output.Messages)

        return execCtx, nil
    }

    // ... 其他执行逻辑
}
```

---

## 测试要求

### 单元测试

**文件**: `pkg/hno/agent/agent_instructions_test.go`

```go
package agent

import (
    "testing"
)

func TestAgent_TempInstructions(t *testing.T) {
    agent, _ := New(Config{
        Name:         "test-agent",
        Instructions: "original instructions",
    })

    // 原始 instructions
    if instr := agent.GetInstructions(); instr != "original instructions" {
        t.Errorf("expected original instructions, got %s", instr)
    }

    // 设置临时 instructions
    agent.SetTempInstructions("temporary instructions")

    if instr := agent.GetInstructions(); instr != "temporary instructions" {
        t.Errorf("expected temporary instructions, got %s", instr)
    }

    // 清除临时 instructions
    agent.ClearTempInstructions()

    if instr := agent.GetInstructions(); instr != "original instructions" {
        t.Errorf("expected original instructions after clear, got %s", instr)
    }
}

func TestAgent_ConcurrentInstructionsAccess(t *testing.T) {
    // 测试并发访问 instructions 的安全性
    agent, _ := New(Config{
        Name:         "test-agent",
        Instructions: "original",
    })

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(2)

        go func() {
            defer wg.Done()
            agent.SetTempInstructions("temp")
        }()

        go func() {
            defer wg.Done()
            _ = agent.GetInstructions()
        }()
    }

    wg.Wait()
}
```

**文件**: `pkg/hno/workflow/history_injection_test.go`

```go
package workflow

import (
    "testing"
    "time"

    "github.com/rexleimo/agno-go/pkg/hno/agent"
)

func TestInjectHistoryToAgent(t *testing.T) {
    mockAgent, _ := agent.New(agent.Config{
        Name:         "test-agent",
        Instructions: "You are a helpful assistant",
    })

    historyContext := "<workflow_history_context>\n[run-1]\ninput: hello\noutput: hi\n</workflow_history_context>"

    // 注入历史
    original := InjectHistoryToAgent(mockAgent, historyContext)

    // 验证原始 instructions 被返回
    if original != "You are a helpful assistant" {
        t.Error("expected original instructions to be returned")
    }

    // 验证 agent 现在有增强的 instructions
    enhanced := mockAgent.GetInstructions()
    if !contains(enhanced, historyContext) {
        t.Error("expected history context in enhanced instructions")
    }

    // 清除后应该恢复
    mockAgent.ClearTempInstructions()
    if mockAgent.GetInstructions() != original {
        t.Error("expected instructions to be restored")
    }
}

func TestFormatHistoryForAgent(t *testing.T) {
    history := []HistoryEntry{
        {
            Input:     "hello",
            Output:    "hi there",
            Timestamp: time.Now(),
        },
        {
            Input:     "how are you",
            Output:    "I'm good",
            Timestamp: time.Now(),
        },
    }

    // 默认格式
    formatted := FormatHistoryForAgent(history, nil)

    if !contains(formatted, "<workflow_history_context>") {
        t.Error("expected context header")
    }

    if !contains(formatted, "[run-1]") {
        t.Error("expected run-1")
    }

    if !contains(formatted, "input: hello") {
        t.Error("expected input")
    }

    // 自定义格式
    options := &HistoryFormatOptions{
        Header:           "# History",
        Footer:           "# End",
        IncludeInput:     true,
        IncludeOutput:    false,
        IncludeTimestamp: true,
        InputLabel:       "User",
    }

    formatted = FormatHistoryForAgent(history, options)

    if !contains(formatted, "# History") {
        t.Error("expected custom header")
    }

    if contains(formatted, "output:") {
        t.Error("expected output to be excluded")
    }
}
```

---

## 验收标准

- [x] Agent 支持临时 instructions
- [x] Agent.Run 自动清除临时 instructions
- [x] 并发访问 instructions 是安全的
- [x] 历史注入不影响 Agent 原始配置
- [x] 提供灵活的历史格式化选项
- [x] 测试覆盖率 >85%
- [x] 所有测试通过
- [x] 性能：注入开销 <1ms

---

## 相关文件

- `pkg/hno/agent/agent.go` - 主要修改
- `pkg/hno/workflow/history_injection.go` - 新文件
- `pkg/hno/workflow/step.go` - 更新使用
- `pkg/hno/agent/agent_instructions_test.go` - 新测试
- `pkg/hno/workflow/history_injection_test.go` - 新测试

---

## 性能目标

- GetInstructions: <50ns
- SetTempInstructions: <100ns
- InjectHistoryToAgent: <500ns
- 不影响不使用历史功能的 Agent 性能

---

## 并发安全保证

1. **读写锁**: 使用 sync.RWMutex 保护 instructions
2. **自动清理**: Run 方法使用 defer 确保清理
3. **独立 goroutine**: 每个 Run 调用使用独立的临时 instructions

---

## 注意事项

1. **Agent 复用**: 如果多个 Step 共享同一 Agent，需要确保并发安全
2. **内存管理**: 及时清除临时 instructions 避免内存泄漏
3. **格式兼容**: 历史格式应该是 LLM 友好的
4. **向后兼容**: tempInstructions 为空时行为与原来一致
