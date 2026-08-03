package workflow

import (
	"context"
	"fmt"

	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// executeResult carries the outcome of a step-execution run.
// executeResult 携带步骤执行的结果。
type executeResult struct {
	execCtx    *ExecutionContext
	lastStepID string
	events     map[string]run.Events // stepID -> events collected during execution
	err        error
}

// executeSteps runs the workflow steps from startIdx to the end on top of the
// shared run.Loop kernel (builder/executor separation): no persistence, no
// run-context assembly — those live in Workflow.Run.
//
// executeSteps 在共享 run.Loop 内核之上从 startIdx 运行工作流步骤直至
// 结束（builder/executor 分离）：不做持久化、不装配运行上下文——
// 这些都在 Workflow.Run 中。
func executeSteps(ctx context.Context, steps []Node, startIdx int, execCtx *ExecutionContext) executeResult {
	// Mutable current context: each step may replace it.
	// 可变当前上下文：每一步都可能替换它。
	current := execCtx
	loop := &run.Loop{
		Next: func(history []run.UnitOutput) (run.Unit, string, error) {
			idx := startIdx + len(history)
			if idx >= len(steps) {
				return nil, "", nil
			}
			step := steps[idx]
			unit := run.NewUnitFunc(step.GetID(), func(ctx context.Context, _ string) (string, run.Events, error) {
				result, err := step.Execute(ctx, current)
				if err != nil {
					return "", nil, err
				}
				current = result
				return result.Output, nil, nil
			})
			return unit, "", nil
		},
	}

	history, err := loop.Run(ctx)
	events := collectStepEvents(current, history)
	if err != nil {
		return executeResult{
			execCtx:    current,
			lastStepID: lastStepIDOf(history),
			events:     events,
			err: types.NewError(types.ErrCodeUnknown,
				fmt.Sprintf("step %s failed", lastStepIDOf(history)), err),
		}
	}

	return executeResult{
		execCtx:    current,
		lastStepID: lastStepIDOf(history),
		events:     events,
	}
}

// lastStepIDOf returns the ID of the last executed step, or "" if none.
// lastStepIDOf 返回最后执行步骤的 ID；未执行时为 ""。
func lastStepIDOf(history []run.UnitOutput) string {
	if len(history) == 0 {
		return ""
	}
	return history[len(history)-1].UnitID
}

// collectStepEvents extracts per-step events from the execution context.
// collectStepEvents 从执行上下文提取每个步骤的事件。
func collectStepEvents(execCtx *ExecutionContext, history []run.UnitOutput) map[string]run.Events {
	events := make(map[string]run.Events)
	for _, h := range history {
		if stepEvents := extractStepEvents(execCtx, h.UnitID); len(stepEvents) > 0 {
			events[h.UnitID] = stepEvents
		}
	}
	return events
}
