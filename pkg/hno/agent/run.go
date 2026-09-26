package agent

import (
	"context"
	"errors"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/hooks"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/runner"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

func (a *Agent) Run(ctx context.Context, input string) (*RunOutput, error) {
	defer a.ClearTempInstructions()

	if input == "" {
		return nil, types.NewInvalidInputError("input cannot be empty", nil)
	}

	ctx, runCtx := ensureRunContext(ctx)
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

	var tools []models.ToolDefinition
	if len(a.Toolkits) > 0 {
		tools = toolkit.ToModelToolDefinitions(a.Toolkits)
	}

	executor := &agentToolExecutor{agent: a}

	var cacheKey string
	var cacheHit bool
	turn := 0

	// Cache lookup before runner (old code did cache check in loop)
	if a.cacheEnabled {
		messages := a.Memory.GetMessages(a.UserID)
		if currentInstructions != a.Instructions && currentInstructions != "" {
			messages = a.updateSystemMessage(messages, currentInstructions)
		}
		req := &models.InvokeRequest{Messages: messages, Tools: tools}
		attachRunContextToRequest(ctx, req)

		cachedResp, key, ok, cacheErr := a.tryCacheGet(ctx, req)
		cacheKey = key
		if cacheErr != nil {
			a.logger.Warn("cache lookup failed", "error", cacheErr)
		} else if ok {
			// Cache hit: write cached response to Memory and return immediately
			reasoningContent := a.extractReasoning(ctx, cachedResp)
			assistantMsg := &types.Message{
				Role:             types.RoleAssistant,
				Content:          cachedResp.Content,
				ToolCalls:        cachedResp.ToolCalls,
				ReasoningContent: reasoningContent,
			}
			a.Memory.Add(assistantMsg, a.UserID)
			cacheHit = true

			output.Status = RunStatusCompleted
			output.CompletedAt = time.Now().UTC()
			output.Content = cachedResp.Content
			output.Messages = a.Memory.GetMessages(a.UserID)
			output.StopReason = string(runner.StopNoToolCalls)
			output.Metadata["loops"] = 0
			output.Metadata["usage"] = cachedResp.Usage
			output.Metadata["cache_hit"] = true
			addRunContextMetadata(output, runCtx)

			if cachedResp.Content != "" {
				output.appendEvent(run.NewRunContentEvent(runID, a.ID, string(types.RoleAssistant), cachedResp.Content, 0))
			}
			output.appendEvent(run.NewRunCompletedEvent(runID, a.ID, "", string(output.Status), cachedResp.Content))

			a.scrubRunOutputWithContext(output, initialMessageCount)
			return output, nil
		}
	}

	r, err := runner.New(runner.Config{
		Model:         a.Model,
		Tools:         tools,
		MaxTurns:      a.MaxLoops,
		ToolCallLimit: a.ToolCallLimit,
		MessageBuilder: &agentMessageBuilder{
			agent:        a,
			instructions: currentInstructions,
			tools:        tools,
		},
		ToolExecutor: executor,
		OnStep: func(evt runner.StepEvent) {
			if evt.State == runner.StateAwaitModel {
				turn = evt.Turn
				reasoningContent := a.extractReasoning(ctx, evt.Response)
				assistantMsg := &types.Message{
					Role:             types.RoleAssistant,
					Content:          evt.Response.Content,
					ToolCalls:        evt.Response.ToolCalls,
					ReasoningContent: reasoningContent,
				}
				a.Memory.Add(assistantMsg, a.UserID)

				// Cache write: only if no tool calls and not from cache
				if a.cacheEnabled && !evt.Response.HasToolCalls() && !cacheHit {
					if cacheKey == "" {
						messages := a.Memory.GetMessages(a.UserID)
						if currentInstructions != a.Instructions && currentInstructions != "" {
							messages = a.updateSystemMessage(messages, currentInstructions)
						}
						req := &models.InvokeRequest{Messages: messages, Tools: tools}
						attachRunContextToRequest(ctx, req)
						cacheKey = a.buildCacheKey(req)
					}
					a.tryCacheSet(ctx, cacheKey, evt.Response)
				}
			}
		},
		OnSkippedToolCalls: func(skipped []types.ToolCall) {
			for _, c := range skipped {
				msg := &types.Message{
					Role:       types.RoleTool,
					ToolCallID: c.ID,
					Content:    "tool call limit reached; call not executed",
				}
				a.Memory.Add(msg, a.UserID)
			}
		},
		Logger: a.logger,
	})
	if err != nil {
		a.logger.Error("failed to create runner", "error", err)
		return nil, types.NewError(types.ErrCodeUnknown, "failed to create runner", err)
	}

	messages := a.Memory.GetMessages(a.UserID)
	finalResponse, _, stopReason, err := r.Run(ctx, messages)

	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			// 已经取消，标记后返回
			a.markRunCancelled(output, 0, false, err, initialMessageCount)
			return output, types.NewCancellationError("agent run cancelled", err)
		}
		a.logger.Error("runner execution failed", "error", err)
		return nil, types.NewAPIError("runner execution failed", err)
	}

	if stopReason == runner.StopLimitReached && turn >= a.MaxLoops {
		a.logger.Warn("max loops reached", "max_loops", a.MaxLoops)
		return nil, types.NewError(types.ErrCodeUnknown, "max tool calling loops reached", nil)
	}

	if finalResponse == nil {
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
	output.StopReason = string(stopReason)
	output.Metadata["loops"] = turn
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
