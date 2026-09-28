package agent

import (
	"context"
	"fmt"

	"github.com/rexleimo/agno-go/pkg/hno/run"
	"github.com/rexleimo/agno-go/pkg/hno/runner"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// wiredStreamModes lists the modes with a runtime producer; any other mode fails the
// call before the stream starts, and wiring a new mode means adding its row here plus
// its producer's judgement rows (母约 §5).
// Values/Updates/Checkpoints/Debug/Custom 按图节点或检查点键控，归属 P4 第 3 片。
var wiredStreamModes = map[run.StreamMode]bool{
	run.StreamMessages: true,
	run.StreamTasks:    true,
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
// goroutine that drives the kernel, so ordering is a property of that coroutine.
// streamEmitter 是流式运行把产出值变成事件的唯一出口。它持有按族门控（未选中的族
// 既不记账也不上通道）、记账序列（RunOutput.Events）与内容事件的序列号。发射发生在
// 驱动内核的那条协程上，因此次序是该协程的构造属性，不是锁的产物。
type streamEmitter struct {
	modes    map[run.StreamMode]bool
	runID    string
	agentID  string
	eventsCh chan<- run.BaseRunOutputEvent
	output   *RunOutput
	sequence int
}

func (e *streamEmitter) enabled(mode run.StreamMode) bool {
	return e.modes[mode]
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
	return e.emit(ctx, run.NewNodeStartedEvent(e.runID, call.Function.Name, 1, call.Function.Arguments))
}

// taskFinished closes one call: the handler error decides the event kind, never the
// message text. attempt stays 1 because the agent path has no per-node retry.
// taskFinished 结束一个调用：事件种类由 handler 交回的 error 决定，绝不由消息文本决定。
// attempt 恒为 1，因为 agent 路径没有逐调用重试。
func (e *streamEmitter) taskFinished(ctx context.Context, res toolResult) error {
	if res.err != nil {
		return e.emit(ctx, run.NewTaskErrorEvent(e.runID, res.node, res.message))
	}
	return e.emit(ctx, run.NewNodeCompletedEvent(e.runID, res.node, 1, res.message))
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

// tasksExecutor returns the kernel's tool executor for this selection: only a Tasks
// run gets the event-wrapped batch, so the Messages path keeps the executor it had
// before this slice and cannot drift.
// tasksExecutor 按本次模式选择返回内核的工具执行器：只有选中 Tasks 才换上带事件发射的批次
// 包装，Messages 路径因此沿用本片之前的执行器，不可能漂移。
func (e *streamEmitter) tasksExecutor(a *Agent) runner.ToolExecutor {
	if !e.enabled(run.StreamTasks) {
		return nil
	}
	return &taskToolExecutor{agent: a, emitter: e}
}
