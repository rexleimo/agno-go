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

// executeSteps compiles steps[startIdx:] into a declaration graph and drives it
// on the shared graph engine kernel (G10 OPT-β): no persistence, no run-context
// assembly — those live in Workflow.Run. The public behavior of the old
// run.Loop-based linear kernel is preserved byte for byte; see graph_compiler.go.
//
// Cancellation keeps the old kernel's boundary semantics: the scheduler runs on a
// context derived from the caller's with cancellation isolated (values kept), the
// ledger hook stops the scheduler only after a completion has been recorded, and
// node adapters receive the caller's ctx so an in-flight step can still abort on
// cancellation exactly like it did under run.Loop.
//
// executeSteps 把 steps[startIdx:] 编译为声明图并跑在共享图引擎内核上
// （G10 OPT-β）：不做持久化、不装配运行上下文——这些都在 Workflow.Run 中。
// 旧 run.Loop 线性内核的公共行为逐字节保留，见 graph_compiler.go。
//
// 取消保留旧内核的边界语义：调度器跑在从调用方 ctx 派生、取消隔离的上下文上
// （值保留），台账钩子只在一次完成记账之后才停调度器；叶适配器拿到调用方
// ctx，在飞步照旧能感知取消并自行中止。
func executeSteps(ctx context.Context, steps []Node, startIdx int, execCtx *ExecutionContext) executeResult {
	// 旧内核在每次 Next 之前判取消：入口处的取消一个步都不跑。
	// 复刻这个次序，先于任何编译与派发。
	if err := ctx.Err(); err != nil {
		return executeResult{
			execCtx: execCtx,
			events:  map[string]run.Events{},
			err:     types.NewError(types.ErrCodeUnknown, fmt.Sprintf("step %s failed", ""), err),
		}
	}
	tail := steps[startIdx:]
	if len(tail) == 0 {
		return executeResult{execCtx: execCtx, events: map[string]run.Events{}}
	}
	cp, err := compileSteps(tail, execCtx, ctx)
	if err != nil {
		return executeResult{
			execCtx: execCtx,
			events:  map[string]run.Events{},
			err:     fmt.Errorf("workflow: compile steps: %w", err),
		}
	}
	schedCtx, stopSched := context.WithCancel(context.WithoutCancel(ctx))
	defer stopSched()
	cp.ledger.watch = ctx
	cp.ledger.stopSched = stopSched

	res, runErr := cp.g.Run(schedCtx, &carrier{ec: execCtx})
	executed, lastEC, _ := cp.ledger.snapshot()
	lastStepID := lastOf(executed)

	if runErr != nil {
		if lastEC == nil {
			lastEC = execCtx
		}
		return executeResult{
			execCtx:    lastEC,
			lastStepID: lastStepID,
			events:     cp.ledger.collectEvents(nil, true),
			err:        cp.wrapError(runErr, lastStepID),
		}
	}

	final := lastEC
	if car, ok := res.Output().(*carrier); ok && car.ec != nil {
		final = car.ec
	}
	if final == nil {
		final = execCtx
	}
	return executeResult{
		execCtx:    final,
		lastStepID: lastStepID,
		events:     cp.ledger.collectEvents(final, false),
	}
}

// lastOf returns the last executed step ID, or "" if none.
// lastOf 返回最后执行步骤的 ID；未执行时为 ""。
func lastOf(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return ids[len(ids)-1]
}
