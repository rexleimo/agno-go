package graph

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

// queueItem 是生产者 goroutine 投向消费者的唯一消息。
// in 与 attempt 供消费者发射 trace 事件用：事件在消费侧串行发出（零锁不破），
// 因此生产者必须把发射所需的字段一并捎过来。
type queueItem struct {
	name    string
	out     any
	err     error
	in      any
	attempt int
	// sends 是该完成项（作为 Sender）本次激活派发的扇出；via/idx 标记该完成项本身
	// 是哪个 AddJoinSend source 的第 idx 个 Send 激活（普通激活两字段为零值）。
	sends []Send
	via   string
	idx   int
}

// activation 是一次待派发的节点执行，in 为其输入。
// via/idx 非零值时它是一次 Send 派发：via 是等待它的 AddJoinSend source 名。
type activation struct {
	name string
	in   any
	via  string
	idx  int
}

// plan 是一次 Run 在进入时捕获的执行输入：节点集合、边声明表与输出节点名。
// 调度器只读它，因此 Run 开始之后对 builder 的任何结构修改都不能推翻 Validate
// 在构建期得出的结论，也不能改写本次路由。
type plan struct {
	nodes    map[string]Node
	policies map[string]nodePolicy
	edges    []edgeDecl
	output   string
}

// capturePlan 在 Validate 通过之后复制拓扑。复制是必要的：不复制则调度器读到的是
// 活对象，节点内一次 AddEdge 就能绕过构建期校验（悬空端点或空谓词在执行期崩溃）。
func (g *Graph) capturePlan() plan {
	nodes := make(map[string]Node, len(g.nodes))
	for name, n := range g.nodes {
		nodes[name] = n
	}
	policies := make(map[string]nodePolicy, len(g.policies))
	for name, pol := range g.policies {
		policies[name] = pol
	}
	edges := make([]edgeDecl, len(g.edges))
	copy(edges, g.edges)
	return plan{nodes: nodes, policies: policies, edges: edges, output: g.output}
}

// scheduler 实现 §3.3 的单消费者零锁模型：
// 生产者 goroutine 只执行节点并向 queue 发送；消费者（调用 Run 的 goroutine）
// 是 queue 的唯一读者，也是 pending/running/result/steps 的唯一写者，因此这些
// 状态不需要任何锁。
type scheduler struct {
	plan      plan
	ctx       context.Context
	queue     chan queueItem
	done      chan struct{}
	result    *Result
	pending   []activation
	running   int
	steps     int // 本次 Run 已派发的激活数，只由消费者写
	stepLimit int // 进入 Run 时从 builder 取一次值，之后改图不能改写本次预算
	// maxConcurrency 是同时在途激活数的上限，0 或负数表示不限（母约 §3.3:190 的
	// maxConcurrency > 0 前置守卫）。它同样只在进入 Run 时取一次值。
	maxConcurrency int
	// durability/checkpointer 在进入 Run 时从 builder 取一次值（stepLimit/maxConcurrency
	// 同一套「取一次值」不变量），提交侧状态只由消费者读写（Async 冲刷器仅共享 commits 通道）。
	durability   Durability
	checkpointer Checkpointer
	seq          int
	exitQueue    []Checkpoint
	commits      chan Checkpoint
	flushDone    chan error
	// waits/feeds 描述本次 Run 的汇聚屏障（每个目标等待的去重前驱集合及其反向索引），
	// joinCollected 收集已经到达的前驱输出。三者与 pending 一样只由唯一消费者读写，
	// 因此同样不需要锁（票面 §3.3）。
	waits         map[string]map[string]bool
	feeds         map[string][]string
	joinCollected map[string]map[string]any
	// send 屏障（G5/AddJoinSend）：pending 是每个 source 名下在途 Send 激活数，
	// collected 按 source 聚合各激活的输出（下标 = 派发序号），fired 是「本轮已激活过」
	// 的守卫（source 计数归零激活一次后置位，再入正时复位）。与 pending 一样只由唯一
	// 消费者读写，零锁不破。
	sendPending    map[string]int
	sendCollected  map[string][]any
	sendFired      map[string]bool
	sendSourcesOf  map[string]map[string]bool
	sendTargetsOf  map[string][]string
	sendTargetList []string
	// HITL 挂起态（母约 §9.2）：挂起后派发冻结、在途排干收账；只由消费者读写。
	suspended      bool
	waitings       []WaitingNode
	suspendSinkErr error
}

