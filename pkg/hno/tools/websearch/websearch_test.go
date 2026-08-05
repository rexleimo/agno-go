package websearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newWebSearchTestToolkit(t *testing.T) *WebSearchToolkit {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search" {
			var request map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
				return
			}
			if request["api_key"] != "test-key" {
				http.Error(w, `{"error":"bad key"}`, http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"results":[{"title":"Real result","url":"https://example.com/real","content":"Result content","score":0.91}]}`))
			return
		}
		if r.URL.Path == "/content" {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("real fetched content"))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	return NewWithConfig(Config{APIKey: "test-key", BaseURL: server.URL})
}

func TestWebSearchToolkit_New(t *testing.T) {
	tk := New()
	if tk == nil || tk.Name() != "web_search" || len(tk.Functions()) != 2 {
		t.Fatalf("unexpected toolkit initialization")
	}
}

func TestWebSearchToolkit_WebSearch(t *testing.T) {
	tk := newWebSearchTestToolkit(t)
	result, err := tk.Execute(context.Background(), "web_search", map[string]interface{}{"query": "test query"})
	if err != nil {
		t.Fatalf("web_search failed: %v", err)
	}
	resultMap := result.(map[string]interface{})
	if resultMap["query"] != "test query" {
		t.Fatalf("unexpected query: %#v", resultMap["query"])
	}
	results := resultMap["results"].([]map[string]interface{})
	if len(results) != 1 || results[0]["title"] != "Real result" {
		t.Fatalf("unexpected API results: %#v", results)
	}
}

func TestWebSearchToolkit_WebSearchWithMaxResults(t *testing.T) {
	tk := newWebSearchTestToolkit(t)
	result, err := tk.Execute(context.Background(), "web_search", map[string]interface{}{
		"query": "test query", "max_results": 1.0,
	})
	if err != nil {
		t.Fatalf("web_search failed: %v", err)
	}
	if result.(map[string]interface{})["count"] != 1 {
		t.Fatalf("unexpected count: %#v", result)
	}
}

func TestWebSearchToolkit_MissingQueryAndAPIKey(t *testing.T) {
	if _, err := New().Execute(context.Background(), "web_search", map[string]interface{}{}); err == nil {
		t.Error("expected missing query error")
	}
	if _, err := NewWithConfig(Config{BaseURL: "https://example.test"}).Execute(context.Background(), "web_search", map[string]interface{}{"query": "test"}); err == nil {
		t.Error("expected missing API key error")
	}
}

func TestWebSearchToolkit_ExtractWebContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("real fetched content"))
	}))
	defer server.Close()

	result, err := New().Execute(context.Background(), "extract_web_content", map[string]interface{}{"url": server.URL})
	if err != nil {
		t.Fatalf("extract_web_content failed: %v", err)
	}
	resultMap := result.(map[string]interface{})
	if resultMap["url"] != server.URL || resultMap["status"] != 200 || resultMap["content"] != "real fetched content" {
		t.Fatalf("unexpected extracted content: %#v", resultMap)
	}
}

func TestWebSearchToolkit_ExtractWebContentValidation(t *testing.T) {
	tk := New()
	if _, err := tk.Execute(context.Background(), "extract_web_content", map[string]interface{}{"url": "not-a-valid-url"}); err == nil {
		t.Error("expected invalid URL error")
	}
	if _, err := tk.Execute(context.Background(), "extract_web_content", map[string]interface{}{}); err == nil {
		t.Error("expected missing URL error")
	}
}
