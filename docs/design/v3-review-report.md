# Agno-Go v3.0 规划交付物审查报告

**审查员**: 独立审查（非作者）  
**审查时间**: 2026-09-26  
**审查对象**: 
- `docs/design/v3-delivery-ticket.json`
- `docs/design/v3-decision-ticket.md`
- `docs/design/v3-baseline-evidence.md`
- `docs/design/v3-platform.md`

---

## 1. 总体结论

**有条件通过** — 交付票结构完整，大部分基线证据可核实，依赖图逻辑正确。存在 **2 个阻塞级问题**（Schema 校验未跑通、参考仓库 LOC 无法核实）和 **3 个重要问题**（基线数据偏差、并行分组依据不足、验收标准模糊）。

修复阻塞问题后可进入执行。

---

## 2. 阻塞级问题（必须修）

### B1. Schema 校验未能执行

**问题**: 交付票声称符合 rex-harness schema，但因 Node.js heredoc 语法限制无法运行校验器验证。

**证据**: 
```bash
# 尝试运行 harness 校验
node << 'EOF'
import { normalizePlanningArtifact } from '...';
# 失败：Contains brace with quote character (expansion obfuscation)
```

**影响**: 无法确认 JSON 结构、字段类型、必填项是否完全符合 harness 要求。

**建议**: 
1. 用独立脚本文件（非 heredoc）跑校验：
   ```bash
   node /tmp/validate-ticket.mjs
   ```
2. 或提供 harness CLI 的原始输出截图作为证据

---

### B2. 参考仓库 LOC 数据无法核实

**问题**: `v3-baseline-evidence.md` §4 声称从 `/tmp/fwcmp/adk-go` 和 `/tmp/fwcmp/langgraph` 采集 LOC 数据，但：
- `/tmp/fwcmp/adk-go` 存在但 `cd + find` 命令需批准，未能重跑
- `/tmp/fwcmp/langgraph` **不存在**

**证据**:
```bash
$ test -f /tmp/fwcmp/adk-go/workflow/graph.go
# adk-go accessible

$ test -f /tmp/fwcmp/langgraph/langgraph/types.py
# langgraph not accessible
```

**影响**: 
- adk-go 6,244 行、LangGraph 19 万行/4,400 行核心等关键论据无法验证
- §4「LangGraph 维护成本可控」的推论依据不足

**建议**:
1. 补充参考仓库克隆命令到证据文件，使审查者可重现
2. 或改为引用这些仓库的 GitHub 统计 API / cloc badge
3. 明确标注「参考仓库数据来自作者环境，审查时仓库不存在」

---

## 3. 重要问题

### I1. 基线证据部分数据有偏差

**问题**: 对照实测，3 处数据不一致。

| 项 | 声称值 | 实测值 | 位置 |
|---|---|---|---|
| skills LOC | 398 | **684** | `v3-baseline-evidence.md` §3 M3 行 |
| 循环实现位置 | `pkg/hno/run/loop.go` | 实为 `pkg/hno/team/team.go:295` 的 **注释** | §2 第 4 行 |
| M5 workflow builder | 未做（声称 grep 空） | 实为**已有但非图式** Builder：线性 `[]Step` | §3 M5 行 |

**证据**:
```bash
$ wc -l pkg/hno/skills/*.go | tail -1
     684 total    # 不是 398

$ grep -n "run.Loop" pkg/hno/team/team.go | head -1
295:	// Adapt the team to the shared run.Loop kernel: the scheduler's Next
# 第 295 行是注释，非调用

$ ls pkg/hno/workflow/*.go
# executor.go 存在，但为线性驱动非图
```

**影响**: 基线偏差降低证据可信度。Skills LOC 低估可能低估迁移成本。

**建议**: 重跑采集命令并更新 `v3-baseline-evidence.md`。

---

### I2. 并行分组 2 的并行性依据不足

**问题**: `parallelGroups[1]` 包含 4 项并行工作：
```json
["work-p1-graph-engine", "work-p4-streammode", "work-p7-observability", "work-p8-store"]
```

