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
// 切片 30 加固（S24-STD-2）：D1/D2/D5 第二段这三行的**判据逐字未动**，动的是「前提如何建立」。
// 加固前那三行把前提建立在**观测**上，机器一忙前提就没建立、或前提「看起来成立」而结论仍然红；
// 加固后前提由夹具**构造**。逐子用例对照：
//
//	D1  加固前：等第一个后继开始运行，读 started < 2000 且 started < 预算，再探一次结果通道。
//	          started 数节点体启动、steps 数激活交出，负载下两者差一个数量级 ⇒ 观测的是过期瞬时值。
//	    加固后：entry 一次交出 2000 条无条件后继，Sync 档的 entry 提交把消费者钉在
//	          steps==1、started==0、一条都没派发的位置上；在该点 cancel 之后再放行 ⇒
//	          安全阀只可能在取消之后、且只能在 dispatch 的循环头撞上。
//	D2  加固前：同一个窗口观测后取消，指望消费者恰好 parked 在 select 且队列里已躺着业务错误
//	          （实测每轮 5.8%，35/600）。
//	    加固后：entry 的三条后继在同一次派发里全部交出；tail 的提交是停靠点，停住后先放闸
//	          （failer 停在无缓冲投递上）、再 cancel、再放行 ⇒ 两条 select 分支同时就绪，
//	          命中率抬到约 50%/轮，「伪装」这件事变成可复现的构造而不是运气。
//	D5′ 加固前：同一个观测式窗口。
//	    加固后：槽位上限 5 与「登记后上闸、零投递」的节点体让「派发停住、积压在 pending」成为
//	          算术后果；取消之后没有任何投递，所以只有取消分支能交出结论。
//
// 五处「派发窗口」守卫一一映射成「停靠点」守卫（仍属前提红分型，见 p1r17AwaitDock 的 ①…⑤ 注释）。
// 四个常量（p1r17Grace/p1r17Succ/p1r17StepBudget/p1r17RaceRounds）取值不变。
//
// D3/D4/D5 第一段/D6 的前提已由构造或 channel 会合担保，本片一行未动。
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
	"sync"
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

// errP1R17Business 是测试自造的业务形状错误。加固形里它由闸门放行之后才交出
// （p1r17InHandGraph 的 failer），反向对照行里它由节点直接交出且全程没有取消 ——
// 两种场合都不该出现在「调用方已取消」的那次结论里。
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

// p1r17Dock 是切片 30 的加固会合点：把 Sync 档的 Checkpointer 当成一次「位置证明」。
//
// 引擎在 Sync 档由消费者同步调用 sink.Append（durability.go:103），调用点落在 complete 之后、
// 下一轮 dispatch 之前（scheduler.go:170-173）。让 Append 停住就把消费者钉死在一个可复算的
// 位置上：该节点的后继激活已经在 pending 里、一条都还没派发，steps 也停在派发之前。
// 「取消」与「安全阀/队列项」的先后不再靠观测一个可能过期的瞬时值，而由放行次序构造。
//
// 这条缝只当会合点用：不读引擎内部态、不断言持久化形状（契约 forbiddenShortcuts 第 5 条）。
type p1r17Dock struct {
	want        string
	entered     chan struct{} // 关闭＝消费者已进入停靠点
	release     chan struct{} // 测试侧放行
	releaseOnce sync.Once
	armed       atomic.Bool // 只停一次

	// nodeAtDock 由消费者在 close(entered) 之前写、测试在 <-entered 之后读，
	// happens-before 由这条 channel 担保，因此不需要锁；CAS 之后的 Append 不再写它。
	nodeAtDock string
}

func newP1R17Dock(t *testing.T, want string) *p1r17Dock {
	t.Helper()
	d := &p1r17Dock{want: want, entered: make(chan struct{}), release: make(chan struct{})}
	// 守卫失败也必须放行，否则消费者永远停在 Append 里，留下的停摆 goroutine 会拖累后续复跑。
	t.Cleanup(d.open)
	return d
}

// Append 实现 graph.Checkpointer：只在 want 那一次提交上停靠。
func (d *p1r17Dock) Append(_ context.Context, cp graph.Checkpoint) error {
	if cp.Node != d.want || !d.armed.CompareAndSwap(false, true) {
		return nil
	}
	d.nodeAtDock = cp.Node
	close(d.entered)
	<-d.release
	return nil
}

