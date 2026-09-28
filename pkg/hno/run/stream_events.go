package run

import (
	"encoding/json"
	"time"
)

// NodeStartedEvent 宣告一个节点（或任务）开始执行。
type NodeStartedEvent struct {
	eventBase
	RunID   string `json:"run_id,omitempty"`
	Node    string `json:"node,omitempty"`
	Attempt int    `json:"attempt,omitempty"`
	Input   any    `json:"input,omitempty"`
}

// NodeCompletedEvent 宣告一个节点执行完成并携带其输出。
type NodeCompletedEvent struct {
	eventBase
	RunID   string `json:"run_id,omitempty"`
	Node    string `json:"node,omitempty"`
	Attempt int    `json:"attempt,omitempty"`
	Output  any    `json:"output,omitempty"`
}

// TaskErrorEvent 报告一次任务执行的失败（含归因位置）。
type TaskErrorEvent struct {
	eventBase
	RunID   string `json:"run_id,omitempty"`
	Node    string `json:"node,omitempty"`
	Message string `json:"message,omitempty"`
}

// CheckpointEvent 记录一次检查点（状态快照的引用由生产者解释）。
type CheckpointEvent struct {
	eventBase
	RunID   string `json:"run_id,omitempty"`
	Label   string `json:"label,omitempty"`
	Payload any    `json:"payload,omitempty"`
}

// StateUpdateEvent 携带一次状态增量（StreamUpdates 的事件族）。
type StateUpdateEvent struct {
	eventBase
	RunID string `json:"run_id,omitempty"`
	Node  string `json:"node,omitempty"`
	Patch any    `json:"patch,omitempty"`
}

