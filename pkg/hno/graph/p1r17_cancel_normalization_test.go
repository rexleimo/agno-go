package graph_test

// 本文件是 P1 票面 B9（取消归一化）这一片的行为族，契约见
// docs/design/v3-test-scope-p1-graph-slice17.json（验收行 D1–D6 与本文件的 Test 一一对应，
// D7 是命令背书的结构性护栏行，刻意不写成「测试读自身源码」的自指断言）。
//
// 票面 :68 要求的是：节点阻塞期间 cancel ctx 时 Run 必须交出取消语义，
// 「不得伪装成节点业务错误」。契约 M-1 量出这句字面场景在修复前是绿的（各 300/300），
// 真正的缺口在消费者的另两个写点（M-2：整个调度器只在消费者 parked 时咨询 ctx）：
//   - D1 取消落在「消费者正在一次性派发一大批后继」的窗口里，安全阀在这之后才撞上 →
//     交出的却是 ErrStepLimitExceeded（M-3 实测 40/40）；
//   - D2 同一窗口里失败节点交出的业务错误顶掉取消语义（M-4 实测 17.5%~22.5%，竞态）。
//
// 观察面只用 graph 的公共入口与 errors.Is，不读 cfg/scheduler/pending/running，
// 不直接调用 dispatch/spawn/complete；goroutine 数与 queue 长度是票面 :40 明令禁止的
// 观察面，本文件一行都不碰（契约 M-7）。全部时序由 channel 握手担保（票面 :75）：
// 取消时刻由「第一个后继确实已经开始运行」「节点确实已进入阻塞」这类事件触发，
// 文件里没有一处 time.Sleep 参与定序，上界阀只把停摆变成可判定的失败输出。
//
// D1/D2 的前置自证写在一起，缺一就不能判那一轮：
//   - started < 全部后继 ⇒ 图尚未收敛（出口节点要等所有兄弟完成）；
//   - 非阻塞探一次 Run 的结果通道仍无值 ⇒ 取消之前 Run 还在飞，也就是安全阀还没撞上。
//     少了这一条，「派发窗口」会变成一句靠运气成立的前提，D1 的绿也就无从解释。
//
// 竞态行 D2 取「跨轮 0 容忍」（契约 raceAssertionForm）：不得把期望写成允许百分之几，
// 那等于把今天的漏检率回填成容差。
//
// 反向牙齿（契约 D6）刻意做厚：归一化最容易写坏成「任何时候都先返回 ctx.Err()」，
// 那会同时抹掉节点业务错误、正常收敛结论与未取消时的步数超限结论，所以这三件事
// 各自都有断言，且都带一个不会被取消沾到的独立场景。

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

const (
	// p1r17Grace 是一次 Run 的时间上界。它到点不是「算通过」，而是把停摆报成失败
	// （graph_test.go:623、p1r13:72、p1r14:46、p1r16:35 用的是同一条仓库惯例）。
	p1r17Grace = 10 * time.Second

	// p1r17Succ 是给消费者制造「正在一次性派发一大批后继」这段窗口的后继数。它只是把
	// 窗口撑到取消能落进去，不是被断言的期望值；被断言的是取消时图仍在派发。
	p1r17Succ = 2000

	// p1r17StepBudget 是 D1 的步数预算：它必须远小于 p1r17Succ，才能让「预算耗尽」这件
	// 事整件落在取消之后；也必须大于取消时刻已派发的步数，否则前提自证会先失败。
	p1r17StepBudget = 100

	// p1r17RaceRounds 是竞态行 D2 的轮数。本轮按 30 轮实测的每轮命中率约 5.8%（35/600），
	// 单次执行抓到红的概率只有 17/20 —— 判据形式（跨轮 0 容忍）不能动，能动的是样本量，
	// 所以取 100 轮：按同一命中率估算抓到红的概率为 1-0.942^100 ≈ 99.8%。轮数只能往上加，
	// 不得为了让某一行绿而往下减（契约 forbiddenShortcuts 第 1 条）。
	p1r17RaceRounds = 100
)

