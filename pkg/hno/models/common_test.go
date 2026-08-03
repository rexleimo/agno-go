package models

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Fatal("NewHTTPClient() returned nil")
	}
	if client.client == nil {
		t.Error("HTTPClient.client is nil")
	}
}

func TestPostJSON(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type: application/json")
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("Expected Authorization header")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := NewHTTPClient()
	headers := map[string]string{
		"Authorization": "Bearer test-key",
	}
	body := map[string]string{"test": "data"}

	resp, err := client.PostJSON(context.Background(), server.URL, headers, body)
	if err != nil {
		t.Fatalf("PostJSON() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestPostJSON_InvalidBody(t *testing.T) {
	client := NewHTTPClient()

	// Try to marshal an invalid value (chan cannot be marshaled)
	invalidBody := make(chan int)

	_, err := client.PostJSON(context.Background(), "http://example.com", nil, invalidBody)
	if err == nil {
		t.Error("Expected error for invalid body, got nil")
	}
}

func TestReadJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success", "value": 42}`))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get response: %v", err)
	}

	var result map[string]interface{}
	err = ReadJSONResponse(resp, &result)
	if err != nil {
		t.Fatalf("ReadJSONResponse() error = %v", err)
	}

	if result["message"] != "success" {
		t.Errorf("message = %v, want 'success'", result["message"])
	}

	if result["value"] != float64(42) {
		t.Errorf("value = %v, want 42", result["value"])
	}
}

func TestReadJSONResponse_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get response: %v", err)
	}

	var result map[string]interface{}
	err = ReadJSONResponse(resp, &result)
	if err == nil {
		t.Error("Expected error for non-200 status, got nil")
	}
}

func TestReadJSONResponse_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get response: %v", err)
	}

	var result map[string]interface{}
	err = ReadJSONResponse(resp, &result)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestReadErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get response: %v", err)
	}

	err = ReadErrorResponse(resp)
	if err == nil {
		t.Error("Expected error from ReadErrorResponse, got nil")
	}
}

func TestConvertMessages(t *testing.T) {
	messages := []*types.Message{
		{Role: types.RoleUser, Content: "Hello"},
		{Role: types.RoleAssistant, Content: "Hi there!"},
	}

	result := ConvertMessages(messages)

	if len(result) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(result))
	}

	if result[0]["role"] != string(types.RoleUser) {
		t.Errorf("First message role = %v, want %v", result[0]["role"], types.RoleUser)
	}

	if result[0]["content"] != "Hello" {
		t.Errorf("First message content = %v, want 'Hello'", result[0]["content"])
	}

	if result[1]["role"] != string(types.RoleAssistant) {
		t.Errorf("Second message role = %v, want %v", result[1]["role"], types.RoleAssistant)
	}

	if result[1]["content"] != "Hi there!" {
		t.Errorf("Second message content = %v, want 'Hi there!'", result[1]["content"])
	}
}

func TestConvertMessages_Empty(t *testing.T) {
	result := ConvertMessages([]*types.Message{})
	if len(result) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(result))
	}
}

func TestMergeConfig(t *testing.T) {
	tests := []struct {
		name        string
		reqTemp     float64
		modelTemp   float64
		reqTokens   int
		modelTokens int
		wantTemp    float64
		wantTokens  int
	}{
		{
			name:        "request overrides model",
			reqTemp:     0.8,
			modelTemp:   0.5,
			reqTokens:   500,
			modelTokens: 1000,
			wantTemp:    0.8,
			wantTokens:  500,
		},
		{
			name:        "model defaults when request is zero",
			reqTemp:     0,
			modelTemp:   0.7,
			reqTokens:   0,
			modelTokens: 2000,
			wantTemp:    0.7,
			wantTokens:  2000,
		},
		{
			name:        "mixed override",
			reqTemp:     0.9,
			modelTemp:   0.5,
			reqTokens:   0,
			modelTokens: 1500,
			wantTemp:    0.9,
			wantTokens:  1500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTemp, gotTokens := MergeConfig(tt.reqTemp, tt.modelTemp, tt.reqTokens, tt.modelTokens)
			if gotTemp != tt.wantTemp {
				t.Errorf("temperature = %v, want %v", gotTemp, tt.wantTemp)
			}
			if gotTokens != tt.wantTokens {
				t.Errorf("maxTokens = %v, want %v", gotTokens, tt.wantTokens)
			}
		})
	}
}

func TestBuildToolDefinitions(t *testing.T) {
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: FunctionSchema{
				Name:        "test_func",
				Description: "A test function",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{
							"type":        "string",
							"description": "First parameter",
						},
					},
				},
			},
		},
	}

	result := BuildToolDefinitions(tools)

	if len(result) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(result))
	}

	if result[0]["type"] != "function" {
		t.Errorf("type = %v, want 'function'", result[0]["type"])
	}

	fn := result[0]["function"].(map[string]interface{})
	if fn["name"] != "test_func" {
		t.Errorf("name = %v, want 'test_func'", fn["name"])
	}

	if fn["description"] != "A test function" {
		t.Errorf("description = %v, want 'A test function'", fn["description"])
	}
}

func TestBuildToolDefinitions_Empty(t *testing.T) {
	result := BuildToolDefinitions(nil)
	if result != nil {
		t.Errorf("Expected nil for empty tools, got %v", result)
	}

	result = BuildToolDefinitions([]ToolDefinition{})
	if result != nil {
		t.Errorf("Expected nil for empty tools slice, got %v", result)
	}
}
