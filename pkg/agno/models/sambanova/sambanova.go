package sambanova

import (
	"time"

	"github.com/rexleimo/agno-go/pkg/agno/models/openai"
)

const defaultBaseURL = "https://api.sambanova.ai/v1"

// Sambanova is the OpenAI-compatible sambanova provider.
// Sambanova 是基于 OpenAI 兼容协议的 sambanova provider。
type Sambanova = openai.OpenAICompat

type Config struct {
	APIKey      string
	BaseURL     string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
}

// New creates a new sambanova model instance.
// New 创建新的 sambanova 模型实例。
func New(modelID string, config Config) (*Sambanova, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	cfg := openai.CompatConfig{
		Provider:      "sambanova",
		APIKey:        config.APIKey,
		BaseURL:       baseURL,
		Temperature:   config.Temperature,
		MaxTokens:     config.MaxTokens,
		Timeout:       config.Timeout,
		RequireAPIKey: true,
	}

	return openai.NewCompat(modelID, cfg)
}
