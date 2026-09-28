// Package sidecar 是会话事件侧车存储（母约 §9 G7 第 2 片，D1=OPT-1 裁决的落地形状，
// 落位 pkg/hno/session/sidecar）。
//
// OPT-1 纪律（D1 裁决材料 §1.5 探针 [A]/[B]/[B2]）：
//   - 侧车与 Session 结构体零耦合：不新增字段、不改四个 marshal 边界的任何一个，
//     事件写入绝不路由过 Storage.Update（Update 每次触摸重打 UpdatedAt，
//     会破坏对外 JSON 的字节恒等）；
//   - 侧车按 (session_id, run_key) 记键，存通用 JSON 记录，不 import 引擎类型
//     （graph ↔ 侧车的适配住在接线层）；
//   - 两类数据、两种失败语义：
//     事件流（观测类，fail-open）—— 写失败不阻塞不污染主会话路径；
//     挂起记录（恢复关键，fail-closed）—— 写失败必须上交，绝不静默丢弃。
package sidecar

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"
)

// 中断档位镜像切片 27 的 InterruptMode（0=Rerun 默认，1=Handoff），
// 用整型 + 命名常量而非 import 引擎包——侧车层保持与引擎类型零耦合。
const (
	ModeRerun   int = 0
	ModeHandoff int = 1
)

var (
	// ErrNotFound 表示请求的挂起记录不存在。
	ErrNotFound = errors.New("sidecar: suspended run not found")
	// ErrInvalidRecord 表示挂起记录缺关键键（session/run/等待项）。
	ErrInvalidRecord = errors.New("sidecar: invalid suspended run record")
)

// Event 是侧车事件流的一条记录（观测类数据）。
type Event struct {
	Seq     int64          `json:"seq"`
	RunKey  string         `json:"run_key,omitempty"`
	Kind    string         `json:"kind"`
	At      time.Time      `json:"at"`
	Payload map[string]any `json:"payload,omitempty"`
}

// InterruptRecord 是一次待答中断的持久化形状（切片 27 Interrupt 四字段 + Mode
// 的 generic 化）。
type InterruptRecord struct {
	InterruptID    string         `json:"interrupt_id"`
	Message        string         `json:"message,omitempty"`
	ResponseSchema map[string]any `json:"response_schema,omitempty"`
	Payload        any            `json:"payload,omitempty"`
	Mode           int            `json:"mode"`
}

// WaitingNodeRecord 是一个等待人类响应的节点（切片 27 WaitingNode 的持久化形状）。
type WaitingNodeRecord struct {
	Node      string          `json:"node"`
	In        any             `json:"in,omitempty"`
	Interrupt InterruptRecord `json:"interrupt"`
}

// SuspendedRun 是一次挂起的恢复关键持久化记录：重建一张可 Resume 的图所需的
// 全量状态（等待项 + 已完成读数 + 引擎账目）。
type SuspendedRun struct {
	SessionID string              `json:"session_id"`
	RunKey    string              `json:"run_key"`
	Waiting   []WaitingNodeRecord `json:"waiting"`
	Values    map[string]any      `json:"values,omitempty"`
	Completed []string            `json:"completed,omitempty"`
	// Seq/Steps 是挂起时的引擎账目（检查点 Seq 续号、步数预算已消费数），
	// 跨进程装回时原值恢复——预算跨恢复累计、Seq 不跳号。
	Seq       int       `json:"seq"`
	Steps     int       `json:"steps"`
	CreatedAt time.Time `json:"created_at"`
}

// Sidecar 是侧车存储的接口（Checkpointer 先例：接口在本层、实现属调用方）。
// 落地期至少有内存实现；跨进程持久化的真实 DB 适配（独立列/表）是下一步薄层。
type Sidecar interface {
	// AppendEvent 追加一条事件，返回侧车分配的流内序号。fail-open 数据：
	// 调用方（接线层）对错误只记不计，绝不因此阻塞会话/运行路径。
	AppendEvent(ctx context.Context, sessionID string, e Event) (int64, error)
	// Events 返回某会话的完整事件流（按 Seq 升序）——派生视图（UI/调试）。
	Events(ctx context.Context, sessionID string) ([]Event, error)
	// EventsByRun 返回某 run 的事件切片——派生视图。
	EventsByRun(ctx context.Context, sessionID, runKey string) ([]Event, error)
	// PendingInterrupts 返回该会话当前全部待答中断（跨 run 聚合）——审批 UI
	// 的派生视图；持久化形状，不含任何引擎类型。
	PendingInterrupts(ctx context.Context, sessionID string) ([]InterruptRecord, error)
	// SaveSuspendedRun 保存/覆盖一条挂起记录。fail-closed 数据：错误必须上交。
	SaveSuspendedRun(ctx context.Context, rec SuspendedRun) error
	// SuspendedRun 读取一条挂起记录（ok=false 表示不存在）。
	SuspendedRun(ctx context.Context, sessionID, runKey string) (SuspendedRun, bool, error)
	// ClearSuspendedRun 在恢复完成后清除挂起记录（幂等：不存在不算错误）。
	ClearSuspendedRun(ctx context.Context, sessionID, runKey string) error
}

