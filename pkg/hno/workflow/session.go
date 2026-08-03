package workflow

import (
	"fmt"
	"sync"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// HistoryEntry represents a single entry in the workflow history
// HistoryEntry 表示工作流历史中的单个条目
type HistoryEntry struct {
	// Input is the input for this history entry
	// Input 是此历史条目的输入
	Input string `json:"input"`

	// Output is the output for this history entry
	// Output 是此历史条目的输出
	Output string `json:"output"`

	// Timestamp is when this entry was created
	// Timestamp 是此条目创建的时间
	Timestamp time.Time `json:"timestamp"`
}

// CancellationRecord captures workflow cancellation details.
type CancellationRecord struct {
	RunID      string                 `json:"run_id"`
	Reason     string                 `json:"reason"`
	StepID     string                 `json:"step_id,omitempty"`
	Snapshot   map[string]interface{} `json:"snapshot,omitempty"`
	OccurredAt time.Time              `json:"occurred_at"`
}

func cloneMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// WorkflowSession manages multiple workflow runs for a single session
// WorkflowSession 管理单个会话的多个工作流运行
type WorkflowSession struct {
	// mu protects concurrent access to the session
	// mu 保护对会话的并发访问
	mu sync.RWMutex

	// SessionID is the unique identifier for this session
	// SessionID 是此会话的唯一标识符
	SessionID string `json:"session_id"`

	// WorkflowID is the ID of the workflow this session belongs to
	// WorkflowID 是此会话所属工作流的 ID
	WorkflowID string `json:"workflow_id"`

	// UserID is the user who owns this session
	// UserID 是拥有此会话的用户
	UserID string `json:"user_id,omitempty"`

	// Runs contains all workflow runs in this session
	// Runs 包含此会话中的所有工作流运行
	Runs []*WorkflowRun `json:"runs"`

	// CreatedAt is the timestamp when the session was created
	// CreatedAt 是会话创建的时间戳
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the timestamp when the session was last updated
	// UpdatedAt 是会话最后更新的时间戳
	UpdatedAt time.Time `json:"updated_at"`

	// Metadata contains additional session metadata
	// Metadata 包含额外的会话元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Cancellations captures cancellation snapshots for later recovery.
	Cancellations []*CancellationRecord `json:"cancellations,omitempty"`
}

// NewWorkflowSession creates a new workflow session
// NewWorkflowSession 创建新的工作流会话
func NewWorkflowSession(sessionID, workflowID, userID string) *WorkflowSession {
	now := time.Now()
	return &WorkflowSession{
		SessionID:     sessionID,
		WorkflowID:    workflowID,
		UserID:        userID,
		Runs:          make([]*WorkflowRun, 0),
		CreatedAt:     now,
		UpdatedAt:     now,
		Metadata:      make(map[string]interface{}),
		Cancellations: make([]*CancellationRecord, 0),
	}
}

// AddRun adds a workflow run to the session
// AddRun 将工作流运行添加到会话
func (s *WorkflowSession) AddRun(run *WorkflowRun) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Runs = append(s.Runs, run)
	s.UpdatedAt = time.Now()
}

// AddCancellation records a cancellation snapshot.
func (s *WorkflowSession) AddCancellation(record *CancellationRecord) {
	if record == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Cancellations = append(s.Cancellations, record)
	s.UpdatedAt = time.Now()
}

// GetCancellations returns a copy of cancellation records.
func (s *WorkflowSession) GetCancellations() []*CancellationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.Cancellations) == 0 {
		return nil
	}
	list := make([]*CancellationRecord, len(s.Cancellations))
	for i, record := range s.Cancellations {
		if record == nil {
			continue
		}
		cloned := *record
		if record.Snapshot != nil {
			cloned.Snapshot = cloneMap(record.Snapshot)
		}
		list[i] = &cloned
	}
	return list
}

// GetRuns returns all runs in the session (thread-safe copy)
// GetRuns 返回会话中的所有运行（线程安全副本）
func (s *WorkflowSession) GetRuns() []*WorkflowRun {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	// 返回副本以防止外部修改
	runs := make([]*WorkflowRun, len(s.Runs))
	copy(runs, s.Runs)
	return runs
}

// GetLastRun returns the most recent run in the session
// GetLastRun 返回会话中最近的运行
func (s *WorkflowSession) GetLastRun() *WorkflowRun {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.Runs) == 0 {
		return nil
	}

	return s.Runs[len(s.Runs)-1]
}

