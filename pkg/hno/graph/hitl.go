// HITL 中断/恢复缝（母约 §9.2 G7）。节点请求中断走哨兵错误：RequestInterrupt 返回的
// 载体作为 Node.Run 的 error 穿过既有 queueItem.err 通道抵达唯一消费者（生产者零写
// 调度器字段，零锁不破，Node.Run 签名零改动）。挂起以 *Suspension 经 error 通道交出
// （res 恒为 nil），errors.Is(ErrSuspended) / errors.As(*Suspension) 辨认，与"跑完了
// 的 Result"类型上不可混淆（无半截 Result）。
package graph

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math/rand"
	"slices"
	"time"
)

// InterruptMode 决定恢复时等待节点如何消费人类的响应。
type InterruptMode int

const (
	// ResumeRerun 重入：等待节点带着响应重新执行（默认档）。
	ResumeRerun InterruptMode = iota
	// ResumeHandoff 交接：响应直接成为等待节点的输出喂给后继，节点不再执行。
	ResumeHandoff
)

// Interrupt 是一次 HITL 中断请求（母约 §9.2 形状）；Mode 是草图四字段外由本片补出的调度语义。
type Interrupt struct {
	InterruptID    string
	Message        string
	ResponseSchema map[string]any // JSON Schema 子集：type=object + required + properties.type（见 validateResponse）
	Payload        any
	Mode           InterruptMode
}

// ErrSuspended 标记"这次 Run 挂起等待人类"，由 *Suspension 的 Is 供 errors.Is 辨认。
var ErrSuspended = errors.New("graph: run suspended awaiting human response")

// ErrNothingToResume 表示没有可恢复的挂起（从未挂起/已恢复完）：重复 Resume 是幂等 no-op。
var ErrNothingToResume = errors.New("graph: nothing to resume")

// ErrInvalidResponse 表示恢复响应未通过 ResponseSchema 校验；挂起原样保留，可修正后重试。
var ErrInvalidResponse = errors.New("graph: response does not match interrupt schema")

// RequestInterrupt 构造一次中断请求；节点把它作为 Run 的 error 返回即请求挂起（不可重试的终态）。
func RequestInterrupt(i Interrupt) error { return &interruptSignal{interrupt: i} }

// interruptSignal 是 Interrupt 的错误载体：恒不写调度器状态（消费者才写）、恒不被重试。
type interruptSignal struct {
	interrupt Interrupt
}

func (s *interruptSignal) Error() string {
	return fmt.Sprintf("graph: node requests human input: %s", s.interrupt.Message)
}

// interruptFrom 把一个错误还原为中断载体（errors.As 口径，容忍节点侧 %w 包装）。
func interruptFrom(err error) (*interruptSignal, bool) {
	var sig *interruptSignal
	return sig, errors.As(err, &sig)
}

// WaitingNode 是一个等待人类响应的节点：Rerun 用 In 原样重入，Handoff 把响应当作该节点的输出。
type WaitingNode struct {
	Node      string
	In        any
	Interrupt Interrupt
}

// Suspension 是一次挂起的可观察产物，经 error 通道交出。
type Suspension struct {
	// Interrupts 是同时挂起的等待项（并行分支各自中断时多于一个），按挂起判定次序排列。
	Interrupts []WaitingNode
	// Values/Completed 是挂起前已完成节点的公共读数（与 Result 同形）。
	Values    map[string]any
	Completed []string
	// seq/steps 是引擎续跑用的内部账目：Resume 延续同一份预算与提交序号；跨进程重建不属本片。
	seq   int
	steps int
}

func (s *Suspension) Error() string {
	ids := make([]string, len(s.Interrupts))
	for i, w := range s.Interrupts {
		ids[i] = w.Interrupt.InterruptID
	}
	return fmt.Sprintf("graph: run suspended awaiting human response: interrupt(s) %v", ids)
}

// Is 使 errors.Is(err, ErrSuspended) 对挂起错误成立。
func (s *Suspension) Is(target error) bool { return target == ErrSuspended }

