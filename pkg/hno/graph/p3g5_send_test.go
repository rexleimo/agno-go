// 切片 26（母约 §6 G5：动态扇出 Send + AddJoinSend）契约 D1–D9。
// package graph_test，只用导出 API；计数与输入记录均为夹具自有状态（契约 allowedTestSeam）。
//
// 并发安全口径：会被并行激活多次的节点（sum 一族）刻意写成纯函数体——「各收独有输入」
// 由聚合输入的派发序钉住，不依赖 goroutine 完成次序（契约 forbiddenObservations：
// 不用 time.Sleep 定序）；多次激活的写者（reduce 一族）的相邻两次激活之间都有屏障或
// 条件边建立的 happens-before，因此夹具不需要锁。
package graph_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

// countCompleted 数 Completed() 里某名字出现的次数：并行分支的执行次数由引擎的
// 完成账本观察，不经夹具计数器。
func countCompleted(res *graph.Result, name string) int {
	n := 0
	for _, c := range res.Completed() {
		if c == name {
			n++
		}
	}
	return n
}

// D1 扇出正例：source 对 3 项各派发一个 Send 到 summarize → summarize 恰执行 3 次、
// 每次收到该项独有输入；reduce 恰执行 1 次且输入含全部 3 份输出（按派发序）。
func TestP3G5_D1_FanoutDeliversPerDispatchInputsAndBarrierAggregatesOnce(t *testing.T) {
	reduceRuns := 0
	var reduceIn any
	g := graph.New()
	g.AddNode(graph.SenderFunc("fan", func(ctx context.Context, in any) (any, []graph.Send, error) {
		return "fanout", []graph.Send{
			{Node: "sum", In: 0},
			{Node: "sum", In: 1},
			{Node: "sum", In: 2},
		}, nil
	}))
	g.AddNode(graph.NodeFunc("sum", func(ctx context.Context, in any) (any, error) {
		return fmt.Sprintf("o%v", in), nil
	}))
	g.AddNode(graph.NodeFunc("reduce", func(ctx context.Context, in any) (any, error) {
		reduceRuns++
		reduceIn = in
		return "reduced", nil
	}))
	g.AddJoinSend("sum", "reduce")
	g.SetEntry("fan")
	g.SetOutput("reduce")
	res, err := g.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run err = %v, want nil", err)
	}
	if got := countCompleted(res, "sum"); got != 3 {
		t.Errorf("sum 执行次数 = %d, want 3（3 个 Send 各激活一次、各收独有输入）", got)
	}
	if reduceRuns != 1 {
		t.Fatalf("reduce 执行次数 = %d, want 1（屏障恰激活一次）", reduceRuns)
	}
	agg, ok := reduceIn.(map[string][]any)
	if !ok {
		t.Fatalf("reduce 输入类型 = %T, want map[string][]any", reduceIn)
	}
	if len(agg) != 1 {
		t.Errorf("reduce 聚合键集 = %d 个键, want 1（只声明了 sum 一个 source）", len(agg))
	}
	got, want := agg["sum"], []any{"o0", "o1", "o2"}
	if len(got) != len(want) {
		t.Fatalf("reduce 聚合输入长度 = %d, want %d（按派发序）", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("reduce 聚合输入[%d] = %v, want %v（按派发序逐一对应）", i, got[i], want[i])
		}
	}
	if res.Output() != "reduced" {
		t.Errorf("Output() = %v, want reduced", res.Output())
	}
}

// D2 零派发语义：source 激活但派发 0 个 Send → target 不激活；图上再无待派激活时
// Run 交 err==nil 而 Output()==nil（R19-Q1 同族形状按「不激活」钉死）。
func TestP3G5_D2_ZeroDispatchLeavesTargetUnactivatedAndRunConverges(t *testing.T) {
	targetRuns := 0
	g := graph.New()
	g.AddNode(graph.SenderFunc("fan", func(ctx context.Context, in any) (any, []graph.Send, error) {
		return "empty", nil, nil
	}))
	g.AddNode(graph.NodeFunc("sum", func(ctx context.Context, in any) (any, error) {
		return "s", nil
	}))
	g.AddNode(graph.NodeFunc("reduce", func(ctx context.Context, in any) (any, error) {
		targetRuns++
		return "reduced", nil
	}))
	g.AddJoinSend("sum", "reduce")
	g.SetEntry("fan")
	g.SetOutput("reduce")
	res, err := g.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run err = %v, want nil（零派发不是错误）", err)
	}
	if res == nil {
		t.Fatal("res = nil, want 非 nil 的 Result")
	}
	if targetRuns != 0 {
		t.Errorf("target 执行次数 = %d, want 0（零派发不激活 target）", targetRuns)
	}
	if res.Output() != nil {
		t.Errorf("Output() = %v, want nil", res.Output())
	}
}

