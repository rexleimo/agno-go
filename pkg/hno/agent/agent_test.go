package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/cache"
	"github.com/rexleimo/agno-go/pkg/hno/memory"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/tools/calculator"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// MockModel is a simple mock for testing
type MockModel struct {
	models.BaseModel
	InvokeFunc       func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error)
	InvokeStreamFunc func(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error)
}

func (m *MockModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	if m.InvokeFunc != nil {
		return m.InvokeFunc(ctx, req)
	}
	return &types.ModelResponse{
		ID:      "test-response",
		Content: "Mock response",
		Model:   "mock-model",
	}, nil
}

func (m *MockModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	if m.InvokeStreamFunc != nil {
		return m.InvokeStreamFunc(ctx, req)
	}
	ch := make(chan types.ResponseChunk)
	close(ch)
	return ch, nil
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				Name:  "TestAgent",
				Model: &MockModel{BaseModel: models.BaseModel{ID: "test", Provider: "mock"}},
			},
			wantErr: false,
		},
		{
			name: "missing model",
			config: Config{
				Name: "TestAgent",
			},
			wantErr: true,
			errMsg:  "model is required",
		},
		{
			name: "with default values",
			config: Config{
				Model: &MockModel{BaseModel: models.BaseModel{ID: "test", Provider: "mock"}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if agent == nil {
					t.Error("New() returned nil agent")
					return
				}
				// Check defaults
				if agent.MaxLoops <= 0 {
					t.Error("MaxLoops should have default value > 0")
				}
				if agent.Memory == nil {
					t.Error("Memory should be initialized")
				}
			}
		})
	}
}

