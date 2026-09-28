# 图引擎（Graph Engine）- 零锁控制流图

用五种边与运行期扇出构建 DAG、并行分支、条件路由与汇聚屏障，而调度器内部一把
锁也没有。

---

## 什么是图引擎？

**图引擎**（`pkg/hno/graph`）是 HNO 的控制流图内核。你声明节点与五种边；
引擎在构建期校验整张图，然后从入口节点驱动执行直到收敛。

它是 v3 编排线，**不是** [Workflow](/zh/guide/workflow) 原语的改写。两套 API
今天并存：

- `pkg/hno/workflow` —— v1 基于步骤的编排器（Step/Condition/Loop/Parallel/Router）。
- `pkg/hno/graph` —— 本页讲述的 v3 图引擎（DAG、条件与兜底路由、汇聚屏障、
  节点策略、持久化档位）。

按需选择包，不要在同一条流水线里混用。

### 核心特性

- **五种路由**：无条件边、条件边、兜底边、汇聚（Join）边、汇聚-扇出
  （Join-Send）边。
- **动态扇出**：运行中的节点可以在运行期派发 `Send` 工作项 —— 数量与输入都在
  运行期决定；再以显式的 `AddJoinSend(source, target)` 声明把这一波派发汇聚成
  一次聚合激活。
- **构建期校验**：悬空边、重复节点名、不可达节点、不可断开的环、前驱不足的
  汇聚，全部由 `Validate()` 用可归因的错误文案拒绝 —— 在任何节点跑起来之前。
- **运行期护栏**：步数阀（默认 1000）、panic 屏障、取消归一、可选的并发上限
  FIFO。
- **零锁调度器**：一个消费者 goroutine 独占全部可变状态；生产者只向 channel
  发送。引擎文件里不出现任何 `sync.` 原语，整包测试在 `-race` 下全绿。
- **节点策略**：重试、超时、缓存、追踪通过变参选项逐节点挂载 —— 见
  [节点策略](/zh/guide/node-policies)。
- **持久化档位**：Sync/Async/Exit 三档检查点 + 调用方自带的 sink —— 见
  [图持久化](/zh/advanced/graph-durability)。

---

## 零锁模型

多数图执行器用互斥锁保护共享状态。本引擎按构造不需要它们：

- 每个节点激活跑在自己的**生产者 goroutine** 上。生产者执行节点代码，然后向
  channel 发送恰好一条完成项。它们从不写调度器状态。
- 调用 `Run` 的那个 goroutine 是**唯一消费者**：它是队列的唯一读者，也是
  pending 列表、运行计数、结果与步数账本的唯一写者。

只有一个 goroutine 触碰的状态不需要锁。这个断言可以直接在引擎源码上审计
（`graph.go`、`scheduler.go`、`policy.go`、`durability.go`）：

```bash
grep -o 'sync\.' pkg/hno/graph/graph.go pkg/hno/graph/scheduler.go \
  pkg/hno/graph/policy.go pkg/hno/graph/durability.go | wc -l
# 0
```

---

## 一张完整的图

