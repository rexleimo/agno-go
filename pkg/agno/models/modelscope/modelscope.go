package modelscope

import (
	"github.com/rexleimo/agno-go/pkg/agno/models/openai"
)

const defaultBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"

// Modelscope is the OpenAI-compatible modelscope provider.
// Modelscope 是基于 OpenAI 兼容协议的 modelscope provider。
type Modelscope = openai.OpenAICompat

type Config struct {
	APIKey      string // DASHSCOPE_API_KEY from Alibaba Cloud
	BaseURL     string
	Temperature float64
	MaxTokens   int
}

// New creates a new modelscope model instance.
// New 创建新的 modelscope 模型实例。
func New(modelID string, config Config) (*Modelscope, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	cfg := openai.CompatConfig{
		Provider:      "modelscope",
		APIKey:        config.APIKey,
		BaseURL:       baseURL,
		Temperature:   config.Temperature,
		MaxTokens:     config.MaxTokens,
		RequireAPIKey: true,
	}

	return openai.NewCompat(modelID, cfg)
}
