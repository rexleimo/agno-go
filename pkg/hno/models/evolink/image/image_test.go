package image

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestImageInvoke(t *testing.T) {
	var calls int32
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/images/generations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"task_id": "timg1"})
	})
	mux.HandleFunc("/v1/tasks/timg1", func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&calls, 1)
		w.Header().Set("content-type", "application/json")
		if c < 2 {
			json.NewEncoder(w).Encode(map[string]any{"id": "timg1", "status": "processing"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"id": "timg1", "status": "completed", "data": map[string]any{"images": []any{"https://ex/i1.png"}}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	m, err := New("evo-gpt-4o-images", Config{APIKey: "k", BaseURL: srv.URL, Timeout: 5 * time.Second, Size: "1:1", N: 1, Model: ModelGPT4O})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	resp, err := m.Invoke(context.Background(), &models.InvokeRequest{Messages: []*types.Message{types.NewUserMessage("a cat")}})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if resp.ID != "timg1" {
		t.Fatalf("unexpected task id: %s", resp.ID)
	}
}

func TestImageInvokeModelOverride(t *testing.T) {
	var gotModel string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/images/generations", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if v, ok := payload["model"].(string); ok {
			gotModel = v
		}
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"task_id": "timg2"})
	})
	mux.HandleFunc("/v1/tasks/timg2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "timg2", "status": "completed"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	m, err := New("wan2.5", Config{APIKey: "k", BaseURL: srv.URL, Timeout: 5 * time.Second, Model: ModelGPT4O})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	req := &models.InvokeRequest{
		Messages: []*types.Message{types.NewUserMessage("render")},
		Extra:    map[string]interface{}{"model": string(ModelWan25TextToImage)},
	}
	if _, err := m.Invoke(context.Background(), req); err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if gotModel != string(ModelWan25TextToImage) {
		t.Fatalf("expected model override %s, got %s", ModelWan25TextToImage, gotModel)
	}
}