// Run 从入口节点驱动图执行到收敛。
//
// 本次 Run 执行的是进入 Run 时那份已通过 Validate 的拓扑快照；Run 返回之后再改图
// 会影响下一次 Run，但不会影响已经开始的这一次。
func (g *Graph) Run(ctx context.Context, in any) (*Result, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	s := g.newScheduler(ctx, 0, 0)
	s.pending = []activation{{name: g.entry, in: in}}
	s.beginCommits()
	res, runErr := s.consume()
	if susp, _ := suspensionFrom(runErr); susp != nil {
		g.pending = susp
	}
	return s.finishCommits(res, runErr)
}

// consume 是可变状态的唯一写入者，也是取消被咨询的位置：select 的取消分支、取到队列项之后，
// 以及它调用的 dispatch，各问一次，因此两条交出错误结论的路径都不会绕过调用方的取消。
// running == 0 那条成功路径刻意不问：切片 17 量不到那个窗口（0/300），已登记为不带断言的已知取舍。
//
// HITL：中断载体在 item.err 里被第一个辨认——挂起后派发冻结（不再启动新节点），
// 在途节点排干：正常完成照常收账提交，排干期的真实错误照旧整单失败（错误优先于挂起）。
func (s *scheduler) consume() (*Result, error) {
	defer close(s.done)
	for {
		if !s.suspended {
			if err := s.dispatch(); err != nil {
				return nil, err
			}
		}
		if s.running == 0 {
			if s.suspended {
				return nil, s.suspensionError()
			}
			sort.Strings(s.result.completed)
			return s.result, nil
		}
		select {
		case <-s.ctx.Done():
			return nil, s.ctx.Err()
		case item := <-s.queue:
			s.running--
			// 先问取消，再决定手上这一项算不算结论：select 的随机挑选让「调用方已经 cancel」与
			// 「节点交出错误」同时成立，此时票面 :68 要的仍是取消语义。
			if err := s.ctx.Err(); err != nil {
				return nil, err
			}
			if item.err != nil {
				s.plan.policies[item.name].trace.emit(NodeEvent{
					Node: item.name, Attempt: item.attempt, Err: item.err, In: item.in,
				})
				if sig, ok := interruptFrom(item.err); ok {
					if err := s.suspend(item, sig); err != nil {
						return nil, err
					}
					continue
				}
				return nil, item.err
			}
			s.complete(item)
			// 提交时机全在消费者侧：Sync 的「下一步开始前」就落在这里——complete 之后、
			// 下一轮 dispatch 之前；Exit/Async 的落位在 record 内部各归其档。
			if err := s.record(item); err != nil {
				return nil, err
			}
			// Send 记账同样全在消费者侧：派发该完成项自己的 sends、并把「它是某 source
			// 的 Send 激活」这件事记进屏障，归零即激活 AddJoinSend 目标。
			if err := s.absorbSends(item); err != nil {
				return nil, err
			}
		}
	}
}

// suspend 把一个中断请求记入挂起账：等待项入册、挂起条目按档位提交。空 InterruptID
// 与并行挂起项里的重复 InterruptID 都被拒绝——它们在 Resume 的响应路由里不可区分。
func (s *scheduler) suspend(item queueItem, sig *interruptSignal) error {
	i := sig.interrupt
	if i.InterruptID == "" {
		return fmt.Errorf("graph: node %q requested an interrupt with an empty InterruptID", item.name)
	}
	for _, w := range s.waitings {
		if w.Interrupt.InterruptID == i.InterruptID {
			return fmt.Errorf("graph: duplicate InterruptID %q among concurrently pending interrupts (nodes %q and %q)", i.InterruptID, w.Node, item.name)
		}
	}
	s.suspended = true
	s.waitings = append(s.waitings, WaitingNode{Node: item.name, In: item.in, Interrupt: i})
	if s.checkpointer != nil {
		s.seq++
		if err := s.commit(Checkpoint{Seq: s.seq, Node: item.name, Kind: EntryInterrupt, Input: item.in, Interrupt: &i}); err != nil && s.suspendSinkErr == nil {
			// 落盘失败不整单吞掉：退出时并入挂起错误，「挂起了」与「没落盘」都要可见。
			s.suspendSinkErr = err
		}
	}
	return nil
}