但 `v3-platform.md` §10（里程碑阶段划分）未明确说明为何这 4 项可并行。审查发现：
- `p1` 修改 `pkg/hno/graph/`（新建）
- `p4` 修改 `pkg/hno/run/` 事件体系，可能影响 `agent/stream.go`
- `p7` 修改 `pkg/hno/runner/` 与 `observability/`
- `p8` 新建 `pkg/hno/store/`

文件所有权**不重叠**，但 `p7` 依赖 `p0`（runner 收口）且 `p1` 也依赖 `p0`。**为何 p7 不依赖 p1？**

**证据**: 交付票 `work-p7-observability` 的 `dependsOn: ["work-p0-loop-consolidation"]`，未列 `p1`。

**影响**: 若 p7 实际需要图引擎节点钩子，分组 2 的并行假设失效。

**建议**: 
1. 在 `v3-platform.md` §10 补充并行阶段的职责边界说明
2. 或在 `p7` 完成标准中明确「不改 runner 接口，仅接线现有零件」

---

### I3. 部分验收标准不可测

**问题**: 3 处完成标准含模糊或不可验证表述。

| 工作项 | 模糊标准 | 不可测原因 |
|---|---|---|
| `work-p1-graph-engine` | "引擎核心非测试 LOC 控制在 1500 以内" | 何为"核心"？若 `scheduler.go` 1600 行但其他文件小，算超吗？ |
| `work-p5-event-hitl` | "Session 对外 JSON 结构不变" | 未定义"不变"的精确含义：字段顺序？nil vs omitempty？ |
| `work-p9-release` | "对比文档说明为何不引入 Pregel BSP" | 何为"说明"足够？一句话？一页？ |

**证据**: 上述引用来自 `v3-delivery-ticket.json` 各工作项的 `completionCriteria`。

**影响**: 交付时可能因标准歧义产生争议。

**建议**: 改为可执行验证：
- P1 LOC → `find pkg/hno/graph -name '*.go' ! -name '*_test.go' | xargs wc -l | tail -1` 输出 ≤ 1500
- P5 不变 → `internal/session/contract` 9 个测试全绿（已有此条，但应作为唯一判据）
- P9 说明 → 新增 `docs/design/v3-pregel-decision.md` 文件且 ≥ 500 字

---

## 4. 次要问题 / 改进建议

### M1. `work-p5` 与决策票 D1 的表述冲突

**问题**: 
- D1 裁定「不改 Session 对外 JSON」
- 但 `work-p5` 完成标准含「Session 对外 JSON 结构不变」

两处表述一致，但 `work-p5` 的完成标准还包括「事件类型仅 7 种」，而 D1 未说明事件内部化是否需修改 `Session` 的**内部字段**（如 `session.State`）。

**建议**: 在 D1 补充「事件流作为 internal 存储形态，RunOutput 派生视图保持不变，session.State 字段可内部重构」以消除歧义。

---

### M2. Frontier.ready 包含无依赖但非独立的项

**问题**: `frontier.ready` 含 `work-p8-store`，其无依赖，但 `v3-platform.md` §8 标注「此项引入全新概念，范围最大」。

按常理，「范围最大」的项应在其他风险项稳定后启动（如 P0 收口后）。但交付票允许它与 P0 并行。

**影响**: 若 P8 先完成而 P0/P1 延期，Store API 可能因后续图引擎集成需求而二次修改。

**建议**: 考虑把 P8 的 `dependsOn` 改为 `["work-p1-graph-engine"]`，使其在图引擎 v1 完成后再设计存储接口，避免抢跑。

---

### M3. 缺少 BC 变更影响面的量化

**问题**: `v3-platform.md` 和决策票多次提及「破坏性变更 BC-1 至 BC-5」，但交付票未列出：
- 每项 BC 影响哪些外部 API
- 是否有用户代码需迁移
- 迁移工作量预估

