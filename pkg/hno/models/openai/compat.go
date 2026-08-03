package openai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
	"github.com/sashabaranov/go-openai"
)

// OpenAICompat is a shared provider for OpenAI-compatible APIs. All
// providers that speak the OpenAI chat-completions protocol (DeepSeek,
// Groq, ModelScope, OpenRouter, etc.) are thin wrappers around this type;
// they only supply their name and default base URL. See NewCompat.
//
// OpenAICompat 是 OpenAI 兼容 API 的共享 provider。所有使用 OpenAI
// chat-completions 协议的 provider（DeepSeek、Groq、ModelScope、
// OpenRouter 等）都是此类型的薄包装；它们只需提供自己的名称与默认
// Base URL。参见 NewCompat。
type OpenAICompat struct {
	models.BaseModel
	client *openai.Client
	config CompatConfig
}

// CompatConfig holds the configuration for an OpenAI-compatible provider.
//
// CompatConfig 保存 OpenAI 兼容 provider 的配置。
type CompatConfig struct {
	// Provider is the provider name recorded in BaseModel (e.g. "deepseek").
	// Provider 是记录在 BaseModel 中的 provider 名（如 "deepseek"）。
	Provider string
	// APIKey used for authorization. May be empty for local servers.
	// APIKey 用于鉴权。本地服务器可为空。
	APIKey string
	// BaseURL overrides the default endpoint. Empty means default.
	// BaseURL 覆盖默认端点。空表示使用默认值。
	BaseURL string
	// DefaultBaseURL is used when BaseURL is empty.
	// DefaultBaseURL 在 BaseURL 为空时使用。
	DefaultBaseURL string
	// Temperature and MaxTokens fallbacks when the request does not set them.
	// Temperature 与 MaxTokens 在请求未设置时的回退值。
	Temperature float64
	MaxTokens   int
	// Timeout for HTTP requests; defaults to 60s.
	// Timeout HTTP 请求超时；默认 60 秒。
	Timeout time.Duration
	// ExtraHeaders are added to every request (e.g. OpenRouter Referer).
	// ExtraHeaders 附加到每个请求（如 OpenRouter 的 Referer）。
	ExtraHeaders map[string]string
	// RequireAPIKey forces APIKey validation in NewCompat.
	// RequireAPIKey 在 NewCompat 中强制校验 APIKey。
	RequireAPIKey bool
}

// NewCompat creates an OpenAI-compatible model instance.
//
// NewCompat 创建 OpenAI 兼容模型实例。
func NewCompat(modelID string, cfg CompatConfig) (*OpenAICompat, error) {
	if cfg.RequireAPIKey && cfg.APIKey == "" {
		return nil, types.NewInvalidConfigError(cfg.Provider+" API key is required", nil)
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = cfg.DefaultBaseURL
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}

	clientConfig := openai.DefaultConfig(cfg.APIKey)
	clientConfig.BaseURL = cfg.BaseURL

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	httpClient := &http.Client{Timeout: timeout}

	// Apply extra headers via a custom round tripper (e.g. OpenRouter
	// Referer/Title). 通过自定义 round tripper 应用额外请求头
	// （如 OpenRouter 的 Referer/Title）。
	if len(cfg.ExtraHeaders) > 0 {
		httpClient.Transport = &headerRoundTripper{
			base:    http.DefaultTransport,
			headers: cfg.ExtraHeaders,
		}
	}
	clientConfig.HTTPClient = httpClient

	return &OpenAICompat{
		BaseModel: models.BaseModel{
			ID:       modelID,
			Provider: cfg.Provider,
			Name:     modelID,
		},
		client: openai.NewClientWithConfig(clientConfig),
		config: cfg,
	}, nil
}

// headerRoundTripper adds static headers to every outgoing request.
// headerRoundTripper 为每个出站请求添加静态请求头。
type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

// RoundTrip implements http.RoundTripper.
// RoundTrip 实现 http.RoundTripper。
func (h headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range h.headers {
		req.Header.Set(k, v)
	}
	return h.base.RoundTrip(req)
}

// Invoke calls the provider API synchronously.
// Invoke 同步调用 provider API。
func (c *OpenAICompat) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	chatReq := c.buildChatRequest(req)

	resp, err := c.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, types.NewAPIError("failed to call "+c.config.Provider+" API", err)
	}

	if len(resp.Choices) == 0 {
		return nil, types.NewAPIError("no response from "+c.config.Provider, nil)
	}

	choice := resp.Choices[0]
	modelResp := &types.ModelResponse{
		ID:      resp.ID,
		Content: choice.Message.Content,
		Model:   resp.Model,
		Usage: types.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Metadata: types.Metadata{
			FinishReason: string(choice.FinishReason),
		},
	}

	// Convert tool calls if present
	// 转换工具调用（如有）
	modelResp.ToolCalls = toToolCalls(choice.Message.ToolCalls)

	return modelResp, nil
}

