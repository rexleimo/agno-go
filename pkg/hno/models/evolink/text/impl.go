package text

import (
	"context"
	"fmt"

	evolink "github.com/rexleimo/agno-go/pkg/hno/models/evolink"
)

// Options contains text generation parameters
type Options struct {
	Model       string
	Temperature float64
	MaxTokens   int
	Messages    []map[string]interface{}
}

// Response is a minimal chat completion response
type Response struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content string `json:"content"`
	// Optional token usage if API provides it
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Complete sends a chat completion request
func Complete(ctx context.Context, c *evolink.Client, opts Options) (*Response, error) {
	if len(opts.Messages) == 0 {
		return nil, fmt.Errorf("messages are required")
	}
	payload := map[string]interface{}{
		"model":    opts.Model,
		"messages": opts.Messages,
	}
	if opts.Temperature > 0 {
		payload["temperature"] = opts.Temperature
	}
	if opts.MaxTokens > 0 {
		payload["max_tokens"] = opts.MaxTokens
	}
	var resp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := c.PostJSON(ctx, "/v1/chat/completions", payload, &resp); err != nil {
		return nil, err
	}
	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}
	return &Response{
		ID:      resp.ID,
		Model:   resp.Model,
		Content: content,
		Usage:   resp.Usage,
	}, nil
}
