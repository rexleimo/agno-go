package graph_test

// 本文件是 P1 票面 B8（并发上限 + FIFO 等待队列）这一片的行为族，契约见
// docs/design/v3-test-scope-p1-graph-slice16.json（验收行 C1–C9 与本文件的 Test 一一对应，
// C9 是命令背书的结构性护栏行，刻意不写成「测试读自身源码」的自指断言）。
//
// 观察面只用 graph 的公共入口（New/WithMaxConcurrency/WithStepLimit/AddNode/NodeFunc/
// AddEdge/SetEntry/SetOutput/Run + Result 的读方法），不读 cfg/scheduler/pending/running，
// 也不直接调用 dispatch/spawn/complete（契约 allowedTestSeam.forbiddenObservation）。
//
// 时序一律由 channel 握手担保，不用 sleep 定序（票面 :75）：
//   - 节点的起止用无缓冲 channel 向单一登记器同步登记，登记本身就是一次握手，
//     所以登记顺序就是真实发生顺序，且节点返回前它的事件必已被登记器收下；
//   - 「若干个兄弟必须同时在飞」由登记器在起始事件计数到达阈值时关闭的 gate 担保；
//   - 有界等待（p1r16LedgerBound、runWithinGrace）只把「上限被实现成永不放行」这种停摆
//     变成可判定的失败输出，没有任何一行的绿建立在「没超时就算过」上；
//   - 忙等（p1r16Busy）只把在飞区间撑到足以被另一个 goroutine 观测，不参与定序。

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

const (
	// p1r16LedgerBound 是登记器侧每次握手的时间上界。它的存在只为了把一个失灵的
	// 钳制（激活被扣在队列里却永不放行）从挂死变成失败，与 graph_test.go:623、
	// p1r13:72、p1r14:46 用的是同一条仓库惯例。
	p1r16LedgerBound = 2 * time.Second

	// p1r16AbsentWindow 是给「执行尝试」登记表留出的一趟有界观测：一次 spawn 之后节点体
	// 第一句就是一次带缓冲的非阻塞写入。窗口内没出现，才判该激活从未被派发（C8 子判据 (1)）。
	p1r16AbsentWindow = 200 * time.Millisecond

	// p1r16BusyIters 是在飞区间的确定性长度，见文件头注释。
	p1r16BusyIters = 200000
)

// p1r16Ledger 是测试侧唯一的轨迹观察点：单个 goroutine 拥有事件序列与起始计数，
// 因此不需要任何锁就能给出一条全序轨迹（契约 testSideInstrumentation）。
type p1r16Ledger struct {
	events   chan string
	queries  chan p1r16Query
	gates    chan p1r16Gate
	stop     chan struct{}
	attempts chan string
}

type p1r16Query struct {
	reply chan []string
}

type p1r16Gate struct {
	need  int
	ready chan struct{}
}

// p1r16Sink 是忙等结果的出口：值经带缓冲 channel 交出，而不是写进一个包级变量 ——
// 后者会被 -race 判成数据竞争（多个兄弟节点真的同时在飞时，这就是测试自己引入的竞争）。
var p1r16Sink = make(chan int, 1024)

func p1r16Busy() {
	x := 0
	for i := 0; i < p1r16BusyIters; i++ {
		x += i % 7
	}
	select {
	case p1r16Sink <- x:
	default:
	}
}

func p1r16Start(name string) string { return "start " + name }

func p1r16End(name string) string { return "end " + name }

func newP1R16Ledger() *p1r16Ledger {
	l := &p1r16Ledger{
		events:   make(chan string),
		queries:  make(chan p1r16Query),
		gates:    make(chan p1r16Gate),
		stop:     make(chan struct{}),
		attempts: make(chan string, 64),
	}
	go l.loop()
	return l
}

