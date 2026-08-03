package workflow

import (
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// ExecutionContext holds the execution state and data
// ExecutionContext 保存执行状态和数据
type ExecutionContext struct {
	// Input is the input for the workflow execution
	// Input 是工作流执行的输入
	Input string `json:"input"`

	// Output is the output from the workflow execution
	// Output 是工作流执行的输出
	Output string `json:"output"`

	// Data holds temporary execution data
	// Data 保存临时执行数据
	Data map[string]interface{} `json:"data"`

	// Metadata holds additional metadata
	// Metadata 保存额外的元数据
	Metadata map[string]interface{} `json:"metadata"`

	// SessionState holds session-level state that persists across steps
	// SessionState 保存跨步骤持久化的会话级状态
	SessionState *SessionState `json:"session_state,omitempty"`

	// SessionID is the unique session identifier
	// SessionID 是唯一的会话标识符
	SessionID string `json:"session_id,omitempty"`

	// UserID is the user identifier for multi-tenant scenarios
	// UserID 是多租户场景的用户标识符
	UserID string `json:"user_id,omitempty"`

	// WorkflowHistory contains recent workflow run history
	// WorkflowHistory 包含最近的工作流运行历史
	WorkflowHistory []HistoryEntry `json:"workflow_history,omitempty"`

	// HistoryContext is the formatted history context string
	// HistoryContext 是格式化的历史上下文字符串
	HistoryContext string `json:"history_context,omitempty"`
}

// NewExecutionContext creates a new execution context
// NewExecutionContext 创建新的执行上下文
func NewExecutionContext(input string) *ExecutionContext {
	return &ExecutionContext{
		Input:           input,
		Data:            make(map[string]interface{}),
		Metadata:        make(map[string]interface{}),
		SessionState:    NewSessionState(),
		WorkflowHistory: make([]HistoryEntry, 0),
	}
}

// NewExecutionContextWithSession creates a new execution context with session info
// NewExecutionContextWithSession 创建带会话信息的新执行上下文
func NewExecutionContextWithSession(input, sessionID, userID string) *ExecutionContext {
	return &ExecutionContext{
		Input:           input,
		Data:            make(map[string]interface{}),
		Metadata:        make(map[string]interface{}),
		SessionState:    NewSessionState(),
		SessionID:       sessionID,
		UserID:          userID,
		WorkflowHistory: make([]HistoryEntry, 0),
	}
}

// Set stores a value in the context data
// Set 在上下文数据中存储值
func (ec *ExecutionContext) Set(key string, value interface{}) {
	ec.Data[key] = value
}

// Get retrieves a value from the context data
// Get 从上下文数据检索值
func (ec *ExecutionContext) Get(key string) (interface{}, bool) {
	val, ok := ec.Data[key]
	return val, ok
}

// SetSessionState stores a value in the session state
// SetSessionState 在会话状态中存储值
func (ec *ExecutionContext) SetSessionState(key string, value interface{}) {
	if ec.SessionState == nil {
		ec.SessionState = NewSessionState()
	}
	ec.SessionState.Set(key, value)
}

// GetSessionState retrieves a value from the session state
// GetSessionState 从会话状态检索值
func (ec *ExecutionContext) GetSessionState(key string) (interface{}, bool) {
	if ec.SessionState == nil {
		return nil, false
	}
	return ec.SessionState.Get(key)
}

// GetWorkflowHistory returns the workflow history
// GetWorkflowHistory 获取工作流历史
func (ec *ExecutionContext) GetWorkflowHistory() []HistoryEntry {
	return ec.WorkflowHistory
}

// SetWorkflowHistory sets the workflow history
// SetWorkflowHistory 设置工作流历史
func (ec *ExecutionContext) SetWorkflowHistory(history []HistoryEntry) {
	ec.WorkflowHistory = history
}

// GetHistoryContext returns the formatted history context
// GetHistoryContext 获取格式化的历史上下文
func (ec *ExecutionContext) GetHistoryContext() string {
	return ec.HistoryContext
}

// SetHistoryContext sets the formatted history context
// SetHistoryContext 设置格式化的历史上下文
func (ec *ExecutionContext) SetHistoryContext(context string) {
	ec.HistoryContext = context
}

// HasHistory checks if there is any history
// HasHistory 检查是否有历史记录
func (ec *ExecutionContext) HasHistory() bool {
	return len(ec.WorkflowHistory) > 0
}

// GetHistoryCount returns the number of history entries
// GetHistoryCount 获取历史记录数量
func (ec *ExecutionContext) GetHistoryCount() int {
	return len(ec.WorkflowHistory)
}

// GetLastHistoryEntry returns the last history entry
// GetLastHistoryEntry 获取最后一个历史条目
func (ec *ExecutionContext) GetLastHistoryEntry() *HistoryEntry {
	if len(ec.WorkflowHistory) == 0 {
		return nil
	}
	return &ec.WorkflowHistory[len(ec.WorkflowHistory)-1]
}

// GetHistoryInput returns the input at the specified index
// GetHistoryInput 获取指定索引的历史输入
// index 0 表示最早的历史，-1 表示最近的历史
// index 0 means earliest history, -1 means most recent
func (ec *ExecutionContext) GetHistoryInput(index int) string {
	if len(ec.WorkflowHistory) == 0 {
		return ""
	}

	if index < 0 {
		// 负索引从末尾开始
		// Negative index from the end
		index = len(ec.WorkflowHistory) + index
	}

	if index < 0 || index >= len(ec.WorkflowHistory) {
		return ""
	}

	return ec.WorkflowHistory[index].Input
}

// GetHistoryOutput returns the output at the specified index
// GetHistoryOutput 获取指定索引的历史输出
func (ec *ExecutionContext) GetHistoryOutput(index int) string {
	if len(ec.WorkflowHistory) == 0 {
		return ""
	}

	if index < 0 {
		index = len(ec.WorkflowHistory) + index
	}

	if index < 0 || index >= len(ec.WorkflowHistory) {
		return ""
	}

	return ec.WorkflowHistory[index].Output
}

// GetMessages retrieves message history from session state
// GetMessages 获取消息历史
func (ec *ExecutionContext) GetMessages() []*types.Message {
	if messages, ok := ec.GetSessionState("messages"); ok {
		if msgList, ok := messages.([]*types.Message); ok {
			return msgList
		}
	}
	return []*types.Message{}
}

// AddMessage adds a message to the history
// AddMessage 添加消息到历史
func (ec *ExecutionContext) AddMessage(msg *types.Message) {
	messages := ec.GetMessages()
	messages = append(messages, msg)
	ec.SetSessionState("messages", messages)
}

// AddMessages adds multiple messages to the history
// AddMessages 批量添加消息
func (ec *ExecutionContext) AddMessages(msgs []*types.Message) {
	messages := ec.GetMessages()
	messages = append(messages, msgs...)
	ec.SetSessionState("messages", messages)
}

// ClearMessages clears all message history
// ClearMessages 清空消息历史
func (ec *ExecutionContext) ClearMessages() {
	ec.SetSessionState("messages", []*types.Message{})
}

// ApplySessionState 用快照初始化会话状态。
func (ec *ExecutionContext) ApplySessionState(snapshot map[string]interface{}) {
	if len(snapshot) == 0 {
		return
	}
	if ec.SessionState == nil {
		ec.SessionState = NewSessionState()
	}
	for k, v := range snapshot {
		ec.SessionState.Set(k, v)
	}
}

// ExportSessionState 导出当前会话状态快照。
func (ec *ExecutionContext) ExportSessionState() map[string]interface{} {
	if ec.SessionState == nil {
		return map[string]interface{}{}
	}
	return ec.SessionState.ToMap()
}

// MergeMetadata merges metadata into execution context.
func (ec *ExecutionContext) MergeMetadata(metadata map[string]interface{}) {
	if len(metadata) == 0 {
		return
	}
	if ec.Metadata == nil {
		ec.Metadata = make(map[string]interface{}, len(metadata))
	}
	for k, v := range metadata {
		ec.Metadata[k] = v
	}
}

// SetRunContextMetadata stores the run context payload into metadata under the
// "run_context" key so callers can inspect correlation identifiers (run_id,
// session_id, workflow_id, user_id, etc.) without reaching into context.Context.
func (ec *ExecutionContext) SetRunContextMetadata(runCtx map[string]interface{}) {
	if len(runCtx) == 0 {
		return
	}
	if ec.Metadata == nil {
		ec.Metadata = make(map[string]interface{}, 1)
	}
	ec.Metadata["run_context"] = runCtx
}
