package graph_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

// 票面 B2（docs/design/v3-test-scope-p1-graph.md:61）：并行扇出 + Join 汇聚屏障。
// 契约 docs/design/v3-test-scope-p1-graph-slice19.json 的 D1–D7 逐行一测。
//
// 观察面只有导出 API：New/AddNode/NodeFunc/AddEdge/AddConditional/AddJoin/SetEntry/
// SetOutput/Validate/Run 与 Result.Output/Value/Completed，外加 errors.Is。
// scheduler 的 pending/running/queue、queue 长度、goroutine 数、内部计时都在票面 :40
// 的禁止清单上，本片一行未碰。节点之间的先后一律由测试自有的 channel 握手担保
// （票面 :75 禁止用 sleep 定序，:77 禁止把 Join 实现成「等固定时长再聚合」）。
// 下面的 time.After 只作停摆上界阀：它把「永远不返回」变成一条可判定的失败输出，
// 不参与任何定序，也不作为通过的依据。

const p1r19Valve = 10 * time.Second

// p1r19Outcome 是一次 Run 的交出物，只为把 Run 放进有界阀内观察而存在。
type p1r19Outcome struct {
	res *graph.Result
	err error
}

// p1r19RunBounded 在阀内执行一次 Run：阀到点即判红，绝不写成「没超时就算过」。
func p1r19RunBounded(t *testing.T, g *graph.Graph) p1r19Outcome {
	t.Helper()
	out := make(chan p1r19Outcome, 1)
	go func() {
		res, err := g.Run(context.Background(), "seed")
		out <- p1r19Outcome{res: res, err: err}
	}()
	select {
	case got := <-out:
		return got
	case <-time.After(p1r19Valve):
		t.Fatalf("Run 未在 %v 的有界阀内交出结论：汇聚屏障把执行停成了挂死", p1r19Valve)
		return p1r19Outcome{}
	}
}

// TestP1R19_JoinBarrierRunsExactlyOnce 观察契约 D1：entry→{a,b}→join 能跑，
// 且 join 在本次 Run 里恰好执行一次。
//
// 「恰好」不是「至少」：把 Join 当成无条件边路由的朴素实现会让 a 完成时激活一次、
// b 完成时再激活一次 —— 两次执行在测试自有的信号计数上可见。
func TestP1R19_JoinBarrierRunsExactlyOnce(t *testing.T) {
	identity := func(_ context.Context, in any) (any, error) { return in, nil }
	joinEntered := make(chan struct{}, 8)

	g := graph.New().
		AddNode(graph.NodeFunc("entry", identity)).
		AddNode(graph.NodeFunc("a", identity)).
		AddNode(graph.NodeFunc("b", identity)).
		AddNode(graph.NodeFunc("join", func(_ context.Context, in any) (any, error) {
			joinEntered <- struct{}{}
			return "joined", nil
		})).
		SetEntry("entry").
		SetOutput("join")
	g = g.AddEdge("entry", "a").AddEdge("entry", "b").AddJoin([]string{"a", "b"}, "join")

	if err := g.Validate(); err != nil {
		t.Fatalf("含 Join 汇聚屏障的合法图被 Validate 拒绝：%v（票面 :61 要求 entry→{a,b}→join 可执行）", err)
	}
	out := p1r19RunBounded(t, g)
	if out.err != nil {
		t.Fatalf("Run 未执行汇聚屏障图：%v", out.err)
	}
	if out.res == nil {
		t.Fatal("err==nil 却交出 nil Result")
	}

	// 有界非阻塞清点：Run 已返回，信号通道里剩下的就是本次执行的全部进入次数。
	// 这里不能用 `for range` —— joinEntered 从不关闭，range 会在取完那一个之后永久阻塞。
	entered := 0
drain:
	for {
		select {
		case <-joinEntered:
			entered++
		default:
			break drain
		}
	}
	if entered != 1 {
		t.Errorf("join 在本次 Run 中进入了 %d 次, want 恰好 1 次（屏障的判据是凑齐之前不激活、凑齐之后只激活一次）", entered)
	}
	if got := out.res.Output(); got != any("joined") {
		t.Errorf("Result.Output() = %v, want %v", got, "joined")
	}
	joined := 0
	for _, name := range out.res.Completed() {
		if name == "join" {
			joined++
		}
	}
	if joined != 1 {
		t.Errorf("Result.Completed() = %v，其中 join 出现 %d 次, want 1 次", out.res.Completed(), joined)
	}
}