// suspensionFrom 把一个错误还原为挂起产物。
func suspensionFrom(err error) (*Suspension, bool) {
	var susp *Suspension
	return susp, errors.As(err, &susp)
}

// responsesKeyType 携带本次 Resume 的响应穿过 ctx，重入体经 InterruptResponse 读取。
type responsesKeyType struct{}

// InterruptResponse 返回本次恢复中调用方为 interruptID 提供的响应；普通 Run 的 ctx 恒查不到值。
func InterruptResponse(ctx context.Context, interruptID string) (any, bool) {
	m, _ := ctx.Value(responsesKeyType{}).(map[string]any)
	v, ok := m[interruptID]
	return v, ok
}

// Resume 恢复一次挂起的执行（母约 §9.2 的签名）。语义三条：任一响应未过其 ResponseSchema
// → 包 ErrInvalidResponse 且挂起原样保留（节点保持 Waiting）；无挂起或响应集不恰好覆盖
// 全部待答中断（多给/少给）→ ErrNothingToResume / 点名错误；通过后按各 Interrupt.Mode 归
// Rerun（带响应重入）/Handoff（响应即输出喂后继）。Resume 自身可再次挂起；同一张图上的
// Run/Resume 调用链不并发（builder 同一口径）。
func (g *Graph) Resume(ctx context.Context, responses map[string]any) (*Result, error) {
	if g.pending == nil {
		return nil, ErrNothingToResume
	}
	susp := g.pending
	known := map[string]bool{}
	for _, w := range susp.Interrupts {
		resp, ok := responses[w.Interrupt.InterruptID]
		if !ok {
			return nil, fmt.Errorf("graph: missing response for interrupt %q: %w", w.Interrupt.InterruptID, ErrInvalidResponse)
		}
		if err := validateResponse(w.Interrupt.ResponseSchema, resp); err != nil {
			return nil, fmt.Errorf("graph: interrupt %q: %w", w.Interrupt.InterruptID, err)
		}
		known[w.Interrupt.InterruptID] = true
	}
	for id := range responses {
		if !known[id] {
			return nil, fmt.Errorf("graph: response for unknown interrupt %q", id)
		}
	}
	g.pending = nil
	s := g.newScheduler(ctx, susp.seq, susp.steps)
	// 防御性拷贝：旧 Suspension 是调用方仍持有的产物，续跑不得改写它的视图（S19-STD-1 同族）。
	s.result.values = maps.Clone(susp.Values)
	s.result.completed = slices.Clone(susp.Completed)
	s.ctx = context.WithValue(ctx, responsesKeyType{}, responses)
	// 冲刷器先于任何 record 启动：Async 档的提交通道由 beginCommits 创建。
	s.beginCommits()
	for _, w := range susp.Interrupts {
		resp := responses[w.Interrupt.InterruptID]
		switch w.Interrupt.Mode {
		case ResumeHandoff:
			// 交接：响应即该节点的输出，走与完成同一条记账路径，节点体不再执行。
			item := queueItem{name: w.Node, out: resp, in: w.In}
			s.complete(item)
			if err := s.record(item); err != nil {
				return nil, err
			}
		default: // ResumeRerun（含零值）
			s.pending = append(s.pending, activation{name: w.Node, in: w.In})
		}
	}
	res, runErr := s.consume()
	if susp2, _ := suspensionFrom(runErr); susp2 != nil {
		g.pending = susp2
	}
	return s.finishCommits(res, runErr)
}

