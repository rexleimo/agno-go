package graph_test

// 本文件是 P1 票面 B4（stepLimit 安全阀）这一片的行为族，与 graph_test.go 同包、
// 复用其夹具 looping 与 cycleLimit。票面 §5 允许「新增 pkg/hno/graph 下的同包测试
// 文件」，单独成文件的唯一目的是让这一片的行为族与文件边界对齐，便于按切片审差分。
//
// 观察面守票面 §1：步数只能由调用方自己提供的节点体计数、或 Run 的返回值来判定；
// scheduler 内部字段、queue 长度、goroutine 数、内部计时都不是本族的证据。

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

// alwaysTrueCycle 构造 §3.5 第 7 项刻意放行的那张图：a→b 无条件、b→a 条件边但谓词
// 恒真。它不是「不可断开的环」（那种构建期就被拒），母约 §3.3:192 明写兜住它的责任
// 在步数上限，所以本族的每条红都必须先证明 Validate 放行了这张图。
//
// counters 按节点名报告「我被启动了一次」，这是调用方手里的公共观测面：上限若发生在
// 完成而不是派发之前，多启动的那一次在这里暴露。
func alwaysTrueCycle(counters map[string]*atomic.Int64, opts ...graph.Option) *graph.Graph {
	node := func(name string) graph.Node {
		c := counters[name]
		return graph.NodeFunc(name, looping(func() { c.Add(1) }))
	}
	return graph.New(opts...).
		AddNode(node("a")).
		AddNode(node("b")).
		AddEdge("a", "b").
		AddConditional("b", "a", func(any) bool { return true }).
		SetEntry("a").SetOutput("b")
}

// returnGrace 是给 runBounded 的后备期限：deadline 到点却被取消的 Run 必须自己回来，
// 再多等一个 grace 仍不回来才判基础设施失败，而不是让整条聚焦命令挂到 -timeout。
const returnGrace = time.Second

// runBounded 是本行为族自己的执行夹具。它与 graph_test.go 的 runWithinLimit 承担相反
// 的判据，所以不能复用：那条夹具把「不自退出」本身当作拒绝信号（不可断开的环应当在
// 构建期就被拒），而本族的图是 §3.5 第 7 项刻意放行的恒真环——它必须由 Run 自己带着
// ErrStepLimitExceeded 回来。用 runWithinLimit 会让本次缺失的行为表现成一句「构建期
// 该拒」的错判文案，把读者指向与票面 B4 相反的修法。
//
// 期限仍然只是后备：ctx 在 cycleLimit 到点时超时，于是一个不自收敛的图会把
// context.DeadlineExceeded 交给断言，失败输出直接落在「错误身份不对」这条真实缺口上。
func runBounded(t *testing.T, g *graph.Graph) (*graph.Result, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), cycleLimit)
	defer cancel()

	type outcome struct {
		res *graph.Result
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		res, err := g.Run(ctx, "seed")
		done <- outcome{res, err}
	}()

	select {
	case o := <-done:
		return o.res, o.err
	case <-time.After(cycleLimit + returnGrace):
		t.Fatalf("Run 在 %v 内没有交出任何结果：本族要求超限的图自己带着 ErrStepLimitExceeded 回来", cycleLimit+returnGrace)
		return nil, nil
	}
}

func startCounters() map[string]*atomic.Int64 {
	return map[string]*atomic.Int64{"a": {}, "b": {}}
}

// TestP1R13_StepLimitStopsBreakableCycle 覆盖契约 T1：显式 WithStepLimit(n) 时，超过
// n 步才能结束的图必须以 ErrStepLimitExceeded 结束、不交出 Result，且被启动的激活数
// 恰好是 n（上限既不提前也不放宽）。
func TestP1R13_StepLimitStopsBreakableCycle(t *testing.T) {
	counters := startCounters()
	const limit = 3
	g := alwaysTrueCycle(counters, graph.WithStepLimit(limit))

	if err := g.Validate(); err != nil {
		t.Fatalf("可断开的条件环被构建期拒绝，步数上限这条路根本没机会跑到: %v", err)
	}

	res, err := runBounded(t, g)
	if !errors.Is(err, graph.ErrStepLimitExceeded) {
		t.Fatalf("Run 的错误 = %v, want errors.Is(err, graph.ErrStepLimitExceeded)", err)
	}
	if res != nil {
		t.Errorf("超限后仍返回 Result %v，调用方还能读到 Output()，与票面 B4「Output() 不可用」矛盾", res)
	}
	if got := counters["a"].Load() + counters["b"].Load(); got != limit {
		t.Errorf("被启动的激活数 = %d, want %d（上限必须在恰好 n 次启动处收口）", got, limit)
	}
}

