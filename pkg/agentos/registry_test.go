package agentos

import (
	"fmt"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models/openai"
)

func createTestAgent(t *testing.T, name string) *agent.Agent {
	t.Helper()

	model, err := openai.New("gpt-3.5-turbo", openai.Config{
		APIKey: "test-key",
	})
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	ag, err := agent.New(agent.Config{
		Name:  name,
		Model: model,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	return ag
}

func TestNewAgentRegistry(t *testing.T) {
	registry := NewAgentRegistry()

	if registry == nil {
		t.Fatal("Expected non-nil registry")
	}

	if registry.Count() != 0 {
		t.Errorf("Count = %d, want 0", registry.Count())
	}
}

func TestAgentRegistry_Register(t *testing.T) {
	registry := NewAgentRegistry()
	ag := createTestAgent(t, "test-agent")

	err := registry.Register("agent-1", ag)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("Count = %d, want 1", registry.Count())
	}

	// Verify we can retrieve it
	retrieved, err := registry.Get("agent-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Name != "test-agent" {
		t.Errorf("Agent name = %s, want 'test-agent'", retrieved.Name)
	}
}

func TestAgentRegistry_Register_EmptyID(t *testing.T) {
	registry := NewAgentRegistry()
	ag := createTestAgent(t, "test-agent")

	err := registry.Register("", ag)
	if err == nil {
		t.Error("Expected error for empty agent ID")
	}
}

func TestAgentRegistry_Register_NilAgent(t *testing.T) {
	registry := NewAgentRegistry()

	err := registry.Register("agent-1", nil)
	if err == nil {
		t.Error("Expected error for nil agent")
	}
}

func TestAgentRegistry_Register_Duplicate(t *testing.T) {
	registry := NewAgentRegistry()
	ag1 := createTestAgent(t, "agent-1")
	ag2 := createTestAgent(t, "agent-2")

	// Register first agent
	err := registry.Register("same-id", ag1)
	if err != nil {
		t.Fatalf("First Register() error = %v", err)
	}

	// Try to register second agent with same ID
	err = registry.Register("same-id", ag2)
	if err == nil {
		t.Error("Expected error for duplicate agent ID")
	}
}

func TestAgentRegistry_Unregister(t *testing.T) {
	registry := NewAgentRegistry()
	ag := createTestAgent(t, "test-agent")

	registry.Register("agent-1", ag)

	// Unregister
	err := registry.Unregister("agent-1")
	if err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	if registry.Count() != 0 {
		t.Errorf("Count = %d, want 0", registry.Count())
	}

	// Verify it's gone
	_, err = registry.Get("agent-1")
	if err == nil {
		t.Error("Expected error for unregistered agent")
	}
}

func TestAgentRegistry_Unregister_NotFound(t *testing.T) {
	registry := NewAgentRegistry()

	err := registry.Unregister("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent agent")
	}
}

func TestAgentRegistry_Get(t *testing.T) {
	registry := NewAgentRegistry()
	ag := createTestAgent(t, "test-agent")

	registry.Register("agent-1", ag)

	retrieved, err := registry.Get("agent-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Name != "test-agent" {
		t.Errorf("Agent name = %s, want 'test-agent'", retrieved.Name)
	}
}

func TestAgentRegistry_Get_NotFound(t *testing.T) {
	registry := NewAgentRegistry()

	_, err := registry.Get("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent agent")
	}
}

func TestAgentRegistry_Get_EmptyID(t *testing.T) {
	registry := NewAgentRegistry()

	_, err := registry.Get("")
	if err == nil {
		t.Error("Expected error for empty agent ID")
	}
}

func TestAgentRegistry_List(t *testing.T) {
	registry := NewAgentRegistry()

	ag1 := createTestAgent(t, "agent-1")
	ag2 := createTestAgent(t, "agent-2")
	ag3 := createTestAgent(t, "agent-3")

	registry.Register("id-1", ag1)
	registry.Register("id-2", ag2)
	registry.Register("id-3", ag3)

	agents := registry.List()

	if len(agents) != 3 {
		t.Errorf("List count = %d, want 3", len(agents))
	}

	if agents["id-1"].Name != "agent-1" {
		t.Error("Agent id-1 not found in list")
	}
}

func TestAgentRegistry_Count(t *testing.T) {
	registry := NewAgentRegistry()

	if registry.Count() != 0 {
		t.Errorf("Initial count = %d, want 0", registry.Count())
	}

	ag1 := createTestAgent(t, "agent-1")
	registry.Register("id-1", ag1)

	if registry.Count() != 1 {
		t.Errorf("Count = %d, want 1", registry.Count())
	}

	ag2 := createTestAgent(t, "agent-2")
	registry.Register("id-2", ag2)

	if registry.Count() != 2 {
		t.Errorf("Count = %d, want 2", registry.Count())
	}

	registry.Unregister("id-1")

	if registry.Count() != 1 {
		t.Errorf("Count = %d, want 1", registry.Count())
	}
}

func TestAgentRegistry_Exists(t *testing.T) {
	registry := NewAgentRegistry()
	ag := createTestAgent(t, "test-agent")

	if registry.Exists("agent-1") {
		t.Error("Agent should not exist yet")
	}

	registry.Register("agent-1", ag)

	if !registry.Exists("agent-1") {
		t.Error("Agent should exist")
	}

	registry.Unregister("agent-1")

	if registry.Exists("agent-1") {
		t.Error("Agent should not exist after unregister")
	}
}

func TestAgentRegistry_Clear(t *testing.T) {
	registry := NewAgentRegistry()

	ag1 := createTestAgent(t, "agent-1")
	ag2 := createTestAgent(t, "agent-2")

	registry.Register("id-1", ag1)
	registry.Register("id-2", ag2)

	if registry.Count() != 2 {
		t.Errorf("Count = %d, want 2", registry.Count())
	}

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("Count = %d, want 0 after clear", registry.Count())
	}
}

func TestAgentRegistry_GetIDs(t *testing.T) {
	registry := NewAgentRegistry()

	ag1 := createTestAgent(t, "agent-1")
	ag2 := createTestAgent(t, "agent-2")

	registry.Register("id-1", ag1)
	registry.Register("id-2", ag2)

	ids := registry.GetIDs()

	if len(ids) != 2 {
		t.Errorf("GetIDs count = %d, want 2", len(ids))
	}

	// Verify both IDs are present
	hasID1 := false
	hasID2 := false
	for _, id := range ids {
		if id == "id-1" {
			hasID1 = true
		}
		if id == "id-2" {
			hasID2 = true
		}
	}

	if !hasID1 || !hasID2 {
		t.Error("Not all agent IDs returned")
	}
}

func TestAgentRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewAgentRegistry()
	done := make(chan bool, 10)

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func(idx int) {
			ag := createTestAgent(t, "test-agent")
			agentID := fmt.Sprintf("agent-%d", idx)
			err := registry.Register(agentID, ag)
			if err != nil {
				t.Errorf("Concurrent Register() error = %v", err)
			}
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			_ = registry.Count()
			_ = registry.GetIDs()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	if registry.Count() != 5 {
		t.Errorf("Final count = %d, want 5", registry.Count())
	}
}
