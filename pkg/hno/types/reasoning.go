package types

// ReasoningContent 表示模型的推理内容
// ReasoningContent represents the model's reasoning process
type ReasoningContent struct {
	// Content 是完整的推理内容
	// Content is the full reasoning content
	Content string `json:"content"`

	// RedactedContent 是脱敏后的推理内容(可选)
	// RedactedContent is the redacted version of reasoning (optional)
	RedactedContent *string `json:"redacted_content,omitempty"`

	// TokenCount 是推理内容的 token 数量(可选)
	// TokenCount is the number of tokens in reasoning (optional)
	TokenCount *int `json:"token_count,omitempty"`
}

// NewReasoningContent 创建一个新的 ReasoningContent
// NewReasoningContent creates a new ReasoningContent
func NewReasoningContent(content string) *ReasoningContent {
	return &ReasoningContent{
		Content: content,
	}
}

// WithRedacted 设置脱敏内容
// WithRedacted sets the redacted content
func (r *ReasoningContent) WithRedacted(redacted string) *ReasoningContent {
	r.RedactedContent = &redacted
	return r
}

// WithTokenCount 设置 token 数量
// WithTokenCount sets the token count
func (r *ReasoningContent) WithTokenCount(count int) *ReasoningContent {
	r.TokenCount = &count
	return r
}
