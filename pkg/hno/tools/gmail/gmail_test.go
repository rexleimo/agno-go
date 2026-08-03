package gmail

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMarkAsRead(t *testing.T) {
	var captured modifyRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gmail/v1/users/me/messages/ABC123/modify" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer token" {
			t.Fatalf("unexpected auth header %s", auth)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		resp := modifyResponse{ID: "ABC123"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tk, err := New(Config{
		BaseURL:     server.URL,
		AccessToken: "token",
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	res, err := tk.Execute(context.Background(), "gmail_mark_as_read", map[string]interface{}{
		"message_id": "ABC123",
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if len(captured.RemoveLabels) != 1 || captured.RemoveLabels[0] != "UNREAD" {
		t.Fatalf("expected UNREAD removal, got %v", captured.RemoveLabels)
	}

	data := res.(map[string]interface{})
	if data["status"] != "read" {
		t.Fatalf("unexpected status %v", data["status"])
	}
}
