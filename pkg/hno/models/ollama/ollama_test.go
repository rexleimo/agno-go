package ollama

import (
	"context"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		config  Config
		wantErr bool
	}{
		{
			name:    "default config",
			modelID: "llama2",
			config:  Config{},
			wantErr: false,
		},
		{
			name:    "custom base URL",
			modelID: "mistral",
			config: Config{
				BaseURL: "http://custom:11434",
			},
			wantErr: false,
		},
		{
			name:    "with options",
			modelID: "codellama",
			config: Config{
				BaseURL:     "http://localhost:11434",
				Temperature: 0.8,
				MaxTokens:   4096,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := New(tt.modelID, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if model.GetID() != tt.modelID {
					t.Errorf("GetID() = %v, want %v", model.GetID(), tt.modelID)
				}
				if model.GetProvider() != "ollama" {
					t.Errorf("GetProvider() = %v, want ollama", model.GetProvider())
				}
			}
		})
	}
}

func TestBuildOllamaRequest(t *testing.T) {
	model, _ := New("llama2", Config{
		Temperature: 0.7,
		MaxTokens:   1000,
	})

	tests := []struct {
		name string
		req  *models.InvokeRequest
		want int // expected number of messages
	}{
		{
			name: "basic messages",
			req: &models.InvokeRequest{
				Messages: []*types.Message{
					{Role: types.RoleSystem, Content: "You are helpful"},
					{Role: types.RoleUser, Content: "Hello"},
				},
			},
			want: 2,
		},
		{
			name: "with tools",
			req: &models.InvokeRequest{
				Messages: []*types.Message{
					{Role: types.RoleUser, Content: "Calculate 2+2"},
				},
				Tools: []models.ToolDefinition{
					{
						Type: "function",
						Function: models.FunctionSchema{
							Name:        "calculator",
							Description: "Perform calculations",
							Parameters: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"operation": map[string]interface{}{
										"type": "string",
									},
								},
							},
						},
					},
				},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ollamaReq := model.buildOllamaRequest(tt.req)
			if len(ollamaReq.Messages) != tt.want {
				t.Errorf("buildOllamaRequest() messages = %v, want %v", len(ollamaReq.Messages), tt.want)
			}
			if tt.req.Tools != nil && len(ollamaReq.Tools) != len(tt.req.Tools) {
				t.Errorf("buildOllamaRequest() tools = %v, want %v", len(ollamaReq.Tools), len(tt.req.Tools))
			}
		})
	}
}

func TestConvertResponse(t *testing.T) {
	model, _ := New("llama2", Config{})

	ollamaResp := &OllamaResponse{
		Model: "llama2",
		Message: OllamaMessage{
			Role:    "assistant",
			Content: "Hello, world!",
		},
		Done:            true,
		DoneReason:      "stop",
		PromptEvalCount: 10,
		EvalCount:       5,
	}

	modelResp := model.convertResponse(ollamaResp)

	if modelResp.Content != "Hello, world!" {
		t.Errorf("Content = %v, want 'Hello, world!'", modelResp.Content)
	}
	if modelResp.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %v, want 15", modelResp.Usage.TotalTokens)
	}
}

