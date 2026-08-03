package run

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// testUnit is a scripted unit for loop tests.
// testUnit 是循环测试用的脚本化单元。
type testUnit struct {
	id     string
	output string
	err    error
}

func (u *testUnit) ID() string { return u.id }

func (u *testUnit) Execute(_ context.Context, input string) (string, Events, error) {
	if u.err != nil {
		return "", nil, u.err
	}
	return u.output + ":" + input, nil, nil
}

func TestLoop_Sequential(t *testing.T) {
	units := []Unit{
		&testUnit{id: "a", output: "A"},
		&testUnit{id: "b", output: "B"},
		&testUnit{id: "c", output: "C"},
	}

	var order []string
	history, err := RunLoop(context.Background(), func(hist []UnitOutput) (Unit, string, error) {
		if len(hist) >= len(units) {
			return nil, "", nil
		}
		u := units[len(hist)]
		order = append(order, u.ID())
		return u, fmt.Sprintf("in%d", len(hist)), nil
	})
	if err != nil {
		t.Fatalf("RunLoop: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("history = %d outputs, want 3", len(history))
	}
	// Each unit receives the previous output chained? No: input is scripted.
	// 每个单元收到脚本化输入。
	if history[0].Output != "A:in0" || history[2].Output != "C:in2" {
		t.Errorf("outputs = %+v", history)
	}
	if len(order) != 3 || order[0] != "a" || order[2] != "c" {
		t.Errorf("execution order = %v", order)
	}
}

func TestLoop_OnStep(t *testing.T) {
	units := []Unit{&testUnit{id: "x", output: "1"}, &testUnit{id: "y", output: "2"}}
	var seen []string
	l := &Loop{
		Next: func(hist []UnitOutput) (Unit, string, error) {
			if len(hist) >= len(units) {
				return nil, "", nil
			}
			return units[len(hist)], "", nil
		},
		OnStep: func(out UnitOutput) {
			seen = append(seen, out.UnitID)
		},
	}
	if _, err := l.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(seen) != 2 || seen[0] != "x" || seen[1] != "y" {
		t.Errorf("OnStep order = %v", seen)
	}
}

func TestLoop_ErrorPropagates(t *testing.T) {
	boom := errors.New("boom")
	units := []Unit{
		&testUnit{id: "ok", output: "1"},
		&testUnit{id: "bad", err: boom},
	}
	_, err := RunLoop(context.Background(), func(hist []UnitOutput) (Unit, string, error) {
		if len(hist) >= len(units) {
			return nil, "", nil
		}
		return units[len(hist)], "", nil
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}

func TestLoop_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel before the loop starts: the first select must observe it.
	// 循环开始前取消：第一个 select 必须观察到。
	cancel()
	_, err := RunLoop(ctx, func(hist []UnitOutput) (Unit, string, error) {
		return &testUnit{id: "a", output: "1"}, "", nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestLoop_CancelMidLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	units := []Unit{&testUnit{id: "a", output: "1"}, &testUnit{id: "b", output: "2"}}
	_, err := RunLoop(ctx, func(hist []UnitOutput) (Unit, string, error) {
		if len(hist) == 1 {
			// Cancel mid-loop; keep returning units so the loop keeps
			// iterating and the cancellation check fires on the next round.
			// 循环中途取消；继续返回单元让循环继续迭代，
			// 下一轮取消检查触发。
			cancel()
		}
		if len(hist) >= len(units) {
			return nil, "", nil
		}
		return units[len(hist)], "", nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestLoop_NextError(t *testing.T) {
	nextErr := errors.New("scheduler failure")
	_, err := RunLoop(context.Background(), func(hist []UnitOutput) (Unit, string, error) {
		return nil, "", nextErr
	})
	if !errors.Is(err, nextErr) {
		t.Fatalf("err = %v, want scheduler failure", err)
	}
}

func TestUnitFunc_Adapter(t *testing.T) {
	u := NewUnitFunc("fn", func(_ context.Context, input string) (string, Events, error) {
		return "got " + input, nil, nil
	})
	if u.ID() != "fn" {
		t.Errorf("ID = %q", u.ID())
	}
	out, _, err := u.Execute(context.Background(), "hi")
	if err != nil || out != "got hi" {
		t.Errorf("Execute = %q, %v", out, err)
	}
}
