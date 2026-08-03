package runner

// State represents the current phase of the agentic loop.
// State 表示代理循环的当前阶段。
type State int

const (
	// StateAwaitModel: waiting for the next model response.
	// StateAwaitModel: 等待下一次模型响应。
	StateAwaitModel State = iota
	// StateToolCallsPending: model requested tool calls, executing them.
	// StateToolCallsPending: 模型请求了工具调用，正在执行。
	StateToolCallsPending
	// StateDone: loop finished (either a final response or a limit was hit).
	// StateDone: 循环结束（得到最终响应或达到上限）。
	StateDone
)

// String returns a human-readable state name.
// String 返回可读的状态名称。
func (s State) String() string {
	switch s {
	case StateAwaitModel:
		return "await_model"
	case StateToolCallsPending:
		return "tool_calls_pending"
	case StateDone:
		return "done"
	default:
		return "unknown"
	}
}