func (d *p1r17Dock) open() { d.releaseOnce.Do(func() { close(d.release) }) }

// p1r17Docked 是加固行共用的挂法：档位显式写出，不依赖 DurabilitySync 的零值便利。
func p1r17Docked(d *p1r17Dock, opts ...graph.Option) []graph.Option {
	return append(opts, graph.WithCheckpointer(d), graph.WithDurability(graph.DurabilitySync))
}

// p1r17FanDockGraph 是 D1 的加固夹具：entry 一次性声明 succCount 条无条件后继，
// 每条后继都汇到唯一出口。sink 停在 entry 那次提交上，于是停靠时刻的形状是
// 「succCount 条激活全部在 pending、零条被派发、steps==1」——这是构造出来的事实，
// 不是被看到的事实。放行后消费者必然回到 dispatch 的循环头，安全阀只能在那里撞上。
func p1r17FanDockGraph(t *testing.T, succCount int, d *p1r17Dock, opts ...graph.Option) (*graph.Graph, *atomic.Int64) {
	t.Helper()

	var started atomic.Int64
	g := graph.New(p1r17Docked(d, opts...)...)
	g.AddNode(graph.NodeFunc("entry", func(ctx context.Context, in any) (any, error) { return in, nil }))
	g.AddNode(graph.NodeFunc("p1r17exit", func(ctx context.Context, in any) (any, error) { return in, nil }))
	for i := 0; i < succCount; i++ {
		name := fmt.Sprintf("p1r17f%05d", i)
		g.AddNode(graph.NodeFunc(name, func(ctx context.Context, in any) (any, error) {
			started.Add(1)
			return in, nil
		}))
		g.AddEdge("entry", name)
		g.AddEdge(name, "p1r17exit")
	}
	g.SetEntry("entry").SetOutput("p1r17exit")

	if err := g.Validate(); err != nil {
		t.Fatalf("夹具构建期被拒（%v）——这不是行为观察，先修夹具", err)
	}
	return g, &started
}

// p1r17InHandGraph 是 D2 的加固夹具：entry 的三条后继各自的职责是把「业务错误已在手上」
// 与「ctx 已取消」构造成同时成立，而不是指望撞上：
//
//	failer → 闸门放行之后才交出 errP1R17Business（队列是无缓冲的，它必然停在投递上）
//	holder → 永不完成（图因此不可能合法收敛，取消是唯一可交出的结论）
//	tail   → 立即完成且无出边，它那次 Sync 提交就是停靠点
//
// 停在 tail 的提交上 ⇒ pending 已空（三条后继在同一次派发里全部交出），放行后消费者只能
// 回到 select；此时 failer 的投递与 ctx.Done() 两条就绪分支同时可走，归属由 ② 号守卫决定。
func p1r17InHandGraph(t *testing.T, d *p1r17Dock, failGate chan struct{}, opts ...graph.Option) (*graph.Graph, *atomic.Int64) {
	t.Helper()

	var started atomic.Int64
	// allStarted 是三条后继的自登记栅栏：failer/holder/tail 各自在阻塞或提交之前 +1 并 Done，
	// tail 等到栅栏齐了才交出那次作为停靠点的提交。少了这道会合，三条 goroutine 的
	// 启动次序是并发的，D2 的「结论已在手上」守卫会在合法轮次上误报。
	var allStarted sync.WaitGroup
	allStarted.Add(3)
	register := func() { started.Add(1); allStarted.Done() }

	g := graph.New(p1r17Docked(d, opts...)...)
	g.AddNode(graph.NodeFunc("entry", func(ctx context.Context, in any) (any, error) { return in, nil }))
	g.AddNode(graph.NodeFunc("failer", func(ctx context.Context, in any) (any, error) {
		register()
		<-failGate
		return nil, errP1R17Business
	}))
	g.AddNode(graph.NodeFunc("holder", func(ctx context.Context, in any) (any, error) {
		register()
		<-make(chan struct{}) // 永不完成：本行的结论只可能来自取消
		return in, nil
	}))
	g.AddNode(graph.NodeFunc("tail", func(ctx context.Context, in any) (any, error) {
		register()
		allStarted.Wait()
		return in, nil
	}))
	g.AddEdge("entry", "failer")
	g.AddEdge("entry", "holder")
	g.AddEdge("entry", "tail")
	g.SetEntry("entry").SetOutput("tail")

	if err := g.Validate(); err != nil {
		t.Fatalf("夹具构建期被拒（%v）——这不是行为观察，先修夹具", err)
	}
	return g, &started
}

