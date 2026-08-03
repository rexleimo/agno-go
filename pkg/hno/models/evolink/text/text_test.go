package text

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestTextInvoke(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "c1", "model": "evo-gpt-4o",
			"choices": []any{map[string]any{"message": map[string]any{"content": "hello"}}},
			"usage":   map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	m, err := New("evo-gpt-4o", Config{APIKey: "k", BaseURL: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	resp, err := m.Invoke(context.Background(), &models.InvokeRequest{Messages: []*types.Message{types.NewUserMessage("hi")}})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if resp.Content != "hello" {
		t.Fatalf("want hello got %q", resp.Content)
	}
}
