package workflow

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// 本文件是 G10 OPT-β 的编译器：把 []Step 声明编译为一张 *graph.Graph 并跑在图内核上。
// 公共 API 零变化——编译只发生在 executeSteps 内部，会话/历史/持久化/取消/metrics
// 仍由 Workflow.Run 负责。
//
// 边上流动的唯一值是 carrier 信封：ec 是当前执行上下文（引用语义，与旧内核的指针
// 替换穿线同构），take/route/cont 只由控制节点写入、只被它们的出边谓词读取。
// 别名隔离纪律（摸底材料 [P6] DATA RACE 的解）：扇出边交给每个激活的是同一个信封，
// 但 Condition/Router 的互补谓词保证恰一支激活；Parallel 是唯一并发扇出，分支头节点
// 先克隆私有 ExecutionContext 再执行（照抄 parallel.go 的克隆语义），信封与源 EC 在
// split 返回之后没有任何写入者。pkg/hno/graph 为此零改动。

// maxCompileDepth 是复合节点展开的深度上限，兜底「不可比较的 Node 实现的自引用环」
// （可比较类型由 cycleCheck 按身份精确拒绝）。
const maxCompileDepth = 10000

// maxLoopBudget 是单 Loop 允许声明迭代数的上限。旧内核对该值没有上限（用户声明
// 十亿次迭代就会真的跑十亿次）；编译期预算要把它乘进 WithStepLimit，溢出的上界
// 既算不出来也不该静默截断，因此显式拒绝。
const maxLoopBudget = 1_000_000_000

// endNodeName 是终止节点的图内名字：恒等穿出，SetOutput 指向它，成功路径的
// Result.Output() 即最终 ExecutionContext。
const endNodeName = "end"

// carrier 是编译图内边上流动的信封。
type carrier struct {
	ec    *ExecutionContext
	take  bool   // Condition 决策判定（仅其出边谓词读取）
	route string // Router 选中路由键（仅其出边谓词读取）
	cont  bool   // Loop 控制器继续判定（仅其出边谓词读取）
}

// ownerKind 区分错误归属链上的复合种类。
type ownerKind int

const (
	ownerParallel ownerKind = iota
	ownerLoop
)

// owner 是一个编译节点的复合归属：分支体内的节点失败时，错误按归属链由内向外
// 补上 parallel/loop 前缀（与旧内核里复合节点 Execute 的包装次序逐字对齐）。
type owner struct {
	kind   ownerKind
	loopID string
	ctl    string // Loop 控制器节点名，用于读在飞迭代号
}

// execLedger 是消费侧台账：WithTrace 的 Hook 由图调度器的唯一消费者串行发射，
// 因此 executed（已完成主干步，执行序）、lastEC（最近成功穿出的上下文）、failing
// （失败节点）都不需要额外的同步；loopIter 由控制器 goroutine 写、消费者读，用
// 互斥锁保护。账本刻意不读 Result.Completed()——那是字典序，不是执行序。
//
// watch/stopSched 承载取消的边界语义（复刻旧 run.Loop「取消在下一步边界生效」）：
// 调度器拿到的是从 watch 派生的隔离上下文（值保留、取消隔离），每次完成事件记账后
// 若发现调用方 ctx 已取消才 stopSched()——刚完成的这步因此一定被记进台账，恰如旧
// 内核里在飞步总是先跑完再判取消。
type execLedger struct {
	mu         sync.Mutex
	seed       *ExecutionContext
	lastEC     *ExecutionContext
	executed   []string
	recorded   map[string]bool
	failing    string
	loopIter   map[string]int
	hookEvents map[string]run.Events
	watch      context.Context
	stopSched  context.CancelFunc
}

func newExecLedger(seed *ExecutionContext) *execLedger {
	return &execLedger{
		seed:       seed,
		lastEC:     seed,
		recorded:   map[string]bool{},
		loopIter:   map[string]int{},
		hookEvents: map[string]run.Events{},
	}
}

