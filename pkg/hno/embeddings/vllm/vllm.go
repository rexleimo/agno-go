package vllm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Embedding implements vectordb.EmbeddingFunction for a VLLM server
type Embedding struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// Config for VLLM embeddings provider
type Config struct {
	// Model to use (required)
	Model string
	// BaseURL for VLLM server (OpenAI-compatible). Default: http://localhost:8000/v1
	BaseURL string
	// APIKey optional bearer for remote deployments
	APIKey string
	// HTTPClient optional custom client
	HTTPClient *http.Client
}

type embeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type embeddingResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
}

// New creates a VLLM embedding function (OpenAI-compatible /v1/embeddings)
func New(cfg Config) (*Embedding, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:8000/v1"
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Embedding{
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		baseURL:    cfg.BaseURL,
		httpClient: cfg.HTTPClient,
	}, nil
}

// Embed generates embeddings for multiple texts
func (e *Embedding) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	reqBody := embeddingRequest{Input: texts, Model: e.model}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/embeddings", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("vllm embeddings failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	var er embeddingResponse
	if err := json.Unmarshal(body, &er); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	embeddings := make([][]float32, len(texts))
	for _, d := range er.Data {
		if d.Index >= 0 && d.Index < len(texts) {
			embeddings[d.Index] = d.Embedding
		}
	}
	for i, emb := range embeddings {
		if len(emb) == 0 {
			return nil, fmt.Errorf("missing embedding at index %d", i)
		}
	}
	return embeddings, nil
}

// EmbedSingle generates an embedding for one text
func (e *Embedding) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	res, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return res[0], nil
}