// suspensionError 把挂起账组装成交出产物：Completed 按字典序，sink 首错经 errors.Join
// 并入（errors.Is/errors.As 的辨认不受影响）。
func (s *scheduler) suspensionError() error {
	susp := &Suspension{
		Interrupts: s.waitings,
		Values:     s.result.values,
		Completed:  s.result.completed,
		seq:        s.seq,
		steps:      s.steps,
	}
	sort.Strings(susp.Completed)
	if s.suspendSinkErr != nil {
		return errors.Join(susp, s.suspendSinkErr)
	}
	return susp
}

// dispatch 把等待队列按声明顺序派发，打满执行槽就停下等下一次完成，每派发一次激活消耗一步预算。
//
// 计步发生在「把激活交给节点之前」：一批并发后继里若有一个会让步数越过上限，它必须
// 根本不被启动。按「已完成数」计步的实现会先把整批放出去，多跑出来的那些副作用是调用方
// 收不回来的 —— 安全阀的价值恰好在于「不再往前走」，而不是「走了再报错」。
//
// 并发上限同理停在派发之前：越界的激活留在 pending 头部，既不消耗槽位也不消耗步数预算。
// 这里不需要任何锁——running/pending/steps 只由唯一消费者读写，而「停在这里」本身就是
// 消费者让出队列的方式。判据写在循环头而不是循环之外：每个完成事件后都会重新进入这里，
// 只在首轮判上限的实现会在第一个 completion 之后把整批一次放出。
//
// 取消判据写在循环头、且在把激活交给节点之前：消费者忙在这一个循环里时不会回到 select，
// 于是派发窗口里的取消只有这里能看见（变异 τ：只拿掉这一条 → D1 判红）。它与预算判据的先后
// 不单独可观察（变异 ο：互换次序 → 整族判绿），所以这里的不变量是「两条都在交给节点之前问到」，
// 不是它们的次序。
func (s *scheduler) dispatch() error {
	for len(s.pending) > 0 {
		if err := s.ctx.Err(); err != nil {
			return err
		}
		if s.steps >= s.stepLimit {
			return fmt.Errorf("graph: step limit %d reached before the graph converged: %w", s.stepLimit, ErrStepLimitExceeded)
		}
		if s.maxConcurrency > 0 && s.running >= s.maxConcurrency {
			return nil
		}
		act := s.pending[0]
		s.pending = s.pending[1:]
		s.steps++
		s.spawn(act)
	}
	return nil
}

// spawn 占用一个执行槽并建立该激活的生产者 goroutine。
func (s *scheduler) spawn(act activation) {
	s.running++
	go func() {
		out, sends, err, attempts := s.runNode(act)
		item := queueItem{name: act.name, out: out, sends: sends, err: err, in: act.in, attempt: attempts, via: act.via, idx: act.idx}
		select {
		case s.queue <- item:
		case <-s.done:
		}
	}()
}

// runNode 在生产者 goroutine 里执行一个激活的策略包络，并把节点体内的任意 panic
// 转成错误（panic 恢复保持在节点帧之上，节点自己的 defer 先跑完）。
//
// 包络次序：缓存查询（命中则跳过执行）→ 至多 attempts 次带期限的尝试 → 成功才写缓存。
// 重试与期限都在这条生产者 goroutine 内生效，不产生新的激活、不消耗步数预算
// （步数计的是 dispatch 的激活数，切片 22 契约 D5）；退避等待随时让位给调用方取消。
//
// 结果仍只经由 queueItem 交给唯一消费者，崩溃的 goroutine 不写任何 scheduler 字段。
func (s *scheduler) runNode(act activation) (out any, sends []Send, err error, attempts int) {
	defer func() {
		if v := recover(); v != nil {
			out, sends, err = nil, nil, fmt.Errorf("graph: node %q panicked while running: %v", act.name, v)
		}
	}()
	pol := s.plan.policies[act.name]
	if pol.cache != nil && pol.cache.Store != nil {
		if v, ok, gerr := pol.cache.Store.GetAny(s.ctx, pol.cache.keyFor(act.in)); gerr == nil && ok {
			return v, nil, nil, 1
		}
	}
	max := pol.retry.attempts()
	// PerAttempt=false 的期限是整个节点的预算（含全部重试尝试）：循环外挂一次；
	// PerAttempt=true 则每次尝试各自新建期限（D8 钉的形状）。
	base := s.ctx
	if pol.timeout != nil && !pol.timeout.PerAttempt {
		if d, ok := pol.timeout.deadline(); ok {
			var cancel context.CancelFunc
			base, cancel = context.WithTimeout(s.ctx, d)
			defer cancel()
		}
	}
	for attempt := 1; ; attempt++ {
		attempts = attempt
		out, sends, err = s.tryNode(pol, act, base)
		// 中断是终态：重试不重发、退避不占用——「等人类」不是可重试的失败（切片 22 语义）。
		if _, isInterrupt := interruptFrom(err); isInterrupt {
			break
		}
		if err == nil || attempt >= max || !pol.retry.should(err) {
			break
		}
		if d := pol.retry.delayFor(attempt); d > 0 {
			select {
			case <-s.ctx.Done():
				return nil, nil, s.ctx.Err(), attempt
			case <-time.After(d):
			}
		}
	}
	if err == nil && pol.cache != nil && pol.cache.Store != nil {
		_ = pol.cache.Store.SetAny(s.ctx, pol.cache.keyFor(act.in), out, pol.cache.TTL)
	}
	return out, sends, err, attempts
}