// p1r17GatedFanGraph 是 D5 第二段的加固夹具：entry 一次性声明 succCount 条无条件后继，
// 且**每条节点体都闸在 gate 上**（零投递）。于是「槽位派满、派发停住、积压留在 pending」
// 是 WithMaxConcurrency 与闸门的算术后果，不需要被看到；startedUpTo 在第 capN 个节点体
// 真的启动时关闭（= 上限值，与 :249 那条越界即停的判据同源）。
func p1r17GatedFanGraph(t *testing.T, succCount, capN int, gate chan struct{}, opts ...graph.Option) (
	*graph.Graph, *atomic.Int64, chan struct{}) {
	t.Helper()

	var started atomic.Int64
	startedUpTo := make(chan struct{})
	g := graph.New(opts...)
	g.AddNode(graph.NodeFunc("entry", func(ctx context.Context, in any) (any, error) { return in, nil }))
	g.AddNode(graph.NodeFunc("p1r17exit", func(ctx context.Context, in any) (any, error) { return in, nil }))
	for i := 0; i < succCount; i++ {
		name := fmt.Sprintf("p1r17f%05d", i)
		g.AddNode(graph.NodeFunc(name, func(ctx context.Context, in any) (any, error) {
			// 先自登记再上闸：本行的前提是「第 capN 个节点体确实已开始运行、且一条都不投递」，
			// 登记在闸门之后就让这个前提永远无法建立。
			if int(started.Add(1)) == capN {
				close(startedUpTo)
			}
			<-gate
			return in, nil
		}))
		g.AddEdge("entry", name)
		g.AddEdge(name, "p1r17exit")
	}
	g.SetEntry("entry").SetOutput("p1r17exit")

	if err := g.Validate(); err != nil {
		t.Fatalf("夹具构建期被拒（%v）——这不是行为观察，先修夹具", err)
	}
	return g, &started, startedUpTo
}

// p1r17AwaitDock 等消费者真的停在停靠点上，并把原夹具那 5 处「派发窗口」守卫一一对应成
// 「停靠点」守卫（契约 D4：仍属前提红分型，不是判据红）：
//
//	① 结果通道已有值 → Run 提前交出，停靠点没建立（旧 :168）
//	② grace 内没停靠  → 消费者没进入提交点，前提不成立（旧 :170-171）
//	③ 停靠到的不是 want 那条提交 → 会合点被别的节点占了（新增，停靠点特有）
//	④ 停靠时刻已有后继启动 → 派发没停在停靠点之后，位置证明失效（旧 :173-180 两条合并）
//	⑤ 放行之前 Run 已交出 → 窗口没建立（旧 :182-183）
//
// 返回的是停靠时刻的后继启动数（加固后它是 0），供判据行如实插值。
func p1r17AwaitDock(t *testing.T, out chan p1r17Outcome, d *p1r17Dock, started *atomic.Int64) int64 {
	t.Helper()
	select {
	case o := <-out:
		t.Fatalf("取消之前 Run 已经交出（err=%v, result=%v）：停靠点没建立，本行前提不成立", o.err, o.res)
	case <-d.entered:
	case <-time.After(p1r17Grace):
		t.Fatal("消费者没有停在任何 Sync 提交上：停靠点没建立，本行前提不成立")
	}
	if d.nodeAtDock != d.want {
		t.Fatalf("停靠到的提交是 %q，期望 %q：会合点被别的节点占了，位置证明不成立", d.nodeAtDock, d.want)
	}
	if ran := started.Load(); ran != 0 {
		t.Fatalf("取消时刻已有 %d 个后继开始运行：派发没有停在停靠点之后，"+
			"这一轮无法证明「超限只可能在取消之后撞上」，先修夹具而不是放宽判据", ran)
	}
	if !p1r17StillInFlight(out) {
		t.Fatal("取消之前 Run 已经交出：停靠点没建立，先修夹具")
	}
	return started.Load()
}

