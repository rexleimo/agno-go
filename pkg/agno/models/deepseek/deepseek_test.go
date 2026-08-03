package deepseek

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rexleimo/agno-go/pkg/agno/models"
	"github.com/rexleimo/agno-go/pkg/agno/types"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid config",
			modelID: "deepseek-chat",
			config: Config{
				APIKey: "test-key",
			},
			wantErr: false,
		},
		{
			name:    "missing API key",
			modelID: "deepseek-chat",
			config:  Config{},
			wantErr: true,
		},
		{
			name:    "custom base URL",
			modelID: "deepseek-reasoner",
			config: Config{
				APIKey:  "test-key",
				BaseURL: "https://custom.api.com",
			},
			wantErr: false,
		},
		{
			name:    "with temperature and max tokens",
			modelID: "deepseek-chat",
			config: Config{
				APIKey:      "test-key",
				Temperature: 0.7,
				MaxTokens:   2000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := New(tt.modelID, tt.config)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if model.GetID() != tt.modelID {
				t.Errorf("expected model ID %s, got %s", tt.modelID, model.GetID())
			}

			if model.GetProvider() != "deepseek" {
				t.Errorf("expected provider 'deepseek', got %s", model.GetProvider())
			}
		})
	}
}

func TestInvoke(t *testing.T) {
	tests := []struct {
		name           string
		modelID        string
		request        *models.InvokeRequest
		serverResponse map[string]interface{}
		wantContent    string
		wantErr        bool
	}{
		{
			name:    "simple text response",
			modelID: "deepseek-chat",
			request: &models.InvokeRequest{
				Messages: []*types.Message{
					{Role: types.RoleUser, Content: "Hello"},
				},
			},
			serverResponse: map[string]interface{}{
				"id":      "chatcmpl-123",
				"object":  "chat.completion",
				"created": 1677652288,
				"model":   "deepseek-chat",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "Hello! How can I help you today?",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     5,
					"completion_tokens": 10,
					"total_tokens":      15,
				},
			},
			wantContent: "Hello! How can I help you today?",
			wantErr:     false,
		},
		{
			name:    "with system message",
			modelID: "deepseek-chat",
			request: &models.InvokeRequest{
				Messages: []*types.Message{
					{Role: types.RoleSystem, Content: "You are a helpful assistant"},
					{Role: types.RoleUser, Content: "Hi"},
				},
			},
			serverResponse: map[string]interface{}{
				"id":      "chatcmpl-456",
				"object":  "chat.completion",
				"created": 1677652288,
				"model":   "deepseek-chat",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "Hi there! How can I assist you?",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 8,
					"total_tokens":      18,
				},
			},
			wantContent: "Hi there! How can I assist you?",
			wantErr:     false,
		},
		{
			name:    "with function calling",
			modelID: "deepseek-chat",
			request: &models.InvokeRequest{
				Messages: []*types.Message{
					{Role: types.RoleUser, Content: "What's the weather?"},
				},
				Tools: []models.ToolDefinition{
					{
						Type: "function",
						Function: models.FunctionSchema{
							Name:        "get_weather",
							Description: "Get weather information",
							Parameters: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"location": map[string]interface{}{
										"type": "string",
									},
								},
							},
						},
					},
				},
			},
			serverResponse: map[string]interface{}{
				"id":      "chatcmpl-789",
				"object":  "chat.completion",
				"created": 1677652288,
				"model":   "deepseek-chat",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "",
							"tool_calls": []map[string]interface{}{
								{
									"id":   "call_123",
									"type": "function",
									"function": map[string]interface{}{
										"name":      "get_weather",
										"arguments": `{"location":"San Francisco"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     20,
					"completion_tokens": 15,
					"total_tokens":      35,
				},
			},
			wantContent: "",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				// Verify auth header
				authHeader := r.Header.Get("Authorization")
				if authHeader != "Bearer test-key" {
					t.Errorf("expected Authorization header 'Bearer test-key', got %s", authHeader)
				}

				// Send response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			// Create DeepSeek model with test server URL
			model, err := New(tt.modelID, Config{
				APIKey:  "test-key",
				BaseURL: server.URL,
			})
			if err != nil {
				t.Fatalf("failed to create model: %v", err)
			}

			// Invoke model
			resp, err := model.Invoke(context.Background(), tt.request)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Content != tt.wantContent {
				t.Errorf("expected content %q, got %q", tt.wantContent, resp.Content)
			}

			// Verify usage
			usage := tt.serverResponse["usage"].(map[string]interface{})
			if resp.Usage.TotalTokens != int(usage["total_tokens"].(int)) {
				t.Errorf("expected total tokens %d, got %d",
					int(usage["total_tokens"].(int)),
					resp.Usage.TotalTokens)
			}

			// Verify tool calls if present
			if len(tt.request.Tools) > 0 && tt.wantContent == "" {
				if len(resp.ToolCalls) == 0 {
					t.Error("expected tool calls, got none")
				}
			}
		})
	}
}

func TestInvokeError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "400 bad request",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "401 unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "429 rate limit",
			statusCode: http.StatusTooManyRequests,
			wantErr:    true,
		},
		{
			name:       "500 server error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"message": "test error",
						"type":    "invalid_request_error",
					},
				})
			}))
			defer server.Close()

			model, err := New("deepseek-chat", Config{
				APIKey:  "test-key",
				BaseURL: server.URL,
			})
			if err != nil {
				t.Fatalf("failed to create model: %v", err)
			}

			req := &models.InvokeRequest{
				Messages: []*types.Message{
					{Role: types.RoleUser, Content: "test"},
				},
			}

			_, err = model.Invoke(context.Background(), req)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestInvokeStream(t *testing.T) {
	modelID := "deepseek-chat"
	expectedChunks := []string{"Hello", " there", "!"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("streaming not supported")
		}

		for _, content := range expectedChunks {
			chunk := map[string]interface{}{
				"id":      "chatcmpl-123",
				"object":  "chat.completion.chunk",
				"created": 1677652288,
				"model":   "deepseek-chat",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"delta": map[string]interface{}{
							"content": content,
						},
					},
				},
			}

			data, _ := json.Marshal(chunk)
			w.Write([]byte("data: "))
			w.Write(data)
			w.Write([]byte("\n\n"))
			flusher.Flush()
		}

		w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	model, err := New(modelID, Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	req := &models.InvokeRequest{
		Messages: []*types.Message{
			{Role: types.RoleUser, Content: "test"},
		},
	}

	chunks, err := model.InvokeStream(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var receivedContent []string
	for chunk := range chunks {
		if chunk.Error != nil && chunk.Error.Error() != "EOF" {
			t.Fatalf("chunk error: %v", chunk.Error)
		}
		if chunk.Content != "" {
			receivedContent = append(receivedContent, chunk.Content)
		}
		if chunk.Done {
			break
		}
	}

	if len(receivedContent) < 1 {
		t.Errorf("expected at least 1 chunk, got %d", len(receivedContent))
	}
}
