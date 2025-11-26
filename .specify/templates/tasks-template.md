---

description: "功能实施任务模板"
---

# 任务清单：[FEATURE NAME]

**输入**：`/specs/[###-feature-name]/` 中的设计文档
**前置**：plan.md（必需）、spec.md（用户故事必需）、research.md、data-model.md、contracts/

**测试**：所有功能必须交付对应的 Go 单元/契约/供应商集成测试；示例请根据规格替换。

**覆盖率**：`make test`、`make providers-test`、`make coverage` 必须可运行并产出 ≥85% 的综合覆盖率；任务需写明测试文件与命令。

**栈约束**：任务必须指向真实路径，覆盖 Go 模块（`go/internal/...`、`go/pkg/...`、`go/tests/...`）、供应商适配器（`go/pkg/providers/<provider>/`）、治具/契约（`specs/.../contracts/fixtures`）、基准/报告（`specs/.../artifacts/`）、脚本（`scripts/`）以及 `Makefile`（自动化入口）。禁止将 `./agno` Python 代码作为运行时依赖。

**文档与多语言**：涉及文档的任务必须指向 VitePress 工程中的真实路径（或对应仓库），说明影响的路由/侧边栏节点，并标明目标语言（en/zh/ja/ko）；如暂缺翻译，需在任务中显式标记待补项并规划补齐顺序。

**组织方式**：任务按用户故事分组，确保每个故事都能独立实现与测试。

## 格式：`[ID] [P?] [Story] 描述`

- **[P]**：可并行执行（不同文件，且无依赖）
- **[Story]**：所属用户故事（如 US1、US2、US3）
- 描述中必须包含精确文件路径

## 路径约定

- **核心运行时（Go）**：`go/cmd/`（入口）、`go/internal/{agent,runtime,model,memory,tool}/`（核心能力）、`go/pkg/`（可复用组件）、`go/tests/{contract,providers,bench}/`（测试）。
- **模型供应商适配器**：`go/pkg/providers/<provider>/`（ollm/Gemini/OpenAI/GLM4/OpenRouter/SiliconFlow/Cerebras/ModelScope/Groq 客户端）及其配置/错误映射。
- **状态/工具**：`go/pkg/memory/`、`go/pkg/tools/` 或等价目录；需保持与核心接口一致。
- **治具与基准**：`specs/<feature>/contracts/fixtures/`（Python 对照治具）、`specs/<feature>/contracts/deviations.md`、`specs/<feature>/artifacts/`（覆盖率/基准/报告）。
- **自动化与配置**：`Makefile`（fmt/lint/test/providers-test/coverage/bench/gen-fixtures/release）、`.env.example`（供应商变量占位）、`scripts/`（Go/标准工具脚本）。
- **文档（VitePress 官方文档站）**：VitePress 文档工程源码（仓库/目录由项目说明或规格定义）；任务中需引用具体路径与受影响的语言（en/zh/ja/ko），并与 `specs/<feature>/quickstart.md` 对齐。
- 下文示例均以该结构为例，交付时请替换为真实路径。

<!-- 
  ============================================================================
  重要：以下任务仅作示例。
  /speckit.tasks 必须替换为基于以下信息生成的真实任务：
  - spec.md 中带优先级的用户故事（P1、P2、P3...）
  - plan.md 描述的功能需求
  - data-model.md 中的实体
  - contracts/ 中的端点
  任务必须按用户故事组织，以便：
  - 独立实现
  - 独立测试
  - 按 MVP 增量交付
  不要在最终 tasks.md 中保留这些示例。
  ============================================================================
-->

## Phase 1: Setup（共享基础设施）

**目的**：搭建 Go-only 骨架、自动化入口、供应商/治具配置

- [ ] T001 在 `go/` 初始化 Go module（`go.mod`）、`cmd/agno/main.go` 与 `internal/{agent,runtime,model,memory,tool}/` 目录
- [ ] T002 [P] 扩展根 `Makefile`：`fmt`、`lint`、`test`、`providers-test`、`coverage`、`bench`、`gen-fixtures`、`release`，并添加 `help` 说明
- [ ] T003 [P] 在 `.env.example` 中列出 ollm、Gemini、OpenAI、GLM4、OpenRouter、SiliconFlow、Cerebras、ModelScope、Groq 的必需/可选变量，附备注
- [ ] T004 [P] 配置 `golangci-lint`, `gofumpt`, `go test ./... -coverpkg=./...`, `benchstat` 等基础工具，`Makefile` 中映射目标
- [ ] T005 创建/更新 `.github/workflows/ci.yml`，复用 make 目标（fmt/lint/test/providers-test/coverage/bench），输出报告到 `specs/<feature>/artifacts/`
- [ ] T006 [P] 在 `scripts/` 添加 Go/标准工具脚本用于生成脱敏 fixtures（读取 `./agno` 参考输出），并在 `specs/<feature>/contracts/fixtures/` 放置示例
---