func (l *p1r16Ledger) loop() {
	var seen []string
	var gates []p1r16Gate
	starts := 0
	for {
		select {
		case e := <-l.events:
			seen = append(seen, e)
			if !strings.HasPrefix(e, "start ") {
				continue
			}
			starts++
			keep := gates[:0]
			for _, g := range gates {
				if starts >= g.need {
					close(g.ready)
					continue
				}
				keep = append(keep, g)
			}
			gates = keep
		case q := <-l.queries:
			q.reply <- append([]string(nil), seen...)
		case g := <-l.gates:
			if starts >= g.need {
				close(g.ready)
				continue
			}
			gates = append(gates, g)
		case <-l.stop:
			return
		}
	}
}

// register 同步登记一个起止事件；返回 false 表示登记器已经不在了（测试结束后
// 仍在收尾的节点体走这一支，不会 panic，也不会影响任何断言）。
func (l *p1r16Ledger) register(e string) bool {
	select {
	case l.events <- e:
		return true
	case <-time.After(p1r16LedgerBound):
		return false
	case <-l.stop:
		return false
	}
}

func (l *p1r16Ledger) snapshot() ([]string, bool) {
	q := p1r16Query{reply: make(chan []string, 1)}
	select {
	case l.queries <- q:
	case <-time.After(p1r16LedgerBound):
		return nil, false
	case <-l.stop:
		return nil, false
	}
	select {
	case s := <-q.reply:
		return s, true
	case <-time.After(p1r16LedgerBound):
		return nil, false
	}
}

// traceOrFail 在 Run 返回后取全量轨迹，作为本族所有顺序/峰值判据的证据来源。
func (l *p1r16Ledger) traceOrFail(t *testing.T) []string {
	t.Helper()
	trace, ok := l.snapshot()
	if !ok {
		t.Fatalf("取不到登记器轨迹：本族的每条判据都以这条轨迹为观察面")
	}
	return trace
}

func (l *p1r16Ledger) waitStarted(need int) bool {
	g := p1r16Gate{need: need, ready: make(chan struct{})}
	select {
	case l.gates <- g:
	case <-time.After(p1r16LedgerBound):
		return false
	}
	select {
	case <-g.ready:
		return true
	case <-time.After(p1r16LedgerBound):
		return false
	}
}

func (l *p1r16Ledger) drainAttempts(window time.Duration) []string {
	deadline := time.After(window)
	var got []string
	for {
		select {
		case name := <-l.attempts:
			got = append(got, name)
		case <-deadline:
			return got
		}
	}
}

func (l *p1r16Ledger) close() { close(l.stop) }

// sibling 是一个可观测节点：登记起始 → 取入口快照 → 等 gate（need>0 时）→ 忙等 → 登记结束。
// 它把入口快照作为输出交出，于是「后启动者能否观测到前者已完成」经 Result.Value 这个
// 公共出口可判（票面 :92 行 4）。need 未打开时返回值带上 gate-timeout 标记，供 C4/C5/C7 判定。
func (l *p1r16Ledger) sibling(name string, need int) graph.Node {
	return graph.NodeFunc(name, func(_ context.Context, in any) (any, error) {
		select {
		case l.attempts <- name:
		default:
		}
		if !l.register(p1r16Start(name)) {
			return "register-timeout", nil
		}
		snap, ok := l.snapshot()
		if !ok {
			return "snapshot-timeout", nil
		}
		observed := strings.Join(snap, " | ")
		if need > 0 && !l.waitStarted(need) {
			return observed + " | gate-timeout", nil
		}
		p1r16Busy()
		l.register(p1r16End(name))
		return observed, nil
	})
}

// p1r16SiblingNames 是兄弟节点的声明顺序名。C2 的逐字串行轨迹直接依赖这份顺序。
func p1r16SiblingNames(n int) []string {
	names := make([]string, 0, n)
	for i := 0; i < n; i++ {
		if i == 0 {
			names = append(names, "a")
			continue
		}
		names = append(names, fmt.Sprintf("s%d", i))
	}
	return names
}

