package run

import "errors"

// StreamMode 选择一次流式运行吐出的事件族（母约 §5 G4，借鉴 LangGraph stream_mode）。
// 切片 23 只交付协议层：常量、序数与 ErrUnsupportedStreamMode；各模式的运行期语义
// （谁发事件、何时发）由各自的生产者接线片钉（S23-SPEC-1）。
type StreamMode int

const (
	// StreamValues 吐每步后的完整状态。
	StreamValues StreamMode = iota
	// StreamUpdates 吐各节点返回的增量。
	StreamUpdates
	// StreamMessages 吐 token 级 LLM 消息（今天 run_content 的归属）。
	StreamMessages
	// StreamTasks 吐任务开始/结束（含结果与错误）。
	StreamTasks
	// StreamCheckpoints 吐检查点事件。
	StreamCheckpoints
	// StreamDebug 吐 checkpoints + tasks 的并集。
	StreamDebug
	// StreamCustom 吐节点内主动写入的自定义事件。
	StreamCustom

	// 六个新事件种类的 wire 名称。旧常量 EventTypeRunContent /
	// EventTypeRunCompleted 原样保留在 events.go，零改动。
	EventTypeNodeStarted   = "node_started"
	EventTypeNodeCompleted = "node_completed"
	EventTypeTaskError     = "task_error"
	EventTypeCheckpoint    = "checkpoint"
	EventTypeStateUpdate   = "state_update"
	EventTypeCustom        = "custom"
)

// ErrUnsupportedStreamMode 在选择了尚未接线生产者的模式时由 Agent.RunStreamMode
// 返回（fail-closed：不静默降级成零事件流）。
var ErrUnsupportedStreamMode = errors.New("run: stream mode has no wired producer")
