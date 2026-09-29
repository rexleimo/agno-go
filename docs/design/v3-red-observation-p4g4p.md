# 切片 34 RED 观察（P4 第 3 片，母约 G4：剩余五模式生产者接线）

工作项 `work-p4-g4-producers` / 契约 `docs/design/v3-test-scope-p4-g4-producers.json`（D1–D9）。
契约 redProtocol 两段制；**迁移前置**：三份既有测试的授权迁移行在第一段落之前落好（否则
第一段落的红会被过时的「五模式 fail-closed」断言污染，违反归因纪律）。本片绑定场景命令逐字：

```bash
go test ./pkg/hno/agent -run TestP4G4P_ -count=1 -race -v
```

## 1. 第一段（声明与接线，发射体故意空转）——逐行红形

第一段字节（`stream_producers.go` 内两处占位：`turnCompleted` 体 `return nil`、`custom` 体
门后 `return nil`）：wiredStreamModes 扩到七、`enabled()` 带Debug 并集分支、
`taskStarted`/`taskFinished` 带族门、`streamExecutor` 双族选择、`turnObserver` 接上
`onAssistantTurn`（stream.go 一行）、`WriteCustomEvent` 导出与 ctx writer 安装。

| 项 | 命令 | exit | receipt |
|---|---|---|---|
| 第一段可编译（redProtocol 的前提证明） | `go vet ./pkg/hno/agent` | 0 | `receipt:eb18e52c-0772-4512-a25f-8eecbf313c4a` |
| 绑定命令（-race -v） | 5 个测试函数判红 / 3 个通过 | 1 | `receipt:fa6aafaa-b247-48e4-9a1d-ee319b40a519` |
| `-v` 读数 `--- FAIL` 计数 | 5（明文复跑 `/tmp/p4g4p/seg1-raw.txt`，非 receipt 记录） | 1 | 过程转录 |

逐行红因（`-v` 实读）：

| 行 | 红因类别 | 实读 |
|---|---|---|
| D1 | **绿** | 模式表扩到七 + unknown 序数守门已足以让五个模式启动流（D1 只断「流启动了」，族内容由 D3–D7 管） |
| D2 | **绿** | 未知序数 fail-closed 由归一化循环独立成立，与新发射语义无关 |
| D3 | **行为红** | `stream kinds = [] (0 events), want [state_update state_update]`——发射体空转，通道上没有逐回合族事件（绝对序列断言落空，非编译红） |
| D4 | **行为红** | 同形（0 events） |
| D5 | **行为红** | 同形（0 events） |
| D6 | **行为红** | Tasks 半在场（族门内移后并集门打开 taskStarted）、Checkpoints 半缺席——绝对序列 `[node_started node_started node_completed task_error]` 差两条 checkpoint |
| D7 | **行为红** | handler 写入 err==nil（writer 已安装、查找成功）但通道 0 条 custom 事件——**静默成功**正是第二段要消灭的形状 |
| D8 | **绿（漂移哨兵）** | 旧两族在第一段就不得被打破——族门内移行为中性由本行当场证明 |

**vacuous-green 如实登记**：D7 的两个失败面子用例（bare ctx / Messages 运行无 writer）在
第一段即绿——writer 缺席本来就是构造行为，不依赖发射语义；第二段它们转为「写入真正上流」
正面语义绿的对照半，其牙齿由变异 m-custom-gate（writer 恒安装）承担。

## 2. 第二段（发射语义）——全绿

第二段 = 最终字节：`turnCompleted` 三体（Updates/Values/Checkpoints）与 `custom` 写透填齐。
两段之间仅 `stream_producers.go` 变动，测试文件零改动。

| 项 | 结果 | receipt |
|---|---|---|
| 绑定命令 8 个测试函数全 `--- PASS` | exit 0 | `receipt:4082e7da-164d-4c67-b25a-99b510f4c327` |

实现期自纠一条（如实登记）：D7 第三子用例的期望序列初稿写成 `[run_content ×2]`，而夹具脚本
回合 2 只有一个内容分块——**测试夹具算术错**，非实现缺陷；改为 `[run_content]` 后全绿。
该修正发生在第二段绑定命令首次全绿之前，两段之间测试文件零改动的红线不受影响
（该文件本就是本片新增文件，不属于「既有测试」）。

## 3. 契约依据

- 契约 redProtocol 原文：第一段落的红必须全部来自 D3–D7 的行为断言（编译红即分段切错）——
  本段实测红形逐条对上（`stream kinds = []` 家族，无一条编译/setup 失败）。
- 迁移的三份既有测试在第一段落即全绿（runstreammode_test.go 的未知序数名单、
  stream_tasks_test.go 的未知序数名单 + Debug 转正子用例、p7g9s2_agent_span_test.go 的
  未知序数负控），它们钉「迁移不放宽」这一半，不充当行为红。