```go
package main

import (
	"context"
	"fmt"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

func main() {
	g := graph.New(
		graph.WithStepLimit(500),    // 默认 1000；0/负数会被 Validate 拒绝
		graph.WithMaxConcurrency(8), // 0/负数表示不限；打满后 FIFO 排队
	)

	// 从 "fetch" 扇出两条并行分支
	g.AddNode(graph.NodeFunc("fetch", func(ctx context.Context, in any) (any, error) {
		return fmt.Sprintf("rows for %v", in), nil
	}))
	g.AddNode(graph.NodeFunc("left", func(ctx context.Context, in any) (any, error) {
		return fmt.Sprintf("left(%v)", in), nil
	}))
	g.AddNode(graph.NodeFunc("right", func(ctx context.Context, in any) (any, error) {
		return fmt.Sprintf("right(%v)", in), nil
	}))
	g.AddNode(graph.NodeFunc("merge", func(ctx context.Context, in any) (any, error) {
		// 汇聚输入是以前驱名为键的 map[string]any
		parts := in.(map[string]any)
		return fmt.Sprintf("merged: %v", parts), nil
	}))
	g.AddNode(graph.NodeFunc("route", func(ctx context.Context, in any) (any, error) {
		return in, nil
	}))
	g.AddNode(graph.NodeFunc("heavy", func(ctx context.Context, in any) (any, error) {
		return fmt.Sprintf("heavy(%v)", in), nil
	}))
	g.AddNode(graph.NodeFunc("light", func(ctx context.Context, in any) (any, error) {
		return fmt.Sprintf("light(%v)", in), nil
	}))

	g.AddEdge("fetch", "left")
	g.AddEdge("fetch", "right")
	g.AddJoin([]string{"left", "right"}, "merge")
	g.AddEdge("merge", "route")
	g.AddConditional("route", "heavy", func(out any) bool {
		return len(fmt.Sprint(out)) > 40 // 按载荷形状选分支
	})
	g.AddDefault("route", "light")

	g.SetEntry("fetch")
	g.SetOutput("merge") // merge 的返回值即本次运行的输出；
	// heavy/light 在它之后照样跑 —— 输出节点不必是终点

	res, err := g.Run(context.Background(), "demo")
	if err != nil {
		panic(err)
	}
	fmt.Println(res.Output())        // 输出节点的返回值
	out, ok := res.Value("route")    // 任意已完成节点的输出
	fmt.Println(out, ok)
	fmt.Println(res.Completed())     // 已完成节点名，按字典序排序
}
```

---

## 路由语义

一个节点完成后，由消费者决定激活哪些后继。同一节点的所有命中后继作为独立
激活并发运行。

### 1. 无条件边 —— `AddEdge(from, to)`

`from` 完成必然激活 `to`，输入即 `from` 的输出。

### 2. 条件边 —— `AddConditional(from, to, pred)`

仅当 `pred(out)` 对本次输出返回 `true` 时激活。同一源的多条条件边**互不排斥**
—— 命中几条就激活几条。`nil` 谓词是一条缺少路由输入的声明，构建期直接拒绝；
引擎绝不会把它当成「永不命中」而静默丢边。

### 3. 兜底边 —— `AddDefault(from, to)`

兜底语义：仅当 `from` 本次完成时**没有任何**无条件边或条件边命中时才激活。
具体边命中则兜底不触发。

### 4. 汇聚边 —— `AddJoin(from, to)`

声明一道屏障：`to` 在 `from` 里**每一个**声明前驱都完成之后被激活一次。它的
输入不是某个前驱的输出，而是以前驱名为键的 `map[string]any`。

- 少于 2 个去重前驱的汇聚构不成屏障 —— `Validate()` 会点名目标并拒绝。这种
  情况请用 `AddEdge`。
- 前驱因条件环重跑时键集合不再增长，因此汇聚目标会带着刷新后的 map 再次被
  激活。
- 把汇聚输入 map 当作节点 `Run` 期间的只读数据，不要在节点外保留引用：引擎
  当前交出的是它继续写入的同一个 map 对象（交出快照已登记为待办工程项）。
- 诚实说明：若屏障的前驱始终没有凑齐且不再有待派激活，今天的 `Run` 会交出
  nil error 与空输出，而不是一句诊断。该形状是已登记的未裁决项，不是契约。

### 5. 汇聚-扇出边 —— `AddJoinSend(source, target)`

为**运行期**扇出声明汇聚：等 `source` 名下本轮派发的全部 `Send` 激活完成后，
`to` 以聚合输入被激活一次。与 Join 边不同，它没有声明的前驱名单 —— 工作项
经由只在运行期存在的 `Send` 值抵达。完整语义（生命周期、零派发行为、与
`AddJoin` 的选型）见下文「动态扇出」。

---

## 动态扇出 —— `Send` 与 `AddJoinSend`

静态边在 `Run` 开始之前就把形状定死了。`Send` 这条缝补上另一半：运行中的
节点自己决定**派发多少个**工作项、每项带什么输入 —— 这正是静态边做不到的。
配上 `AddJoinSend` 汇聚声明，map-reduce 形状无需预声明任何分支。

### map-reduce 的故事

