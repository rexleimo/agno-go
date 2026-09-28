// pending 恢复入口（母约 §9.2 G7 第 2 片）。切片 27 契约把跨进程恢复登记为
// 「经 Graph 重建 + pending 恢复入口」，且其引擎核心「本片不暴露内部字段」——
// 即该入口留给本片补上。形状以「挂起账目读出口 + 安装口 + 不覆盖守卫」为限，
// 是切片 27 契约预留面的最小实现；更丰富的恢复 API 属后续契约：
//   - 读侧：Suspension.ResumeAccount 取挂起时的引擎账目（检查点 seq 续号 +
//     步数预算累计）。两者是未导出字段，进程外无从读取，侧车要持久化它们
//     就必须有引擎侧读出口；
//   - 写侧：RestorePending 把从侧车记录重建的 Suspension 安装到一张全新重建的
//     图上（图不序列化：新进程用同一 builder 声明重建图实例），新进程由此
//     直接走切片 27 的既有 Resume 语义。
//
// 除该入口外不为测试导出任何 pending 内部态。
package graph

import "errors"

// ResumeAccount 返回挂起时的引擎账目：检查点序号续号与已消费步数。
// 侧车持久化它们，跨进程恢复时经 RestorePending 原值装回（预算跨恢复累计、
// 检查点 Seq 不跳号——切片 27 D7/D12 的语义在重建后保持）。
func (s *Suspension) ResumeAccount() (seq, steps int) {
	return s.seq, s.steps
}

// SetResumeAccount 装回引擎账目（读出口的对偶，安装前由重建方调用）。
func (s *Suspension) SetResumeAccount(seq, steps int) {
	s.seq = seq
	s.steps = steps
}

// ErrPendingInstalled 表示图上已有挂起、本次安装未发生。接线层用它区分
// 「已装过（可继续 Resume，如上一次尝试只是校验失败）」与真正的无处可恢复；
// 它不被 ErrSuspended/ErrNothingToResume 的 errors.Is 链吸收。
var ErrPendingInstalled = errors.New("graph: pending suspension already installed")

// RestorePending 把一张从侧车记录重建的挂起安装到（重建出来的）图上，
// 使新进程可以对该图直接 Resume。图上已有挂起时返回 ErrPendingInstalled
// （不覆盖——覆盖可能冲掉重悬链产生的新挂起），由接线层决定是否视为良性。
func RestorePending(g *Graph, susp *Suspension) error {
	if g.pending != nil {
		return ErrPendingInstalled
	}
	g.pending = susp
	return nil
}
