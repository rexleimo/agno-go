package workflow

import (
	"context"
	"fmt"
)

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
