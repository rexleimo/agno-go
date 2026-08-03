package types

// ModelResponse represents the response from a language model
type ModelResponse struct {
	ID        string     `json:"id,omitempty"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Usage     Usage      `json:"usage,omitempty"`
	Model     string     `json:"model,omitempty"`
	Metadata  Metadata   `json:"metadata,omitempty"`

	// ReasoningContent 包含模型的推理过程(仅推理模型)
	// ReasoningContent contains the model's reasoning process (reasoning models only)
	ReasoningContent *ReasoningContent `json:"reasoning_content,omitempty"`
}

// Usage contains token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Metadata contains additional response metadata
type Metadata struct {
	FinishReason string                 `json:"finish_reason,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

// ResponseChunk represents a streaming response chunk
type ResponseChunk struct {
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Done      bool       `json:"done"`
	Error     error      `json:"error,omitempty"`
}

// HasToolCalls checks if the response contains tool calls
func (r *ModelResponse) HasToolCalls() bool {
	return len(r.ToolCalls) > 0
}

// IsEmpty checks if the response is empty
func (r *ModelResponse) IsEmpty() bool {
	return r.Content == "" && len(r.ToolCalls) == 0
}
