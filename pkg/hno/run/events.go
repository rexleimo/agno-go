package run

import "time"

const (
	EventTypeRunContent   = "run_content"
	EventTypeRunCompleted = "run_completed"
)

// Events represents a collection of base run output events that can be marshaled
// to and from JSON.
type Events []BaseRunOutputEvent

// BaseRunOutputEvent describes the minimal contract for run output events that
// can be persisted in workflow history or emitted to downstream consumers.
type BaseRunOutputEvent interface {
	EventType() string
	Timestamp() time.Time
}

type eventBase struct {
	eventType string
	timestamp time.Time
}

func (b eventBase) EventType() string {

	return b.eventType
}

func (b eventBase) Timestamp() time.Time {

	return b.timestamp
}

// RunContentEvent captures incremental assistant output for agent or team runs.
type RunContentEvent struct {
	eventBase
	RunID    string `json:"run_id,omitempty"`
	AgentID  string `json:"agent_id,omitempty"`
	TeamID   string `json:"team_id,omitempty"`
	Sequence int    `json:"sequence,omitempty"`
	Role     string `json:"role,omitempty"`
	Content  string `json:"content,omitempty"`
}

// RunCompletedEvent signals that a run has reached a terminal state.
type RunCompletedEvent struct {
	eventBase
	RunID   string `json:"run_id,omitempty"`
	AgentID string `json:"agent_id,omitempty"`
	TeamID  string `json:"team_id,omitempty"`
	Status  string `json:"status,omitempty"`
	Output  string `json:"content,omitempty"`
}

// NewRunContentEvent constructs a content event for an agent run.
func NewRunContentEvent(runID, agentID, role, content string, sequence int) *RunContentEvent {
	return &RunContentEvent{
		eventBase: eventBase{eventType: EventTypeRunContent, timestamp: time.Now().UTC()},
		RunID:     runID,
		AgentID:   agentID,
		Sequence:  sequence,
		Role:      role,
		Content:   content,
	}
}

// NewTeamRunContentEvent constructs a content event emitted from a team run.
func NewTeamRunContentEvent(runID, teamID, agentID, role, content string, sequence int) *RunContentEvent {

	evt := NewRunContentEvent(runID, agentID, role, content, sequence)
	evt.TeamID = teamID
	return evt
}

// NewRunCompletedEvent constructs a completion event for an agent run.
func NewRunCompletedEvent(runID, agentID, teamID, status, output string) *RunCompletedEvent {
	return &RunCompletedEvent{
		eventBase: eventBase{eventType: EventTypeRunCompleted, timestamp: time.Now().UTC()},
		RunID:     runID,
		AgentID:   agentID,
		TeamID:    teamID,
		Status:    status,
		Output:    output,
	}
}
