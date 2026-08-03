package run

import (
	"context"
	"fmt"
	"testing"
)

// This test proves the extension story: a brand-new orchestrator type
// (a simple supervisor that delegates to workers until one succeeds) can be
// built on the shared loop kernel in a few dozen lines — no new loop code.
//
// 本测试验证扩展性：全新的编排器类型（简单的 supervisor，委派给
// workers 直到一个成功）只需几十行即可构建在共享循环内核之上——
// 无需任何新的循环代码。

// supervisor is a minimal orchestrator built entirely on run.Loop.
// supervisor 是完全构建在 run.Loop 之上的最小编排器。
type supervisor struct {
	workers []Unit // workers in priority order / 按优先级排列的 workers
}

// newSupervisor creates a supervisor over the given workers.
// newSupervisor 创建覆盖给定 workers 的 supervisor。
func newSupervisor(workers ...Unit) *supervisor {
	return &supervisor{workers: workers}
}

// run delegates to each worker in order until one returns non-empty output.
// run 依次委派给每个 worker，直到某个返回非空输出。
func (s *supervisor) run(ctx context.Context, input string) (string, error) {
	loop := &Loop{
		Next: func(history []UnitOutput) (Unit, string, error) {
			if len(history) >= len(s.workers) {
				return nil, "", nil
			}
			return s.workers[len(history)], input, nil
		},
	}
	history, err := loop.Run(ctx)
	if err != nil {
		return "", err
	}
	if len(history) == 0 {
		return "", nil
	}
	// First worker that produced output wins.
	// 第一个产生输出的 worker 胜出。
	for _, h := range history {
		if h.Output != "" {
			return h.Output, nil
		}
	}
	return history[len(history)-1].Output, nil
}

func TestSupervisor_BuiltOnSharedKernel(t *testing.T) {
	// Two failing workers then one success: supervisor skips failures.
	// 两个失败的 worker 然后一个成功：supervisor 跳过失败。
	ctx := context.Background()

	// workers[0] and workers[1] "fail" by returning empty output (a unit
	// that errors would abort the loop — supervisors handle that via
	// error semantics; here we model failure as empty output).
	// workers[0] 和 workers[1] 通过返回空输出"失败"（返回错误的单元会
	// 中止循环——supervisor 通过错误语义处理；这里用空输出模拟失败）。
	fallback := func(id string) Unit {
		return NewUnitFunc(id, func(_ context.Context, _ string) (string, Events, error) {
			return "", nil, nil
		})
	}
	success := NewUnitFunc("success", func(_ context.Context, _ string) (string, Events, error) {
		return "answer", nil, nil
	})

	s := newSupervisor(fallback("w1"), fallback("w2"), success)
	out, err := s.run(ctx, "q")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out != "answer" {
		t.Errorf("output = %q, want answer (first success wins)", out)
	}
}

func TestSupervisor_Ordering(t *testing.T) {
	// The FIRST worker that returns output wins, in priority order.
	// 按优先级顺序，第一个返回输出的 worker 胜出。
	a := NewUnitFunc("a", func(_ context.Context, _ string) (string, Events, error) {
		return "from-a", nil, nil
	})
	b := NewUnitFunc("b", func(_ context.Context, _ string) (string, Events, error) {
		return "from-b", nil, nil
	})

	s := newSupervisor(a, b)
	out, err := s.run(context.Background(), "q")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out != "from-a" {
		t.Errorf("output = %q, want from-a (priority order)", out)
	}
}

// A unit that errors must abort the loop and propagate.
// 返回错误的单元必须中止循环并传播错误。
func TestSupervisor_ErrorPropagates(t *testing.T) {
	bad := NewUnitFunc("bad", func(_ context.Context, _ string) (string, Events, error) {
		return "", nil, fmt.Errorf("worker failed")
	})
	s := newSupervisor(bad)
	if _, err := s.run(context.Background(), "q"); err == nil {
		t.Fatal("expected error propagation")
	}
}
