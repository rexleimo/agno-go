// Package graph 是 v3 G2 的零锁控制流图调度内核。
//
// 设计定稿见 docs/design/v3-platform.md §3：单消费者模型借鉴 adk-go
// workflow/scheduler.go（生产者只向 channel 发送，消费者独占可变状态），
// 构建期校验借鉴 adk-go workflow/validation.go 的精简版。
//
// 依赖方向单向：本包不依赖 workflow/agent/runner。
package graph

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// ErrStepLimitExceeded 表示图执行步数达到 WithStepLimit 上限。
// 图允许有环，因此必须有这个安全阀（对应 LangGraph 的 recursion_limit）。
var ErrStepLimitExceeded = errors.New("graph: step limit exceeded")

// Node 是图节点。刻意极简——节点内部想怎么循环都行。
type Node interface {
	// Name 返回节点在图内的唯一标识。
	Name() string
	// Run 执行一个节点，in 是它的前驱（或入口输入）产出的值。
	Run(ctx context.Context, in any) (any, error)
}

// nodeFunc 把普通函数适配为 Node。
type nodeFunc struct {
	name string
	fn   func(context.Context, any) (any, error)
}

func (n nodeFunc) Name() string { return n.name }

func (n nodeFunc) Run(ctx context.Context, in any) (any, error) { return n.fn(ctx, in) }

// NodeFunc 用函数构造一个 Node。
func NodeFunc(name string, fn func(ctx context.Context, in any) (any, error)) Node {
	return nodeFunc{name: name, fn: fn}
}

// Predicate 是条件边的路由判定，入参是源节点的输出。
type Predicate func(out any) bool

// Option 配置 Graph 的调度参数。
type Option func(*config)

// defaultStepLimit 是未显式设置 WithStepLimit 时生效的预算（母约 §7:354）。
// 它必须有值：图允许有环，而没有预算的环就是永不返回的 Run。
const defaultStepLimit = 1000

// maxRenderedCycleNames 是环文案一次交出的被点名节点数上限（含收尾重复的那一格）。
// 判据 7 的致命文案与判据 8 的提示共用这段渲染，因此界对两者同时生效：短环照旧整条写出，
// 长环改为「点名开头 + 省略 + 说出规模」。
const maxRenderedCycleNames = 6

// cycleHeadWhenBounded 是环被截断时保留的开头点名格数（收尾另占一格）。留出的一格不是余量：
// 被截的文案最多点 5 个名，完整写出的文案最多点 6 个，于是「点名数正好顶满上限」本身就说明
// 这条环没有被省略——两种形状不会互相冒充。
const cycleHeadWhenBounded = maxRenderedCycleNames - 2

// config 保存调度参数；stepLimitSet 区分「用户显式设置」与「默认值」，
// 构建期告警（判据 8）只对未显式设置的环图发出。
type config struct {
	maxConcurrency int
	stepLimit      int
	stepLimitSet   bool
	durability     Durability
	checkpointer   Checkpointer
}

// WithMaxConcurrency 限制同时在途的节点数，0 或负数表示不限。
// 打满时新激活进入 FIFO 等待队列，完成一个自动派发一个。
// 与 WithStepLimit 不同，这里不校验取值：判据是 maxConcurrency > 0 才设槽，
// 于是「不限」是这条守卫直接给出的语义而非遗漏（负数是否算非法声明另见 R16-UNADJ-1）。
func WithMaxConcurrency(n int) Option {
	return func(c *config) { c.maxConcurrency = n }
}

// WithStepLimit 设置单次 Run 的最大步数，未设置时默认 1000。
// n 必须是正数：0 或负数既不是「不限步数」也不是「一步都不许跑」，Validate 会把它
// 连同那条声明一起拒掉。
func WithStepLimit(n int) Option {
	return func(c *config) {
		c.stepLimit = n
		c.stepLimitSet = true
	}
}

// Result 是一次图执行的可观察结果。
type Result struct {
	output    any
	values    map[string]any
	completed []string
}

// Output 返回 SetOutput 指定节点的返回值。
func (r *Result) Output() any { return r.output }

