package glm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rexleimo/agno-go/pkg/agno/models"
	"github.com/rexleimo/agno-go/pkg/agno/types"
)

const (
	// DefaultBaseURL is the default GLM API endpoint
	// DefaultBaseURL 是默认的 GLM API 端点
	DefaultBaseURL = "https://open.bigmodel.cn/api/paas/v4"
)

// GLM represents the Zhipu AI GLM model client
// GLM 表示智谱AI GLM模型客户端
type GLM struct {
	models.BaseModel
	config     Config
	httpClient *http.Client
	keyID      string
	keySecret  string
}

// Config contains GLM-specific configuration
// Config 包含 GLM 特定的配置
type Config struct {
	APIKey      string  // Format: {key_id}.{key_secret} / 格式: {key_id}.{key_secret}
	BaseURL     string  // Default: https://open.bigmodel.cn/api/paas/v4 / 默认值
	Temperature float64 // Temperature parameter / 温度参数
	MaxTokens   int     // Max tokens to generate / 生成的最大 token 数
	TopP        float64 // Top-p sampling parameter / Top-p 采样参数
	DoSample    bool    // Whether to use sampling / 是否使用采样
}

// New creates a new GLM model instance with the given configuration.
// Returns an error if the API key is missing or malformed.
// New 使用给定的配置创建一个新的 GLM 模型实例。
// 如果 API key 缺失或格式错误，返回错误。
func New(modelID string, config Config) (*GLM, error) {
	if config.APIKey == "" {
		return nil, types.NewInvalidConfigError("GLM API key is required", nil)
	}

	// Parse API key into keyID and keySecret
	// 解析 API key 为 keyID 和 keySecret
	keyID, keySecret, err := parseAPIKey(config.APIKey)
	if err != nil {
		return nil, types.NewInvalidConfigError(fmt.Sprintf("invalid GLM API key: %v", err), err)
	}

	// Set default base URL if not provided
	// 如果未提供，设置默认 base URL
	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}

	return &GLM{
		BaseModel: models.BaseModel{
			ID:       modelID,
			Provider: "glm",
		},
		config:     config,
		httpClient: &http.Client{},
		keyID:      keyID,
		keySecret:  keySecret,
	}, nil
}

// Invoke calls the GLM API synchronously
// Invoke 同步调用 GLM API
func (g *GLM) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	// Build GLM request
	// 构建 GLM 请求
	glmReq, err := g.buildGLMRequest(req, false)
	if err != nil {
		return nil, types.NewInvalidInputError(fmt.Sprintf("failed to build GLM request: %v", err), err)
	}

	// Make HTTP request
	// 发起 HTTP 请求
	glmResp, err := g.makeRequest(ctx, glmReq)
	if err != nil {
		return nil, err
	}

	// Convert to ModelResponse
	// 转换为 ModelResponse
	return g.convertToModelResponse(glmResp), nil
}

// InvokeStream calls the GLM API with streaming response
// InvokeStream 使用流式响应调用 GLM API
func (g *GLM) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	// Build GLM request with streaming enabled
	// 构建启用流式传输的 GLM 请求
	glmReq, err := g.buildGLMRequest(req, true)
	if err != nil {
		return nil, types.NewInvalidInputError(fmt.Sprintf("failed to build GLM request: %v", err), err)
	}

	// Create streaming HTTP request
	// 创建流式 HTTP 请求
	httpReq, err := g.createHTTPRequest(ctx, glmReq)
	if err != nil {
		return nil, err
	}

	// Execute request
	// 执行请求
	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, types.NewAPIError("failed to call GLM API", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, types.NewAPIError(fmt.Sprintf("GLM API error (status %d): %s", resp.StatusCode, string(body)), nil)
	}

	// Create channel for streaming chunks
	// 为流式块创建通道
	chunks := make(chan types.ResponseChunk)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		// Use the shared SSE decoder.
		// 使用共享 SSE 解码器。
		decoder := models.NewSSEDecoder(resp.Body)
		for {
			data, err := decoder.Next()
			if err != nil {
				if err != io.EOF {
					chunks <- types.ResponseChunk{Done: true, Error: err}
				}
				return
			}
			if len(data) == 0 {
				continue
			}

			var streamResp glmStreamResponse
			if err := json.Unmarshal(data, &streamResp); err != nil {
				chunks <- types.ResponseChunk{
					Done:  true,
					Error: fmt.Errorf("failed to parse streaming response: %w", err),
				}
				return
			}

			// Extract content from delta
			// 从 delta 提取内容
			if len(streamResp.Choices) > 0 {
				choice := streamResp.Choices[0]
				chunk := types.ResponseChunk{}

				if choice.Delta != nil {
					chunk.Content = choice.Delta.Content

					// Handle tool calls in stream
					// 处理流中的工具调用
					if len(choice.Delta.ToolCalls) > 0 {
						chunk.ToolCalls = make([]types.ToolCall, len(choice.Delta.ToolCalls))
						for i, tc := range choice.Delta.ToolCalls {
							chunk.ToolCalls[i] = types.ToolCall{
								ID:   tc.ID,
								Type: tc.Type,
								Function: types.ToolCallFunction{
									Name:      tc.Function.Name,
									Arguments: tc.Function.Arguments,
								},
							}
						}
					}
				}

				// Send chunk
				// 发送块
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
		}
	}()

	return chunks, nil
}

