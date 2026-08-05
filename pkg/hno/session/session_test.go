package session

import (
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func TestNewSession(t *testing.T) {
	sessionID := "test-session-123"
	agentID := "test-agent-456"

	session := NewSession(sessionID, agentID)

	if session.SessionID != sessionID {
		t.Errorf("SessionID = %v, want %v", session.SessionID, sessionID)
	}

	if session.AgentID != agentID {
		t.Errorf("AgentID = %v, want %v", session.AgentID, agentID)
	}

	if session.Runs == nil {
		t.Error("Runs should be initialized")
	}

	if session.Metadata == nil {
		t.Error("Metadata should be initialized")
	}

	if session.State == nil {
		t.Error("State should be initialized")
	}

	if session.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	if session.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestSession_AddRun(t *testing.T) {
	session := NewSession("sess-1", "agent-1")

	run1 := &agent.RunOutput{
		Content: "First response",
	}

	run2 := &agent.RunOutput{
		Content: "Second response",
	}

	// Add first run
	initialUpdatedAt := session.UpdatedAt
	time.Sleep(1 * time.Millisecond) // Ensure time difference
	session.AddRun(run1)

	if len(session.Runs) != 1 {
		t.Errorf("Expected 1 run, got %d", len(session.Runs))
	}

	if session.Runs[0].Content != "First response" {
		t.Errorf("Run content = %v, want 'First response'", session.Runs[0].Content)
	}

	if !session.UpdatedAt.After(initialUpdatedAt) {
		t.Error("UpdatedAt should be updated after adding run")
	}

	// Add second run
	session.AddRun(run2)

	if len(session.Runs) != 2 {
		t.Errorf("Expected 2 runs, got %d", len(session.Runs))
	}
}

func TestSession_GetRunCount(t *testing.T) {
	session := NewSession("sess-1", "agent-1")

	if session.GetRunCount() != 0 {
		t.Errorf("Initial run count = %d, want 0", session.GetRunCount())
	}

	session.AddRun(&agent.RunOutput{Content: "Run 1"})
	if session.GetRunCount() != 1 {
		t.Errorf("Run count = %d, want 1", session.GetRunCount())
	}

	session.AddRun(&agent.RunOutput{Content: "Run 2"})
	if session.GetRunCount() != 2 {
		t.Errorf("Run count = %d, want 2", session.GetRunCount())
	}
}

func TestSession_GetLastRun(t *testing.T) {
	session := NewSession("sess-1", "agent-1")

	// No runs yet
	if session.GetLastRun() != nil {
		t.Error("Expected nil for empty session")
	}

	// Add runs
	run1 := &agent.RunOutput{Content: "Run 1"}
	run2 := &agent.RunOutput{Content: "Run 2"}

	session.AddRun(run1)
	lastRun := session.GetLastRun()
	if lastRun == nil || lastRun.Content != "Run 1" {
		t.Errorf("Last run = %v, want 'Run 1'", lastRun)
	}

	session.AddRun(run2)
	lastRun = session.GetLastRun()
	if lastRun == nil || lastRun.Content != "Run 2" {
		t.Errorf("Last run = %v, want 'Run 2'", lastRun)
	}
}

func TestSession_CalculateTotalTokens(t *testing.T) {
	session := NewSession("sess-1", "agent-1")

	session.AddRun(&agent.RunOutput{Metadata: map[string]interface{}{
		"usage": types.Usage{PromptTokens: 10, CompletionTokens: 6},
	}})
	session.AddRun(&agent.RunOutput{Metadata: map[string]interface{}{
		"usage": map[string]interface{}{"total_tokens": float64(9)},
	}})
	session.AddRun(&agent.RunOutput{Metadata: map[string]interface{}{
		"usage": &types.Usage{TotalTokens: 4},
	}})

	tokens := session.CalculateTotalTokens()
	if tokens != 29 {
		t.Errorf("Total tokens = %d, want 29", tokens)
	}
}

func TestSession_GenerateSummary(t *testing.T) {
	session := NewSession("sess-1", "agent-1")

	// Add some runs
	session.AddRun(&agent.RunOutput{Content: "Run 1"})
	session.AddRun(&agent.RunOutput{Content: "Run 2"})
	session.AddRun(&agent.RunOutput{Content: "Run 3"})

	summaryContent := "This is a test summary"
	session.GenerateSummary(summaryContent)

	if session.Summary == nil {
		t.Fatal("Summary should not be nil")
	}

	if session.Summary.Content != summaryContent {
		t.Errorf("Summary content = %v, want %v", session.Summary.Content, summaryContent)
	}

	if session.Summary.RunCount != 3 {
		t.Errorf("Summary run count = %d, want 3", session.Summary.RunCount)
	}

	if session.Summary.CreatedAt.IsZero() {
		t.Error("Summary CreatedAt should be set")
	}
}

func TestSession_Metadata(t *testing.T) {
	session := NewSession("sess-1", "agent-1")

	// Add metadata
	session.Metadata["user_name"] = "John Doe"
	session.Metadata["session_type"] = "test"

	if session.Metadata["user_name"] != "John Doe" {
		t.Error("Metadata user_name not set correctly")
	}

	if session.Metadata["session_type"] != "test" {
		t.Error("Metadata session_type not set correctly")
	}
}

func TestSession_State(t *testing.T) {
	session := NewSession("sess-1", "agent-1")

	// Add state
	session.State["current_step"] = 1
	session.State["completed"] = false

	if session.State["current_step"] != 1 {
		t.Error("State current_step not set correctly")
	}

	if session.State["completed"] != false {
		t.Error("State completed not set correctly")
	}
}

func TestSession_UserAndTeamIDs(t *testing.T) {
	session := NewSession("sess-1", "agent-1")

	session.UserID = "user-123"
	session.TeamID = "team-456"
	session.WorkflowID = "workflow-789"

	if session.UserID != "user-123" {
		t.Error("UserID not set correctly")
	}

	if session.TeamID != "team-456" {
		t.Error("TeamID not set correctly")
	}

	if session.WorkflowID != "workflow-789" {
		t.Error("WorkflowID not set correctly")
	}
}