// Value 返回指定节点的输出；第二个返回值表示该节点是否产出过。
func (r *Result) Value(name string) (any, bool) {
	v, ok := r.values[name]
	return v, ok
}

// Completed 返回已完成节点名的字典序切片。
// 刻意按字典序而非完成顺序：并行分支之间没有确定的全局时序，
// 调用方不应依赖时序做断言。
func (r *Result) Completed() []string { return r.completed }

// Graph 由 builder 构造，构造期完成全部校验。
type Graph struct {
	cfg config
	// dupNames 是 AddNode 的同名注册账本，按声明顺序累积（切片 21，母约 §3.5 第 2 项）。
	// 它只是构建期的内部事实，不是公共面读数：判据是 Validate/Run 的返回值。
	dupNames []string
	nodes    map[string]Node
	policies map[string]nodePolicy
	edges    []edgeDecl
	entry    string
	output   string
	// pending 是本图最近一次 Run/Resume 挂起的产物（切片 27，母约 §9.2），由
	// Run/Resume 的调用链独占写：Resume 校验通过才消费置 nil；调用链不并发。
	pending *Suspension
}

// edgeKind 是一条边的路由语义。调度器只会路由自己认识的种类。
type edgeKind int

const (
	edgeUnconditional edgeKind = iota // §3.4 规则 1：全部激活
	edgeConditional                   // §3.4 规则 2：Predicate(out)==true 才激活
	edgeDefault                       // §3.4 规则 4：没有任何具体边命中时的兜底
	edgeJoin                          // §3.4 规则 3：全部声明前驱完成后以聚合输入激活一次
	edgeJoinSend                      // §6/G5：source 名下全部 Send 激活完成后以聚合输入激活 target 一次
)

// routable 声明该边种类是否已被调度器实现。
//
// 它支撑 fail-closed 不变式：不可路由的声明必须在构建期被拒，不允许被静默丢弃后
// 跑出一个 err==nil 且 Output()==nil 的空 Result。这里刻意用白名单而不是黑名单：
// 新增种类默认落在「拒绝」一侧，放行必须是有意识的一次修改。
func (k edgeKind) routable() bool {
	switch k {
	case edgeUnconditional, edgeConditional, edgeDefault, edgeJoin, edgeJoinSend:
		return true
	default:
		return false
	}
}

// requiresPredicate 声明该边种类是否必须自带谓词才可路由。
// 与 routable()/label() 同址，调用方因此不需要对 edgeKind 常量做裸比较。
func (k edgeKind) requiresPredicate() bool {
	return k == edgeConditional
}

// label 是错误文案里的种类名，用于把拒绝定位到具体声明。
func (k edgeKind) label() string {
	switch k {
	case edgeUnconditional:
		return "unconditional edge"
	case edgeConditional:
		return "conditional edge"
	case edgeDefault:
		return "default edge"
	case edgeJoin:
		return "join edge"
	case edgeJoinSend:
		return "join-send edge"
	default:
		return "edge"
	}
}

// edgeDecl 是一条已声明的边，保留声明顺序。
// p 只对 edgeConditional 有意义，其余种类忽略。
type edgeDecl struct {
	kind edgeKind
	from string
	to   string
	p    Predicate
}

// New 创建空图。
func New(opts ...Option) *Graph {
	g := &Graph{nodes: map[string]Node{}, policies: map[string]nodePolicy{}}
	for _, opt := range opts {
		opt(&g.cfg)
	}
	// 预算在这里落地，调度器因此永远读到一个可用数字，而不是靠自己去猜零值的含义。
	if !g.cfg.stepLimitSet {
		g.cfg.stepLimit = defaultStepLimit
	}
	return g
}

