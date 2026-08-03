package reasoning

import (
	"context"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// MockModel 用于测试 / MockModel for testing
type MockModel struct {
	id       string
	provider string
}

func (m *MockModel) GetID() string       { return m.id }
func (m *MockModel) GetProvider() string { return m.provider }
func (m *MockModel) GetName() string     { return m.id }
func (m *MockModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	return nil, nil
}
func (m *MockModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	return nil, nil
}

// MockReasoningModel 支持模拟 SupportsReasoning 接口
type MockReasoningModel struct {
	MockModel
	reasoningEnabled bool
}

func (m *MockReasoningModel) SupportsReasoning() bool {
	return m.reasoningEnabled
}

func TestIsReasoningModel_OpenAI(t *testing.T) {
	tests := []struct {
		name     string
		modelID  string
		provider string
		want     bool
	}{
		{
			name:     "OpenAI o1-preview",
			modelID:  "o1-preview",
			provider: "openai",
			want:     true,
		},
		{
			name:     "OpenAI o1-mini",
			modelID:  "o1-mini",
			provider: "openai",
			want:     true,
		},
		{
			name:     "OpenAI o3",
			modelID:  "o3-mini",
			provider: "openai",
			want:     true,
		},
		{
			name:     "OpenAI GPT-4",
			modelID:  "gpt-4",
			provider: "openai",
			want:     false,
		},
		{
			name:     "Non-OpenAI",
			modelID:  "o1-preview",
			provider: "anthropic",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &MockModel{id: tt.modelID, provider: tt.provider}
			got := IsReasoningModel(model)
			if got != tt.want {
				t.Errorf("IsReasoningModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsReasoningModel_Gemini(t *testing.T) {
	tests := []struct {
		name  string
		model models.Model
		want  bool
	}{
		{
			name:  "Gemini 2.5 Flash Thinking",
			model: &MockModel{id: "gemini-2.5-flash-thinking", provider: "gemini"},
			want:  true,
		},
		{
			name:  "Gemini with thinking keyword",
			model: &MockModel{id: "gemini-thinking-exp", provider: "gemini"},
			want:  true,
		},
		{
			name:  "Gemini 2.0",
			model: &MockModel{id: "gemini-2.0-flash", provider: "gemini"},
			want:  false,
		},
		{
			name: "SupportsReasoning interface enabled",
			model: &MockReasoningModel{
				MockModel:        MockModel{id: "gemini-custom", provider: "gemini"},
				reasoningEnabled: true,
			},
			want: true,
		},
		{
			name: "SupportsReasoning interface disabled",
			model: &MockReasoningModel{
				MockModel:        MockModel{id: "gemini-custom", provider: "gemini"},
				reasoningEnabled: false,
			},
			want: false,
		},
		{
			name:  "Non-Gemini",
			model: &MockModel{id: "gemini-2.5-flash-thinking", provider: "openai"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsReasoningModel(tt.model)
			if got != tt.want {
				t.Errorf("IsReasoningModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsReasoningModel_Anthropic(t *testing.T) {
	tests := []struct {
		name  string
		model models.Model
		want  bool
	}{
		{
			name: "Anthropic thinking enabled",
			model: &MockReasoningModel{
				MockModel:        MockModel{id: "claude-3-5-sonnet", provider: "anthropic"},
				reasoningEnabled: true,
			},
			want: true,
		},
		{
			name: "Anthropic thinking disabled",
			model: &MockReasoningModel{
				MockModel:        MockModel{id: "claude-3-5-sonnet", provider: "anthropic"},
				reasoningEnabled: false,
			},
			want: false,
		},
		{
			name:  "Non-Anthropic provider",
			model: &MockReasoningModel{MockModel: MockModel{id: "claude-3-opus", provider: "openai"}, reasoningEnabled: true},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsReasoningModel(tt.model)
			if got != tt.want {
				t.Errorf("IsReasoningModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsReasoningModel_VertexAI(t *testing.T) {
	tests := []struct {
		name  string
		model models.Model
		want  bool
	}{
		{
			name: "VertexAI thinking enabled",
			model: &MockReasoningModel{
				MockModel:        MockModel{id: "claude-3-5-sonnet@20240620", provider: "vertexai"},
				reasoningEnabled: true,
			},
			want: true,
		},
		{
			name: "VertexAI alias provider",
			model: &MockReasoningModel{
				MockModel:        MockModel{id: "claude-3-5-sonnet@20240620", provider: "vertex-ai"},
				reasoningEnabled: true,
			},
			want: true,
		},
		{
			name: "VertexAI thinking disabled",
			model: &MockReasoningModel{
				MockModel:        MockModel{id: "claude-3-5-sonnet@20240620", provider: "vertexai"},
				reasoningEnabled: false,
			},
			want: false,
		},
		{
			name:  "Non-Vertex provider",
			model: &MockReasoningModel{MockModel: MockModel{id: "claude-3-5-sonnet@20240620", provider: "anthropic"}, reasoningEnabled: true},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsReasoningModel(tt.model)
			if got != tt.want {
				t.Errorf("IsReasoningModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractReasoning_OpenAI(t *testing.T) {
	ctx := context.Background()
	model := &MockModel{id: "o1-preview", provider: "openai"}

	tests := []struct {
		name     string
		response *types.ModelResponse
		wantNil  bool
	}{
		{
			name: "with <think> tags",
			response: &types.ModelResponse{
				Content: "Some text <think>reasoning here</think> more text",
			},
			wantNil: false,
		},
		{
			name: "without tags",
			response: &types.ModelResponse{
				Content: "No reasoning tags",
			},
			wantNil: true,
		},
		{
			name: "with existing ReasoningContent",
			response: &types.ModelResponse{
				Content:          "Content",
				ReasoningContent: types.NewReasoningContent("existing reasoning"),
			},
			wantNil: false,
		},
		{
			name:     "nil response",
			response: nil,
			wantNil:  true,
		},
		{
			name: "empty content",
			response: &types.ModelResponse{
				Content: "",
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractReasoning(ctx, model, tt.response)
			if err != nil {
				t.Errorf("ExtractReasoning() error = %v", err)
			}
			if (got == nil) != tt.wantNil {
				t.Errorf("ExtractReasoning() got nil = %v, want nil = %v", got == nil, tt.wantNil)
			}
			if !tt.wantNil && got != nil && got.Content == "" {
				t.Error("ExtractReasoning() returned non-nil but empty content")
			}
		})
	}
}

func TestExtractReasoning_ReasoningProviders(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		model    models.Model
		response *types.ModelResponse
		wantNil  bool
	}{
		{
			name:  "Gemini reasoning content",
			model: &MockModel{id: "gemini-2.5-flash", provider: "gemini"},
			response: &types.ModelResponse{
				ReasoningContent: types.NewReasoningContent("gemini thoughts"),
			},
			wantNil: false,
		},
		{
			name: "Anthropic reasoning content",
			model: &MockReasoningModel{
				MockModel:        MockModel{id: "claude-3-5-sonnet", provider: "anthropic"},
				reasoningEnabled: true,
			},
			response: &types.ModelResponse{
				ReasoningContent: types.NewReasoningContent("anthropic thinking"),
			},
			wantNil: false,
		},
		{
			name: "VertexAI reasoning content",
			model: &MockReasoningModel{
				MockModel:        MockModel{id: "claude-3-5-sonnet@20240620", provider: "vertexai"},
				reasoningEnabled: true,
			},
			response: &types.ModelResponse{
				ReasoningContent: types.NewReasoningContent("vertex reasoning"),
			},
			wantNil: false,
		},
		{
			name:     "Gemini without reasoning",
			model:    &MockModel{id: "gemini-2.5-flash", provider: "gemini"},
			response: &types.ModelResponse{},
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractReasoning(ctx, tt.model, tt.response)
			if err != nil {
				t.Fatalf("ExtractReasoning() unexpected error = %v", err)
			}
			if (got == nil) != tt.wantNil {
				t.Fatalf("ExtractReasoning() nil = %v, wantNil = %v", got == nil, tt.wantNil)
			}
			if !tt.wantNil && got.Content == "" {
				t.Errorf("ExtractReasoning() returned empty content")
			}
		})
	}
}

func TestExtractReasoning_UnsupportedProvider(t *testing.T) {
	ctx := context.Background()
	model := &MockModel{id: "some-model", provider: "unsupported"}

	response := &types.ModelResponse{
		Content: "Test content",
	}

	got, err := ExtractReasoning(ctx, model, response)
	if err != nil {
		t.Errorf("ExtractReasoning() unexpected error = %v", err)
	}
	if got != nil {
		t.Errorf("ExtractReasoning() for unsupported provider should return nil, got %v", got)
	}
}

func TestWrapReasoningContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "normal content",
			content: "reasoning content",
			want:    "<thinking>\nreasoning content\n</thinking>",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name:    "multiline content",
			content: "line1\nline2\nline3",
			want:    "<thinking>\nline1\nline2\nline3\n</thinking>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WrapReasoningContent(tt.content)
			if got != tt.want {
				t.Errorf("WrapReasoningContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

// MockDetector 用于测试 / MockDetector for testing
type MockDetector struct {
	provider string
}

func (d *MockDetector) IsReasoningModel(model models.Model) bool {
	return true
}

func (d *MockDetector) Provider() string {
	return d.provider
}

func TestRegistry_ThreadSafety(t *testing.T) {
	registry := NewRegistry()

	// 测试并发注册 / Test concurrent registration
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			detector := &MockDetector{provider: "test"}
			registry.RegisterDetector(detector)
			_, _ = registry.GetDetector("test")
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成 / Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestReasoningContent_Builders(t *testing.T) {
	// Test NewReasoningContent
	rc := types.NewReasoningContent("test content")
	if rc == nil {
		t.Fatal("NewReasoningContent() returned nil")
	}
	if rc.Content != "test content" {
		t.Errorf("NewReasoningContent() content = %q, want %q", rc.Content, "test content")
	}

	// Test WithRedacted
	redacted := "redacted"
	rc = rc.WithRedacted(redacted)
	if rc.RedactedContent == nil {
		t.Fatal("WithRedacted() did not set RedactedContent")
	}
	if *rc.RedactedContent != redacted {
		t.Errorf("WithRedacted() = %q, want %q", *rc.RedactedContent, redacted)
	}

	// Test WithTokenCount
	count := 100
	rc = rc.WithTokenCount(count)
	if rc.TokenCount == nil {
		t.Fatal("WithTokenCount() did not set TokenCount")
	}
	if *rc.TokenCount != count {
		t.Errorf("WithTokenCount() = %d, want %d", *rc.TokenCount, count)
	}

	// Test chaining
	rc2 := types.NewReasoningContent("chain test").
		WithRedacted("redacted chain").
		WithTokenCount(50)

	if rc2.Content != "chain test" {
		t.Errorf("Chaining failed: Content = %q", rc2.Content)
	}
	if rc2.RedactedContent == nil || *rc2.RedactedContent != "redacted chain" {
		t.Error("Chaining failed: RedactedContent not set correctly")
	}
	if rc2.TokenCount == nil || *rc2.TokenCount != 50 {
		t.Error("Chaining failed: TokenCount not set correctly")
	}
}