// TestP1R13_LimitRefusesActivationsAtDispatch 覆盖契约 T1 的另一半：上限判定发生在
// 「把一次激活交给节点之前」，而不是一批并发激活全部跑完之后。
//
// 图形是 entry→{x,y} 的合法扇出，limit=2，且第一个后继 x 会一直阻塞到测试结束。
// 只在派发前计步的实现会在 x 之后拒掉 y，于是 Run 能在 x 仍卡着的时候带着哨兵回来；
// 把整批并发激活先放出去、再按「已完成数」判超限的实现要么让 x 和 y 一起跑起来，
// 要么根本回不来。这条夹具刻意不读 x 的启动计数：x 被派发不等于它的节点体在 Run
// 返回前已经跑上，那一读会把断言变成和调度时序赛跑（T1/T2/T4 的精确总数没有这个
// 问题，那里的每一步都是上一步完成之后才派出去的）。
func TestP1R13_LimitRefusesActivationsAtDispatch(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	var y atomic.Int64
	g := graph.New(graph.WithStepLimit(2)).
		AddNode(graph.NodeFunc("entry", looping(noCount))).
		AddNode(graph.NodeFunc("x", func(ctx context.Context, in any) (any, error) {
			select {
			case <-release:
			case <-ctx.Done():
			}
			return in, nil
		})).
		AddNode(graph.NodeFunc("y", looping(func() { y.Add(1) }))).
		AddEdge("entry", "x").
		AddEdge("entry", "y").
		SetEntry("entry").SetOutput("y")

	if err := g.Validate(); err != nil {
		t.Fatalf("合法的扇出图被构建期拒绝: %v", err)
	}

	res, err := runBounded(t, g)
	if !errors.Is(err, graph.ErrStepLimitExceeded) {
		t.Fatalf("Run 的错误 = %v, want errors.Is(err, graph.ErrStepLimitExceeded)：一个后继仍卡在节点里时，超限必须已经判定完成", err)
	}
	if res != nil {
		t.Errorf("超限后仍返回 Result %v", res)
	}
	if got := y.Load(); got != 0 {
		t.Errorf("第二个兄弟节点 y 被启动了 %d 次：计步发生在完成而不是派发之前时，一整批并发激活会被整体放行", got)
	}
}

// TestP1R13_ExactlyBudgetedGraphStillSucceeds 覆盖契约 T1 的「不提前」半边：一张正好
// 用满 n 步就收敛的图必须成功交出 Result，而不是被安全阀判成超限。
//
// 这条是 REFACTOR 的变异矩阵补出来的：把「先判预算再派发」改成「先派发再判预算」的
// 变异体在其它行上都能给出同样的启动次数，只有这张图能看出它把最后一步跑完的图也说成了
// 失败 —— 对调用方而言那是「结果明明拿到了，却被告诉图跑飞了」。
func TestP1R13_ExactlyBudgetedGraphStillSucceeds(t *testing.T) {
	var a, b, c atomic.Int64
	g := graph.New(graph.WithStepLimit(3)).
		AddNode(graph.NodeFunc("a", looping(func() { a.Add(1) }))).
		AddNode(graph.NodeFunc("b", looping(func() { b.Add(1) }))).
		AddNode(graph.NodeFunc("c", looping(func() { c.Add(1) }))).
		AddEdge("a", "b").
		AddEdge("b", "c").
		SetEntry("a").SetOutput("c")

	if err := g.Validate(); err != nil {
		t.Fatalf("合法的三步链被构建期拒绝: %v", err)
	}
	res, err := runBounded(t, g)
	if err != nil {
		t.Fatalf("恰好用满 3 步预算的图被判失败: %v", err)
	}
	if got, want := res.Completed(), []string{"a", "b", "c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Completed() = %v, want %v", got, want)
	}
	if got, want := res.Output(), "seed"; got != want {
		t.Errorf("Result.Output() = %v, want %v", got, want)
	}
	if got := a.Load() + b.Load() + c.Load(); got != 3 {
		t.Errorf("三个节点各执行一次的期望被破坏，总执行数 = %d", got)
	}
}