// p1r16Fan 构造一张合法图：entry 按声明顺序无条件 fan-out 到 n 个兄弟，兄弟全部汇聚到 done，
// done 是声明的输出。n 个兄弟彼此没有前驱关系，因此在没有上限时它们就是「同时在飞」的那一批。
func p1r16Fan(l *p1r16Ledger, n, need int, opts ...graph.Option) *graph.Graph {
	passthrough := func(_ context.Context, in any) (any, error) { return in, nil }
	g := graph.New(opts...).
		AddNode(graph.NodeFunc("entry", passthrough)).
		AddNode(graph.NodeFunc("done", func(_ context.Context, _ any) (any, error) { return "done", nil })).
		SetEntry("entry").
		SetOutput("done")
	for _, name := range p1r16SiblingNames(n) {
		g = g.AddNode(l.sibling(name, need)).AddEdge("entry", name).AddEdge(name, "done")
	}
	return g
}

// p1r16Peak 是轨迹上的最大同时在途节点数：票面 :92 行 4 的「区间不重叠」在 cap=1 时
// 等价于峰值为 1，而 cap>1 时它是唯一不依赖 goroutine 起始次序的可观察量。
func p1r16Peak(trace []string) int {
	cur, max := 0, 0
	for _, e := range trace {
		if strings.HasPrefix(e, "start ") {
			cur++
			if cur > max {
				max = cur
			}
			continue
		}
		cur--
	}
	return max
}

// TestP1R16_PeakInFlightRespectsCap 是契约 C1（本片的 red 候选行）：
// WithMaxConcurrency(1) 必须真的把同时在途的节点数限制住。改动前实测峰值 = 3
// （docs/design/v3-test-scope-p1-graph-slice16.json 的 M-2/M-3）。
func TestP1R16_PeakInFlightRespectsCap(t *testing.T) {
	l := newP1R16Ledger()
	defer l.close()

	g := p1r16Fan(l, 3, 0, graph.WithMaxConcurrency(1))
	res, err := runWithinGrace(t, g, "seed")
	if err != nil {
		t.Fatalf("Run 返回错误 %v：并发上限不得改变这张图的可执行性", err)
	}
	trace := l.traceOrFail(t)
	if got, want := p1r16Peak(trace), 1; got != want {
		t.Errorf("WithMaxConcurrency(1) 下同时在途节点峰值 = %d, want %d；轨迹 = %v", got, want, trace)
	}
	if got, want := len(res.Completed()), 7; got != want {
		t.Errorf("Completed() 有 %d 次激活（%v）, want %d：entry + 3 个兄弟 + done 被三个兄弟各激活一次。上限生效不得让任何节点丢掉",
			got, res.Completed(), want)
	}
}

// TestP1R16_CapOneSerializesInDeclarationOrder 是契约 C2：打满时新激活留在先进先出队列里，
// 完成一个自动派发一个，且派发顺序为声明顺序（票面 :67 前半句的逐字形态）。
// 今天实测「首个登记恰好是声明序第一个」只有 0.7%~2.4%（M-3）。
func TestP1R16_CapOneSerializesInDeclarationOrder(t *testing.T) {
	l := newP1R16Ledger()
	defer l.close()

	g := p1r16Fan(l, 3, 0, graph.WithMaxConcurrency(1))
	res, err := runWithinGrace(t, g, "seed")
	if err != nil {
		t.Fatalf("Run 返回错误 %v", err)
	}
	want := []string{
		p1r16Start("a"), p1r16End("a"),
		p1r16Start("s1"), p1r16End("s1"),
		p1r16Start("s2"), p1r16End("s2"),
	}
	if got := l.traceOrFail(t); !reflect.DeepEqual(got, want) {
		t.Errorf("cap=1 的登记轨迹 = %v, want %v（区间不重叠且按声明顺序逐个放行）", got, want)
	}

	// 上限只该改变时序，不该改变这次执行算出来的东西。
	base := newP1R16Ledger()
	defer base.close()
	wantRes, err := runWithinGrace(t, p1r16Fan(base, 3, 0), "seed")
	if err != nil {
		t.Fatalf("不带上限的同一拓扑 Run 返回错误 %v", err)
	}
	if !reflect.DeepEqual(res.Completed(), wantRes.Completed()) {
		t.Errorf("Completed() = %v, want 与不带上限的 %v 相同", res.Completed(), wantRes.Completed())
	}
	if res.Output() != wantRes.Output() {
		t.Errorf("Output() = %v, want %v", res.Output(), wantRes.Output())
	}
}

