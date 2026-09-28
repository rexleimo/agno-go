package agent

import (
	"context"

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
	outcomes, _ := e.agent.runToolBatch(ctx, calls)
	return outcomes, nil
}
