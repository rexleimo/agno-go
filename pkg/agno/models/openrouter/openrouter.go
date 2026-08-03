package openrouter

import (
	"time"

	"github.com/rexleimo/agno-go/pkg/agno/models/openai"
)

const defaultBaseURL = "https://openrouter.ai/api/v1"

// Openrouter is the OpenAI-compatible openrouter provider.
// Openrouter 是基于 OpenAI 兼容协议的 openrouter provider。
type Openrouter = openai.OpenAICompat

type Config struct {
	APIKey      string
	BaseURL     string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
	Referer     string // optional; sent as HTTP-Referer
	Title       string // optional; sent as X-Title
}

// New creates a new openrouter model instance.
// New 创建新的 openrouter 模型实例。
func New(modelID string, config Config) (*Openrouter, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	cfg := openai.CompatConfig{
		Provider:      "openrouter",
		APIKey:        config.APIKey,
		BaseURL:       baseURL,
		Temperature:   config.Temperature,
		MaxTokens:     config.MaxTokens,
		Timeout:       config.Timeout,
		RequireAPIKey: true,
		ExtraHeaders: map[string]string{
			"HTTP-Referer": config.Referer,
			"X-Title":      config.Title,
		},
	}

	return openai.NewCompat(modelID, cfg)
}
