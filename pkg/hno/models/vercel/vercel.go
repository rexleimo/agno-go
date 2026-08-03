package vercel

import (
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Vercel is the OpenAI-compatible vercel provider.
// Vercel 是基于 OpenAI 兼容协议的 vercel provider。
type Vercel = openai.OpenAICompat

type Config struct {
	APIKey      string
	BaseURL     string // required to point to Vercel inference endpoint
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
}

// New creates a new vercel model instance.
// New 创建新的 vercel 模型实例。
func New(modelID string, config Config) (*Vercel, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		return nil, types.NewInvalidConfigError("vercel BaseURL is required", nil)
	}

	cfg := openai.CompatConfig{
		Provider:      "vercel",
		APIKey:        config.APIKey,
		BaseURL:       baseURL,
		Temperature:   config.Temperature,
		MaxTokens:     config.MaxTokens,
		Timeout:       config.Timeout,
		RequireAPIKey: true,
	}

	return openai.NewCompat(modelID, cfg)
}
