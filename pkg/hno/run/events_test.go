package run

import (
	"encoding/json"
	"testing"
)

func TestEventsRoundTrip(t *testing.T) {
	events := Events{
		NewRunContentEvent("run-1", "agent-1", "assistant", "hello", 0),
		NewRunCompletedEvent("run-1", "agent-1", "", "completed", "done"),
	}
	data, err := json.Marshal(events)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded Events
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(decoded) != len(events) {
		t.Fatalf("expected %d events, got %d", len(events), len(decoded))
	}
	if decoded[0].EventType() != EventTypeRunContent {
		t.Fatalf("expected first event type %s, got %s", EventTypeRunContent, decoded[0].EventType())
	}
	if decoded[1].EventType() != EventTypeRunCompleted {
		t.Fatalf("expected second event type %s, got %s", EventTypeRunCompleted, decoded[1].EventType())
	}
}

func TestDecodeTeamEventWithoutAgentID(t *testing.T) {
	payload := []byte(`[{"event":"team_run_content","team_id":"team-1","content":"hi"}]`)
	var decoded Events
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decoded))
	}
	evt, ok := decoded[0].(*RunContentEvent)
	if !ok {
		t.Fatalf("expected RunContentEvent, got %T", decoded[0])
	}
	if evt.TeamID != "team-1" {
		t.Fatalf("expected team id team-1, got %s", evt.TeamID)
	}
}

func TestGenericRunEventPreservesPayload(t *testing.T) {
	payload := []byte(`[{"event":"run_started","agent_id":"a","created_at":1700000000}]`)
	var decoded Events
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	gen, ok := decoded[0].(*GenericRunEvent)
	if !ok {
		t.Fatalf("expected GenericRunEvent, got %T", decoded[0])
	}
	if gen.EventType() != "run_started" {
		t.Fatalf("expected run_started event, got %s", gen.EventType())
	}
	data, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("re-marshal failed: %v", err)
	}
	var roundtrip []map[string]interface{}
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("roundtrip decode failed: %v", err)
	}
	if len(roundtrip) == 0 || roundtrip[0]["event"] != "run_started" {
		t.Fatalf("expected event persisted after roundtrip")
	}
}
