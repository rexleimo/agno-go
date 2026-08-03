package workflow

import (
	"fmt"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
)

// InjectHistoryToAgent injects history context into agent's temporary instructions
// InjectHistoryToAgent 将历史上下文注入到 agent 的临时指令中
// Returns original instructions for later restoration (if needed)
// 返回原始指令用于后续恢复（如果需要）
func InjectHistoryToAgent(a *agent.Agent, historyContext string) string {
	if a == nil || historyContext == "" {
		return ""
	}

	// Get original instructions
	// 获取原始指令
	originalInstructions := a.GetInstructions()

	// Build enhanced instructions with history
	// 构建包含历史的增强指令
	enhancedInstructions := buildEnhancedInstructions(originalInstructions, historyContext)

	// Set temporary instructions (will be cleared after Run)
	// 设置临时指令（Run 之后会自动清除）
	a.SetTempInstructions(enhancedInstructions)

	return originalInstructions
}

// buildEnhancedInstructions combines original instructions with history context
// buildEnhancedInstructions 将原始指令与历史上下文结合
func buildEnhancedInstructions(original, historyContext string) string {
	if original == "" {
		return historyContext
	}

	// Add history context after original instructions
	// 将历史上下文添加到原始指令之后
	return fmt.Sprintf("%s\n\n%s", original, historyContext)
}

// RestoreAgentInstructions explicitly restores agent's original instructions
// RestoreAgentInstructions 显式恢复 agent 的原始指令
// Note: Agent.Run() already auto-clears temp instructions via defer
// 注意：Agent.Run() 已经通过 defer 自动清除临时指令
// This function is mainly for explicit restoration scenarios
// 此函数主要用于显式恢复场景
func RestoreAgentInstructions(a *agent.Agent) {
	if a == nil {
		return
	}

	a.ClearTempInstructions()
}

// FormatHistoryForAgent formats history context for agent use with flexible options
// FormatHistoryForAgent 使用灵活的选项格式化历史上下文供 agent 使用
func FormatHistoryForAgent(history []HistoryEntry, options *HistoryFormatOptions) string {
	if len(history) == 0 {
		return ""
	}

	// Use default options if not provided
	// 如果未提供则使用默认选项
	if options == nil {
		options = DefaultHistoryFormatOptions()
	}

	context := options.Header + "\n"

	for i, entry := range history {
		runNum := i + 1

		// Format run number with optional timestamp
		// 格式化运行编号，可选时间戳
		if options.IncludeTimestamp {
			context += fmt.Sprintf("[run-%d] (%s)\n",
				runNum,
				entry.Timestamp.Format("2006-01-02 15:04:05"))
		} else {
			context += fmt.Sprintf("[run-%d]\n", runNum)
		}

		// Add input if configured
		// 如果配置则添加输入
		if options.IncludeInput && entry.Input != "" {
			context += fmt.Sprintf("%s: %s\n", options.InputLabel, entry.Input)
		}

		// Add output if configured
		// 如果配置则添加输出
		if options.IncludeOutput && entry.Output != "" {
			context += fmt.Sprintf("%s: %s\n", options.OutputLabel, entry.Output)
		}

		context += "\n" // Empty line between runs / 运行之间的空行
	}

	context += options.Footer
	return context
}

// HistoryFormatOptions defines options for formatting history context
// HistoryFormatOptions 定义格式化历史上下文的选项
type HistoryFormatOptions struct {
	// Header is the opening tag for history context
	// Header 是历史上下文的开始标签
	Header string

	// Footer is the closing tag for history context
	// Footer 是历史上下文的结束标签
	Footer string

	// IncludeInput controls whether to include input in history
	// IncludeInput 控制是否在历史中包含输入
	IncludeInput bool

	// IncludeOutput controls whether to include output in history
	// IncludeOutput 控制是否在历史中包含输出
	IncludeOutput bool

	// IncludeTimestamp controls whether to include timestamps
	// IncludeTimestamp 控制是否包含时间戳
	IncludeTimestamp bool

	// InputLabel is the label for input field
	// InputLabel 是输入字段的标签
	InputLabel string

	// OutputLabel is the label for output field
	// OutputLabel 是输出字段的标签
	OutputLabel string
}

// DefaultHistoryFormatOptions returns default formatting options
// DefaultHistoryFormatOptions 返回默认格式化选项
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
