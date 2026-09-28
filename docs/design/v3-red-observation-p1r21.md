# 切片 21 RED 观察（票面 §4 B6 第 2 项：重复节点名构建期拒绝）

工作项 `work-p1-graph-r21-duplicate-node-name` / activation `2d96c915-5fc4-48c2-baa6-b27b7c27b29e` / 阶段 red。
契约：`docs/design/v3-test-scope-p1-graph-slice21.json`（D1–D5）。
测试文件：`pkg/hno/graph/p1r21_duplicate_node_test.go`（`package graph_test`，只用导出 API）。

## 1. 绑定命令与实测

本片绑定场景命令逐字（所有 receipt 都是这一条命令直接取，无 `sh -c` 包装）：

```bash
go test ./pkg/hno/graph -run TestP1R21_ -count=1 -race -v
```

最终 RED（实现字节未动、夹具修正后）：**exit 1**，`receipt:5964bbf8-f78d-4594-8d4c-e98a87c5519a`。

- `--- FAIL: TestP1R21_DuplicateNodeNameIsRejectedLocatably`（D1）：
  `重复注册同名节点 "entry" 的图被 Validate 放行（返回 nil），构建期一句话都没有`
- `--- FAIL: TestP1R21_RunRefusesGraphWithDuplicateNodeName`（D2）：
  `Run 之前 Validate 返回 nil：重复节点名未在构建期被拒（Run err=<nil>）`
- `--- FAIL: TestP1R21_MultipleDuplicateNamesStillNameOne`（D5）：
  `b、c 各重复一次的图被 Validate 放行（返回 nil），重复名检查整段失效`
- `--- PASS: TestP1R21_DistinctNamesStillValidateAndRun`（D3）、
  `--- PASS: TestP1R21_NewNameAfterRunStaysVisible`（D4）：两条反向对照今天本来就成立，按契约预期绿。

三个失败全是「今天静默覆盖」的公共面形状，无编译失败、无环境失败 —— 契约判据 1 满足。
RED 时 `graph.go = 4cb363b7…`（契约 baselineBytes 原值），实现零改动。

## 2. 夹具缺陷的一次自纠（首次 RED 假绿）

第一次 RED（`receipt:90d305b4-a81c-4d43-81fe-0a7aa2a1c5f6`，exit 1）里 D1 意外 **PASS**：第一版夹具
多注册了一个未接线的 `out` 节点，既有的 `unreachable node "out"` 文案替重复名断言顶了名 ——
`err != nil`、有 `graph: ` 前缀、有引号名字，三条断言全被那条无关错误喂饱。契约 D1 行指定的图是
M1 形2（重复 entry、入口即输出、无多余节点），夹具已按契约修正后重取 RED。
教训与票面 §6 同源：**断言被谁顶名，要靠看失败文案而不是靠推断。**

（更早的 `receipt:80563dbf…` 是上一会话只有 D1/D2 两行时的部分 RED，exit 1，保留作历史。）

## 3. 承接

实现与 GREEN 见 `docs/design/v3-p1-graph-r21-green.md`；REFACTOR 与完整变异矩阵见
`docs/design/v3-p1-graph-r21-refactor.md`。
