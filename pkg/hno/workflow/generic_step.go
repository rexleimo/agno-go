package workflow

import (
	"context"
	"fmt"
)

// StepFunc is the body of a generic step: it transforms an input of type In
// into an output of type Out, with access to the execution context for
// intermediate data.
//
// StepFunc 是泛型步骤的函数体：将 In 类型输入转换为 Out 类型输出，
// 并可访问执行上下文中的中间数据。
type StepFunc[In, Out any] func(ctx context.Context, execCtx *ExecutionContext, input In) (Out, error)

// GenericStep is a type-safe workflow step (aligned with Genkit Flow[In,Out]).
// It wraps a StepFunc and adapts it to the Node interface so generic steps
// compose with existing conditional/loop/parallel constructs.
//
// GenericStep 是类型安全的工作流步骤（对齐 Genkit Flow[In,Out]）。
// 它包装 StepFunc 并适配为 Node 接口，使泛型步骤能与现有
// 条件/循环/并行构件组合。
type GenericStep[In, Out any] struct {
	id     string
	fn     StepFunc[In, Out]
	encode func(Out) string
	decode func(string) (In, error)
}

// NewGenericStep creates a typed step. encode converts the output to a
// string stored in the execution context; decode converts the stored string
// back to In for the next step.
//
// NewGenericStep 创建类型化步骤。encode 将输出转换为存入执行上下文
// 的字符串；decode 将存储的字符串还原为下一步的 In。
func NewGenericStep[In, Out any](
	id string,
	fn StepFunc[In, Out],
	encode func(Out) string,
	decode func(string) (In, error),
) (*GenericStep[In, Out], error) {
	if id == "" {
		return nil, fmt.Errorf("step id is required")
	}
	if fn == nil {
		return nil, fmt.Errorf("step function is required")
	}
	if encode == nil {
		return nil, fmt.Errorf("encode is required")
	}
	if decode == nil {
		return nil, fmt.Errorf("decode is required")
	}
	return &GenericStep[In, Out]{id: id, fn: fn, encode: encode, decode: decode}, nil
}

// GetID returns the step identifier.
// GetID 返回步骤标识符。
func (s *GenericStep[In, Out]) GetID() string { return s.id }

// GetName returns the step name (same as ID by default).
// GetName 返回步骤名称（默认与 ID 相同）。
func (s *GenericStep[In, Out]) GetName() string { return s.id }

// GetType returns the step type.
// GetType 返回步骤类型。
func (s *GenericStep[In, Out]) GetType() string { return "generic" }

// Execute implements Node: it decodes the input from the execution context,
// runs the typed function, and stores the encoded output. The first step
// reads from execCtx.Input; subsequent steps chain via execCtx.Output.
//
// Execute 实现 Node：从执行上下文解码输入，运行类型化函数，
// 并存储编码后的输出。首步读取 execCtx.Input；后续步骤通过
// execCtx.Output 串联。
func (s *GenericStep[In, Out]) Execute(ctx context.Context, execCtx *ExecutionContext) (*ExecutionContext, error) {
	// Chain from the previous output; fall back to the initial input for the
	// first step.
	// 从上一个输出串联；首步回退到初始输入。
	raw := execCtx.Output
	if raw == "" {
		raw = execCtx.Input
	}

	var input In
	if raw != "" {
		decoded, err := s.decode(raw)
		if err != nil {
			return nil, fmt.Errorf("step %s: decode input: %w", s.id, err)
		}
		input = decoded
	}

	out, err := s.fn(ctx, execCtx, input)
	if err != nil {
		return nil, fmt.Errorf("step %s failed: %w", s.id, err)
	}

	execCtx.Output = s.encode(out)
	execCtx.Set(fmt.Sprintf("step_%s_output", s.id), execCtx.Output)
	return execCtx, nil
}
