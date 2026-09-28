// Durability 档位与持久化缝（母约 §7 G6）。
//
// Checkpointer/Checkpoint 声明在本包内（CacheStore 先例）：graph 不依赖
// internal/session，存储适配留给调用方。v1 只交付提交侧的时机控制——
// 三个档位只回答「节点完成后的事件何时向 sink 落」，读取与恢复不在本缝。
// 切片 27 起扩展出 HITL 中断条目（EntryInterrupt），按同一档位语义落进同一条流。
package graph

import (
	"context"
	"errors"
	"fmt"
)

// Durability 控制图在每个节点完成后向 Checkpointer 提交事件的时机（母约 §7）。
// 零值即 DurabilitySync：未声明档位的图按最安全档执行。
type Durability int

const (
	// DurabilitySync 在下一步开始前同步落盘（默认，最安全）。
	DurabilitySync Durability = iota
	// DurabilityAsync 在下一步执行时异步落盘：提交交由后台冲刷 goroutine，
	// 不挡派发；Run 返回前冲刷完毕，条目不丢、先后不乱，但不对
	// 「后继节点体可见前驱条目」作任何承诺。
	DurabilityAsync
	// DurabilityExit 仅图退出时落盘：运行中 sink 零条目，Run 返回前一次性落齐。
	DurabilityExit
)

// EntryKind 区分条目流里的两种事实。零值即 EntryCompletion：切片 24 的完成条目
// 在字节与读数上都不变，新事实必须显式选择自己的种类。
type EntryKind int

const (
	// EntryCompletion 节点完成条目（切片 24 既有形状：Seq/Node/Output）。
	EntryCompletion EntryKind = iota
	// EntryInterrupt HITL 中断条目：Node 是等待节点，Input 是激活输入，Interrupt 是中断请求全量。
	EntryInterrupt
)

// Checkpoint 是单条持久化条目：Seq 由消费者在提交时机盖章，从 1 起连续递增，次序即
// 完成处理次序。Kind/Input/Interrupt 是切片 27 扩展的三字段，完成条目上恒为零值/nil。
type Checkpoint struct {
	Seq       int
	Node      string
	Output    any
	Kind      EntryKind
	Input     any
	Interrupt *Interrupt
}

// Checkpointer 是持久化 sink。提交失败永不静默：失败可 errors.Is 到
// sink 交出的那个错误，Run 不把该次执行当成结论交出。
type Checkpointer interface {
	Append(ctx context.Context, cp Checkpoint) error
}

// WithDurability 设置持久化档位。取值一次：进入 Run 时从 builder 读取，
// 与 stepLimit/maxConcurrency 同一套不变量。
func WithDurability(d Durability) Option { return func(c *config) { c.durability = d } }

// WithCheckpointer 挂载持久化 sink；不挂（nil）即不持久化，
// 声明了档位而没有 sink 的图照常执行。
func WithCheckpointer(cp Checkpointer) Option { return func(c *config) { c.checkpointer = cp } }

// beginCommits 在进入 consume 之前启动 Async 档的后台冲刷 goroutine。
// 只有 Async 且已挂 sink 时才存在这条 goroutine；冲刷器只拥有自己的
// 首错误栈，与引擎之间仅共享 commits 通道（零锁模型不破）。
func (s *scheduler) beginCommits() {
	if s.checkpointer == nil || s.durability != DurabilityAsync {
		return
	}
	s.commits = make(chan Checkpoint)
	done := make(chan error)
	go func() {
		var first error
		for cp := range s.commits {
			if err := s.checkpointer.Append(s.ctx, cp); err != nil && first == nil {
				first = err
			}
		}
		done <- first
	}()
	s.flushDone = done
}

// record 在消费者侧处理一个节点完成事件的提交：Seq 在这里盖章，
// 三个档位各自落位。Sync 的失败向上交（Run 不交出结论）；Async 的失败由
// 冲刷器记账、Run 退出时上交；Exit 只积累、退出时一次落齐。
func (s *scheduler) record(item queueItem) error {
	if s.checkpointer == nil {
		return nil
	}
	s.seq++
	return s.commit(Checkpoint{Seq: s.seq, Node: item.name, Output: item.out})
}

// commit 按档位落位一条已盖章的条目：Sync 即时、Async 交冲刷器、Exit 积累；完成条目
// 与中断条目走同一时机语义。
func (s *scheduler) commit(cp Checkpoint) error {
	switch s.durability {
	case DurabilitySync:
		if err := s.checkpointer.Append(s.ctx, cp); err != nil {
			return fmt.Errorf("graph: checkpoint sink: %w", err)
		}
	case DurabilityAsync:
		s.commits <- cp
	default:
		s.exitQueue = append(s.exitQueue, cp)
	}
	return nil
}

// finishCommits 在 Run 交出结论之前收尾提交侧：Async 冲刷完毕、Exit 一次落齐；已完成
// 节点的条目在失败路径上也照常交付。挂起也是一种结论——sink 错误在挂起路径上经
// errors.Join 并入（两件事都可达），且不掩盖运行本身的首要错误。
func (s *scheduler) finishCommits(res *Result, runErr error) (*Result, error) {
	if s.checkpointer == nil {
		return res, runErr
	}
	suspended := runErr != nil && errors.Is(runErr, ErrSuspended)
	switch s.durability {
	case DurabilityAsync:
		close(s.commits)
		if err := <-s.flushDone; err != nil {
			if runErr == nil {
				return nil, fmt.Errorf("graph: checkpoint sink: %w", err)
			}
			if suspended {
				return nil, errors.Join(runErr, fmt.Errorf("graph: checkpoint sink: %w", err))
			}
		}
	case DurabilityExit:
		for _, cp := range s.exitQueue {
			if err := s.checkpointer.Append(s.ctx, cp); err != nil {
				if runErr == nil {
					return nil, fmt.Errorf("graph: checkpoint sink: %w", err)
				}
				if suspended {
					return nil, errors.Join(runErr, fmt.Errorf("graph: checkpoint sink: %w", err))
				}
			}
		}
	}
	return res, runErr
}
