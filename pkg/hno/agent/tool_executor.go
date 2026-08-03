package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/rexleimo/agno-go/pkg/hno/observability"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// buildFunctionIndex merges all toolkit functions into a single map for
// O(1) dispatch. Built once per agent.
// buildFunctionIndex 将所有 toolkit 函数合并为单个 map 用于 O(1) 分发。
// 每个 agent 只构建一次。
func (a *Agent) buildFunctionIndex() map[string]*toolkit.Function {
	if a.functionIndex != nil {
		return a.functionIndex
	}
	index := make(map[string]*toolkit.Function)
	for _, tk := range a.Toolkits {
		for name, fn := range tk.Functions() {
			index[name] = fn
		}
	}
	a.functionIndex = index
	return index
}

// executeToolCalls executes all tool calls (concurrently) and adds results
// to memory. Each call is wrapped in an execute_tool span.
//
// executeToolCalls 并发执行所有工具调用并将结果写入记忆。
// 每次调用都包在 execute_tool span 中。
func (a *Agent) executeToolCalls(ctx context.Context, toolCalls []types.ToolCall) error {
	if len(toolCalls) == 0 {
		return nil
	}

	index := a.buildFunctionIndex()

	// Execute all tool calls concurrently; results are ordered so memory
	// receives them in call order (models expect deterministic tool messages).
	// 并发执行所有工具调用；结果按调用顺序排列（模型期望确定性的工具消息）。
	results := make([]toolResult, len(toolCalls))
	var wg sync.WaitGroup
	for i, tc := range toolCalls {
		wg.Add(1)
		go func(idx int, call types.ToolCall) {
			defer wg.Done()
			results[idx] = a.executeOneTool(ctx, index, call)
		}(i, tc)
	}
	wg.Wait()

	// Append tool messages in deterministic order.
	// 按确定性顺序追加工具消息。
	for _, res := range results {
		a.Memory.Add(types.NewToolMessage(res.callID, res.message), a.UserID)
	}
	return nil
}

// toolResult carries the outcome of a single tool call.
// toolResult 携带单个工具调用的结果。
type toolResult struct {
	callID  string
	message string
}

// executeOneTool dispatches a single tool call with validation and tracing.
// executeOneTool 分发单个工具调用（含校验与追踪）。
func (a *Agent) executeOneTool(ctx context.Context, index map[string]*toolkit.Function, tc types.ToolCall) toolResult {
	ctx, span := observability.StartToolSpan(ctx, tc.Function.Name)
	defer span.End()

	fn := index[tc.Function.Name]
	if fn == nil {
		a.logger.Warn("tool not found", "function", tc.Function.Name)
		return toolResult{callID: tc.ID, message: fmt.Sprintf("function %s not found in any toolkit", tc.Function.Name)}
	}

	args, err := toolkit.ParseArguments(tc.Function.Arguments)
	if err != nil {
		a.logger.Error("argument parsing failed", "function", tc.Function.Name, "error", err)
		return toolResult{callID: tc.ID, message: fmt.Sprintf("failed to parse arguments: %v", err)}
	}

	if err := toolkit.ValidateArgs(fn, args); err != nil {
		a.logger.Error("argument validation failed", "function", tc.Function.Name, "error", err)
		return toolResult{callID: tc.ID, message: err.Error()}
	}

	a.logger.Info("executing tool", "function", tc.Function.Name, "args", args)
	result, err := fn.Handler(ctx, args)
	if err != nil {
		a.logger.Error("tool execution failed", "function", tc.Function.Name, "error", err)
		return toolResult{callID: tc.ID, message: fmt.Sprintf("tool execution error: %v", err)}
	}

	resultStr, err := toolkit.FormatResult(result)
	if err != nil {
		resultStr = fmt.Sprintf("%v", result)
	}
	a.logger.Info("tool executed successfully", "function", tc.Function.Name)
	return toolResult{callID: tc.ID, message: resultStr}
}