// AddNode 加入节点，返回自身以支持链式调用。变参 NodeOption 挂载节点级策略
// （Retry/Cache/Timeout/Trace，见 policy.go）；不传 options 的既有调用源兼容不变。
//
// 同名注册在 map 里仍是后注册者覆盖先注册者（既有担保「in-flight Run 的捕获隔离」
// 依赖这个形状，见 graph_test.go 的并发变异子用例），但覆盖本身不再是静默的：
// 这次注册被记进 dupNames，Validate 在构建期点名拒绝（母约 §3.5 第 2 项）。
// 记账而不在本方法处报错，是因为 builder 方法返回 *Graph、没有错误通道——
// 「增量检查」在此 API 形状下的唯一诚实读法就是记账 + Run 前全量复查。
// 策略表随同名覆盖一起被替换：后注册者的 options 完整接管该节点。
func (g *Graph) AddNode(n Node, opts ...NodeOption) *Graph {
	if _, exists := g.nodes[n.Name()]; exists {
		g.dupNames = append(g.dupNames, n.Name())
	}
	if len(opts) > 0 {
		var pol nodePolicy
		for _, opt := range opts {
			opt(&pol)
		}
		g.policies[n.Name()] = pol
	}
	g.nodes[n.Name()] = n
	return g
}

// AddEdge 加一条无条件边：from 完成后 to 必然被激活（§3.4 规则 1）。
func (g *Graph) AddEdge(from, to string) *Graph {
	return g.declare(edgeUnconditional, from, to, nil)
}

// AddConditional 加一条条件边：仅当 p 对 from 本次输出返回 true 时激活 to
// （§3.4 规则 2）。同一个源节点的多条条件边互不排斥，命中几条就激活几条。
// p 为 nil 时该声明缺少路由输入，Validate 会拒绝它 —— 引擎不会把它当成
// 「永不命中」而静默产出一个没有输出的结果。
func (g *Graph) AddConditional(from, to string, p Predicate) *Graph {
	return g.declare(edgeConditional, from, to, p)
}

// AddDefault 加一条兜底边：仅当 from 没有任何无条件边或条件边命中时才激活 to
// （§3.4 规则 4）。已有具体边命中时，兜底边不得触发。
func (g *Graph) AddDefault(from, to string) *Graph {
	return g.declare(edgeDefault, from, to, nil)
}

// AddJoin 声明把 from 的多个前驱汇聚到 to（§3.4 规则 3）。
//
// 激活语义：to 在 from 里每一个前驱都完成之后才被激活，它的输入不是某个前驱的输出，
// 而是 map[string]any —— 以完成前驱的名字为键、以其本次输出为值。前驱在本次 Run 各完成
// 一次时 to 恰好跑一次；前驱因条件环重跑时键集合不再增长，于是 to 会被再次激活。
//
// 凑不齐的屏障不会把「已完成的这些」当成结论交出去。图仍在产生激活时，未收敛由
// WithStepLimit 的安全阀结束；汇聚前驱始终没跑且再无待派激活时，今天交出的是 err==nil
// 而 Output()==nil 的 Result，该形状尚未裁决（票面 §15.5 的 R19-Q1）。
//
// 前驱少于 2 个的汇聚声明不构成屏障（单前驱就是一条无条件边，写成 AddJoin 只会让读者
// 以为这里有屏障），Validate 会点名那个目标并拒绝，Run 随之拒绝。
func (g *Graph) AddJoin(from []string, to string) *Graph {
	for _, f := range from {
		g.declare(edgeJoin, f, to, nil)
	}
	return g
}

func (g *Graph) declare(kind edgeKind, from, to string, p Predicate) *Graph {
	g.edges = append(g.edges, edgeDecl{kind: kind, from: from, to: to, p: p})
	return g
}

// SetEntry 指定入口节点。
func (g *Graph) SetEntry(name string) *Graph {
	g.entry = name
	return g
}

// SetOutput 指定输出节点，其返回值即 Result.Output。
// 名字不是已注册节点（或从未调用本方法）时该声明缺少路由目标，Validate 会拒绝它 ——
// 引擎不会跑完一张图后交出 err==nil 而 Output()==nil 的结果。
// 边界：节点存在但从入口不可达同样会交出那个 nil，因此也被 Validate 拒绝（§3.5 第 4
// 项）。仍不判的是节点可达、只是本次谓词没把它跑到，那属于运行期事实；注册名为空串
// 的节点与「从未声明」在公共面同形，也仍不在本检查范围内。
func (g *Graph) SetOutput(name string) *Graph {
	g.output = name
	return g
}

