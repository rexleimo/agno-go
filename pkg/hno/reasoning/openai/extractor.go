package openai

import (
	"context"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Extractor 从 OpenAI 响应中提取推理内容
// Extractor extracts reasoning content from OpenAI responses
type Extractor struct{}

// Extract 提取推理内容
// Extracts reasoning content
func (e *Extractor) Extract(ctx context.Context, response *types.ModelResponse) (*types.ReasoningContent, error) {
	if response == nil || response.Content == "" {
		return nil, nil
	}

	content := response.Content
	reasoningText := ""

	// 尝试提取 <think> 标签内容 / Try to extract <think> tags
	if strings.Contains(content, "<think>") && strings.Contains(content, "</think>") {
		startIdx := strings.Index(content, "<think>") + len("<think>")
		endIdx := strings.Index(content, "</think>")
		if startIdx < endIdx {
			reasoningText = strings.TrimSpace(content[startIdx:endIdx])
		}
	}

	// 如果没有找到标签,检查响应中是否已经有 ReasoningContent
	// If no tags found, check if response already has ReasoningContent
	if reasoningText == "" && response.ReasoningContent != nil {
		return response.ReasoningContent, nil
	}

	// 如果仍然为空,返回 nil
	// If still empty, return nil
	if reasoningText == "" {
		return nil, nil
	}

	return types.NewReasoningContent(reasoningText), nil
}

// Provider 返回提供商名称
// Returns the provider name
func (e *Extractor) Provider() string {
	return "openai"
}
