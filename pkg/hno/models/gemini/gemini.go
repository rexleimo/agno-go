package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

const (
	defaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"
)

// Gemini wraps the Google Gemini API client
type Gemini struct {
	models.BaseModel
	config     Config
	httpClient *http.Client
}

// Config contains Gemini-specific configuration
type Config struct {
	APIKey          string
	BaseURL         string
	Temperature     float64
	MaxTokens       int
	ThinkingBudget  int
	IncludeThoughts *bool
}

// New creates a new Gemini model instance
func New(modelID string, config Config) (*Gemini, error) {
	if config.APIKey == "" {
		return nil, types.NewInvalidConfigError("Gemini API key is required", nil)
	}

	if config.BaseURL == "" {
		config.BaseURL = defaultBaseURL
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 8192
	}

	return &Gemini{
		BaseModel: models.BaseModel{
			ID:       modelID,
			Provider: "gemini",
			Name:     modelID,
		},
		config:     config,
		httpClient: &http.Client{},
	}, nil
}

// SupportsReasoning returns whether the current configuration enables reasoning features
func (g *Gemini) SupportsReasoning() bool {
	if g == nil {
		return false
	}

	modelID := strings.ToLower(g.ID)
	if strings.Contains(modelID, "2.5") ||
		strings.Contains(modelID, "thinking") ||
		strings.Contains(modelID, "reasoning") {
		return true
	}

	if g.config.ThinkingBudget > 0 {
		return true
	}

	if g.config.IncludeThoughts != nil && *g.config.IncludeThoughts {
		return true
	}

	return false
}

// Invoke calls the Gemini API synchronously
func (g *Gemini) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	geminiReq := g.buildGeminiRequest(req)

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", g.config.BaseURL, g.ID, g.config.APIKey)

	var geminiResp GeminiResponse
	err := models.PostJSON(ctx, g.httpClient, url, nil, geminiReq, &geminiResp)
	if err != nil {
		return nil, err
	}

	return g.convertResponse(&geminiResp), nil
}

// InvokeStream calls the Gemini API with streaming response
func (g *Gemini) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	geminiReq := g.buildGeminiRequest(req)

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s&alt=sse", g.config.BaseURL, g.ID, g.config.APIKey)

	resp, err := models.PostJSONRaw(ctx, g.httpClient, url, nil, geminiReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, types.NewAPIError(fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(body)), nil)
	}

	chunks := make(chan types.ResponseChunk)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		// Use the shared SSE decoder (models.SSEDecoder) instead of a
		// provider-local copy.
		// 使用共享 SSE 解码器（models.SSEDecoder），而非 provider 本地拷贝。
		decoder := models.NewSSEDecoder(resp.Body)
		for {
			data, err := decoder.Next()
			if err != nil {
				if err != io.EOF {
					chunks <- types.ResponseChunk{
						Done:  true,
						Error: err,
					}
				} else {
					chunks <- types.ResponseChunk{Done: true}
				}
				return
			}

			var geminiResp GeminiResponse
			if err := json.Unmarshal(data, &geminiResp); err != nil {
				continue
			}

			chunk := g.convertToChunk(&geminiResp)
			select {
			case chunks <- chunk:
			case <-ctx.Done():
				chunks <- types.ResponseChunk{
					Done:  true,
					Error: ctx.Err(),
				}
				return
			}
		}
	}()

	return chunks, nil
}

// buildGeminiRequest converts InvokeRequest to Gemini API request

// convertResponse converts Gemini response to ModelResponse