// TestP1R16_LaterSiblingObservesEarlierCompletion 是契约 C3：后启动的节点在入口就能观测到
// 前序节点已经完成，且这一观测经 Result.Value 交出（票面 :92 行 4 的后半句）。
// 今天实测该观测里含 end:a 的比例为 0%~0.8%（M-3）。
func TestP1R16_LaterSiblingObservesEarlierCompletion(t *testing.T) {
	l := newP1R16Ledger()
	defer l.close()

	g := p1r16Fan(l, 3, 0, graph.WithMaxConcurrency(1))
	res, err := runWithinGrace(t, g, "seed")
	if err != nil {
		t.Fatalf("Run 返回错误 %v", err)
	}

	for _, pair := range []struct {
		later, earlierDone, laterNotYet string
	}{
		{"s1", "end a", "start s2"},
		{"s2", "end s1", ""},
	} {
		v, ok := res.Value(pair.later)
		if !ok {
			t.Fatalf("Result.Value(%q) 没有产出：这个节点应当被放行过", pair.later)
		}
		if !strings.Contains(fmt.Sprint(v), pair.earlierDone) {
			t.Errorf("%s 在入口观测到的已完成集合不含 %q；实际交出 = %q", pair.later, pair.earlierDone, v)
		}
		if pair.laterNotYet != "" && strings.Contains(fmt.Sprint(v), pair.laterNotYet) {
			t.Errorf("%s 在入口就观测到了尚未被放行的 %s；实际交出 = %q", pair.later, pair.laterNotYet, v)
		}
	}
}

// TestP1R16_ZeroMeansUnlimited 是契约 C4（green-at-red 护栏行）：WithMaxConcurrency(0) 就是
// 文档承诺的「不限」。这一行钉住的是实现方向 —— 钳制必须以「上限为正数」为前置条件，
// 否则 0 会变成一个谁也拿不到槽位的队列。
func TestP1R16_ZeroMeansUnlimited(t *testing.T) {
	l := newP1R16Ledger()
	defer l.close()

	g := p1r16Fan(l, 3, 3, graph.WithMaxConcurrency(0))
	res, err := runWithinGrace(t, g, "seed")
	if err != nil {
		t.Fatalf("WithMaxConcurrency(0) 的 Run 返回错误 %v：0 表示不限", err)
	}
	for _, name := range p1r16SiblingNames(3) {
		v, ok := res.Value(name)
		if !ok {
			t.Fatalf("%s 没有产出", name)
		}
		if strings.HasSuffix(fmt.Sprint(v), "gate-timeout") {
			t.Errorf("%s 等不到「三个兄弟同时在飞」的 gate：%v —— 0 被当成了 0 个槽位，即「不限」写成了「全串行」", name, v)
		}
	}
	if got, want := p1r16Peak(l.traceOrFail(t)), 3; got != want {
		t.Errorf("WithMaxConcurrency(0) 下峰值在飞 = %d, want %d（不限 = 三个兄弟能同时在飞）", got, want)
	}
}