// TestP1R19_JoinInputIsKeyedByPredecessorName 观察契约 D2：join 收到的输入含 a、b
// 两者的输出，并按前驱名键控。
//
// 两个前驱故意交出不同类型（string 与 int），这样「只交最后一个前驱」与「聚合了但键名
// 错位」都逃不掉：前者少一个键，后者取不到对应值。
func TestP1R19_JoinInputIsKeyedByPredecessorName(t *testing.T) {
	received := make(chan any, 1)

	g := graph.New().
		AddNode(graph.NodeFunc("entry", func(_ context.Context, in any) (any, error) { return in, nil })).
		AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "out-a", nil })).
		AddNode(graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) { return 42, nil })).
		AddNode(graph.NodeFunc("join", func(_ context.Context, in any) (any, error) {
			received <- in
			return "joined", nil
		})).
		SetEntry("entry").
		SetOutput("join")
	g = g.AddEdge("entry", "a").AddEdge("entry", "b").AddJoin([]string{"a", "b"}, "join")

	out := p1r19RunBounded(t, g)
	if out.err != nil {
		t.Fatalf("Run 未执行汇聚屏障图：%v", out.err)
	}
	close(received)
	var got any
	for v := range received {
		got = v
	}

	byName, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("join 收到的输入是 %#v，不是按前驱名键控的聚合，票面 :61 要求输入含 a、b 两者的输出", got)
	}
	if v, present := byName["a"]; !present || v != any("out-a") {
		t.Errorf("join 的输入里前驱 \"a\" 的值 = %#v (present=%v), want \"out-a\"", v, present)
	}
	if v, present := byName["b"]; !present || v != any(42) {
		t.Errorf("join 的输入里前驱 \"b\" 的值 = %#v (present=%v), want 42", v, present)
	}
	if len(byName) != 2 {
		t.Errorf("join 的输入含 %d 个键 %v, want 恰好前驱 \"a\" 与 \"b\" 两个键", len(byName), p1r19Keys(byName))
	}
}

// p1r19Keys 只用于失败输出，把 map 的键集合稳定地打出来。
func p1r19Keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestP1R19_JoinActivatesOnlyAfterBothPredecessorsFinished 观察契约 D3：
// a 与 b 均先于 join 完成。
//
// 先后由测试自有握手担保：a、b 在各自返回之前把一个完成信号投进 channel，join 只做一次
// 非阻塞清点。信号在节点返回前投出，节点返回之后消费者才知道它完成，因此「join 被正确激活」
// 蕴含清点结果必须是 2。清点不足 2 的唯一解释是 join 在凑齐之前就被激活。
func TestP1R19_JoinActivatesOnlyAfterBothPredecessorsFinished(t *testing.T) {
	finished := make(chan string, 4)

	g := graph.New().
		AddNode(graph.NodeFunc("entry", func(_ context.Context, in any) (any, error) { return in, nil })).
		AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) {
			finished <- "a"
			return "out-a", nil
		})).
		AddNode(graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) {
			finished <- "b"
			return "out-b", nil
		})).
		AddNode(graph.NodeFunc("join", func(_ context.Context, _ any) (any, error) {
			seen := 0
			for {
				select {
				case <-finished:
					seen++
					continue
				default:
					return seen, nil
				}
			}
		})).
		SetEntry("entry").
		SetOutput("join")
	g = g.AddEdge("entry", "a").AddEdge("entry", "b").AddJoin([]string{"a", "b"}, "join")

	out := p1r19RunBounded(t, g)
	if out.err != nil {
		t.Fatalf("Run 未执行汇聚屏障图：%v", out.err)
	}
	got, present := out.res.Value("join")
	if !present {
		t.Fatalf("Result.Value(\"join\") 不存在，Completed()=%v", out.res.Completed())
	}
	if got != any(2) {
		t.Errorf("join 被激活时只清点到自己已完成的 %d 个前驱, want 2（a、b 必须都先于 join 完成）", got)
	}
	for _, want := range []string{"a", "b"} {
		found := false
		for _, name := range out.res.Completed() {
			if name == want {
				found = true
			}
		}
		if !found {
			t.Errorf("Result.Completed() = %v，缺少前驱 %q", out.res.Completed(), want)
		}
	}
}

