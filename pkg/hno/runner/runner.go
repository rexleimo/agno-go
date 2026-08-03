package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

const (
	// DefaultMaxTurns is the default hard cap on model calls per run.
	// DefaultMaxTurns 是每次运行模型调用的默认硬上限。
	DefaultMaxTurns = 10
	// DefaultToolCallLimit is the default cap on tool executions per run (0 = unlimited).
	// DefaultToolCallLimit 是每次运行工具执行的默认上限（0 = 不限）。
	DefaultToolCallLimit = 0
)

// ToolCallOutcome is the result of executing a single model-requested tool call.
// ToolCallOutcome 是执行单个模型请求的工具调用的结果。
type ToolCallOutcome struct {
	// Call is the original tool call from the model.
	// Call 是模型发起的原始工具调用。
	Call types.ToolCall
	// Message is the tool result message to append to the conversation.
	// Message 是要追加到对话中的工具结果消息。
	Message *types.Message
	// StopLoop requests the runner to stop after this tool call.
	// StopLoop 请求 runner 在此工具调用后停止。
	StopLoop bool
	// HITLBlocked marks that the tool requires human approval.
	// HITLBlocked 标记该工具需要人工审批。
	HITLBlocked bool
}

// ToolExecutor executes tool calls and returns per-call outcomes.
// ToolExecutor 执行工具调用并返回每个调用的结果。
type ToolExecutor interface {
	// Execute runs all tool calls in order (implementations may parallelize).
	// Execute 按顺序运行所有工具调用（实现可以并行化）。
	Execute(ctx context.Context, calls []types.ToolCall) ([]ToolCallOutcome, error)
}

// ToolExecutorFunc adapts a function to the ToolExecutor interface.
// ToolExecutorFunc 将函数适配为 ToolExecutor 接口。
type ToolExecutorFunc func(ctx context.Context, calls []types.ToolCall) ([]ToolCallOutcome, error)

// Execute implements ToolExecutor.
func (f ToolExecutorFunc) Execute(ctx context.Context, calls []types.ToolCall) ([]ToolCallOutcome, error) {
	return f(ctx, calls)
}

// MessageBuilder builds the invoke request for a loop iteration.
// MessageBuilder 为每次循环迭代构建调用请求。
type MessageBuilder interface {
	// Build assembles messages and tool definitions into an InvokeRequest.
	// Build 将消息和工具定义组装为 InvokeRequest。
	Build(ctx context.Context, messages []*types.Message, tools []models.ToolDefinition) (*models.InvokeRequest, error)
}

// MessageBuilderFunc adapts a function to the MessageBuilder interface.
// MessageBuilderFunc 将函数适配为 MessageBuilder 接口。
type MessageBuilderFunc func(ctx context.Context, messages []*types.Message, tools []models.ToolDefinition) (*models.InvokeRequest, error)

// Build implements MessageBuilder.
func (f MessageBuilderFunc) Build(ctx context.Context, messages []*types.Message, tools []models.ToolDefinition) (*models.InvokeRequest, error) {
	return f(ctx, messages, tools)
}

// StepEvent is emitted after each loop iteration when OnStep is configured.
// StepEvent 在配置了 OnStep 时于每次循环迭代后发出。
type StepEvent struct {
	Turn     int                  // 1-based model call count / 从 1 开始的模型调用次数
	State    State                // loop phase / 循环阶段
	Response *types.ModelResponse // model response of this turn / 本轮模型响应
	Stop     StopReason           // non-empty when the loop will terminate / 循环将要终止时非空
}

// Config configures the Runner.
// Config 配置 Runner。
type Config struct {
	// Model is the LLM provider adapter (single-shot invoke only).
	// Model 是 LLM 提供商适配器（仅单次调用）。
	Model models.Model
	// Tools are the tool definitions sent to the model each turn.
	// Tools 是每轮发送给模型的工具定义。
	Tools []models.ToolDefinition
	// MaxTurns caps model calls per run (default DefaultMaxTurns).
	// MaxTurns 限制每次运行的模型调用次数（默认 DefaultMaxTurns）。
	MaxTurns int
	// ToolCallLimit caps tool executions per run (default DefaultToolCallLimit = unlimited).
	// ToolCallLimit 限制每次运行的工具执行次数（默认 DefaultToolCallLimit = 不限）。
	ToolCallLimit int
	// MessageBuilder assembles InvokeRequests (default: plain builder).
	// MessageBuilder 组装 InvokeRequest（默认：普通构建器）。
	MessageBuilder MessageBuilder
	// ToolExecutor executes tool calls (REQUIRED).
	// ToolExecutor 执行工具调用（必需）。
	ToolExecutor ToolExecutor
	// OnStep is invoked after every loop iteration (optional).
	// OnStep 在每次循环迭代后调用（可选）。
	OnStep func(StepEvent)
	// Logger for structured diagnostics (optional).
	// Logger 用于结构化诊断（可选）。
	Logger *slog.Logger
}

// Runner implements the agentic tool-call loop as an explicit state machine.
// The loop is the single source of truth shared by sync and streaming runs.
// Runner 以显式状态机实现代理工具调用循环。
// 该循环是同步与流式运行共享的唯一事实来源。
type Runner struct {
	model          models.Model
	tools          []models.ToolDefinition
	maxTurns       int
	toolCallLimit  int
	messageBuilder MessageBuilder
	toolExecutor   ToolExecutor
	onStep         func(StepEvent)
	logger         *slog.Logger
}

