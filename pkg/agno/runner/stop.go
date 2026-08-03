package runner

// StopReason explains why the agentic loop terminated.
// StopReason 解释代理循环终止的原因。
type StopReason string

const (
	// StopNoToolCalls: the model produced a final answer without tool calls.
	// StopNoToolCalls: 模型生成了无工具调用的最终回答。
	StopNoToolCalls StopReason = "no_tool_calls"
	// StopLimitReached: max_turns or tool_call_limit was hit.
	// StopLimitReached: 达到 max_turns 或 tool_call_limit 上限。
	StopLimitReached StopReason = "limit_reached"
	// StopAfterToolCall: a tool requested the loop to stop (stop_after_tool_call).
	// StopAfterToolCall: 工具请求循环停止（stop_after_tool_call）。
	StopAfterToolCall StopReason = "stop_after_tool_call"
	// StopHITLBlocked: a tool requires human approval and no approval was provided.
	// StopHITLBlocked: 工具需要人工审批且未获批准。
	StopHITLBlocked StopReason = "hitl_blocked"
	// StopRequirements: pending requirements (e.g. member-agent bubbles) blocked progress.
	// StopRequirements: 未满足的需求阻塞了进度。
	StopRequirements StopReason = "requirements_pending"
	// StopCancelled: context was cancelled during the loop.
	// StopCancelled: 循环期间上下文被取消。
	StopCancelled StopReason = "cancelled"
)

// String returns a human-readable stop reason.
// String 返回可读的停止原因。
func (r StopReason) String() string {
	return string(r)
}
