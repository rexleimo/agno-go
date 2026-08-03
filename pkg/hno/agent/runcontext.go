package agent

import (
	"context"

	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/run"
)

// ctxKey is a private type to avoid key collisions in context
type ctxKey string

const ctxKeyRunContextID ctxKey = "hno.run_context_id"

// WithRunContext returns a child context carrying a run-context identifier
func WithRunContext(ctx context.Context, id string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if id == "" {
		return ctx
	}
	rc, _ := run.FromContext(ctx)
	if rc == nil {
		rc = run.NewContext()
	} else {
		rc = rc.Clone()
	}
	rc.RunID = id
	ctx = run.WithContext(ctx, rc)
	return context.WithValue(ctx, ctxKeyRunContextID, id)
}

// RunContextID retrieves the run-context identifier from context, if present
func RunContextID(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if rc, ok := run.FromContext(ctx); ok && rc != nil && rc.RunID != "" {
		return rc.RunID, true
	}
	v := ctx.Value(ctxKeyRunContextID)
	if s, ok := v.(string); ok && s != "" {
		return s, true
	}
	return "", false
}

// ensureRunContext guarantees a run context exists on the given context.
// ensureRunContext 确保给定的 context 上存在运行上下文。
func ensureRunContext(ctx context.Context) (context.Context, *run.RunContext) {
	if ctx == nil {
		ctx = context.Background()
	}
	rc, ok := run.FromContext(ctx)
	if !ok || rc == nil {
		rc = run.NewContext()
	}
	rc.EnsureRunID()
	ctx = run.WithContext(ctx, rc)
	ctx = context.WithValue(ctx, ctxKeyRunContextID, rc.RunID)
	return ctx, rc
}

// addRunContextMetadata attaches run-context metadata to the output.
// addRunContextMetadata 将运行上下文元数据附加到输出。
func addRunContextMetadata(output *RunOutput, rc *run.RunContext) {
	if output == nil || rc == nil {
		return
	}
	if output.Metadata == nil {
		output.Metadata = make(map[string]interface{})
	}
	if meta := buildRunContextMetadata(rc); len(meta) > 0 {
		output.Metadata["run_context"] = meta
	}
}

// buildRunContextMetadata renders run-context fields as a metadata map.
// buildRunContextMetadata 将运行上下文字段渲染为元数据映射。
func buildRunContextMetadata(rc *run.RunContext) map[string]interface{} {
	if rc == nil {
		return nil
	}
	contextMeta := map[string]interface{}{}
	if rc.RunID != "" {
		contextMeta["run_id"] = rc.RunID
	}
	if rc.ParentRunID != "" {
		contextMeta["parent_run_id"] = rc.ParentRunID
	}
	if rc.SessionID != "" {
		contextMeta["session_id"] = rc.SessionID
	}
	if rc.UserID != "" {
		contextMeta["user_id"] = rc.UserID
	}
	if rc.WorkflowID != "" {
		contextMeta["workflow_id"] = rc.WorkflowID
	}
	if rc.TeamID != "" {
		contextMeta["team_id"] = rc.TeamID
	}
	if rc.Metadata != nil && len(rc.Metadata) > 0 {
		contextMeta["metadata"] = rc.Metadata
	}
	return contextMeta
}

// attachRunContextToRequest copies selected run context fields into the InvokeRequest.Extra
// so model implementations can forward them to downstream providers (for tracing or telemetry).
// attachRunContextToRequest 将选定的运行上下文字段复制到 InvokeRequest.Extra 中，
// 以便模型实现可以将它们转发给下游提供商（用于追踪或遥测）。
func attachRunContextToRequest(ctx context.Context, req *models.InvokeRequest) {
	if req == nil {
		return
	}
	rc, ok := run.FromContext(ctx)
	if !ok || rc == nil {
		return
	}
	meta := buildRunContextMetadata(rc)
	if len(meta) == 0 {
		return
	}
	if req.Extra == nil {
		req.Extra = make(map[string]interface{})
	}
	// Use a stable key so providers can rely on it.
	req.Extra["run_context"] = meta
}
