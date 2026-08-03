package vllm

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestVLLM_EmbedSingleAndBatch(t *testing.T) {
    // Mock OpenAI-compatible /v1/embeddings endpoint
    mux := http.NewServeMux()
    mux.HandleFunc("/v1/embeddings", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _ = r.ParseForm()
        // Return two embeddings of small dimension for determinism
        resp := map[string]interface{}{
            "data": []map[string]interface{}{
                {"index": 0, "embedding": []float32{0.1, 0.2, 0.3}},
                {"index": 1, "embedding": []float32{0.4, 0.5, 0.6}},
            },
            "model": "test-embed",
        }
        json.NewEncoder(w).Encode(resp)
    })
    srv := httptest.NewServer(mux)
    defer srv.Close()

    e, err := New(Config{Model: "test-embed", BaseURL: srv.URL + "/v1"})
    if err != nil {
        t.Fatalf("New() error = %v", err)
    }

    // Single
    emb, err := e.EmbedSingle(context.Background(), "hello")
    if err != nil {
        t.Fatalf("EmbedSingle error = %v", err)
    }
    if len(emb) != 3 || emb[0] != 0.1 {
        t.Fatalf("unexpected single embedding: %#v", emb)
    }

    // Batch
    embs, err := e.Embed(context.Background(), []string{"a", "b"})
    if err != nil {
        t.Fatalf("Embed error = %v", err)
    }
    if len(embs) != 2 || len(embs[1]) != 3 || embs[1][2] != 0.6 {
        t.Fatalf("unexpected batch embeddings: %#v", embs)
    }
}

func TestVLLM_Error(t *testing.T) {
    mux := http.NewServeMux()
    mux.HandleFunc("/v1/embeddings", func(w http.ResponseWriter, r *http.Request) {
        http.Error(w, "nope", http.StatusBadRequest)
    })
    srv := httptest.NewServer(mux)
    defer srv.Close()

    e, err := New(Config{Model: "test-embed", BaseURL: srv.URL + "/v1"})
    if err != nil {
        t.Fatalf("New() error = %v", err)
    }
    if _, err := e.EmbedSingle(context.Background(), "x"); err == nil {
        t.Fatalf("expected error from server")
    }
}