## Phase 2: Foundational（阻塞性前置）

**目的**：准备核心接口、供应商路由、治具/契约/基准基础，解除后续故事阻塞

**⚠️ 关键**：在此阶段完成前，禁止开始用户故事任务。

示例（按项目调整）：

- [ ] T007 在 `go/internal/agent/` 定义 Agent、Workflow/Step Engine、Memory/Tool/MCP 接口与公共类型
- [ ] T008 [P] 在 `go/internal/model/` 定义模型客户端接口、错误规约与路由器，提供 `go/pkg/providers/mock` 作为占位实现
- [ ] T009 [P] 在 `go/tests/contract/` 添加契约/fixture 加载与 golden 断言骨架，引用 `specs/<feature>/contracts/fixtures/`
- [ ] T010 [P] 实现 `make providers-test` 框架：读取 `.env`，串行/并行调用供应商 stub，输出报告到 `specs/<feature>/artifacts/`
- [ ] T011 在 `Makefile` 添加 `constitution-check` 聚合目标，串联 fmt/lint/test/providers-test/coverage/bench

**检查点**：基础完备，可开始并行处理用户故事

---

## Phase 3: 用户故事 1 - [标题]（优先级：P1）🎯 MVP

**目标**：[该故事交付的能力]

**独立测试**：[如何单独验证该故事]

### 用户故事 1 的测试（可选）⚠️

> **注意：若包含测试，先编写并确保失败，再实现功能。**

- [ ] T012 [P] [US1] 在 `go/tests/contract/<context>_contract_test.go` 中编写契约/golden 测试，引用 `specs/<feature>/contracts/fixtures/<case>.json`
- [ ] T013 [P] [US1] 在 `go/tests/providers/<provider>_integration_test.go` 中编写供应商集成测试（需真实 key），验证 chat/流式/错误路径

### 用户故事 1 的实现

- [ ] T014 [P] [US1] 在 `go/internal/<context>/` 中实现核心逻辑/接口适配（如 Agent/Workflow/Memory/Tool）
- [ ] T015 [P] [US1] 在 `go/pkg/providers/<provider>/` 中实现/扩展供应商客户端与错误映射
- [ ] T016 [US1] 在 `go/internal/runtime/` 或 `cmd/agno/` 中公开 API/CLI，并与路由/中间件对齐
- [ ] T017 [US1] 更新 `specs/<feature>/contracts/fixtures/` 与 `specs/<feature>/contracts/deviations.md`，同步 `specs/<feature>/quickstart.md` 示例
- [ ] T018 [US1] 更新 `Makefile` 相关目标与日志，确保 `make providers-test`/`make coverage` 覆盖新能力

**检查点**：故事 1 应可独立运行并测试

---

## Phase 4: 用户故事 2 - [标题]（优先级：P2）

**目标**：[该故事交付的能力]

**独立测试**：[如何单独验证]

### 用户故事 2 的测试（可选）⚠️

- [ ] T019 [P] [US2] 在 `go/tests/contract/<context>_contract_test.go` 中补充契约/golden 测试，覆盖新增参数/分支
- [ ] T020 [P] [US2] 在 `go/tests/providers/<provider>_integration_test.go` 中扩展供应商集成测试，验证真实 key 下的新能力

### 用户故事 2 的实现

- [ ] T021 [P] [US2] 在 `go/internal/<context>/` 中扩展用例逻辑/接口，保持与 US1 可独立启停
- [ ] T022 [US2] 在 `go/pkg/providers/<provider>/` 中追加新 API/参数支持或路由策略
- [ ] T023 [US2] 在 `go/internal/runtime/` 或 `cmd/agno/` 暴露相关端点/CLI 选项，并对接新契约
- [ ] T024 [US2] 更新 `specs/<feature>/contracts/fixtures/`、`deviations.md` 与 `quickstart.md`，同步 `Makefile`/脚本变更

**检查点**：故事 1 与 2 均可独立运行

---

## Phase 5: 用户故事 3 - [标题]（优先级：P3）

**目标**：[该故事交付的能力]

**独立测试**：[如何单独验证]

### 用户故事 3 的测试（可选）⚠️

- [ ] T025 [P] [US3] 在 `go/tests/contract/<context>_contract_test.go` 中编写契约/golden 测试，覆盖边界/异常
- [ ] T026 [P] [US3] 在 `go/tests/providers/<provider>_integration_test.go` 中编写集成测试，验证异常映射/超时/重试

