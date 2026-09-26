package agent

import (
	"context"
	"sync"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/runner"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// agentMessageBuilder 适配器，将 Agent 的 updateSystemMessage 逻辑暴露给 runner
// agentMessageBuilder adapter exposes Agent's updateSystemMessage logic to runner
type agentMessageBuilder struct {
	agent        *Agent
	instructions string
	tools        []models.ToolDefinition
}

func (b *agentMessageBuilder) Build(ctx context.Context, messages []*types.Message, tools []models.ToolDefinition) (*models.InvokeRequest, error) {
	// 如果当前指令与 Agent 初始指令不同，更新系统消息
	// Update system message if current instructions differ from agent's initial instructions
	finalMessages := messages
	if b.instructions != b.agent.Instructions && b.instructions != "" {
		finalMessages = b.agent.updateSystemMessage(messages, b.instructions)
	}

	req := &models.InvokeRequest{
		Messages: finalMessages,
		Tools:    tools,
	}
	attachRunContextToRequest(ctx, req)
	return req, nil
}

// agentToolExecutor 适配器，复用 Agent 既有的工具执行逻辑
// agentToolExecutor adapter reuses Agent's existing tool execution logic
type agentToolExecutor struct {
	agent *Agent
}

func (e *agentToolExecutor) Execute(ctx context.Context, calls []types.ToolCall) ([]runner.ToolCallOutcome, error) {
	if len(calls) == 0 {
		return nil, nil
	}

	index := e.agent.buildFunctionIndex()

	// 并发执行所有工具调用，保持结果顺序
	// Execute all tool calls concurrently, preserve result order
	results := make([]toolResult, len(calls))
	var wg sync.WaitGroup

	for i, tc := range calls {
		wg.Add(1)
		go func(idx int, call types.ToolCall) {
			defer wg.Done()
			results[idx] = e.agent.executeOneTool(ctx, index, call)
		}(i, tc)
	}
	wg.Wait()

	// 转换为 runner.ToolCallOutcome 并同步写入 Memory
	// Convert to runner.ToolCallOutcome and sync write to Memory
	outcomes := make([]runner.ToolCallOutcome, len(results))
	for i, res := range results {
		msg := types.NewToolMessage(res.callID, res.message)
		e.agent.Memory.Add(msg, e.agent.UserID)

		outcomes[i] = runner.ToolCallOutcome{
			Call:     calls[i],
			Message:  msg,
			StopLoop: res.stopLoop,
		}
	}

	return outcomes, nil
}
