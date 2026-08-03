package gemini

import (
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/models"
)

type reasoningCapable interface {
	SupportsReasoning() bool
}

// Detector 检测 Gemini 推理模型
// Detector detects Gemini reasoning models
type Detector struct{}

// IsReasoningModel 检查是否为 Gemini 推理模型
// Checks if this is a Gemini reasoning model
func (d *Detector) IsReasoningModel(model models.Model) bool {
	if model == nil {
		return false
	}

	if !strings.EqualFold(model.GetProvider(), "gemini") {
		return false
	}

	if capable, ok := model.(reasoningCapable); ok {
		if capable.SupportsReasoning() {
			return true
		}
	}

	modelID := strings.ToLower(model.GetID())
	if strings.Contains(modelID, "2.5") ||
		strings.Contains(modelID, "thinking") ||
		strings.Contains(modelID, "flash-thinking") {
		return true
	}

	return false
}

// Provider 返回提供商名称
// Returns the provider name
func (d *Detector) Provider() string {
	return "gemini"
}