// errP1R17Business 是测试自造的业务形状错误。夹具里的节点只在 ctx.Done() 之后才交出它，
// 所以「它出现在 Run 的错误里」这件事本身就等价于「取消被伪装成节点业务错误」。
var errP1R17Business = errors.New("p1r17: store connection reset by peer")

// errProbeEOF 是「节点包装了别的哨兵」这一种出口用的哨兵，与取消和业务错误都不同。
var errProbeEOF = errors.New("p1r17: unexpected end of probe stream")

// p1r17Outcome 是一次 Run 的交出物。stalled 为真时 res/err 无意义：那是阀到点，判失败。
type p1r17Outcome struct {
	res     *graph.Result
	err     error
	stalled bool
}

// p1r17RunAsync 把一次 Run 交给一条 goroutine，并交出结果通道（带缓冲，节点侧永不卡投递）。
// 取消类断言必须先把 Run 放出去，否则「取消落在消费者忙窗口里」这一前提根本不存在。
func p1r17RunAsync(g *graph.Graph, ctx context.Context) chan p1r17Outcome {
	out := make(chan p1r17Outcome, 1)
	go func() {
		res, err := g.Run(ctx, "seed")
		out <- p1r17Outcome{res: res, err: err}
	}()
	return out
}

// p1r17Await 在有界阀内取回一次 Run 的交出物；阀到点报停摆，不报「没超时就算过」。
func p1r17Await(t *testing.T, out chan p1r17Outcome) p1r17Outcome {
	t.Helper()
	select {
	case o := <-out:
		return o
	case <-time.After(p1r17Grace):
		return p1r17Outcome{stalled: true}
	}
}

func p1r17Run(t *testing.T, g *graph.Graph, ctx context.Context) p1r17Outcome {
	t.Helper()
	return p1r17Await(t, p1r17RunAsync(g, ctx))
}

// p1r17StillInFlight 非阻塞地探一次结果通道：有值就说明 Run 在取消之前已经交出，
// 本次观察的窗口没有建立，必须报前提失败而不是把结论记在错误语义上。
func p1r17StillInFlight(out chan p1r17Outcome) bool {
	select {
	case <-out:
		return false
	default:
		return true
	}
}

// p1r17RaceGraph 搭出「取消能落进派发窗口」的夹具：
//
//	entry → {failer, spawner}
//	spawner → f0 … f{succCount-1} → exit（唯一出口）
//
// spawner 由 start 闸门放行；它一成功，消费者就要在一次 dispatch 里放出 succCount 条激活。
// failer 阻塞到 ctx 取消之后才交出业务错误。firstStarted 在第一个 f 节点真的开始运行时关闭，
// started 是已经启动的后继数 —— 两者都来自节点体内的自登记，不是私有字段读法。
func p1r17RaceGraph(t *testing.T, succCount int, failerBody func(context.Context, any) (any, error), opts ...graph.Option) (
	*graph.Graph, chan struct{}, *atomic.Int64, chan struct{}) {
	t.Helper()

	start := make(chan struct{})
	firstStarted := make(chan struct{})
	var started atomic.Int64

	g := graph.New(opts...)
	g.AddNode(graph.NodeFunc("entry", func(ctx context.Context, in any) (any, error) { return in, nil }))
	g.AddNode(graph.NodeFunc("spawner", func(ctx context.Context, in any) (any, error) {
		<-start
		return in, nil
	}))
	g.AddNode(graph.NodeFunc("failer", failerBody))
	g.AddEdge("entry", "failer")
	g.AddEdge("entry", "spawner")

	for i := 0; i < succCount; i++ {
		name := fmt.Sprintf("p1r17f%05d", i)
		g.AddNode(graph.NodeFunc(name, func(ctx context.Context, in any) (any, error) {
			if started.Add(1) == 1 {
				close(firstStarted)
			}
			return in, nil
		}))
		g.AddEdge("spawner", name)
		g.AddEdge(name, "p1r17exit")
	}
	g.AddNode(graph.NodeFunc("p1r17exit", func(ctx context.Context, in any) (any, error) { return in, nil }))
	g.SetEntry("entry").SetOutput("p1r17exit")

	if err := g.Validate(); err != nil {
		t.Fatalf("夹具构建期被拒（%v）——这不是行为观察，先修夹具", err)
	}
	return g, start, &started, firstStarted
}