// New creates a Runner with defaults applied.
// New 创建 Runner 并应用默认值。
func New(cfg Config) (*Runner, error) {
	if cfg.Model == nil {
		return nil, errors.New("runner: model is required")
	}
	if cfg.ToolExecutor == nil {
		return nil, errors.New("runner: tool executor is required")
	}

	maxTurns := cfg.MaxTurns
	if maxTurns <= 0 {
		maxTurns = DefaultMaxTurns
	}
	toolCallLimit := cfg.ToolCallLimit
	if toolCallLimit < 0 {
		toolCallLimit = DefaultToolCallLimit
	}

	messageBuilder := cfg.MessageBuilder
	if messageBuilder == nil {
		messageBuilder = MessageBuilderFunc(defaultMessageBuilder)
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Runner{
		model:          cfg.Model,
		tools:          cfg.Tools,
		maxTurns:       maxTurns,
		toolCallLimit:  toolCallLimit,
		messageBuilder: messageBuilder,
		toolExecutor:   cfg.ToolExecutor,
		onStep:         cfg.OnStep,
		logger:         logger,
	}, nil
}

// Run executes the loop until a final answer, a limit, or cancellation.
// Run 执行循环，直到得到最终答案、达到上限或被取消。
func (r *Runner) Run(ctx context.Context, messages []*types.Message) (*types.ModelResponse, []*types.Message, StopReason, error) {
	state := StateAwaitModel
	var (
		response    *types.ModelResponse
		allMessages []*types.Message
		reason      StopReason
	)

	allMessages = cloneMessages(messages)

	turn := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, allMessages, StopCancelled, fmt.Errorf("runner: %w", err)
		}

		switch state {
		case StateAwaitModel:
			turn++ // count model calls, not loop iterations / 计数模型调用而非循环迭代
			req, err := r.messageBuilder.Build(ctx, allMessages, r.tools)
			if err != nil {
				return nil, allMessages, StopCancelled, fmt.Errorf("runner: build request: %w", err)
			}

			resp, err := r.model.Invoke(ctx, req)
			if err != nil {
				return nil, allMessages, StopCancelled, fmt.Errorf("runner: model invoke: %w", err)
			}
			response = resp

			// Record the assistant turn in the conversation.
			allMessages = append(allMessages, &types.Message{
				Role:             types.RoleAssistant,
				Content:          resp.Content,
				ToolCalls:        resp.ToolCalls,
				ReasoningContent: resp.ReasoningContent,
			})

			if r.onStep != nil {
				r.onStep(StepEvent{Turn: turn, State: StateAwaitModel, Response: resp})
			}

			if !resp.HasToolCalls() {
				reason = StopNoToolCalls
				state = StateDone
				break
			}
			if r.maxTurns > 0 && turn >= r.maxTurns {
				reason = StopLimitReached
				state = StateDone
				break
			}
			state = StateToolCallsPending

		case StateToolCallsPending:
			if response == nil || len(response.ToolCalls) == 0 {
				reason = StopNoToolCalls
				state = StateDone
				break
			}

			executed := countToolMessages(allMessages)

			// Determine how many calls we may run this round.
			// 确定本轮可以运行多少个调用。
			callsToRun := response.ToolCalls
			limitHit := false
			if r.toolCallLimit > 0 {
				remaining := r.toolCallLimit - executed
				if remaining <= 0 {
					reason = StopLimitReached
					state = StateDone
					break
				}
				if len(callsToRun) > remaining {
					callsToRun = callsToRun[:remaining]
					limitHit = true
				}
			}

			outcomes, err := r.toolExecutor.Execute(ctx, callsToRun)
			if err != nil {
				return nil, allMessages, StopCancelled, fmt.Errorf("runner: execute tools: %w", err)
			}

			stopLoop := false
			hitlBlocked := false
			for _, oc := range outcomes {
				if oc.Message != nil {
					allMessages = append(allMessages, oc.Message)
				}
				if oc.StopLoop {
					stopLoop = true
				}
				if oc.HITLBlocked {
					hitlBlocked = true
				}
			}

			// Report skipped calls when the limit cut the batch short.
			// 当上限截断了批次时，报告被跳过的调用。
			if limitHit {
				for _, c := range response.ToolCalls[len(callsToRun):] {
					allMessages = append(allMessages, &types.Message{
						Role:       types.RoleTool,
						ToolCallID: c.ID,
						Content:    "tool call limit reached; call not executed",
					})
				}
			}

			if r.onStep != nil {
				r.onStep(StepEvent{Turn: turn, State: StateToolCallsPending, Response: response})
			}

			switch {
			case hitlBlocked:
				reason = StopHITLBlocked
				state = StateDone
			case stopLoop:
				reason = StopAfterToolCall
				state = StateDone
			case limitHit:
				reason = StopLimitReached
				state = StateDone
			default:
				state = StateAwaitModel
			}

		case StateDone:
			return response, allMessages, reason, nil
		}
	}
}

// defaultMessageBuilder assembles a plain InvokeRequest from messages and tools.
// defaultMessageBuilder 从消息和工具组装普通 InvokeRequest。
func defaultMessageBuilder(_ context.Context, messages []*types.Message, tools []models.ToolDefinition) (*models.InvokeRequest, error) {
	return &models.InvokeRequest{
		Messages: messages,
		Tools:    tools,
	}, nil
}

// cloneMessages returns a shallow copy of the message slice.
// cloneMessages 返回消息切片的浅拷贝。
func cloneMessages(messages []*types.Message) []*types.Message {
	if len(messages) == 0 {
		return []*types.Message{}
	}
	out := make([]*types.Message, len(messages))
	copy(out, messages)
	return out
}

// countToolMessages counts tool-role messages in the slice.
// countToolMessages 统计切片中工具角色的消息数量。
func countToolMessages(messages []*types.Message) int {
	n := 0
	for _, m := range messages {
		if m != nil && m.Role == types.RoleTool {
			n++
		}
	}
	return n
}