// Validate 执行构建期校验。
//
// 落地了 §3.5 的哪些必查项以 docs/design 下的切片契约为准，这里只固定一条调用方依赖的
// 次序：先判声明本身是否成立（步数预算、入口、输出、边两端、条件边谓词、边种类可路由），
// 再判汇聚声明凑不出屏障（前驱数 < 2），然后判图能否执行（不可达节点，然后是不可断开的环），
// 末位判注册账本（同名节点被重复注册）。
// 次序本身可观察——一张既不可达又含环的图
// 报的是可达性——所以由测试钉住，改动它需要新的 RED。
func (g *Graph) Validate() error {
	if err := g.rejectInvalidStepLimit(); err != nil {
		return err
	}
	if _, ok := g.nodes[g.entry]; !ok {
		return fmt.Errorf("graph: entry node %q does not exist", g.entry)
	}
	if _, ok := g.nodes[g.output]; !ok {
		return fmt.Errorf("graph: output node %q does not exist", g.output)
	}
	for _, e := range g.edges {
		if _, ok := g.nodes[e.from]; !ok {
			return fmt.Errorf("graph: %s from node %q does not exist", e.kind.label(), e.from)
		}
		if _, ok := g.nodes[e.to]; !ok {
			return fmt.Errorf("graph: %s to node %q does not exist", e.kind.label(), e.to)
		}
		if e.kind.requiresPredicate() && e.p == nil {
			return fmt.Errorf("graph: %s from %q to %q declares no predicate", e.kind.label(), e.from, e.to)
		}
		if !e.kind.routable() {
			return fmt.Errorf("graph: %s from %q to %q is declared but not routable by this engine", e.kind.label(), e.from, e.to)
		}
	}
	if err := g.rejectUnderfedJoinTargets(); err != nil {
		return err
	}
	if err := g.rejectUnreachableNodes(); err != nil {
		return err
	}
	if err := g.rejectUnconditionalCycles(); err != nil {
		return err
	}
	return g.rejectDuplicateNodeNames()
}

// joinBarriers 把汇聚声明收成两张表：waits 是「目标 → 它等待的去重前驱集合」，
// feeds 是它的反向索引（前驱 → 目标列表，按声明顺序累积，因此屏障前进的顺序不依赖 map 迭代）。
//
// 它是「哪些前驱算同一个屏障」的唯一定义：构建期判前驱数与运行期判屏障凑齐都读它。
// 两处各自数一遍时，完全可能一处按名字去重、另一处按声明条数算，那对
// AddJoin([]string{"a","a"}, "j") 会给出相反的结论 —— 而两个结论都是静默的。
func joinBarriers(edges []edgeDecl) (waits map[string]map[string]bool, feeds map[string][]string) {
	waits = map[string]map[string]bool{}
	feeds = map[string][]string{}
	for _, e := range edges {
		if e.kind != edgeJoin {
			continue
		}
		if waits[e.to] == nil {
			waits[e.to] = map[string]bool{}
		}
		if waits[e.to][e.from] {
			continue
		}
		waits[e.to][e.from] = true
		feeds[e.from] = append(feeds[e.from], e.to)
	}
	return waits, feeds
}

// rejectUnderfedJoinTargets 是票面 :65「Join 前驱数 < 2」的落点：汇聚屏障的语义就是
// 「等两个以上的前驱」，只声明一个前驱的 AddJoin 没有屏障可等 —— 把它当成无条件边放行，
// 调用方读到的是一个自己没写过的语义（AddJoin 与 AddEdge 的差别消失了），而聚合输入的
// 那个 map 也永远只有一个键。
//
// 判据放在逐条边的端点检查之后：端点根本不存在的声明更基础，先报它才能让
// 「join 边指向未注册节点」这类错误仍由端点文案定位（既有测试钉着）。
// 一次只点一个目标名，取字典序第一个：与 rejectUnreachableNodes 同理，节点集是 map，
// 稳定文案优先于一次报全。
func (g *Graph) rejectUnderfedJoinTargets() error {
	waits, _ := joinBarriers(g.edges)
	var thin []string
	for target, preds := range waits {
		if len(preds) < 2 {
			thin = append(thin, target)
		}
	}
	if len(thin) == 0 {
		return nil
	}
	sort.Strings(thin)
	return fmt.Errorf("graph: join target %q must declare at least 2 distinct predecessors, got %d", thin[0], len(waits[thin[0]]))
}

