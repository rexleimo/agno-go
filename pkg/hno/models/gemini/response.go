package gemini

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func (g *Gemini) convertResponse(resp *GeminiResponse) *types.ModelResponse {
	if resp == nil {
		return &types.ModelResponse{
			Model: g.ID,
		}
	}

	modelResp := &types.ModelResponse{
		Model: g.ID,
		Usage: types.Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		},
	}

	if len(resp.Candidates) == 0 {
		return modelResp
	}

	candidate := resp.Candidates[0]
	modelResp.Metadata.FinishReason = candidate.FinishReason
	if resp.UsageMetadata.ThoughtsTokenCount > 0 {
		if modelResp.Metadata.Extra == nil {
			modelResp.Metadata.Extra = make(map[string]interface{})
		}
		modelResp.Metadata.Extra["thoughts_token_count"] = resp.UsageMetadata.ThoughtsTokenCount
	}

	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder

	// Extract content, reasoning and tool calls
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			if part.Thought {
				if reasoningBuilder.Len() > 0 {
					reasoningBuilder.WriteString("\n")
				}
				reasoningBuilder.WriteString(part.Text)
			} else {
				contentBuilder.WriteString(part.Text)
			}
		}

		if part.FunctionCall != nil {
			modelResp.ToolCalls = append(modelResp.ToolCalls, models.NewToolCall(
				models.NewToolCallID(),
				part.FunctionCall.Name,
				models.MarshalMap(part.FunctionCall.Args),
			))
		}
	}

	content := strings.TrimSpace(contentBuilder.String())
	if content != "" {
		modelResp.Content = content
	}

	if reasoningBuilder.Len() > 0 {
		reasoning := strings.TrimSpace(reasoningBuilder.String())
		if reasoning != "" {
			rc := types.NewReasoningContent(reasoning)
			if resp.UsageMetadata.ThoughtsTokenCount > 0 {
				rc = rc.WithTokenCount(resp.UsageMetadata.ThoughtsTokenCount)
			}
			modelResp.ReasoningContent = rc
		}
	}

	return modelResp
}

// convertToChunk converts Gemini response to ResponseChunk for streaming
func (g *Gemini) convertToChunk(resp *GeminiResponse) types.ResponseChunk {
	chunk := types.ResponseChunk{}

	if len(resp.Candidates) == 0 {
		chunk.Done = true
		return chunk
	}

	candidate := resp.Candidates[0]

	// Check if done
	if candidate.FinishReason != "" && candidate.FinishReason != "STOP" {
		chunk.Done = true
		return chunk
	}

	// Extract content
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			chunk.Content += part.Text
		}

		if part.FunctionCall != nil {
			argsJSON, _ := json.Marshal(part.FunctionCall.Args)
			chunk.ToolCalls = append(chunk.ToolCalls, types.ToolCall{
				ID:   models.NewToolCallID(),
				Type: "function",
				Function: types.ToolCallFunction{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	return chunk
}

// GeminiResponse represents the Gemini API response
type GeminiResponse struct {
	Candidates    []Candidate   `json:"candidates"`
	UsageMetadata UsageMetadata `json:"usageMetadata"`
}

// Candidate represents a response candidate
type Candidate struct {
	Content      Content `json:"content"`
	FinishReason string  `json:"finishReason,omitempty"`
	Index        int     `json:"index"`
}

// UsageMetadata represents usage information
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
	ThoughtsTokenCount   int `json:"thoughtsTokenCount,omitempty"`
}

// ValidateConfig validates the Gemini configuration
func ValidateConfig(config Config) error {
	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}