```go
package main

import (
	"context"

	"github.com/rexleimo/agno-go/pkg/hno/graph"
)

func main() {
	g := graph.New().
		AddNode(graph.SenderFunc("fanout", func(ctx context.Context, in any) (any, []graph.Send, error) {
			items := in.([]string) // 数量与参数都在运行期决定
			sends := make([]graph.Send, 0, len(items))
			for _, it := range items {
				sends = append(sends, graph.Send{Node: "summarize", In: it})
			}
			return nil, sends, nil
		})).
		AddNode(graph.NodeFunc("summarize", func(ctx context.Context, in any) (any, error) {
			return summarizeOne(in.(string)), nil // 每个 Send 一次激活；输入即该次 Send 的 In
		})).
		AddNode(graph.NodeFunc("reduce", func(ctx context.Context, in any) (any, error) {
			// 聚合输入：map[string][]any，键 = 声明的 source 名，
			// 值 = 按派发序的各次输出
			outs := in.(map[string][]any)["summarize"]
			return mergeAll(outs), nil
		})).
		AddJoinSend("summarize", "reduce"). // 声明汇聚
		SetEntry("fanout").
		SetOutput("reduce")

	res, err := g.Run(context.Background(), []string{"alpha", "beta", "gamma"})
	_ = res
	_ = err
}
```

`Run` 之上发生的事：

1. `fanout` 激活一次，返回三个 `Send`。每一个都被当作 `summarize` 的普通
   激活派发，各自带自己的 `In` —— 三项、三次激活、三份独立输出。
2. `summarize` 名下的每次完成都记在 `AddJoinSend("summarize", "reduce")`
   屏障上。当 `summarize` 的在途计数归零，`reduce` 被激活**一次**，输入是
   以 source 名为键的 `map[string][]any`，值按派发序排列。
3. `reduce` 的返回值即本次运行的输出。

注意这里没有 `AddEdge("fanout", "summarize")` —— `Send` 值本身就是派发。
而没有 `AddJoinSend` 声明的话，`Validate()` 会以不可达为由拒绝 `reduce`：
这条声明贡献一条 `summarize -> reduce` 的可达性边，`summarize` 本身也被视为
可达（`Send` 的目标名是运行期值，构建期无法反驳「会有 Sender 指名它」——
与可达性不看谓词是同一乐观先例）。

### 钉死的运行期语义

- **`Sender` 是可选扩展面。** 方法名刻意不叫 `Run`：`Node.Run` 已占用两值
  签名，一个类型不可能同时实现两种形状。调度器在生产者路径对每次激活做类型
  断言 —— `Sender` 节点走 `SendRun`、其 sends 被派发；普通节点走 `Node.Run`、
  不受影响。`SenderFunc` 把函数适配成这样的节点。
- **fail-closed 的目标校验。** `Send` 指向未注册节点会让本次 Run 失败并点名，
  且派发前整体校验全部目标 —— 不做部分派发。构建期判不了这件事（目标只在
  运行期出现），所以这是一条拒绝静默丢弃的运行期检查：

  ```
  graph: node "fanout" sent to unregistered node "ghost"
  ```

- **每个 `Send` 都计一步。** 派发的 sends 与 join-send 目标激活各消耗一步，
  无界扇出（每轮再派发的环）由步数阀收口
  （`errors.Is(err, graph.ErrStepLimitExceeded)`），而不是靠内存撑住。
- **多波扇出逐波凑齐。** source 因条件环重跑再派发时屏障重新武装：每波各自
  独立收敛，target 只收到当波的输出。
- **多条声明可组合。** 同一 source 可声明多个 target（计数归零时各自激活
  一次）；同一 target 可声明多个 source（等全部 source 的计数同时归零才
  激活一次）。
- **普通边共存。** source 也可以同时有无条件边或条件边出边 —— 两路各自
  生效：普通后继按路由规则激活，join-send target 等它的 sends。

### 零派发不等于一次激活

source 激活但返回空 sends（空批次是正常数据，不是错误）时，在途计数从不
离开零、没有任何完成事件触发屏障检查、target 不被激活。若图上再无待派
激活，`Run` 交出 `err == nil` 与 `res.Output() == nil` —— 与「前驱始终没凑齐
的 `AddJoin` 屏障」同一形状。这是今天钉死的行为，也是已登记的未裁决项
（R19-Q1 族），不是修复承诺；可能出现零派发波时请检查 `res.Value("reduce")`。

### AddJoin 与 AddJoinSend：怎么选

