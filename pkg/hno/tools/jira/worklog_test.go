package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddWorklog(t *testing.T) {
	var captured worklogRequest

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/DEMO-1/worklog" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("missing authorization header")
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		resp := worklogResponse{
			ID:   "1001",
			Self: server.URL + "/rest/api/3/worklog/1001",
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tk, err := New(Config{
		BaseURL:   server.URL,
		AuthToken: "token",
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	res, err := tk.Execute(context.Background(), "jira_add_worklog", map[string]interface{}{
		"issue_id":           "DEMO-1",
		"time_spent_seconds": 900.0,
		"started":            "2024-10-01T09:00:00.000+0000",
		"comment":            "Worked on implementation",
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if captured.TimeSpentSeconds != 900 {
		t.Fatalf("unexpected timeSpentSeconds %d", captured.TimeSpentSeconds)
	}
	if captured.Comment != "Worked on implementation" {
		t.Fatalf("unexpected comment %s", captured.Comment)
	}

	data := res.(map[string]interface{})
	if data["id"] != "1001" {
		t.Fatalf("unexpected id %v", data["id"])
	}
}