**建议**: 在 `work-p9-release` 的 `evidenceRefs` 中新增 `artifact:docs/design/v3-breaking-changes.md`，列出 5 项 BC 的 API diff 与迁移检查清单。

---

## 5. 依赖图正确性核查

### 5.1 环、悬空依赖、自依赖

✅ **通过**。遍历全部 9 个工作项：
- 无环：依赖链最长为 `P0 → P1 → P2 → P6 → P9`（4 层）
- 无悬空：所有 `dependsOn` 引用的 ID 均存在
- 无自依赖

---

### 5.2 Frontier.ready 的依赖满足性

✅ **通过**。`frontier.ready` 含 2 项：
- `work-p0-loop-consolidation`: `dependsOn: []` ✓
- `work-p8-store`: `dependsOn: []` ✓

`frontier.blocked` 的 7 项均有未满足依赖，阻塞理由与 `dependsOn` 一致。

---

### 5.3 并行分组的依赖一致性

✅ **通过**（结构上）。5 组分组：
```json
[
  ["work-p0-loop-consolidation"],
  ["work-p1-graph-engine", "work-p4-streammode", "work-p7-observability", "work-p8-store"],
  ["work-p2-node-policies", "work-p3-send-durability", "work-p5-event-hitl"],
  ["work-p6-workflow-migration"],
  ["work-p9-release"]
]
```

逐组验证：
- 组 1：P0 无依赖 ✓
- 组 2：P1/P4/P7 依赖 P0，P8 无依赖，组内无相互依赖 ✓
- 组 3：P2/P3 依赖 P1，P5 依赖 P0+P1，组内无相互依赖 ✓
- 组 4：P6 依赖 P1/P2/P3/P4（组 2+3 全部），单项 ✓
- 组 5：P9 依赖 P6/P7/P8（前 4 组中必要项），单项 ✓

**但**：见重要问题 I2，P7 与 P1 的并行性依据不足。

---

## 6. 交付票与设计文档一致性

### 6.1 工作项与设计文档 §10 的对应

✅ **通过**。交付票 9 项工作与 `v3-platform.md` §2 范围表的 G1–G10 一一对应：

| 交付票 | 设计文档 | 标题匹配 |
|---|---|---|
| work-p0 | G1 | agent 循环收口 ✓ |
| work-p1 | G2 | 图引擎 v1 ✓ |
| work-p2 | G3 | 节点策略四合一 ✓ |
| work-p3 | G5+G6 | Send 扇出 + Durability ✓ |
| work-p4 | G4 | StreamMode 七模式 ✓ |
| work-p5 | G7 | 会话事件化 + HITL ✓ |
| work-p6 | G10 | workflow 迁移 ✓ |
| work-p7 | G9 | 观测接线 ✓ |
| work-p8 | G8 | Store 长期记忆 ✓ |
| work-p9 | — | 发布（对应 §11） ✓ |

---

### 6.2 决策票 D1–D4 在交付票中的体现

✅ **通过**（除 M1 表述冲突外）。

| 决策 | 交付票体现 | 一致性 |
|---|---|---|
| D1 不改 Session JSON | `work-p5` 完成标准第 2 条 | ✓（见 M1 需澄清内部字段） |
| D2 Store 进 v3.0 | `work-p8` 独立工作项 | ✓ |
| D3 graph.Node 用 any | `work-p1` 完成标准「Node.Run 签名 (ctx, any) (any, error)」 | ✓ |
| D4 P0 先发 minor | `work-p0` 完成标准第 5 条 | ✓ |

---

## 7. 已核实为正确的部分

以下基线证据与实测一致：