| | `AddJoin(from []string, to)` | `AddJoinSend(source, target)` |
|---|---|---|
| 前驱何时确定 | 构建期（声明的名单） | 运行期（本轮有哪些 `Send` 指名 source） |
| 屏障何时凑齐 | 每个声明前驱都完成 | source 名下的在途 `Send` 计数归零 |
| 目标输入 | 以前驱名为键的 `map[string]any` | 以 source 名为键的 `map[string][]any`，值按派发序 |
| 构建期判据 | 去重前驱少于 2 个被 `Validate()` 拒绝 | 端点存在；贡献一条 `source -> target` 可达性边；不数前驱 |
| 适用场景 | 分支集合在代码里写死 | 工作项数量与载荷是运行期数据（map-reduce、逐项分发） |

两类声明刻意各用各的判据：`AddJoin` 的前驱不足拒绝原样保留，`AddJoinSend`
声明也从不查询它。

---

## 构建期校验

`Validate()` 在 `Run` 顶部自动执行，也可显式调用。规则是 fail-closed：不可用
的声明连同它的名字一起被拒绝，绝不静默丢弃后跑出一张缺了零件的图。

| # | 构建期拒绝 | 错误文案（逐字） |
|---|---|---|
| 1 | `WithStepLimit(0)` 或负数 | `graph: WithStepLimit(0) is not a usable step limit: a positive number of steps is required` |
| 2 | 入口未注册 | `graph: entry node "..." does not exist` |
| 3 | 输出节点未注册 | `graph: output node "..." does not exist` |
| 4 | 边端点不是节点 | `graph: unconditional edge from node "..." does not exist`（`to node` 及 conditional/default/join/join-send 同理） |
| 5 | 条件边缺谓词 | `graph: conditional edge from "..." to "..." declares no predicate` |
| 6 | 引擎不认识的边种类 | `graph: ... edge from "..." to "..." is declared but not routable by this engine` |
| 7 | 汇聚前驱 < 2 | `graph: join target "..." must declare at least 2 distinct predecessors, got 1` |
| 8 | 节点从入口不可达 | `graph: unreachable node "...": not reachable from entry "..." through any declared edge` |
| 9 | 环上每条边都必定被走 | `graph: unconditional cycle "a" -> "b" -> "a": every edge on it is always taken, so Run cannot terminate` |
| 10 | 重复节点名 | `graph: node name "..." is declared more than once` |

关于这份清单的几点说明：

- **环是合法的。** 只有「每条边都必定被走」的环（无条件边，加上源节点没有
  条件替代的兜底边）会被拒绝，因为那种 `Run` 永不终止。任何一条条件边能断开
  的环都合法；谓词恒真的最坏情况由步数阀兜住。
- **可达只按已声明的边算**，不分种类也不看谓词：构建期判不了谓词，把「可达」
  实现成「真的跑过」等于把一切条件分支判为非法。一个乐观先例：`AddJoinSend`
  声明的 source 被视为可达（它的流量来自构建期看不见的运行期 `Send` 值），
  且该声明贡献一条 `source -> target` 边。
- **文案可归因且稳定。** 多数检查一次只报一项（map 来源的集合取字典序第一），
  同一张坏图永远报同一句话，可以直接贴进 issue。长环渲染为开头点名 + 省略号
  + 规模数字，不会把几千个名字倒进你的终端。
- **重复注册是记账，不是立即报错。** 同名 `AddNode` 仍在表里覆盖先注册者
  （进行中的 Run 持有拓扑快照、不受影响），但这次撞名被记进账本，`Validate()`
  随后拒绝。builder 方法返回 `*Graph`、没有错误通道 —— 记账 + 复查是这个 API
  形状下唯一诚实的增量检查。
- **次序可观察且被测试钉住**：先判声明层（预算、入口、输出、端点、谓词、可
  路由种类），再判汇聚前驱不足、不可达、不可断开的环，末位判重复名。一张既
  不可达又含环的图报的是可达性。

### `Warnings()` —— 非致命通道

`Validate()` 只拒跑不了的图。`Warnings()` 报告跑得了但值得再看一眼的东西，
两者刻意分离：