func TestAgent_Run_SimpleResponse(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return &types.ModelResponse{
				ID:      "test-1",
				Content: "Hello, this is a test response",
				Model:   "test",
			}, nil
		},
	}

	agent, err := New(Config{
		Name:  "TestAgent",
		Model: mockModel,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	output, err := agent.Run(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if output.Content != "Hello, this is a test response" {
		t.Errorf("Run() content = %v, want %v", output.Content, "Hello, this is a test response")
	}

	if len(output.Messages) < 2 {
		t.Errorf("Run() should have at least 2 messages (user + assistant)")
	}

	if output.Status != RunStatusCompleted {
		t.Fatalf("expected run to be completed, got %s", output.Status)
	}

	if output.RunID == "" {
		t.Fatalf("expected run id to be set")
	}
}

func TestAgent_RunStream_Basic(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeStreamFunc: func(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
			ch := make(chan types.ResponseChunk, 2)
			ch <- types.ResponseChunk{Content: "Hello"}
			ch <- types.ResponseChunk{Content: " world"}
			close(ch)
			return ch, nil
		},
	}

	ag, err := New(Config{
		Name:  "StreamAgent",
		Model: mockModel,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	result, err := ag.RunStream(context.Background(), "Hi")
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if result == nil {
		t.Fatal("RunStream() returned nil result")
	}

	var chunks []string
	for evt := range result.Events {
		contentEvt, ok := evt.(*run.RunContentEvent)
		if !ok {
			t.Fatalf("expected RunContentEvent, got %T", evt)
		}
		chunks = append(chunks, contentEvt.Content)
	}

	done := <-result.Done
	if done.Err != nil {
		t.Fatalf("RunStream() Done error = %v", done.Err)
	}
	if done.Output == nil {
		t.Fatal("RunStream() Done output is nil")
	}

	if want := "Hello world"; done.Output.Content != want {
		t.Errorf("RunStream() content = %q, want %q", done.Output.Content, want)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 streamed chunks, got %d", len(chunks))
	}
	if len(done.Output.Events) != 3 {
		t.Fatalf("expected 3 events (2 content + 1 completed), got %d", len(done.Output.Events))
	}
}

func TestAgent_Run_EmitsEvents(t *testing.T) {
	agent, err := New(Config{
		Name: "events-agent",
		Model: &MockModel{
			BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
			InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
				return &types.ModelResponse{ID: "evt", Content: "event payload", Model: "mock"}, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	output, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(output.Events) < 1 {
		t.Fatalf("expected events to be recorded")
	}
	contentEvent, ok := output.Events[0].(*run.RunContentEvent)
	if !ok {
		t.Fatalf("expected first event to be RunContentEvent, got %T", output.Events[0])
	}
	if contentEvent.Content != "event payload" {
		t.Fatalf("unexpected event content %q", contentEvent.Content)
	}
	completed := output.Events[len(output.Events)-1]
	if completed.EventType() != "run_completed" {
		t.Fatalf("expected terminal event to be run_completed, got %s", completed.EventType())
	}
}

func TestAgent_Run_EmptyInput(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
	}

	agent, err := New(Config{
		Name:  "TestAgent",
		Model: mockModel,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	_, err = agent.Run(context.Background(), "")
	if err == nil {
		t.Error("Run() should return error for empty input")
	}
}

func TestAgent_Run_WithToolCalls(t *testing.T) {
	callCount := 0
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			callCount++

			// First call: return tool call
			if callCount == 1 {
				return &types.ModelResponse{
					ID:    "test-1",
					Model: "test",
					ToolCalls: []types.ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: types.ToolCallFunction{
								Name:      "add",
								Arguments: `{"a": 5, "b": 3}`,
							},
						},
					},
				}, nil
			}

			// Second call: return final answer
			return &types.ModelResponse{
				ID:      "test-2",
				Content: "The result is 8",
				Model:   "test",
			}, nil
		},
	}

	agent, err := New(Config{
		Name:     "TestAgent",
		Model:    mockModel,
		Toolkits: []toolkit.Toolkit{calculator.New()},
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	output, err := agent.Run(context.Background(), "What is 5 + 3?")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 model calls (tool call + final), got %d", callCount)
	}

	if output.Content != "The result is 8" {
		t.Errorf("Run() content = %v, want %v", output.Content, "The result is 8")
	}

	if output.Status != RunStatusCompleted {
		t.Fatalf("expected run status completed, got %s", output.Status)
	}

	// Check metadata
	loops, ok := output.Metadata["loops"].(int)
	if !ok || loops != 2 {
		t.Errorf("Run() loops = %v, want 2", loops)
	}
}

func TestAgent_Run_UsesCache(t *testing.T) {
	provider, err := cache.NewMemoryProvider(8, time.Minute)
	if err != nil {
		t.Fatalf("NewMemoryProvider error = %v", err)
	}

	callCount := 0
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			callCount++
			return &types.ModelResponse{
				ID:      "cached-response",
				Content: "Cached result",
				Model:   "test",
			}, nil
		},
	}

	agent, err := New(Config{
		Name:          "CacheAgent",
		Model:         mockModel,
		EnableCache:   true,
		CacheProvider: provider,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	first, err := agent.Run(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 provider call, got %d", callCount)
	}
	if hit, _ := first.Metadata["cache_hit"].(bool); hit {
		t.Fatalf("first run should not be cache hit")
	}

	agent.ClearMemory()
	second, err := agent.Run(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected cached run without provider call, got %d", callCount)
	}
	if hit, _ := second.Metadata["cache_hit"].(bool); !hit {
		t.Fatalf("expected cache hit on second run")
	}

	if second.Content != "Cached result" {
		t.Fatalf("unexpected content: %v", second.Content)
	}

	if second.Status != RunStatusCompleted {
		t.Fatalf("expected cached run to be completed")
	}
}

func TestAgent_Run_ContextCancelled(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return nil, context.Canceled
		},
	}

	agent, err := New(Config{
		Name:  "CancelAgent",
		Model: mockModel,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	output, runErr := agent.Run(ctx, "Hello")
	if runErr == nil {
		t.Fatalf("expected cancellation error")
	}

	agnoErr, ok := runErr.(*types.HnoError)
	if !ok || agnoErr.Code != types.ErrCodeCancelled {
		t.Fatalf("expected cancellation error code, got %#v", runErr)
	}

	if output == nil {
		t.Fatalf("expected run output even on cancellation")
	}
	if output.Status != RunStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", output.Status)
	}
	if output.CancellationReason == "" {
		t.Fatalf("expected cancellation reason to be populated")
	}
}

func TestAgent_Run_MaxLoops(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			// Always return tool calls to trigger max loops
			return &types.ModelResponse{
				ID:    "test",
				Model: "test",
				ToolCalls: []types.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: types.ToolCallFunction{
							Name:      "add",
							Arguments: `{"a": 1, "b": 1}`,
						},
					},
				},
			}, nil
		},
	}

	agent, err := New(Config{
		Name:     "TestAgent",
		Model:    mockModel,
		Toolkits: []toolkit.Toolkit{calculator.New()},
		MaxLoops: 3,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	_, err = agent.Run(context.Background(), "Test")
	if err == nil {
		t.Error("Run() should return error when max loops reached")
	}
}

func TestAgent_ClearMemory(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
	}

	agent, err := New(Config{
		Name:         "TestAgent",
		Model:        mockModel,
		Instructions: "You are a helpful assistant",
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Add some messages
	agent.Memory.Add(types.NewUserMessage("Hello"))
	agent.Memory.Add(types.NewAssistantMessage("Hi there"))

	if len(agent.Memory.GetMessages()) < 3 { // system + user + assistant
		t.Error("Should have at least 3 messages")
	}

	// Clear memory
	agent.ClearMemory()

	messages := agent.Memory.GetMessages()
	if len(messages) != 1 {
		t.Errorf("After clear, should have 1 message (system), got %d", len(messages))
	}

	if messages[0].Role != types.RoleSystem {
		t.Error("First message after clear should be system message")
	}
}

func TestAgent_WithCustomMemory(t *testing.T) {
	mockModel := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
	}

	customMemory := memory.NewInMemory(5)
	customMemory.Add(types.NewUserMessage("Previous message"))

	agent, err := New(Config{
		Name:   "TestAgent",
		Model:  mockModel,
		Memory: customMemory,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	messages := agent.Memory.GetMessages()
	if len(messages) != 1 {
		t.Errorf("Should preserve custom memory, got %d messages", len(messages))
	}
}

// TestAgent_MultiTenant tests multi-tenant memory isolation
// 测试多租户内存隔离
func TestAgent_MultiTenant(t *testing.T) {
	model := &MockModel{
		BaseModel: models.BaseModel{ID: "test", Provider: "mock"},
		InvokeFunc: func(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			// Echo back the number of messages in memory
			return &types.ModelResponse{
				ID:      "test-response",
				Content: fmt.Sprintf("I see %d messages in history", len(req.Messages)),
				Model:   "test",
			}, nil
		},
	}

	// Create two agents with same memory but different userIDs
	sharedMemory := memory.NewInMemory(100)

	agent1, _ := New(Config{
		ID:     "agent1",
		Name:   "Agent for User 1",
		Model:  model,
		UserID: "user1",
		Memory: sharedMemory,
	})

	agent2, _ := New(Config{
		ID:     "agent2",
		Name:   "Agent for User 2",
		Model:  model,
		UserID: "user2",
		Memory: sharedMemory,
	})

	// User 1 sends first message
	output1, err := agent1.Run(context.Background(), "Hello from user1")
	if err != nil {
		t.Fatalf("User1 run failed: %v", err)
	}

	// When Run() is called, it gets messages BEFORE adding the assistant response
	// So model sees: [user message] = 1 message
	if !strings.Contains(output1.Content, "1 messages") {
		t.Errorf("User1 model should see 1 message, got: %s", output1.Content)
	}

	// User 2 sends first message (should start fresh)
	output2, err := agent2.Run(context.Background(), "Hello from user2")
	if err != nil {
		t.Fatalf("User2 run failed: %v", err)
	}

	// User 2 also sees 1 message (their user message)
	if !strings.Contains(output2.Content, "1 messages") {
		t.Errorf("User2 model should see 1 message in their own context, got: %s", output2.Content)
	}

	// User 1 sends second message
	output1b, err := agent1.Run(context.Background(), "Second message from user1")
	if err != nil {
		t.Fatalf("User1 second run failed: %v", err)
	}

	// User 1 model should see 3 messages: [user1, assistant1, user2]
	if !strings.Contains(output1b.Content, "3 messages") {
		t.Errorf("User1 model should see 3 messages after second interaction, got: %s", output1b.Content)
	}

	// Verify memory isolation: user1 has 4 messages, user2 has 2 messages
	user1Size := sharedMemory.Size("user1")
	user2Size := sharedMemory.Size("user2")

	if user1Size != 4 {
		t.Errorf("User1 should have 4 messages in memory, got %d", user1Size)
	}

	if user2Size != 2 {
		t.Errorf("User2 should have 2 messages in memory, got %d", user2Size)
	}

	// Clear user1's memory
	agent1.ClearMemory()

	// User1 should start fresh
	if sharedMemory.Size("user1") != 0 {
		t.Errorf("User1 memory should be cleared, got %d messages", sharedMemory.Size("user1"))
	}

	// User2's memory should be unaffected
	if sharedMemory.Size("user2") != 2 {
		t.Errorf("User2 memory should be unaffected, got %d messages", sharedMemory.Size("user2"))
	}
}
