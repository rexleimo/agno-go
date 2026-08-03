package agent

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/agno/hooks"
	"github.com/rexleimo/agno-go/pkg/agno/models"
	"github.com/rexleimo/agno-go/pkg/agno/run"
	"github.com/rexleimo/agno-go/pkg/agno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/agno/types"
)

func singleDoneChannel(output *RunOutput, err error) <-chan RunStreamDone {
	ch := make(chan RunStreamDone, 1)
	ch <- RunStreamDone{
		Output: output,
		Err:    err,
	}
	return ch
}

// RunStream executes the agent using the model's streaming API and returns
// a pair of channels: one for incremental content events and one that carries
// the final RunOutput once aggregation completes.
//
// Unlike the initial single-pass implementation, the streaming path now runs
// the full tool-call loop: tool calls returned by the model are executed and
// their results are fed back into the conversation before the next streaming
// round, mirroring the synchronous Run behaviour.
// RunStream 使用模型的流式 API 执行 agent，并返回一对通道：
// 一个用于增量内容事件，一个在聚合完成后携带最终的 RunOutput。
//
// 与最初的单遍实现不同，流式路径现在运行完整的工具调用循环：
// 模型返回的工具调用会被执行，其结果在下一轮流式调用前回填到对话中，
// 与同步 Run 的行为保持一致。
func (a *Agent) RunStream(ctx context.Context, input string) (*RunStreamResult, error) {
	defer a.ClearTempInstructions()

	if strings.TrimSpace(input) == "" {
		return nil, types.NewInvalidInputError("input cannot be empty", nil)
	}

	ctx, runCtx := ensureRunContext(ctx)
	if runCtx != nil && runCtx.UserID == "" && a.UserID != "" {
		runCtx.UserID = a.UserID
	}
	runID := runCtx.RunID

	currentInstructions := a.GetInstructions()
	a.logger.Info("agent run (stream) started", "agent_id", a.ID, "input", input)

	initialMessageCount := len(a.Memory.GetMessages(a.UserID))

	if len(a.PreHooks) > 0 {
		a.logger.Debug("executing pre-hooks (stream)", "count", len(a.PreHooks))
		hookInput := hooks.NewHookInput(input).
			WithAgentID(a.ID).
			WithMessages([]interface{}{})

		if err := hooks.ExecuteHooks(ctx, a.PreHooks, hookInput); err != nil {
			a.logger.Error("pre-hook failed (stream)", "error", err)
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

	eventsCh := make(chan run.BaseRunOutputEvent)
	doneCh := make(chan RunStreamDone, 1)

	go func() {
		defer close(eventsCh)

		var finalResponse *types.ModelResponse
		loopCount := 0
		sequence := 0

		for loopCount < a.MaxLoops {
			if ctxErr := ctx.Err(); ctxErr != nil {
				cancelled := a.markRunCancelled(output, loopCount, false, ctxErr, initialMessageCount)
				doneCh <- RunStreamDone{
					Output: cancelled,
					Err:    types.NewCancellationError("agent run cancelled", ctxErr),
				}
				return
			}

			loopCount++

			// Prepare messages and request for this streaming round.
			// 为本轮流式调用准备消息与请求。
			messages := a.Memory.GetMessages(a.UserID)
			if currentInstructions != a.Instructions && currentInstructions != "" {
				messages = a.updateSystemMessage(messages, currentInstructions)
			}

			req := &models.InvokeRequest{Messages: messages}
			if len(a.Toolkits) > 0 {
				req.Tools = toolkit.ToModelToolDefinitions(a.Toolkits)
			}
			attachRunContextToRequest(ctx, req)

			resp, err := a.streamOnce(ctx, req, eventsCh, output, &sequence)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
					cancelled := a.markRunCancelled(output, loopCount, false, err, initialMessageCount)
					doneCh <- RunStreamDone{
						Output: cancelled,
						Err:    types.NewCancellationError("agent run cancelled", err),
					}
					return
				}
				a.logger.Error("model streaming invocation failed", "error", err)
				doneCh <- RunStreamDone{
					Output: nil,
					Err:    types.NewAPIError("model streaming invocation failed", err),
				}
				return
			}

			// Store the assistant turn (content + tool calls + reasoning).
			// 存储助手回合（内容 + 工具调用 + 推理）。
			reasoningContent := a.extractReasoning(ctx, resp)
			assistantMsg := &types.Message{
				Role:             types.RoleAssistant,
				Content:          resp.Content,
				ToolCalls:        resp.ToolCalls,
				ReasoningContent: reasoningContent,
			}
			a.Memory.Add(assistantMsg, a.UserID)

			if !resp.HasToolCalls() {
				finalResponse = resp
				break
			}

			// Execute tool calls; results are appended to memory so the next
			// streaming round sees them.
			// 执行工具调用；结果追加到 memory，使下一轮流式调用能看到它们。
			a.logger.Info("executing tool calls (stream)", "count", len(resp.ToolCalls))
			if err := a.executeToolCalls(ctx, resp.ToolCalls); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
					cancelled := a.markRunCancelled(output, loopCount, false, err, initialMessageCount)
					doneCh <- RunStreamDone{
						Output: cancelled,
						Err:    types.NewCancellationError("agent run cancelled", err),
					}
					return
				}
				a.logger.Error("tool execution failed (stream)", "error", err)
				doneCh <- RunStreamDone{
					Output: nil,
					Err:    types.NewToolExecutionError("tool execution failed", err),
				}
				return
			}
		}

		if finalResponse == nil {
			a.logger.Warn("max loops reached (stream)", "max_loops", a.MaxLoops)
			doneCh <- RunStreamDone{
				Output: nil,
				Err:    types.NewError(types.ErrCodeUnknown, "max tool calling loops reached", nil),
			}
			return
		}

		if len(a.PostHooks) > 0 {
			a.logger.Debug("executing post-hooks (stream)", "count", len(a.PostHooks))
			hookInput := hooks.NewHookInput(input).
				WithOutput(finalResponse.Content).
				WithAgentID(a.ID).
				WithMessages([]interface{}{})

			if err := hooks.ExecuteHooks(ctx, a.PostHooks, hookInput); err != nil {
				a.logger.Error("post-hook failed (stream)", "error", err)
				doneCh <- RunStreamDone{
					Output: nil,
					Err:    types.NewOutputCheckError("post-hook validation failed", err),
				}
				return
			}
		}

		a.logger.Info("agent run (stream) completed", "agent_id", a.ID)

		output.Status = RunStatusCompleted
		output.CompletedAt = time.Now().UTC()
		output.Content = finalResponse.Content
		output.Messages = a.Memory.GetMessages(a.UserID)
		output.Metadata["loops"] = loopCount
		output.Metadata["usage"] = finalResponse.Usage
		output.Metadata["cache_hit"] = false
		addRunContextMetadata(output, runCtx)

		completed := run.NewRunCompletedEvent(runID, a.ID, "", string(output.Status), output.Content)
		output.appendEvent(completed)

		a.scrubRunOutputWithContext(output, initialMessageCount)

		doneCh <- RunStreamDone{
			Output: output,
			Err:    nil,
		}
	}()

	return &RunStreamResult{
		Events: eventsCh,
		Done:   doneCh,
	}, nil
}

