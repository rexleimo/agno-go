package runner

import "github.com/rexleimo/agno-go/pkg/hno/types"

const toolCallLimitMessage = "tool call limit reached; call not executed"

// decideToolBatch splits one round's tool calls into those that may run and
// those the per-run limit cuts off.
//
// Contract the kernel relies on:
//   - limit <= 0: every call runs, nothing is skipped, limitHit is false.
//   - limit exhausted before this round: nothing runs and EVERY call is
//     returned as skipped, so the caller still pairs each call with a tool
//     message instead of leaving an unanswered tool call in history.
//   - partial room: the first `remaining` calls run, the rest are skipped.
//
// decideToolBatch 把一轮的工具调用拆成「可执行」与「被本次运行上限截断」两部分。
//
// 内核依赖的契约：
//   - limit <= 0：全部执行，不跳过，limitHit 为 false。
//   - 本轮开始前上限已用尽：不执行任何调用，且把全部调用作为 skipped 返回，
//     让调用方仍能为每个调用配对 tool 消息，而不是留下无人应答的 tool call。
//   - 剩余名额不足：前 remaining 个执行，其余跳过。
func decideToolBatch(executed, limit int, calls []types.ToolCall) (toRun, skipped []types.ToolCall, limitHit bool) {
	if limit <= 0 {
		return calls, nil, false
	}

	remaining := limit - executed
	if remaining <= 0 {
		return nil, calls, true
	}
	if len(calls) > remaining {
		return calls[:remaining], calls[remaining:], true
	}
	return calls, nil, false
}

// NewToolCallLimitMessage builds the paired tool response for a skipped call.
// NewToolCallLimitMessage 为被跳过的调用构造配对的 tool 响应。
func NewToolCallLimitMessage(call types.ToolCall) *types.Message {
	return types.NewToolMessage(call.ID, toolCallLimitMessage)
}
