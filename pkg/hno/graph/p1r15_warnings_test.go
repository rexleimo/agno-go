package graph_test

// 本文件是 P1 票面 B7（判据 8：stepLimit 缺失告警）的行为族，契约见
// docs/design/v3-test-scope-p1-graph-slice15.json。观察面只用 graph 的公共入口
// （New/AddNode/AddEdge/AddConditional/Validate/Warnings/Run + Result 的读方法），
// 不读 cfg、不调私有 helper、不断内部遍历次序（票面 §1 禁观察面）。

import (
	"context"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

// isStepLimitCycleWarning 判一条提示是否在说「这张图有环，而你没设步数上限」。
//
// 两个词都要：只提 ring/cycle 的提示不会告诉调用方能拧哪个旋钮，只提 step limit 的
// 提示可能是别的判据的抱怨。票面 B7 的后半句要求「显式设置后不含它」，所以这条告警
// 必须在两次调用之间可认出（契约 B7-AMEND-1）。
func isStepLimitCycleWarning(w error) bool {
	if w == nil {
		return false
	}
	msg := strings.ToLower(w.Error())
	namesKnob := strings.Contains(msg, "step") && strings.Contains(msg, "limit")
	namesCycle := strings.Contains(msg, "cycle") || strings.Contains(msg, "loop")
	return namesKnob && namesCycle
}

// conditionalCycleGraph 搭一张票面 B7 的触发形状：entry → ping ⇄ pong，回来那条是
// 条件边，因此 §3.5 第 7 项放行（有条件边可断，环可被跳过），而环确实存在。
func conditionalCycleGraph(opts ...graph.Option) *graph.Graph {
	noop := func(_ context.Context, in any) (any, error) { return in, nil }
	never := func(any) bool { return false }
	return graph.New(opts...).
		AddNode(graph.NodeFunc("entry", noop)).
		AddNode(graph.NodeFunc("ping", noop)).
		AddNode(graph.NodeFunc("pong", noop)).
		AddEdge("entry", "ping").
		AddConditional("ping", "pong", never).
		AddConditional("pong", "ping", never).
		SetEntry("entry").SetOutput("entry")
}

// TestP1R15_ConditionalCycleWithoutStepLimitWarns 覆盖契约 T1：含合法条件环且未显式
// WithStepLimit 的图，Warnings() 必须含这条告警。
func TestP1R15_ConditionalCycleWithoutStepLimitWarns(t *testing.T) {
	g := conditionalCycleGraph()
	if err := g.Validate(); err != nil {
		t.Fatalf("含合法条件环的图被 Validate() 拒绝（%v）：票面 B7 要求这种图合法可跑，告警只是提示", err)
	}

	warnings := g.Warnings()
	var hits []string
	for _, w := range warnings {
		if isStepLimitCycleWarning(w) {
			hits = append(hits, w.Error())
		}
	}
	if len(hits) == 0 {
		t.Fatalf("Warnings() 交出 %d 条 %v，其中没有一条点名「步数上限 + 环」：含环又没设预算的图不给任何提示，调用方要等到 Run 跑了 1000 步撞上 ErrStepLimitExceeded 才知道该写 WithStepLimit", len(warnings), warnings)
	}
}

// stepLimitWarnings 挑出告警集合里那条点名旋钮的提示，供「不含它」的各行复用同一识别判据
// ——T1 的「含」与 T2/T3 的「不含」必须认同一个对象，否则两组断言可以各自漂移。
func stepLimitWarnings(warnings []error) []string {
	var hits []string
	for _, w := range warnings {
		if isStepLimitCycleWarning(w) {
			hits = append(hits, w.Error())
		}
	}
	return hits
}

// acyclicGraph 是 T3 的形状：无环且不设上限。判据 8 只在「存在环时」才警告，因此这张图
// 安静是它的一部分；少了这一行，「只要没显式设上限就提示」的实现能同时骗过 T1 与 T2。
func acyclicGraph(opts ...graph.Option) *graph.Graph {
	noop := func(_ context.Context, in any) (any, error) { return in, nil }
	return graph.New(opts...).
		AddNode(graph.NodeFunc("entry", noop)).
		AddNode(graph.NodeFunc("last", noop)).
		AddEdge("entry", "last").
		SetEntry("entry").SetOutput("entry")
}

// TestP1R15_ExplicitStepLimitSilencesTheWarning 覆盖契约 T2（票面 :66 后半句）：同一张
// 含环图显式 WithStepLimit 之后，Warnings() 不含这条告警。
func TestP1R15_ExplicitStepLimitSilencesTheWarning(t *testing.T) {
	g := conditionalCycleGraph(graph.WithStepLimit(50))
	if err := g.Validate(); err != nil {
		t.Fatalf("带 WithStepLimit(50) 的合法图被 Validate() 拒绝: %v", err)
	}
	if hits := stepLimitWarnings(g.Warnings()); len(hits) != 0 {
		t.Errorf("显式设置了步数上限，Warnings() 仍交出 %v：提示没在承诺的关闭条件下闭嘴，调用方会以为旋钮没生效", hits)
	}
}

// TestP1R15_AcyclicGraphWithoutStepLimitDoesNotWarn 覆盖契约 T3（判据 8 的触发条件）。
func TestP1R15_AcyclicGraphWithoutStepLimitDoesNotWarn(t *testing.T) {
	g := acyclicGraph()
	if err := g.Validate(); err != nil {
		t.Fatalf("无环合法图被 Validate() 拒绝: %v", err)
	}
	if hits := stepLimitWarnings(g.Warnings()); len(hits) != 0 {
		t.Errorf("无环图收到 %d 条步数上限告警 %v：把提示实现成「没设上限就啰嗦」会让每条正常图都带噪声，也把判据 8 的触发条件丢掉了", len(hits), hits)
	}
}

// TestP1R15_WarningDoesNotChangeValidationOrRun 覆盖契约 T4：告警是提示，不是拒绝，也不是
// 执行变更（票面 :119 禁止污染 Validate() 的致命语义）。
//
// 期望的 Completed()/Output() 取自实现前对同一拓扑的实测（/tmp/p1-cycle15/main.go：
// completed=[entry ping]、output=seed），不是把当前输出抄成期望。
func TestP1R15_WarningDoesNotChangeValidationOrRun(t *testing.T) {
	g := conditionalCycleGraph()
	if err := g.Validate(); err != nil {
		t.Fatalf("被警告的图 Validate() 交出非 nil error %v：判据 8 是提示，票面 :119 明令不得并入致命语义", err)
	}
	if hits := stepLimitWarnings(g.Warnings()); len(hits) == 0 {
		t.Fatal("本行要同时看见「有提示」与「执行不受影响」，提示缺席时它退化成一条空断言")
	}

	res, err := g.Run(context.Background(), "seed")
	if err != nil {
		t.Errorf("含环未设上限的图 Run 交出 error %v：提示不得改变执行结果", err)
	}
	if res == nil {
		t.Fatal("Run 交出 nil Result，无从核对可观察结果")
	}
	if got, want := strings.Join(res.Completed(), ","), "entry,ping"; got != want {
		t.Errorf("Completed() = %q, want %q（实现前实测值）：加提示不该动这张图的收敛", got, want)
	}
	if got, want := res.Output(), any("seed"); got != want {
		t.Errorf("Output() = %v, want %v", got, want)
	}
}

// TestP1R15_SelfLoopConditionalEdgeAlsoWarns 覆盖契约 T5：环的定义覆盖单节点自环。
// 判据 7 说合法环「由条件边构成」，没有把环长限定为 ≥2；只认两节点互指的实现会漏掉
// `body --cond--> body` 这种最常见的重试回路写法。
func TestP1R15_SelfLoopConditionalEdgeAlsoWarns(t *testing.T) {
	noop := func(_ context.Context, in any) (any, error) { return in, nil }
	never := func(any) bool { return false }
	g := graph.New().
		AddNode(graph.NodeFunc("entry", noop)).
		AddNode(graph.NodeFunc("body", noop)).
		AddEdge("entry", "body").
		AddConditional("body", "body", never).
		SetEntry("entry").SetOutput("entry")

	if err := g.Validate(); err != nil {
		t.Fatalf("自环条件边图被 Validate() 拒绝（%v）：实测本片设计阶段该拓扑合法，前提变了就不是本行的问题", err)
	}
	if hits := stepLimitWarnings(g.Warnings()); len(hits) == 0 {
		t.Errorf("自环条件边图没有收到步数上限告警（Warnings()=%v）：单节点自环同样能在谓词恒真时永不返回", g.Warnings())
	}
}

// TestP1R15_WarningsAreDeterministicAcrossRepeatedQueries 覆盖契约 T6：同一张图重复询问
// 得到确定答案。nodes 是 map，遍历不排序就会给出漂动的文案——而 graph.go 的
// rejectUnreachableNodes 早已确立「一次只点一个可定位对象、稳定文案优先」的仓库约定。
//
// 为什么图里要放两个环：起点若取自 map 遍历序，漂动有两种形态——换报另一个环，或把同一个环
// 报成不同的旋转（"pong" -> "ping" -> "pong"）。两个互不相交的环把这两种都看住。
//
// 设计时曾以为「单环图无论起点取自 nodes 还是 edges 都报同一个环，本行会退化成永真断言」，
// 这句是错的：实测（/tmp/p1-cycle15/probe_single_cycle_t6.py，变异 (f) 跑 4 轮，未变异对照 1 轮）
// 单环形状同样 4/4 判红，因为旋转后的文案就已经不同。两个环因此不是本行成立的前提，
// 而是覆盖面的加强；把它改回单环不会让本行变哑，但会丢掉「换环」这一形态。
func TestP1R15_WarningsAreDeterministicAcrossRepeatedQueries(t *testing.T) {
	noop := func(_ context.Context, in any) (any, error) { return in, nil }
	never := func(any) bool { return false }
	g := graph.New().
		AddNode(graph.NodeFunc("entry", noop)).
		AddNode(graph.NodeFunc("ping", noop)).
		AddNode(graph.NodeFunc("pong", noop)).
		AddNode(graph.NodeFunc("alpha", noop)).
		AddNode(graph.NodeFunc("beta", noop)).
		AddEdge("entry", "ping").
		AddEdge("entry", "alpha").
		AddConditional("ping", "pong", never).
		AddConditional("pong", "ping", never).
		AddConditional("alpha", "beta", never).
		AddConditional("beta", "alpha", never).
		SetEntry("entry").SetOutput("entry")

	if err := g.Validate(); err != nil {
		t.Fatalf("含两个条件环的合法图被 Validate() 拒绝: %v", err)
	}

	first := g.Warnings()
	hits := stepLimitWarnings(first)
	if len(hits) == 0 {
		t.Fatal("第一次询问没有交出步数上限告警，本行的确定性断言会空转")
	}
	for i := 1; i <= 20; i++ {
		next := g.Warnings()
		if len(next) != len(first) {
			t.Fatalf("第 %d 次询问长度变为 %d，首次为 %d", i, len(next), len(first))
		}
		if other := stepLimitWarnings(next); len(other) == 0 || other[0] != hits[0] {
			t.Fatalf("第 %d 次询问的告警与首次不一致：%v vs %q：同图报出不同的环，说明定起点用的是 map 遍历序", i, other, hits[0])
		}
	}
}

// TestP1R15_QueryingWarningsDoesNotMutateTheGraph 覆盖契约 T7：Warnings() 只读。
// 一张被询问过的图与一张没被询问过的同形图必须跑出同样的结果，且提示不会「问一次就没」。
// 少了这一行，「警告时顺手把预算补上」这种以写代读的实现可以一路绿到底。
func TestP1R15_QueryingWarningsDoesNotMutateTheGraph(t *testing.T) {
	asked := conditionalCycleGraph()
	unasked := conditionalCycleGraph()

	for i := 0; i < 2; i++ {
		if hits := stepLimitWarnings(asked.Warnings()); len(hits) == 0 {
			t.Fatalf("第 %d 次询问的 Warnings() 里没有步数上限告警：第 1 次就没有是提示缺席，问过之后才没有则是 Warnings() 消费/改写了图状态", i+1)
		}
	}

	resAsked, errAsked := asked.Run(context.Background(), "seed")
	resUnasked, errUnasked := unasked.Run(context.Background(), "seed")
	if errAsked != nil || errUnasked != nil {
		t.Fatalf("两次 Run 的 error 应都为 nil，实际 %v / %v", errAsked, errUnasked)
	}
	if got, want := strings.Join(resAsked.Completed(), ","), strings.Join(resUnasked.Completed(), ","); got != want {
		t.Errorf("被询问过的图 Completed()=%q，未询问的=%q：询问本身改变了执行", got, want)
	}
	if resAsked.Output() != resUnasked.Output() {
		t.Errorf("被询问过的图 Output()=%v，未询问的=%v", resAsked.Output(), resUnasked.Output())
	}
}