// p1r17AwaitInHandDock 是 D2 的停靠守卫。它与 D1 的那条共用会合点原语，但前提不同：
// D1 要证明「派发还没往前走」，D2 要证明「失败节点的结论已经在手上、且消费者确实回到不了
// 成功路径」。三条后继都在同一次派发里交出，所以停靠时刻的预期是 started==3（全部启动），
// 而图尚未收敛（holder 永不完成）与 Run 仍在飞才是要紧的自证。同样 5 处守卫、仍属前提红分型：
//
//	① 结果通道已有值 → Run 提前交出（旧 :168）
//	② grace 内没停靠 → 消费者没进入提交点（旧 :170-171）
//	③ 停靠到的不是 tail 那次提交 → 会合点错位（新增）
//	④ 三条后继没全部启动 → 失败项不在手上，本轮不构成「同时成立」（旧 :173-180 的对应位）
//	⑤ 放行之前 Run 已交出 → 窗口没建立（旧 :182-183）
func p1r17AwaitInHandDock(t *testing.T, out chan p1r17Outcome, d *p1r17Dock, started *atomic.Int64, succTotal int) {
	t.Helper()
	select {
	case o := <-out:
		t.Fatalf("取消之前 Run 已经交出（err=%v, result=%v）：停靠点没建立，本行前提不成立", o.err, o.res)
	case <-d.entered:
	case <-time.After(p1r17Grace):
		t.Fatal("消费者没有停在任何 Sync 提交上：停靠点没建立，本行前提不成立")
	}
	if d.nodeAtDock != d.want {
		t.Fatalf("停靠到的提交是 %q，期望 %q：会合点被别的节点占了，位置证明不成立", d.nodeAtDock, d.want)
	}
	// 三条后继必须都已启动，否则「失败节点的结论已经在手上」这句前提就没成立，
	// 这一轮判绿只是运气（tail 的节点体在启动前会等到第 3 个登记，见 p1r17InHandGraph）。
	if ran := started.Load(); ran != int64(succTotal) {
		t.Fatalf("停靠时刻只启动了 %d/%d 条后继：failer 的结论不在手上，本轮不构成"+
			"「业务错误与取消同时成立」，先修夹具而不是放宽判据", ran, succTotal)
	}
	if !p1r17StillInFlight(out) {
		t.Fatal("取消之前 Run 已经交出：停靠点没建立，先修夹具")
	}
}

