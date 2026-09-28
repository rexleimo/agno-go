package graph

import "context"

// Send 是节点在运行时决定的一次扇出派发（母约 §6）：Node 命名本次派发的目标节点，
// In 是该次派发独有的输入。数量与参数都在运行期决定——静态边做不到的正是这一半。
type Send struct {
	Node string
	In   any
}

// Sender 是节点的可选扩展面：实现它的节点在常规输出之外，于同一次激活内派发若干 Send。
//
// 方法名刻意不是 Run：Node.Run 已占用两返回值签名，Go 的方法集按签名精确匹配，
// 一个类型不可能同时实现 Run(ctx,in)(any,error) 与 Run(ctx,in)(any,[]Send,error)——
// 母约 §6 草图按字面落地会让每个 Sender 都无法经 AddNode 进入节点表。
// 调度器在生产者路径对每次激活做类型断言：断言成功走 SendRun（其返回的 sends 被派发），
// 断言失败走 Node.Run（没有任何派发）——普通节点零改变。
type Sender interface {
	SendRun(ctx context.Context, in any) (out any, sends []Send, err error)
}

// senderFunc 同时实现 Node 与 Sender 两个视角：Run 是丢弃 sends 的常规视图
// （引擎对 Sender 节点从不走它），SendRun 是调度器断言到的权威视图。
type senderFunc struct {
	name string
	fn   func(context.Context, any) (any, []Send, error)
}

func (n senderFunc) Name() string { return n.name }

func (n senderFunc) Run(ctx context.Context, in any) (any, error) {
	out, _, err := n.fn(ctx, in)
	return out, err
}

func (n senderFunc) SendRun(ctx context.Context, in any) (any, []Send, error) {
	return n.fn(ctx, in)
}

// SenderFunc 用函数构造一个支持运行期扇出的 Node。
func SenderFunc(name string, fn func(ctx context.Context, in any) (any, []Send, error)) Node {
	return senderFunc{name: name, fn: fn}
}

// AddJoinSend 声明：等 source 名下本轮派发的全部 Send 激活完成后，激活 target 一次
// （R19-Q2 裁决 OPT-C，2026-09-28）。构建期不数前驱名、不经 §3.5 第 6 项——它声明的
// 是派发记账的归零，不是静态前驱集合；运行期屏障按「source 的 pending Send 计数归零」
// 判凑齐。凑不齐（本轮没有任何 Send 指名 source，或屏障被步数安全阀截断）不构成新的
// 静默成功路径：target 本轮不被激活，Run 沿 R19-Q1 的既有形状收场。
//
// 该声明向可达性邻接表贡献 source→target 一条边（R19-Q2 实测 [A2]：只被运行期 Send
// 喂养的节点在纯声明边上不可达），并且 source 本身被视为可达——Send 的目标名是运行期
// 值，构建期无法反驳「会有 Sender 指名它」（与条件边不计谓词的同一乐观先例）。
// 多条声明允许：同一 source 可声明多个 target（各自在归零时各激活一次）；同一 target
// 可声明多个 source（等全部 source 的计数同时归零才激活一次）。
func (g *Graph) AddJoinSend(source, target string) *Graph {
	return g.declare(edgeJoinSend, source, target, nil)
}

// joinSendTables 把 AddJoinSend 声明收成运行期屏障的三张表（构建期算一次，对偶 joinBarriers）：
// sourcesOf 是「target → 它等待的 source 集合」（按名字去重，重复声明折叠），
// targetsOf 是「source → 目标列表」（按声明顺序，归零时按它激活），
// targetList 是去重后的目标名（声明序，多个 target 同时就绪时激活顺序确定）。
func joinSendTables(edges []edgeDecl) (sourcesOf map[string]map[string]bool, targetsOf map[string][]string, targetList []string) {
	sourcesOf = map[string]map[string]bool{}
	targetsOf = map[string][]string{}
	seen := map[string]bool{}
	for _, e := range edges {
		if e.kind != edgeJoinSend {
			continue
		}
		if sourcesOf[e.to] == nil {
			sourcesOf[e.to] = map[string]bool{}
		}
		if sourcesOf[e.to][e.from] {
			continue
		}
		sourcesOf[e.to][e.from] = true
		targetsOf[e.from] = append(targetsOf[e.from], e.to)
		if !seen[e.to] {
			seen[e.to] = true
			targetList = append(targetList, e.to)
		}
	}
	return sourcesOf, targetsOf, targetList
}
