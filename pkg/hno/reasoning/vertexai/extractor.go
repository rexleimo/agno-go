package vertexai

import (
	"context"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Extractor 从 VertexAI 响应中提取推理内容
// Extractor extracts reasoning content from VertexAI responses
type Extractor struct{}

// Provider 返回提供商名称
func (e *Extractor) Provider() string {
	return "vertexai"
}

// Extract 提取推理内容
func (e *Extractor) Extract(ctx context.Context, response *types.ModelResponse) (*types.ReasoningContent, error) {
	if response == nil || response.ReasoningContent == nil {
		return nil, nil
	}

	return response.ReasoningContent, nil
}
