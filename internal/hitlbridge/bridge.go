// Package hitlbridge 是 graph 引擎与会话事件侧车（pkg/hno/session/sidecar）的
// 接线层：共享服务落 internal/（仓库惯例），公共暴露留第 3 片。
//
// 职责三条（G7 第 2 片的「跨进程恢复接线」）：
//  1. Capture：把一次 *Suspension（切片 27 形状）连同伴答账目转为侧车持久化
//     记录；事件流追加走 fail-open，挂起记录保存走 fail-closed；
//  2. Install：从侧车记录重建 *Suspension 并装回一张重建的图（pending 恢复
//     入口），新进程由此可直接 Resume；
//  3. 不引入反向依赖：本包 import graph 与 sidecar，二者互不 import。
//
// 图不序列化：跨进程恢复 = 快照字节水合进全新侧车 + 同一 builder 声明重建全新
// 图实例，恢复搬动的只有挂起态——这是唯一诚实的重建语义。
package hitlbridge

import (
	"context"
	"errors"
	"fmt"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
	"github.com/rexleimo/agno-go/pkg/hno/session/sidecar"
)

// Capture 把一次挂起存入侧车：挂起记录 fail-closed（错误原样上交，丢失挂起 =
// 审批永远悬死），事件流 fail-open（追加错误由本层吞掉，不阻塞主路径）。
// 引擎账目（seq/steps）经挂起产物的读出口取得——这是 pending 恢复入口的读侧。
func Capture(ctx context.Context, sc sidecar.Sidecar, sessionID, runKey string, susp *graph.Suspension) error {
	seq, steps := susp.ResumeAccount()
	rec := sidecar.SuspendedRun{
		SessionID: sessionID,
		RunKey:    runKey,
		Seq:       seq,
		Steps:     steps,
		Values:    map[string]any{},
		Completed: append([]string(nil), susp.Completed...),
	}
	for k, v := range susp.Values {
		rec.Values[k] = v
	}
	for _, w := range susp.Interrupts {
		rec.Waiting = append(rec.Waiting, sidecar.WaitingNodeRecord{
			Node: w.Node,
			In:   w.In,
			Interrupt: sidecar.InterruptRecord{
				InterruptID:    w.Interrupt.InterruptID,
				Message:        w.Interrupt.Message,
				ResponseSchema: w.Interrupt.ResponseSchema,
				Payload:        w.Interrupt.Payload,
				Mode:           int(w.Interrupt.Mode),
			},
		})
	}
	if err := sc.SaveSuspendedRun(ctx, rec); err != nil {
		return fmt.Errorf("hitlbridge: save suspended run (fail-closed): %w", err)
	}
	for _, w := range susp.Interrupts {
		_, _ = sc.AppendEvent(ctx, sessionID, sidecar.Event{
			RunKey: runKey,
			Kind:   "hitl.interrupt",
			Payload: map[string]any{
				"interrupt_id": w.Interrupt.InterruptID,
				"node":         w.Node,
				"mode":         int(w.Interrupt.Mode),
			},
		})
	}
	return nil
}

// ResumeSaved 从侧车记录恢复：读回挂起记录、重建 *Suspension、安装到一张
// 重建的图上，并执行一次 Resume。挂起记录缺失或损坏 → 上交错误（fail-closed）；
// 恢复完成（无新挂起）→ 清除挂起记录并补一条 hitl.resumed 事件（fail-open）。
// 返回的 res/err 就是切片 27 Resume 的原生产物（ErrNothingToResume 幂等语义
// 与「校验失败挂起保留」原样保留，可修正后重试）。
func ResumeSaved(ctx context.Context, sc sidecar.Sidecar, g *graph.Graph, sessionID, runKey string, responses map[string]any) (*graph.Result, error) {
	rec, ok, err := sc.SuspendedRun(ctx, sessionID, runKey)
	if err != nil {
		return nil, fmt.Errorf("hitlbridge: load suspended run (fail-closed): %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("hitlbridge: %w: %s/%s", sidecar.ErrNotFound, sessionID, runKey)
	}
	if err := Install(g, rec); err != nil && !errors.Is(err, graph.ErrPendingInstalled) {
		// 已装过（如同进程内上一次 Resume 只是校验失败、挂起原样保留）不算错；
		// 其余安装失败上交（fail-closed）。
		return nil, err
	}
	res, resumeErr := g.Resume(ctx, responses)
	if resumeErr == nil {
		_ = sc.ClearSuspendedRun(ctx, sessionID, runKey)
		_, _ = sc.AppendEvent(ctx, sessionID, sidecar.Event{
			RunKey:  runKey,
			Kind:    "hitl.resumed",
			Payload: map[string]any{"interrupt_ids": interruptIDs(rec)},
		})
	}
	return res, resumeErr
}

// Install 把侧车记录重建为 *Suspension 并装回图（pending 恢复入口的写侧）：
// 等待项、已完成读数与引擎账目（seq/steps）全部按记录原值重建——账目清零回装
// 会让检查点 Seq 跳号、步数预算重置（切片 27 D7/D12 的跨进程版破）。
func Install(g *graph.Graph, rec sidecar.SuspendedRun) error {
	susp := &graph.Suspension{
		Values:    map[string]any{},
		Completed: append([]string(nil), rec.Completed...),
	}
	susp.SetResumeAccount(rec.Seq, rec.Steps)
	for k, v := range rec.Values {
		susp.Values[k] = v
	}
	for _, w := range rec.Waiting {
		susp.Interrupts = append(susp.Interrupts, graph.WaitingNode{
			Node: w.Node,
			In:   w.In,
			Interrupt: graph.Interrupt{
				InterruptID:    w.Interrupt.InterruptID,
				Message:        w.Interrupt.Message,
				ResponseSchema: w.Interrupt.ResponseSchema,
				Payload:        w.Interrupt.Payload,
				Mode:           graph.InterruptMode(w.Interrupt.Mode),
			},
		})
	}
	return graph.RestorePending(g, susp)
}

func interruptIDs(rec sidecar.SuspendedRun) []string {
	ids := make([]string, 0, len(rec.Waiting))
	for _, w := range rec.Waiting {
		ids = append(ids, w.Interrupt.InterruptID)
	}
	return ids
}
