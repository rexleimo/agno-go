package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIEmbedding implements the EmbeddingFunction interface using OpenAI's API
type OpenAIEmbedding struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// Config holds configuration for OpenAI embeddings
type Config struct {
	// APIKey for OpenAI API
	APIKey string

	// Model to use (default: text-embedding-3-small)
	// Options: text-embedding-3-small, text-embedding-3-large, text-embedding-ada-002
	Model string

	// BaseURL for OpenAI API (default: https://api.openai.com/v1)
	BaseURL string

	// HTTPClient to use for requests (optional)
	HTTPClient *http.Client
}

// embeddingRequest represents the request to OpenAI embeddings API
type embeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// embeddingResponse represents the response from OpenAI embeddings API
type embeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// errorResponse represents an error response from OpenAI API
type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// New creates a new OpenAI embedding function
func New(config Config) (*OpenAIEmbedding, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Set defaults
	if config.Model == "" {
		config.Model = "text-embedding-3-small"
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &OpenAIEmbedding{
		apiKey:     config.APIKey,
		model:      config.Model,
		baseURL:    config.BaseURL,
		httpClient: config.HTTPClient,
	}, nil
}

// Embed generates embeddings for multiple texts
func (e *OpenAIEmbedding) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// OpenAI API has a limit on batch size, split if necessary
	const maxBatchSize = 2048
	if len(texts) > maxBatchSize {
		// Process in batches
		var allEmbeddings [][]float32
		for i := 0; i < len(texts); i += maxBatchSize {
			end := i + maxBatchSize
			if end > len(texts) {
				end = len(texts)
			}
			batch := texts[i:end]
			embeddings, err := e.embed(ctx, batch)
			if err != nil {
				return nil, fmt.Errorf("failed to embed batch %d-%d: %w", i, end, err)
			}
			allEmbeddings = append(allEmbeddings, embeddings...)
		}
		return allEmbeddings, nil
	}

	return e.embed(ctx, texts)
}

// embed is the internal method that calls OpenAI API
func (e *OpenAIEmbedding) embed(ctx context.Context, texts []string) ([][]float32, error) {
	// Create request
	reqBody := embeddingRequest{
		Input: texts,
		Model: e.model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := e.baseURL + "/embeddings"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	// Send request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("API error: %s (type: %s)", errResp.Error.Message, errResp.Error.Type)
	}

	// Parse response
	var embResp embeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract embeddings in the correct order
	embeddings := make([][]float32, len(texts))
	for _, data := range embResp.Data {
		if data.Index >= 0 && data.Index < len(texts) {
			embeddings[data.Index] = data.Embedding
		}
	}

	// Verify all embeddings were generated
	for i, emb := range embeddings {
		if len(emb) == 0 {
			return nil, fmt.Errorf("missing embedding for text at index %d", i)
		}
	}

	return embeddings, nil
}

// EmbedSingle generates embedding for a single text
func (e *OpenAIEmbedding) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding generated")
	}
	return embeddings[0], nil
}

// GetModel returns the model name being used
func (e *OpenAIEmbedding) GetModel() string {
	return e.model
}

// GetDimensions returns the expected embedding dimensions for the model
func (e *OpenAIEmbedding) GetDimensions() int {
	switch e.model {
	case "text-embedding-3-small":
		return 1536
	case "text-embedding-3-large":
		return 3072
	case "text-embedding-ada-002":
		return 1536
	default:
		return 1536 // Default to most common size
	}
}
