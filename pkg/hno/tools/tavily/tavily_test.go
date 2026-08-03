package tavily

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/search" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var payload searchRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.APIKey != "tavily-key" {
			t.Fatalf("expected api key")
		}
		resp := searchResponse{
			Answer: "The answer",
			Results: []map[string]interface{}{
				{"title": "Result 1"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tk, err := New(Config{
		APIKey:  "tavily-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	resp, err := tk.Execute(context.Background(), "tavily_search", map[string]interface{}{
		"query": "hello world",
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	data := resp.(map[string]interface{})
	if data["answer"] != "The answer" {
		t.Fatalf("unexpected answer %v", data["answer"])
	}
}

func TestReader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/reader" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var payload readerRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if !payload.Extract {
			t.Fatalf("expected extract true")
		}
		resp := readerResponse{
			Results: []map[string]interface{}{
				{"content": "Article text"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tk, err := New(Config{
		APIKey:  "tavily-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	resp, err := tk.Execute(context.Background(), "tavily_reader", map[string]interface{}{
		"query": "https://example.com",
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	data := resp.(map[string]interface{})
	results := data["results"].([]map[string]interface{})
	if results[0]["content"] != "Article text" {
		t.Fatalf("unexpected content %v", results[0]["content"])
	}
}
