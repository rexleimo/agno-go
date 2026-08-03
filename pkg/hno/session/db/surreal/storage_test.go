package surreal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/session"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newClientWithTransport(t *testing.T, cfg ClientConfig, fn roundTripFunc) *Client {
	t.Helper()
	cfg.HTTPClient = &http.Client{Transport: fn}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	return client
}

func jsonResponse(body interface{}) *http.Response {
	payload, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(payload)),
	}
}

func TestStorage_Create(t *testing.T) {
	var capturedRequest *http.Request
	var requestPayload map[string]interface{}

	handler := func(req *http.Request) (*http.Response, error) {
		capturedRequest = req.Clone(context.Background())
		defer req.Body.Close()
		if err := json.NewDecoder(req.Body).Decode(&requestPayload); err != nil {
			t.Fatalf("failed to decode request payload: %v", err)
		}
		response := []map[string]interface{}{
			{
				"status": "OK",
				"result": []map[string]interface{}{
					{
						"session_id": "session-1",
						"agent_id":   "agent-1",
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z",
					},
				},
			},
		}
		return jsonResponse(response), nil
	}

	client := newClientWithTransport(t, ClientConfig{
		BaseURL:   "http://mock.local",
		Namespace: "test-ns",
		Database:  "test-db",
		Username:  "root",
		Password:  "pass",
	}, handler)

	storage, err := NewStorage(client, nil)
	if err != nil {
		t.Fatalf("NewStorage error: %v", err)
	}

	sess := session.NewSession("session-1", "agent-1")
	if err := storage.Create(context.Background(), sess); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	if capturedRequest.URL.Path != "/sql" {
		t.Fatalf("expected /sql path, got %s", capturedRequest.URL.Path)
	}
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("root:pass"))
	if capturedRequest.Header.Get("Authorization") != expectedAuth {
		t.Fatalf("unexpected auth header: %s", capturedRequest.Header.Get("Authorization"))
	}
	if capturedRequest.Header.Get("NS") != "test-ns" || capturedRequest.Header.Get("DB") != "test-db" {
		t.Fatalf("missing namespace or database headers")
	}

	query, _ := requestPayload["query"].(string)
	if !strings.Contains(query, "type::thing('sessions'") {
		t.Fatalf("query missing type::thing: %s", query)
	}

	vars, _ := requestPayload["vars"].(map[string]interface{})
	if vars["session_id"] != "session-1" {
		t.Fatalf("unexpected session_id var: %v", vars["session_id"])
	}
	data, _ := vars["data"].(map[string]interface{})
	if data["agent_id"] != "agent-1" {
		t.Fatalf("unexpected payload agent_id: %v", data["agent_id"])
	}
}

func TestStorage_Get(t *testing.T) {
	handler := func(req *http.Request) (*http.Response, error) {
		response := []map[string]interface{}{
			{
				"status": "OK",
				"result": []map[string]interface{}{
					{
						"session_id": "session-2",
						"agent_id":   "agent-2",
						"user_id":    "user-1",
						"created_at": "2024-01-02T00:00:00Z",
						"updated_at": "2024-01-02T00:00:00Z",
					},
				},
			},
		}
		return jsonResponse(response), nil
	}

	client := newClientWithTransport(t, ClientConfig{BaseURL: "http://mock.local"}, handler)
	storage, err := NewStorage(client, nil)
	if err != nil {
		t.Fatalf("NewStorage error: %v", err)
	}

	sess, err := storage.Get(context.Background(), "session-2")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if sess.AgentID != "agent-2" || sess.UserID != "user-1" {
		t.Fatalf("unexpected session contents: %+v", sess)
	}
}

func TestStorage_ListWithFilters(t *testing.T) {
	var receivedQuery string
	handler := func(req *http.Request) (*http.Response, error) {
		defer req.Body.Close()
		var payload map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}
		receivedQuery, _ = payload["query"].(string)

		response := []map[string]interface{}{
			{
				"status": "OK",
				"result": []map[string]interface{}{
					{
						"session_id": "session-3",
						"agent_id":   "agent-1",
						"user_id":    "user-1",
						"created_at": "2024-01-03T00:00:00Z",
						"updated_at": "2024-01-03T01:00:00Z",
					},
				},
			},
		}
		return jsonResponse(response), nil
	}

	client := newClientWithTransport(t, ClientConfig{BaseURL: "http://mock.local"}, handler)
	storage, _ := NewStorage(client, nil)

	results, err := storage.List(context.Background(), map[string]interface{}{
		"agent_id": "agent-1",
		"user_id":  "user-1",
	})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 session, got %d", len(results))
	}
	if !strings.Contains(receivedQuery, "agent_id = $agent_id") || !strings.Contains(receivedQuery, "user_id = $user_id") {
		t.Fatalf("query missing filters: %s", receivedQuery)
	}
}

