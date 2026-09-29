package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/runner"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// wiredStreamModes lists the modes with a runtime producer; since this slice every
// protocol mode is wired. The gate's remaining job is fail-closed for mode ordinals
// that the protocol does not define at all, and wiring a future mode still means
// adding its row here plus its producer's judgement rows (母约 §5).
// 七模式全部接线。本表的剩余职责是对协议之外的模式序数 fail-closed；未来新增模式仍意味着
// 在此加一行并补其生产者的判据行（母约 §5）。
var wiredStreamModes = map[run.StreamMode]bool{
	run.StreamValues:      true,
	run.StreamUpdates:     true,
	run.StreamMessages:    true,
	run.StreamTasks:       true,
	run.StreamCheckpoints: true,
	run.StreamDebug:       true,
	run.StreamCustom:      true,
}

// normalizeStreamModes collapses the variadic selection into a set: no modes means
// StreamMessages, repeats collapse, and any mode without a wired producer fails the
// whole call closed naming the first such mode. Because the selection is a set, union
// semantics hold and argument order cannot change what gets emitted.
// normalizeStreamModes 把变参选择收成集合：无模式默认 StreamMessages、重复去重、任一未接线
// 模式让整次调用 fail-closed 并点名首个未接线模式。集合语义即并集语义，与给出次序无关。
func normalizeStreamModes(modes []run.StreamMode) (map[run.StreamMode]bool, error) {
	if len(modes) == 0 {
		modes = []run.StreamMode{run.StreamMessages}
	}

	set := make(map[run.StreamMode]bool, len(modes))
	for _, mode := range modes {
		if !wiredStreamModes[mode] {
			return nil, fmt.Errorf("agent: stream mode %d has no wired producer: %w", int(mode), run.ErrUnsupportedStreamMode)
		}
		set[mode] = true
	}
	return set, nil
}

// streamEmitter is the single place a streaming run turns a produced value into an
// event: it owns the family gate (an unselected family is neither recorded nor sent),
// the ledger (RunOutput.Events) and the content sequence numbers. Emission runs on the
// goroutine that drives the kernel, so ordering is a property of that coroutine. The
// per-turn families advance turn/content counters on that same coroutine; only custom
// writes carry a mutex, because tool handlers emit from their own goroutines.
// streamEmitter 是流式运行把产出值变成事件的唯一出口。它持有按族门控（未选中的族
// 既不记账也不上通道）、记账序列（RunOutput.Events）与内容事件的序列号。发射发生在
// 驱动内核的那条协程上，因此次序是该协程的构造属性，不是锁的产物。逐回合族在同一条
// 协程上推进回合/内容计数；只有 custom 写入持锁，因为工具 handler 在自己的协程上发射。
type streamEmitter struct {
	modes        map[run.StreamMode]bool
	runID        string
	agentID      string
	eventsCh     chan<- run.BaseRunOutputEvent
	output       *RunOutput
	sequence     int
	turn         int
	contentSoFar string
}

func (e *streamEmitter) enabled(mode run.StreamMode) bool {
	if e.modes[mode] {
		return true
	}
	if e.modes[run.StreamDebug] {
		return mode == run.StreamCheckpoints || mode == run.StreamTasks
	}
	return false
}