// onEvent 处理一条节点级事件。成功事件推进 lastEC 与主干步台账；失败事件只在
// 首次出现时记下失败节点（消费侧先收先报，与旧 Parallel 的 errors[0] 同语义）。
// 调用方取消只在主干步边界（收集器完成）停调度器——与旧内核「在飞步跑完、取消
// 在下一步边界生效」同粒度，刚完成的这步因此一定入账。
func (l *execLedger) onEvent(spineID string, ev graph.NodeEvent) {
	l.mu.Lock()
	if ev.Err != nil {
		if l.failing == "" {
			l.failing = ev.Node
		}
	} else {
		car, _ := ev.Out.(*carrier)
		if car != nil {
			l.lastEC = car.ec
		}
		if spineID != "" && !l.recorded[spineID] {
			l.recorded[spineID] = true
			l.executed = append(l.executed, spineID)
			// 事件值在完成钩子里就地捕获：失败/取消路径上可能还有在飞 goroutine 在写
			// 共享 Data map，返回后再读会有竞态。成功路径仍从最终 EC 重查（见 collectEvents）。
			var ec *ExecutionContext
			if car != nil {
				ec = car.ec
			}
			if stepEvents := extractStepEvents(ec, spineID); len(stepEvents) > 0 {
				l.hookEvents[spineID] = stepEvents
			}
		}
	}
	watch, stop := l.watch, l.stopSched
	l.mu.Unlock()
	if spineID != "" && watch != nil && stop != nil && watch.Err() != nil {
		stop()
	}
}

func (l *execLedger) markLoopIter(ctl string, iteration int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.loopIter[ctl] = iteration
}

func (l *execLedger) loopIteration(ctl string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.loopIter[ctl]
}

// snapshot 返回台账读数副本。
func (l *execLedger) snapshot() (executed []string, lastEC *ExecutionContext, failing string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	executed = append([]string{}, l.executed...)
	return executed, l.lastEC, l.failing
}

// collectEvents 复刻旧 collectStepEvents：成功路径从最终 EC 按执行序反查每主干步
// 的 step_<id>_events。失败路径改用钩子快照（见 onEvent），避免与在飞写者竞态。
func (l *execLedger) collectEvents(final *ExecutionContext, failed bool) map[string]run.Events {
	l.mu.Lock()
	executed := append([]string{}, l.executed...)
	hooked := make(map[string]run.Events, len(l.hookEvents))
	for k, v := range l.hookEvents {
		hooked[k] = v
	}
	l.mu.Unlock()

	events := make(map[string]run.Events)
	if failed {
		for _, id := range executed {
			if ev := hooked[id]; len(ev) > 0 {
				events[id] = ev
			}
		}
		return events
	}
	for _, id := range executed {
		if stepEvents := extractStepEvents(final, id); len(stepEvents) > 0 {
			events[id] = stepEvents
		}
	}
	return events
}

// compiledGraph 是一次 executeSteps 的编译产物。每次调用都产生全新实例，控制器的
// 迭代计数等每 Run 状态天然不跨 Run 泄漏。
type compiledGraph struct {
	g      *graph.Graph
	ledger *execLedger
	owners map[string][]owner
}

// wrapError 复刻旧内核的错误聚合：主干包 types.NewError(ErrCodeUnknown,
// "step <最后成功主干步> failed", inner)，复合前缀按归属链由内向外补齐（owners
// 按嵌套序存储为外层在前，包装时倒序迭代 = 最内层前缀最先包上）。
func (c *compiledGraph) wrapError(runErr error, lastSpineID string) error {
	inner := runErr
	_, _, failing := c.ledger.snapshot()
	if failing != "" {
		chain := c.owners[failing]
		for i := len(chain) - 1; i >= 0; i-- {
			switch o := chain[i]; o.kind {
			case ownerParallel:
				inner = fmt.Errorf("parallel execution failed: %w", inner)
			case ownerLoop:
				inner = fmt.Errorf("loop %s iteration %d failed: %w", o.loopID, c.ledger.loopIteration(o.ctl), inner)
			}
		}
	}
	return types.NewError(types.ErrCodeUnknown, fmt.Sprintf("step %s failed", lastSpineID), inner)
}

