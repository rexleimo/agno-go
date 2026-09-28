package agent

import (
	"context"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/hooks"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/runner"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// RunStream streams the agent run in the default StreamMessages mode. It is a
// pure wrapper over RunStreamMode(ctx, input, run.StreamMessages) and keeps its
// original signature and behavior; the mode selection (including the
// fail-closed verdict for not-yet-wired modes) lives there.
// RunStream 以默认的 StreamMessages 模式流式执行 agent。它是
// RunStreamMode(ctx, input, run.StreamMessages) 的纯包装，签名与行为不变；
// 模式选择（含未接线模式的 fail-closed 判定）住在 RunStreamMode。
func (a *Agent) RunStream(ctx context.Context, input string) (*RunStreamResult, error) {
	return a.RunStreamMode(ctx, input, run.StreamMessages)
}

// runStreamMessages is the StreamMessages producer: it executes the agent
// through the same runner kernel as Run, driving each model turn through the
// model's streaming API and returning a pair of channels: one for incremental
// content events and one that carries the final RunOutput once aggregation
// completes.
//
// The tool-call loop itself lives in pkg/hno/runner; this function only decides
// how a single turn is invoked (stream, fan chunks out as content events,
// aggregate them) and how the kernel's verdict is reported back to the caller.
// runStreamMessages 是 StreamMessages 的生产者：通过与 Run 相同的 runner 内核执行
// agent，每次模型回合走模型的流式 API，并返回一对通道：一个用于增量内容事件，
// 一个在聚合完成后携带最终的 RunOutput。
//
// tool 循环本身住在 pkg/hno/runner；本函数只决定单次回合怎么调用（流式、把分块扇出为
// 内容事件、聚合它们），以及内核的判定怎么回报给调用方。
//
// Cache is bypassed for streaming runs: a streamed turn is never looked up in
// nor written back to the response cache.
// 流式运行绕过缓存：流式回合既不查缓存，也不回写缓存。
func (a *Agent) runStreamMessages(ctx context.Context, input string) (*RunStreamResult, error) {
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

	var tools []models.ToolDefinition
	if len(a.Toolkits) > 0 {
		tools = toolkit.ToModelToolDefinitions(a.Toolkits)
	}

	eventsCh := make(chan run.BaseRunOutputEvent)
	doneCh := make(chan RunStreamDone, 1)
	result := &RunStreamResult{
		Events: eventsCh,
		Done:   doneCh,
	}

	state := &kernelState{}
	sequence := 0

	go func() {
		defer close(eventsCh)

		// sendDone publishes the kernel's verdict: the reason is stored before
		// the value goes out, so RunStreamResult.StopReason is race-free.
		// sendDone 发布内核的判定：先写入原因再发送终值，使 StopReason 无竞态。
		sendDone := func(final *RunOutput, runErr error, reason runner.StopReason) {
			result.setStopReason(reason.String())
			doneCh <- RunStreamDone{
				Output:     final,
				Err:        runErr,
				StopReason: reason.String(),
			}
		}

		invoker := runner.TurnInvokerFunc(func(turnCtx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
			return a.streamOnce(turnCtx, req, eventsCh, output, &sequence)
		})

		r, err := a.newKernel(ctx, tools, currentInstructions, state, invoker, nil)
		if err != nil {
			a.logger.Error("failed to create runner (stream)", "error", err)
			// The loop never started, so there is no stage to blame; the error
			// carries the detail. 循环从未开始，没有阶段可归因，细节由错误本身携带。
			sendDone(nil, types.NewError(types.ErrCodeUnknown, "failed to create runner", err), runner.StopReason(""))
			return
		}

		finalResponse, _, stopReason, err := r.Run(ctx, a.Memory.GetMessages(a.UserID))
		if err != nil {
			switch stopReason {
			case runner.StopCancelled:
				cancelled := a.markRunCancelled(output, state.turn, false, err, initialMessageCount)
				a.logger.Info("agent run (stream) cancelled", "error", err)
				sendDone(cancelled, types.NewCancellationError("agent run cancelled", err), stopReason)
			case runner.StopToolFailure:
				a.logger.Error("tool execution failed (stream)", "error", err)
				sendDone(nil, types.NewToolExecutionError("tool execution failed", err), stopReason)
			default:
				a.logger.Error("model streaming invocation failed", "error", err)
				sendDone(nil, types.NewAPIError("model streaming invocation failed", err), stopReason)
			}
			return
		}

		if maxLoopsExceeded(stopReason, state.turn, a.MaxLoops) {
			a.logger.Warn("max loops reached (stream)", "max_loops", a.MaxLoops)
			sendDone(nil, types.NewError(types.ErrCodeUnknown, "max tool calling loops reached", nil), stopReason)
			return
		}

		if finalResponse == nil {
			a.logger.Error("no response from model (stream)")
			sendDone(nil, types.NewError(types.ErrCodeUnknown, "no response from model", nil), stopReason)
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
				sendDone(nil, types.NewOutputCheckError("post-hook validation failed", err), stopReason)
				return
			}
		}

		a.logger.Info("agent run (stream) completed", "agent_id", a.ID)

		output.Status = RunStatusCompleted
		output.CompletedAt = time.Now().UTC()
		output.Content = finalResponse.Content
		output.Messages = a.Memory.GetMessages(a.UserID)
		output.StopReason = stopReason.String()
		output.Metadata["loops"] = state.turn
		output.Metadata["usage"] = finalResponse.Usage
		output.Metadata["cache_hit"] = false
		addRunContextMetadata(output, runCtx)

		completed := run.NewRunCompletedEvent(runID, a.ID, "", string(output.Status), output.Content)
		output.appendEvent(completed)

		a.scrubRunOutputWithContext(output, initialMessageCount)

		sendDone(output, nil, stopReason)
	}()

	return result, nil
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
