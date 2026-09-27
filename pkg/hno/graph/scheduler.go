package graph

import (
	"context"
	"sort"
)

// queueItem 是生产者 goroutine 投向消费者的唯一消息。
type queueItem struct {
	name string
	out  any
	err  error
}

// activation 是一次待派发的节点执行，in 为其输入。
type activation struct {
	name string
	in   any
}

// scheduler 实现 §3.3 的单消费者零锁模型：
// 生产者 goroutine 只执行节点并向 queue 发送；消费者（调用 Run 的 goroutine）
// 是 queue 的唯一读者，也是 pending/running/result 的唯一写者，因此这些
// 状态不需要任何锁。
type scheduler struct {
	g       *Graph
	ctx     context.Context
	queue   chan queueItem
	done    chan struct{}
	result  *Result
	pending []activation
	running int
}

// Run 从入口节点驱动图执行到收敛。
func (g *Graph) Run(ctx context.Context, in any) (*Result, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	s := &scheduler{
		g:       g,
		ctx:     ctx,
		queue:   make(chan queueItem),
		done:    make(chan struct{}),
		result:  &Result{values: make(map[string]any)},
		pending: []activation{{name: g.entry, in: in}},
	}
	return s.consume(ctx)
}

// consume 是可变状态的唯一写入者。
func (s *scheduler) consume(ctx context.Context) (*Result, error) {
	defer close(s.done)
	for {
		s.dispatch()
		if s.running == 0 {
			sort.Strings(s.result.completed)
			return s.result, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case item := <-s.queue:
			s.running--
			if item.err != nil {
				return nil, item.err
			}
			s.complete(item)
		}
	}
}

// dispatch 把等待队列按声明顺序全部派发。
func (s *scheduler) dispatch() {
	for len(s.pending) > 0 {
		act := s.pending[0]
		s.pending = s.pending[1:]
		s.spawn(act)
	}
}

// spawn 占用一个执行槽并建立该激活的生产者 goroutine。
func (s *scheduler) spawn(act activation) {
	s.running++
	go func() {
		out, err := s.g.nodes[act.name].Run(s.ctx, act.in)
		item := queueItem{name: act.name, out: out, err: err}
		select {
		case s.queue <- item:
		case <-s.done:
		}
	}()
}

// complete 记录节点输出并激活其后继。
func (s *scheduler) complete(item queueItem) {
	s.result.values[item.name] = item.out
	s.result.completed = append(s.result.completed, item.name)
	if item.name == s.g.output {
		s.result.output = item.out
	}
	s.pending = append(s.pending, s.successors(item.name, item.out)...)
}

// successors 是 §3.4 路由核心的落点：给定刚完成的节点与其输出，返回应被激活的
// 后继。目前只有无条件边参与路由，条件边 / Join / Default 在此函数内扩展。
func (s *scheduler) successors(name string, out any) []activation {
	var acts []activation
	for _, e := range s.g.edges {
		if e.from == name {
			acts = append(acts, activation{name: e.to, in: out})
		}
	}
	return acts
}
