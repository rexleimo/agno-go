package reasoning

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/reasoning/anthropic"
	"github.com/rexleimo/agno-go/pkg/hno/reasoning/gemini"
	"github.com/rexleimo/agno-go/pkg/hno/reasoning/openai"
	"github.com/rexleimo/agno-go/pkg/hno/reasoning/vertexai"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// ReasoningDetector 检测模型是否支持推理能力
// ReasoningDetector detects if a model supports reasoning capabilities
type ReasoningDetector interface {
	// IsReasoningModel 判断给定模型是否支持推理
	// Determines if the given model supports reasoning
	IsReasoningModel(model models.Model) bool

	// Provider 返回该检测器所属的提供商名称
	// Returns the provider name this detector belongs to
	Provider() string
}

// ReasoningExtractor 从模型响应中提取推理内容
// ReasoningExtractor extracts reasoning content from model responses
type ReasoningExtractor interface {
	// Extract 从响应中提取推理内容
	// Extracts reasoning content from the response
	Extract(ctx context.Context, response *types.ModelResponse) (*types.ReasoningContent, error)

	// Provider 返回该提取器所属的提供商名称
	// Returns the provider name this extractor belongs to
	Provider() string
}

// Registry 管理所有推理检测器和提取器
// Registry manages all reasoning detectors and extractors
type Registry struct {
	detectors  map[string]ReasoningDetector
	extractors map[string]ReasoningExtractor
	mu         sync.RWMutex
}

// NewRegistry 创建一个新的注册中心
// NewRegistry creates a new registry
func NewRegistry() *Registry {
	return &Registry{
		detectors:  make(map[string]ReasoningDetector),
		extractors: make(map[string]ReasoningExtractor),
	}
}

// RegisterDetector 注册一个检测器
// Registers a detector
func (r *Registry) RegisterDetector(detector ReasoningDetector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := normalizeProvider(detector.Provider())
	r.detectors[key] = detector
}

// RegisterExtractor 注册一个提取器
// Registers an extractor
func (r *Registry) RegisterExtractor(extractor ReasoningExtractor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := normalizeProvider(extractor.Provider())
	r.extractors[key] = extractor
}

// GetDetector 获取指定提供商的检测器
// Gets the detector for a specific provider
func (r *Registry) GetDetector(provider string) (ReasoningDetector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := normalizeProvider(provider)
	detector, ok := r.detectors[key]
	return detector, ok
}

// GetExtractor 获取指定提供商的提取器
// Gets the extractor for a specific provider
func (r *Registry) GetExtractor(provider string) (ReasoningExtractor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := normalizeProvider(provider)
	extractor, ok := r.extractors[key]
	return extractor, ok
}

// DefaultRegistry 是全局默认注册中心
// DefaultRegistry is the global default registry
var DefaultRegistry = NewRegistry()

// IsReasoningModel 检查模型是否支持推理(使用默认注册中心)
// Checks if a model supports reasoning (using default registry)
func IsReasoningModel(model models.Model) bool {
	if model == nil {
		return false
	}
	detector, ok := DefaultRegistry.GetDetector(model.GetProvider())
	if !ok {
		return false
	}
	return detector.IsReasoningModel(model)
}

// ExtractReasoning 从响应中提取推理内容(使用默认注册中心)
// Extracts reasoning content from response (using default registry)
func ExtractReasoning(ctx context.Context, model models.Model, response *types.ModelResponse) (*types.ReasoningContent, error) {
	if model == nil {
		return nil, nil
	}
	extractor, ok := DefaultRegistry.GetExtractor(model.GetProvider())
	if !ok {
		return nil, nil // 不支持的提供商,返回 nil
	}
	return extractor.Extract(ctx, response)
}

func normalizeProvider(provider string) string {
	p := strings.TrimSpace(strings.ToLower(provider))
	switch p {
	case "vertex-ai", "google-vertexai", "google_vertexai", "google-vertex-ai":
		return "vertexai"
	case "claude":
		return "anthropic"
	default:
		return p
	}
}

// WrapReasoningContent 将推理内容包装成 <thinking> 格式
// Wraps reasoning content in <thinking> tags
func WrapReasoningContent(content string) string {
	if content == "" {
		return ""
	}
	return fmt.Sprintf("<thinking>\n%s\n</thinking>", content)
}

func init() {
	// 注册 OpenAI 推理支持
	// Register OpenAI reasoning support
	DefaultRegistry.RegisterDetector(&openai.Detector{})
	DefaultRegistry.RegisterExtractor(&openai.Extractor{})

	// 注册 Gemini 推理支持
	// Register Gemini reasoning support
	DefaultRegistry.RegisterDetector(&gemini.Detector{})
	DefaultRegistry.RegisterExtractor(&gemini.Extractor{})

	// 注册 Anthropic 推理支持
	// Register Anthropic reasoning support
	DefaultRegistry.RegisterDetector(&anthropic.Detector{})
	DefaultRegistry.RegisterExtractor(&anthropic.Extractor{})

	// 注册 VertexAI 推理支持
	// Register VertexAI reasoning support
	DefaultRegistry.RegisterDetector(&vertexai.Detector{})
	DefaultRegistry.RegisterExtractor(&vertexai.Extractor{})
}