func TestStorage_Delete(t *testing.T) {
	handler := func(req *http.Request) (*http.Response, error) {
		response := []map[string]interface{}{
			{
				"status": "OK",
				"result": []map[string]interface{}{
					{"id": "sessions:session-4"},
				},
			},
		}
		return jsonResponse(response), nil
	}

	client := newClientWithTransport(t, ClientConfig{BaseURL: "http://mock.local"}, handler)
	storage, _ := NewStorage(client, nil)

	if err := storage.Delete(context.Background(), "session-4"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
}

func TestStorage_BulkUpsertSessions(t *testing.T) {
	var requestPayload map[string]interface{}
	handler := func(req *http.Request) (*http.Response, error) {
		defer req.Body.Close()
		if err := json.NewDecoder(req.Body).Decode(&requestPayload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}
		response := []map[string]interface{}{
			{
				"status": "OK",
				"result": nil,
			},
		}
		return jsonResponse(response), nil
	}

	client := newClientWithTransport(t, ClientConfig{BaseURL: "http://mock.local"}, handler)
	storage, _ := NewStorage(client, nil)

	sessions := []*session.Session{
		session.NewSession("bulk-1", "agent-x"),
		session.NewSession("bulk-2", "agent-x"),
	}

	if err := storage.BulkUpsertSessions(context.Background(), sessions); err != nil {
		t.Fatalf("BulkUpsertSessions error: %v", err)
	}

	query, _ := requestPayload["query"].(string)
	if !strings.Contains(query, "FOR $item IN $records") {
		t.Fatalf("expected FOR loop query, got %s", query)
	}
	vars, _ := requestPayload["vars"].(map[string]interface{})
	if _, ok := vars["records"]; !ok {
		t.Fatalf("missing records var in payload")
	}
}

func TestStorage_Metrics(t *testing.T) {
	callCount := 0
	handler := func(req *http.Request) (*http.Response, error) {
		callCount++
		defer req.Body.Close()
		var payload map[string]interface{}
		_ = json.NewDecoder(req.Body).Decode(&payload)
		query, _ := payload["query"].(string)

		switch {
		case strings.Contains(query, "time::now() - 24h"):
			return jsonResponse([]map[string]interface{}{
				{"status": "OK", "result": []map[string]interface{}{{"total": 2}}},
			}), nil
		case strings.Contains(query, "time::now() - 1h"):
			return jsonResponse([]map[string]interface{}{
				{"status": "OK", "result": []map[string]interface{}{{"total": 1}}},
			}), nil
		default:
			return jsonResponse([]map[string]interface{}{
				{"status": "OK", "result": []map[string]interface{}{{"total": 5}}},
			}), nil
		}
	}

	client := newClientWithTransport(t, ClientConfig{BaseURL: "http://mock.local"}, handler)
	storage, _ := NewStorage(client, nil)

	metrics, err := storage.Metrics(context.Background())
	if err != nil {
		t.Fatalf("Metrics error: %v", err)
	}
	if metrics.TotalSessions != 5 || metrics.ActiveLast24h != 2 || metrics.UpdatedLastHour != 1 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
	if callCount != 3 {
		t.Fatalf("expected 3 calls, got %d", callCount)
	}
}

func TestApplyPayloadToSession(t *testing.T) {
	payload := map[string]interface{}{
		"id":         "sessions:test-1",
		"session_id": "test-1",
		"agent_id":   "agent-1",
		"created_at": "2024-01-04T00:00:00Z",
		"updated_at": "2024-01-04T00:10:00Z",
		"runs": []map[string]interface{}{
			{"content": "hello"},
		},
	}

	var sess session.Session
	if err := applyPayloadToSession(payload, &sess); err != nil {
		t.Fatalf("applyPayloadToSession error: %v", err)
	}
	if sess.SessionID != "test-1" || sess.AgentID != "agent-1" {
		t.Fatalf("unexpected session: %+v", sess)
	}
	if len(sess.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(sess.Runs))
	}
}

func TestSessionToPayload(t *testing.T) {
	sess := session.NewSession("payload-1", "agent-a")
	sess.Metadata = map[string]interface{}{"hello": "world"}
	sess.Runs = []*agent.RunOutput{
		{Content: "hi"},
	}

	payload, err := sessionToPayload(sess, "sessions")
	if err != nil {
		t.Fatalf("sessionToPayload error: %v", err)
	}
	if payload["session_id"] != "payload-1" {
		t.Fatalf("missing session_id")
	}
	if _, ok := payload["metadata"].(map[string]interface{}); !ok {
		t.Fatalf("metadata not mapped")
	}
	if payload["id"] != "sessions:payload-1" {
		t.Fatalf("unexpected id: %v", payload["id"])
	}
}
