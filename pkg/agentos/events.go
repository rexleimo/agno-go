package agentos

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/media"
)

// EventType 表示事件类型
// EventType represents the type of event
type EventType string

const (
	// EventRunStart Agent 运行开始
	// EventRunStart indicates agent run has started
	EventRunStart EventType = "run_start"

	// EventReasoning 推理事件
	// EventReasoning indicates reasoning content has been produced
	EventReasoning EventType = "reasoning"

	// EventToolCall 工具调用事件
	// EventToolCall indicates a tool call event
	EventToolCall EventType = "tool_call"

	// EventToken 令牌生成事件（流式输出）
	// EventToken indicates a token generation event (streaming)
	EventToken EventType = "token"

	// EventStepStart 步骤开始
	// EventStepStart indicates a step has started
	EventStepStart EventType = "step_start"

	// EventStepEnd 步骤结束
	// EventStepEnd indicates a step has ended
	EventStepEnd EventType = "step_end"

	// EventError 错误事件
	// EventError indicates an error occurred
	EventError EventType = "error"

	// EventComplete 运行完成
	// EventComplete indicates run has completed
	EventComplete EventType = "complete"
)

// Event 表示一个事件
// Event represents an event
type Event struct {
	// Type 事件类型
	// Type is the event type
	Type EventType `json:"type"`

	// Timestamp 事件时间戳
	// Timestamp is the event timestamp
	Timestamp time.Time `json:"timestamp"`

	// Data 事件数据
	// Data is the event data
	Data interface{} `json:"data,omitempty"`

	// SessionID 会话 ID（可选）
	// SessionID is the session ID (optional)
	SessionID string `json:"session_id,omitempty"`

    // AgentID Agent ID（可选）
    // AgentID is the agent ID (optional)
    AgentID string `json:"agent_id,omitempty"`

    // RunContextID 运行上下文 ID（可选，用于事件关联）
    // RunContextID is the run context identifier for correlating events (optional)
    RunContextID string `json:"run_context_id,omitempty"`
}

// RunStartData 运行开始事件数据
// RunStartData is the data for run start event
type RunStartData struct {
	// Input 输入内容
	// Input is the input content
	Input string `json:"input"`

	// SessionID 会话 ID
	// SessionID is the session ID
	SessionID string `json:"session_id,omitempty"`

	// Media 附带的媒体资源
	// Media contains normalized media attachments
	Media []media.Attachment `json:"media,omitempty"`
}

// ToolCallData 工具调用事件数据
// ToolCallData is the data for tool call event
type ToolCallData struct {
	// ToolName 工具名称
	// ToolName is the tool name
	ToolName string `json:"tool_name"`

	// Arguments 工具参数
	// Arguments are the tool arguments
	Arguments map[string]interface{} `json:"arguments"`

	// Result 工具结果（可选）
	// Result is the tool result (optional)
	Result interface{} `json:"result,omitempty"`
}

// TokenData 令牌事件数据
// TokenData is the data for token event
type TokenData struct {
	// Token 生成的令牌
	// Token is the generated token
	Token string `json:"token"`

	// Index 令牌索引
	// Index is the token index
	Index int `json:"index"`
}

// StepData 步骤事件数据
// StepData is the data for step events
type StepData struct {
	// StepName 步骤名称
	// StepName is the step name
	StepName string `json:"step_name"`

	// StepIndex 步骤索引
	// StepIndex is the step index
	StepIndex int `json:"step_index"`

	// Description 步骤描述
	// Description is the step description
	Description string `json:"description,omitempty"`
}

// ErrorData 错误事件数据
// ErrorData is the data for error event
type ErrorData struct {
	// Error 错误消息
	// Error is the error message
	Error string `json:"error"`

	// Code 错误代码
	// Code is the error code
	Code string `json:"code,omitempty"`

	// Details 错误详情
	// Details are error details
	Details interface{} `json:"details,omitempty"`
}

// CompleteData 完成事件数据
// CompleteData is the data for complete event
type CompleteData struct {
	// Output 输出内容
	// Output is the output content
	Output string `json:"output"`

	// Duration 运行时长（秒）
	// Duration is the run duration in seconds
	Duration float64 `json:"duration"`

	// TokenCount 令牌数量（可选）
	// TokenCount is the token count (optional)
	TokenCount int `json:"token_count,omitempty"`

	// Reasoning 推理摘要（可选）
	// Reasoning is an optional reasoning summary
	Reasoning *ReasoningSummary `json:"reasoning,omitempty"`

	// Usage 用量统计（可选）
	// Usage provides usage statistics (optional)
	Usage *UsageMetrics `json:"usage,omitempty"`

	// Status 运行状态
	Status string `json:"status,omitempty"`

	// CacheHit 缓存命中
	CacheHit bool `json:"cache_hit,omitempty"`

	// RunID 运行标识
	RunID string `json:"run_id,omitempty"`

	// CancellationReason 取消原因
	CancellationReason string `json:"cancellation_reason,omitempty"`
}

// ReasoningSummary provides a compact reasoning representation
type ReasoningSummary struct {
	Content         string  `json:"content"`
	TokenCount      *int    `json:"token_count,omitempty"`
	RedactedContent *string `json:"redacted_content,omitempty"`
	Model           string  `json:"model,omitempty"`
	Provider        string  `json:"provider,omitempty"`
}

// ReasoningData 推理事件数据
// ReasoningData is the payload for reasoning events
type ReasoningData struct {
	Content         string  `json:"content"`
	TokenCount      *int    `json:"token_count,omitempty"`
	RedactedContent *string `json:"redacted_content,omitempty"`
	MessageIndex    int     `json:"message_index"`
	Model           string  `json:"model,omitempty"`
	Provider        string  `json:"provider,omitempty"`
}

// UsageMetrics captures token usage details
type UsageMetrics struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
	ReasoningTokens  int `json:"reasoning_tokens,omitempty"`
}

// ToSSE 将事件转换为 SSE 格式
// ToSSE converts event to SSE format
func (e *Event) ToSSE() string {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Sprintf("event: error\ndata: {\"error\":\"failed to marshal event\"}\n\n")
	}
	return fmt.Sprintf("event: %s\ndata: %s\n\n", e.Type, string(data))
}

// NewEvent 创建一个新事件
// NewEvent creates a new event
func NewEvent(eventType EventType, data interface{}) *Event {
	return &Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// EventFilter 事件过滤器
// EventFilter is an event filter
type EventFilter struct {
	// Types 允许的事件类型（如果为空则允许所有）
	// Types are the allowed event types (if empty, all types are allowed)
	Types map[EventType]bool
}

// NewEventFilter 创建事件过滤器
// NewEventFilter creates an event filter from a list of type strings
func NewEventFilter(types []string) *EventFilter {
	filter := &EventFilter{
		Types: make(map[EventType]bool),
	}

	if len(types) == 0 {
		// 如果没有指定类型，允许所有类型
		// If no types specified, allow all types
		return filter
	}

	for _, t := range types {
		filter.Types[EventType(t)] = true
	}

	return filter
}

// ShouldSend 判断是否应该发送此事件
// ShouldSend determines if the event should be sent
func (f *EventFilter) ShouldSend(event *Event) bool {
	// 如果没有设置过滤器，发送所有事件
	// If no filter is set, send all events
	if len(f.Types) == 0 {
		return true
	}

	// 检查事件类型是否在允许列表中
	// Check if event type is in the allowed list
	return f.Types[event.Type]
}