// exitEdge 描述一个子图的完成出口：from 节点完成后按 cond（nil=无条件）激活调用方
// 指定的下一目标。
type exitEdge struct {
	from string
	cond graph.Predicate
}

// subGraph 是一次 compileNode 的产物：入口名 + 全部完成出口。
type subGraph struct {
	entry string
	exits []exitEdge
}

// compileCtx 持有编译中的图与台账。nodeCtx 是调用方的原始 ctx：叶适配器用它执行
// 节点（在飞步能像旧内核一样感知取消并自行中止），而调度器本身拿的是隔离上下文。
type compileCtx struct {
	g       *graph.Graph
	ledger  *execLedger
	owners  map[string][]owner
	nodeCtx context.Context
}

// traceOpt 给每个节点挂消费侧台账钩子。spineID 非空表示该节点是某主干步的入口，
// 完成时记一步「已执行」。
func (c *compileCtx) traceOpt(spineID string) graph.NodeOption {
	led := c.ledger
	return graph.WithTrace(graph.TraceConfig{
		Enabled: true,
		Hook:    func(ev graph.NodeEvent) { led.onEvent(spineID, ev) },
	})
}

// addNode 注册一个适配节点并登记归属链。
func (c *compileCtx) addNode(name, spineID string, fn func(context.Context, any) (any, error), os []owner) {
	c.g.AddNode(graph.NodeFunc(name, fn), c.traceOpt(spineID))
	if len(os) > 0 {
		c.owners[name] = os
	}
}

// wire 把子图出口接到下一目标，保留出口自带的条件谓词。
func (c *compileCtx) wire(exits []exitEdge, to string) {
	for _, e := range exits {
		if e.cond != nil {
			c.g.AddConditional(e.from, to, e.cond)
		} else {
			c.g.AddEdge(e.from, to)
		}
	}
}

// withOwner 追加一个归属（拷贝底层，避免共享底数组互踩）。
func withOwner(os []owner, o owner) []owner {
	out := make([]owner, 0, len(os)+1)
	out = append(out, os...)
	return append(out, o)
}

// sameNode 按底层指针/可等值性判断两个 Node 是否同一实例，避免把不可比较类型
// 直接当 map 键用（会 panic）。
func sameNode(a, b Node) bool {
	ra, rb := reflect.ValueOf(a), reflect.ValueOf(b)
	if ra.Type() != rb.Type() {
		return false
	}
	switch ra.Kind() {
	case reflect.Pointer, reflect.Chan, reflect.Func, reflect.Map, reflect.Slice, reflect.UnsafePointer:
		return ra.Pointer() == rb.Pointer()
	}
	if ra.Type().Comparable() {
		return a == b
	}
	return false
}

func nodeInPath(n Node, path []Node) bool {
	for _, p := range path {
		if sameNode(n, p) {
			return true
		}
	}
	return false
}

// compileNode 把一个 workflow.Node 编译为子图。复合节点按类型断言分派（旧内核
// 也只认具体类型的 Execute 实现；用户自定义 GetType 的不透明节点走叶适配）。
// 主干步的「已执行」记账不在入口——复合步要等整个子图成功穿出才算执行（与旧
// 内核「Composite.Execute 成功返回才进 history」同粒度），记账点由 compileSteps
// 在每步出口接的收集器节点承担。
func (c *compileCtx) compileNode(prefix string, n Node, os []owner, path []Node) (subGraph, error) {
	if n == nil {
		return subGraph{}, fmt.Errorf("workflow: step at %q is nil", prefix)
	}
	if len(path) >= maxCompileDepth {
		return subGraph{}, fmt.Errorf("workflow: node nesting deeper than %d at %q", maxCompileDepth, prefix)
	}
	switch t := n.(type) {
	case *Condition:
		return c.compileCondition(prefix, t, os, path)
	case *Router:
		return c.compileRouter(prefix, t, os, path)
	case *Loop:
		return c.compileLoop(prefix, t, os, path)
	case *Parallel:
		return c.compileParallel(prefix, t, os, path)
	default:
		name := prefix
		nodeCtx := c.nodeCtx
		c.addNode(name, "", func(ctx context.Context, in any) (any, error) {
			car := in.(*carrier)
			out, err := n.Execute(nodeCtx, car.ec)
			if err != nil {
				return nil, err
			}
			return &carrier{ec: out}, nil
		}, os)
		return subGraph{entry: name, exits: []exitEdge{{from: name}}}, nil
	}
}

