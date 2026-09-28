package agent

import (
	"context"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/hooks"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/runner"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// Run executes the agent with the given input and returns the finished output.
// Run 用给定输入执行 agent，返回完成后的输出。
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

	var cacheKey string
	var cacheHit bool
	state := &kernelState{}

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

	r, err := a.newKernel(ctx, tools, currentInstructions, state, nil, func(resp *types.ModelResponse) {
		if !a.cacheEnabled || resp.HasToolCalls() || cacheHit {
			return
		}
		if cacheKey == "" {
			messages := a.Memory.GetMessages(a.UserID)
			if currentInstructions != a.Instructions && currentInstructions != "" {
				messages = a.updateSystemMessage(messages, currentInstructions)
			}
			req := &models.InvokeRequest{Messages: messages, Tools: tools}
			attachRunContextToRequest(ctx, req)
			cacheKey = a.buildCacheKey(req)
		}
		a.tryCacheSet(ctx, cacheKey, resp)
	}, nil)
	if err != nil {
		a.logger.Error("failed to create runner", "error", err)
		return nil, types.NewError(types.ErrCodeUnknown, "failed to create runner", err)
	}

	messages := a.Memory.GetMessages(a.UserID)
	finalResponse, _, stopReason, err := r.Run(ctx, messages)

	if err != nil {
		// The kernel classifies the stage it failed in; the agent only maps that
		// verdict to an error. Streaming does the same, so both paths agree.
		// 失败阶段由内核判定，agent 只做映射；流式路径同构，两条路径口径一致。
		if stopReason == runner.StopCancelled {
			// 已经取消，标记后返回
			a.markRunCancelled(output, state.turn, false, err, initialMessageCount)
			return output, types.NewCancellationError("agent run cancelled", err)
		}
		a.logger.Error("runner execution failed", "error", err)
		return nil, types.NewAPIError("runner execution failed", err)
	}

	if maxLoopsExceeded(stopReason, state.turn, a.MaxLoops) {
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
	output.Metadata["loops"] = state.turn
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