// MemorySidecar 是 Sidecar 的内存实现（MemoryStorage 先例的旁路容器：
// 与会话行零耦合，写入不触碰任何 Session 值）。
type MemorySidecar struct {
	mu        sync.RWMutex
	events    map[string][]Event
	nextSeq   map[string]int64
	suspended map[string]SuspendedRun // 键：sessionID + "\x00" + runKey
}

// NewMemorySidecar 构造内存侧车。
func NewMemorySidecar() *MemorySidecar {
	return &MemorySidecar{
		events:    map[string][]Event{},
		nextSeq:   map[string]int64{},
		suspended: map[string]SuspendedRun{},
	}
}

func suspendedKey(sessionID, runKey string) string { return sessionID + "\x00" + runKey }

// AppendEvent 实现 Sidecar：会话内单调 Seq 由侧车分配，写入不触碰任何 Session 值。
func (m *MemorySidecar) AppendEvent(ctx context.Context, sessionID string, e Event) (int64, error) {
	if sessionID == "" {
		return 0, errors.New("sidecar: empty session id")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextSeq[sessionID]++
	e.Seq = m.nextSeq[sessionID]
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	m.events[sessionID] = append(m.events[sessionID], e)
	return e.Seq, nil
}

// Events 实现 Sidecar：完整事件流，按 Seq 升序。
func (m *MemorySidecar) Events(ctx context.Context, sessionID string) ([]Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Event, len(m.events[sessionID]))
	copy(out, m.events[sessionID])
	return out, nil
}

// EventsByRun 实现 Sidecar：单个 run 的事件切片（与全流同源，Seq 升序）。
func (m *MemorySidecar) EventsByRun(ctx context.Context, sessionID, runKey string) ([]Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Event
	for _, e := range m.events[sessionID] {
		if e.RunKey == runKey {
			out = append(out, e)
		}
	}
	return out, nil
}

// PendingInterrupts 实现 Sidecar：从全部未清除的挂起记录聚合待答中断，
// 按记录创建时间与等待项声明序稳定排序。
func (m *MemorySidecar) PendingInterrupts(ctx context.Context, sessionID string) ([]InterruptRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var recs []SuspendedRun
	for _, rec := range m.suspended {
		if rec.SessionID == sessionID {
			recs = append(recs, rec)
		}
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].CreatedAt.Before(recs[j].CreatedAt) })
	var out []InterruptRecord
	for _, rec := range recs {
		for _, w := range rec.Waiting {
			out = append(out, w.Interrupt)
		}
	}
	return out, nil
}

// SaveSuspendedRun 实现 Sidecar：半截记录拒收（fail-closed 数据不接受缺关键键的形状）。
func (m *MemorySidecar) SaveSuspendedRun(ctx context.Context, rec SuspendedRun) error {
	if rec.SessionID == "" || rec.RunKey == "" || len(rec.Waiting) == 0 {
		return ErrInvalidRecord
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}
	m.suspended[suspendedKey(rec.SessionID, rec.RunKey)] = rec
	return nil
}

// SuspendedRun 实现 Sidecar。
func (m *MemorySidecar) SuspendedRun(ctx context.Context, sessionID, runKey string) (SuspendedRun, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rec, ok := m.suspended[suspendedKey(sessionID, runKey)]
	return rec, ok, nil
}

// ClearSuspendedRun 实现 Sidecar（幂等：不存在不算错误）。
func (m *MemorySidecar) ClearSuspendedRun(ctx context.Context, sessionID, runKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.suspended, suspendedKey(sessionID, runKey))
	return nil
}

// Snapshot 把内存侧车的全部状态序列化为 JSON 字节——侧车记录形状可经通用
// 编码无损往返的证明，也是真实 DB 适配（列/表）将要持有的字节形态。
// Snapshot 属内存实现的试验/传输面，不在 Sidecar 接口上（持久后端自身即持久）。
func (m *MemorySidecar) Snapshot() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	type snapshot struct {
		Events    map[string][]Event      `json:"events"`
		Suspended map[string]SuspendedRun `json:"suspended"`
		NextSeq   map[string]int64        `json:"next_seq"`
	}
	return json.Marshal(snapshot{Events: m.events, Suspended: m.suspended, NextSeq: m.nextSeq})
}

// RestoreSnapshot 从 Snapshot 字节水合一个（通常全新的）内存侧车——模拟重启的
// 恢复路径：新进程拿到的是字节，不是原对象。字节里缺省的字段保持原状不覆盖。
func (m *MemorySidecar) RestoreSnapshot(b []byte) error {
	var snapshot struct {
		Events    map[string][]Event      `json:"events"`
		Suspended map[string]SuspendedRun `json:"suspended"`
		NextSeq   map[string]int64        `json:"next_seq"`
	}
	if err := json.Unmarshal(b, &snapshot); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if snapshot.Events != nil {
		m.events = snapshot.Events
	}
	if snapshot.Suspended != nil {
		m.suspended = snapshot.Suspended
	}
	if snapshot.NextSeq != nil {
		m.nextSeq = snapshot.NextSeq
	}
	return nil
}