// TestP1R16_NegativeDoesNotStall 是契约 C5（green-at-red 护栏行）：负数上限不得让 Run 停摆，
// 其效果与「不限」一致。依据是母约 :190 那条钳制自带 maxConcurrency > 0 的前置守卫；
// 「在构建期拒绝负数」是另一种处置，本切片未授权（契约 R16-UNADJ-1）。
func TestP1R16_NegativeDoesNotStall(t *testing.T) {
	l := newP1R16Ledger()
	defer l.close()

	g := p1r16Fan(l, 3, 3, graph.WithMaxConcurrency(-1))
	res, err := runWithinGrace(t, g, "seed")
	if err != nil {
		t.Fatalf("WithMaxConcurrency(-1) 的 Run 返回错误 %v", err)
	}
	base := newP1R16Ledger()
	defer base.close()
	wantRes, err := runWithinGrace(t, p1r16Fan(base, 3, 3), "seed")
	if err != nil {
		t.Fatalf("不带上限的基线图 Run 返回错误 %v", err)
	}
	if !reflect.DeepEqual(res.Completed(), wantRes.Completed()) {
		t.Errorf("WithMaxConcurrency(-1) 的 Completed() = %v, want 与不限基线 %v 相同（负数不该扣住任何激活）",
			res.Completed(), wantRes.Completed())
	}
	if res.Output() != wantRes.Output() {
		t.Errorf("WithMaxConcurrency(-1) 的 Output() = %v, want %v", res.Output(), wantRes.Output())
	}
	for _, name := range p1r16SiblingNames(3) {
		// 在场性必须先判：基线是在同一份实现上跑的，一次「谁都没跑起来」的退化会让
		// 上面两条相等性判据两边同为空而通过。契约 C5 要求的是「与 C4 同样通过三方 gate
		// 握手」，握手没发生就必须判红，而不是让 vacuous 的相等蒙过去。
		v, ok := res.Value(name)
		if !ok {
			t.Fatalf("%s 没有产出：负数上限把激活扣死在等待队列里，图一步都没走", name)
		}
		if strings.HasSuffix(fmt.Sprint(v), "gate-timeout") {
			t.Errorf("WithMaxConcurrency(-1) 下 %s 等不到同时在飞的 gate：%v —— 钳制漏了「上限为正数」这一前置条件", name, v)
		}
	}
}

// TestP1R16_CapBoundsPeakAcrossBiggerFanout 是契约 C6：上限是「不得超过 n」而不是「只能串行」。
// 兄弟数取 6 而不是 4：改动前实测 4 个兄弟时有 2/20 轮碰巧没有被放出 3 个以上（峰值判据的
// 漏检），6 个同时放出后「无上限」就必然越过 2 —— 这是契约 completionCriteria 第 2 条要求的
// 加强夹具，不是放宽期望。
func TestP1R16_CapBoundsPeakAcrossBiggerFanout(t *testing.T) {
	l := newP1R16Ledger()
	defer l.close()

	g := p1r16Fan(l, 6, 0, graph.WithMaxConcurrency(2))
	res, err := runWithinGrace(t, g, "seed")
	if err != nil {
		t.Fatalf("WithMaxConcurrency(2) 的 Run 返回错误 %v", err)
	}
	trace := l.traceOrFail(t)
	if got := p1r16Peak(trace); got > 2 {
		t.Errorf("WithMaxConcurrency(2) 下峰值在飞 = %d, want <= 2；轨迹 = %v", got, trace)
	}
	for _, name := range p1r16SiblingNames(6) {
		if !p1r16Paired(trace, name) {
			t.Errorf("轨迹 %v 里 %s 的起止没有成对出现：被扣在等待队列里的激活必须最终被放行，不能饿死", trace, name)
		}
	}

	base := newP1R16Ledger()
	defer base.close()
	wantRes, err := runWithinGrace(t, p1r16Fan(base, 6, 0), "seed")
	if err != nil {
		t.Fatalf("不带上限的同一拓扑 Run 返回错误 %v", err)
	}
	if !reflect.DeepEqual(res.Completed(), wantRes.Completed()) {
		t.Errorf("Completed() = %v, want 与不带上限的 %v 相同", res.Completed(), wantRes.Completed())
	}
}

func p1r16Paired(trace []string, name string) bool {
	starts, ends := 0, 0
	for _, e := range trace {
		if e == p1r16Start(name) {
			starts++
		}
		if e == p1r16End(name) {
			ends++
		}
	}
	return starts > 0 && starts == ends
}