```go
g := graph.New() // 未声明 WithStepLimit
g.AddNode(graph.NodeFunc("a", fnA))
g.AddNode(graph.NodeFunc("b", fnB))
g.AddEdge("a", "b")
g.AddConditional("b", "a", func(out any) bool { return shouldLoop(out) }) // 可断开的环

for _, w := range g.Warnings() {
	fmt.Println(w)
}
// graph: cycle "a" -> "b" -> "a" has no explicit budget: WithStepLimit was not set,
//        so the default stepLimit=1000 is what bounds this graph
```

上面的环是合法的（条件边能断开它，`Validate()` 通过）—— 但若谓词恒真，这次
运行会一路顶着默认预算 1000 步磨下去。这条警告告诉你：挡在那里的只有你自己
从没选过的那个数字。`Warnings()` 只读 —— 不改配置、不影响执行、问几次答案一
致。把危险说成非法，等于替调用方决定这张图不能跑；警告只陈述事实。

---

## 运行期护栏

构建期看不见谓词与节点行为，运行期自带阀门，全部由唯一消费者执行：

### 步数阀

每个派发的激活消耗一步。步数在图收敛前达到上限时，`Run` 返回包装
`graph.ErrStepLimitExceeded` 的错误：

```go
if errors.Is(err, graph.ErrStepLimitExceeded) { ... }
// graph: step limit 1000 reached before the graph converged: graph: step limit exceeded
```

计步发生在**把激活交给节点之前**：会让步数越界的激活根本不会被启动，不会有
收不回的副作用逃过阀门。节点策略里的重试不消耗步数（它们发生在一次激活内）
—— 见[节点策略](/zh/guide/node-policies)。

### panic 屏障

任何节点内的 panic 在节点帧之上被恢复（节点自己的 `defer` 先跑完），转成点名
节点的错误：

```
graph: node "fetch" panicked while running: runtime error: index out of range [5]
```

崩溃的 goroutine 仍经由队列汇报；它从不直接写调度器状态。

### 取消归一

传入可取消的 context。当调用方取消与节点错误同时成立时，`Run` 报告的是
**取消**（`ctx.Err()`）而不是节点错误 —— 消费者在把出队项当作结论之前会再问
一次 context。

### 并发上限 FIFO

`WithMaxConcurrency(n)` 限制在途激活数。槽位打满时新激活进 FIFO 等待队列；
每完成一个自动派发下一个。`0` 或负数表示不限 —— 这是守卫自身给出的语义，
不是遗漏。

### 每次 Run 的拓扑快照

`Run` 执行的是进入时通过 `Validate()` 的那份拓扑快照。`Run` 开始后再改
builder 只影响下一次运行 —— 节点内一次随手 `AddEdge` 没法把未校验的声明偷运
进进行中的执行。`stepLimit`、`maxConcurrency` 与持久化设置在同一不变量下于
入口读取一次。

---

## Typed 节点

`NodeFunc` 刻意是 `any` 类型的 —— 图不理解你的数据。想在叶子处要编译期类型
安全时，用 `Typed` 适配具体函数：

```go
upper := graph.Typed[string, string]("upper", func(ctx context.Context, in string) (string, error) {
	return strings.ToUpper(in), nil
})
g.AddNode(upper)
```

输入的动态类型与 `TIn` 不符时，节点返回可归因的错误而不是 panic：

```
graph: node "upper": input type mismatch: expected string, got int
```

`nil` 输入渲染为 `no value`，而不是格式化噪声。

---

## Roadmap

本页只描述已落地的行为。一个相邻能力已设计、**尚未交付**：

- **从检查点恢复 / 重放，以及 HITL（`RequestInterrupt` / `Resume`）** ——
  今天的持久化缝只有提交侧；把检查点读回来恢复一张挂起的图、以及人机协同的
  中断/恢复，规划在 G7（契约：
  [v3-test-scope-p1-graph-slice27.json](https://github.com/rexleimo/HNO/blob/main/docs/design/v3-test-scope-p1-graph-slice27.json)）。
  见[图持久化](/zh/advanced/graph-durability)。

---

## 下一步

- 为单个节点挂载[节点策略](/zh/guide/node-policies)（重试、超时、缓存、追踪）。
- 用[图持久化](/zh/advanced/graph-durability)持久化已完成节点。
- 查阅[Graph API 参考](/zh/api/graph)的完整签名。
- 与 [Workflow](/zh/guide/workflow)（v1 步骤编排器）对比选型。
