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

// config 保存调度参数；stepLimitSet 区分「用户显式设置」与「默认值」，
// 构建期告警（判据 8）只对未显式设置的环图发出。
type config struct {
	maxConcurrency int
	stepLimit      int
	stepLimitSet   bool
}

// WithMaxConcurrency 限制同时在途的节点数，0 表示不限。
// 打满时新激活进入 FIFO 等待队列，完成一个自动派发一个。
func WithMaxConcurrency(n int) Option {
	return func(c *config) { c.maxConcurrency = n }
}

// WithStepLimit 设置单次 Run 的最大步数，未设置时默认 1000。
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
	cfg    config
	nodes  map[string]Node
	edges  []edge
	entry  string
	output string
}

// edge 是一条已声明的无条件边，保留声明顺序。
type edge struct{ from, to string }

// New 创建空图。
func New(opts ...Option) *Graph {
	g := &Graph{nodes: map[string]Node{}}
	for _, opt := range opts {
		opt(&g.cfg)
	}
	return g
}

// AddNode 加入节点，返回自身以支持链式调用。
func (g *Graph) AddNode(n Node) *Graph {
	g.nodes[n.Name()] = n
	return g
}

// AddEdge 加一条无条件边。
func (g *Graph) AddEdge(from, to string) *Graph {
	g.edges = append(g.edges, edge{from: from, to: to})
	return g
}

// AddConditional 加一条条件边：源节点输出使谓词为真时激活 to。
func (g *Graph) AddConditional(from, to string, p Predicate) *Graph { return g }

// AddDefault 加一条兜底边：源节点没有任何具体边命中时激活 to。
func (g *Graph) AddDefault(from, to string) *Graph { return g }

// AddJoin 把多个前驱汇聚到 to，to 只在全部前驱完成后激活一次。
func (g *Graph) AddJoin(from []string, to string) *Graph { return g }

// SetEntry 指定入口节点。
func (g *Graph) SetEntry(name string) *Graph {
	g.entry = name
	return g
}

// SetOutput 指定输出节点，其返回值即 Result.Output。
func (g *Graph) SetOutput(name string) *Graph {
	g.output = name
	return g
}

// Validate 执行构建期校验。
//
// 当前覆盖 §3.5 中与「引擎无法安全执行」直接相关的两项：入口缺失与悬空边。
// 其余校验项按契约 §11.3 的次序各自先取得 RED 再实现。
func (g *Graph) Validate() error {
	if _, ok := g.nodes[g.entry]; !ok {
		return fmt.Errorf("graph: entry node %q does not exist", g.entry)
	}
	for _, e := range g.edges {
		if _, ok := g.nodes[e.from]; !ok {
			return fmt.Errorf("graph: edge from node %q does not exist", e.from)
		}
		if _, ok := g.nodes[e.to]; !ok {
			return fmt.Errorf("graph: edge to node %q does not exist", e.to)
		}
	}
	return nil
}

// Warnings 返回非致命的构建期提示，与 Validate 的致命语义分离。
func (g *Graph) Warnings() []error { return nil }

// Typed 把带具体类型的函数适配为 Node，提供编译期类型安全出口。
// 输入实际类型与 TIn 不符时返回错误而不是 panic。
func Typed[TIn, TOut any](name string, fn func(ctx context.Context, in TIn) (TOut, error)) Node {
	return NodeFunc(name, func(ctx context.Context, in any) (any, error) {
		typed, ok := in.(TIn)
		if !ok {
			return nil, errors.New("graph: input type mismatch")
		}
		return fn(ctx, typed)
	})
}