// D3 普通出边共存：source 同时有无条件出边与 AddJoinSend 声明 → 普通后继照常激活，
// join-send target 仍等全部 Send。
func TestP3G5_D3_UnconditionalEdgeAndJoinSendCoexist(t *testing.T) {
	var plainIns []any
	reduceRuns := 0
	var reduceAgg map[string][]any
	g := graph.New()
	g.AddNode(graph.SenderFunc("fan", func(ctx context.Context, in any) (any, []graph.Send, error) {
		return "F", []graph.Send{{Node: "sum", In: "x"}, {Node: "sum", In: "y"}}, nil
	}))
	g.AddNode(graph.NodeFunc("plain", func(ctx context.Context, in any) (any, error) {
		plainIns = append(plainIns, in)
		return "p", nil
	}))
	g.AddNode(graph.NodeFunc("sum", func(ctx context.Context, in any) (any, error) {
		return "s(" + in.(string) + ")", nil
	}))
	g.AddNode(graph.NodeFunc("reduce", func(ctx context.Context, in any) (any, error) {
		reduceRuns++
		reduceAgg = in.(map[string][]any)
		return "reduced", nil
	}))
	g.AddJoinSend("sum", "reduce")
	g.AddEdge("fan", "plain")
	g.SetEntry("fan")
	g.SetOutput("reduce")
	res, err := g.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run err = %v, want nil", err)
	}
	if len(plainIns) != 1 || plainIns[0] != "F" {
		t.Errorf("普通出边后继输入 = %v, want [F]（无条件边照常激活、不吞并）", plainIns)
	}
	if got := countCompleted(res, "sum"); got != 2 {
		t.Errorf("sum 执行次数 = %d, want 2（join-send 路不吞并 Send 派发）", got)
	}
	if reduceRuns != 1 {
		t.Fatalf("reduce 执行次数 = %d, want 1（仍等全部 Send 凑齐）", reduceRuns)
	}
	if got := reduceAgg["sum"]; len(got) != 2 || got[0] != "s(x)" || got[1] != "s(y)" {
		t.Errorf("reduce 聚合输入 = %v, want [s(x) s(y)]", got)
	}
}

// D4 安全阀：source 每轮派发的 Send 引入新激活 → 图不收敛时被 WithStepLimit 拦住，
// 返回包 ErrStepLimitExceeded 的错误、res=nil。
func TestP3G5_D4_UnboundedFanoutIsStoppedByStepLimit(t *testing.T) {
	reduceRuns := 0
	g := graph.New(graph.WithStepLimit(50))
	g.AddNode(graph.SenderFunc("fan", func(ctx context.Context, in any) (any, []graph.Send, error) {
		sends := make([]graph.Send, 0, 100)
		for i := 0; i < 100; i++ {
			sends = append(sends, graph.Send{Node: "sum", In: i})
		}
		return "F", sends, nil
	}))
	g.AddNode(graph.NodeFunc("sum", func(ctx context.Context, in any) (any, error) {
		return fmt.Sprintf("o%v", in), nil
	}))
	g.AddNode(graph.NodeFunc("reduce", func(ctx context.Context, in any) (any, error) {
		reduceRuns++
		return "reduced", nil
	}))
	g.AddJoinSend("sum", "reduce")
	g.SetEntry("fan")
	g.SetOutput("reduce")
	res, err := g.Run(context.Background(), nil)
	if !errors.Is(err, graph.ErrStepLimitExceeded) {
		t.Fatalf("Run err = %v, want 包 ErrStepLimitExceeded（扇出激活照常计步）", err)
	}
	if res != nil {
		t.Errorf("res = %v, want nil（撞阀不交 Result）", res)
	}
	if reduceRuns != 0 {
		t.Errorf("reduce 执行次数 = %d, want 0（屏障没凑齐不激活）", reduceRuns)
	}
}

