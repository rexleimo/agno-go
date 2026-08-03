package claude

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestToolkit_InvokeSkill(t *testing.T) {
	var received skillRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/agent-skills/messages" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-key" {
			t.Fatalf("missing api key header")
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		resp := skillResponse{
			SkillID:        received.SkillID,
			ConversationID: "conv-123",
			Output: map[string]interface{}{
				"summary": "processed " + received.Input,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tk, err := New(Config{
		BaseURL:      server.URL,
		APIKey:       "test-key",
		DefaultSkill: "skill-1",
	})
	if err != nil {
		t.Fatalf("failed to create toolkit: %v", err)
	}

	result, err := tk.Execute(context.Background(), "invoke_claude_skill", map[string]interface{}{
		"input": "hello world",
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	output := result.(map[string]interface{})
	if output["skill_id"] != "skill-1" {
		t.Fatalf("expected skill_id skill-1, got %v", output["skill_id"])
	}
	if output["conversation_id"] != "conv-123" {
		t.Fatalf("expected conversation_id conv-123, got %v", output["conversation_id"])
	}

	skillOutput := output["output"].(map[string]interface{})
	if skillOutput["summary"] != "processed hello world" {
		t.Fatalf("unexpected output %v", skillOutput["summary"])
	}

	if received.Input != "hello world" {
		t.Fatalf("request input mismatch: %s", received.Input)
	}
}
