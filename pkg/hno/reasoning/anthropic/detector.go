package anthropic

import (
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/models"
)

type reasoningCapable interface {
	SupportsReasoning() bool
}

// Detector 检测 Anthropic 推理模型
// Detector detects Anthropic reasoning models
type Detector struct{}

// IsReasoningModel 检查是否为 Anthropic 推理模型
// IsReasoningModel checks whether the given model is an Anthropic reasoning model
func (d *Detector) IsReasoningModel(model models.Model) bool {
	if model == nil {
		return false
	}

	provider := strings.ToLower(model.GetProvider())
	if provider != "anthropic" && provider != "claude" {
		return false
	}

	modelID := strings.ToLower(model.GetID())
	if strings.Contains(modelID, "@") {
		// IDs with version suffix (@YYYYMMDD) belong to Vertex AI mirror models
		return false
	}

	if capable, ok := model.(reasoningCapable); ok {
		return capable.SupportsReasoning()
	}

	return false
}

// Provider 返回提供商名称
// Provider returns the provider name
func (d *Detector) Provider() string {
	return "anthropic"
}
