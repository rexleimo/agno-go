package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/rexleimo/agno-go/pkg/hno/observability"
	"github.com/rexleimo/agno-go/pkg/hno/runner"
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
	a.runToolBatch(ctx, toolCalls)
	return nil
}

// runToolBatch is the single place a batch of tool calls is executed: the calls run
// concurrently, the results are merged back in call order, each tool message is
// written to memory in that same order, and the per-call outcome carries the error
// the handler returned (if any). Both the synchronous kernel adapter and the
// StreamTasks producer run through here, so the two paths cannot drift apart in
// ordering or in success/failure classification.
// runToolBatch 是工具批次的唯一执行处：调用并发执行、结果按调用声明序汇回、工具消息
// 按同一顺序写入 Memory，且逐调用的成败带着 handler 交回的原始 error。同步内核适配器
// 与 StreamTasks 生产者都走这里，两条路径无法在次序或成败分类上各自漂移。
func (a *Agent) runToolBatch(ctx context.Context, calls []types.ToolCall) ([]runner.ToolCallOutcome, []toolResult) {
	if len(calls) == 0 {
		return nil, nil
	}

	index := a.buildFunctionIndex()

	// Execute all tool calls concurrently; results are ordered so memory
	// receives them in call order (models expect deterministic tool messages).
	// 并发执行所有工具调用；结果按调用顺序排列（模型期望确定性的工具消息）。
	results := make([]toolResult, len(calls))
	var wg sync.WaitGroup
	for i, tc := range calls {
		wg.Add(1)
		go func(idx int, call types.ToolCall) {
			defer wg.Done()
			results[idx] = a.executeOneTool(ctx, index, call)
		}(i, tc)
	}
	wg.Wait()

	// Append tool messages in deterministic order and pair each with its outcome.
	// 按确定性顺序追加工具消息，并逐个配上其判定。
	outcomes := make([]runner.ToolCallOutcome, len(results))
	for i, res := range results {
		msg := types.NewToolMessage(res.callID, res.message)
		a.Memory.Add(msg, a.UserID)
		outcomes[i] = runner.ToolCallOutcome{
			Call:     calls[i],
			Message:  msg,
			StopLoop: res.stopLoop,
		}
	}
	return outcomes, results
}

// toolResult carries the outcome of a single tool call. err is the error the call
// ran into, before it was folded into the message text: classifying a task as failed
// reads err, never the text.
// toolResult 携带单个工具调用的结果。err 是该调用撞上的错误（在被折进消息文本之前）：
// 判定任务成败只看 err，不看文本。
type toolResult struct {
	callID   string
	node     string // 被调用的工具函数名 / name of the invoked tool function
	message  string
	err      error
	stopLoop bool // 工具是否请求终止循环 / whether the tool requests loop termination
}

// executeOneTool dispatches a single tool call with validation and tracing.
// executeOneTool 分发单个工具调用（含校验与追踪）。
func (a *Agent) executeOneTool(ctx context.Context, index map[string]*toolkit.Function, tc types.ToolCall) toolResult {
	ctx, span := observability.StartToolSpan(ctx, tc.Function.Name)
	defer span.End()

	fn := index[tc.Function.Name]
	if fn == nil {
		err := fmt.Errorf("function %s not found in any toolkit", tc.Function.Name)
		a.logger.Warn("tool not found", "function", tc.Function.Name)
		return toolResult{callID: tc.ID, node: tc.Function.Name, message: err.Error(), err: err}
	}

	args, err := toolkit.ParseArguments(tc.Function.Arguments)
	if err != nil {
		a.logger.Error("argument parsing failed", "function", tc.Function.Name, "error", err)
		// The message text keeps the pre-slice-31 wording (ParseArguments already wraps
		// with this prefix, so the tool message carries it twice); this slice adds the
		// error to the result without rewriting what memory has always stored.
		// 消息文本保留切片 31 之前的措辞（ParseArguments 自带该前缀，工具消息里出现两次）；
		// 本片只把错误挂上结果，不改记忆早已存下的字。
		return toolResult{
			callID:  tc.ID,
			node:    tc.Function.Name,
			message: fmt.Sprintf("failed to parse arguments: %v", err),
			err:     err,
		}
	}

	if err := toolkit.ValidateArgs(fn, args); err != nil {
		a.logger.Error("argument validation failed", "function", tc.Function.Name, "error", err)
		return toolResult{callID: tc.ID, node: tc.Function.Name, message: err.Error(), err: err}
	}

	a.logger.Info("executing tool", "function", tc.Function.Name, "args", args)
	result, err := fn.Handler(ctx, args)
	if err != nil {
		a.logger.Error("tool execution failed", "function", tc.Function.Name, "error", err)
		return toolResult{
			callID:  tc.ID,
			node:    tc.Function.Name,
			message: fmt.Sprintf("tool execution error: %v", err),
			err:     err,
		}
	}

	resultStr, err := toolkit.FormatResult(result)
	if err != nil {
		resultStr = fmt.Sprintf("%v", result)
	}
	a.logger.Info("tool executed successfully", "function", tc.Function.Name)
	return toolResult{callID: tc.ID, node: tc.Function.Name, message: resultStr, stopLoop: fn.StopLoop}
}
