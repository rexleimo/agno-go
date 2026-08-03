package openai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
	"github.com/sashabaranov/go-openai"
)

// OpenAI wraps the OpenAI client
type OpenAI struct {
	models.BaseModel
	client *openai.Client
	config Config
}

// Config contains OpenAI-specific configuration
// Config 包含OpenAI特定配置
type Config struct {
	APIKey      string
	BaseURL     string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration // Request timeout / 请求超时时间
}

// New creates a new OpenAI model instance
func New(modelID string, config Config) (*OpenAI, error) {
	if config.APIKey == "" {
		return nil, types.NewInvalidConfigError("OpenAI API key is required", nil)
	}

	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	// Set timeout on HTTP client
	// 在HTTP客户端上设置超时
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second // Default 60 seconds / 默认60秒
	}
	clientConfig.HTTPClient = &http.Client{
		Timeout: timeout,
	}

	return &OpenAI{
		BaseModel: models.BaseModel{
			ID:       modelID,
			Provider: "openai",
		},
		client: openai.NewClientWithConfig(clientConfig),
		config: config,
	}, nil
}

// Invoke calls the OpenAI API synchronously
func (o *OpenAI) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	chatReq := o.buildChatRequest(req)

	resp, err := o.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, types.NewAPIError("failed to call OpenAI API", err)
	}

	if len(resp.Choices) == 0 {
		return nil, types.NewAPIError("no response from OpenAI", nil)
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

// InvokeStream calls the OpenAI API with streaming response
func (o *OpenAI) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	chatReq := o.buildChatRequest(req)
	chatReq.Stream = true

	stream, err := o.client.CreateChatCompletionStream(ctx, chatReq)
	if err != nil {
		return nil, types.NewAPIError("failed to create OpenAI stream", err)
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

// ValidateConfig validates the OpenAI configuration
func ValidateConfig(config Config) error {
	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}