// p1r17AwaitDispatchWindow 等第一个后继真的开始运行，并在同一处完成两项前置自证：
// 图尚未收敛、Run 还在飞。任一不成立就报「窗口没建立」，与「取消语义不对」是两类失败。
func p1r17AwaitDispatchWindow(t *testing.T, out chan p1r17Outcome, firstStarted chan struct{}, started *atomic.Int64, total, budget int) int64 {
	t.Helper()
	select {
	case <-firstStarted:
	case o := <-out:
		t.Fatalf("取消之前 Run 已经交出（err=%v, result=%v）：派发窗口没建立，本行前提不成立", o.err, o.res)
	case <-time.After(p1r17Grace):
		t.Fatal("没有任何后继节点开始运行：消费者没进入派发窗口，本行前提不成立")
	}
	ran := started.Load()
	if ran >= int64(total) {
		t.Fatalf("取消时刻已启动 %d 个后继，等于全部 %d 个：图可能已经合法收敛，"+
			"这一轮无法证明「取消时仍在派发」，先加强夹具而不是放宽判据", ran, total)
	}
	if ran >= int64(budget) {
		t.Fatalf("取消时刻已启动 %d 个后继，不少于步数预算 %d：安全阀可能已经在取消之前撞上，"+
			"这一步的超限不是被取消掩盖的", ran, budget)
	}
	if !p1r17StillInFlight(out) {
		t.Fatal("取消之前 Run 已经交出：派发窗口没建立，先修夹具")
	}
	return ran
}

// p1r17BusinessAfterCancel 是 D2 用的节点出口：被取消之后交出「业务形状」的错误，
// 而不是把 ctx.Err() 原样交回。
func p1r17BusinessAfterCancel(ctx context.Context, in any) (any, error) {
	<-ctx.Done()
	return nil, errP1R17Business
}

// TestP1R17_CancellationWinsOverStepLimitDuringDispatch 是契约 D1。
//
// 预算 p1r17StepBudget 条，而取消之前图仍在派发那一大批后继（前置自证保证 Run 还没交出，
// 也就是阀还没撞上），于是「预算耗尽」整件都发生在调用方已经取消之后：消费者当时还在
// dispatch 的循环里，从没有回头咨询过 ctx。调用方得到的必须是「我取消了」，不能是
// 「图跑飞了」——后者是切片 13 的 T5（p1r13_step_limit_test.go:236）已经立下的意图，
// 只是那一行让节点原样交回 ctx.Err()，管不到这条路径。
func TestP1R17_CancellationWinsOverStepLimitDuringDispatch(t *testing.T) {
	g, start, started, firstStarted := p1r17RaceGraph(t, p1r17Succ,
		p1r17BusinessAfterCancel, graph.WithStepLimit(p1r17StepBudget))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := p1r17RunAsync(g, ctx)
	close(start)

	ranAtCancel := p1r17AwaitDispatchWindow(t, out, firstStarted, started, p1r17Succ, p1r17StepBudget)
	cancel()

	res := p1r17Await(t, out)
	if res.stalled {
		t.Fatal("取消后 Run 未在有界阀内返回：取消没被交给调用方")
	}
	if !errors.Is(res.err, context.Canceled) {
		t.Errorf("取消落在派发窗口内时 Run 的错误 = %v, want errors.Is(err, context.Canceled)"+
			"（取消时刻已启动 %d/%d 个后继，安全阀是在那之后才撞上的）", res.err, ranAtCancel, p1r17Succ)
	}
	if errors.Is(res.err, graph.ErrStepLimitExceeded) {
		t.Errorf("取消被报成步数超限：%v —— 预算是取消之后才被耗尽的，调用方要的是取消语义", res.err)
	}
	if res.res != nil {
		t.Errorf("取消后仍交出 Result %v", res.res)
	}
}