func TestConvertResponseWithToolCalls(t *testing.T) {
	model, _ := New("llama2", Config{})

	ollamaResp := &OllamaResponse{
		Model: "llama2",
		Message: OllamaMessage{
			Role:    "assistant",
			Content: "I'll calculate that.",
			ToolCalls: []OllamaToolCall{
				{
					Function: OllamaFunctionCall{
						Name: "calculator",
						Arguments: map[string]interface{}{
							"operation": "add",
							"a":         2,
							"b":         2,
						},
					},
				},
			},
		},
		Done:            true,
		PromptEvalCount: 15,
		EvalCount:       20,
	}

	modelResp := model.convertResponse(ollamaResp)

	if len(modelResp.ToolCalls) != 1 {
		t.Errorf("ToolCalls length = %v, want 1", len(modelResp.ToolCalls))
	}
	if modelResp.ToolCalls[0].Function.Name != "calculator" {
		t.Errorf("ToolCall name = %v, want 'calculator'", modelResp.ToolCalls[0].Function.Name)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  Config{},
			wantErr: false,
		},
		{
			name: "with options",
			config: Config{
				BaseURL:     "http://localhost:11434",
				Temperature: 0.7,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildOllamaRequest_WithOptions(t *testing.T) {
	model, _ := New("llama2", Config{
		Temperature: 0.5,
		MaxTokens:   1000,
	})

	tests := []struct {
		name       string
		req        *models.InvokeRequest
		wantTemp   bool
		wantTokens bool
	}{
		{
			name: "use request temperature",
			req: &models.InvokeRequest{
				Messages: []*types.Message{
					{Role: types.RoleUser, Content: "Hello"},
				},
				Temperature: 0.9,
			},
			wantTemp: true,
		},
		{
			name: "use config temperature",
			req: &models.InvokeRequest{
				Messages: []*types.Message{
					{Role: types.RoleUser, Content: "Hello"},
				},
			},
			wantTemp: true,
		},
		{
			name: "use request max tokens",
			req: &models.InvokeRequest{
				Messages: []*types.Message{
					{Role: types.RoleUser, Content: "Hello"},
				},
				MaxTokens: 500,
			},
			wantTokens: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ollamaReq := model.buildOllamaRequest(tt.req)
			if tt.wantTemp {
				if _, ok := ollamaReq.Options["temperature"]; !ok {
					t.Error("Expected temperature option to be set")
				}
			}
			if tt.wantTokens {
				if _, ok := ollamaReq.Options["num_predict"]; !ok {
					t.Error("Expected num_predict option to be set")
				}
			}
		})
	}
}

func TestGetProvider(t *testing.T) {
	model, _ := New("llama2", Config{})
	if model.GetProvider() != "ollama" {
		t.Errorf("GetProvider() = %v, want ollama", model.GetProvider())
	}
}

func TestGetID(t *testing.T) {
	modelID := "mistral"
	model, _ := New(modelID, Config{})
	if model.GetID() != modelID {
		t.Errorf("GetID() = %v, want %v", model.GetID(), modelID)
	}
}

func TestConvertResponse_EmptyContent(t *testing.T) {
	model, _ := New("llama2", Config{})

	ollamaResp := &OllamaResponse{
		Model: "llama2",
		Message: OllamaMessage{
			Role:    "assistant",
			Content: "",
		},
		Done:            true,
		PromptEvalCount: 5,
		EvalCount:       0,
	}

	modelResp := model.convertResponse(ollamaResp)

	if modelResp.Content != "" {
		t.Errorf("Content = %v, want empty string", modelResp.Content)
	}
	if len(modelResp.ToolCalls) != 0 {
		t.Errorf("ToolCalls length = %v, want 0", len(modelResp.ToolCalls))
	}
}

func TestConvertResponse_MultipleToolCalls(t *testing.T) {
	model, _ := New("llama2", Config{})

	ollamaResp := &OllamaResponse{
		Model: "llama2",
		Message: OllamaMessage{
			Role:    "assistant",
			Content: "Using multiple tools",
			ToolCalls: []OllamaToolCall{
				{
					Function: OllamaFunctionCall{
						Name: "tool1",
						Arguments: map[string]interface{}{
							"arg": "value1",
						},
					},
				},
				{
					Function: OllamaFunctionCall{
						Name: "tool2",
						Arguments: map[string]interface{}{
							"arg": "value2",
						},
					},
				},
			},
		},
		Done:            true,
		PromptEvalCount: 10,
		EvalCount:       20,
	}

	modelResp := model.convertResponse(ollamaResp)

	if len(modelResp.ToolCalls) != 2 {
		t.Errorf("ToolCalls length = %v, want 2", len(modelResp.ToolCalls))
	}
	if modelResp.ToolCalls[0].Function.Name != "tool1" {
		t.Errorf("First tool name = %v, want tool1", modelResp.ToolCalls[0].Function.Name)
	}
	if modelResp.ToolCalls[1].Function.Name != "tool2" {
		t.Errorf("Second tool name = %v, want tool2", modelResp.ToolCalls[1].Function.Name)
	}
}

func TestBuildOllamaRequest_AllMessageRoles(t *testing.T) {
	model, _ := New("llama2", Config{})

	req := &models.InvokeRequest{
		Messages: []*types.Message{
			{Role: types.RoleSystem, Content: "You are helpful"},
			{Role: types.RoleUser, Content: "Hello"},
			{Role: types.RoleAssistant, Content: "Hi there"},
			{Role: types.RoleTool, Content: "result", ToolCallID: "call_123"},
		},
	}

	ollamaReq := model.buildOllamaRequest(req)

	if len(ollamaReq.Messages) != 4 {
		t.Errorf("Expected 4 messages, got %d", len(ollamaReq.Messages))
	}

	expectedRoles := []string{"system", "user", "assistant", "tool"}
	for i, expectedRole := range expectedRoles {
		if ollamaReq.Messages[i].Role != expectedRole {
			t.Errorf("Message %d role = %v, want %v", i, ollamaReq.Messages[i].Role, expectedRole)
		}
	}
}

func TestDefaultMaxTokens(t *testing.T) {
	model, _ := New("llama2", Config{})

	// Check that default max tokens is set
	if model.config.MaxTokens != 2048 {
		t.Errorf("Default MaxTokens = %v, want 2048", model.config.MaxTokens)
	}
}

func TestCustomBaseURL(t *testing.T) {
	customURL := "http://custom:11434"
	model, _ := New("llama2", Config{
		BaseURL: customURL,
	})

	if model.config.BaseURL != customURL {
		t.Errorf("BaseURL = %v, want %v", model.config.BaseURL, customURL)
	}
}

func TestDefaultBaseURL(t *testing.T) {
	model, _ := New("llama2", Config{})

	if model.config.BaseURL != defaultBaseURL {
		t.Errorf("BaseURL = %v, want %v", model.config.BaseURL, defaultBaseURL)
	}
}

// Integration test - only runs if Ollama is running locally
func TestInvoke_Integration(t *testing.T) {
	t.Skip("Skipping integration test: requires Ollama running locally")

	model, err := New("llama2", Config{
		BaseURL:     "http://localhost:11434",
		Temperature: 0.7,
		MaxTokens:   100,
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	req := &models.InvokeRequest{
		Messages: []*types.Message{
			{Role: types.RoleUser, Content: "Say hello in one word"},
		},
	}

	resp, err := model.Invoke(context.Background(), req)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if resp.Content == "" {
		t.Error("Response content is empty")
	}

	t.Logf("Response: %s", resp.Content)
	t.Logf("Tokens: %d", resp.Usage.TotalTokens)
}
