package agent

import (
	"log/slog"
	"sync"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/cache"
	"github.com/rexleimo/agno-go/pkg/hno/hooks"
	"github.com/rexleimo/agno-go/pkg/hno/memory"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

const defaultCacheTTL = 5 * time.Minute

// RunStatus represents the lifecycle status of a run.
type RunStatus string

const (
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusCancelled RunStatus = "cancelled"
	RunStatusError     RunStatus = "error"
)

// Agent represents an AI agent
type Agent struct {
	ID            string
	Name          string
	Model         models.Model
	Toolkits      []toolkit.Toolkit
	Memory        memory.Memory
	Instructions  string
	MaxLoops      int          // Maximum tool calling loops
	ToolCallLimit int          // 工具调用次数上限（本次运行累计，0 = 不限）/ Tool call limit (per run, 0 = unlimited)
	UserID        string       // User ID for multi-tenant memory isolation / 多租户内存隔离的用户ID
	PreHooks      []hooks.Hook // Hooks executed before processing input
	PostHooks     []hooks.Hook // Hooks executed after generating output
	logger        *slog.Logger
	cache         cache.Provider
	cacheTTL      time.Duration
	cacheEnabled  bool

	// Storage control / 存储控制
	storeToolMessages    bool // Whether to store tool messages in RunOutput / 是否在 RunOutput 中存储工具消息
	storeHistoryMessages bool // Whether to store history messages in RunOutput / 是否在 RunOutput 中存储历史消息

	// Temporary instructions support for workflow history injection
	// 临时 instructions 支持,用于工作流历史注入
	tempInstructions string       // Temporary instructions (single execution only) / 临时指令（仅单次执行）
	instructionsMu   sync.RWMutex // Protects instructions modification / 保护指令修改

	// functionIndex is a lazily built merged map of all toolkit functions
	// for O(1) tool dispatch.
	// functionIndex 是懒构建的合并函数索引，用于 O(1) 工具分发。
	functionIndex map[string]*toolkit.Function
}

// Config contains agent configuration

// RunOutput contains the result of agent execution
type RunOutput struct {
	RunID              string                 `json:"run_id,omitempty"`
	Status             RunStatus              `json:"status"`
	StartedAt          time.Time              `json:"started_at"`
	CompletedAt        time.Time              `json:"completed_at"`
	CancellationReason string                 `json:"cancellation_reason,omitempty"`
	Content            string                 `json:"content"`
	Messages           []*types.Message       `json:"messages"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	Events             run.Events             `json:"events,omitempty"`
	StopReason         string                 `json:"stop_reason,omitempty"` // 循环终止原因 / Reason the loop terminated
}

// RunStreamDone represents the terminal result of a streaming run.
// It always carries either a non-nil Output or a non-nil Err.
type RunStreamDone struct {
	Output     *RunOutput
	Err        error
	StopReason string
}

// RunStreamResult groups the channels produced by a streaming run.
// Events streams incremental RunContentEvent values while Done carries
// the final RunOutput once the model stream completes.
type RunStreamResult struct {
	Events <-chan run.BaseRunOutputEvent
	Done   <-chan RunStreamDone

	// stopReasonMu guards stopReason, which the run goroutine writes before it
	// sends on Done.
	// stopReasonMu 保护 stopReason；运行协程在发送 Done 之前写入该值。
	stopReasonMu sync.Mutex
	stopReason   string
}

// StopReason reports why the loop terminated. Read it after receiving the
// terminal value from Done; before that it is empty.
// StopReason 报告循环终止原因。请在收到 Done 的终值后读取；在此之前它为空。
func (r *RunStreamResult) StopReason() string {
	r.stopReasonMu.Lock()
	defer r.stopReasonMu.Unlock()
	return r.stopReason
}

func (r *RunStreamResult) setStopReason(reason string) {
	r.stopReasonMu.Lock()
	r.stopReason = reason
	r.stopReasonMu.Unlock()
}

// ClearMemory 清除此用户的Agent对话历史
func (a *Agent) ClearMemory() {
	a.Memory.Clear(a.UserID)
	// Re-add system message
	// 重新添加系统消息
	if a.Instructions != "" {
		a.Memory.Add(types.NewSystemMessage(a.Instructions), a.UserID)
	}
}

// GetID returns the agent ID
// GetID 返回 agent ID
func (a *Agent) GetID() string {
	return a.ID
}

// GetInstructions returns the current instructions (temporary or permanent)
// GetInstructions 返回当前指令（临时或永久）
func (a *Agent) GetInstructions() string {
	a.instructionsMu.RLock()
	defer a.instructionsMu.RUnlock()

	// Temporary instructions take precedence
	// 临时指令优先
	if a.tempInstructions != "" {
		return a.tempInstructions
	}
	return a.Instructions
}

// SetInstructions permanently sets the agent's instructions
// SetInstructions 永久设置 agent 的指令
func (a *Agent) SetInstructions(instructions string) {
	a.instructionsMu.Lock()
	defer a.instructionsMu.Unlock()

	a.Instructions = instructions
}

// SetTempInstructions temporarily sets instructions (only affects next Run)
// SetTempInstructions 临时设置指令（仅影响下一次 Run）
func (a *Agent) SetTempInstructions(instructions string) {
	a.instructionsMu.Lock()
	defer a.instructionsMu.Unlock()

	a.tempInstructions = instructions
}

// ClearTempInstructions clears temporary instructions
// ClearTempInstructions 清除临时指令
func (a *Agent) ClearTempInstructions() {
	a.instructionsMu.Lock()
	defer a.instructionsMu.Unlock()

	a.tempInstructions = ""
}

// updateSystemMessage updates or adds system message with new instructions
// updateSystemMessage 更新或添加带有新指令的系统消息
// 它会过滤掉 Role == RoleTool 的消息，并清除其他消息中的工具相关字段
