package agent

import (
	"context"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/observability"
	"github.com/rexleimo/agno-go/pkg/hno/runner"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// runKernel drives the shared runner loop under the run-level invoke_agent span.
// The span opens here and ends here, in one function, which is why Agent.Run and the
// streaming goroutine cannot drift apart: each has exactly one drive site and both
// call this. The ctx handed to the kernel carries the span, so the per-attempt chat
// spans (runner) and the execute_tool spans (this package) become its children; that
// parenting is the whole point of the seam.
// runKernel 在运行级 invoke_agent span 之下驱动共享的 runner 循环。span 只在这一个
// 函数里开启、也只在这里结束，因此同步入口与流式协程无法各自漂移：两条路径各只有一个
// 驱动点，且都调用本函数。交给内核的 ctx 携带该 span，于是 runner 的逐尝试 chat span
// 与本包的 execute_tool span 成为它的子 span —— 这个父子关系正是这条缝的全部意义。
func (a *Agent) runKernel(
	ctx context.Context,
	runID string,
	r *runner.Runner,
	messages []*types.Message,
) (*types.ModelResponse, []*types.Message, runner.StopReason, error) {
	spanCtx, span := observability.StartAgentSpan(ctx, a.Name, runID)
	defer span.End()

	return r.Run(spanCtx, messages)
}

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
// toolExecutor decides who executes a tool batch; nil uses the plain agent executor.
// The StreamTasks producer passes a wrapper that emits task events around that same
// batch, which is why the injection point lives here rather than in the caller.
//
// newKernel 构造 Agent.Run 与 Agent.RunStream 共用的唯一循环。循环策略只在此处
// 接线，使同步与流式两条路径无法各自漂移。
// invoker 决定单次回合怎么调用；nil 时内核默认走 Model.Invoke。
// toolExecutor 决定谁执行工具批次；nil 时走既有的 agent 执行器。StreamTasks 生产者传入
// 一个在同一批次外围发射任务事件的包装件，所以注入口住在这里而不是调用方。
func (a *Agent) newKernel(
	ctx context.Context,
	tools []models.ToolDefinition,
	instructions string,
	state *kernelState,
	invoker runner.TurnInvoker,
	onAssistantTurn func(*types.ModelResponse),
	toolExecutor runner.ToolExecutor,
) (*runner.Runner, error) {
	if toolExecutor == nil {
		toolExecutor = &agentToolExecutor{agent: a}
	}

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
		ToolExecutor: toolExecutor,
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
