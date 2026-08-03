package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// InvokeStructured invokes the model with Ollama's native `format: json`
// mode, which constrains the output to a single JSON object.
// InvokeStructured 使用 Ollama 原生的 `format: json` 模式调用模型，
// 该模式将输出约束为单个 JSON 对象。
func (o *Ollama) InvokeStructured(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	if req.ResponseFormat == models.ResponseFormatText {
		return o.Invoke(ctx, req)
	}

	ollamaReq := o.buildOllamaRequest(req)
	ollamaReq.Format = "json"

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, types.NewAPIError("failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.config.BaseURL+"/api/chat", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, types.NewAPIError("failed to create HTTP request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return nil, types.NewAPIError("failed to call Ollama API", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, types.NewAPIError(fmt.Sprintf("API error: %s", string(body)), nil)
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, types.NewAPIError("failed to decode response", err)
	}

	return o.convertResponse(&ollamaResp), nil
}
