package groq

import (
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

const defaultBaseURL = "https://api.groq.com/openai/v1"

// Groq is the OpenAI-compatible groq provider.
// Groq 是基于 OpenAI 兼容协议的 groq provider。
type Groq = openai.OpenAICompat

type Config struct {
	APIKey      string        // Groq API Key / Groq API 密钥
	BaseURL     string        // API Base URL / API 基础 URL
	Temperature float64       // Temperature parameter / 温度参数
	MaxTokens   int           // Max tokens to generate / 生成的最大 token 数
	Timeout     time.Duration // Request timeout / 请求超时时间
}

// New creates a new groq model instance.
// New 创建新的 groq 模型实例。
func New(modelID string, config Config) (*Groq, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	cfg := openai.CompatConfig{
		Provider:      "groq",
		APIKey:        config.APIKey,
		BaseURL:       baseURL,
		Temperature:   config.Temperature,
		MaxTokens:     config.MaxTokens,
		Timeout:       config.Timeout,
		RequireAPIKey: true,
	}

	return openai.NewCompat(modelID, cfg)
}