// buildGLMRequest converts InvokeRequest to GLM request format
// buildGLMRequest 将 InvokeRequest 转换为 GLM 请求格式
func (g *GLM) buildGLMRequest(req *models.InvokeRequest, stream bool) (*glmRequest, error) {
	glmReq := &glmRequest{
		Model:    g.ID,
		Messages: make([]glmMessage, len(req.Messages)),
		Stream:   stream,
	}

	// Convert messages
	// 转换消息
	for i, msg := range req.Messages {
		glmMsg := glmMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
		}

		// Convert tool calls
		// 转换工具调用
		if len(msg.ToolCalls) > 0 {
			glmMsg.ToolCalls = make([]glmToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				glmMsg.ToolCalls[j] = glmToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: glmToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		glmReq.Messages[i] = glmMsg
	}

	// Convert tools
	// 转换工具
	if len(req.Tools) > 0 {
		glmReq.Tools = make([]glmTool, len(req.Tools))
		for i, tool := range req.Tools {
			glmReq.Tools[i] = glmTool{
				Type: tool.Type,
				Function: glmFunctionSchema{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			}
		}
		glmReq.ToolChoice = "auto"
	}

	// Set temperature
	// 设置温度
	if req.Temperature > 0 {
		temp := req.Temperature
		glmReq.Temperature = &temp
	} else if g.config.Temperature > 0 {
		temp := g.config.Temperature
		glmReq.Temperature = &temp
	}

	// Set max tokens
	// 设置最大 token 数
	if req.MaxTokens > 0 {
		glmReq.MaxTokens = &req.MaxTokens
	} else if g.config.MaxTokens > 0 {
		maxTokens := g.config.MaxTokens
		glmReq.MaxTokens = &maxTokens
	}

	// Set top_p if configured
	// 如果配置了，设置 top_p
	if g.config.TopP > 0 {
		topP := g.config.TopP
		glmReq.TopP = &topP
	}

	// Set do_sample if configured
	// 如果配置了，设置 do_sample
	if g.config.DoSample {
		doSample := true
		glmReq.DoSample = &doSample
	}

	return glmReq, nil
}

// makeRequest makes an HTTP request to GLM API and returns the response
// makeRequest 向 GLM API 发起 HTTP 请求并返回响应
func (g *GLM) makeRequest(ctx context.Context, glmReq *glmRequest) (*glmResponse, error) {
	httpReq, err := g.createHTTPRequest(ctx, glmReq)
	if err != nil {
		return nil, err
	}

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, types.NewAPIError("failed to call GLM API", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewAPIError("failed to read GLM response", err)
	}

	// Check for errors
	// 检查错误
	if resp.StatusCode != http.StatusOK {
		var errResp glmErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, types.NewAPIError(fmt.Sprintf("GLM API error: %s (code: %s)", errResp.Error.Message, errResp.Error.Code), nil)
		}
		return nil, types.NewAPIError(fmt.Sprintf("GLM API error (status %d): %s", resp.StatusCode, string(body)), nil)
	}

	// Parse response
	// 解析响应
	var glmResp glmResponse
	if err := json.Unmarshal(body, &glmResp); err != nil {
		return nil, types.NewAPIError("failed to parse GLM response", err)
	}

	return &glmResp, nil
}

// createHTTPRequest creates an HTTP request with JWT authentication
// createHTTPRequest 创建带有 JWT 认证的 HTTP 请求
func (g *GLM) createHTTPRequest(ctx context.Context, glmReq *glmRequest) (*http.Request, error) {
	// Generate JWT token
	// 生成 JWT 令牌
	token, err := generateJWT(g.keyID, g.keySecret)
	if err != nil {
		return nil, types.NewAPIError("failed to generate JWT token", err)
	}

	// Marshal request body
	// 序列化请求体
	body, err := json.Marshal(glmReq)
	if err != nil {
		return nil, types.NewInvalidInputError("failed to marshal request", err)
	}

	// Create HTTP request
	// 创建 HTTP 请求
	url := fmt.Sprintf("%s/chat/completions", g.config.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, types.NewAPIError("failed to create HTTP request", err)
	}

	// Set headers
	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	return httpReq, nil
}

// convertToModelResponse converts GLM response to ModelResponse
// convertToModelResponse 将 GLM 响应转换为 ModelResponse
func (g *GLM) convertToModelResponse(glmResp *glmResponse) *types.ModelResponse {
	if len(glmResp.Choices) == 0 {
		return &types.ModelResponse{}
	}

	choice := glmResp.Choices[0]
	modelResp := &types.ModelResponse{
		ID:      glmResp.ID,
		Content: choice.Message.Content,
		Model:   glmResp.Model,
		Usage: types.Usage{
			PromptTokens:     glmResp.Usage.PromptTokens,
			CompletionTokens: glmResp.Usage.CompletionTokens,
			TotalTokens:      glmResp.Usage.TotalTokens,
		},
		Metadata: types.Metadata{
			FinishReason: choice.FinishReason,
		},
	}

	// Convert tool calls if present
	// 如果存在，转换工具调用
	if len(choice.Message.ToolCalls) > 0 {
		modelResp.ToolCalls = make([]types.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			modelResp.ToolCalls[i] = types.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: types.ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	return modelResp
}
