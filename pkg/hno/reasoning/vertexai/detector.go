package vertexai

import (
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/models"
)

type reasoningCapable interface {
	SupportsReasoning() bool
}

// Detector 检测 VertexAI 推理模型
// Detector detects VertexAI reasoning models
type Detector struct{}

// Provider 返回提供商名称
// Provider returns the provider name handled by this detector
func (d *Detector) Provider() string {
	return "vertexai"
}

// IsReasoningModel 判断模型是否为 VertexAI 推理模型
// IsReasoningModel determines if the given model is a VertexAI reasoning model
func (d *Detector) IsReasoningModel(model models.Model) bool {
	if model == nil {
		return false
	}

	provider := strings.ToLower(model.GetProvider())
	switch provider {
	case "vertexai", "vertex-ai", "google-vertexai", "google_vertexai":
		// continue
	default:
		return false
	}

	if capable, ok := model.(reasoningCapable); ok {
		return capable.SupportsReasoning()
	}

	return false
}
