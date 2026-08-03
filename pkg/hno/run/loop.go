package run

import (
	"context"
)

// Unit is a single executable element in an orchestration loop: an agent
// invocation in a team, or a step in a workflow. Implementations adapt
// their concrete execution to this interface so Team and Workflow share
// the same loop kernel.
//
// Unit 是编排循环中的单个可执行元素：团队中的一次 agent 调用，
// 或工作流中的一个步骤。实现方将具体执行适配到该接口，使
// Team 与 Workflow 共享同一循环内核。
type Unit interface {
	// ID returns the unit identifier.
	// ID 返回单元标识符。
	ID() string

	// Execute runs the unit with the given input and returns the output
	// and any events produced.
	// Execute 用给定输入运行单元，返回输出与产生的事件。
	Execute(ctx context.Context, input string) (output string, events Events, err error)
}

// UnitOutput is the outcome of one executed unit.
// UnitOutput 是单个单元执行的结果。
type UnitOutput struct {
	UnitID string
	Output string
	Events Events
}

// UnitFunc adapts a function to the Unit interface.
// UnitFunc 将函数适配为 Unit 接口。
type UnitFunc struct {
	id string
	fn func(ctx context.Context, input string) (string, Events, error)
}

// NewUnitFunc wraps a function as a Unit.
// NewUnitFunc 将函数包装为 Unit。
func NewUnitFunc(id string, fn func(ctx context.Context, input string) (string, Events, error)) *UnitFunc {
	return &UnitFunc{id: id, fn: fn}
}

// ID implements Unit.
// ID 实现 Unit。
func (u *UnitFunc) ID() string { return u.id }

// Execute implements Unit.
// Execute 实现 Unit。
func (u *UnitFunc) Execute(ctx context.Context, input string) (string, Events, error) {
	return u.fn(ctx, input)
}

// Loop is the shared orchestration kernel. It repeatedly asks Next for the
// next unit and its input, executes it, and records the output until Next
// signals completion (nil unit) or the context is cancelled.
//
// Loop 是共享编排内核。它反复向 Next 请求下一个单元及其输入，
// 执行并记录输出，直到 Next 返回 nil（完成）或 context 取消。
type Loop struct {
	// Next returns the next unit and its input, or (nil, "", nil) when the
	// loop is complete. Implementations own the orchestration strategy:
	// sequential ordering, scheduler selection, convergence checks.
	// Next 返回下一个单元及其输入；循环完成时返回 (nil, "", nil)。
	// 实现方拥有编排策略：顺序、调度器选择、收敛检查。
	Next func(history []UnitOutput) (Unit, string, error)

	// OnStep is invoked after each unit executes (optional).
	// OnStep 在每次单元执行后调用（可选）。
	OnStep func(output UnitOutput)
}

// Run drives the loop until completion, cancellation, or error. It returns
// the ordered outputs of every executed unit.
//
// Run 驱动循环直至完成、取消或出错。返回所有已执行单元的有序输出。
func (l *Loop) Run(ctx context.Context) ([]UnitOutput, error) {
	history := make([]UnitOutput, 0)

	for {
		// Non-blocking cancellation check: unlike select with default, this
		// is deterministic when the context is already cancelled.
		// 非阻塞取消检查：相比带 default 的 select，context 已取消时
		// 该检查是确定性的。
		if err := ctx.Err(); err != nil {
			return history, err
		}

		unit, input, err := l.Next(history)
		if err != nil {
			return history, err
		}
		if unit == nil {
			return history, nil
		}

		output, events, err := unit.Execute(ctx, input)
		if err != nil {
			return history, err
		}

		out := UnitOutput{UnitID: unit.ID(), Output: output, Events: events}
		history = append(history, out)
		if l.OnStep != nil {
			l.OnStep(out)
		}
	}
}

// RunLoop is a convenience wrapper: it builds a Loop from a Next function
// and runs it.
// RunLoop 便捷包装：用 Next 函数构建 Loop 并运行。
func RunLoop(ctx context.Context, next func(history []UnitOutput) (Unit, string, error)) ([]UnitOutput, error) {
	return (&Loop{Next: next}).Run(ctx)
}
