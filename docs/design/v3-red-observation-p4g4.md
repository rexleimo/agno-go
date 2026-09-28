# 切片 23 RED 观察（母约 §5 G4：StreamMode 协议层）

工作项 `work-p4-run-g4-streammode-protocol` / 契约 `docs/design/v3-test-scope-p1-graph-slice23.json`（D1–D12）。
测试文件：`pkg/hno/run/streammode_test.go`（`package run_test`）与 `pkg/hno/agent/runstreammode_test.go`
（`package agent_test`，自带确定性流式 stub，沿用 agent 包既有 `MockModel` 的形状）。
只用导出 API；不观察 agent 内部循环私有字段、channel 容量、goroutine 数。

## 1. redProtocol 两段制的第一段

本片与切片 22 同构：新增公共面符号。第一段只落 API 面——`run/modes.go`（StreamMode 类型、
七常量、六 wire 名、`ErrUnsupportedStreamMode`）、`run/stream_events.go`（六类型 + New\* 构造函数 +
canonical JSON 方法）、`agent.RunStreamMode`（模式校验 + 委托 `runStreamMessages`）、`RunStream`
改纯包装。`events_json.go` 的 `decodeEvent` 尚未扩展。绑定命令可编译，行为行按设计期 M1/M2
实测的形状真实判红。

本片绑定场景命令逐字（所有 receipt 都是这一条命令、直接取，无 `sh -c` 包装）：

```bash
go test ./pkg/hno/run ./pkg/hno/agent -run 'TestP4G4_' -count=1 -race -v
```

## 2. RED 实测（receipt:2e128444-7995-4112-bc4c-eccca61b8aa3，exit 1）

**判红 4 个 Test**（全部落在「decodeEvent 未扩展」的公共面形状上，零编译失败、零环境失败）：

| Test | 行 | 失败文案（摘） |
|---|---|---|
| TestP4G4_NewEventRoundTripTyped | D4 | `expected *run.NodeStartedEvent, got *run.GenericRunEvent`；`node_completed must not be swallowed into RunCompletedEvent`；task_error/checkpoint/state_update/custom 同落 `*run.GenericRunEvent` |
| TestP4G4_ZeroValueEventMarshalUsesCanonicalFallback | D4/D2 第二层 | `empty-kind literal must round-trip typed as NodeCompletedEvent, got *run.RunCompletedEvent`（wire 名已被序列化侧兜底为 node_completed，但解码侧 still 被含 completed 的旧归一化吃掉——正是 M2 缺陷二的形状） |
| TestP4G4_MixedArrayRoundTrip | D11 | `decoded[1] = *run.GenericRunEvent, want *run.NodeStartedEvent` |
| TestP4G4_DecodeOrderExactBeforeContains | D12 | `exact canonical name must win over contains normalization, got *run.RunCompletedEvent` |

**天然绿 8 个 Test**（反向对照，按实现前后都必须绿的预期）：D1（常量序数）、D3（旧 golden）、
D5（未知 kind 兜底）、D6（team_run_content/run_completed/大小写空白宽容）、D8、D9、D10
（RunStreamMode 三态）。

## 3. D2 的判红时刻澄清（如实披露，不造假红）

契约 redProtocol 预测第一段「D2/D4/D11 判红」；本片实测 D2（构造函数盖章）在第一段**绿**——
构造函数与 per-type JSON 属 API 面（切片 22 先例：policy.go 全量落在第一段），且本片第一段
已包含它们，所以 D2 的断言（EventType 各归其名、Timestamp 非零）在第一段字节上即可成立。
D2 的**真实判红证据在设计期 M2**：字面量构造的事件 eventType 为空、MarshalJSON 原样吐空
event 字段、解码被旧归一化吃掉（契约 M2 登记：探针 receipt:d1234a72-4f4b-49e0-bce2-d66b85e7c129
与 receipt:a83e432f-41ad-4865-bc8d-5aaffa9c5b63，各 exit 1——两份为契约登记的设计期回执，
本片未重跑，出处见契约 evidenceRefs）。本片不为此制造第一段假红，D2 的牙齿由变异 m6
（构造函数漏盖种类）在 GREEN 字节上另行证明。

## 4. 承接

第二段（`decodeEvent` 扩展六精确匹配 case 并置于 contains 归一化之前）与 GREEN 见
`docs/design/v3-p4-run-g4-green.md`；变异矩阵与契约对账见 `docs/design/v3-p4-run-g4-refactor.md`。
