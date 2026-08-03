package models

import (
	"context"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// ResponseFormat controls the model output format.
// ResponseFormat 控制模型输出格式。
type ResponseFormat string

const (
	// ResponseFormatText: plain text output (default).
	// ResponseFormatText: 纯文本输出（默认）。
	ResponseFormatText ResponseFormat = "text"
	// ResponseFormatJSONObject: a JSON object without a schema constraint.
	// ResponseFormatJSONObject: 无 schema 约束的 JSON 对象。
	ResponseFormatJSONObject ResponseFormat = "json_object"
	// ResponseFormatJSONSchema: JSON output constrained by a schema.
	// ResponseFormatJSONSchema: 受 schema 约束的 JSON 输出。
	ResponseFormatJSONSchema ResponseFormat = "json_schema"
)

// ImageInput represents an image passed to a multimodal model.
// ImageInput 表示传递给多模态模型的图像。
type ImageInput struct {
	// URL of a remote image (http/https).
	// URL 远程图像地址（http/https）。
	URL string `json:"url,omitempty"`
	// Base64 encoded image data (alternative to URL).
	// Base64 编码的图像数据（URL 的替代方案）。
	Base64 string `json:"base64,omitempty"`
	// MediaType such as "image/png" (required when Base64 is set).
	// MediaType 如 "image/png"（设置 Base64 时必填）。
	MediaType string `json:"media_type,omitempty"`
}

// InvokeRequest contains parameters for model invocation
// InvokeRequest 包含模型调用参数
type InvokeRequest struct {
	Messages    []*types.Message
	Tools       []ToolDefinition
	Temperature float64
	MaxTokens   int
	Stream      bool
	Extra       map[string]interface{}

	// ResponseFormat requests structured output (optional; providers that
	// implement StructuredOutputProvider may honour it).
	// ResponseFormat 请求结构化输出（可选；实现 StructuredOutputProvider 的提供商可支持）。
	ResponseFormat ResponseFormat `json:"response_format,omitempty"`
	// Schema is the JSON schema used when ResponseFormat is json_schema.
	// Schema 是 ResponseFormat 为 json_schema 时使用的 JSON schema。
	Schema map[string]interface{} `json:"schema,omitempty"`
	// Images carries multimodal inputs (optional).
	// Images 携带多模态输入（可选）。
	Images []ImageInput `json:"images,omitempty"`
}

// ToolDefinition defines a tool that can be called by the model
// ToolDefinition 定义模型可调用的工具
type ToolDefinition struct {
	Type     string         `json:"type"` // "function"
	Function FunctionSchema `json:"function"`
}

// FunctionSchema defines the schema of a callable function
// FunctionSchema 定义可调用函数的 schema
type FunctionSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// Model represents a language model interface. Implementations are pure
// adapters for a single model invocation; the agentic loop lives in the
// runner package, not in the model.
// Model 表示语言模型接口。实现是单次模型调用的纯适配器；
// agent 循环位于 runner 包中，而非模型内部。
type Model interface {
	// Invoke calls the model synchronously
	// Invoke 同步调用模型
	Invoke(ctx context.Context, req *InvokeRequest) (*types.ModelResponse, error)

	// InvokeStream calls the model with streaming response
	// InvokeStream 以流式响应调用模型
	InvokeStream(ctx context.Context, req *InvokeRequest) (<-chan types.ResponseChunk, error)

	// GetProvider returns the model provider name
	// GetProvider 返回模型提供商名称
	GetProvider() string

	// GetID returns the model identifier
	// GetID 返回模型标识符
	GetID() string

	// GetName returns the model name
	// GetName 返回模型名称
	GetName() string
}

// Optional capability interfaces. Providers implement any subset; consumers
// detect them with the As* helpers and degrade gracefully when absent.
// 可选能力接口。提供商实现任意子集；消费者使用 As* 辅助函数探测，
// 缺失时优雅降级。

// StructuredOutputModel supports schema-constrained JSON output.
// StructuredOutputModel 支持受 schema 约束的 JSON 输出。
type StructuredOutputModel interface {
	// InvokeStructured invokes the model with a response-format constraint.
	// InvokeStructured 以响应格式约束调用模型。
	InvokeStructured(ctx context.Context, req *InvokeRequest) (*types.ModelResponse, error)
}

// MultimodalModel supports image inputs.
// MultimodalModel 支持图像输入。
type MultimodalModel interface {
	// SupportsImages reports whether the model accepts image inputs.
	// SupportsImages 报告模型是否接受图像输入。
	SupportsImages() bool
}

// ReasoningModel supports reasoning-content extraction.
// ReasoningModel 支持推理内容提取。
type ReasoningModel interface {
	// ExtractReasoning extracts reasoning content from a response.
	// ExtractReasoning 从响应中提取推理内容。
	ExtractReasoning(ctx context.Context, resp *types.ModelResponse) (*types.ReasoningContent, error)
}

// AsStructuredOutput returns the StructuredOutputModel implementation, or nil.
// AsStructuredOutput 返回 StructuredOutputModel 实现，若无则返回 nil。
func AsStructuredOutput(m Model) StructuredOutputModel {
	if so, ok := m.(StructuredOutputModel); ok {
		return so
	}
	return nil
}

// AsMultimodal returns the MultimodalModel implementation, or nil.
// AsMultimodal 返回 MultimodalModel 实现，若无则返回 nil。
func AsMultimodal(m Model) MultimodalModel {
	if mm, ok := m.(MultimodalModel); ok {
		return mm
	}
	return nil
}

// AsReasoning returns the ReasoningModel implementation, or nil.
// AsReasoning 返回 ReasoningModel 实现，若无则返回 nil。
func AsReasoning(m Model) ReasoningModel {
	if rm, ok := m.(ReasoningModel); ok {
		return rm
	}
	return nil
}

// BaseModel provides common functionality for model implementations
// BaseModel 为模型实现提供公共功能
type BaseModel struct {
	ID       string
	Name     string
	Provider string
}

// GetProvider returns the model provider
// GetProvider 返回模型提供商
func (m *BaseModel) GetProvider() string {
	return m.Provider
}

// GetID returns the model ID
// GetID 返回模型 ID
func (m *BaseModel) GetID() string {
	return m.ID
}

// GetName returns the model name
// GetName 返回模型名称
func (m *BaseModel) GetName() string {
	if m.Name != "" {
		return m.Name
	}
	return m.ID
}
