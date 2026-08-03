package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Workflow represents a multi-step process
// Workflow 代表多步骤流程
type Workflow struct {
	ID     string
	Name   string
	Steps  []Node
	logger *slog.Logger

	// History-related fields
	// 历史相关字段
	enableHistory     bool
	historyStore      WorkflowStorage
	numHistoryRuns    int
	addHistoryToSteps bool
}

// Node represents a node in the workflow graph
type Node interface {
	Execute(ctx context.Context, input *ExecutionContext) (*ExecutionContext, error)
	GetID() string
	GetType() NodeType
}

// NodeType represents the type of workflow node
type NodeType string

const (
	NodeTypeStep      NodeType = "step"
	NodeTypeCondition NodeType = "condition"
	NodeTypeLoop      NodeType = "loop"
	NodeTypeParallel  NodeType = "parallel"
	NodeTypeRouter    NodeType = "router"
)

// Config contains workflow configuration
// Config 包含工作流配置
type Config struct {
	ID     string
	Name   string
	Steps  []Node
	Logger *slog.Logger

	// EnableHistory enables workflow history tracking
	// EnableHistory 启用工作流历史跟踪
	EnableHistory bool `json:"enable_history"`

	// HistoryStore is the storage backend for workflow history
	// HistoryStore 是工作流历史的存储后端
	HistoryStore WorkflowStorage `json:"-"`

	// NumHistoryRuns is the number of recent runs to include in history context
	// NumHistoryRuns 是历史上下文中包含的最近运行数量
	NumHistoryRuns int `json:"num_history_runs"`

	// AddHistoryToSteps automatically adds history context to all steps
	// AddHistoryToSteps 自动将历史上下文添加到所有步骤
	AddHistoryToSteps bool `json:"add_history_to_steps"`
}

