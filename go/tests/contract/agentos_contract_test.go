package contract_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
	"github.com/rexleimo/agno-go/internal/runtime"
	"github.com/rexleimo/agno-go/pkg/memory"
	"github.com/rexleimo/agno-go/pkg/providers/stub"
)

func TestOpenAPISpecExists(t *testing.T) {
	path := filepath.Join("..", "..", "..", "specs", "001-agno-agents-refactor", "contracts", "openapi.yaml")
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("openapi.yaml missing at %s", path)
		}
		t.Fatalf("openapi.yaml stat failed at %s: %v", path, err)
	}
}

func TestAgentSessionMessageContract(t *testing.T) {
	mem := memory.NewInMemoryStore()
	router := model.NewRouter()
	router.RegisterChatProvider(stub.New(agent.ProviderOpenAI, model.ProviderAvailable, nil))
	svc := runtime.NewService(mem, router)
	server := runtime.NewServer(router.Statuses, "test", svc)

	handler := server.Router

	agentID := createAgent(t, handler)
	sessionID := createSession(t, handler, agentID)

	nonStream := postMessage(t, handler, agentID, sessionID, false)
	if nonStream.MessageID == "" {
		t.Fatalf("missing messageId in non-stream response")
	}
	if _, err := uuid.Parse(nonStream.MessageID); err != nil {
		t.Fatalf("invalid messageId: %v", err)
	}
	if nonStream.Content == "" || !strings.HasPrefix(nonStream.Content, "echo:") {
		t.Fatalf("unexpected assistant content: %q", nonStream.Content)
	}
	if nonStream.State != string(agent.SessionCompleted) {
		t.Fatalf("expected state %s, got %s", agent.SessionCompleted, nonStream.State)
	}
	if nonStream.Usage.PromptTokens == 0 {
		t.Fatalf("expected usage tokens to be populated")
	}

	streamEvents := postMessageStream(t, handler, agentID, sessionID)
	var sawEnd bool
	var tokens []string
	for _, ev := range streamEvents {
		if ev.Type == "token" {
			tokens = append(tokens, strings.TrimSpace(ev.Delta))
		}
		if ev.Done {
			sawEnd = true
		}
	}
	if len(tokens) == 0 {
		t.Fatalf("expected streaming tokens, got none")
	}
	if !sawEnd {
		t.Fatalf("expected end event in stream")
	}
}

type agentCreateResponse struct {
	AgentID string `json:"agentId"`
}

type sessionResponse struct {
	SessionID string `json:"sessionId"`
	AgentID   string `json:"agentId"`
	State     string `json:"state"`
}

type messageResponse struct {
	MessageID string           `json:"messageId"`
	Content   string           `json:"content"`
	ToolCalls []agent.ToolCall `json:"toolCalls"`
	Usage     agent.Usage      `json:"usage"`
	State     string           `json:"state"`
}

func createAgent(t *testing.T, handler http.Handler) uuid.UUID {
	t.Helper()
	payload := map[string]any{
		"name": "contract-agent",
		"model": map[string]any{
			"provider": "openai",
			"modelId":  "gpt-test",
			"stream":   true,
		},
	}
	resp := doPost(t, handler, "/agents", payload)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create agent status = %d", resp.Code)
	}
	var out agentCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode create agent response: %v", err)
	}
	id, err := uuid.Parse(out.AgentID)
	if err != nil {
		t.Fatalf("agentId not uuid: %v", err)
	}
	return id
}

func createSession(t *testing.T, handler http.Handler, agentID uuid.UUID) uuid.UUID {
	t.Helper()
	url := "/agents/" + agentID.String() + "/sessions"
	resp := doPost(t, handler, url, map[string]any{})
	if resp.Code != http.StatusCreated {
		t.Fatalf("create session status = %d", resp.Code)
	}
	var out sessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode create session response: %v", err)
	}
	if out.AgentID != agentID.String() {
		t.Fatalf("agentId mismatch: expected %s got %s", agentID, out.AgentID)
	}
	id, err := uuid.Parse(out.SessionID)
	if err != nil {
		t.Fatalf("sessionId not uuid: %v", err)
	}
	return id
}

func postMessage(t *testing.T, handler http.Handler, agentID, sessionID uuid.UUID, stream bool) messageResponse {
	t.Helper()
	url := "/agents/" + agentID.String() + "/sessions/" + sessionID.String() + "/messages"
	payload := map[string]any{
		"messages": []map[string]any{
			{"role": "user", "content": "hello contract"},
		},
	}
	resp := doPost(t, handler, url, payload)

	if stream {
		t.Fatalf("use postMessageStream for streaming calls")
	}
	if resp.Code != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("non-stream message status = %d body=%s", resp.Code, string(body))
	}
	var out messageResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode message response: %v", err)
	}
	return out
}

func postMessageStream(t *testing.T, handler http.Handler, agentID, sessionID uuid.UUID) []model.ChatStreamEvent {
	t.Helper()
	url := "/agents/" + agentID.String() + "/sessions/" + sessionID.String() + "/messages?stream=true"
	payload := map[string]any{
		"messages": []map[string]any{
			{"role": "user", "content": "stream hello"},
		},
	}
	resp := doPost(t, handler, url, payload)

	if resp.Code != http.StatusMultiStatus {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("stream message status = %d body=%s", resp.Code, string(body))
	}
	if ct := resp.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %s", ct)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read stream body: %v", err)
	}
	return parseSSE(raw)
}

func parseSSE(raw []byte) []model.ChatStreamEvent {
	lines := strings.Split(string(raw), "\n")
	var events []model.ChatStreamEvent
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		line = strings.TrimPrefix(line, "data: ")
		if line == "" {
			continue
		}
		var ev model.ChatStreamEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		events = append(events, ev)
	}
	return events
}

func doPost(t *testing.T, handler http.Handler, url string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}
