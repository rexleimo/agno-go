package agent

import (
	"context"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/runner"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// kernelState receives the loop counters the agent needs back from the kernel.
// kernelState 接回 agent 需要的内核循环计数。
type kernelState struct {
	turn int
}

// newKernel builds the single runner loop shared by Agent.Run and
// Agent.RunStream. Loop policy is wired here and nowhere else, so the
// synchronous and streaming paths cannot drift apart.
//
// invoker decides how one model turn is called; nil uses the kernel default of
// a synchronous Model.Invoke.
//
// newKernel 构造 Agent.Run 与 Agent.RunStream 共用的唯一循环。循环策略只在此处
// 接线，使同步与流式两条路径无法各自漂移。
// invoker 决定单次回合怎么调用；nil 时内核默认走 Model.Invoke。
func (a *Agent) newKernel(
	ctx context.Context,
	tools []models.ToolDefinition,
	instructions string,
	state *kernelState,
	invoker runner.TurnInvoker,
	onAssistantTurn func(*types.ModelResponse),
) (*runner.Runner, error) {
	return runner.New(runner.Config{
		Model:         a.Model,
		Invoker:       invoker,
		Tools:         tools,
		MaxTurns:      a.MaxLoops,
		ToolCallLimit: a.ToolCallLimit,
		MessageBuilder: &agentMessageBuilder{
			agent:        a,
			instructions: instructions,
			tools:        tools,
		},
		ToolExecutor: &agentToolExecutor{agent: a},
		OnStep: func(evt runner.StepEvent) {
			if evt.State != runner.StateAwaitModel {
				return
			}
			state.turn = evt.Turn
			a.Memory.Add(a.assistantMessage(ctx, evt.Response), a.UserID)
			if onAssistantTurn != nil {
				onAssistantTurn(evt.Response)
			}
		},
		OnSkippedToolCalls: func(skipped []types.ToolCall) {
			for _, call := range skipped {
				a.Memory.Add(runner.NewToolCallLimitMessage(call), a.UserID)
			}
		},
		Logger: a.logger,
	})
}

// assistantMessage records one model turn in memory, including reasoning.
// assistantMessage 把一次模型回合（含推理内容）写入记忆。
func (a *Agent) assistantMessage(ctx context.Context, resp *types.ModelResponse) *types.Message {
	return &types.Message{
		Role:             types.RoleAssistant,
		Content:          resp.Content,
		ToolCalls:        resp.ToolCalls,
		ReasoningContent: a.extractReasoning(ctx, resp),
	}
}

// maxLoopsExceeded distinguishes a MaxLoops stop from a tool-call-limit stop:
// the kernel reports both as runner.StopLimitReached.
// maxLoopsExceeded 区分 MaxLoops 耗尽与工具调用上限：内核把两者都报为
// runner.StopLimitReached。
func maxLoopsExceeded(reason runner.StopReason, turn, maxLoops int) bool {
	return reason == runner.StopLimitReached && turn >= maxLoops
}
