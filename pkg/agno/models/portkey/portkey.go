package portkey

import (
	"time"

	"github.com/rexleimo/agno-go/pkg/agno/models/openai"
)

const defaultBaseURL = "https://api.portkey.ai/v1"

// Portkey is the OpenAI-compatible portkey provider.
// Portkey 是基于 OpenAI 兼容协议的 portkey provider。
type Portkey = openai.OpenAICompat

type Config struct {
	APIKey      string
	BaseURL     string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
}

// New creates a new portkey model instance.
// New 创建新的 portkey 模型实例。
func New(modelID string, config Config) (*Portkey, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	cfg := openai.CompatConfig{
		Provider:      "portkey",
		APIKey:        config.APIKey,
		BaseURL:       baseURL,
		Temperature:   config.Temperature,
		MaxTokens:     config.MaxTokens,
		Timeout:       config.Timeout,
		RequireAPIKey: true,
	}

	return openai.NewCompat(modelID, cfg)
}