// tryNode 执行一次节点尝试：PerAttempt=true 时给这次尝试挂上从 base 派生的新期限，
// 否则直接沿用 base（PerAttempt=false 的期限已在 runNode 循环外挂好）。
//
// Sender 断言在这里发生（每次激活一次）：断言成功走 SendRun、其 sends 随完成事件交给
// 消费者派发；断言失败走 Node.Run、sends 为 nil——普通节点零改变（G5 契约钉的形状）。
func (s *scheduler) tryNode(pol nodePolicy, act activation, base context.Context) (any, []Send, error) {
	ctx := base
	if pol.timeout != nil && pol.timeout.PerAttempt {
		if d, ok := pol.timeout.deadline(); ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(base, d)
			defer cancel()
		}
	}
	if snd, ok := s.plan.nodes[act.name].(Sender); ok {
		return snd.SendRun(ctx, act.in)
	}
	out, err := s.plan.nodes[act.name].Run(ctx, act.in)
	return out, nil, err
}

// complete 记录节点输出并激活其后继：普通边后继由 successors 给出，汇聚屏障的激活由
// joinReady 给出。两路都只 append 到 pending，仍是消费者独占的写。
func (s *scheduler) complete(item queueItem) {
	s.plan.policies[item.name].trace.emit(NodeEvent{
		Node: item.name, Attempt: item.attempt, Out: item.out, In: item.in,
	})
	s.result.values[item.name] = item.out
	s.result.completed = append(s.result.completed, item.name)
	if item.name == s.plan.output {
		s.result.output = item.out
	}
	s.pending = append(s.pending, s.successors(item.name, item.out)...)
	s.pending = append(s.pending, s.joinReady(item.name, item.out)...)
}

// joinReady 是 §3.4 规则 3 的落点：前驱每完成一个，就把它的输出按前驱名记进该目标的聚合
// 输入；只有收集到的键集合覆盖了目标声明的全部前驱时，才把目标激活一次（票面 :61）。
//
// 激活时交出的输入就是那个 map 本身，不是「最后一个前驱的输出」——汇聚的意义在于让节点
// 看得见全部并行的分支，否则 AddJoin 与无条件边没有区别。
//
// 「凑齐」判据用键集合覆盖而不是计数归零：计数器要把同名的重复声明与前驱重跑都算对，
// 而集合本身就是公共面看到的那个形状（Result 里 join 的输入即它）。
func (s *scheduler) joinReady(name string, out any) []activation {
	var ready []activation
	for _, target := range s.feeds[name] {
		collected := s.joinCollected[target]
		if collected == nil {
			collected = map[string]any{}
			s.joinCollected[target] = collected
		}
		collected[name] = out
		if len(collected) < len(s.waits[target]) {
			continue
		}
		ready = append(ready, activation{name: target, in: collected})
	}
	return ready
}

