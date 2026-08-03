package deepseek

import (
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

const defaultBaseURL = "https://api.deepseek.com/v1"

// DeepSeek is the OpenAI-compatible DeepSeek provider.
// DeepSeek 是基于 OpenAI 兼容协议的 DeepSeek provider。
type DeepSeek = openai.OpenAICompat

// Config contains DeepSeek-specific configuration
// Config 包含 DeepSeek 特定配置
type Config struct {
	APIKey      string
	BaseURL     string
	Temperature float64
	MaxTokens   int
}

// New creates a new DeepSeek model instance.
// DeepSeek API is fully compatible with OpenAI API format.
// New 创建新的 DeepSeek 模型实例。DeepSeek API 与 OpenAI API 格式完全兼容。
func New(modelID string, config Config) (*DeepSeek, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return openai.NewCompat(modelID, openai.CompatConfig{
		Provider:      "deepseek",
		APIKey:        config.APIKey,
		BaseURL:       baseURL,
		Temperature:   config.Temperature,
		MaxTokens:     config.MaxTokens,
		RequireAPIKey: true,
	})
}