// TestP1R13_StepLimitDefaultsWhenUnset 覆盖契约 T2：不带任何选项时默认 1000 生效
// （母约 §7:354）。这条今天完全不存在——WithStepLimit 的导出文档「未设置时默认 1000」
// 是一句空承诺。
func TestP1R13_StepLimitDefaultsWhenUnset(t *testing.T) {
	counters := startCounters()
	g := alwaysTrueCycle(counters)

	if err := g.Validate(); err != nil {
		t.Fatalf("可断开的条件环被构建期拒绝: %v", err)
	}

	res, err := runBounded(t, g)
	if !errors.Is(err, graph.ErrStepLimitExceeded) {
		t.Fatalf("未设置选项的图没有落在默认 1000 上: Run 的错误 = %v, want ErrStepLimitExceeded", err)
	}
	if res != nil {
		t.Errorf("默认超限后仍返回 Result %v", res)
	}
	if got := counters["a"].Load() + counters["b"].Load(); got != 1000 {
		t.Errorf("被启动的激活数 = %d, want 1000（导出的默认承诺）", got)
	}
}

// TestP1R13_EachRunGetsItsOwnBudget 覆盖契约 T4：预算属于单次 Run。同一张超限图连
// 续执行两次，两次都必须各自跑到上限并给出同一条哨兵错误——把计数器挂在 *Graph 上
// 的实现会让第二次立即失败（调用方看到的是「这张图坏了」而不是「这次跑超了」）。
func TestP1R13_EachRunGetsItsOwnBudget(t *testing.T) {
	counters := startCounters()
	const limit = 3
	g := alwaysTrueCycle(counters, graph.WithStepLimit(limit))

	if err := g.Validate(); err != nil {
		t.Fatalf("可断开的条件环被构建期拒绝: %v", err)
	}

	for run := 1; run <= 2; run++ {
		res, err := runBounded(t, g)
		if !errors.Is(err, graph.ErrStepLimitExceeded) {
			t.Fatalf("第 %d 次 Run 的错误 = %v, want ErrStepLimitExceeded", run, err)
		}
		if res != nil {
			t.Errorf("第 %d 次超限仍返回 Result %v", run, res)
		}
		if got := counters["a"].Load() + counters["b"].Load(); got != int64(run)*limit {
			t.Errorf("第 %d 次 Run 之后累计启动 %d 次, want %d（每次 Run 各有 %d 步预算）", run, got, run*limit, limit)
		}
	}
}