// D5 可达性：AddJoinSend 贡献 source→target 的声明边——只靠 join-send 入边的 target
// 不再被判不可达；不声明时仍被拒（反向钉）。
func TestP3G5_D5_JoinSendDeclaresReachabilityBothWays(t *testing.T) {
	build := func(withJoin bool) *graph.Graph {
		g := graph.New()
		g.AddNode(graph.SenderFunc("fan", func(ctx context.Context, in any) (any, []graph.Send, error) {
			return "F", []graph.Send{{Node: "sum", In: "x"}}, nil
		}))
		g.AddNode(graph.NodeFunc("sum", func(ctx context.Context, in any) (any, error) { return "s", nil }))
		g.AddNode(graph.NodeFunc("reduce", func(ctx context.Context, in any) (any, error) { return "r", nil }))
		if withJoin {
			g.AddJoinSend("sum", "reduce")
		}
		g.SetEntry("fan")
		g.SetOutput("reduce")
		return g
	}
	if err := build(true).Validate(); err != nil {
		t.Errorf("withJoin Validate() = %v, want nil（AddJoinSend 贡献 source→target 可达性边）", err)
	}
	err := build(false).Validate()
	if err == nil {
		t.Fatal("withoutJoin Validate() = nil, want 可达性拒绝（不过度贡献）")
	}
	if msg := err.Error(); !strings.Contains(msg, "unreachable") || !strings.Contains(msg, "reduce") {
		t.Errorf("withoutJoin 错误 = %q, want 点名 reduce 的 unreachable", msg)
	}
}

// D6 AddJoin 判据原样保留（反向对照）：单前驱 AddJoin 仍被 §3.5 第 6 项点名拒绝——
// 新声明不得顺手改变旧声明语义。
func TestP3G5_D6_SinglePredecessorAddJoinIsStillRejected(t *testing.T) {
	g := graph.New()
	g.AddNode(graph.NodeFunc("a", func(ctx context.Context, in any) (any, error) { return "a", nil }))
	g.AddNode(graph.NodeFunc("join", func(ctx context.Context, in any) (any, error) { return "j", nil }))
	g.AddJoin([]string{"a"}, "join")
	g.SetEntry("a")
	g.SetOutput("join")
	err := g.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want §3.5 第 6 项拒绝（两类声明判据不互通）")
	}
	for _, want := range []string{`join target "join"`, "at least 2"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("错误 = %v, want 含 %q（AddJoin 语义原样保留）", err, want)
		}
	}
}

// D7 Send 指向未注册节点 → 运行期点名错误（节点名 + 归因），Run 拒绝、res=nil，
// 不静默丢该派发（整体校验在任何派发入队之前，不做部分派发）。
func TestP3G5_D7_SendToUnregisteredNodeFailsClosedAtRuntime(t *testing.T) {
	sumRuns := 0
	g := graph.New()
	g.AddNode(graph.SenderFunc("fan", func(ctx context.Context, in any) (any, []graph.Send, error) {
		return "F", []graph.Send{{Node: "sum", In: "x"}, {Node: "ghost", In: "y"}}, nil
	}))
	g.AddNode(graph.NodeFunc("sum", func(ctx context.Context, in any) (any, error) {
		sumRuns++
		return "s", nil
	}))
	g.AddJoinSend("sum", "sum")
	g.SetEntry("fan")
	g.SetOutput("sum")
	res, err := g.Run(context.Background(), nil)
	if err == nil {
		t.Fatal("Run err = nil, want 运行期点名未注册目标（fail-closed，不静默丢）")
	}
	msg := err.Error()
	if !strings.HasPrefix(msg, "graph: ") {
		t.Errorf("错误 = %q, want graph: 前缀", msg)
	}
	if !strings.Contains(msg, "ghost") {
		t.Errorf("错误 = %q, want 点名 ghost", msg)
	}
	if res != nil {
		t.Errorf("res = %v, want nil", res)
	}
	if sumRuns != 0 {
		t.Errorf("sum 执行次数 = %d, want 0（未注册目标使整批派发被拒，不做部分派发）", sumRuns)
	}
}

