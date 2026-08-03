package openai

import (
	"context"
	"encoding/json"

	"github.com/rexleimo/agno-go/pkg/agno/models"
	"github.com/rexleimo/agno-go/pkg/agno/types"
	"github.com/sashabaranov/go-openai"
)

// SupportsImages reports whether the model accepts image inputs.
// SupportsImages 报告模型是否接受图像输入。
func (o *OpenAI) SupportsImages() bool {
	switch o.ID {
	case "gpt-4o", "gpt-4o-mini", "gpt-4-vision-preview", "gpt-4o-2024-08-06", "gpt-4.1", "gpt-4.1-mini":
		return true
	default:
		return false
	}
}

// InvokeStructured invokes the model with a response-format constraint
// (json_object or json_schema). Providers without this capability fall back
// to plain Invoke; callers use models.AsStructuredOutput to detect support.
// InvokeStructured 以响应格式约束（json_object 或 json_schema）调用模型。
// 没有此能力的提供商回退到普通 Invoke；调用方使用 models.AsStructuredOutput 探测支持。
func (o *OpenAI) InvokeStructured(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	if req.ResponseFormat == models.ResponseFormatText {
		return o.Invoke(ctx, req)
	}

	chatReq := o.buildChatRequest(req)

	switch req.ResponseFormat {
	case models.ResponseFormatJSONObject:
		chatReq.ResponseFormat = &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatTypeJSONObject}
	case models.ResponseFormatJSONSchema:
		if req.Schema != nil {
			schemaJSON, _ := json.Marshal(req.Schema)
			chatReq.ResponseFormat = &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
				JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
					Name:   "structured_output",
					Schema: json.RawMessage(schemaJSON),
					Strict: true,
				},
			}
		}
	}

	resp, err := o.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, types.NewAPIError("failed to call OpenAI API (structured)", err)
	}
	if len(resp.Choices) == 0 {
		return nil, types.NewAPIError("no response from OpenAI (structured)", nil)
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

	if len(choice.Message.ToolCalls) > 0 {
		modelResp.ToolCalls = make([]types.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			modelResp.ToolCalls[i] = types.ToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
				Function: types.ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	return modelResp, nil
}