// GetHistory returns the most recent N completed runs as history entries
// GetHistory 返回最近 N 个已完成的运行作为历史条目
func (s *WorkflowSession) GetHistory(numRuns int) []HistoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.Runs) == 0 {
		return nil
	}

	// Filter completed runs only
	// 仅过滤已完成的运行
	completedRuns := make([]*WorkflowRun, 0)
	for _, run := range s.Runs {
		if run.IsCompleted() {
			completedRuns = append(completedRuns, run)
		}
	}

	if len(completedRuns) == 0 {
		return nil
	}

	// Get the most recent N runs
	// 获取最近的 N 个运行
	startIdx := 0
	if numRuns > 0 && len(completedRuns) > numRuns {
		startIdx = len(completedRuns) - numRuns
	}

	recentRuns := completedRuns[startIdx:]

	// Convert to history entries
	// 转换为历史条目
	history := make([]HistoryEntry, len(recentRuns))
	for i, run := range recentRuns {
		history[i] = HistoryEntry{
			Input:     run.Input,
			Output:    run.Output,
			Timestamp: run.CompletedAt,
		}
	}

	return history
}

// GetHistoryContext returns formatted workflow history context for agent use
// GetHistoryContext 返回格式化的工作流历史上下文供 agent 使用
func (s *WorkflowSession) GetHistoryContext(numRuns int) string {
	history := s.GetHistory(numRuns)

	if len(history) == 0 {
		return ""
	}

	// Format as context string
	// 格式化为上下文字符串
	context := "<workflow_history_context>\n"

	for i, entry := range history {
		context += fmt.Sprintf("[run-%d]\n", i+1)
		if entry.Input != "" {
			context += fmt.Sprintf("input: %s\n", entry.Input)
		}
		if entry.Output != "" {
			context += fmt.Sprintf("output: %s\n", entry.Output)
		}
		context += "\n" // Empty line between runs
	}

	context += "</workflow_history_context>"

	return context
}

// GetHistoryMessages returns conversation messages from recent runs
// GetHistoryMessages 返回最近运行的对话消息
func (s *WorkflowSession) GetHistoryMessages(numRuns int) []*types.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.Runs) == 0 {
		return nil
	}

	// Filter completed runs
	// 过滤已完成的运行
	completedRuns := make([]*WorkflowRun, 0)
	for _, run := range s.Runs {
		if run.IsCompleted() {
			completedRuns = append(completedRuns, run)
		}
	}

	if len(completedRuns) == 0 {
		return nil
	}

	// Get the most recent N runs
	// 获取最近的 N 个运行
	startIdx := 0
	if numRuns > 0 && len(completedRuns) > numRuns {
		startIdx = len(completedRuns) - numRuns
	}

	recentRuns := completedRuns[startIdx:]

	// Collect all messages
	// 收集所有消息
	messages := make([]*types.Message, 0)
	for _, run := range recentRuns {
		messages = append(messages, run.Messages...)
	}

	return messages
}

// CountRuns returns the total number of runs in the session
// CountRuns 返回会话中运行的总数
func (s *WorkflowSession) CountRuns() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.Runs)
}

// CountCompletedRuns returns the number of completed runs
// CountCompletedRuns 返回已完成运行的数量
func (s *WorkflowSession) CountCompletedRuns() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, run := range s.Runs {
		if run.IsCompleted() {
			count++
		}
	}

	return count
}

// CountSuccessfulRuns returns the number of successful runs
// CountSuccessfulRuns 返回成功运行的数量
func (s *WorkflowSession) CountSuccessfulRuns() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, run := range s.Runs {
		if run.IsSuccessful() {
			count++
		}
	}

	return count
}

// CountFailedRuns returns the number of failed runs
// CountFailedRuns 返回失败运行的数量
func (s *WorkflowSession) CountFailedRuns() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, run := range s.Runs {
		if run.Status == RunStatusFailed {
			count++
		}
	}

	return count
}

// Clear removes all runs from the session
// Clear 移除会话中的所有运行
func (s *WorkflowSession) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Runs = make([]*WorkflowRun, 0)
	s.UpdatedAt = time.Now()
}

// GetMetadata retrieves a metadata value
// GetMetadata 检索元数据值
func (s *WorkflowSession) GetMetadata(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, exists := s.Metadata[key]
	return value, exists
}

// SetMetadata sets a metadata value
// SetMetadata 设置元数据值
func (s *WorkflowSession) SetMetadata(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Metadata[key] = value
	s.UpdatedAt = time.Now()
}