// TestP1R17_CancellationWinsOverStepLimitDuringDispatch 是契约 D1（切片 30 加固形）。
//
// 预算 p1r17StepBudget 条。加固前「取消时图仍在派发」是靠读 started 的瞬时值观测的，
// 而 started 数的是节点体启动、steps 数的是激活交出，负载下两者能差一个数量级，
// 于是前提本身成了时序竞赛的产物（S24-STD-2 的三轮实测）。加固后同样的事是**构造**出来的：
// entry 把 p1r17Succ 条无条件后继一次性交进 pending，Sync 档的提交把消费者钉在
// 「steps==1、零条后继启动、一条都还没派发」这个位置上；测试在这个点上取消、再放行。
// 因此安全阀只可能在取消之后撞上，且撞它的必然是 dispatch 的循环头本身
// ——拿掉派发侧的取消判据（变异 τ）或把它挪到预算判据之后（变异 ο），这一步就交出超限。
//
// 调用方得到的必须是「我取消了」，不能是「图跑飞了」——后者是切片 13 的 T5
// （p1r13_step_limit_test.go:236）已经立下的意图，只是那一行让节点原样交回 ctx.Err()，管不到这条路径。
func TestP1R17_CancellationWinsOverStepLimitDuringDispatch(t *testing.T) {
	dock := newP1R17Dock(t, "entry")
	g, started := p1r17FanDockGraph(t, p1r17Succ, dock, graph.WithStepLimit(p1r17StepBudget))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := p1r17RunAsync(g, ctx)

	// ranAtCancel 加固后的含义：停靠时刻已启动的后继数，被证明为 0（位置读法，不是抽样）。
	ranAtCancel := p1r17AwaitDock(t, out, dock, started)
	cancel()
	dock.open()

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
// 票面 :68 后半句的逐字落点（切片 30 加固形）。
//
// 加固前这一行指望运气：它观测到「第一个后继已经开始运行」就取消，盼望消费者恰好 parked 在
// select、且队列里已经躺着一条业务错误 —— 实测每轮命中 5.8%（35/600），负载下连这点命中都不稳。
// 加固后这两件事由夹具**构造成同时成立**：entry 的三条后继在同一次派发里全部交出，
// tail 立即完成并用它那次 Sync 提交把消费者钉住；测试随后放行 failer（队列无缓冲，它的投递
// 必然停在消费者手上）、再取消、最后才放行停靠点。消费者回到 select 时，
// `<-ctx.Done()` 与「failer 的投递」两条分支同时就绪，select 随机挑一条 ——
// 挑中队列项就必须靠 ② 号守卫（scheduler.go:155）把这一项归到取消语义上。
//
// 命中率因此从 5.8% 抬到约 50%/轮：跨轮 0 容忍的形式不动（p1r17RaceRounds 仍 100，只增不减），
// 但这一行的牙齿不再依赖机器空闲。空转自查由变异 m9（整条删除 ②）承担：它必须杀红本行，
// 若 0 命中说明本轮形状其实由 ① 独自回答，按空转绿处理、回契约。
//
// 牙齿归属如实登记：这一轮判红来自「两条守卫同时拿掉」（变异 κ）。单独移除派发侧守卫（τ）时
// 本行判绿，说明因果落在队列项侧那条守卫上；但只要派发侧守卫还在，消费者就先在 dispatch 里
// 交出取消、根本走不到队列项分支，所以没有任何单条变异能把本行单独判红（χ/ρ 各 500 轮 0 命中）。
func TestP1R17_CancellationIsNotDisguisedAsNodeBusinessErrorAcrossRounds(t *testing.T) {
	const succTotal = 3 // {failer, holder, tail}，加固夹具的全部后继

	for round := 1; round <= p1r17RaceRounds; round++ {
		failGate := make(chan struct{})
		dock := newP1R17Dock(t, "tail")
		g, started := p1r17InHandGraph(t, dock, failGate, graph.WithStepLimit(1000000))

		ctx, cancel := context.WithCancel(context.Background())
		out := p1r17RunAsync(g, ctx)

		p1r17AwaitInHandDock(t, out, dock, started, succTotal)
		close(failGate) // 业务错误此刻正停在投递上
		cancel()        // 「该项已在手上」与「调用方已取消」同时成立
		dock.open()

		res := p1r17Await(t, out)
		cancel()
		if res.stalled {
			t.Fatalf("第 %d 轮：取消后 Run 未在有界阀内返回", round)
		}
		if errors.Is(res.err, errP1R17Business) && !errors.Is(res.err, context.Canceled) {
			// 插值来源与这句自证文字随加固形改写过（判据谓词本身逐字未动）：加固后业务错误由
			// 闸门放行产生、且放闸严格先于 cancel，所以「交出它」证明的是消费者在 ctx 已取消的
			// 情况下挑了队列项，而不是「错误只可能在 ctx.Done 之后才产生」。
			t.Errorf("第 %d 轮：取消被伪装成节点业务错误 %v（取消时刻已启动 %d/%d 个后继，"+
				"业务错误在放闸之后、取消之前就已停在投递上）", round, res.err, started.Load(), succTotal)
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
		// 加固形（切片 30）：槽位上限与闸门把「派发停住、积压留在 pending」变成算术后果——
		// entry 一次性交进 p1r17Succ 条无条件后继，每条节点体自登记之后闸在 gate 上（零投递），
		// 于是 :249 那条「越界的激活留在 pending 头部」必然在第 capN 条之后生效，
		// 消费者只能回到 select 并空等。此时唯一能让 Run 交出结论的路径就是取消分支（①）：
		// 拿掉它（变异 m5）不留任何投递，本行由「未在有界阀内返回」接住，判红且不混分类。
		const capN = 5
		gate := make(chan struct{})
		g, started, fifthStarted := p1r17GatedFanGraph(t, p1r17Succ, capN, gate,
			graph.WithStepLimit(p1r17StepBudget), graph.WithMaxConcurrency(capN))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		out := p1r17RunAsync(g, ctx)

		select {
		case <-fifthStarted:
		case o := <-out:
			t.Fatalf("取消之前 Run 已经交出（err=%v, result=%v）：派发停住的前提没建立，本行前提不成立", o.err, o.res)
		case <-time.After(p1r17Grace):
			t.Fatal("没有后继节点开始运行：槽位没打满，本行前提不成立")
		}
		ranAtCancel := started.Load()
		if ranAtCancel != capN {
			t.Fatalf("取消时刻已启动 %d 个后继，不等于槽位上限 %d：派发没停在 :249 那条判据上，"+
				"先修夹具而不是放宽判据", ranAtCancel, capN)
		}
		if !p1r17StillInFlight(out) {
			t.Fatal("取消之前 Run 已经交出：积压没建立，先修夹具")
		}
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
