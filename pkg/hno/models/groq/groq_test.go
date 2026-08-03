package groq

import (
	"testing"
)

func TestGroq_GetProvider(t *testing.T) {
	model, err := New(ModelLlama38B, Config{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	if model.GetProvider() != "groq" {
		t.Errorf("GetProvider() = %v, want groq", model.GetProvider())
	}
}

func TestGroq_GetID(t *testing.T) {
	modelID := ModelLlama370B
	model, err := New(modelID, Config{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	if model.GetID() != modelID {
		t.Errorf("GetID() = %v, want %v", model.GetID(), modelID)
	}
}

// Additional tests for error handling and edge cases

func TestGetModelInfo(t *testing.T) {
	tests := []struct {
		name      string
		modelID   string
		wantFound bool
		wantName  string
	}{
		{
			name:      "llama 3.1 8b",
			modelID:   ModelLlama38B,
			wantFound: true,
			wantName:  "LLaMA 3.1 8B Instant",
		},
		{
			name:      "llama 3.1 70b",
			modelID:   ModelLlama370B,
			wantFound: true,
			wantName:  "LLaMA 3.1 70B Versatile",
		},
		{
			name:      "mixtral",
			modelID:   ModelMixtral8x7B,
			wantFound: true,
			wantName:  "Mixtral 8x7B",
		},
		{
			name:      "unknown model",
			modelID:   "unknown-model",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, found := GetModelInfo(tt.modelID)
			if found != tt.wantFound {
				t.Errorf("GetModelInfo() found = %v, want %v", found, tt.wantFound)
			}
			if tt.wantFound && info.Name != tt.wantName {
				t.Errorf("GetModelInfo() name = %v, want %v", info.Name, tt.wantName)
			}
		})
	}
}

// TestAvailableModels tests that all model constants are in the map
// 测试所有模型常量都在映射表中
func TestAvailableModels(t *testing.T) {
	expectedModels := []string{
		ModelLlama38B,
		ModelLlama370B,
		ModelLlama3405B,
		ModelMixtral8x7B,
		ModelGemma2_9B,
		ModelWhisperLarge,
		ModelLlamaGuard3,
	}

	for _, modelID := range expectedModels {
		t.Run(modelID, func(t *testing.T) {
			info, found := GetModelInfo(modelID)
			if !found {
				t.Errorf("Model %s not found in AvailableModels", modelID)
			}
			if info.ID != modelID {
				t.Errorf("Model ID mismatch: got %s, want %s", info.ID, modelID)
			}
		})
	}
}