// compileCondition：决策节点求判定并写 condition_<id>_result，两条互补谓词的条件边
// 各接一个分支子图，双侧出口汇给调用方。nil 分支 = 恒等头（旧语义 no-op）。
func (c *compileCtx) compileCondition(prefix string, cond *Condition, os []owner, path []Node) (subGraph, error) {
	if nodeInPath(cond, path) {
		return subGraph{}, fmt.Errorf("workflow: cyclic node reference at %q", prefix)
	}
	path = append(path, cond)
	name := prefix + ".cond"
	c.addNode(name, "", func(ctx context.Context, in any) (any, error) {
		car := in.(*carrier)
		take := cond.Condition(car.ec)
		car.ec.Set(fmt.Sprintf("condition_%s_result", cond.ID), take)
		return &carrier{ec: car.ec, take: take}, nil
	}, os)
	tb, err := c.compileBranch(prefix+".t", cond.TrueNode, os, path)
	if err != nil {
		return subGraph{}, err
	}
	c.g.AddConditional(name, tb.entry, func(out any) bool { return out.(*carrier).take })
	fb, err := c.compileBranch(prefix+".f", cond.FalseNode, os, path)
	if err != nil {
		return subGraph{}, err
	}
	c.g.AddConditional(name, fb.entry, func(out any) bool { return !out.(*carrier).take })
	exits := make([]exitEdge, 0, len(tb.exits)+len(fb.exits))
	exits = append(exits, tb.exits...)
	exits = append(exits, fb.exits...)
	return subGraph{entry: name, exits: exits}, nil
}

// compileBranch 编译一个分支：nil 即恒等头，否则递归 compileNode。
func (c *compileCtx) compileBranch(prefix string, n Node, os []owner, path []Node) (subGraph, error) {
	if n == nil {
		name := prefix + ".nop"
		c.addNode(name, "", func(ctx context.Context, in any) (any, error) {
			return &carrier{ec: in.(*carrier).ec}, nil
		}, nil)
		return subGraph{entry: name, exits: []exitEdge{{from: name}}}, nil
	}
	return c.compileNode(prefix, n, os, path)
}

// compileRouter：决策节点求路由键并写 router_<id>_selected，每条路由一条键匹配的
// 条件边；AddDefault 挂未命中错误节点，逐字复刻旧 router.go 的报错而不静默兜底。
func (c *compileCtx) compileRouter(prefix string, r *Router, os []owner, path []Node) (subGraph, error) {
	if nodeInPath(r, path) {
		return subGraph{}, fmt.Errorf("workflow: cyclic node reference at %q", prefix)
	}
	path = append(path, r)
	name := prefix + ".rt"
	c.addNode(name, "", func(ctx context.Context, in any) (any, error) {
		car := in.(*carrier)
		key := r.Router(car.ec)
		car.ec.Set(fmt.Sprintf("router_%s_selected", r.ID), key)
		return &carrier{ec: car.ec, route: key}, nil
	}, os)
	keys := make([]string, 0, len(r.Routes))
	for k := range r.Routes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var exits []exitEdge
	for i, k := range keys {
		rb, err := c.compileBranch(fmt.Sprintf("%s.k%d", prefix, i), r.Routes[k], os, path)
		if err != nil {
			return subGraph{}, err
		}
		key := k
		c.g.AddConditional(name, rb.entry, func(out any) bool { return out.(*carrier).route == key })
		exits = append(exits, rb.exits...)
	}
	miss := prefix + ".rt.miss"
	c.addNode(miss, "", func(ctx context.Context, in any) (any, error) {
		car := in.(*carrier)
		return nil, fmt.Errorf("router %s: route '%s' not found", r.ID, car.route)
	}, nil)
	c.g.AddDefault(name, miss)
	return subGraph{entry: name, exits: exits}, nil
}

