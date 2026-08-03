package glm

// glmMessage represents a message in GLM API format
// glmMessage 表示 GLM API 格式的消息
type glmMessage struct {
	Role       string        `json:"role"`
	Content    string        `json:"content"`
	Name       string        `json:"name,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
	ToolCalls  []glmToolCall `json:"tool_calls,omitempty"`
}

// glmToolCall represents a tool call in GLM API format
// glmToolCall 表示 GLM API 格式的工具调用
type glmToolCall struct {
	ID       string              `json:"id"`
	Type     string              `json:"type"`
	Function glmToolCallFunction `json:"function"`
}

// glmToolCallFunction contains function call details
// glmToolCallFunction 包含函数调用详情
type glmToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// glmTool represents a tool definition in GLM API format
// glmTool 表示 GLM API 格式的工具定义
type glmTool struct {
	Type     string            `json:"type"`
	Function glmFunctionSchema `json:"function"`
}

// glmFunctionSchema defines the schema of a callable function
// glmFunctionSchema 定义可调用函数的架构
type glmFunctionSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// glmRequest represents the request to GLM API
// glmRequest 表示对 GLM API 的请求
type glmRequest struct {
	Model       string       `json:"model"`
	Messages    []glmMessage `json:"messages"`
	Temperature *float64     `json:"temperature,omitempty"`
	TopP        *float64     `json:"top_p,omitempty"`
	MaxTokens   *int         `json:"max_tokens,omitempty"`
	Stream      bool         `json:"stream,omitempty"`
	Tools       []glmTool    `json:"tools,omitempty"`
	ToolChoice  string       `json:"tool_choice,omitempty"`
	DoSample    *bool        `json:"do_sample,omitempty"`
	RequestID   string       `json:"request_id,omitempty"`
	UserID      string       `json:"user_id,omitempty"`
}

// glmResponse represents the response from GLM API
// glmResponse 表示来自 GLM API 的响应
type glmResponse struct {
	ID      string      `json:"id"`
	Created int64       `json:"created"`
	Model   string      `json:"model"`
	Choices []glmChoice `json:"choices"`
	Usage   glmUsage    `json:"usage"`
}

// glmChoice represents a single response choice
// glmChoice 表示单个响应选项
type glmChoice struct {
	Index        int         `json:"index"`
	Message      glmMessage  `json:"message"`
	FinishReason string      `json:"finish_reason"`
	Delta        *glmMessage `json:"delta,omitempty"` // For streaming
}

// glmUsage contains token usage information
// glmUsage 包含 token 使用信息
type glmUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// glmErrorResponse represents an error response from GLM API
// glmErrorResponse 表示来自 GLM API 的错误响应
type glmErrorResponse struct {
	Error glmError `json:"error"`
}

// glmError contains error details
// glmError 包含错误详情
type glmError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

// glmStreamResponse represents a streaming response chunk
// glmStreamResponse 表示流式响应块
type glmStreamResponse struct {
	ID      string      `json:"id"`
	Created int64       `json:"created"`
	Model   string      `json:"model"`
	Choices []glmChoice `json:"choices"`
}
