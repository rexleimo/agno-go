package types

import "testing"

func TestModelResponse_HasToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		response *ModelResponse
		want     bool
	}{
		{
			name: "with tool calls",
			response: &ModelResponse{
				ToolCalls: []ToolCall{
					{ID: "call_1", Type: "function"},
				},
			},
			want: true,
		},
		{
			name: "without tool calls",
			response: &ModelResponse{
				Content: "hello",
			},
			want: false,
		},
		{
			name: "empty tool calls",
			response: &ModelResponse{
				ToolCalls: []ToolCall{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.response.HasToolCalls(); got != tt.want {
				t.Errorf("HasToolCalls() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelResponse_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		response *ModelResponse
		want     bool
	}{
		{
			name:     "empty response",
			response: &ModelResponse{},
			want:     true,
		},
		{
			name: "with content",
			response: &ModelResponse{
				Content: "hello",
			},
			want: false,
		},
		{
			name: "with tool calls",
			response: &ModelResponse{
				ToolCalls: []ToolCall{{ID: "call_1"}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.response.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