// rejectInvalidStepLimit 判的是 WithStepLimit 这条声明本身是否成立，与拓扑无关：
// 非正数没有可读的含义 —— 当成「不限步数」会让安全阀在它最该起作用的那张图上关掉，
// 当成「一步都不许跑」会让 Run 在入口就拒绝，两者都不是调用方写下那个数字时的意图。
//
// 只判「显式写过且非正」：没写过这条声明的图（例如零值 Graph）不该收到一句关于自己
// 没写过的旋钮的拒绝，它的真实问题由后面的拓扑检查说出。
// 文案点名那条声明，因为它定位的是调用方自己写下的旋钮，而不是图的形状。
func (g *Graph) rejectInvalidStepLimit() error {
	if !g.cfg.stepLimitSet || g.cfg.stepLimit > 0 {
		return nil
	}
	return fmt.Errorf("graph: WithStepLimit(%d) is not a usable step limit: a positive number of steps is required", g.cfg.stepLimit)
}

// rejectUnreachableNodes 是 §3.5 第 4 项的落点：从入口沿已声明边走不到的节点永远不会
// 被执行，静默放行等于把一个少跑了节点的 Result 交给调用方 —— 输出节点正是这种形状里
// 最坏的一种，Run 会返回 err==nil 而 Output()==nil。
//
// 可达只按声明的边算，不分边种类、也不看谓词：构建期判不了谓词会命中什么，把「可达」
// 实现成「这次真的跑过」等于把一切条件分支判为非法。
//
// 一次只点一个节点名，取字典序第一个：节点集是 map，注册顺序在公共面不可恢复，而稳定
// 的文案比一次报全更重要（聚合诊断另立契约）。
func (g *Graph) rejectUnreachableNodes() error {
	succ := g.adjacency(func(edgeDecl) bool { return true })
	// AddJoinSend 的 source 只被运行期 Send 指名，构建期无法反驳「会有 Sender 派发给它」
	//（与「可达不看谓词」的同一乐观先例），因此视为可达并从它继续传播。
	reached := map[string]bool{g.entry: true}
	queue := []string{g.entry}
	for _, e := range g.edges {
		if e.kind == edgeJoinSend && !reached[e.from] {
			reached[e.from] = true
			queue = append(queue, e.from)
		}
	}
	for len(queue) > 0 {
		for _, next := range succ[queue[0]] {
			if reached[next] {
				continue
			}
			reached[next] = true
			queue = append(queue, next)
		}
		queue = queue[1:]
	}

	var missing []string
	for name := range g.nodes {
		if !reached[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf("graph: unreachable node %q: not reachable from entry %q through any declared edge", missing[0], g.entry)
}

// adjacency 按声明顺序把 keep 认可的边收成 from→后继表，是两个可达性检查共用的骨架。
func (g *Graph) adjacency(keep func(edgeDecl) bool) map[string][]string {
	adj := map[string][]string{}
	for _, e := range g.edges {
		if keep(e) {
			adj[e.from] = append(adj[e.from], e.to)
		}
	}
	return adj
}

// alwaysTakenAdjacency 返回「起点一被激活就必定被走」的边的邻接表，按声明顺序保留目标。
//
// 无条件边（§3.4 规则 1）必定被走。兜底边（规则 4）只在起点没有任何条件边替代时必定
// 被走：有条件边时它可能被跳过，经过该边的环因此可断。条件边永不必定被走。
func (g *Graph) alwaysTakenAdjacency() map[string][]string {
	breakable := map[string]bool{}
	for _, e := range g.edges {
		if e.kind == edgeConditional {
			breakable[e.from] = true
		}
	}

	return g.adjacency(func(e edgeDecl) bool {
		return e.kind == edgeUnconditional || (e.kind == edgeDefault && !breakable[e.from])
	})
}

// rejectUnconditionalCycles 是 §3.5 第 7 项的落点：环上每条边都必定被走时，Run 不会
// 终止，所以这张图必须在构建期被拒，而不是留给调用方一个永不返回的执行。
//
// 判据取「不可断开」而非「环上边种类必须全为无条件边」：一条可命中的条件边、或一条有
// 条件替代的兜底边，都让调度器在某一步拥有不再激活后继的可能，那正是 §3.5 要放行的
// 合法环。起点按声明顺序遍历，因此同一张图永远报出同一个环。
//
// 边界：谓词恒真的条件边在这里算可断开，因为构建期无法判谓词。那类环由步数上限兜住
// （scheduler.dispatch），本检查不冒充它。
func (g *Graph) rejectUnconditionalCycles() error {
	if cycle, alwaysTaken := firstCycleOf(g.alwaysTakenAdjacency(), g.edges); alwaysTaken {
		return fmt.Errorf("graph: unconditional cycle %s: every edge on it is always taken, so Run cannot terminate", cycle)
	}
	return nil
}

// cycleOf 把闭合于 closing 的那段路径还原成 `"a" -> "b" -> "a"` 形式的可定位文案。
//
// 界：节点数在 maxRenderedCycleNames 以内时逐格点名，收尾回到哪一格照样写出；超过时只点
// 开头若干格，中间用省略号，并把环的规模作为数字交给读的人。这段文案的两个天然用法是
// 「启动时打印一次」和「贴进 issue」，整条列出 400 个节点两者都放不下（实测 4532 字节，
// 20000 节点时 260134 字节）。
func cycleOf(path []string, closing string) string {
	start := 0
	for i, name := range path {
		if name == closing {
			start = i
			break
		}
	}

	nodes := path[start:]
	bounded := len(nodes)+1 > maxRenderedCycleNames
	shown := nodes
	if bounded {
		shown = nodes[:cycleHeadWhenBounded]
	}

	var sb strings.Builder
	for _, name := range shown {
		fmt.Fprintf(&sb, "%q -> ", name)
	}
	if bounded {
		fmt.Fprintf(&sb, "… -> %q (%d nodes)", closing, len(nodes))
		return sb.String()
	}
	fmt.Fprintf(&sb, "%q", closing)
	return sb.String()
}

// rejectDuplicateNodeNames 是母约 §3.5 第 2 项（票面 B6 第 2 项）的落点：同一张图上
// 重复注册同名节点时，此前的形状是静默覆盖 —— 后注册者悄悄顶掉先注册者，构建期一句话
// 没有，而跑出来的结论来自一张调用方没有写过的图（覆盖可从 Output 观察，切片 21 契约
// M1 形2 实测）。
//
// 检查放在末位：声明是否成立、图能否执行都比「名字撞了」更基础 —— 一张连端点都缺失
// 或根本不可达的图，先报那些事实才能让调用方定位到那一行声明。次序由 Validate 的文档
// 固定，改动它需要新的 RED。
//
// 一次只点一个名字，取账本首个（即声明顺序里第一次撞名的那格）：账本是声明序，结果稳定；
// 多个重复名时点哪一个、一次报几个都不在公共面担保范围（切片 21 契约 observabilityLimit）。
func (g *Graph) rejectDuplicateNodeNames() error {
	if len(g.dupNames) == 0 {
		return nil
	}
	return fmt.Errorf("graph: node name %q is declared more than once", g.dupNames[0])
}

// Warnings 返回非致命的构建期提示，与 Validate 的致命语义分离。
//
// §3.5 判据 8 落在这里。判据 7 只拒「每条边都必定被走」的环，而被放行的条件环照样能在
// 谓词恒真时把 Run 一路推到步数上限；没写过 WithStepLimit 的调用方届时读到的是
// defaultStepLimit —— 一个他没有选过的数字。提示的价值正在于此：不写就没人会在第 1000
// 步之前想到预算这件事。
//
// 两条边界，都来自票面对判据 8 的措辞（它是「警告」而不是「错误」）：
//   - 只读。不改 cfg、不消费状态，同一张图问几次答案一致；以写代读（顺手把预算补上）会把
//     一句提示变成静默的执行变更。
//   - 不进 Validate()。把危险说成非法，等于替调用方决定这张图不能跑。
//
// 环的判据取「全部已声明边」而不是「必定被走的边」：一张能通过判据 7 的图，剩下的环必然
// 至少有一条边可以被跳过。本方法不查图是否合法（非法图上该说什么属未裁决面，见切片 15 契约
// R15-UNADJ-1）。
//
// 文案受两条约束，都来自「本方法只有构建期可见的事实可说」：
//   - 断言的旋钮与默认数值要写出来（WithStepLimit、defaultStepLimit），因为那正是调用方
//     还没选过的东西；只说「有环没预算」等于让人自己去翻文档。
//   - 不断言某一次执行会怎样停。本方法既没跑过图也不知道谓词，而同一族自己的正例实测是
//     「在自己的收敛上停下、没撞到 1000」；非法图上更是一次都不跑。仓库对同一类问题已有判红
//     口径（graph_test.go 把「把永远不会跑的东西说成运行时风险」当缺陷），这里沿用同一取向。
func (g *Graph) Warnings() []error {
	if g.cfg.stepLimitSet {
		return nil
	}
	cycle, ok := firstCycleOf(g.adjacency(func(edgeDecl) bool { return true }), g.edges)
	if !ok {
		return nil
	}
	return []error{fmt.Errorf(
		"graph: cycle %s has no explicit budget: WithStepLimit was not set, so the default stepLimit=%d is what bounds this graph",
		cycle, defaultStepLimit)}
}

// firstCycleOf 在给定邻接表上找第一个环，返回 cycleOf 还原出的可定位文案。
//
// 起点取自 edges 而不是 nodes：nodes 是 map，遍历序不可恢复，用它定序会让同一张图报出
// 不同的环。edges 是声明序，因此结果稳定——这条约定从 §3.5 第 7 项原样继承过来。
//
// 两处判据共用这一套遍历，差别只在喂进来的邻接表：判据 7 只给必定被走的边（环上每条边
// 都跳不掉才算死循环），判据 8 给全部已声明边（能被放行的环照样吃得完默认预算）。
func firstCycleOf(adj map[string][]string, edges []edgeDecl) (string, bool) {
	const (
		unvisited = iota
		onPath
		settled
	)

	state := map[string]int{}
	var path []string
	var cycle string

	var walk func(string) bool
	walk = func(name string) bool {
		state[name] = onPath
		path = append(path, name)
		for _, next := range adj[name] {
			switch state[next] {
			case onPath:
				cycle = cycleOf(path, next)
				return true
			case unvisited:
				if walk(next) {
					return true
				}
			}
		}
		path = path[:len(path)-1]
		state[name] = settled
		return false
	}

	for _, e := range edges {
		if state[e.from] == unvisited && walk(e.from) {
			return cycle, true
		}
	}
	return "", false
}

// Typed 把带具体类型的函数适配为 Node，提供编译期类型安全出口。
// 输入实际类型与 TIn 不符时返回错误而不是 panic。
//
// 错误必须可归因（票面 :70「可判定」按 §11.1 先例精确化）：图上每个 Typed 节点都在同一处
// 断言失败，常量文案无法告诉调用方坏的是哪一个节点、期望什么类型、实际收到什么类型。
// nil 单独渲染成 "no value"：reflect.TypeOf(nil) 返回 nil Type，用 %s 打印它交出的是
// 格式化噪声 `%!s(<nil>)`，而「什么都没收到」正是调用方需要读出来的那件事。
func Typed[TIn, TOut any](name string, fn func(ctx context.Context, in TIn) (TOut, error)) Node {
	expected := reflect.TypeOf((*TIn)(nil)).Elem()
	return NodeFunc(name, func(ctx context.Context, in any) (any, error) {
		typed, ok := in.(TIn)
		if !ok {
			return nil, fmt.Errorf("graph: node %q: input type mismatch: expected %s, got %s", name, expected, actualTypeLabel(in))
		}
		return fn(ctx, typed)
	})
}

// actualTypeLabel 描述调用方实际交来的值，用于类型不符错误的 got 槽位。
func actualTypeLabel(in any) string {
	if in == nil {
		return "no value"
	}
	return reflect.TypeOf(in).String()
}
