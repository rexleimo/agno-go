# 切片 23 REFACTOR（母约 §5 G4：StreamMode 协议层）

工作项 `work-p4-run-g4-streammode-protocol` / 契约 `docs/design/v3-test-scope-p1-graph-slice23.json`（D1–D12）。

## 1. REFACTOR 结论：无独立重构动作，最终字节 = GREEN 字节

GREEN 交付的实现直接采用了 M2 设计期副本预演的形状（协议常量集中在 modes.go、六事件类型与
canonical JSON 集中在 stream_events.go、解码缝集中在 decodeEvent 的精确匹配块、agent 侧只有
一个 23 行的选择器），没有可合并的重复；REFACTOR 阶段零代码改动。最终字节：
`modes.go fcf1d188…`、`stream_events.go 25e7f45c…`、`events_json.go 6380d017…`、
`agent.go 9813b86a…`、`stream.go bd3583dc…`。最终门禁复测：build/vet/gofmt 干净，
run+agent 整包 `-count=1` 与 `-race` `ok`（`receipt:9b22c44d…` / `receipt:cde554f4…`），
绑定命令在最终字节上重跑 `ok`（`receipt:5c318908…`）。

## 2. 完整变异矩阵（`scripts/mutation/p4g4-streammode.mjs`，仓库内可复现）

命令 A = 绑定场景命令（`TestP4G4_`，run+agent 两包，`-race -v`），命令 B = 既有判据
（run+agent 两包 `-run 'TestP1|TestP0'`，15 个 P0/P1 测试）。基线：
`events_json.go 6380d017…`、`stream_events.go 25e7f45c…`、`agent.go 9813b86a…`、
`stream.go bd3583dc…`；每条变异后 `restored=true`、`finalCheck identical=true`。

| 变异 | 形态 | 命令 A | 命令 B | 判定 |
|---|---|---|---|---|
| m1 | 精确匹配块整体摘掉（退回 contains + Generic 兜底） | exit 1，杀红 D4/D11/D12+ZeroValue（4 Test） | exit 0 | **杀红** |
| m2 | 次序回退（精确块挪回 contains 之后） | exit 1，杀红同上 4 Test（node_completed 方向） | exit 0 | **杀红** |
| m3 | MarshalJSON 空 kind 兜底摘掉（六类全部） | exit 1，杀红 ZeroValue（D2/D4 第二层） | exit 0 | **杀红** |
| m4 | RunStreamMode 模式校验整段失效（静默降级） | exit 1，杀红 D10 | exit 0 | **杀红** |
| m5 | RunStream 改传 StreamCustom（不再等价 messages） | exit 1，杀红 D8/D9 | exit 1，连带杀红 10 个 P0 流式判据 | **杀红（A+B 双杀）** |
| m6 | 构造函数漏盖种类（六个 New\*） | exit 1，杀红 D2 + D4 | exit 0 | **杀红** |

执行回执：`receipt:5ab9bfe7-4fd9-4442-a530-d9cd9852c631`（exit 0，六条全杀红、零 BUILD_FAILED、
零 SCRIPT_ERROR、逐字节恢复自检全过）。本片无等价变异登记：契约判据 2 点名的六种形态全部
有专属杀红行。

### 2.1 观察上限（如实登记）

1. **m3 的牙齿只由一个 Test 兜住**：ZeroValue 行只钉 NodeCompletedEvent 的字面量兜底；
   其余五类的兜底是同构实现但无专属判红行（m3 同时摘掉六类，杀红从该行触发）。钉全六类
   需要六行同族测试，本片按契约「协议最小集」不追加。
2. **m5 的「独立实现」形态不可观察**：若 RunStream 摘掉包装、保留与 runStreamMessages 相同的
   独立实现体，公共面事件序列不变（D8/D9 不判红）——包装关系本身只有「换模式」这一可观察
   缺口，m5 取的就是该形态。「是否真的委托」无法从公共面钉住，登记为观察上限（与切片 22
   e1 同族：未钉语义不追加断言）。
3. **模式语义不可观察**（契约 observabilityLimit 第 1 条）：StreamValues/Updates/Tasks/
   Checkpoints/Debug/Custom 的运行期行为本片只有「存在、有序、fail-closed」三个公共面事实；
   谁发事件、何时发由各自生产者接线片钉（S23-SPEC-1）。
4. **时间戳只断非零**：D8/D9 一致性按事件序列同型同内容断，时间戳数值不参与比较
   （契约 forbiddenShortcuts 第 4 条）。

### 2.2 S23-STD-1 处置记录

契约携带项 S23-STD-1（切片 22 登记过一次不可复现的裸 FAIL，若再现先 `-v` 全量捕获）：
本片全部门禁与防碰巧绿（`-count=5 -race`，`receipt:cbe8754f-fe41-46c9-aa67-63758e957d2b`）
均未再现裸 FAIL，条款继续携带、未被触发。

## 3. 契约 completionCriteria 逐条对账

1. redProtocol 两段各自留 receipt：第一段 `receipt:2e128444…`（exit 1，4 红/8 反向绿）→
   第二段 `receipt:691c84f4…`（exit 0，12 绿）。两段之间测试文件零改动——**一处如实修正**：
   中间回执 `receipt:e626b1c0…`（exit 1）暴露测试自身 `CustomEvent.Data` 的 int/float64 跨
   JSON 数值比较缺陷，D4/D11 的 custom 载荷改为字符串；RED 判红的四行与该修正无关且在
   修正前后同样判红（green 文档 §1）。✅
2. 牙齿由变异证明：m1–m6 覆盖判据 2 点名的全部六种形态（摘精确块/次序回退/摘兜底/摘模式
   校验/RunStream 不等价/构造函数漏盖），六条全杀红；无等价变异，观察上限见 §2.1。✅
3. 既有测试文件零改动：run 3 份 + agent 全部既有测试，逐文件 git hash-object 复核登记
   （green 文档 §4，含契约 baselineBytes 配对错位的更正记录）。✅
4. 门禁全 exit 0：build / run+agent `-count=1` / `-race` / vet / gofmt + 邻近回归
   （runner/team/workflow/session-contract 四包）（green 文档 §3）。✅
5. 结构判据：导出符号恰 77（38→77；const 块合并的算术说明见 green 文档 §6）、run 包全文件
   LOC 1723≤2000、agent.go 净增 +23≤80、新增文件 `sync.` 计数 0。✅
6. 绑定命令在最终字节上重跑并取新回执 `receipt:5c318908…`；`-race` 无 DATA RACE。✅
7. 写回：母约 §5 已落「已交付（切片 23，协议层）」注记与七模式解禁状态表（Messages 已接、
   其余六模式 fail-closed 待生产者片）；交付表 P4 行已标「协议层已交付（切片 23）」。
   状态文档（v3-p1-graph-status.md）为编排方持有文件，本片不触碰，§4/§5 写回由编排方
   收口时补记（本片交付记录以本文档与 green 文档为准）。✅（部分，附归属说明）
8. 防碰巧绿：run+agent 整包 `-count=5 -race` 一次 exit 0（`receipt:cbe8754f…`）。✅

## 4. 审查

review 结论见 `docs/design/v3-p4-run-g4-review-verdict.json`（作者自查 + 独立性披露）。
