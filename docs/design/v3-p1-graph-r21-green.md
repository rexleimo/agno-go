# 切片 21 GREEN（票面 §4 B6 第 2 项：重复节点名构建期拒绝）

工作项 `work-p1-graph-r21-duplicate-node-name` / activation `2d96c915-5fc4-48c2-baa6-b27b7c27b29e` / 阶段 green。
契约：`docs/design/v3-test-scope-p1-graph-slice21.json`（D1–D5）。
RED 观察见 `docs/design/v3-red-observation-p1r21.md`；REFACTOR 后的最终字节与完整变异矩阵见
`docs/design/v3-p1-graph-r21-refactor.md`（本文件 §3 的哈希是 GREEN 时字节，未被 REFACTOR 改动）。
本片绑定场景命令逐字（每一条 receipt 都是这一条命令、直接取，无 `sh -c` 包装）：

```bash
go test ./pkg/hno/graph -run TestP1R21_ -count=1 -race -v
```

## 1. GREEN 判据与实测

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| D1–D5 全绿（契约绑定命令，逐字） | `go test ./pkg/hno/graph -run TestP1R21_ -count=1 -race -v` | 0 | `receipt:aaea7171-83d4-41be-a832-a4dbbcf3f326` |
| 前一步 RED（同一命令，实现之前） | 同上 | 1 | `receipt:5964bbf8-f78d-4594-8d4c-e98a87c5519a` |

`-v` 逐行：5 个 Test 全部 `--- PASS`，末行 `ok github.com/rexleimo/agno-go/pkg/hno/graph 2.118s`。

## 2. 实现差异（有边界）

只动 `pkg/hno/graph/graph.go`（`4cb363b7…` → `b7790b9f…`，净增 +34 行，580 → 614 行）。
`scheduler.go` 零改动（契约 sourceSeam 的预期为真）；既有 9 份测试文件逐字节未动（§3 表）。

1. `Graph` 新增非导出字段 `dupNames []string`：`AddNode` 的同名注册账本，按声明顺序累积。
   字段注释写明它只是构建期的内部事实、不是公共面读数（票面 :40 与契约 forbiddenObservations）。
2. `AddNode`：命中已有名字时先记账再照旧覆盖 map。覆盖语义保留 ——
   `graph_test.go:953` 的 in-flight Run 捕获隔离子用例依赖它，该子用例在最终字节上照旧通过。
   记账而不在此处报错，是 API 形状（返回 `*Graph`、无错误通道）下「增量检查」的唯一诚实读法。
3. `Validate` 末位新增 `rejectDuplicateNodeNames()`：账本非空时返回
   `graph: node name %q is declared more than once`，点名账本首个（声明序里第一次撞名的那格，
   结果稳定，不依赖 map 遍历）。`Run` 第一行就是 `Validate`，错误原样透传、`res==nil`（D2）。
4. `Validate` 文档的次序段同步为四步（声明成立 → 屏障 → 可执行 → 注册账本）。

### 2.1 导出面与结构判据

`go doc -all ./pkg/hno/graph | grep -cE '^(func|type|var|const) '` = **24**（零新增导出符号）；
`grep -o 'sync\.' pkg/hno/graph/graph.go pkg/hno/graph/scheduler.go | wc -l` = **0**（零锁不变式未破）。

## 3. 门禁（契约判据 4–7，全部在最终字节上重跑）

| 门禁 | 结果 | receipt |
|---|---|---|
| `go build ./...` | exit 0 | （非 receipt 记录） |
| `go test ./pkg/hno/graph/... -count=1` | `ok 1.083s` exit 0 | `receipt:0f363770-b290-4477-ac2a-c3c5bd14b594` |
| `go test -race ./pkg/hno/graph/... -count=1` | `ok 2.544s` exit 0，无 DATA RACE | `receipt:9665d14d-350e-4b49-8f3c-0a235da3579e` |
| `go vet ./pkg/hno/graph/...` | exit 0 | （非 receipt 记录） |
| `gofmt -l pkg/hno/graph` | 无输出 | （非 receipt 记录） |
| 邻近回归（agent/runner/session-contract） | 三包全 `ok` exit 0 | `receipt:06927728-9537-4483-aa0c-2f158dce541d` |
| 非测试 LOC（判据 ≤1500） | 838 → **872** | （`find … xargs wc -l`） |
| 导出符号（判据 =24） | **24** | （`go doc -all … grep -c`） |
| `sync.` 计数（判据 =0） | **0** | （`grep -o … wc -l`） |

防「碰巧绿」（非 receipt 记录，均真实执行）：
`go test ./pkg/hno/graph -run TestP1R21_ -count=30 -race -timeout 300s` → `ok` exit 0
（`receipt:d47899e6-2dc3-49a1-b3fd-e89d02b0c092`）；
`go test ./pkg/hno/graph/... -count=5 -race -timeout 400s` → `ok` exit 0
（`receipt:ea69d208-3e0a-4278-b3f2-141d7d1336e7`）。

既有 9 份测试文件与 `scheduler.go` 逐字节复核（`git hash-object --no-filters`，与契约 baselineBytes 一致）：
`graph_test.go 624f1962…`、`p1r13 bc4ab8e7…`、`p1r14 0ed607db…`、`p1r15 3ea06d27…`、`p1r15b 0285ab35…`、
`p1r16 6934a4f2…`、`p1r17 cbc8b692…`、`p1r18 10bce61b…`、`p1r19 8b4b7234…`、`scheduler.go c878a6c7…`。

## 4. 每行由哪段实现担保

| 行 | 断言的公共事实 | 实现落点 |
|---|---|---|
| D1 | 同名注册两次 → `Validate()` 非 nil，`graph: ` 前缀 + 引号点名 | `AddNode` 记账 + `rejectDuplicateNodeNames` 的 `%q` 文案 |
| D2 | `Run` 拒绝、`res==nil`、错误与 `Validate` 一致，不交出后注册者的输出 | `Run` 首行 `Validate` 原样透传 |
| D3 | 全程不同名的合法图 `Validate()==nil` 且 `Run` 照常 | 账本只在撞名时记；检查不碰注册自由 |
| D4 | `Run` 之后注册**新名字**对下一次 `Run` 可见 | 同上 —— 检查只认「撞名」，不是「注册过」 |
| D5 | 多个重复名时错误点名其中之一（引号边界） | `%q` 渲染账本首个，恒为真实撞过名的名字 |

## 5. 事故披露：一次 `git checkout` 与逐字节恢复（详录见票面 §17.4）

变异阶段中途 `git checkout pkg/hno/graph/graph.go` 把本文件回退到 HEAD 存根（`02122d9d…`，
事故中 `receipt:9c9c8ac3…` 留下了存根态的 exit 1 记录）。恢复：切片 20 实验副本
`/tmp/sinkexp/pkg/hno/graph/graph.go`（`git hash-object` = `4cb363b7…`，与契约基线全同）→ 重放本片
三处 GREEN 编辑 → 复核 `b7790b9f…` 与事故前 GREEN 字节**逐字节相同** → 绑定命令复跑 exit 0
（`receipt:bd24a87d-73fa-4678-accb-04e8904529b9`）。切片 2–21 未提交状态因此暴露了单拷贝脆弱性
（`S18-SPEC-2` 沿用）；在拿到提交授权前，变异/实验脚本不得对本仓工作树执行 git 破坏性命令。

## 6. 本阶段未闭环

- REFACTOR 阶段结论与完整变异矩阵见 `docs/design/v3-p1-graph-r21-refactor.md`。
- `golangci-lint` / `make lint` 仍未运行（与前序切片同样的欠项）。