// TestP1R17_CancellationIsNotDisguisedAsNodeBusinessErrorAcrossRounds 是契约 D2，
// 票面 :68 后半句的逐字落点。
//
// 夹具与 D1 同形，只是把步数预算放到 1000000 让安全阀够不着，于是竞争回到消费者的 select：
// 它从 queue 取走失败节点交出的业务错误时，两件事已经同时成立 —— 该项已在手上、ctx 已取消。
// RED 时的实现（切片 16 收尾字节：取到队列项后 `return nil, item.err` 不看 ctx）于是随机挑一个，
// 挑中 queue 就把「我取消了」说成「节点业务失败」（M-4 的早期夹具：8/40、7/40、9/40；
// 本夹具实测：35/600 轮，即 17/20 次执行判红 —— 轮数因此取 100 而不是 30）。
//
// 牙齿归属如实登记：这一轮判红来自「两条守卫同时拿掉」（变异 κ）。单独移除派发侧守卫（τ）时
// 本行判绿，说明因果落在队列项侧那条守卫上；但只要派发侧守卫还在，消费者就先在 dispatch 里
// 交出取消、根本走不到队列项分支，所以没有任何单条变异能把本行单独判红（χ/ρ 各 500 轮 0 命中）。
//
// 判据形式是跨轮 0 容忍（契约 raceAssertionForm）：任何一轮交出业务错误即整行红。
func TestP1R17_CancellationIsNotDisguisedAsNodeBusinessErrorAcrossRounds(t *testing.T) {
	for round := 1; round <= p1r17RaceRounds; round++ {
		g, start, started, firstStarted := p1r17RaceGraph(t, p1r17Succ,
			p1r17BusinessAfterCancel, graph.WithStepLimit(1000000))

		ctx, cancel := context.WithCancel(context.Background())
		out := p1r17RunAsync(g, ctx)
		close(start)

		ranAtCancel := p1r17AwaitDispatchWindow(t, out, firstStarted, started, p1r17Succ, 1000000)
		cancel()

		res := p1r17Await(t, out)
		cancel()
		if res.stalled {
			t.Fatalf("第 %d 轮：取消后 Run 未在有界阀内返回", round)
		}
		if errors.Is(res.err, errP1R17Business) && !errors.Is(res.err, context.Canceled) {
			t.Errorf("第 %d 轮：取消被伪装成节点业务错误 %v（取消时刻已启动 %d/%d 个后继，"+
				"业务错误只可能在 ctx.Done 之后产生）", round, res.err, ranAtCancel, p1r17Succ)
		} else if !errors.Is(res.err, context.Canceled) {
			t.Errorf("第 %d 轮：Run 的错误 = %v, want errors.Is(err, context.Canceled)", round, res.err)
		}
		if res.res != nil {
			t.Errorf("第 %d 轮：取消后仍交出 Result %v", round, res.res)
		}
	}
}

