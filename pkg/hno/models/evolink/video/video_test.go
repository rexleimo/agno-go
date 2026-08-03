package video

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

func TestVideoInvokeDefaultModel(t *testing.T) {
	var gotModel string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/videos/generations", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if v, ok := payload["model"].(string); ok {
			gotModel = v
		}
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"task_id": "tv1"})
	})
	mux.HandleFunc("/v1/tasks/tv1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "tv1", "status": "completed", "data": map[string]interface{}{"url": "https://example.com"}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	m, err := New("sora", Config{APIKey: "k", BaseURL: srv.URL, Timeout: 5 * time.Second, Model: ModelSora2})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	resp, err := m.Invoke(context.Background(), &models.InvokeRequest{Messages: []*types.Message{types.NewUserMessage("hello")}})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if resp.ID != "tv1" {
		t.Fatalf("expected task id tv1, got %s", resp.ID)
	}
	if gotModel != string(ModelSora2) {
		t.Fatalf("expected model %s, got %s", ModelSora2, gotModel)
	}
}

func TestVideoInvokeModelOverride(t *testing.T) {
	var gotModel string
	var gotImages []interface{}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/videos/generations", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		if v, ok := payload["model"].(string); ok {
			gotModel = v
		}
		if imgs, ok := payload["image_urls"].([]interface{}); ok {
			gotImages = imgs
		}
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"task_id": "tv2"})
	})
	mux.HandleFunc("/v1/tasks/tv2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "tv2", "status": "completed"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	m, err := New("default", Config{APIKey: "k", BaseURL: srv.URL, Timeout: 5 * time.Second, Model: ModelSora2})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	req := &models.InvokeRequest{
		Messages: []*types.Message{types.NewUserMessage("render")},
		Extra: map[string]interface{}{
			"model":      string(ModelVeo31Fast),
			"image_urls": []string{"https://example.com/frame.png"},
		},
	}
	if _, err := m.Invoke(context.Background(), req); err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if gotModel != string(ModelVeo31Fast) {
		t.Fatalf("expected override %s, got %s", ModelVeo31Fast, gotModel)
	}
	if len(gotImages) != 1 {
		t.Fatalf("expected image_urls forwarded, got %#v", gotImages)
	}
}