// newScheduler 从指定的续跑账目重建调度器（Run 与 Resume 共用的构造点）。屏障在进调度时
// 从快照算一次：运行中改图不能给已开始的这次执行增删屏障（「取一次值」不变量）。
func (g *Graph) newScheduler(ctx context.Context, seq, steps int) *scheduler {
	p := g.capturePlan()
	waits, feeds := joinBarriers(p.edges)
	sendSourcesOf, sendTargetsOf, sendTargetList := joinSendTables(p.edges)
	return &scheduler{
		plan:           p,
		ctx:            ctx,
		queue:          make(chan queueItem),
		done:           make(chan struct{}),
		result:         &Result{values: make(map[string]any)},
		stepLimit:      g.cfg.stepLimit,
		maxConcurrency: g.cfg.maxConcurrency,
		waits:          waits,
		feeds:          feeds,
		joinCollected:  map[string]map[string]any{},
		rnd:            rand.New(rand.NewSource(time.Now().UnixNano())),
		sendPending:    map[string]int{},
		sendCollected:  map[string][]any{},
		sendFired:      map[string]bool{},
		sendSourcesOf:  sendSourcesOf,
		sendTargetsOf:  sendTargetsOf,
		sendTargetList: sendTargetList,
		durability:     g.cfg.durability,
		checkpointer:   g.cfg.checkpointer,
		seq:            seq,
		steps:          steps,
	}
}

// validateResponse 按本引擎解释的 JSON Schema 子集校验恢复响应。子集恰为：顶层
// type=="object"（缺省同义）+ required（键存在性）+ properties[name].type（值形状）。
// 未声明的多余键放行；schema 为空放行；声明了子集之外的东西（顶层 type 非 object、
// property type 未知、property spec 不是对象）→ 报不支持而不是放行（fail-closed）。
func validateResponse(schema map[string]any, resp any) error {
	if len(schema) == 0 {
		return nil
	}
	if t, ok := schema["type"]; ok && t != "object" {
		ts, _ := t.(string)
		return fmt.Errorf("%w: unsupported ResponseSchema type %q: only \"object\" schemas are interpreted", ErrInvalidResponse, ts)
	}
	respMap, ok := resp.(map[string]any)
	if !ok {
		return fmt.Errorf("%w: response must be an object, got %T", ErrInvalidResponse, resp)
	}
	if req, ok := schema["required"]; ok {
		names, err := requiredNames(req)
		if err != nil {
			return err
		}
		for _, n := range names {
			if _, ok := respMap[n]; !ok {
				return fmt.Errorf("%w: missing required field %q", ErrInvalidResponse, n)
			}
		}
	}
	props, ok := schema["properties"]
	if !ok {
		return nil
	}
	pm, ok := props.(map[string]any)
	if !ok {
		return fmt.Errorf("%w: unsupported ResponseSchema properties: expected an object", ErrInvalidResponse)
	}
	names := slices.Sorted(maps.Keys(pm))
	for _, n := range names {
		spec, ok := pm[n].(map[string]any)
		if !ok {
			return fmt.Errorf("%w: unsupported property spec for %q: expected an object", ErrInvalidResponse, n)
		}
		want, _ := spec["type"].(string)
		if want == "" {
			continue
		}
		if err := checkShape(n, want, respMap[n]); err != nil {
			return err
		}
	}
	return nil
}

// requiredNames 接受 []string 与 JSON 解码产出的 []any-of-string 两种形状。
func requiredNames(req any) ([]string, error) {
	switch list := req.(type) {
	case []string:
		return list, nil
	case []any:
		names := make([]string, len(list))
		for i, v := range list {
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("%w: unsupported required entry %T: expected a string", ErrInvalidResponse, v)
			}
			names[i] = s
		}
		return names, nil
	default:
		return nil, fmt.Errorf("%w: unsupported required: expected an array of strings", ErrInvalidResponse)
	}
}

// checkShape 校验单个属性值的形状；缺失的属性只由 required 管，这里跳过。
func checkShape(name, want string, v any) error {
	ok := false
	switch want {
	case "string":
		_, ok = v.(string)
	case "number", "integer":
		switch v.(type) {
		case int, int32, int64, float32, float64:
			ok = true
		}
	case "boolean":
		_, ok = v.(bool)
	case "object":
		_, ok = v.(map[string]any)
	case "array":
		_, ok = v.([]any)
	case "null":
		ok = v == nil
	default:
		return fmt.Errorf("%w: unsupported property type %q for %q", ErrInvalidResponse, want, name)
	}
	if !ok {
		return fmt.Errorf("%w: field %q should be %s, got %T", ErrInvalidResponse, name, want, v)
	}
	return nil
}
