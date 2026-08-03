package lmstudio

import (
	"time"

	"github.com/rexleimo/agno-go/pkg/agno/models/openai"
)

const defaultBaseURL = "http://localhost:1234/v1"

// Lmstudio is the OpenAI-compatible lmstudio provider.
// Lmstudio 是基于 OpenAI 兼容协议的 lmstudio provider。
type Lmstudio = openai.OpenAICompat

type Config struct {
	APIKey      string // optional; LM Studio typically does not require
	BaseURL     string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
}

// New creates a new lmstudio model instance.
// New 创建新的 lmstudio 模型实例。
func New(modelID string, config Config) (*Lmstudio, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	cfg := openai.CompatConfig{
		Provider:      "lmstudio",
		APIKey:        config.APIKey,
		BaseURL:       baseURL,
		Temperature:   config.Temperature,
		MaxTokens:     config.MaxTokens,
		Timeout:       config.Timeout,
		RequireAPIKey: false,
	}

	return openai.NewCompat(modelID, cfg)
}