// compileLoop：控制器节点自数激活次数作为迭代号（同图每次 Run 新编译、环内激活
// 严格串行，闭包计数无并发），continue 条件边进体、体尾无条件回边、done 条件边
// 出口。done 时在穿出 EC 写 loop_<id>_iterations（含 0 次迭代）。
func (c *compileCtx) compileLoop(prefix string, l *Loop, os []owner, path []Node) (subGraph, error) {
	if nodeInPath(l, path) {
		return subGraph{}, fmt.Errorf("workflow: cyclic node reference at %q", prefix)
	}
	path = append(path, l)
	if l.MaxIteration > maxLoopBudget {
		return subGraph{}, fmt.Errorf("workflow: loop %s MaxIteration %d exceeds the compile-time budget cap %d", l.ID, l.MaxIteration, maxLoopBudget)
	}
	name := prefix + ".lp"
	led := c.ledger
	iteration := 0
	c.addNode(name, "", func(ctx context.Context, in any) (any, error) {
		car := in.(*carrier)
		cur := iteration
		iteration++
		if cur >= l.MaxIteration || !l.Condition(car.ec, cur) {
			car.ec.Set(fmt.Sprintf("loop_%s_iterations", l.ID), cur)
			return &carrier{ec: car.ec, cont: false}, nil
		}
		led.markLoopIter(name, cur)
		return &carrier{ec: car.ec, cont: true}, nil
	}, os)
	body, err := c.compileNode(prefix+".b", l.Body, withOwner(os, owner{kind: ownerLoop, loopID: l.ID, ctl: name}), path)
	if err != nil {
		return subGraph{}, err
	}
	c.g.AddConditional(name, body.entry, func(out any) bool { return out.(*carrier).cont })
	for _, e := range body.exits {
		c.g.AddEdge(e.from, name)
	}
	return subGraph{entry: name, exits: []exitEdge{{from: name, cond: func(out any) bool { return !out.(*carrier).cont }}}}, nil
}

// compileParallel：split 捕获源 EC，无条件扇出到每分支的克隆头（[P6] 竞态的解——
// 分支头各自克隆私有 EC，信封与源 EC 扇出后无人再写），分支体经尾汇点用 AddJoin
// 等全前驱（split + 全部尾汇点）激活 merge，merge 照抄旧 parallel.go 的合并语义。
func (c *compileCtx) compileParallel(prefix string, p *Parallel, os []owner, path []Node) (subGraph, error) {
	if nodeInPath(p, path) {
		return subGraph{}, fmt.Errorf("workflow: cyclic node reference at %q", prefix)
	}
	path = append(path, p)
	split := prefix + ".pa.s"
	c.addNode(split, "", func(ctx context.Context, in any) (any, error) {
		return &carrier{ec: in.(*carrier).ec}, nil
	}, os)
	merge := prefix + ".pa.m"
	joinFrom := []string{split}
	tails := make([]string, len(p.Nodes))
	parOwner := withOwner(os, owner{kind: ownerParallel})
	for i := range p.Nodes {
		head := fmt.Sprintf("%s.pa.c%d", prefix, i)
		c.addNode(head, "", func(ctx context.Context, in any) (any, error) {
			return &carrier{ec: cloneBranchEC(in.(*carrier).ec)}, nil
		}, nil)
		c.g.AddEdge(split, head)
		child := p.Nodes[i]
		br, err := c.compileNode(fmt.Sprintf("%s.pa.b%d", prefix, i), child, parOwner, path)
		if err != nil {
			return subGraph{}, err
		}
		c.g.AddEdge(head, br.entry)
		tail := fmt.Sprintf("%s.pa.t%d", prefix, i)
		c.addNode(tail, "", func(ctx context.Context, in any) (any, error) { return in, nil }, nil)
		c.wire(br.exits, tail)
		joinFrom = append(joinFrom, tail)
		tails[i] = tail
	}
	c.g.AddJoin(joinFrom, merge)
	c.addNode(merge, "", func(ctx context.Context, in any) (any, error) {
		m := in.(map[string]any)
		src := m[split].(*carrier).ec
		results := make([]*ExecutionContext, len(p.Nodes))
		for i := range p.Nodes {
			results[i] = m[tails[i]].(*carrier).ec
		}
		mergeParallelResults(p, src, results)
		return &carrier{ec: src}, nil
	}, os)
	return subGraph{entry: split, exits: []exitEdge{{from: merge}}}, nil
}

