package agent

import (
	"context"
	"errors"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/hooks"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error) {
	defer a.ClearTempInstructions()

	if input == "" {
		return nil, types.NewInvalidInputError("input cannot be empty", nil)
	}

	ctx, runCtx := ensureRunContext(ctx)
	// Enrich run context with known identifiers so downstream models can access them
	if runCtx != nil && runCtx.UserID == "" && a.UserID != "" {
		runCtx.UserID = a.UserID
	}
	runID := runCtx.RunID

	currentInstructions := a.GetInstructions()
	a.logger.Info("agent run started", "agent_id", a.ID, "input", input)

	initialMessageCount := len(a.Memory.GetMessages(a.UserID))

	if len(a.PreHooks) > 0 {
		a.logger.Debug("executing pre-hooks", "count", len(a.PreHooks))
		hookInput := hooks.NewHookInput(input).
			WithAgentID(a.ID).
			WithMessages([]interface{}{})

		if err := hooks.ExecuteHooks(ctx, a.PreHooks, hookInput); err != nil {
			a.logger.Error("pre-hook failed", "error", err)
			return nil, types.NewInputCheckError("pre-hook validation failed", err)
		}
	}

	userMsg := types.NewUserMessage(input)
	a.Memory.Add(userMsg, a.UserID)

	output := &RunOutput{
		RunID:     runID,
		Status:    RunStatusRunning,
		StartedAt: time.Now().UTC(),
		Metadata:  map[string]interface{}{},
	}

	var finalResponse *types.ModelResponse
	loopCount := 0
	cacheHit := false

	for loopCount < a.MaxLoops {
		if ctxErr := ctx.Err(); ctxErr != nil {
			cancelled := a.markRunCancelled(output, loopCount, cacheHit, ctxErr, initialMessageCount)
			return cancelled, types.NewCancellationError("agent run cancelled", ctxErr)
		}

		loopCount++

		messages := a.Memory.GetMessages(a.UserID)
		if currentInstructions != a.Instructions && currentInstructions != "" {
			messages = a.updateSystemMessage(messages, currentInstructions)
		}

		req := &models.InvokeRequest{Messages: messages}
		if len(a.Toolkits) > 0 {
			req.Tools = toolkit.ToModelToolDefinitions(a.Toolkits)
		}
		attachRunContextToRequest(ctx, req)

		var (
			resp      *types.ModelResponse
			invokeErr error
			fromCache bool
			cacheKey  string
		)

		if a.cacheEnabled {
			cachedResp, key, ok, cacheErr := a.tryCacheGet(ctx, req)
			cacheKey = key
			if cacheErr != nil {
				a.logger.Warn("cache lookup failed", "error", cacheErr)
			} else if ok {
				resp = cachedResp
				fromCache = true
				cacheHit = true
			}
		}

		if !fromCache {
			resp, invokeErr = a.Model.Invoke(ctx, req)
			if invokeErr != nil {
				if errors.Is(invokeErr, context.Canceled) || errors.Is(invokeErr, context.DeadlineExceeded) || ctx.Err() != nil {
					cancelled := a.markRunCancelled(output, loopCount, cacheHit, invokeErr, initialMessageCount)
					return cancelled, types.NewCancellationError("agent run cancelled", invokeErr)
				}
				a.logger.Error("model invocation failed", "error", invokeErr)
				return nil, types.NewAPIError("model invocation failed", invokeErr)
			}
			if a.cacheEnabled {
				if cacheKey == "" {
					cacheKey = a.buildCacheKey(req)
				}
			}
		}

		reasoningContent := a.extractReasoning(ctx, resp)
		assistantMsg := &types.Message{
			Role:             types.RoleAssistant,
			Content:          resp.Content,
			ToolCalls:        resp.ToolCalls,
			ReasoningContent: reasoningContent,
		}
		a.Memory.Add(assistantMsg, a.UserID)

		if !resp.HasToolCalls() {
			if a.cacheEnabled && !fromCache {
				a.tryCacheSet(ctx, cacheKey, resp)
			}
			finalResponse = resp
			break
		}

		a.logger.Info("executing tool calls", "count", len(resp.ToolCalls))
		if err := a.executeToolCalls(ctx, resp.ToolCalls); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				cancelled := a.markRunCancelled(output, loopCount, cacheHit, err, initialMessageCount)
				return cancelled, types.NewCancellationError("agent run cancelled", err)
			}
			a.logger.Error("tool execution failed", "error", err)
			return nil, types.NewToolExecutionError("tool execution failed", err)
		}
	}

	if finalResponse == nil {
		if loopCount >= a.MaxLoops {
			a.logger.Warn("max loops reached", "max_loops", a.MaxLoops)
			return nil, types.NewError(types.ErrCodeUnknown, "max tool calling loops reached", nil)
		}
		return nil, types.NewError(types.ErrCodeUnknown, "no response from model", nil)
	}

	if len(a.PostHooks) > 0 {
		a.logger.Debug("executing post-hooks", "count", len(a.PostHooks))
		hookInput := hooks.NewHookInput(input).
			WithOutput(finalResponse.Content).
			WithAgentID(a.ID).
			WithMessages([]interface{}{})

		if err := hooks.ExecuteHooks(ctx, a.PostHooks, hookInput); err != nil {
			a.logger.Error("post-hook failed", "error", err)
			return nil, types.NewOutputCheckError("post-hook validation failed", err)
		}
	}

	a.logger.Info("agent run completed", "agent_id", a.ID)

	output.Status = RunStatusCompleted
	output.CompletedAt = time.Now().UTC()
	output.Content = finalResponse.Content
	output.Messages = a.Memory.GetMessages(a.UserID)
	output.Metadata["loops"] = loopCount
	output.Metadata["usage"] = finalResponse.Usage
	output.Metadata["cache_hit"] = cacheHit
	addRunContextMetadata(output, runCtx)

	sequence := len(output.Events)
	if finalResponse.Content != "" {
		output.appendEvent(run.NewRunContentEvent(runID, a.ID, string(types.RoleAssistant), finalResponse.Content, sequence))
		sequence++
	}
	output.appendEvent(run.NewRunCompletedEvent(runID, a.ID, "", string(output.Status), finalResponse.Content))

	a.scrubRunOutputWithContext(output, initialMessageCount)

	return output, nil
}