// emit records first, then sends; a cancelled context surfaces its error to the
// caller, exactly as the pre-slice-31 content path did.
// emit 先记账再上通道；ctx 取消时把错误交回调用方，与切片 31 之前的 content 路径同一手语义。
func (e *streamEmitter) emit(ctx context.Context, evt run.BaseRunOutputEvent) error {
	e.output.appendEvent(evt)

	select {
	case e.eventsCh <- evt:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// content emits the Messages family. Chunk boundaries and sequence numbering are
// unchanged from the slice-23 behaviour line; only the family gate is new.
// content 发射 Messages 族。分块边界与序列号与切片 23 的行为线一致，新增的只有族门。
func (e *streamEmitter) content(ctx context.Context, content string) error {
	if !e.enabled(run.StreamMessages) {
		return nil
	}

	evt := run.NewRunContentEvent(e.runID, e.agentID, string(types.RoleAssistant), content, e.sequence)
	e.sequence++
	return e.emit(ctx, evt)
}

// taskStarted emits the Tasks family opening event for one dispatched call: the node
// is the tool function name and the input is the argument text the model produced.
// taskStarted 发射 Tasks 族的开始事件：node 是工具函数名，input 是模型给出的原始参数文本。
func (e *streamEmitter) taskStarted(ctx context.Context, call types.ToolCall) error {
	if !e.enabled(run.StreamTasks) {
		return nil
	}
	return e.emit(ctx, run.NewNodeStartedEvent(e.runID, call.Function.Name, 1, call.Function.Arguments))
}

// taskFinished closes one call: the handler error decides the event kind, never the
// message text. attempt stays 1 because the agent path has no per-node retry.
// taskFinished 结束一个调用：事件种类由 handler 交回的 error 决定，绝不由消息文本决定。
// attempt 恒为 1，因为 agent 路径没有逐调用重试。
func (e *streamEmitter) taskFinished(ctx context.Context, res toolResult) error {
	if !e.enabled(run.StreamTasks) {
		return nil
	}
	if res.err != nil {
		return e.emit(ctx, run.NewTaskErrorEvent(e.runID, res.node, res.message))
	}
	return e.emit(ctx, run.NewNodeCompletedEvent(e.runID, res.node, 1, res.message))
}

// turnCompleted emits the per-turn families after one model turn finished and its
// assistant message was recorded (kernel.go OnStep's moment): Updates carries the
// turn's content delta, Values the running state (content-so-far including this
// turn, plus the memory message count), Checkpoints a snapshot of that state.
// Within one turn the order is pinned: delta, then state, then its checkpoint.
// turnCompleted 在一次模型回合完成且其 assistant 消息记账后（kernel.go OnStep 的时刻）
// 发射逐回合族：Updates 携带该回合的内容增量，Values 携带运行态（含本回合的
// content-so-far 与 Memory 消息数），Checkpoints 对该状态做快照。同一回合内次序钉死：
// 先增量、后全量、再快照。
func (e *streamEmitter) turnCompleted(ctx context.Context, resp *types.ModelResponse, messageCount int) error {
	e.turn++
	e.contentSoFar += resp.Content
	if e.enabled(run.StreamUpdates) {
		patch := map[string]any{"turn": e.turn, "delta": resp.Content}
		if err := e.emit(ctx, run.NewStateUpdateEvent(e.runID, "", patch)); err != nil {
			return err
		}
	}
	if e.enabled(run.StreamValues) {
		if err := e.emit(ctx, run.NewStateUpdateEvent(e.runID, "", e.snapshot(messageCount))); err != nil {
			return err
		}
	}
	if e.enabled(run.StreamCheckpoints) {
		label := fmt.Sprintf("turn-%d", e.turn)
		if err := e.emit(ctx, run.NewCheckpointEvent(e.runID, label, e.snapshot(messageCount))); err != nil {
			return err
		}
	}
	return nil
}

// snapshot builds one running-state payload: the turn ordinal, the content so far
// (including the turn just completed) and the memory message count. A fresh map per
// event, so the Values event and its Checkpoint never alias one payload.
// snapshot 构造一份运行态载荷：回合序数、content-so-far（含刚完成的回合）与 Memory
// 消息数。每次新 map，Values 事件与其 Checkpoint 永不共享同一份载荷。
func (e *streamEmitter) snapshot(messageCount int) map[string]any {
	return map[string]any{"turn": e.turn, "content": e.contentSoFar, "messages": messageCount}
}

// turnObserver adapts the kernel's per-turn hook (a void signature) to the emitter:
// the memory message count is read at the same moment, after kernel.go recorded the
// assistant message. A turn event dropped on a cancelled context is cancellation's
// consequence — the run is ending and done.Err/StopReason already report it; the
// hook has no error channel and adding one would touch the zero-change runner.
// turnObserver 把内核的逐回合钩子（void 签名）接到发射门：消息数在同一时刻读取
// （kernel.go 已把 assistant 记账之后）。ctx 取消时被丢弃的回合事件是取消的后果——
// 运行正在结束，终态错误已由 done.Err/StopReason 报告；钩子没有错误通道，
// 加一个就要动零改动的 runner。
func (e *streamEmitter) turnObserver(ctx context.Context, messageCount func() int) func(*types.ModelResponse) {
	return func(resp *types.ModelResponse) {
		_ = e.turnCompleted(ctx, resp, messageCount())
	}
}

// customWriterKey is the context key under which a run installs its custom writer.
// customWriterKey 是运行安装 custom writer 的 context 键。
type customWriterKey struct{}

// customWriter serializes handler-originated custom emissions: tool handlers run
// concurrently, so the whole emit (ledger append + channel send) happens under one
// mutex, keeping ledger order identical to channel order. Only this path locks —
// during a tool batch the kernel goroutine is parked in the batch join, so custom
// writes never overlap content/task emissions.
// customWriter 把 handler 发起的 custom 发射串行化：工具 handler 并发运行，所以整个
// emit（记账+上通道）都在一把锁内，保证账面序与通道序一致。只有这条路径持锁——工具批次
// 执行期间内核协程停在批次汇合处，custom 写入与 content/task 发射天然不相交。
type customWriter struct {
	emitter *streamEmitter
	ctx     context.Context
	mu      sync.Mutex
}

func withCustomWriter(ctx context.Context, emitter *streamEmitter) context.Context {
	return context.WithValue(ctx, customWriterKey{}, &customWriter{emitter: emitter, ctx: ctx})
}

// WriteCustomEvent writes one custom event into the run that installed a custom
// writer on ctx — a streaming run with StreamCustom selected, called from inside a
// tool handler (the agent path's "node body"). Without such a writer the call fails
// closed with an error instead of silently dropping the event. The subtype/data pair
// lands on the wire as run.CustomEvent{Subtype, Data}.
// WriteCustomEvent 向 ctx 上安装了 custom writer 的那次运行写入一条自定义事件——
// 即选中 StreamCustom 的流式运行、从工具 handler 体内（agent 路径的「节点体」）调用。
// 没有 writer 时调用 fail-closed 返回错误，绝不静默吞掉事件。subtype/data 以
// run.CustomEvent{Subtype, Data} 的线形状落地。
func WriteCustomEvent(ctx context.Context, subtype string, data any) error {
	if ctx == nil {
		return errors.New("agent: no custom event writer on the context (StreamCustom not selected)")
	}
	w, _ := ctx.Value(customWriterKey{}).(*customWriter)
	if w == nil {
		return errors.New("agent: no custom event writer on the context (StreamCustom not selected)")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.emitter.custom(w.ctx, subtype, data)
}

// custom emits the Custom family. The writer is only ever installed when the family
// is selected (absence by construction, not a filter), so the gate here is the
// second half of that doctrine, not a licence to install loosely.
// custom 发射 Custom 族。writer 只在该族被选中时安装（构造性缺席，不是过滤器），
// 这里的族门是该纪律的另一半，不是宽松安装的许可。
func (e *streamEmitter) custom(ctx context.Context, subtype string, data any) error {
	if !e.enabled(run.StreamCustom) {
		return nil
	}
	return e.emit(ctx, run.NewCustomEvent(e.runID, subtype, data))
}

// taskToolExecutor wraps the shared tool batch so a StreamTasks run reports each
// dispatched call as a task: started events go out on the kernel's own coroutine before
// any goroutine is spawned, closing events follow the batch merge. A call cut off by
// ToolCallLimit never reaches Execute, so its silence is construction, not a filter.
// started 在派发任何 goroutine 之前由内核协程按声明序发出，结束事件在批次汇合之后按结果序
// 发出；被 ToolCallLimit 截断的调用到不了 Execute，因此「不发声」是构造后果而不是过滤结果。
type taskToolExecutor struct {
	agent   *Agent
	emitter *streamEmitter
}

func (e *taskToolExecutor) Execute(ctx context.Context, calls []types.ToolCall) ([]runner.ToolCallOutcome, error) {
	if e.emitter.enabled(run.StreamCustom) {
		ctx = withCustomWriter(ctx, e.emitter)
	}

	for _, call := range calls {
		if err := e.emitter.taskStarted(ctx, call); err != nil {
			return nil, err
		}
	}

	outcomes, results := e.agent.runToolBatch(ctx, calls)

	for _, res := range results {
		if err := e.emitter.taskFinished(ctx, res); err != nil {
			return nil, err
		}
	}
	return outcomes, nil
}

// streamExecutor returns the kernel's tool executor for this selection: a run with
// Tasks (or Debug, whose union opens the Tasks gate) gets the event-wrapped batch,
// and a run with Custom gets the same wrapper for the ctx-installed writer; a
// Messages-only run keeps the executor it had before these slices and cannot drift.
// streamExecutor 按本次模式选择返回内核的工具执行器：选中 Tasks（或 Debug——其并集打开
// Tasks 门）的运行换上带事件发射的批次包装，选中 Custom 的运行经同一包装安装 ctx writer；
// Messages-only 的运行沿用这些切片之前的执行器，不可能漂移。
func (e *streamEmitter) streamExecutor(a *Agent) runner.ToolExecutor {
	if !e.enabled(run.StreamTasks) && !e.enabled(run.StreamCustom) {
		return nil
	}
	return &taskToolExecutor{agent: a, emitter: e}
}