// successors 是 §3.4 路由核心的落点：给定刚完成的节点与其输出，返回应被激活的后继。
// 无条件边与命中的条件边一起构成「具体边」；只有具体边一条都没命中时才走兜底边。
// 缺少谓词的条件边不会到达这里：Validate 在构建期拒绝这类声明，而调度器只读进入 Run 时
// 捕获的 plan，运行中改图无法把新声明送进这里。因此条件边在此处无需再判 nil，重复判断
// 正是静默丢边的来源。
//
// 汇聚边在这里没有对应分支不是漏判：join 声明的激活时刻属于屏障（joinReady），把它在这里
// 当成无条件边激活会绕过 §3.4 规则 3 的「等全部前驱」。这个分工由 joinBarriers 只收 edgeJoin、
// 这里只收其余三种共同担保；一侧新增种类时另一侧要同时表态。
func (s *scheduler) successors(name string, out any) []activation {
	var concrete, fallback []activation
	for _, e := range s.plan.edges {
		if e.from != name {
			continue
		}
		switch e.kind {
		case edgeUnconditional:
			concrete = append(concrete, activation{name: e.to, in: out})
		case edgeConditional:
			if e.p(out) {
				concrete = append(concrete, activation{name: e.to, in: out})
			}
		case edgeDefault:
			fallback = append(fallback, activation{name: e.to, in: out})
		}
	}
	if len(concrete) > 0 {
		return concrete
	}
	return fallback
}

// absorbSends 是 G5 Send 记账的落点，两个方向都在唯一消费者内串行完成：
//
// 方向一：该完成项本身是某个 AddJoinSend source 的第 idx 个 Send 激活——把输出按
// 派发序号记进聚合器并递减该 source 的在途计数；计数归零时检查全部就绪的 target。
//
// 方向二：该完成项（作为 Sender）派发了自己的 sends——先整体校验目标名（任何一个
// 未注册即失败，不做部分派发），再按声明顺序赋予派发序号、入队激活并递增对应 source
// 的在途计数；计数由 0 转正时复位其 target 的「已激活」守卫（下一波重新可激活）。
//
// 本轮从未有过 Send 指名的 source，计数恒为 0、没有任何完成事件触发检查——其 target
// 不被激活，Run 沿 R19-Q1 的既有形状收场（G5 契约钉死的零派发语义）。
func (s *scheduler) absorbSends(item queueItem) error {
	if item.via != "" {
		s.sendCollected[item.via][item.idx] = item.out
		s.sendPending[item.via]--
		if s.sendPending[item.via] == 0 {
			s.pending = append(s.pending, s.joinSendReady()...)
		}
	}
	if len(item.sends) > 0 {
		for _, snd := range item.sends {
			if _, ok := s.plan.nodes[snd.Node]; !ok {
				return fmt.Errorf("graph: node %q sent to unregistered node %q", item.name, snd.Node)
			}
		}
		for _, snd := range item.sends {
			idx := len(s.sendCollected[snd.Node])
			s.sendCollected[snd.Node] = append(s.sendCollected[snd.Node], nil)
			s.sendPending[snd.Node]++
			if s.sendPending[snd.Node] == 1 {
				for _, t := range s.sendTargetsOf[snd.Node] {
					s.sendFired[t] = false
				}
			}
			s.pending = append(s.pending, activation{name: snd.Node, in: snd.In, via: snd.Node, idx: idx})
		}
	}
	return nil
}

// joinSendReady 在某个 source 的计数归零时被调：对每个「全部 source 计数同时为零且
// 本波尚未激活过」的 target，以 map[string][]any（键 = source 名，值为按派发序号排列的
// 各激活输出）激活一次；聚合 map 与内层切片都是激活时新建/移交的，不再与调度器状态
// 同源（S19-STD-1 的别名形状在这里结构性不存在）。激活后聚合器复位，下一波从零积累。
func (s *scheduler) joinSendReady() []activation {
	var ready []activation
	var drained []string
	for _, target := range s.sendTargetList {
		if s.sendFired[target] {
			continue
		}
		sources := s.sendSourcesOf[target]
		settled := true
		for src := range sources {
			if s.sendPending[src] > 0 {
				settled = false
				break
			}
		}
		if !settled {
			continue
		}
		agg := make(map[string][]any, len(sources))
		for src := range sources {
			agg[src] = s.sendCollected[src]
		}
		s.sendFired[target] = true
		ready = append(ready, activation{name: target, in: agg})
		drained = append(drained, target)
	}
	// 复位放在全部就绪 target 的聚合都取走之后：同一次归零触发的多个 target
	// 看到的是同一波输出，不能被先处理的 target 清空。
	for _, target := range drained {
		for src := range s.sendSourcesOf[target] {
			s.sendCollected[src] = nil
		}
	}
	return ready
}
