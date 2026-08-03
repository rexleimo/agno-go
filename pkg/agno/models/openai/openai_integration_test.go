package openai

import (
	"context"
	"os"
	"testing"

	"github.com/rexleimo/agno-go/pkg/agno/models"
	"github.com/rexleimo/agno-go/pkg/agno/types"
)

// Integration tests that require OPENAI_API_KEY
// These tests increase coverage of Invoke/InvokeStream methods

func TestOpenAI_Invoke_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	// Use a small, fast model for testing
	model, err := New("gpt-3.5-turbo", Config{
		APIKey:      apiKey,
		Temperature: 0.1,
		MaxTokens:   50,
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	tests := []struct {
		name     string
		messages []*types.Message
		wantErr  bool
	}{
		{
			name: "simple user message",
			messages: []*types.Message{
				{Role: types.RoleUser, Content: "Say 'test' and nothing else"},
			},
			wantErr: false,
		},
		{
			name: "with system message",
			messages: []*types.Message{
				{Role: types.RoleSystem, Content: "You are a helpful assistant"},
				{Role: types.RoleUser, Content: "Hello"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := model.Invoke(context.Background(), &models.InvokeRequest{
				Messages: tt.messages,
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("Invoke() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp == nil {
					t.Error("Expected non-nil response")
					return
				}
				if resp.Content == "" {
					t.Error("Expected non-empty content")
				}
				if resp.Model == "" {
					t.Error("Expected non-empty model")
				}
				if resp.Usage.TotalTokens == 0 {
					t.Error("Expected non-zero token usage")
				}
			}
		})
	}
}

func TestOpenAI_InvokeStream_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	model, err := New("gpt-3.5-turbo", Config{
		APIKey:      apiKey,
		Temperature: 0.1,
		MaxTokens:   50,
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	chunks, err := model.InvokeStream(context.Background(), &models.InvokeRequest{
		Messages: []*types.Message{
			{Role: types.RoleUser, Content: "Count from 1 to 3"},
		},
	})

	if err != nil {
		t.Fatalf("InvokeStream() error = %v", err)
	}

	var receivedChunks int
	var totalContent string
	for chunk := range chunks {
		receivedChunks++
		if chunk.Error != nil {
			if chunk.Done {
				// Stream ended, this is expected
				break
			}
			t.Fatalf("Received error chunk: %v", chunk.Error)
		}
		totalContent += chunk.Content
	}

	if receivedChunks == 0 {
		t.Error("Expected to receive at least one chunk")
	}

	if totalContent == "" {
		t.Error("Expected non-empty content from stream")
	}
}

func TestOpenAI_Invoke_WithTools_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	model, err := New("gpt-3.5-turbo", Config{
		APIKey:      apiKey,
		Temperature: 0.1,
		MaxTokens:   100,
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Define a simple tool
	tools := []models.ToolDefinition{
		{
			Type: "function",
			Function: models.FunctionSchema{
				Name:        "get_weather",
				Description: "Get the weather for a location",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "The city name",
						},
					},
					"required": []string{"location"},
				},
			},
		},
	}

	resp, err := model.Invoke(context.Background(), &models.InvokeRequest{
		Messages: []*types.Message{
			{Role: types.RoleUser, Content: "What's the weather in San Francisco?"},
		},
		Tools: tools,
	})

	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	// The model should either respond with text or call the tool
	if resp.Content == "" && len(resp.ToolCalls) == 0 {
		t.Error("Expected either content or tool calls")
	}

	// If tool calls were made, verify structure
	if len(resp.ToolCalls) > 0 {
		tc := resp.ToolCalls[0]
		if tc.Function.Name != "get_weather" {
			t.Errorf("Expected tool call to get_weather, got %s", tc.Function.Name)
		}
		if tc.Function.Arguments == "" {
			t.Error("Expected non-empty arguments")
		}
	}
}

func TestOpenAI_Invoke_ContextCancellation(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	model, err := New("gpt-3.5-turbo", Config{
		APIKey: apiKey,
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = model.Invoke(ctx, &models.InvokeRequest{
		Messages: []*types.Message{
			{Role: types.RoleUser, Content: "Hello"},
		},
	})

	if err == nil {
		t.Error("Expected error from cancelled context")
	}
}

func TestOpenAI_InvokeStream_ContextCancellation(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	model, err := New("gpt-3.5-turbo", Config{
		APIKey: apiKey,
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chunks, err := model.InvokeStream(ctx, &models.InvokeRequest{
		Messages: []*types.Message{
			{Role: types.RoleUser, Content: "Count from 1 to 100"},
		},
	})

	if err != nil {
		t.Fatalf("InvokeStream() error = %v", err)
	}

	// Cancel context after receiving first chunk
	var receivedFirst bool
	for chunk := range chunks {
		if !receivedFirst {
			receivedFirst = true
			cancel()
		}
		if chunk.Error != nil && chunk.Done {
			// Stream ended with error, expected
			break
		}
	}

	if !receivedFirst {
		t.Error("Expected to receive at least one chunk before cancellation")
	}
}

func TestOpenAI_Invoke_EmptyResponse(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	model, err := New("gpt-3.5-turbo", Config{
		APIKey:      apiKey,
		MaxTokens:   1, // Very low token limit
		Temperature: 0,
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	resp, err := model.Invoke(context.Background(), &models.InvokeRequest{
		Messages: []*types.Message{
			{Role: types.RoleUser, Content: "Write a long essay"},
		},
	})

	// Should not error, but response might be very short or truncated
	if err != nil {
		t.Logf("Invoke() with MaxTokens=1 returned error: %v", err)
	}

	if resp != nil && resp.Metadata.FinishReason != "" {
		t.Logf("Finish reason: %s", resp.Metadata.FinishReason)
	}
}

func TestOpenAI_InvokeStream_EmptyChunks(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	model, err := New("gpt-3.5-turbo", Config{
		APIKey:      apiKey,
		Temperature: 0,
		MaxTokens:   5,
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	chunks, err := model.InvokeStream(context.Background(), &models.InvokeRequest{
		Messages: []*types.Message{
			{Role: types.RoleUser, Content: "Hi"},
		},
	})

	if err != nil {
		t.Fatalf("InvokeStream() error = %v", err)
	}

	var chunkCount int
	for chunk := range chunks {
		chunkCount++
		// Some chunks might be empty (structure only)
		if chunk.Error != nil && chunk.Done {
			break
		}
	}

	// Should receive at least one chunk (even if empty)
	if chunkCount == 0 {
		t.Error("Expected to receive at least one chunk")
	}
}
