package openai

import (
	"encoding/json"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
	"github.com/sashabaranov/go-openai"
)

// partialToolCall accumulates a streaming tool-call across chunks.
// partialToolCall 跨 chunk 累积流式工具调用。
type partialToolCall struct {
	id        string
	typ       string
	name      string
	arguments string
}

// buildChatRequest converts InvokeRequest to OpenAI ChatCompletionRequest.
// buildChatRequest 将 InvokeRequest 转换为 OpenAI ChatCompletionRequest。
func (o *OpenAI) buildChatRequest(req *models.InvokeRequest) openai.ChatCompletionRequest {
	chatReq := openai.ChatCompletionRequest{
		Model:    o.ID,
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
	} else if o.config.Temperature > 0 || o.config.TemperatureSet {
		chatReq.Temperature = float32(o.config.Temperature)
	}
	if o.config.Seed != nil {
		chatReq.Seed = o.config.Seed
	}

	// Set max tokens
	// 设置最大 token 数
	if req.MaxTokens > 0 {
		chatReq.MaxTokens = req.MaxTokens
	} else if o.config.MaxTokens > 0 {
		chatReq.MaxTokens = o.config.MaxTokens
	}

	return chatReq
}

// toToolCalls converts OpenAI tool calls to framework ToolCall values.
// toToolCalls 将 OpenAI 工具调用转换为框架 ToolCall 值。
func toToolCalls(calls []openai.ToolCall) []types.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]types.ToolCall, len(calls))
	for i, tc := range calls {
		out[i] = models.NewToolCall(tc.ID, tc.Function.Name, tc.Function.Arguments)
	}
	return out
}
