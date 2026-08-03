package anthropic

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

const (
	defaultBaseURL = "https://api.anthropic.com/v1"
	apiVersion     = "2023-06-01"
)

var nonThinkingModels = map[string]struct{}{
	"claude-3-haiku-20240307":   {},
	"claude-3-5-haiku-20241022": {},
	"claude-3-5-haiku-latest":   {},
}

// Anthropic wraps the Anthropic Claude API client
type Anthropic struct {
	models.BaseModel
	config     Config
	httpClient *http.Client
}

// Config contains Anthropic-specific configuration
// Config 包含Anthropic特定配置
type Config struct {
	APIKey            string
	BaseURL           string
	Temperature       float64
	MaxTokens         int
	Timeout           time.Duration // Request timeout / 请求超时时间
	Thinking          *ThinkingConfig
	Betas             []string
	ContextManagement map[string]interface{}
}

// ThinkingConfig represents Anthropic extended thinking configuration
type ThinkingConfig struct {
	Type            string `json:"type"`
	BudgetTokens    int    `json:"budget_tokens"`
	MaxOutputTokens int    `json:"max_output_tokens,omitempty"`
}

// New creates a new Anthropic Claude model instance
func New(modelID string, config Config) (*Anthropic, error) {
	if config.APIKey == "" {
		return nil, types.NewInvalidConfigError("Anthropic API key is required", nil)
	}

	if config.BaseURL == "" {
		config.BaseURL = defaultBaseURL
	}

	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}

	// Set default timeout if not specified
	// 如果未指定则设置默认超时时间
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second // Default 60 seconds / 默认60秒
	}

	return &Anthropic{
		BaseModel: models.BaseModel{
			ID:       modelID,
			Provider: "anthropic",
			Name:     modelID,
		},
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// SupportsReasoning indicates whether the model is configured for extended thinking
func (a *Anthropic) SupportsReasoning() bool {
	if a == nil || a.config.Thinking == nil {
		return false
	}

	cfg := a.config.Thinking
	if !strings.EqualFold(cfg.Type, "enabled") {
		return false
	}

	if cfg.BudgetTokens <= 0 {
		return false
	}

	modelID := strings.ToLower(a.ID)
	if _, exists := nonThinkingModels[modelID]; exists {
		return false
	}

	return true
}

// Invoke calls the Anthropic API synchronously
func (a *Anthropic) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	claudeReq := a.buildClaudeRequest(req)

	var claudeResp ClaudeResponse
	err := models.PostJSON(ctx, a.httpClient, a.config.BaseURL+"/messages", a.authHeaders(), claudeReq, &claudeResp)
	if err != nil {
		return nil, err
	}

	return a.convertResponse(&claudeResp), nil
}

// InvokeStream calls the Anthropic API with streaming response

// buildClaudeRequest converts InvokeRequest to Claude API request

// convertResponse converts Claude response to ModelResponse
func (a *Anthropic) convertResponse(resp *ClaudeResponse) *types.ModelResponse {
	modelResp := &types.ModelResponse{
		ID:    resp.ID,
		Model: resp.Model,
		Usage: types.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		Metadata: types.Metadata{
			FinishReason: resp.StopReason,
		},
	}

	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder
	var redactedReasoning string
	var reasoningSignature string

	// Extract text, reasoning content and tool calls
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				contentBuilder.WriteString(block.Text)
			}
		case "tool_use":
			modelResp.ToolCalls = append(modelResp.ToolCalls, models.NewToolCall(block.ID, block.Name, models.MarshalMap(block.Input)))
		case "thinking":
			if block.Thinking != "" {
				if reasoningBuilder.Len() > 0 {
					reasoningBuilder.WriteString("\n")
				}
				reasoningBuilder.WriteString(block.Thinking)
			}
			if block.Signature != "" {
				reasoningSignature = block.Signature
			}
		case "redacted_thinking":
			if block.Data != "" {
				redactedReasoning = block.Data
			}
		}
	}

	if contentBuilder.Len() > 0 {
		modelResp.Content = strings.TrimSpace(contentBuilder.String())
	}

	if reasoningBuilder.Len() > 0 {
		reasoningText := strings.TrimSpace(reasoningBuilder.String())
		if reasoningText != "" {
			reasoning := types.NewReasoningContent(reasoningText)
			if redactedReasoning != "" {
				reasoning = reasoning.WithRedacted(redactedReasoning)
			}
			if resp.Usage.ThinkingTokens > 0 {
				reasoning = reasoning.WithTokenCount(resp.Usage.ThinkingTokens)
			}
			modelResp.ReasoningContent = reasoning
		}
	}

	if reasoningSignature != "" {
		if modelResp.Metadata.Extra == nil {
			modelResp.Metadata.Extra = make(map[string]interface{})
		}
		modelResp.Metadata.Extra["reasoning_signature"] = reasoningSignature
	}

	if resp.Usage.ThinkingTokens > 0 {
		if modelResp.Metadata.Extra == nil {
			modelResp.Metadata.Extra = make(map[string]interface{})
		}
		modelResp.Metadata.Extra["thinking_tokens"] = resp.Usage.ThinkingTokens
	}
	if resp.ContextManagement != nil {
		if modelResp.Metadata.Extra == nil {
			modelResp.Metadata.Extra = make(map[string]interface{})
		}
		modelResp.Metadata.Extra["context_management"] = resp.ContextManagement
	}

	return modelResp
}

// convertStreamEvent converts stream event to ResponseChunk

// authHeaders returns the required headers for the Anthropic API.
// authHeaders 返回 Anthropic API 所需的请求头。
func (a *Anthropic) authHeaders() map[string]string {
	return map[string]string{
		"x-api-key":         a.config.APIKey,
		"anthropic-version": apiVersion,
	}
}

// ClaudeResponse represents the Anthropic API response
type ClaudeResponse struct {
	ID                string                 `json:"id"`
	Type              string                 `json:"type"`
	Role              string                 `json:"role"`
	Content           []ContentBlock         `json:"content"`
	Model             string                 `json:"model"`
	StopReason        string                 `json:"stop_reason"`
	Usage             ClaudeUsage            `json:"usage"`
	ContextManagement map[string]interface{} `json:"context_management,omitempty"`
}

// ContentBlock represents a content block in the response
type ContentBlock struct {
	Type      string                 `json:"type"`
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Thinking  string                 `json:"thinking,omitempty"`
	Signature string                 `json:"signature,omitempty"`
	Data      string                 `json:"data,omitempty"`
}

// ClaudeUsage represents token usage
type ClaudeUsage struct {
	InputTokens    int `json:"input_tokens"`
	OutputTokens   int `json:"output_tokens"`
	ThinkingTokens int `json:"thinking_tokens,omitempty"`
}

// ValidateConfig validates the Anthropic configuration
func ValidateConfig(config Config) error {
	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}