### 用户故事 3 的实现

- [ ] T027 [P] [US3] 在 `go/internal/<context>/` 中建模实体/状态机或并发策略
- [ ] T028 [US3] 在 `go/internal/runtime/` 或 `cmd/agno/` 中实现服务入口/协议层（HTTP/gRPC/CLI），保持可独立启停
- [ ] T029 [US3] 在 `go/pkg/providers/<provider>/` 或 `go/pkg/memory/` 中实现依赖适配器
- [ ] T030 [US3] 更新 `Makefile`、`scripts/` 与 `specs/<feature>/contracts/fixtures/`、`artifacts/`，确保该模块可单独验证

**检查点**：所有用户故事可独立运行

---

[如需更多故事，按同样模式扩展]

---

## Phase N: 抛光与跨领域事项

**目的**：影响多个用户故事的改进

- [ ] TXXX [P] 更新 specs/ 文档或外部文档（如 quickstart）
- [ ] TXXX 代码清理与重构
- [ ] TXXX 全局性能优化
- [ ] TXXX [P] 如需新增的单元测试（tests/unit/）
- [ ] TXXX 安全加固
- [ ] TXXX 验证 quickstart.md 场景

---

## 覆盖率 Gate（所有故事完成后执行）

- [ ] T041 运行 `make test`, `make providers-test`, `make coverage`, 如需性能验证再运行 `make bench`，确保综合覆盖率 ≥85%，并上传覆盖率/基准报告
- [ ] T042 在 `specs/<feature>/artifacts/coverage.txt`（或 CI 构建工件）中更新覆盖率、供应商测试与基准结果链接

---

## 依赖与执行顺序

### 阶段依赖

- **Setup（Phase 1）**：无依赖，可立即开始
- **Foundational（Phase 2）**：依赖 Setup，阻塞所有用户故事
- **用户故事（Phase 3+）**：均依赖 Foundational 完成
  - 若人手充足，可并行
  - 否则按优先级顺序（P1→P2→P3）串行
- **Polish（最终阶段）**：依赖所有目标用户故事完成

### 用户故事依赖

- **US1 (P1)**：Foundational 后即可开始，无其他故事依赖
- **US2 (P2)**：Foundational 后即可开始，若需与 US1 集成也应保持可独立测试
- **US3 (P3)**：同上

### 单个故事内的顺序

- 若有测试，必须“先写测试并看到失败”
- 模型 → 服务 → 端点
- 核心实现完成后再做整合
- 完成一个故事后再进入下一优先级

### 可并行机会

- 所有标记 [P] 的 Setup/Foundational 任务
- Foundational 完成后，各用户故事可并行
- 同一故事内标记 [P] 的测试/模型可并行
- 不同用户故事可由不同成员并行推进

---

## 并行示例：用户故事 1

```bash
# 若需要测试，可同时启动：
Task: "在 go/tests/contract/<context>_contract_test.go 编写契约/golden 测试"
Task: "在 go/tests/providers/<provider>_integration_test.go 编写供应商集成测试"

# 可同时创建的模型：
Task: "在 go/internal/<context>/domain/<aggregate>.go 定义实体/状态"
Task: "在 go/pkg/providers/<provider>/client.go 实现 HTTP 客户端与错误映射"
```

---

## 实施策略

### MVP 优先（仅交付故事 1）

1. 完成 Phase 1
2. 完成 Phase 2（⚠️ 阻塞）
3. 完成 Phase 3（US1）
4. **暂停并验证**：独立测试 US1
5. 若准备好，可部署/演示

### 渐进式交付

1. Setup + Foundational → 基础完成
2. 加入 US1 → 独立测试 → 部署/演示（MVP）
3. 加入 US2 → 独立测试 → 部署/演示
4. 加入 US3 → 独立测试 → 部署/演示
5. 每个故事都能带来增量价值且不破坏前面成果

### 并行团队策略

1. 团队协作完成 Setup + Foundational
2. 基础完成后：
   - 开发者 A：US1
   - 开发者 B：US2
   - 开发者 C：US3
3. 各故事独立完成并集成

---

## 备注

- 标记 [P] 表示不同文件、无依赖，可并行
- [Story] 标签便于溯源到具体用户故事
- 每个故事都应可独立完成与测试
- 若写测试，必须先失败后实现
- 建议每个任务或逻辑组完成后提交
- 任意检查点都可暂停做独立验证
- 避免模糊任务、同文件冲突、跨故事依赖打破独立性
