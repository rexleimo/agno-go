package openai

import (
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/models"
)

// Detector 检测 OpenAI 推理模型
// Detector detects OpenAI reasoning models
type Detector struct{}

// IsReasoningModel 检查是否为 OpenAI 推理模型
// Checks if this is an OpenAI reasoning model
func (d *Detector) IsReasoningModel(model models.Model) bool {
	// 检查提供商 / Check provider
	if model.GetProvider() != "openai" {
		return false
	}

	// 检查模型 ID / Check model ID
	modelID := strings.ToLower(model.GetID())

	// OpenAI 推理模型: o1, o3, o4 系列
	// OpenAI reasoning models: o1, o3, o4 series
	return strings.Contains(modelID, "o1") ||
		strings.Contains(modelID, "o3") ||
		strings.Contains(modelID, "o4")
}

// Provider 返回提供商名称
// Returns the provider name
func (d *Detector) Provider() string {
	return "openai"
}