// TestP1R19_JoinOutputFlowsToDownstreamAndProjection 观察契约 D4：join 的输出作为普通
// 节点的输出继续往下游流，并成为图的输出投影。
//
// 这里把输出节点设成 join 之后的 tail：tail 必须收到 join 的返回值（而不是任何一个前驱的值），
// 而 Output() 必须是 tail 的返回值。票面 §7 与切片 9 反复钉住「err==nil 而 Output()==nil 的
// 静默通道」，汇聚落地后这条通道要由新形状重新关一次。
func TestP1R19_JoinOutputFlowsToDownstreamAndProjection(t *testing.T) {
	tailInput := make(chan any, 1)

	g := graph.New().
		AddNode(graph.NodeFunc("entry", func(_ context.Context, in any) (any, error) { return in, nil })).
		AddNode(graph.NodeFunc("a", func(_ context.Context, _ any) (any, error) { return "out-a", nil })).
		AddNode(graph.NodeFunc("b", func(_ context.Context, _ any) (any, error) { return "out-b", nil })).
		AddNode(graph.NodeFunc("join", func(_ context.Context, _ any) (any, error) { return "joined", nil })).
		AddNode(graph.NodeFunc("tail", func(_ context.Context, in any) (any, error) {
			tailInput <- in
			return "tail-done", nil
		})).
		SetEntry("entry").
		SetOutput("tail")
	g = g.AddEdge("entry", "a").AddEdge("entry", "b").
		AddJoin([]string{"a", "b"}, "join").AddEdge("join", "tail")

	out := p1r19RunBounded(t, g)
	if out.err != nil {
		t.Fatalf("Run 未执行「汇聚后再下游」的图：%v", out.err)
	}
	if got := out.res.Output(); got != any("tail-done") {
		t.Errorf("Result.Output() = %#v, want \"tail-done\"（join 的输出必须能作为前驱继续路由）", got)
	}
	close(tailInput)
	var got any
	for v := range tailInput {
		got = v
	}
	if s, ok := got.(string); ok && (s == "out-a" || s == "out-b") {
		t.Fatalf("tail 收到的是前驱 %q 的输出而不是 join 的输出：%v", s, got)
	}
	if got != any("joined") {
		t.Errorf("tail 收到的输入 = %#v, want \"joined\"（join 的返回值）", got)
	}
}

// TestP1R19_LegalJoinGraphValidatesNilWhileIllegalStillRejected 观察契约 D5：
// Join 路由落地后，合法汇聚图在构建期被放行（票面 :168 说的「被拒绝的集合按种类缩小」），
// 而 R1 的 fail-closed 不变式不放松 —— 端点不存在的 Join 声明仍要在构建期被拒且可定位。
func TestP1R19_LegalJoinGraphValidatesNilWhileIllegalStillRejected(t *testing.T) {
	identity := func(_ context.Context, in any) (any, error) { return in, nil }

	t.Run("合法汇聚图放行", func(t *testing.T) {
		g := graph.New().
			AddNode(graph.NodeFunc("entry", identity)).
			AddNode(graph.NodeFunc("a", identity)).
			AddNode(graph.NodeFunc("b", identity)).
			AddNode(graph.NodeFunc("join", identity)).
			SetEntry("entry").SetOutput("join")
		g = g.AddEdge("entry", "a").AddEdge("entry", "b").AddJoin([]string{"a", "b"}, "join")
		if err := g.Validate(); err != nil {
			t.Errorf("Validate() = %v, want nil（Join 已是要路由的边种类，不该再被当成不受支持）", err)
		}
	})

	t.Run("对照：端点不存在的汇聚声明仍被拒且点名那个端点", func(t *testing.T) {
		g := graph.New().
			AddNode(graph.NodeFunc("entry", identity)).
			AddNode(graph.NodeFunc("a", identity)).
			AddNode(graph.NodeFunc("join", identity)).
			SetEntry("entry").SetOutput("join")
		g = g.AddEdge("entry", "a").AddJoin([]string{"a", "ghost"}, "join")
		err := g.Validate()
		if err == nil {
			t.Fatal("前驱 \"ghost\" 从未注册，Validate 却判定合法")
		}
		if !strings.Contains(err.Error(), `"ghost"`) {
			t.Errorf("错误 %q 没有点名不存在的汇聚前驱 \"ghost\"，无法定位这条声明", err)
		}
		if res, runErr := g.Run(context.Background(), "x"); runErr == nil || res != nil {
			t.Errorf("非法图仍可执行：res=%v err=%v", res, runErr)
		}
	})
}