// New creates a new workflow
// New 创建新的工作流
func New(config Config) (*Workflow, error) {
	if config.ID == "" {
		config.ID = fmt.Sprintf("workflow-%s", config.Name)
	}

	if config.Name == "" {
		config.Name = config.ID
	}

	if config.Logger == nil {
		config.Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	// 历史配置验证和默认值
	// History configuration validation and defaults
	if config.EnableHistory && config.HistoryStore == nil {
		// 使用默认内存存储
		// Use default memory storage
		config.HistoryStore = NewMemoryStorage(100)
	}

	if config.NumHistoryRuns <= 0 {
		config.NumHistoryRuns = 3 // 默认包含最近 3 次运行
	}

	return &Workflow{
		ID:                config.ID,
		Name:              config.Name,
		Steps:             config.Steps,
		logger:            config.Logger,
		enableHistory:     config.EnableHistory,
		historyStore:      config.HistoryStore,
		numHistoryRuns:    config.NumHistoryRuns,
		addHistoryToSteps: config.AddHistoryToSteps,
	}, nil
}

// Run executes the workflow.
// sessionID 参数可选,为空则自动生成。
func (w *Workflow) Run(ctx context.Context, input string, sessionID string, opts ...RunOption) (*ExecutionContext, error) {
	options := evaluateOptions(opts)
	if options.mediaError != nil {
		return nil, types.NewInvalidInputError("invalid media payload", options.mediaError)
	}

	if input == "" && len(options.mediaPayload) == 0 {
		return nil, types.NewInvalidInputError("input cannot be empty", nil)
	}

	if sessionID == "" {
		sessionID = generateSessionID()
	}

	runCtx := options.runContext
	if runCtx == nil {
		if existing, ok := run.FromContext(ctx); ok && existing != nil {
			runCtx = existing.Clone()
		}
	}
	if runCtx == nil {
		runCtx = run.NewContext()
	}
	if runCtx.SessionID == "" {
		runCtx.SessionID = sessionID
	} else {
		sessionID = runCtx.SessionID
	}
	if runCtx.WorkflowID == "" && w.ID != "" {
		runCtx.WorkflowID = w.ID
	}
	runCtx.EnsureRunID()
	ctx = run.WithContext(ctx, runCtx)

	w.logger.Info("workflow started",
		"workflow_id", w.ID,
		"session_id", sessionID,
		"steps", len(w.Steps),
		"resume_from", options.resumeFromStep)

	metrics := NewWorkflowMetrics()
	metrics.Start()
	defer metrics.Stop()

	execCtx := NewExecutionContextWithSession(input, sessionID, options.userID)
	execCtx.SetRunContextMetadata(runContextMetadata(runCtx))
	execCtx.ApplySessionState(options.sessionState)
	execCtx.MergeMetadata(options.metadata)

	if len(options.mediaPayload) > 0 {
		execCtx.SetSessionState("media_payload", options.mediaPayload)
	}

	if w.enableHistory && w.historyStore != nil {
		if err := w.loadHistory(ctx, execCtx); err != nil {
			w.logger.Error("failed to load history", "error", err)
		}
	}

	if w.enableHistory {
		execCtx.SetSessionState("workflow_history_config", &WorkflowHistoryConfig{
			AddHistoryToSteps: w.addHistoryToSteps,
			NumHistoryRuns:    w.numHistoryRuns,
		})
	}

	var workflowRun *WorkflowRun
	if w.enableHistory {
		runID := runCtx.RunID
		workflowRun = NewWorkflowRun(runID, sessionID, w.ID, input)
		workflowRun.MarkStarted()
		if options.resumeFromStep != "" {
			workflowRun.ResumedFrom = options.resumeFromStep
		}
		if len(options.mediaPayload) > 0 {
			if workflowRun.Metadata == nil {
				workflowRun.Metadata = make(map[string]interface{})
			}
			workflowRun.Metadata["media"] = options.mediaPayload
		}
		// Attach run context identifiers to the stored workflow run so history
		// consumers can correlate executions with upstream orchestrators.
		if rcMeta := runContextMetadata(runCtx); len(rcMeta) > 0 {
			if workflowRun.Metadata == nil {
				workflowRun.Metadata = make(map[string]interface{})
			}
			workflowRun.Metadata["run_context"] = rcMeta
		}
	}

	startIdx := 0
	if options.resumeFromStep != "" {
		found := false
		for i, step := range w.Steps {
			if step.GetID() == options.resumeFromStep {
				startIdx = i
				found = true
				break
			}
		}
		if !found {
			return nil, types.NewInvalidInputError("resume step not found", fmt.Errorf("step %s not in workflow", options.resumeFromStep))
		}
	}

	var lastStepID string

	for idx := startIdx; idx < len(w.Steps); idx++ {
		step := w.Steps[idx]

		select {
		case <-ctx.Done():
			if workflowRun != nil {
				reason := ctx.Err()
				snapshot := execCtx.ExportSessionState()
				workflowRun.ApplyCancellation(reason.Error(), lastStepID, snapshot)
				// Create a new context with timeout for persistence operations
				// as the original context is cancelled
				persistCtx, persistCancel := context.WithTimeout(context.Background(), 5*time.Second)
				metrics.Stop()
				w.saveRun(persistCtx, sessionID, workflowRun, metrics)
				w.saveCancellation(persistCtx, sessionID, &CancellationRecord{
					RunID:      workflowRun.RunID,
					Reason:     reason.Error(),
					StepID:     lastStepID,
					Snapshot:   snapshot,
					OccurredAt: time.Now(),
				})
				persistCancel()
			}
			return nil, ctx.Err()
		default:
		}

		sequence := idx - startIdx + 1
		currentStepID := step.GetID()
		lastStepID = currentStepID
		if workflowRun != nil {
			workflowRun.LastStepID = currentStepID
		}

		w.logger.Info("executing step",
			"step_id", currentStepID,
			"step_type", step.GetType(),
			"sequence", sequence)

		result, err := step.Execute(ctx, execCtx)
		if err != nil {
			w.logger.Error("step execution failed",
				"step_id", currentStepID,
				"error", err)

			if workflowRun != nil {
				workflowRun.LastStepID = currentStepID
				workflowRun.MarkFailed(err)
				metrics.Stop()
				w.saveRun(ctx, sessionID, workflowRun, metrics)
			}

			return nil, types.NewError(types.ErrCodeUnknown, fmt.Sprintf("step %s failed", currentStepID), err)
		}

		execCtx = result
		if workflowRun != nil {
			if events := extractStepEvents(execCtx, currentStepID); len(events) > 0 {
				workflowRun.AddEvents(events)
			}
		}
	}

	if workflowRun != nil {
		workflowRun.MarkCompleted(execCtx.Output)
		workflowRun.LastStepID = lastStepID
		workflowRun.Messages = extractMessages(execCtx)
		metrics.Stop()
		w.saveRun(ctx, sessionID, workflowRun, metrics)
	}

	metrics.Stop()
	recordWorkflowMetrics(execCtx, metrics)

	w.logger.Info("workflow completed",
		"workflow_id", w.ID,
		"session_id", sessionID)

	return execCtx, nil
}

// AddStep adds a step to the workflow
func (w *Workflow) AddStep(step Node) {
	w.Steps = append(w.Steps, step)
}

// loadHistory 从存储加载历史并添加到上下文
// loadHistory loads history from storage and adds to context
func (w *Workflow) loadHistory(ctx context.Context, execCtx *ExecutionContext) error {
	if w.historyStore == nil {
		return nil
	}

	// 获取或创建 session
	// Get or create session
	session, err := w.historyStore.GetSession(ctx, execCtx.SessionID)
	if err != nil {
		if err == ErrSessionNotFound {
			// 创建新 session
			// Create new session
			session, err = w.historyStore.CreateSession(
				ctx,
				execCtx.SessionID,
				w.ID,
				execCtx.UserID,
			)
			if err != nil {
				return fmt.Errorf("failed to create session: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get session: %w", err)
		}
	}

	// 获取历史记录
	// Get history
	history := session.GetHistory(w.numHistoryRuns)
	if len(history) == 0 {
		return nil
	}

	// 直接设置到 ExecutionContext
	// Set directly to ExecutionContext
	execCtx.SetWorkflowHistory(history)

	// 如果配置了添加历史到步骤
	// If configured to add history to steps
	if w.addHistoryToSteps {
		historyContext := session.GetHistoryContext(w.numHistoryRuns)
		execCtx.SetHistoryContext(historyContext)

		// 也保存到 session state (向后兼容)
		// Also save to session state (backward compatibility)
		execCtx.SetSessionState("workflow_history_context", historyContext)
	}

	// 将历史数据存储在上下文中（向后兼容）
	// Store history data in context (backward compatibility)
	execCtx.SetSessionState("workflow_history", history)
	execCtx.SetSessionState("workflow_session", session)

	w.logger.Debug("loaded history",
		"session_id", execCtx.SessionID,
		"history_count", len(history))

	return nil
}

// saveRun 保存运行记录到存储
// saveRun saves run record to storage
func (w *Workflow) saveRun(ctx context.Context, sessionID string, run *WorkflowRun, metrics *WorkflowMetrics) error {
	if w.historyStore == nil {
		return nil
	}

	// 获取 session
	// Get session
	session, err := w.historyStore.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	if metrics != nil {
		attachWorkflowMetrics(run, metrics)
	}

	// 添加运行记录
	// Add run record
	session.AddRun(run)

	// 更新 session
	// Update session
	if err := w.historyStore.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	w.logger.Debug("saved run",
		"session_id", sessionID,
		"run_id", run.RunID,
		"status", run.Status)

	return nil
}

func attachWorkflowMetrics(run *WorkflowRun, metrics *WorkflowMetrics) {
	if run == nil || metrics == nil {
		return
	}
	if run.Metadata == nil {
		run.Metadata = make(map[string]interface{})
	}
	snapshot := metrics.Snapshot()
	if len(snapshot) == 0 {
		return
	}
	run.Metadata["metrics"] = snapshot
}

func recordWorkflowMetrics(execCtx *ExecutionContext, metrics *WorkflowMetrics) {
	if execCtx == nil || metrics == nil {
		return
	}
	snapshot := metrics.Snapshot()
	if len(snapshot) == 0 {
		return
	}
	if execCtx.Metadata == nil {
		execCtx.Metadata = make(map[string]interface{})
	}
	execCtx.Metadata["workflow_metrics"] = snapshot
}

func (w *Workflow) saveCancellation(ctx context.Context, sessionID string, record *CancellationRecord) error {
	if w.historyStore == nil || record == nil {
		return nil
	}

	session, err := w.historyStore.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session for cancellation: %w", err)
	}

	session.AddCancellation(record)
	if err := w.historyStore.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to update session cancellation: %w", err)
	}

	w.logger.Debug("saved cancellation",
		"session_id", sessionID,
		"run_id", record.RunID,
		"step_id", record.StepID)
	return nil
}

