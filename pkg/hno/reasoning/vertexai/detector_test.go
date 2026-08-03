package vertexai

import (
	"context"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

type mockReasoningModel struct {
	id               string
	provider         string
	reasoningEnabled bool
}

func (m *mockReasoningModel) GetID() string       { return m.id }
func (m *mockReasoningModel) GetProvider() string { return m.provider }
func (m *mockReasoningModel) GetName() string     { return m.id }
func (m *mockReasoningModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	return nil, nil
}
func (m *mockReasoningModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	return nil, nil
}
func (m *mockReasoningModel) SupportsReasoning() bool { return m.reasoningEnabled }

func TestDetector_IsReasoningModel(t *testing.T) {
	d := &Detector{}

	tests := []struct {
		name  string
		model models.Model
		want  bool
	}{
		{
			name: "vertexai provider with reasoning",
			model: &mockReasoningModel{
				id:               "claude-3-5-sonnet@20240620",
				provider:         "vertexai",
				reasoningEnabled: true,
			},
			want: true,
		},
		{
			name: "vertex-ai alias with reasoning",
			model: &mockReasoningModel{
				id:               "claude-3-5-sonnet@20240620",
				provider:         "vertex-ai",
				reasoningEnabled: true,
			},
			want: true,
		},
		{
			name: "vertexai provider without reasoning interface",
			model: &mockReasoningModel{
				id:               "claude-3-5-sonnet@20240620",
				provider:         "vertexai",
				reasoningEnabled: false,
			},
			want: false,
		},
		{
			name: "non-vertex provider",
			model: &mockReasoningModel{
				id:               "claude-3-5-sonnet@20240620",
				provider:         "anthropic",
				reasoningEnabled: true,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := d.IsReasoningModel(tt.model); got != tt.want {
				t.Errorf("IsReasoningModel() = %v, want %v", got, tt.want)
			}
		})
	}
}
