package workflow

import (
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// RunStatus represents the status of a workflow run
// RunStatus 表示工作流运行的状态
type RunStatus string

const (
	// RunStatusPending indicates the run is pending
	// RunStatusPending 表示运行正在等待
	RunStatusPending RunStatus = "pending"

	// RunStatusRunning indicates the run is in progress
	// RunStatusRunning 表示运行正在进行中
	RunStatusRunning RunStatus = "running"

	// RunStatusCompleted indicates the run completed successfully
	// RunStatusCompleted 表示运行成功完成
	RunStatusCompleted RunStatus = "completed"

	// RunStatusFailed indicates the run failed with an error
	// RunStatusFailed 表示运行失败并出现错误
	RunStatusFailed RunStatus = "failed"

	// RunStatusCancelled indicates the run was cancelled
	// RunStatusCancelled 表示运行被取消
	RunStatusCancelled RunStatus = "cancelled"
)

// WorkflowRun represents a single execution of a workflow
// WorkflowRun 表示工作流的单次执行记录
type WorkflowRun struct {
	// RunID is the unique identifier for this run
	// RunID 是此次运行的唯一标识符
	RunID string `json:"run_id"`

	// SessionID is the session this run belongs to
	// SessionID 是此次运行所属的会话 ID
	SessionID string `json:"session_id"`

	// WorkflowID is the ID of the workflow that was executed
	// WorkflowID 是被执行的工作流的 ID
	WorkflowID string `json:"workflow_id"`

	// Input is the input provided to the workflow
	// Input 是提供给工作流的输入
	Input string `json:"input"`

	// Output is the final output of the workflow
	// Output 是工作流的最终输出
	Output string `json:"output"`

	// Messages contains the conversation history for this run
	// Messages 包含此次运行的对话历史
	Messages []*types.Message `json:"messages,omitempty"`

	// Status indicates the current status of the run
	// Status 表示运行的当前状态
	Status RunStatus `json:"status"`

	// Error contains error message if the run failed
	// Error 包含运行失败时的错误信息
	Error string `json:"error,omitempty"`

	// StartedAt is the timestamp when the run started
	// StartedAt 是运行开始的时间戳
	StartedAt time.Time `json:"started_at"`

	// CompletedAt is the timestamp when the run completed
	// CompletedAt 是运行完成的时间戳
	CompletedAt time.Time `json:"completed_at,omitempty"`

	// Metadata contains additional metadata for this run
	// Metadata 包含此次运行的额外元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// CancellationReason provides context when run is cancelled.
	CancellationReason string `json:"cancellation_reason,omitempty"`

	// CancellationSnapshot stores the last known snapshot before cancellation.
	CancellationSnapshot map[string]interface{} `json:"cancellation_snapshot,omitempty"`

	// LastStepID indicates the last step identifier processed before cancellation/failure.
	LastStepID string `json:"last_step_id,omitempty"`

	// ResumedFrom records the step the run resumed from, if applicable.
	ResumedFrom string `json:"resumed_from,omitempty"`

	// Events captures structured run output events for observability.
	Events run.Events `json:"events,omitempty"`
}

// NewWorkflowRun creates a new workflow run with the given parameters
// NewWorkflowRun 使用给定参数创建新的工作流运行记录
func NewWorkflowRun(runID, sessionID, workflowID, input string) *WorkflowRun {
	return &WorkflowRun{
		RunID:      runID,
		SessionID:  sessionID,
		WorkflowID: workflowID,
		Input:      input,
		Status:     RunStatusPending,
		StartedAt:  time.Now(),
		Messages:   make([]*types.Message, 0),
		Metadata:   make(map[string]interface{}),
		Events:     make(run.Events, 0),
	}
}

// MarkStarted marks the run as started
// MarkStarted 将运行标记为已开始
func (r *WorkflowRun) MarkStarted() {
	r.Status = RunStatusRunning
	r.StartedAt = time.Now()
}

// MarkCompleted marks the run as completed with the given output
// MarkCompleted 使用给定输出将运行标记为已完成
func (r *WorkflowRun) MarkCompleted(output string) {
	r.Status = RunStatusCompleted
	r.Output = output
	r.CompletedAt = time.Now()
}

// MarkFailed marks the run as failed with the given error
// MarkFailed 使用给定错误将运行标记为失败
func (r *WorkflowRun) MarkFailed(err error) {
	r.Status = RunStatusFailed
	r.Error = err.Error()
	r.CompletedAt = time.Now()
}

// MarkCancelled marks the run as cancelled
// MarkCancelled 将运行标记为已取消
func (r *WorkflowRun) MarkCancelled() {
	r.Status = RunStatusCancelled
	r.CompletedAt = time.Now()
}

// ApplyCancellation enriches cancellation metadata.
func (r *WorkflowRun) ApplyCancellation(reason, stepID string, snapshot map[string]interface{}) {
	r.MarkCancelled()
	r.CancellationReason = reason
	r.LastStepID = stepID
	if len(snapshot) > 0 {
		r.CancellationSnapshot = snapshot
	}
}

// AddMessage adds a message to the run's conversation history
// AddMessage 将消息添加到运行的对话历史中
func (r *WorkflowRun) AddMessage(msg *types.Message) {
	if r.Messages == nil {
		r.Messages = make([]*types.Message, 0)
	}
	r.Messages = append(r.Messages, msg)
}

// AddEvents appends structured run events to the workflow run metadata.
func (r *WorkflowRun) AddEvents(events run.Events) {
	if r == nil || len(events) == 0 {
		return
	}
	r.Events = append(r.Events, events...)
}

// Duration returns the duration of the run
// Duration 返回运行的持续时间
func (r *WorkflowRun) Duration() time.Duration {
	if r.CompletedAt.IsZero() {
		return time.Since(r.StartedAt)
	}
	return r.CompletedAt.Sub(r.StartedAt)
}

// IsCompleted returns true if the run has completed (success, failure, or cancelled)
// IsCompleted 如果运行已完成（成功、失败或取消）则返回 true
func (r *WorkflowRun) IsCompleted() bool {
	return r.Status == RunStatusCompleted ||
		r.Status == RunStatusFailed ||
		r.Status == RunStatusCancelled
}

// IsSuccessful returns true if the run completed successfully
// IsSuccessful 如果运行成功完成则返回 true
func (r *WorkflowRun) IsSuccessful() bool {
	return r.Status == RunStatusCompleted
}