// TestP1R17_AlreadyCancelledContextIsReportedAsCancellation 是契约 D3（green-at-red）。
//
// 进入 Run 之前 ctx 就已经取消：三种形状（普通图 / 预算只有 1 条的宽扇出图 / 上限 1 个槽的图）
// 都必须交出取消语义且不交出 Result。M-6 实测它在修复前 60/60 成立，但此前没有任何断言在管这个入口，
// 而归一化最容易写坏的地方恰好就是「还没开始跑」这一支。
//
// 「预算一条」刻意用宽扇出图（5 条无条件后继、预算只有 1）而不是普通图：这样进入 Run 时
// 等待队列非空、派发分支必然被走到，而步数恰好在第二步撞上，于是这一支同时是
// 「预先取消不得被报成步数超限」的牙齿（契约 D3/ο）。
func TestP1R17_AlreadyCancelledContextIsReportedAsCancellation(t *testing.T) {
	shapes := []struct {
		name  string
		build func(t *testing.T) *graph.Graph
	}{
		{"普通图", func(t *testing.T) *graph.Graph { return p1r17SmallFan(t, graph.WithStepLimit(1000)) }},
		{"预算一条的宽扇出", func(t *testing.T) *graph.Graph { return p1r17WideChain(t, 5, graph.WithStepLimit(1)) }},
		{"上限一个槽", func(t *testing.T) *graph.Graph {
			return p1r17SmallFan(t, graph.WithStepLimit(1000), graph.WithMaxConcurrency(1))
		}},
	}
	for _, shape := range shapes {
		t.Run(shape.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // 先取消，再进 Run
			defer cancel()
			out := p1r17Run(t, shape.build(t), ctx)
			if out.stalled {
				t.Fatal("预先取消的 Run 未在有界阀内返回")
			}
			if !errors.Is(out.err, context.Canceled) {
				t.Errorf("预先取消的 Run 的错误 = %v, want errors.Is(err, context.Canceled)", out.err)
			}
			if errors.Is(out.err, graph.ErrStepLimitExceeded) {
				t.Errorf("预先取消的 Run 被报成步数超限：%v", out.err)
			}
			if out.res != nil {
				t.Errorf("预先取消仍交出 Result %v", out.res)
			}
		})
	}
}