// cloneBranchEC 逐字复刻 parallel.go 的分支上下文克隆：SessionState 深拷贝（nil 则
// 新建）、Data/Metadata 逐键复制进新 map、Input/Output/SessionID/UserID 拷贝。
// WorkflowHistory 与 HistoryContext 不拷贝——旧代码就不拷贝，这是被钉住的保真。
func cloneBranchEC(src *ExecutionContext) *ExecutionContext {
	var ss *SessionState
	if src.SessionState != nil {
		ss = src.SessionState.Clone()
	} else {
		ss = NewSessionState()
	}
	branchCtx := &ExecutionContext{
		Input:        src.Input,
		Output:       src.Output,
		Data:         make(map[string]interface{}),
		Metadata:     make(map[string]interface{}),
		SessionState: ss,
		SessionID:    src.SessionID,
		UserID:       src.UserID,
	}
	for k, v := range src.Data {
		branchCtx.Data[k] = v
	}
	for k, v := range src.Metadata {
		branchCtx.Metadata[k] = v
	}
	return branchCtx
}

// mergeParallelResults 逐字复刻 parallel.go 的汇聚段：会话状态三方合并、
// parallel_<id>_branch_<i>_output 键、Data 前缀键、Output 取最后一个分支下标。
func mergeParallelResults(p *Parallel, src *ExecutionContext, results []*ExecutionContext) {
	modifiedSessionStates := make([]*SessionState, 0, len(results))
	for _, result := range results {
		if result != nil && result.SessionState != nil {
			modifiedSessionStates = append(modifiedSessionStates, result.SessionState)
		}
	}
	if len(modifiedSessionStates) > 0 {
		originalSessionState := src.SessionState
		if originalSessionState == nil {
			originalSessionState = NewSessionState()
		}
		src.SessionState = MergeParallelSessionStates(originalSessionState, modifiedSessionStates)
	}
	for i, result := range results {
		if result != nil {
			src.Set(fmt.Sprintf("parallel_%s_branch_%d_output", p.ID, i), result.Output)
			for k, v := range result.Data {
				src.Set(fmt.Sprintf("parallel_%s_branch_%d_%s", p.ID, i, k), v)
			}
		}
	}
	if len(results) > 0 && results[len(results)-1] != nil {
		src.Output = results[len(results)-1].Output
	}
}

// compileSteps 把 steps（已是 startIdx 之后的尾段）编译为一张完整图：每步的子图
// 出口先汇入该步的收集器节点（记账点 = 复合子图整体成功穿出，与旧内核「Execute
// 成功返回才进 history」同粒度），收集器之间无条件边链接，恒等终止节点收口作
// SetOutput。nodeCtx 是调用方原始 ctx，叶适配器编译期就把它闭包进节点执行
// （在飞步感知取消）；调度器上下文与取消边界由 executeSteps 装上（见 executor.go）。
func compileSteps(steps []Node, seed *ExecutionContext, nodeCtx context.Context) (*compiledGraph, error) {
	limit, err := structuralBound(steps)
	if err != nil {
		return nil, err
	}
	g := graph.New(graph.WithStepLimit(limit + 1)) // +1 终止节点
	cc := &compileCtx{g: g, ledger: newExecLedger(seed), owners: map[string][]owner{}, nodeCtx: nodeCtx}
	entry := ""
	prevCollector := ""
	for i, st := range steps {
		if st == nil {
			return nil, fmt.Errorf("workflow: step %d is nil", i)
		}
		sg, err := cc.compileNode(fmt.Sprintf("n%d", i), st, nil, nil)
		if err != nil {
			return nil, err
		}
		// 收集器：本步全部出口的汇点，完成即记「本主干步已执行」。
		spineID := st.GetID()
		collector := fmt.Sprintf("n%d.done", i)
		cc.g.AddNode(graph.NodeFunc(collector, func(ctx context.Context, in any) (any, error) {
			return in, nil
		}), cc.traceOpt(spineID))
		cc.wire(sg.exits, collector)
		if prevCollector != "" {
			cc.g.AddEdge(prevCollector, sg.entry)
		} else {
			entry = sg.entry
		}
		prevCollector = collector
	}
	cc.g.AddNode(graph.NodeFunc(endNodeName, func(ctx context.Context, in any) (any, error) {
		return in, nil
	}), cc.traceOpt(""))
	cc.g.AddEdge(prevCollector, endNodeName)
	cc.g.SetEntry(entry)
	cc.g.SetOutput(endNodeName)
	return &compiledGraph{g: g, ledger: cc.ledger, owners: cc.owners}, nil
}