// TestP1R19_JoinWithFewerThanTwoPredecessorsIsRejectedLocatably 观察契约 D6：
// 票面 :65 的「Join 前驱数 < 2」在 Join 落地后必须成为一条独立的构建期拒绝，
// 而不是退化成今天那句「这个种类本引擎不路由」（measuredBeforeDesign 实测过那句）。
func TestP1R19_JoinWithFewerThanTwoPredecessorsIsRejectedLocatably(t *testing.T) {
	identity := func(_ context.Context, in any) (any, error) { return in, nil }

	tests := []struct {
		name   string
		from   []string
		target string
	}{
		{name: "单前驱", from: []string{"a"}, target: "join"},
		{name: "空前驱", from: nil, target: "join"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := graph.New().
				AddNode(graph.NodeFunc("entry", identity)).
				AddNode(graph.NodeFunc("a", identity)).
				AddNode(graph.NodeFunc("b", identity)).
				AddNode(graph.NodeFunc(tc.target, identity)).
				SetEntry("entry").SetOutput("b")
			g = g.AddEdge("entry", "a").AddEdge("a", "b").AddJoin(tc.from, tc.target)

			err := g.Validate()
			if err == nil {
				t.Fatalf("AddJoin(%v, %q) 前驱不足 2 个却通过校验（票面 :65）", tc.from, tc.target)
			}
			if strings.Contains(err.Error(), "not routable") {
				t.Errorf("错误 %q 仍是「本种类不被路由」那句，不是前驱数不足的判定", err)
			}
			if !strings.Contains(err.Error(), `"`+tc.target+`"`) {
				t.Errorf("错误 %q 未点名汇聚目标 %q，无法定位这条声明", err, tc.target)
			}
		})
	}
}

// TestP1R19_UnexecutedPredecessorEndsAtStepLimitNotSilence 观察契约 D7：
// 声明过的 Join 前驱本次没执行时，屏障永不凑齐，Run 必须走到票面 :63 的步数安全阀，
// 而不是静默交出结果，也不能挂死。
//
// 夹具里 b 有一条会计数的条件自环：它保证「图仍在往前走」，于是没凑齐的屏障真正被安全阀
// 抓住（若整图再无待派激活，Run 会走成功分支，那是另一种形状、另属一个未裁决问题）。
// a 那条条件边永不命中，因此 join 的两个声明前驱里 a 永不完成。
func TestP1R19_UnexecutedPredecessorEndsAtStepLimitNotSilence(t *testing.T) {
	bounces := make(chan int, 8)
	count := 0

	g := graph.New(graph.WithStepLimit(4)).
		AddNode(graph.NodeFunc("entry", func(_ context.Context, in any) (any, error) { return in, nil })).
		AddNode(graph.NodeFunc("a", func(_ context.Context, in any) (any, error) { return in, nil })).
		AddNode(graph.NodeFunc("b", func(_ context.Context, in any) (any, error) {
			count++
			bounces <- count
			return in, nil
		})).
		AddNode(graph.NodeFunc("join", func(_ context.Context, in any) (any, error) { return "joined", nil })).
		SetEntry("entry").SetOutput("join")
	g = g.AddEdge("entry", "b").
		AddConditional("entry", "a", func(any) bool { return false }).
		AddConditional("b", "b", func(any) bool { return true }).
		AddJoin([]string{"a", "b"}, "join")

	if err := g.Validate(); err != nil {
		t.Fatalf("这张带安全阀的汇聚图被 Validate 拒绝：%v", err)
	}
	out := p1r19RunBounded(t, g)
	if out.err == nil {
		t.Fatalf("屏障凑不齐却交出了结果：res=%v（Output()=%#v），票面 :161-168 的 fail-closed 不许静默放行", out.res, out.res.Output())
	}
	if out.res != nil {
		t.Errorf("Run 报错的同时仍交出非 nil Result (%v), want nil", out.res)
	}
	if !errors.Is(out.err, graph.ErrStepLimitExceeded) {
		t.Errorf("Run 的错误 = %v, want errors.Is(err, graph.ErrStepLimitExceeded)（未收敛的图由 :63 的安全阀结束）", out.err)
	}
	if bounces2 := len(bounces); bounces2 == 0 {
		t.Error("b 一次都没跑：安全阀没在真实派发过激活之后才落下")
	}
}
