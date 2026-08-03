package text

import (
	"context"
	"net/http"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	evolinkp "github.com/rexleimo/agno-go/pkg/hno/models/evolink"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Config for Evolink text model
type Config struct {
	APIKey      string
	BaseURL     string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
	HTTPClient  *http.Client
}

// Text implements models.Model for EvoLink chat completions
type Text struct {
	models.BaseModel
	client *evolinkp.Client
	config Config
}

// New creates a new Evolink text model
func New(modelID string, cfg Config) (*Text, error) {
	c, err := evolinkp.NewClient(evolinkp.Config{
		APIKey:     cfg.APIKey,
		BaseURL:    cfg.BaseURL,
		Timeout:    cfg.Timeout,
		HTTPClient: cfg.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return &Text{
		BaseModel: models.BaseModel{ID: modelID, Provider: "evolink"},
		client:    c,
		config:    cfg,
	}, nil
}

// Invoke sends a chat completion request to EvoLink
func (t *Text) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	msgs := models.ConvertMessages(req.Messages)
	resp, err := Complete(ctx, t.client, Options{
		Model:       t.ID,
		Temperature: firstNonZero(req.Temperature, t.config.Temperature),
		MaxTokens:   firstNonZeroInt(req.MaxTokens, t.config.MaxTokens),
		Messages:    msgs,
	})
	if err != nil {
		return nil, err
	}
	return &types.ModelResponse{
		ID:      resp.ID,
		Content: resp.Content,
		Model:   resp.Model,
		Usage: types.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// InvokeStream is not supported for EvoLink text currently
func (t *Text) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	return nil, types.NewAPIError("streaming not supported", nil)
}

func firstNonZero(a, b float64) float64 {
	if a > 0 {
		return a
	}
	return b
}
func firstNonZeroInt(a, b int) int {
	if a > 0 {
		return a
	}
	return b
}