// TestP1R17_BlockedNodeExitsAllNormalizeToCancellation 是契约 D4（green-at-red），
// 票面 :68 那句字面场景本身。
//
// 节点阻塞期间取消，无论它交回 ctx.Err()、自造的业务哨兵，还是包装了另一个无关哨兵的错误，
// Run 都必须交出取消语义。M-1 实测各 300/300 成立（消费者当时 parked 在 select 上）。
// 本行的牙齿弱于 D1/D2（契约已如实登记）：它钉的是「归一化不得把这条已经对的路径改坏」。
// 取消时刻由「节点确实进入了阻塞」这一事件担保，不靠 sleep 定序（:75）。
func TestP1R17_BlockedNodeExitsAllNormalizeToCancellation(t *testing.T) {
	flavors := []struct {
		name string
		body func(ctx context.Context, in any, blocked chan struct{}) (any, error)
	}{
		{"交回 ctx.Err()", func(ctx context.Context, in any, blocked chan struct{}) (any, error) {
			close(blocked)
			<-ctx.Done()
			return nil, ctx.Err()
		}},
		{"交回业务哨兵", func(ctx context.Context, in any, blocked chan struct{}) (any, error) {
			close(blocked)
			<-ctx.Done()
			return nil, errP1R17Business
		}},
		{"交回包装了别的哨兵的错误", func(ctx context.Context, in any, blocked chan struct{}) (any, error) {
			close(blocked)
			<-ctx.Done()
			return nil, fmt.Errorf("p1r17: node read aborted: %w", errProbeEOF)
		}},
	}
	for _, flavor := range flavors {
		t.Run(flavor.name, func(t *testing.T) {
			blocked := make(chan struct{})
			g := p1r17BlockingGraph(t, func(ctx context.Context, in any) (any, error) {
				return flavor.body(ctx, in, blocked)
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			out := p1r17RunAsync(g, ctx)

			select {
			case <-blocked:
			case o := <-out:
				t.Fatalf("节点还没阻塞 Run 就交出了（err=%v）：夹具没建立票面的字面场景", o.err)
			case <-time.After(p1r17Grace):
				t.Fatal("节点没有进入阻塞：本行前提不成立")
			}
			cancel()

			res := p1r17Await(t, out)
			if res.stalled {
				t.Fatal("节点阻塞期间取消后 Run 未在有界阀内返回")
			}
			if !errors.Is(res.err, context.Canceled) {
				t.Errorf("错误 = %v, want errors.Is(err, context.Canceled)", res.err)
			}
			if res.res != nil {
				t.Errorf("取消后仍交出 Result %v", res.res)
			}
		})
	}
}

// TestP1R17_CancellationWhileQueueHasBacklog 是契约 D5（green-at-red）：B8 与 B9 的交叠面。
//
// 第一段子用例把唯一的槽位占住、把其余后继留在等待队列里（width 与 capacity 的算术后果，
// 不是对 queue 长度的观察），取消时没有任何节点会向 queue 投递，所以它同时钉住
// 「取消之后不得再把积压激活放出去」这一条（契约 D5.teethMutant ρ/σ）。
// 第二段子用例是 M-3 的对照行：同一张「取消落在派发窗口」的图，只要把上限设成 5，
// 派发就会停住、消费者几乎立刻回到 select，于是修复前也能交出取消语义（40/40）。
// 它钉住的是「修复不得只在不限并发时生效」，也说明 D1 的红色来自派发窗口而不是来自上限本身。
func TestP1R17_CancellationWhileQueueHasBacklog(t *testing.T) {
	t.Run("一个槽位且有积压", func(t *testing.T) {
		g, started, firstStarted, release := p1r17CappedBacklogGraph(t, 3, 1)
		defer close(release)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		out := p1r17RunAsync(g, ctx)

		select {
		case <-firstStarted:
		case o := <-out:
			t.Fatalf("节点还没占住槽位 Run 就交出了（err=%v）", o.err)
		case <-time.After(p1r17Grace):
			t.Fatal("没有节点开始运行：本行前提不成立")
		}
		if ran := started.Load(); ran != 1 {
			t.Fatalf("槽位上限为 1 时同时启动了 %d 个节点：夹具与 B8 的语义不符，先修夹具", ran)
		}
		if !p1r17StillInFlight(out) {
			t.Fatal("取消之前 Run 已经交出：积压没建立")
		}
		cancel()

		res := p1r17Await(t, out)
		if res.stalled {
			t.Fatal("队列有积压时取消，Run 未在有界阀内返回")
		}
		if !errors.Is(res.err, context.Canceled) {
			t.Errorf("错误 = %v, want errors.Is(err, context.Canceled)", res.err)
		}
		if res.res != nil {
			t.Errorf("取消后仍交出 Result %v", res.res)
		}
		if ran := started.Load(); ran != 1 {
			t.Errorf("取消之后又有节点开始运行（共 %d 个，槽位上限 1）：等待队列里的积压激活被放了出去，"+
				"取消应当停在派发之前", ran)
		}
	})
	t.Run("上限让派发停住时也是取消语义", func(t *testing.T) {
		g, start, started, firstStarted := p1r17RaceGraph(t, p1r17Succ, p1r17BusinessAfterCancel,
			graph.WithStepLimit(p1r17StepBudget), graph.WithMaxConcurrency(5))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		out := p1r17RunAsync(g, ctx)
		close(start)

		ranAtCancel := p1r17AwaitDispatchWindow(t, out, firstStarted, started, p1r17Succ, p1r17StepBudget)
		cancel()

		res := p1r17Await(t, out)
		if res.stalled {
			t.Fatal("取消后 Run 未在有界阀内返回")
		}
		if !errors.Is(res.err, context.Canceled) {
			t.Errorf("错误 = %v, want errors.Is(err, context.Canceled)；预算 %d 在取消之后才可能撞上（取消时刻已启动 %d 个后继）",
				res.err, p1r17StepBudget, ranAtCancel)
		}
		if errors.Is(res.err, graph.ErrStepLimitExceeded) {
			t.Errorf("取消被报成步数超限：%v", res.err)
		}
	})
}

// TestP1R17_UncancelledBusinessErrorStillSurfacesVerbatim 是契约 D6（green-at-red），
// D1/D2 的反向牙齿，也是契约 D1/σ、D3/ο 三条变异的指定杀手。
//
// 没有被取消时，三件事必须保持原样：节点的业务错误逐字交出且不被归一化成取消、正常收敛的图
// 照旧交出 Output 与 Completed、未取消而步数耗尽时仍报 ErrStepLimitExceeded。
// 缺了这一段，「任何时候都先返回 ctx.Err()」这种实现能把 D1/D2 两行红一起变绿 ——
// 那正是票面 :63 的 B4 与 :60 的 B1 不允许的。
func TestP1R17_UncancelledBusinessErrorStillSurfacesVerbatim(t *testing.T) {
	t.Run("未取消时业务错误原样交出", func(t *testing.T) {
		g := p1r17BlockingGraph(t, func(ctx context.Context, in any) (any, error) {
			return nil, errP1R17Business
		})
		out := p1r17Run(t, g, context.Background())
		if out.stalled {
			t.Fatal("未取消的 Run 未在有界阀内返回")
		}
		if !errors.Is(out.err, errP1R17Business) {
			t.Errorf("错误 = %v, want errors.Is(err, %v)", out.err, errP1R17Business)
		}
		if errors.Is(out.err, context.Canceled) {
			t.Errorf("没取消却被归一化成取消：%v", out.err)
		}
		if out.res != nil {
			t.Errorf("节点失败时仍交出 Result %v", out.res)
		}
	})
	t.Run("未取消时正常收敛不受影响", func(t *testing.T) {
		out := p1r17Run(t, p1r17SmallFan(t, graph.WithStepLimit(1000)), context.Background())
		if out.stalled {
			t.Fatal("未取消的正常图未在有界阀内返回")
		}
		if out.err != nil {
			t.Errorf("正常图 Run 返回错误 %v, want nil", out.err)
		}
		if out.res == nil {
			t.Fatal("正常图没交出 Result")
		}
		if got, want := fmt.Sprint(out.res.Output()), "seed"; got != want {
			t.Errorf("Output() = %q, want %q", got, want)
		}
		wantCompleted := []string{"entry", "p1r17a", "p1r17b"}
		if got := out.res.Completed(); !reflect.DeepEqual(got, wantCompleted) {
			t.Errorf("Completed() = %v, want %v（三个节点各执行一次，字典序）", got, wantCompleted)
		}
	})
	t.Run("未取消的步数超限仍报超限", func(t *testing.T) {
		g := p1r17WideChain(t, 20, graph.WithStepLimit(1))
		out := p1r17Run(t, g, context.Background())
		if out.stalled {
			t.Fatal("预算耗尽的 Run 未在有界阀内返回")
		}
		if !errors.Is(out.err, graph.ErrStepLimitExceeded) {
			t.Errorf("未取消时的步数耗尽 = %v, want errors.Is(err, graph.ErrStepLimitExceeded)（票面 :63 的 B4）", out.err)
		}
		if errors.Is(out.err, context.Canceled) {
			t.Errorf("没人取消却报成取消：%v —— 归一化不得吞掉安全阀的结论", out.err)
		}
		if out.res != nil {
			t.Errorf("预算耗尽时仍交出 Result %v", out.res)
		}
	})
}

// p1r17CappedBacklogGraph 是 D5 第一段的夹具：entry 一次性声明 width 条无条件后继，
// 而槽位只有 capacity 个，于是 width-capacity 条激活必然留在等待队列里 —— 这是构建期
// 声明与上限的算术后果，不是对 queue 长度的观察。每个后继登记自己开始后阻塞在 release 上，
// 所以取消到来时没有任何节点会向 queue 投递，消费者面对的只有 ctx.Done 一个就绪项。
func p1r17CappedBacklogGraph(t *testing.T, width, capacity int) (*graph.Graph, *atomic.Int64, chan struct{}, chan struct{}) {
	t.Helper()

	release := make(chan struct{})
	firstStarted := make(chan struct{})
	var started atomic.Int64

	g := graph.New(graph.WithStepLimit(1000), graph.WithMaxConcurrency(capacity))
	g.AddNode(graph.NodeFunc("entry", func(ctx context.Context, in any) (any, error) { return in, nil }))
	for i := 0; i < width; i++ {
		name := fmt.Sprintf("p1r17w%03d", i)
		g.AddNode(graph.NodeFunc(name, func(ctx context.Context, in any) (any, error) {
			if started.Add(1) == 1 {
				close(firstStarted)
			}
			<-release
			return in, nil
		}))
		g.AddEdge("entry", name)
		g.AddEdge(name, "p1r17done")
	}
	g.AddNode(graph.NodeFunc("p1r17done", func(ctx context.Context, in any) (any, error) { return in, nil }))
	g.SetEntry("entry").SetOutput("p1r17done")

	if err := g.Validate(); err != nil {
		t.Fatalf("夹具构建期被拒（%v）——这不是行为观察，先修夹具", err)
	}
	return g, &started, firstStarted, release
}

// p1r17SmallFan 是最普通的图：entry 扇出 a、b，出口是 a。
func p1r17SmallFan(t *testing.T, opts ...graph.Option) *graph.Graph {
	t.Helper()
	g := graph.New(opts...)
	g.AddNode(graph.NodeFunc("entry", func(ctx context.Context, in any) (any, error) { return in, nil }))
	g.AddNode(graph.NodeFunc("p1r17a", func(ctx context.Context, in any) (any, error) { return in, nil }))
	g.AddNode(graph.NodeFunc("p1r17b", func(ctx context.Context, in any) (any, error) { return in, nil }))
	g.AddEdge("entry", "p1r17a")
	g.AddEdge("entry", "p1r17b")
	g.SetEntry("entry").SetOutput("p1r17a")
	if err := g.Validate(); err != nil {
		t.Fatalf("夹具构建期被拒：%v", err)
	}
	return g
}

// p1r17WideChain 是「预算很小、必然耗尽」的图：width 条无条件后继全部指向出口，
// 步数预算只够派发其中一部分。契约 D6 用它守住票面 :63 的 B4（安全阀只在未取消的推进语义里成立），
// D3 用同一形状守住「预先取消不得被报成步数超限」。
func p1r17WideChain(t *testing.T, width int, opts ...graph.Option) *graph.Graph {
	t.Helper()
	g := graph.New(opts...)
	g.AddNode(graph.NodeFunc("entry", func(ctx context.Context, in any) (any, error) { return in, nil }))
	g.AddNode(graph.NodeFunc("p1r17end", func(ctx context.Context, in any) (any, error) { return in, nil }))
	for i := 0; i < width; i++ {
		name := fmt.Sprintf("p1r17c%03d", i)
		g.AddNode(graph.NodeFunc(name, func(ctx context.Context, in any) (any, error) { return in, nil }))
		g.AddEdge("entry", name)
		g.AddEdge(name, "p1r17end")
	}
	g.SetEntry("entry").SetOutput("p1r17end")
	if err := g.Validate(); err != nil {
		t.Fatalf("夹具构建期被拒：%v", err)
	}
	return g
}

// p1r17BlockingGraph 是票面 :68 的字面场景：entry 之后只有一个会长时间阻塞的节点。
func p1r17BlockingGraph(t *testing.T, body func(context.Context, any) (any, error)) *graph.Graph {
	t.Helper()
	g := graph.New(graph.WithStepLimit(1000))
	g.AddNode(graph.NodeFunc("entry", func(ctx context.Context, in any) (any, error) { return in, nil }))
	g.AddNode(graph.NodeFunc("block", body))
	g.AddNode(graph.NodeFunc("p1r17out", func(ctx context.Context, in any) (any, error) { return in, nil }))
	g.AddEdge("entry", "block")
	g.AddEdge("block", "p1r17out")
	g.SetEntry("entry").SetOutput("p1r17out")
	if err := g.Validate(); err != nil {
		t.Fatalf("夹具构建期被拒：%v", err)
	}
	return g
}