// TestP1R16_CapTwoActuallyAdmitsTwo 是契约 C7（green-at-red 护栏行）：cap=2 且只有两个兄弟时，
// 两者必须能真正同时在飞。这一行补的是 C6 管不到的下界 —— 没有它，「把任何上限都实现成 1」
// 这种过紧钳制可以一路全绿（契约 teethMutant ε 的指定杀手就是本行）。
func TestP1R16_CapTwoActuallyAdmitsTwo(t *testing.T) {
	l := newP1R16Ledger()
	defer l.close()

	g := p1r16Fan(l, 2, 2, graph.WithMaxConcurrency(2))
	res, err := runWithinGrace(t, g, "seed")
	if err != nil {
		t.Fatalf("WithMaxConcurrency(2) 的 Run 返回错误 %v", err)
	}
	for _, name := range p1r16SiblingNames(2) {
		v, ok := res.Value(name)
		if !ok {
			t.Fatalf("%s 没有产出", name)
		}
		if strings.HasSuffix(fmt.Sprint(v), "gate-timeout") {
			t.Errorf("上限 2 却没有让两个兄弟同时在飞（%s 交出 %v）：上限被实现得过紧，等于把 cap 当成了 1", name, v)
		}
	}
	if got, want := p1r16Peak(l.traceOrFail(t)), 2; got != want {
		t.Errorf("峰值在飞 = %d, want %d（两个兄弟都该拿到槽位）", got, want)
	}
}

// TestP1R16_CapDoesNotConsumeStepBudget 是契约 C8（green-at-red 护栏行）：并发上限不改写步数
// 语义。子判据 (1) 钉住「越界的激活根本不被启动，而不是先启动再报错」；子判据 (2) 钉住
// 「留在等待队列里的激活不消耗步数预算」——这张图恰好需要 5 次激活，预算给到 5 就该收敛。
func TestP1R16_CapDoesNotConsumeStepBudget(t *testing.T) {
	t.Run("预算用尽时越界的激活不被启动", func(t *testing.T) {
		l := newP1R16Ledger()
		defer l.close()

		g := p1r16Fan(l, 2, 0, graph.WithMaxConcurrency(1), graph.WithStepLimit(2))
		_, err := runWithinGrace(t, g, "seed")
		if !errors.Is(err, graph.ErrStepLimitExceeded) {
			t.Fatalf("Run 的 err = %v, want errors.Is(..., ErrStepLimitExceeded)", err)
		}
		attempts := l.drainAttempts(p1r16AbsentWindow)
		for _, name := range attempts {
			if name == "s1" {
				t.Errorf("等待队列里的 s1 被执行了（尝试表 = %v）：步数越界必须拦在派发之前，而不是放出去再报错", attempts)
			}
		}
	})

	t.Run("等待队列不消耗步数预算", func(t *testing.T) {
		l := newP1R16Ledger()
		defer l.close()

		g := p1r16Fan(l, 2, 0, graph.WithMaxConcurrency(1), graph.WithStepLimit(5))
		res, err := runWithinGrace(t, g, "seed")
		if err != nil {
			t.Errorf("5 次激活就够收敛的图（entry、a、s1、done、done）在 WithStepLimit(5) 下返回 %v：等待队列被当成了步数来计", err)
			return
		}
		base := newP1R16Ledger()
		defer base.close()
		wantRes, err := runWithinGrace(t, p1r16Fan(base, 2, 0, graph.WithStepLimit(5)), "seed")
		if err != nil {
			t.Fatalf("不带上限的同一拓扑在 WithStepLimit(5) 下返回 %v：这条对照本身就该收敛", err)
		}
		if !reflect.DeepEqual(res.Completed(), wantRes.Completed()) {
			t.Errorf("Completed() = %v, want 与不带上限的 %v 相同", res.Completed(), wantRes.Completed())
		}
	})
}
