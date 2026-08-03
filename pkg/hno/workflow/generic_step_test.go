package workflow

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
)

func TestGenericStep_Transform(t *testing.T) {
	// A typed pipeline: int input -> string output.
	// 类型化流水线：int 输入 -> string 输出。
	step, err := NewGenericStep(
		"double",
		func(_ context.Context, ec *ExecutionContext, input int) (string, error) {
			return strconv.Itoa(input * 2), nil
		},
		func(out string) string { return out },
		func(s string) (int, error) { return strconv.Atoi(s) },
	)
	if err != nil {
		t.Fatalf("NewGenericStep: %v", err)
	}

	execCtx := NewExecutionContext("4")
	result, err := step.Execute(context.Background(), execCtx)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Output != "8" {
		t.Errorf("Output = %q, want 8", result.Output)
	}
	got, ok := result.Get("step_double_output")
	if !ok || got != "8" {
		t.Errorf("stored output = %v (ok=%v), want 8", got, ok)
	}
}

func TestGenericStep_JSONCodec(t *testing.T) {
	// JSON-based codec for structured data.
	// 结构化数据的 JSON 编解码。
	type Note struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	step, err := NewGenericStep(
		"summarize",
		func(_ context.Context, _ *ExecutionContext, in Note) (Note, error) {
			return Note{Title: "sum: " + in.Title, Body: in.Body}, nil
		},
		func(out Note) string {
			b, _ := json.Marshal(out)
			return string(b)
		},
		func(s string) (Note, error) {
			var n Note
			err := json.Unmarshal([]byte(s), &n)
			return n, err
		},
	)
	if err != nil {
		t.Fatalf("NewGenericStep: %v", err)
	}

	execCtx := NewExecutionContext(`{"title":"hello","body":"world"}`)
	result, err := step.Execute(context.Background(), execCtx)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Output != `{"title":"sum: hello","body":"world"}` {
		t.Errorf("Output = %q", result.Output)
	}
}

func TestGenericStep_Validation(t *testing.T) {
	if _, err := NewGenericStep[int, int]("", nil, nil, nil); err == nil {
		t.Error("expected error for empty id")
	}
	if _, err := NewGenericStep[int, int]("s", nil, func(i int) string { return "" }, func(s string) (int, error) { return 0, nil }); err == nil {
		t.Error("expected error for nil fn")
	}
	if _, err := NewGenericStep[int, int]("s", func(context.Context, *ExecutionContext, int) (int, error) { return 0, nil }, nil, nil); err == nil {
		t.Error("expected error for nil encode")
	}
}

func TestGenericStep_Chained(t *testing.T) {
	// Two typed steps chained through the execution context.
	// 通过执行上下文串联两个类型化步骤。
	toUpper, err := NewGenericStep(
		"upper",
		func(_ context.Context, _ *ExecutionContext, in string) (string, error) {
			return in + "!", nil
		},
		func(out string) string { return out },
		func(s string) (string, error) { return s, nil },
	)
	if err != nil {
		t.Fatalf("step1: %v", err)
	}
	count, err := NewGenericStep(
		"count",
		func(_ context.Context, _ *ExecutionContext, in string) (int, error) {
			return len(in), nil
		},
		func(out int) string { return strconv.Itoa(out) },
		func(s string) (string, error) { return s, nil },
	)
	if err != nil {
		t.Fatalf("step2: %v", err)
	}

	execCtx := NewExecutionContext("hi")
	execCtx, err = toUpper.Execute(context.Background(), execCtx)
	if err != nil {
		t.Fatalf("step1 execute: %v", err)
	}
	execCtx, err = count.Execute(context.Background(), execCtx)
	if err != nil {
		t.Fatalf("step2 execute: %v", err)
	}
	if execCtx.Output != "3" {
		t.Errorf("chained Output = %q, want 3 (\"hi!\" length)", execCtx.Output)
	}
}