// InvokeStream calls the provider API with streaming response.
// InvokeStream 以流式响应调用 provider API。
func (c *OpenAICompat) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	chatReq := c.buildChatRequest(req)
	chatReq.Stream = true

	stream, err := c.client.CreateChatCompletionStream(ctx, chatReq)
	if err != nil {
		return nil, types.NewAPIError("failed to create "+c.config.Provider+" stream", err)
	}

	chunks := make(chan types.ResponseChunk)

	go func() {
		defer close(chunks)
		defer stream.Close()

		// Accumulate streaming tool-call deltas by index. Many
		// OpenAI-compatible servers (llama.cpp included) send tool-call
		// arguments incrementally across multiple chunks; emitting each
		// delta as a complete call would produce broken calls.
		// 按索引累积流式工具调用增量。许多 OpenAI 兼容服务器
		// （包括 llama.cpp）会跨多个 chunk 增量发送工具调用参数；
		// 将每个增量当作完整调用发出会产生损坏的调用。
		partialCalls := make(map[int]*partialToolCall)

		flushToolCalls := func() {
			if len(partialCalls) == 0 {
				return
			}
			calls := make([]types.ToolCall, 0, len(partialCalls))
			for i := 0; i < len(partialCalls); i++ {
				pc := partialCalls[i]
				if pc == nil || pc.id == "" {
					continue
				}
				calls = append(calls, types.ToolCall{
					ID:   pc.id,
					Type: pc.typ,
					Function: types.ToolCallFunction{
						Name:      pc.name,
						Arguments: pc.arguments,
					},
				})
				delete(partialCalls, i)
			}
			if len(calls) > 0 {
				select {
				case chunks <- types.ResponseChunk{ToolCalls: calls}:
				case <-ctx.Done():
				}
			}
		}

		for {
			response, err := stream.Recv()
			if err != nil {
				// Treat EOF as a normal stream end. Some OpenAI-compatible
				// servers (e.g. llama.cpp) close the connection without
				// sending the standard "[DONE]" sentinel.
				// 将 EOF 视为正常流结束。部分 OpenAI 兼容服务器
				// （如 llama.cpp）在关闭连接时不发送标准的 "[DONE]" 哨兵。
				if errors.Is(err, io.EOF) {
					flushToolCalls()
					chunks <- types.ResponseChunk{Done: true}
					return
				}
				chunks <- types.ResponseChunk{
					Done:  true,
					Error: err,
				}
				return
			}

			if len(response.Choices) == 0 {
				continue
			}

			delta := response.Choices[0].Delta
			chunk := types.ResponseChunk{
				Content: delta.Content,
			}

			// Accumulate tool-call deltas instead of emitting them raw.
			// 累积工具调用增量，而非原样发出。
			for _, tc := range delta.ToolCalls {
				idx := 0
				if tc.Index != nil {
					idx = *tc.Index
				}
				pc := partialCalls[idx]
				if pc == nil {
					pc = &partialToolCall{}
					partialCalls[idx] = pc
				}
				if tc.ID != "" {
					pc.id = tc.ID
				}
				if tc.Type != "" {
					pc.typ = string(tc.Type)
				}
				if tc.Function.Name != "" {
					pc.name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					pc.arguments += tc.Function.Arguments
				}
			}

			select {
			case chunks <- chunk:
			case <-ctx.Done():
				chunks <- types.ResponseChunk{
					Done:  true,
					Error: ctx.Err(),
				}
				return
			}
		}
	}()

	return chunks, nil
}

// buildChatRequest converts InvokeRequest to OpenAI ChatCompletionRequest.
// buildChatRequest 将 InvokeRequest 转换为 OpenAI ChatCompletionRequest。
func (c *OpenAICompat) buildChatRequest(req *models.InvokeRequest) openai.ChatCompletionRequest {
	chatReq := openai.ChatCompletionRequest{
		Model:    c.ID,
		Messages: make([]openai.ChatCompletionMessage, len(req.Messages)),
	}

	// Convert messages
	// 转换消息
	for i, msg := range req.Messages {
		chatMsg := openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
			Name:    msg.Name,
		}

		// Handle tool call responses
		// 处理工具调用响应
		if msg.ToolCallID != "" {
			chatMsg.ToolCallID = msg.ToolCallID
		}

		// Handle tool calls in message
		// 处理消息中的工具调用
		if len(msg.ToolCalls) > 0 {
			chatMsg.ToolCalls = make([]openai.ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				chatMsg.ToolCalls[j] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolType(tc.Type),
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		chatReq.Messages[i] = chatMsg
	}

	// Convert tools
	// 转换工具
	if len(req.Tools) > 0 {
		chatReq.Tools = make([]openai.Tool, len(req.Tools))
		for i, tool := range req.Tools {
			paramsJSON, _ := json.Marshal(tool.Function.Parameters)
			var params map[string]interface{}
			json.Unmarshal(paramsJSON, &params)

			chatReq.Tools[i] = openai.Tool{
				Type: openai.ToolType(tool.Type),
				Function: &openai.FunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  params,
				},
			}
		}
	}

	// Set temperature
	// 设置温度
	if req.Temperature > 0 {
		chatReq.Temperature = float32(req.Temperature)
	} else if c.config.Temperature > 0 {
		chatReq.Temperature = float32(c.config.Temperature)
	}

	// Set max tokens
	// 设置最大 token 数
	if req.MaxTokens > 0 {
		chatReq.MaxTokens = req.MaxTokens
	} else if c.config.MaxTokens > 0 {
		chatReq.MaxTokens = c.config.MaxTokens
	}

	return chatReq
}
