package anthropic

import (
	"context"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Extractor 从 Anthropic 响应中提取推理内容
// Extractor extracts reasoning content from Anthropic responses
type Extractor struct{}

// Extract 提取推理内容
// Extracts reasoning content
func (e *Extractor) Extract(ctx context.Context, response *types.ModelResponse) (*types.ReasoningContent, error) {
	// Anthropic 的推理内容应该已经在响应中
	// Anthropic's reasoning content should already be in the response
	if response == nil || response.ReasoningContent == nil {
		return nil, nil
	}

	return response.ReasoningContent, nil
}

// Provider 返回提供商名称
// Returns the provider name
func (e *Extractor) Provider() string {
	return "anthropic"
}