// TestP1R13_CancellationIsNotReportedAsStepLimit 是契约 T5 的对照行：取消归一化（票面
// B9）不得被新哨兵顶替。节点阻塞期间取消 ctx，Run 交出的必须是 context.Canceled，
// 且明确不是 ErrStepLimitExceeded——否则调用方会把「我取消了」误读成「图跑飞了」。
func TestP1R13_CancellationIsNotReportedAsStepLimit(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	g := graph.New(graph.WithStepLimit(1000)).
		AddNode(graph.NodeFunc("block", func(ctx context.Context, in any) (any, error) {
			select {
			case <-release:
				return in, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		})).
		SetEntry("block").SetOutput("block")

	ctx, cancel := context.WithCancel(context.Background())
	type outcome struct {
		res *graph.Result
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		res, err := g.Run(ctx, "seed")
		done <- outcome{res, err}
	}()

	cancel()
	select {
	case o := <-done:
		if !errors.Is(o.err, context.Canceled) {
			t.Errorf("取消后 Run 的错误 = %v, want errors.Is(err, context.Canceled)", o.err)
		}
		if errors.Is(o.err, graph.ErrStepLimitExceeded) {
			t.Errorf("取消被说成步数超限: %v", o.err)
		}
		if o.res != nil {
			t.Errorf("取消后仍返回 Result %v", o.res)
		}
	case <-time.After(cycleLimit):
		t.Fatal("取消后 Run 未在 cycleLimit 内返回")
	}
}

// TestP1R13_ConvergingCycleStillSucceeds 是契约 T3 的对照行：给了足够预算的收敛环必须
// 照旧成功——安全阀只在超限介入，不得把「有环」本身变成错误。
func TestP1R13_ConvergingCycleStillSucceeds(t *testing.T) {
	trials := 0
	g := graph.New(graph.WithStepLimit(10)).
		AddNode(graph.NodeFunc("a", looping(noCount))).
		AddNode(graph.NodeFunc("b", looping(noCount))).
		AddNode(graph.NodeFunc("end", looping(noCount))).
		AddEdge("a", "b").
		AddConditional("b", "a", func(any) bool { trials++; return trials < 2 }).
		AddConditional("b", "end", func(any) bool { return trials >= 2 }).
		SetEntry("a").SetOutput("end")

	if err := g.Validate(); err != nil {
		t.Fatalf("可断开的条件环被构建期拒绝: %v", err)
	}
	res, err := runBounded(t, g)
	if err != nil {
		t.Fatalf("预算充足的收敛环被误伤: %v", err)
	}
	if got, want := res.Completed(), []string{"a", "a", "b", "b", "end"}; len(got) != len(want) {
		t.Errorf("Completed() = %v, want 与 %v 同长（5 次激活，远低于 10 步预算）", got, want)
	}
	if got, want := res.Output(), "seed"; got != want {
		t.Errorf("Result.Output() = %v, want %v", got, want)
	}
}

// TestP1R13_NonPositiveStepLimitRejectedAtBuildTime 覆盖契约 T6：WithStepLimit(n<=0)
// 不得成为「不限步数」的开关，也不得静默变成「一步都不许跑」。它归入既有声明类拒绝：
// Validate 非 nil、文案能定位到那条声明本身、Run 复用同一条错误、且一个节点都不执行。
//
// 规格权威缺口见契约 amendmentRegistration（S13-AMEND-1）：票面 B4 与母约 §3.3/§7 都
// 没规定 n<=0 的行为，这条是作者裁断，母约写回之前不算有规格背书。
func TestP1R13_NonPositiveStepLimitRejectedAtBuildTime(t *testing.T) {
	for _, n := range []int{0, -1} {
		var executions atomic.Int64
		g := graph.New(graph.WithStepLimit(n)).
			AddNode(graph.NodeFunc("a", looping(func() { executions.Add(1) }))).
			SetEntry("a").SetOutput("a")

		err := g.Validate()
		if err == nil {
			t.Fatalf("WithStepLimit(%d) 被 Validate 放行：0 既不是合法上限也不该被当成不限步数", n)
		}
		if !strings.Contains(err.Error(), "WithStepLimit") {
			t.Errorf("WithStepLimit(%d) 的拒绝文案 %q 未点名那条声明，调用方无从定位到自己写的那个旋钮", n, err)
		}
		if again := g.Validate(); again == nil || again.Error() != err.Error() {
			t.Errorf("WithStepLimit(%d) 重复校验给出的文案不一致: %v vs %v", n, again, err)
		}

		res, runErr := runBounded(t, g)
		if runErr == nil {
			t.Fatalf("WithStepLimit(%d) 的图仍可执行: Output=%v", n, res.Output())
		}
		if runErr.Error() != err.Error() {
			t.Errorf("Run 的拒绝与 Validate 不是同一条文案: %q vs %q", runErr.Error(), err.Error())
		}
		if got := executions.Load(); got != 0 {
			t.Errorf("非法上限的图执行了 %d 次节点，拒绝发生在上限生效之前", got)
		}
	}
}
