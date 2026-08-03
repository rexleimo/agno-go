package internlm

import (
	"time"

	"github.com/rexleimo/agno-go/pkg/agno/models/openai"
	"github.com/rexleimo/agno-go/pkg/agno/types"
)

// Internlm is the OpenAI-compatible internlm provider.
// Internlm 是基于 OpenAI 兼容协议的 internlm provider。
type Internlm = openai.OpenAICompat

type Config struct {
	APIKey      string
	BaseURL     string // required by deployment; no default provided
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
}

// New creates a new internlm model instance.
// New 创建新的 internlm 模型实例。
func New(modelID string, config Config) (*Internlm, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		return nil, types.NewInvalidConfigError("internlm BaseURL is required", nil)
	}

	cfg := openai.CompatConfig{
		Provider:      "internlm",
		APIKey:        config.APIKey,
		BaseURL:       baseURL,
		Temperature:   config.Temperature,
		MaxTokens:     config.MaxTokens,
		Timeout:       config.Timeout,
		RequireAPIKey: true,
	}

	return openai.NewCompat(modelID, cfg)
}
