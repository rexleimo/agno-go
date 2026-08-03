package agentos

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/session"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

type simpleModel struct {
	models.BaseModel
}

func (m *simpleModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	return &types.ModelResponse{Content: "OK", Model: m.ID, Usage: types.Usage{TotalTokens: 1}}, nil
}

func (m *simpleModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	ch := make(chan types.ResponseChunk)
	close(ch)
	return ch, nil
}

func TestNewServer(t *testing.T) {
	server, err := NewServer(nil)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	if server == nil {
		t.Fatal("Expected non-nil server")
	}

	if server.config.Address != ":8080" {
		t.Errorf("Address = %v, want ':8080'", server.config.Address)
	}
}

func TestNewServer_WithConfig(t *testing.T) {
	storage := session.NewMemoryStorage()

	server, err := NewServer(&Config{
		Address:        ":9090",
		SessionStorage: storage,
		Debug:          true,
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	if server.config.Address != ":9090" {
		t.Errorf("Address = %v, want ':9090'", server.config.Address)
	}

	if !server.config.Debug {
		t.Error("Debug should be true")
	}
}

func TestHealthEndpoint(t *testing.T) {
	server, _ := NewServer(nil)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Status = %v, want 'ok'", response["status"])
	}
	if _, ok := response["instantiated_at"]; !ok {
		t.Fatal("expected instantiated_at in payload")
	}
}

func TestHealthEndpoint_CustomPath(t *testing.T) {
	server, _ := NewServer(&Config{HealthPath: "/health-check"})
	req, _ := http.NewRequest("GET", "/health-check", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected custom health path to return 200, got %d", w.Code)
	}
}

func TestCustomHealthRouter(t *testing.T) {
	server, _ := NewServer(nil)
	group := server.GetHealthRouter("/health-check")
	if group == nil {
		t.Fatal("expected health router group")
	}
	group.GET("", server.handleHealth)

	req, _ := http.NewRequest("GET", "/health-check", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestDocsEndpoints(t *testing.T) {
	server, _ := NewServer(nil)

	reqSpec, _ := http.NewRequest("GET", "/openapi.yaml", nil)
	wSpec := httptest.NewRecorder()
	server.router.ServeHTTP(wSpec, reqSpec)
	if wSpec.Code != http.StatusOK {
		t.Fatalf("expected openapi 200, got %d", wSpec.Code)
	}
	if !strings.Contains(wSpec.Body.String(), "openapi") {
		t.Fatal("openapi spec missing content")
	}

	reqDocs, _ := http.NewRequest("GET", "/docs", nil)
	wDocs := httptest.NewRecorder()
	server.router.ServeHTTP(wDocs, reqDocs)
	if wDocs.Code != http.StatusOK {
		t.Fatalf("expected docs 200, got %d", wDocs.Code)
	}
	if !strings.Contains(wDocs.Body.String(), "SwaggerUIBundle") {
		t.Fatal("docs html not rendered")
	}
}

func TestResyncRegistersDocs(t *testing.T) {
	server, _ := NewServer(nil)
	server.Resync()
	req, _ := http.NewRequest("GET", "/docs", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected docs to remain registered after resync")
	}
}

func TestCreateSession(t *testing.T) {
	server, _ := NewServer(nil)

	reqBody := CreateSessionRequest{
		AgentID: "test-agent",
		UserID:  "test-user",
		Name:    "Test Session",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/sessions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusCreated)
	}

	var response SessionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.SessionID == "" {
		t.Error("SessionID should not be empty")
	}

	if response.AgentID != "test-agent" {
		t.Errorf("AgentID = %v, want 'test-agent'", response.AgentID)
	}

	if response.UserID != "test-user" {
		t.Errorf("UserID = %v, want 'test-user'", response.UserID)
	}
}

func TestCreateSession_MissingAgentID(t *testing.T) {
	server, _ := NewServer(nil)

	reqBody := CreateSessionRequest{
		UserID: "test-user",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/sessions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestGetSession(t *testing.T) {
	server, _ := NewServer(nil)

	// First create a session
	sess := session.NewSession("test-session-123", "test-agent")
	server.sessionStorage.Create(context.Background(), sess)

	// Then get it
	req, _ := http.NewRequest("GET", "/api/v1/sessions/test-session-123", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response SessionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.SessionID != "test-session-123" {
		t.Errorf("SessionID = %v, want 'test-session-123'", response.SessionID)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	server, _ := NewServer(nil)

	req, _ := http.NewRequest("GET", "/api/v1/sessions/non-existent", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestUpdateSession(t *testing.T) {
	server, _ := NewServer(nil)

	// Create a session first
	sess := session.NewSession("test-session-123", "test-agent")
	sess.Name = "Original Name"
	server.sessionStorage.Create(context.Background(), sess)

	// Update it
	updateReq := UpdateSessionRequest{
		Name: "Updated Name",
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", "/api/v1/sessions/test-session-123", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response SessionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Name != "Updated Name" {
		t.Errorf("Name = %v, want 'Updated Name'", response.Name)
	}

	if response.Metadata["key"] != "value" {
		t.Error("Metadata not updated correctly")
	}
}

func TestUpdateSession_AGUIStatePersists(t *testing.T) {
	server, _ := NewServer(nil)

	// Create a session first
	sess := session.NewSession("agui-session", "agent-x")
	server.sessionStorage.Create(context.Background(), sess)

	// Update with AGUI substate
	updateReq := UpdateSessionRequest{
		State: map[string]interface{}{
			"agui": map[string]interface{}{
				"pane":    "history",
				"filters": []interface{}{"runs", "errors"},
			},
		},
	}
	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", "/api/v1/sessions/agui-session", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", w.Code, w.Body.String())
	}

	// Read back
	getReq, _ := http.NewRequest("GET", "/api/v1/sessions/agui-session", nil)
	getW := httptest.NewRecorder()
	server.router.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("unexpected get status %d", getW.Code)
	}
	var resp SessionResponse
	if err := json.Unmarshal(getW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse get response: %v", err)
	}
	agui, ok := resp.State["agui"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected agui substate present, got %#v", resp.State)
	}
	if agui["pane"].(string) != "history" {
		t.Fatalf("unexpected pane: %v", agui["pane"])
	}
}

func TestSessionSummaryEndpoints(t *testing.T) {
	server, _ := NewServer(nil)

	sess := session.NewSession("summary-session", "agent-1")
	sess.AddRun(&agent.RunOutput{
		Content: "Assistant response",
		Messages: []*types.Message{
			types.NewUserMessage("Hello"),
			types.NewAssistantMessage("Assistant response"),
		},
	})
	server.sessionStorage.Create(context.Background(), sess)

	postReq, _ := http.NewRequest("POST", "/api/v1/sessions/summary-session/summary", nil)
	postW := httptest.NewRecorder()
	server.router.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Fatalf("POST summary status = %v, want %v", postW.Code, http.StatusOK)
	}

	var postResp SessionSummaryResult
	if err := json.Unmarshal(postW.Body.Bytes(), &postResp); err != nil {
		t.Fatalf("failed to parse summary response: %v", err)
	}
	if postResp.Summary == nil || postResp.Summary.Content == "" {
		t.Fatal("expected non-empty summary content")
	}

	getReq, _ := http.NewRequest("GET", "/api/v1/sessions/summary-session/summary", nil)
	getW := httptest.NewRecorder()
	server.router.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET summary status = %v, want %v", getW.Code, http.StatusOK)
	}

	var getResp SessionSummaryResult
	if err := json.Unmarshal(getW.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to parse get summary response: %v", err)
	}
	if getResp.Summary == nil {
		t.Fatal("expected stored summary")
	}
}

func TestSessionSummaryAsync(t *testing.T) {
	server, _ := NewServer(nil)

	sess := session.NewSession("async-summary", "agent-1")
	sess.AddRun(&agent.RunOutput{Content: "First content"})
	server.sessionStorage.Create(context.Background(), sess)

	req, _ := http.NewRequest("POST", "/api/v1/sessions/async-summary/summary?async=true", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("async summary status = %v, want %v", w.Code, http.StatusAccepted)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		getReq, _ := http.NewRequest("GET", "/api/v1/sessions/async-summary/summary", nil)
		getW := httptest.NewRecorder()
		server.router.ServeHTTP(getW, getReq)

		if getW.Code == http.StatusOK {
			var resp SessionSummaryResult
			if err := json.Unmarshal(getW.Body.Bytes(), &resp); err == nil && resp.Summary != nil {
				return
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("expected async summary to be generated")
}

func TestReuseSessionEndpoint(t *testing.T) {
	server, _ := NewServer(nil)

	sess := session.NewSession("reuse-session", "agent-1")
	server.sessionStorage.Create(context.Background(), sess)

	reqBody := ReuseSessionRequest{TeamID: "team-42"}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/sessions/reuse-session/reuse", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("reuse status = %v, want %v", w.Code, http.StatusOK)
	}

	var resp SessionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse reuse response: %v", err)
	}
	if resp.TeamID != "team-42" {
		t.Fatalf("TeamID = %v, want team-42", resp.TeamID)
	}
}

func TestSessionHistoryLimit(t *testing.T) {
	server, _ := NewServer(nil)

	sess := session.NewSession("history-session", "agent-1")
	sess.AddRun(&agent.RunOutput{
		RunID:       "run-1",
		Status:      agent.RunStatusCompleted,
		StartedAt:   time.Now().Add(-2 * time.Minute),
		CompletedAt: time.Now().Add(-time.Minute),
		Messages: []*types.Message{
			types.NewUserMessage("Hi"),
			types.NewAssistantMessage("Hello"),
		},
	})
	sess.AddRun(&agent.RunOutput{
		RunID:       "run-2",
		Status:      agent.RunStatusCompleted,
		StartedAt:   time.Now().Add(-30 * time.Second),
		CompletedAt: time.Now(),
		Metadata: map[string]interface{}{
			"cache_hit": true,
		},
		Messages: []*types.Message{
			types.NewUserMessage("Status?"),
			types.NewAssistantMessage("Working on it"),
		},
	})
	server.sessionStorage.Create(context.Background(), sess)

	req, _ := http.NewRequest("GET", "/api/v1/sessions/history-session/history?num_messages=1", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("history status = %v, want %v", w.Code, http.StatusOK)
	}

	var resp struct {
		SessionID    string                  `json:"session_id"`
		Messages     []types.Message         `json:"messages"`
		StreamEvents bool                    `json:"stream_events"`
		Summary      *session.SessionSummary `json:"summary"`
		Runs         []SessionRunMetadata    `json:"runs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse history response: %v", err)
	}

	if len(resp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.Messages))
	}

	if resp.Messages[0].Content != "Working on it" {
		t.Fatalf("unexpected message content: %v", resp.Messages[0].Content)
	}
	if len(resp.Runs) == 0 {
		t.Fatalf("expected runs metadata in history response")
	}
	if !resp.Runs[len(resp.Runs)-1].CacheHit {
		t.Fatalf("expected last run cache hit metadata")
	}
}

func TestDeleteSession(t *testing.T) {
	server, _ := NewServer(nil)

	// Create a session first
	sess := session.NewSession("test-session-123", "test-agent")
	server.sessionStorage.Create(context.Background(), sess)

	// Delete it
	req, _ := http.NewRequest("DELETE", "/api/v1/sessions/test-session-123", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify it's deleted
	_, err := server.sessionStorage.Get(context.Background(), "test-session-123")
	if err != session.ErrSessionNotFound {
		t.Error("Session should be deleted")
	}
}

func TestListSessions(t *testing.T) {
	server, _ := NewServer(nil)

	// Create multiple sessions
	sess1 := session.NewSession("sess-1", "agent-1")
	sess1.UserID = "user-1"
	server.sessionStorage.Create(context.Background(), sess1)

	sess2 := session.NewSession("sess-2", "agent-1")
	sess2.UserID = "user-2"
	server.sessionStorage.Create(context.Background(), sess2)

	sess3 := session.NewSession("sess-3", "agent-2")
	sess3.UserID = "user-1"
	server.sessionStorage.Create(context.Background(), sess3)

	// List all sessions
	req, _ := http.NewRequest("GET", "/api/v1/sessions", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	count := int(response["count"].(float64))
	if count != 3 {
		t.Errorf("Count = %v, want 3", count)
	}
}

func TestListSessions_WithFilter(t *testing.T) {
	server, _ := NewServer(nil)

	// Create multiple sessions
	sess1 := session.NewSession("sess-1", "agent-1")
	sess1.UserID = "user-1"
	server.sessionStorage.Create(context.Background(), sess1)

	sess2 := session.NewSession("sess-2", "agent-1")
	sess2.UserID = "user-2"
	server.sessionStorage.Create(context.Background(), sess2)

	sess3 := session.NewSession("sess-3", "agent-2")
	sess3.UserID = "user-1"
	server.sessionStorage.Create(context.Background(), sess3)

	// Filter by agent_id
	req, _ := http.NewRequest("GET", "/api/v1/sessions?agent_id=agent-1", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	count := int(response["count"].(float64))
	if count != 2 {
		t.Errorf("Count = %v, want 2", count)
	}
}

func TestAgentRun(t *testing.T) {
	server, _ := NewServer(nil)

	// Agent needs to be registered first (will return 404 if not registered)
	runReq := AgentRunRequest{
		Input: "Hello, agent!",
	}

	body, _ := json.Marshal(runReq)
	req, _ := http.NewRequest("POST", "/api/v1/agents/test-agent/run", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// Expect 404 since agent is not registered
	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v (agent not registered)", w.Code, http.StatusNotFound)
	}
}

func TestAgentRun_WithSession(t *testing.T) {
	server, _ := NewServer(nil)

	// Create a session first
	sess := session.NewSession("test-session", "test-agent")
	server.sessionStorage.Create(context.Background(), sess)

	runReq := AgentRunRequest{
		Input:     "Hello, agent!",
		SessionID: "test-session",
	}

	body, _ := json.Marshal(runReq)
	req, _ := http.NewRequest("POST", "/api/v1/agents/test-agent/run", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// Expect 404 since agent is not registered
	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v (agent not registered)", w.Code, http.StatusNotFound)
	}
}

func TestAgentRun_Success(t *testing.T) {
	server, _ := NewServer(nil)

	model := &simpleModel{BaseModel: models.BaseModel{ID: "mock-model", Provider: "mock"}}
	agentInstance, err := agent.New(agent.Config{
		Name:  "runner",
		Model: model,
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	if err := server.RegisterAgent("runner", agentInstance); err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	sess := session.NewSession("runner-session", "runner")
	if err := server.sessionStorage.Create(context.Background(), sess); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	body, _ := json.Marshal(AgentRunRequest{Input: "ping", SessionID: "runner-session"})
	req, _ := http.NewRequest("POST", "/api/v1/agents/runner/run", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", w.Code)
	}

	var resp AgentRunResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.RunID == "" {
		t.Fatalf("expected run id")
	}
	if resp.Status != agent.RunStatusCompleted {
		t.Fatalf("expected completed status, got %s", resp.Status)
	}
	if resp.Metadata != nil {
		if hit, ok := resp.Metadata["cache_hit"].(bool); ok && hit {
			t.Fatalf("expected first run cache miss")
		}
	}

	stored, err := server.sessionStorage.Get(context.Background(), "runner-session")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}
	if stored.GetRunCount() != 1 {
		t.Fatalf("expected run to be persisted")
	}
}

func TestAgentRun_MediaOnly(t *testing.T) {
	server, _ := NewServer(nil)

	model := &simpleModel{BaseModel: models.BaseModel{ID: "mock-model", Provider: "mock"}}
	agentInstance, err := agent.New(agent.Config{
		Name:  "media-agent",
		Model: model,
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	if err := server.RegisterAgent("media-agent", agentInstance); err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	payload := map[string]interface{}{
		"media": []map[string]interface{}{
			{"type": "image", "url": "https://example.com/image.png"},
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/agents/media-agent/run", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", w.Code, w.Body.String())
	}

	var resp AgentRunResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Content == "" {
		t.Fatalf("expected non-empty content for media run")
	}

	mediaList, ok := resp.Metadata["media"].([]interface{})
	if !ok || len(mediaList) != 1 {
		t.Fatalf("expected media metadata, got %#v", resp.Metadata["media"])
	}
}

func TestAgentRun_StreamEvents(t *testing.T) {
	server, _ := NewServer(nil)

	model := &simpleModel{BaseModel: models.BaseModel{ID: "mock-model", Provider: "mock"}}
	agentInstance, err := agent.New(agent.Config{
		Name:  "streamer",
		Model: model,
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	if err := server.RegisterAgent("streamer", agentInstance); err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	body, _ := json.Marshal(AgentRunRequest{Input: "stream hello"})
	req, _ := http.NewRequest("POST", "/api/v1/agents/streamer/run?stream_events=true", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", w.Code)
	}

	payload := w.Body.String()
	if !strings.Contains(payload, "event: run_start") {
		t.Fatalf("expected run_start event, got %s", payload)
	}
	if !strings.Contains(payload, "event: token") {
		t.Fatalf("expected token events, got %s", payload)
	}
	if !strings.Contains(payload, "event: complete") {
		t.Fatalf("expected complete event")
	}
}
