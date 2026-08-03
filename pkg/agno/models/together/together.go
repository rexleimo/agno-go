package together

import (
	"time"

	"github.com/rexleimo/agno-go/pkg/agno/models/openai"
)

const defaultBaseURL = "https://api.together.xyz/v1"

// Together is the OpenAI-compatible together provider.
// Together 是基于 OpenAI 兼容协议的 together provider。
type Together = openai.OpenAICompat

type Config struct {
	APIKey      string
	BaseURL     string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
}

// New creates a new together model instance.
// New 创建新的 together 模型实例。
func New(modelID string, config Config) (*Together, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	cfg := openai.CompatConfig{
		Provider:      "together",
		APIKey:        config.APIKey,
		BaseURL:       baseURL,
		Temperature:   config.Temperature,
		MaxTokens:     config.MaxTokens,
		Timeout:       config.Timeout,
		RequireAPIKey: true,
	}

	return openai.NewCompat(modelID, cfg)
}