| 项 | 声称值 | 实测命令 | 结果 |
|---|---|---|---|
| 当前版本 | 1.2.9 | `git describe --tags` | ✓ 一致 |
| 非测试 LOC | 43,684 | `find . -name '*.go' ! -name '*_test.go' | xargs wc -l` | ✓ 一致 |
| 测试 LOC | 34,554 | `find . -name '*_test.go' | xargs wc -l` | ✓ 一致 |
| 模型 provider 数 | 17 | `ls -d pkg/hno/models/*/` | ✓ 一致 |
| 契约测试数 | 9 | `grep -c '^func Test' internal/session/contract/contract_test.go` | ✓ 一致 |
| runner 外部引用 | 0 | `grep -rn 'hno/runner' . | grep -v '^./pkg/hno/runner/'` | ✓ 一致 |
| runner 测试包名 | `package runner` | `grep '^package' pkg/hno/runner/*_test.go` | ✓ 一致（同包测试） |
| 循环行号 | run.go:60 / stream.go:259 / runner.go:180 | `grep -n` | ✓ 全部一致 |
| observability 未接线 | 零调用 | `grep observability.Retry` | ✓ 无输出（未接线） |
| openspec/ 不存在 | — | `ls openspec/` | ✓ No such file |

---

## 8. 无法核实的部分

### 8.1 参考仓库数据（阻塞 B2）

- adk-go workflow LOC（声称 6,244 / 13,800）
- LangGraph 全仓库 LOC（声称 19 万）
- LangGraph 内核 LOC（声称 4,400）
- adk-go 图引擎 6 文件 LOC（声称 2,134）

**原因**: 仓库路径不可访问或命令需环境批准。

---

### 8.2 LangGraph 独有能力位置（§6）

`v3-baseline-evidence.md` §6 声称：
- `Send` 在 `types.py:732`
- `Durability` 在 `types.py:98`
- `StreamMode` 在 `types.py:131`

**原因**: `/tmp/fwcmp/langgraph` 不存在，无法 `sed -n '732p'` 核实。

---

### 8.3 破坏性变更影响面（§7）

声称 `RunStreamResult` 无外部消费方，已核实 `grep` 无输出 ✓。但未能验证：
- 是否有**未 import 但间接依赖**的场景
- `output.Events` 的 3 处消费方是否真能增量兼容

**建议**: 在 P4/P5 实施时补充集成测试，覆盖 team/workflow 的既有流式场景。

---

## 9. 最终建议

### 修复优先级

1. **立即修复**（阻塞）:
   - B1 跑通 schema 校验或提供截图
   - B2 补充参考仓库克隆步骤或改引用 GitHub 统计

2. **发布前修复**（重要）:
   - I1 更新基线证据中的 3 处偏差
   - I2 明确 P7 与 P1 的并行边界
   - I3 具体化 3 处验收标准

3. **迭代改进**（次要）:
   - M1 澄清 Session 内部字段重构范围
   - M2 考虑调整 P8 依赖顺序
   - M3 补充 BC 变更清单文档

---

## 附录：重跑命令清单

供后续审查者核验：

```bash
# 版本
git describe --tags  # → v1.2.9

# LOC
find . -name '*.go' ! -name '*_test.go' ! -path './website/*' | xargs wc -l | tail -1  # → 43,684
find . -name '*_test.go' ! -path './website/*' | xargs wc -l | tail -1               # → 34,554
wc -l pkg/hno/skills/*.go | tail -1                                                    # → 684

# 模块统计
ls -d pkg/hno/models/*/ | wc -l                                                        # → 17
grep -c '^func Test' internal/session/contract/contract_test.go                        # → 9
grep -rn 'hno/runner' --include='*.go' . | grep -v '^./pkg/hno/runner/' | wc -l      # → 0

# 循环位置
grep -n "for loopCount" pkg/hno/agent/run.go      # → 60
grep -n "for {" pkg/hno/agent/stream.go | head -1 # → 259
grep -n "for {" pkg/hno/runner/runner.go | head -1# → 180
grep -n "run.Loop" pkg/hno/team/team.go | head -1 # → 295（注释）

# observability 接线
grep -rn 'observability.Retry\|observability.CircuitBreaker' pkg/ --include='*.go' | grep -v test  # → 空

# 契约
ls openspec/  # → No such file
```

---

**审查签名**: 独立审查员  
**状态**: 有条件通过，修复 B1/B2 后可执行
