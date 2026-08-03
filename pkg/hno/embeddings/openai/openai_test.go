package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				APIKey: "test-api-key",
				Model:  "text-embedding-3-small",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: Config{
				Model: "text-embedding-3-small",
			},
			wantErr: true,
		},
		{
			name: "default model",
			config: Config{
				APIKey: "test-api-key",
			},
			wantErr: false,
		},
		{
			name: "custom base URL",
			config: Config{
				APIKey:  "test-api-key",
				BaseURL: "https://custom.openai.com/v1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emb, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if emb == nil {
					t.Error("New() returned nil embedding")
				}
				if emb.apiKey != tt.config.APIKey {
					t.Errorf("apiKey = %v, want %v", emb.apiKey, tt.config.APIKey)
				}
				// Check default model
				if tt.config.Model == "" && emb.model != "text-embedding-3-small" {
					t.Errorf("default model = %v, want text-embedding-3-small", emb.model)
				}
			}
		})
	}
}

func TestGetDimensions(t *testing.T) {
	tests := []struct {
		model string
		want  int
	}{
		{"text-embedding-3-small", 1536},
		{"text-embedding-3-large", 3072},
		{"text-embedding-ada-002", 1536},
		{"unknown-model", 1536}, // default
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			emb := &OpenAIEmbedding{model: tt.model}
			if got := emb.GetDimensions(); got != tt.want {
				t.Errorf("GetDimensions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmbed(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("Invalid authorization header: %s", r.Header.Get("Authorization"))
		}

		// Parse request
		var req embeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Create mock response
		resp := embeddingResponse{
			Object: "list",
			Model:  req.Model,
			Data: make([]struct {
				Object    string    `json:"object"`
				Index     int       `json:"index"`
				Embedding []float32 `json:"embedding"`
			}, len(req.Input)),
		}

		for i := range req.Input {
			resp.Data[i].Object = "embedding"
			resp.Data[i].Index = i
			// Generate mock embedding (1536 dimensions)
			resp.Data[i].Embedding = make([]float32, 1536)
			for j := range resp.Data[i].Embedding {
				resp.Data[i].Embedding[j] = float32(i*1536+j) * 0.001
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create embedding function with mock server
	emb, err := New(Config{
		APIKey:  "test-api-key",
		Model:   "text-embedding-3-small",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create embedding function: %v", err)
	}

	ctx := context.Background()

	t.Run("single text", func(t *testing.T) {
		embeddings, err := emb.Embed(ctx, []string{"Hello, world!"})
		if err != nil {
			t.Fatalf("Embed() error = %v", err)
		}

		if len(embeddings) != 1 {
			t.Errorf("Expected 1 embedding, got %d", len(embeddings))
		}

		if len(embeddings[0]) != 1536 {
			t.Errorf("Expected embedding dimension 1536, got %d", len(embeddings[0]))
		}
	})

	t.Run("multiple texts", func(t *testing.T) {
		texts := []string{
			"First text",
			"Second text",
			"Third text",
		}

		embeddings, err := emb.Embed(ctx, texts)
		if err != nil {
			t.Fatalf("Embed() error = %v", err)
		}

		if len(embeddings) != len(texts) {
			t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
		}

		for i, embedding := range embeddings {
			if len(embedding) != 1536 {
				t.Errorf("Embedding %d dimension = %d, want 1536", i, len(embedding))
			}
		}
	})

	t.Run("empty input", func(t *testing.T) {
		embeddings, err := emb.Embed(ctx, []string{})
		if err != nil {
			t.Fatalf("Embed() error = %v", err)
		}

		if len(embeddings) != 0 {
			t.Errorf("Expected 0 embeddings, got %d", len(embeddings))
		}
	})
}

func TestEmbedSingle(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := embeddingResponse{
			Object: "list",
			Model:  "text-embedding-3-small",
			Data: []struct {
				Object    string    `json:"object"`
				Index     int       `json:"index"`
				Embedding []float32 `json:"embedding"`
			}{
				{
					Object:    "embedding",
					Index:     0,
					Embedding: make([]float32, 1536),
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	emb, err := New(Config{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create embedding function: %v", err)
	}

	ctx := context.Background()

	embedding, err := emb.EmbedSingle(ctx, "Test text")
	if err != nil {
		t.Fatalf("EmbedSingle() error = %v", err)
	}

	if len(embedding) != 1536 {
		t.Errorf("Expected embedding dimension 1536, got %d", len(embedding))
	}
}

func TestEmbedError(t *testing.T) {
	// Create a mock server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		errResp := errorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			}{
				Message: "Invalid API key",
				Type:    "invalid_request_error",
				Code:    "invalid_api_key",
			},
		}
		json.NewEncoder(w).Encode(errResp)
	}))
	defer server.Close()

	emb, err := New(Config{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create embedding function: %v", err)
	}

	ctx := context.Background()

	_, err = emb.Embed(ctx, []string{"Test"})
	if err == nil {
		t.Error("Expected error for invalid API key, got nil")
	}
}

func TestEmbedIntegration(t *testing.T) {
	// This test requires a real OpenAI API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	emb, err := New(Config{
		APIKey: apiKey,
		Model:  "text-embedding-3-small",
	})
	if err != nil {
		t.Fatalf("Failed to create embedding function: %v", err)
	}

	ctx := context.Background()

	t.Run("real API call", func(t *testing.T) {
		texts := []string{
			"The quick brown fox jumps over the lazy dog",
			"Machine learning is a subset of artificial intelligence",
		}

		embeddings, err := emb.Embed(ctx, texts)
		if err != nil {
			t.Fatalf("Embed() error = %v", err)
		}

		if len(embeddings) != len(texts) {
			t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
		}

		expectedDim := emb.GetDimensions()
		for i, embedding := range embeddings {
			if len(embedding) != expectedDim {
				t.Errorf("Embedding %d dimension = %d, want %d", i, len(embedding), expectedDim)
			}

			// Verify embeddings are normalized (L2 norm should be close to 1)
			var sum float32
			for _, val := range embedding {
				sum += val * val
			}
			// OpenAI embeddings are normalized, but allow some floating point error
			if sum < 0.9 || sum > 1.1 {
				t.Errorf("Embedding %d L2 norm = %f, expected ~1.0", i, sum)
			}
		}
	})

	t.Run("single text", func(t *testing.T) {
		embedding, err := emb.EmbedSingle(ctx, "Hello, world!")
		if err != nil {
			t.Fatalf("EmbedSingle() error = %v", err)
		}

		expectedDim := emb.GetDimensions()
		if len(embedding) != expectedDim {
			t.Errorf("Embedding dimension = %d, want %d", len(embedding), expectedDim)
		}
	})
}

func TestGetModel(t *testing.T) {
	emb := &OpenAIEmbedding{model: "text-embedding-3-large"}
	if got := emb.GetModel(); got != "text-embedding-3-large" {
		t.Errorf("GetModel() = %v, want text-embedding-3-large", got)
	}
}