// structuralBound 计算编译图的激活数上界（WithStepLimit 的精确预算）：每个节点
// 激活次数都有结构界——条件取两侧较大者（恰一支跑）、loop 把 MaxIteration 乘进
// 体内、parallel 按分支累加；每步另加一个收集器。合法有界 workflow 因此永不撞
// 引擎默认预算。
func structuralBound(steps []Node) (int, error) {
	total := 0
	for _, st := range steps {
		if st == nil {
			return 0, fmt.Errorf("workflow: step is nil")
		}
		b, err := nodeBound(st, nil)
		if err != nil {
			return 0, err
		}
		total += b + 1 // +1 该步的收集器节点
	}
	return total, nil
}

func nodeBound(n Node, path []Node) (int, error) {
	if len(path) >= maxCompileDepth {
		return 0, fmt.Errorf("workflow: node nesting deeper than %d", maxCompileDepth)
	}
	if n == nil {
		return 1, nil
	}
	switch t := n.(type) {
	case *Condition:
		if nodeInPath(t, path) {
			return 0, fmt.Errorf("workflow: cyclic node reference")
		}
		path = append(path, t)
		tb, err := nodeBound(t.TrueNode, path)
		if err != nil {
			return 0, err
		}
		fb, err := nodeBound(t.FalseNode, path)
		if err != nil {
			return 0, err
		}
		if tb > fb {
			return 1 + tb, nil
		}
		return 1 + fb, nil
	case *Router:
		if nodeInPath(t, path) {
			return 0, fmt.Errorf("workflow: cyclic node reference")
		}
		path = append(path, t)
		worst := 1 // 未命中错误节点
		keys := make([]string, 0, len(t.Routes))
		for k := range t.Routes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			rb, err := nodeBound(t.Routes[k], path)
			if err != nil {
				return 0, err
			}
			if rb > worst {
				worst = rb
			}
		}
		return 1 + worst, nil
	case *Loop:
		if nodeInPath(t, path) {
			return 0, fmt.Errorf("workflow: cyclic node reference")
		}
		if t.MaxIteration > maxLoopBudget {
			return 0, fmt.Errorf("workflow: loop %s MaxIteration %d exceeds the compile-time budget cap %d", t.ID, t.MaxIteration, maxLoopBudget)
		}
		path = append(path, t)
		body, err := nodeBound(t.Body, path)
		if err != nil {
			return 0, err
		}
		mi := t.MaxIteration
		if mi < 0 {
			mi = 0
		}
		// 控制器至多跑 mi+1 次（mi 次继续 + 1 次 done），体至多 mi 轮。
		return mi + 1 + mi*body, nil
	case *Parallel:
		if nodeInPath(t, path) {
			return 0, fmt.Errorf("workflow: cyclic node reference")
		}
		path = append(path, t)
		total := 2 // split + merge
		for _, child := range t.Nodes {
			cb, err := nodeBound(child, path)
			if err != nil {
				return 0, err
			}
			total += 2 + cb // 克隆头 + 分支体 + 尾汇点
		}
		return total, nil
	default:
		return 1, nil
	}
}
