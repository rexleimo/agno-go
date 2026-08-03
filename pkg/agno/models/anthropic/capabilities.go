package anthropic

import (
	"context"
	"encoding/json"

	"github.com/rexleimo/agno-go/pkg/agno/models"
	"github.com/rexleimo/agno-go/pkg/agno/types"
)

// InvokeStructured invokes the model with structured-output instructions.
// Anthropic's Messages API does not expose a response_format parameter in this
// adapter, so the constraint is injected as a system instruction; the raw
// response is returned as-is for the caller to parse.
// InvokeStructured 以结构化输出指令调用模型。
// 本适配器中 Anthropic Messages API 不暴露 response_format 参数，
// 因此约束以系统指令形式注入；原始响应原样返回，由调用方解析。
func (a *Anthropic) InvokeStructured(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	if req.ResponseFormat == models.ResponseFormatText {
		return a.Invoke(ctx, req)
	}

	cloned := *req
	cloned.Messages = cloneMessagesForStructured(req.Messages)

	schemaJSON := "{}"
	if req.Schema != nil {
		if b, err := json.Marshal(req.Schema); err == nil {
			schemaJSON = string(b)
		}
	}

	instruction := "You MUST respond with a single valid JSON object only. "
	switch req.ResponseFormat {
	case models.ResponseFormatJSONObject:
		instruction += "Do not include any text outside the JSON object."
	case models.ResponseFormatJSONSchema:
		instruction += "The JSON object MUST conform to this JSON schema: " + schemaJSON
	}

	systemMsg := types.NewSystemMessage(instruction)
	cloned.Messages = append([]*types.Message{systemMsg}, cloned.Messages...)

	return a.Invoke(ctx, &cloned)
}

// cloneMessagesForStructured returns a shallow copy of the message slice so the
// caller's original messages are never mutated.
// cloneMessagesForStructured 返回消息切片的浅拷贝，确保调用方的原始消息不被修改。
func cloneMessagesForStructured(messages []*types.Message) []*types.Message {
	if len(messages) == 0 {
		return nil
	}
	out := make([]*types.Message, len(messages))
	copy(out, messages)
	return out
}
