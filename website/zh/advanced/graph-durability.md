# 图持久化 - 检查点档位

控制一张运行中的图何时把已完成节点的事件提交到你的持久化 sink：逐节点同步、
后台异步、或退出时一次落齐。

---

## 概览

图引擎的持久化缝只回答一个问题：**节点完成后，它的事件什么时候到达 sink？**
它只有提交侧 —— 引擎记录发生过什么；（目前）不把检查点读回来。两个声明组合：

```go
g := graph.New(
	graph.WithDurability(graph.DurabilitySync), // 档位
	graph.WithCheckpointer(mySink),             // sink
)
```

- `WithDurability` 三选一。零值即 `DurabilitySync` —— 未声明档位的图按**最安全**
  档执行，不是最快档。
- `WithCheckpointer` 挂 sink。不挂（`nil`）即完全不持久化：图照常跑、不报错。

两者与 `WithStepLimit`、`WithMaxConcurrency` 一样在进入 `Run` 时读取一次 ——
运行中改 builder 改不了本次 Run 的档位。

---

## Checkpointer 契约

```go
type Checkpoint struct {
	Seq    int    // 消费者盖章，从 1 连续递增
	Node   string
	Output any
}

type Checkpointer interface {
	Append(ctx context.Context, cp Checkpoint) error
}
```

这条缝刻意声明在 graph 包内：引擎不依赖任何存储包。把 Postgres、日志文件或
内存切片接进来是你的单方法实现。

条目流是 **append-only** 的：

- 每个完成节点恰一条，携带该次完成时的输出。
- `Seq` 由唯一消费者 goroutine 在提交时机盖章 —— 从 1 起、连续递增、次序即本次
  `Run` 的完成处理次序。条件环下重跑的节点会带着新序号再次提交。
- 这条缝没有读回 API。消费这个流（tail 日志、查表）是 sink 侧的事。

---

## 三档时序

| 档位 | 条目何时到达 sink | 取舍 |
|---|---|---|
| `DurabilitySync`（默认，零值） | 每个节点完成后同步落、且在**下一轮派发之前** | 最安全：任何后继启动前，前驱条目已稳稳落盘。代价是每节点一次 sink 往返。 |
| `DurabilityAsync` | 后台冲刷 goroutine；`Run` 返回前全部冲完 | 派发永不被挡。条目不丢、`Seq` 不乱，但后继可能已经在跑而前驱条目还在途 —— 不要依赖 sink 可见性做顺序判断。 |
| `DurabilityExit` | 内存中积累；`Run` 返回前按序一次落齐 | 运行期 sink 零调用。进程中途崩掉则自启动以来全部丢失。 |

档位时机全在消费者侧，这也是它不花锁的原因：`Async` 冲刷 goroutine 与引擎只
共享一个待提交 channel，提交状态的唯一写者仍是那个消费者。

### Sync 细节

`Sync` 的含义是 happen-before 后继：消费者在记录完成之后、派发下一轮之前调用
`Append`。在 `a -> b` 链上，`a` 的条目在 `b` 拿到激活之前已提交。

### Async 细节

条目立即流向冲刷器但不阻塞派发。`Run` 返回时 —— 无论成败 —— 冲刷器已被排
空并收敛。丢条目、`Seq` 乱序都不可能；**不**承诺的是后继节点体能在 sink 里看
见前驱条目。

### Exit 细节

运行本身对 sink 零调用，结束时一串有序突发全部落地。适合廉价、可整体重跑的
批量图 —— 那里逐节点持久化买不到什么。

---

## 失败语义（fail-closed）

sink 失败永不静默，也绝不让 `Run` 在丢失持久化的情况下交出结论：

- **Sync**：第一次 `Append` 失败立即使 `Run` 失败。不会有后继在未持久化的完成
  之上被派发。
- **Async / Exit**：退出收尾时上交 —— `Run` 失败，而不是带着悄悄缺失的条目返
  回成功。
- 错误包装你的 sink 交出的错误，`errors.Is` / `errors.As` 可归因：

```
graph: checkpoint sink: postgres: connection refused
```

两条边界保证失败报告诚实：

- **运行自身的错误优先。** 图本身失败了就返回那个首要错误；并发的 sink 错误
  不会把它变异成双重报错。
- **失败路径上已完成的工作照常提交。** 运行失败（节点错误、撞步数阀、取消、
  panic 屏障）时，已完成节点的条目仍然冲刷 —— 持久化的意义正是记下发生过的事，
  包括结局不好的那些。

---

## 示例

```go
package main

import (
	"context"
	"fmt"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

type logSink struct{ entries []graph.Checkpoint }

func (s *logSink) Append(ctx context.Context, cp graph.Checkpoint) error {
	fmt.Printf("seq=%d node=%s out=%v\n", cp.Seq, cp.Node, cp.Output)
	s.entries = append(s.entries, cp)
	return nil
}

func main() {
	sink := &logSink{}

	g := graph.New(
		graph.WithDurability(graph.DurabilitySync),
		graph.WithCheckpointer(sink),
	)
	g.AddNode(graph.NodeFunc("a", func(ctx context.Context, in any) (any, error) {
		return "A-out", nil
	}))
	g.AddNode(graph.NodeFunc("b", func(ctx context.Context, in any) (any, error) {
		return "B-out", nil
	}))
	g.AddEdge("a", "b")
	g.SetEntry("a")
	g.SetOutput("b")

	res, err := g.Run(context.Background(), "input")
	fmt.Println(res.Output(), err, len(sink.entries))
	// seq=1 node=a out=A-out   （在 b 被派发之前打印）
	// seq=2 node=b out=B-out
	// B-out <nil> 2
}
```

---

## 刻意还没有的东西

诚实的范围边界，截至本页：

- **恢复 / 重放已交付（G7）。** 挂起的运行作为 `EntryInterrupt` 条目提交到
  这条缝上并从中恢复：[Human-in-the-Loop](/zh/guide/human-in-the-loop) 覆盖
  进程内 `Resume` 与经会话侧车的跨进程重启恢复。
- **没有到流事件的桥接。** 检查点是引擎侧条目；`run.StreamCheckpoints` 协议
  事件族（见 [Run Events API](/zh/api/run-events)）是另一层协议，其生产者接线
  仍在待办。
- **没有存储格式、编码或 TTL 策略。** `Append` 物理上是什么由你的 sink 决定。

---

## 下一步

- 看[图引擎](/zh/guide/graph-engine)了解路由、校验与运行期护栏。
- 给你做检查点的同一批节点挂[节点策略](/zh/guide/node-policies)。
- 查阅 [Graph API 参考](/zh/api/graph)。