// CustomEvent 承载节点内主动写入的自定义载荷（StreamCustom 的事件族）。
type CustomEvent struct {
	eventBase
	RunID   string `json:"run_id,omitempty"`
	Subtype string `json:"subtype,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// 构造函数沿用 NewRunContentEvent 模式：种类与时间戳在构造时盖章。字面量构造的事件
// eventType 为空，其 wire 形状是哑的（会被 decodeEvent 的旧归一化吃掉），所以两层
// 防御缺一不可：构造时盖章 + 序列化侧 canonical 兜底。
func NewNodeStartedEvent(runID, node string, attempt int, input any) *NodeStartedEvent {
	return &NodeStartedEvent{eventBase: eventBase{eventType: EventTypeNodeStarted, timestamp: time.Now().UTC()}, RunID: runID, Node: node, Attempt: attempt, Input: input}
}

func NewNodeCompletedEvent(runID, node string, attempt int, output any) *NodeCompletedEvent {
	return &NodeCompletedEvent{eventBase: eventBase{eventType: EventTypeNodeCompleted, timestamp: time.Now().UTC()}, RunID: runID, Node: node, Attempt: attempt, Output: output}
}

func NewTaskErrorEvent(runID, node, message string) *TaskErrorEvent {
	return &TaskErrorEvent{eventBase: eventBase{eventType: EventTypeTaskError, timestamp: time.Now().UTC()}, RunID: runID, Node: node, Message: message}
}

func NewCheckpointEvent(runID, label string, payload any) *CheckpointEvent {
	return &CheckpointEvent{eventBase: eventBase{eventType: EventTypeCheckpoint, timestamp: time.Now().UTC()}, RunID: runID, Label: label, Payload: payload}
}

func NewStateUpdateEvent(runID, node string, patch any) *StateUpdateEvent {
	return &StateUpdateEvent{eventBase: eventBase{eventType: EventTypeStateUpdate, timestamp: time.Now().UTC()}, RunID: runID, Node: node, Patch: patch}
}

func NewCustomEvent(runID, subtype string, data any) *CustomEvent {
	return &CustomEvent{eventBase: eventBase{eventType: EventTypeCustom, timestamp: time.Now().UTC()}, RunID: runID, Subtype: subtype, Data: data}
}

func (e NodeStartedEvent) MarshalJSON() ([]byte, error) {
	type alias NodeStartedEvent
	payload := struct {
		Event     string `json:"event"`
		CreatedAt int64  `json:"created_at"`
		*alias
	}{
		Event:     canonicalEventType(e.eventBase.eventType, EventTypeNodeStarted),
		CreatedAt: unixSeconds(e.timestamp),
		alias:     (*alias)(&e),
	}
	return json.Marshal(payload)
}

func (e *NodeStartedEvent) UnmarshalJSON(data []byte) error {
	type alias NodeStartedEvent
	aux := struct {
		Event     string `json:"event"`
		CreatedAt int64  `json:"created_at"`
		*alias
	}{alias: (*alias)(e)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if e.eventBase.eventType == "" {
		e.eventBase.eventType = EventTypeNodeStarted
	}
	if aux.CreatedAt > 0 {
		e.eventBase.timestamp = time.Unix(aux.CreatedAt, 0).UTC()
	}
	return nil
}

func (e NodeCompletedEvent) MarshalJSON() ([]byte, error) {
	type alias NodeCompletedEvent
	payload := struct {
		Event     string `json:"event"`
		CreatedAt int64  `json:"created_at"`
		*alias
	}{
		Event:     canonicalEventType(e.eventBase.eventType, EventTypeNodeCompleted),
		CreatedAt: unixSeconds(e.timestamp),
		alias:     (*alias)(&e),
	}
	return json.Marshal(payload)
}

func (e *NodeCompletedEvent) UnmarshalJSON(data []byte) error {
	type alias NodeCompletedEvent
	aux := struct {
		Event     string `json:"event"`
		CreatedAt int64  `json:"created_at"`
		*alias
	}{alias: (*alias)(e)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if e.eventBase.eventType == "" {
		e.eventBase.eventType = EventTypeNodeCompleted
	}
	if aux.CreatedAt > 0 {
		e.eventBase.timestamp = time.Unix(aux.CreatedAt, 0).UTC()
	}
	return nil
}

func (e TaskErrorEvent) MarshalJSON() ([]byte, error) {
	type alias TaskErrorEvent
	payload := struct {
		Event     string `json:"event"`
		CreatedAt int64  `json:"created_at"`
		*alias
	}{
		Event:     canonicalEventType(e.eventBase.eventType, EventTypeTaskError),
		CreatedAt: unixSeconds(e.timestamp),
		alias:     (*alias)(&e),
	}
	return json.Marshal(payload)
}

func (e *TaskErrorEvent) UnmarshalJSON(data []byte) error {
	type alias TaskErrorEvent
	aux := struct {
		Event     string `json:"event"`
		CreatedAt int64  `json:"created_at"`
		*alias
	}{alias: (*alias)(e)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if e.eventBase.eventType == "" {
		e.eventBase.eventType = EventTypeTaskError
	}
	if aux.CreatedAt > 0 {
		e.eventBase.timestamp = time.Unix(aux.CreatedAt, 0).UTC()
	}
	return nil
}

func (e CheckpointEvent) MarshalJSON() ([]byte, error) {
	type alias CheckpointEvent
	payload := struct {
		Event     string `json:"event"`
		CreatedAt int64  `json:"created_at"`
		*alias
	}{
		Event:     canonicalEventType(e.eventBase.eventType, EventTypeCheckpoint),
		CreatedAt: unixSeconds(e.timestamp),
		alias:     (*alias)(&e),
	}
	return json.Marshal(payload)
}

func (e *CheckpointEvent) UnmarshalJSON(data []byte) error {
	type alias CheckpointEvent
	aux := struct {
		Event     string `json:"event"`
		CreatedAt int64  `json:"created_at"`
		*alias
	}{alias: (*alias)(e)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if e.eventBase.eventType == "" {
		e.eventBase.eventType = EventTypeCheckpoint
	}
	if aux.CreatedAt > 0 {
		e.eventBase.timestamp = time.Unix(aux.CreatedAt, 0).UTC()
	}
	return nil
}

func (e StateUpdateEvent) MarshalJSON() ([]byte, error) {
	type alias StateUpdateEvent
	payload := struct {
		Event     string `json:"event"`
		CreatedAt int64  `json:"created_at"`
		*alias
	}{
		Event:     canonicalEventType(e.eventBase.eventType, EventTypeStateUpdate),
		CreatedAt: unixSeconds(e.timestamp),
		alias:     (*alias)(&e),
	}
	return json.Marshal(payload)
}

func (e *StateUpdateEvent) UnmarshalJSON(data []byte) error {
	type alias StateUpdateEvent
	aux := struct {
		Event     string `json:"event"`
		CreatedAt int64  `json:"created_at"`
		*alias
	}{alias: (*alias)(e)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if e.eventBase.eventType == "" {
		e.eventBase.eventType = EventTypeStateUpdate
	}
	if aux.CreatedAt > 0 {
		e.eventBase.timestamp = time.Unix(aux.CreatedAt, 0).UTC()
	}
	return nil
}

func (e CustomEvent) MarshalJSON() ([]byte, error) {
	type alias CustomEvent
	payload := struct {
		Event     string `json:"event"`
		CreatedAt int64  `json:"created_at"`
		*alias
	}{
		Event:     canonicalEventType(e.eventBase.eventType, EventTypeCustom),
		CreatedAt: unixSeconds(e.timestamp),
		alias:     (*alias)(&e),
	}
	return json.Marshal(payload)
}

func (e *CustomEvent) UnmarshalJSON(data []byte) error {
	type alias CustomEvent
	aux := struct {
		Event     string `json:"event"`
		CreatedAt int64  `json:"created_at"`
		*alias
	}{alias: (*alias)(e)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if e.eventBase.eventType == "" {
		e.eventBase.eventType = EventTypeCustom
	}
	if aux.CreatedAt > 0 {
		e.eventBase.timestamp = time.Unix(aux.CreatedAt, 0).UTC()
	}
	return nil
}