// streamOnce consumes a single streaming model invocation: it fans chunks out
// to eventsCh as incremental content events while aggregating them into one
// ModelResponse. It returns the aggregated response (with tool calls, if any).
// streamOnce 消费单次流式模型调用：将分块作为增量内容事件扇出到 eventsCh，
// 同时将它们聚合为一个 ModelResponse。返回聚合后的响应（可能含工具调用）。
func (a *Agent) streamOnce(ctx context.Context, req *models.InvokeRequest, eventsCh chan<- run.BaseRunOutputEvent, output *RunOutput, sequence *int) (*types.ModelResponse, error) {
	stream, err := a.Model.InvokeStream(ctx, req)
	if err != nil {
		return nil, err
	}

	aggregatorCh := make(chan types.ResponseChunk)
	doneAgg := make(chan struct{})

	var (
		resp   *types.ModelResponse
		aggErr error
	)

	go func() {
		defer close(doneAgg)
		resp, aggErr = AggregateResponseStream(ctx, aggregatorCh)
	}()

	aggregatorClosed := false
	closeAggregator := func() {
		if !aggregatorClosed {
			close(aggregatorCh)
			aggregatorClosed = true
		}
	}

	runID := output.RunID
	agentID := a.ID

	for {
		select {
		case <-ctx.Done():
			closeAggregator()
			<-doneAgg
			return nil, ctx.Err()

		case chunk, ok := <-stream:
			if !ok {
				// Streaming complete; finalise aggregation.
				// 流式调用完成；结束聚合。
				closeAggregator()
				<-doneAgg

				if aggErr != nil {
					return nil, aggErr
				}
				if resp == nil {
					resp = &types.ModelResponse{}
				}
				return resp, nil
			}

			// Forward chunk to aggregator so we can reconstruct a final response.
			// 将分块转发给聚合器，以便重建最终响应。
			if !aggregatorClosed {
				select {
				case aggregatorCh <- chunk:
				case <-ctx.Done():
					closeAggregator()
					<-doneAgg
					return nil, ctx.Err()
				}
			}

			if chunk.Error != nil {
				closeAggregator()
				<-doneAgg
				return nil, chunk.Error
			}

			if chunk.Content != "" {
				evt := run.NewRunContentEvent(runID, agentID, string(types.RoleAssistant), chunk.Content, *sequence)
				*sequence++
				output.appendEvent(evt)

				select {
				case eventsCh <- evt:
				case <-ctx.Done():
					closeAggregator()
					<-doneAgg
					return nil, ctx.Err()
				}
			}
		}
	}
}
