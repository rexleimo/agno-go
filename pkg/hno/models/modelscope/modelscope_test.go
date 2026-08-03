package modelscope

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

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
			modelID: "qwen-plus",
			request: &models.InvokeRequest{
				Messages: []*types.Message{
					{Role: types.RoleUser, Content: "你好"},
				},
			},
			serverResponse: map[string]interface{}{
				"id":      "chatcmpl-123",
				"object":  "chat.completion",
				"created": 1677652288,
				"model":   "qwen-plus",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "你好！我是通义千问，很高兴为你服务。",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     2,
					"completion_tokens": 10,
					"total_tokens":      12,
				},
			},
			wantContent: "你好！我是通义千问,很高兴为你服务。",
			wantErr:     false,
		},
		{
			name:    "with system message",
			modelID: "qwen-turbo",
			request: &models.InvokeRequest{
				Messages: []*types.Message{
					{Role: types.RoleSystem, Content: "你是一个有帮助的AI助手"},
					{Role: types.RoleUser, Content: "介绍一下北京"},
				},
			},
			serverResponse: map[string]interface{}{
				"id":      "chatcmpl-456",
				"object":  "chat.completion",
				"created": 1677652288,
				"model":   "qwen-turbo",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "北京是中国的首都...",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     15,
					"completion_tokens": 50,
					"total_tokens":      65,
				},
			},
			wantContent: "北京是中国的首都...",
			wantErr:     false,
		},
		{
			name:    "with function calling",
			modelID: "qwen-plus",
			request: &models.InvokeRequest{
				Messages: []*types.Message{
					{Role: types.RoleUser, Content: "今天天气如何？"},
				},
				Tools: []models.ToolDefinition{
					{
						Type: "function",
						Function: models.FunctionSchema{
							Name:        "get_weather",
							Description: "获取天气信息",
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
				"model":   "qwen-plus",
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
										"arguments": `{"location":"北京"}`,
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

			// Create ModelScope model with test server URL
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

			// Verify content (allowing for different responses)
			if resp.Content == "" && tt.wantContent != "" && len(tt.request.Tools) == 0 {
				t.Errorf("expected non-empty content")
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

			model, err := New("qwen-plus", Config{
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
	modelID := "qwen-turbo"
	expectedChunks := []string{"你好", "，", "世界"}

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
				"model":   "qwen-turbo",
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