// extractMessages 从执行上下文提取消息
// extractMessages extracts messages from execution context
func extractMessages(execCtx *ExecutionContext) []*types.Message {
	// 从 session state 中提取消息历史
	// Extract message history from session state
	if messages, ok := execCtx.GetSessionState("messages"); ok {
		if msgList, ok := messages.([]*types.Message); ok {
			return msgList
		}
	}

	// 如果没有消息历史，创建基本的输入/输出消息
	// If no message history, create basic input/output messages
	return []*types.Message{
		types.NewUserMessage(execCtx.Input),
		types.NewAssistantMessage(execCtx.Output),
	}
}

func extractStepEvents(execCtx *ExecutionContext, stepID string) run.Events {
	if execCtx == nil || stepID == "" {
		return nil
	}
	if raw, ok := execCtx.Get(stepEventsKey(stepID)); ok {
		if events, ok := raw.(run.Events); ok {
			return events
		}
	}
	return nil
}

// generateSessionID 生成唯一的 session ID
// generateSessionID generates a unique session ID
func generateSessionID() string {
	return "session-" + uuid.New().String()
}

// generateRunID 生成唯一的 run ID
// generateRunID generates a unique run ID
func generateRunID() string {
	return "run-" + uuid.New().String()
}

// runContextMetadata extracts a serialisable view of the run context identifiers
// suitable for storing in workflow metadata or logs.
func runContextMetadata(rc *run.RunContext) map[string]interface{} {
	if rc == nil {
		return nil
	}
	meta := map[string]interface{}{}
	if rc.RunID != "" {
		meta["run_id"] = rc.RunID
	}
	if rc.SessionID != "" {
		meta["session_id"] = rc.SessionID
	}
	if rc.WorkflowID != "" {
		meta["workflow_id"] = rc.WorkflowID
	}
	if rc.UserID != "" {
		meta["user_id"] = rc.UserID
	}
	if rc.ParentRunID != "" {
		meta["parent_run_id"] = rc.ParentRunID
	}
	if rc.TeamID != "" {
		meta["team_id"] = rc.TeamID
	}
	if rc.Metadata != nil && len(rc.Metadata) > 0 {
		meta["metadata"] = rc.Metadata
	}
	if len(meta) == 0 {
		return nil
	}
	return meta
}