// D8 非 Sender 节点零影响（反向对照）：不实现 SendRun 的节点在带缝引擎上行为同今天
// （执行次数/输出/Completed 不变）；既有全部测试零改动全过是本行的另一半。
func TestP3G5_D8_NonSenderNodesAreUnaffected(t *testing.T) {
	plainRuns, nextRuns, reduceRuns := 0, 0, 0
	g := graph.New()
	g.AddNode(graph.NodeFunc("plain", func(ctx context.Context, in any) (any, error) {
		plainRuns++
		return "p", nil
	}))
	g.AddNode(graph.NodeFunc("next", func(ctx context.Context, in any) (any, error) {
		nextRuns++
		return "n", nil
	}))
	g.AddNode(graph.NodeFunc("reduce", func(ctx context.Context, in any) (any, error) {
		reduceRuns++
		return "reduced", nil
	}))
	g.AddEdge("plain", "next")
	g.AddJoinSend("plain", "reduce")
	g.SetEntry("plain")
	g.SetOutput("next")
	res, err := g.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run err = %v, want nil（非 Sender 节点零影响）", err)
	}
	if plainRuns != 1 || nextRuns != 1 {
		t.Errorf("plain/next 执行次数 = %d/%d, want 1/1", plainRuns, nextRuns)
	}
	if reduceRuns != 0 {
		t.Errorf("reduce 执行次数 = %d, want 0（无 Send 派发则声明不激活）", reduceRuns)
	}
	if res.Output() != "n" {
		t.Errorf("Output() = %v, want n", res.Output())
	}
	if got := res.Completed(); len(got) != 2 || got[0] != "next" || got[1] != "plain" {
		t.Errorf("Completed() = %v, want [next plain]", got)
	}
}

// D9 多波扇出：source 因条件环重跑再派发第二波 → 每波各自凑齐、target 逐波激活
// （输入只含当波）。波次序由确定性的条件环（谓词读 fan 激活次数）保证，不依赖时序。
func TestP3G5_D9_MultiWaveFanoutSettlesPerWave(t *testing.T) {
	fanRuns := 0
	var reduceWaves []map[string][]any
	g := graph.New()
	g.AddNode(graph.SenderFunc("fan", func(ctx context.Context, in any) (any, []graph.Send, error) {
		fanRuns++
		if fanRuns == 1 {
			return "F", []graph.Send{{Node: "sum", In: "w1"}}, nil
		}
		return "F", []graph.Send{{Node: "sum", In: "w2a"}, {Node: "sum", In: "w2b"}}, nil
	}))
	g.AddNode(graph.NodeFunc("sum", func(ctx context.Context, in any) (any, error) {
		return "s(" + in.(string) + ")", nil
	}))
	g.AddNode(graph.NodeFunc("reduce", func(ctx context.Context, in any) (any, error) {
		agg := in.(map[string][]any)
		cp := map[string][]any{}
		for k, v := range agg {
			cp[k] = append([]any{}, v...)
		}
		reduceWaves = append(reduceWaves, cp)
		return "r", nil
	}))
	g.AddJoinSend("sum", "reduce")
	g.AddConditional("reduce", "fan", func(out any) bool { return fanRuns < 2 })
	g.SetEntry("fan")
	g.SetOutput("reduce")
	res, err := g.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run err = %v, want nil", err)
	}
	if len(reduceWaves) != 2 {
		t.Fatalf("target 激活波数 = %d, want 2（每波各自凑齐、逐波激活）", len(reduceWaves))
	}
	if w := reduceWaves[0]["sum"]; len(w) != 1 || w[0] != "s(w1)" {
		t.Errorf("第 1 波聚合 = %v, want [s(w1)]（输入只含当波）", w)
	}
	if w := reduceWaves[1]["sum"]; len(w) != 2 || w[0] != "s(w2a)" || w[1] != "s(w2b)" {
		t.Errorf("第 2 波聚合 = %v, want [s(w2a) s(w2b)]（不跨波累计、不跨波丢弃）", w)
	}
	if fanRuns != 2 {
		t.Errorf("fan 执行次数 = %d, want 2", fanRuns)
	}
	if res.Output() != "r" {
		t.Errorf("Output() = %v, want r", res.Output())
	}
}
