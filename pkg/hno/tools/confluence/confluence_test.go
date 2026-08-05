package confluence

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newConfluenceTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser@example.com:test-token"))
		if r.Header.Get("Authorization") != expectedAuth {
			http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/wiki/api/v2/spaces":
			_, _ = w.Write([]byte(`{"results":[{"id":"1","key":"DOC","name":"Documentation","type":"global"}]}`))
		case "/wiki/rest/api/content/search":
			cql := r.URL.Query().Get("cql")
			if !strings.Contains(cql, `text~"documentation"`) {
				http.Error(w, `{"message":"missing CQL"}`, http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"results":[{"id":"123","title":"Getting Started","space":{"key":"DOC"},"version":{"number":2}}]}`))
		case "/wiki/rest/api/content/123":
			_, _ = w.Write([]byte(`{"id":"123","title":"Getting Started","body":{"storage":{"value":"<p>Real page content</p>","representation":"storage"}},"version":{"number":2}}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func confluenceArgs() map[string]interface{} {
	return map[string]interface{}{
		"base_url":  "http://unused.example/wiki",
		"username":  "testuser@example.com",
		"api_token": "test-token",
	}
}

func TestConfluenceToolkit_ListSpaces(t *testing.T) {
	server := newConfluenceTestServer(t)
	defer server.Close()
	args := confluenceArgs()
	args["base_url"] = server.URL + "/wiki"

	result, err := New().Execute(context.Background(), "list_spaces", args)
	if err != nil {
		t.Fatalf("list_spaces failed: %v", err)
	}
	spaces := result.(map[string]interface{})["spaces"].([]map[string]interface{})
	if len(spaces) != 1 || spaces[0]["key"] != "DOC" {
		t.Fatalf("unexpected spaces: %#v", spaces)
	}
}

func TestConfluenceToolkit_SearchPages(t *testing.T) {
	server := newConfluenceTestServer(t)
	defer server.Close()
	args := confluenceArgs()
	args["base_url"] = server.URL + "/wiki"
	args["query"] = "documentation"

	result, err := New().Execute(context.Background(), "search_pages", args)
	if err != nil {
		t.Fatalf("search_pages failed: %v", err)
	}
	resultMap := result.(map[string]interface{})
	pages := resultMap["pages"].([]map[string]interface{})
	if len(pages) != 1 || pages[0]["title"] != "Getting Started" {
		t.Fatalf("unexpected pages: %#v", pages)
	}
}

func TestConfluenceToolkit_SearchPagesWithSpaceKey(t *testing.T) {
	server := newConfluenceTestServer(t)
	defer server.Close()
	args := confluenceArgs()
	args["base_url"] = server.URL + "/wiki"
	args["query"] = "documentation"
	args["space_key"] = "DOC"

	result, err := New().Execute(context.Background(), "search_pages", args)
	if err != nil {
		t.Fatalf("search_pages with space key failed: %v", err)
	}
	if result.(map[string]interface{})["space_key"] != "DOC" {
		t.Fatalf("unexpected space key: %#v", result)
	}
}

func TestConfluenceToolkit_GetPageContent(t *testing.T) {
	server := newConfluenceTestServer(t)
	defer server.Close()
	args := confluenceArgs()
	args["base_url"] = server.URL + "/wiki"
	args["page_id"] = "123"

	result, err := New().Execute(context.Background(), "get_page_content", args)
	if err != nil {
		t.Fatalf("get_page_content failed: %v", err)
	}
	page := result.(map[string]interface{})["page"].(map[string]interface{})
	if page["id"] != "123" || page["title"] != "Getting Started" {
		t.Fatalf("unexpected page: %#v", page)
	}
}

func TestConfluenceToolkit_MissingParameters(t *testing.T) {
	if _, err := New().Execute(context.Background(), "list_spaces", map[string]interface{}{
		"username": "testuser", "api_token": "token",
	}); err == nil {
		t.Error("expected missing base_url error")
	}
	if _, err := New().Execute(context.Background(), "search_pages", map[string]interface{}{
		"base_url": "https://example.test/wiki", "username": "testuser", "api_token": "token",
	}); err == nil {
		t.Error("expected missing query error")
	}
}

func TestConfluenceToolkit_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"server error"}`, http.StatusInternalServerError)
	}))
	defer server.Close()
	args := confluenceArgs()
	args["base_url"] = server.URL
	if _, err := New().Execute(context.Background(), "list_spaces", args); err == nil {
		t.Fatal("expected API error")
	}
}

func TestConfluenceToolkit_New(t *testing.T) {
	tk := New()
	if tk == nil || len(tk.Functions()) != 3 {
		t.Fatalf("expected three registered functions")
	}
}
