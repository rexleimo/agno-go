package youtube

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func newYouTubeTestToolkit(t *testing.T) *YouTubeToolkit {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "test-key" {
			http.Error(w, `{"error":{"message":"bad key"}}`, http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/search":
			if r.URL.Query().Get("q") == "" {
				http.Error(w, `{"error":"missing query"}`, http.StatusBadRequest)
				return
			}
			if r.URL.Query().Get("maxResults") == "" {
				http.Error(w, `{"error":"missing maxResults"}`, http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"items":[{"id":{"videoId":"video-1"},"snippet":{"publishedAt":"2024-01-01T00:00:00Z","channelId":"channel-1","title":"Real video","description":"A real API result","channelTitle":"Real Channel"}}]}`))
		case "/videos":
			if r.URL.Query().Get("id") != "dQw4w9WgXcQ" {
				http.Error(w, `{"error":"unexpected id"}`, http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"items":[{"id":"dQw4w9WgXcQ","snippet":{"publishedAt":"2024-01-01T00:00:00Z","channelId":"channel-1","title":"Real video","description":"Real description","channelTitle":"Real Channel","categoryId":"27","tags":["go","api"]},"contentDetails":{"duration":"PT10M30S"},"statistics":{"viewCount":"2500","likeCount":"150","commentCount":"20"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return NewWithConfig(Config{APIKey: "test-key", BaseURL: server.URL})
}

func TestYouTubeToolkit_New(t *testing.T) {
	tk := New()
	if tk == nil || tk.Name() != "youtube" || len(tk.Functions()) != 2 {
		t.Fatalf("unexpected toolkit initialization")
	}
}

func TestYouTubeToolkit_SearchYouTube(t *testing.T) {
	tk := newYouTubeTestToolkit(t)
	result, err := tk.Execute(context.Background(), "search_youtube", map[string]interface{}{
		"query": "machine learning tutorial",
	})
	if err != nil {
		t.Fatalf("search_youtube failed: %v", err)
	}
	resultMap := result.(map[string]interface{})
	if resultMap["query"] != "machine learning tutorial" {
		t.Fatalf("unexpected query: %#v", resultMap["query"])
	}
	results := resultMap["results"].([]map[string]interface{})
	if len(results) != 1 || results[0]["video_id"] != "video-1" || results[0]["title"] != "Real video" {
		t.Fatalf("unexpected API results: %#v", results)
	}
}

func TestYouTubeToolkit_SearchYouTubeWithMaxResults(t *testing.T) {
	tk := newYouTubeTestToolkit(t)
	result, err := tk.Execute(context.Background(), "search_youtube", map[string]interface{}{
		"query": "test query", "max_results": 1.0,
	})
	if err != nil {
		t.Fatalf("search_youtube failed: %v", err)
	}
	if result.(map[string]interface{})["count"] != 1 {
		t.Fatalf("unexpected count: %#v", result)
	}
}

func TestYouTubeToolkit_GetVideoInfo(t *testing.T) {
	tk := newYouTubeTestToolkit(t)
	result, err := tk.Execute(context.Background(), "get_video_info", map[string]interface{}{
		"video_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	})
	if err != nil {
		t.Fatalf("get_video_info failed: %v", err)
	}
	info := result.(map[string]interface{})
	if info["video_id"] != "dQw4w9WgXcQ" || info["title"] != "Real video" || info["views"] != "2500" {
		t.Fatalf("unexpected video info: %#v", info)
	}
	if info["duration_seconds"] != int64(630) {
		t.Fatalf("unexpected duration: %#v", info["duration_seconds"])
	}
}

func TestYouTubeToolkit_GetVideoInfoShortURL(t *testing.T) {
	tk := newYouTubeTestToolkit(t)
	result, err := tk.Execute(context.Background(), "get_video_info", map[string]interface{}{
		"video_url": "https://youtu.be/dQw4w9WgXcQ",
	})
	if err != nil {
		t.Fatalf("get_video_info short URL failed: %v", err)
	}
	if result.(map[string]interface{})["video_id"] != "dQw4w9WgXcQ" {
		t.Fatalf("unexpected video id: %#v", result)
	}
}

func TestYouTubeToolkit_Validation(t *testing.T) {
	tk := NewWithConfig(Config{APIKey: "test-key", BaseURL: "http://127.0.0.1:" + strconv.Itoa(1)})
	if _, err := tk.Execute(context.Background(), "search_youtube", map[string]interface{}{}); err == nil {
		t.Error("expected missing query error")
	}
	if _, err := tk.Execute(context.Background(), "get_video_info", map[string]interface{}{"video_url": "https://example.com/video"}); err == nil {
		t.Error("expected non-YouTube URL error")
	}
	if _, err := tk.Execute(context.Background(), "get_video_info", map[string]interface{}{"video_url": "not-a-valid-url"}); err == nil {
		t.Error("expected invalid URL error")
	}
}

func TestYouTubeToolkit_MissingAPIKey(t *testing.T) {
	tk := NewWithConfig(Config{BaseURL: "https://example.test"})
	_, err := tk.Execute(context.Background(), "search_youtube", map[string]interface{}{"query": "test"})
	if err == nil {
		t.Fatal("expected missing API key error")
	}
}
