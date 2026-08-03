package ollama

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
	defaultBaseURL = "http://localhost:11434"
)

// Ollama wraps the Ollama local model client
type Ollama struct {
	models.BaseModel
	config     Config
	httpClient *http.Client
}

// Config contains Ollama-specific configuration
type Config struct {
	BaseURL     string
	Temperature float64
	MaxTokens   int
}

// New creates a new Ollama model instance
func New(modelID string, config Config) (*Ollama, error) {
	if config.BaseURL == "" {
		config.BaseURL = defaultBaseURL
	}

	if config.MaxTokens == 0 {
		config.MaxTokens = 2048
	}

	return &Ollama{
		BaseModel: models.BaseModel{
			ID:       modelID,
			Provider: "ollama",
			Name:     modelID,
		},
		config:     config,
		httpClient: &http.Client{},
	}, nil
}

// Invoke calls the Ollama API synchronously
func (o *Ollama) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	ollamaReq := o.buildOllamaRequest(req)

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

// InvokeStream calls the Ollama API with streaming response
func (o *Ollama) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	ollamaReq := o.buildOllamaRequest(req)
	ollamaReq.Stream = true

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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, types.NewAPIError(fmt.Sprintf("API error: %s", string(body)), nil)
	}

	chunks := make(chan types.ResponseChunk)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			var streamResp OllamaResponse
			if err := decoder.Decode(&streamResp); err != nil {
				if err != io.EOF {
					chunks <- types.ResponseChunk{
						Done:  true,
						Error: err,
					}
				}
				return
			}

			chunk := types.ResponseChunk{
				Content: streamResp.Message.Content,
				Done:    streamResp.Done,
			}

			select {
			case chunks <- chunk:
				if streamResp.Done {
					return
				}
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

// buildOllamaRequest converts InvokeRequest to Ollama API request
func (o *Ollama) buildOllamaRequest(req *models.InvokeRequest) *OllamaRequest {
	ollamaReq := &OllamaRequest{
		Model:    o.ID,
		Messages: make([]OllamaMessage, 0),
		Stream:   false,
	}

	// Convert messages
	for _, msg := range req.Messages {
		ollamaMsg := OllamaMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
		ollamaReq.Messages = append(ollamaReq.Messages, ollamaMsg)
	}

	// Set options
	options := make(map[string]interface{})

	if req.Temperature > 0 {
		options["temperature"] = req.Temperature
	} else if o.config.Temperature > 0 {
		options["temperature"] = o.config.Temperature
	}

	if req.MaxTokens > 0 {
		options["num_predict"] = req.MaxTokens
	} else if o.config.MaxTokens > 0 {
		options["num_predict"] = o.config.MaxTokens
	}

	if len(options) > 0 {
		ollamaReq.Options = options
	}

	// Convert tools
	if len(req.Tools) > 0 {
		ollamaReq.Tools = make([]OllamaTool, len(req.Tools))
		for i, tool := range req.Tools {
			ollamaReq.Tools[i] = OllamaTool{
				Type: tool.Type,
				Function: OllamaFunction{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			}
		}
	}

	return ollamaReq
}

// convertResponse converts Ollama response to ModelResponse
func (o *Ollama) convertResponse(resp *OllamaResponse) *types.ModelResponse {
	extra := make(map[string]interface{})
	extra["total_duration"] = resp.TotalDuration
	extra["load_duration"] = resp.LoadDuration
	extra["prompt_eval_count"] = resp.PromptEvalCount
	extra["prompt_eval_duration"] = resp.PromptEvalDuration
	extra["eval_count"] = resp.EvalCount
	extra["eval_duration"] = resp.EvalDuration

	modelResp := &types.ModelResponse{
		Model:   resp.Model,
		Content: resp.Message.Content,
		Usage: types.Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
		Metadata: types.Metadata{
			FinishReason: resp.DoneReason,
			Extra:        extra,
		},
	}

	// Handle tool calls if present
	if len(resp.Message.ToolCalls) > 0 {
		modelResp.ToolCalls = make([]types.ToolCall, len(resp.Message.ToolCalls))
		for i, tc := range resp.Message.ToolCalls {
			modelResp.ToolCalls[i] = types.ToolCall{
				ID:   fmt.Sprintf("call_%d", i),
				Type: "function",
				Function: types.ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: jsonToString(tc.Function.Arguments),
				},
			}
		}
	}

	return modelResp
}

// OllamaRequest represents the Ollama API request
type OllamaRequest struct {
	Model    string                 `json:"model"`
	Messages []OllamaMessage        `json:"messages"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
	Tools    []OllamaTool           `json:"tools,omitempty"`
	Format   string                 `json:"format,omitempty"`
}

// OllamaMessage represents a message in the conversation
type OllamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []OllamaToolCall `json:"tool_calls,omitempty"`
}

// OllamaToolCall represents a tool call
type OllamaToolCall struct {
	Function OllamaFunctionCall `json:"function"`
}

// OllamaFunctionCall represents a function call
type OllamaFunctionCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// OllamaTool represents a tool definition
type OllamaTool struct {
	Type     string         `json:"type"`
	Function OllamaFunction `json:"function"`
}

// OllamaFunction represents a function definition
type OllamaFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// OllamaResponse represents the Ollama API response
type OllamaResponse struct {
	Model              string        `json:"model"`
	CreatedAt          string        `json:"created_at"`
	Message            OllamaMessage `json:"message"`
	Done               bool          `json:"done"`
	DoneReason         string        `json:"done_reason,omitempty"`
	TotalDuration      int64         `json:"total_duration,omitempty"`
	LoadDuration       int64         `json:"load_duration,omitempty"`
	PromptEvalCount    int           `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64         `json:"prompt_eval_duration,omitempty"`
	EvalCount          int           `json:"eval_count,omitempty"`
	EvalDuration       int64         `json:"eval_duration,omitempty"`
}

// Helper function to convert map to JSON string
func jsonToString(data map[string]interface{}) string {
	if data == nil {
		return "{}"
	}
	jsonBytes, _ := json.Marshal(data)
	return string(jsonBytes)
}

// ValidateConfig validates the Ollama configuration
func ValidateConfig(config Config) error {
	// Ollama doesn't require API key, just needs to be running locally
	return nil
}
