# 实现计划：Go 版 Agno Agents 重构

**分支**：`001-agno-agents-refactor` | **日期**： 2025-11-26 | **规格**： `/Users/rex/cool.cnb/agno-Go/specs/001-agno-agents-refactor/spec.md`
**输入**：来自 `/Users/rex/cool.cnb/agno-Go/specs/001-agno-agents-refactor/spec.md` 的功能规格

**说明**：该计划按 `/speckit.plan` 流程生成，覆盖 Phase 0–2；后续实现任务将在 `/Users/rex/cool.cnb/agno-Go/specs/001-agno-agents-refactor/tasks.md` 中细化。

## 摘要

聚焦 docs.agno.com/basics 的五个核心场景（基础 agent、记忆会话、RAG、工具+HITL、工作流分支），交付 Go 版 CLI/SDK 入口、治具与 quickstart 运行指南，验证与 Python 版一致。供应商矩阵覆盖 Ollama、Gemini、OpenAI、GLM4、OpenRouter、SiliconFlow、Cerebras、ModelScope、Groq，缺 key/不可达自动跳过并记录。以 Make 为唯一入口（含 constitution-check + fixtures 校验、VitePress 构建），CI 复用同一入口。性能相对 `specs/001-agno-agents-refactor/artifacts/baseline/python-bench.json` p95 -20%、峰值内存 -25%；综合覆盖率与五场景通过率写入 `specs/001-agno-agents-refactor/artifacts/coverage.txt`，<95% 或 PII/Prompt 注入漏/误拦截即阻断。

## 技术背景

**语言/版本**： Go 1.25.1（唯一运行时）  
**主要依赖**： 标准库 `net/http` + `github.com/go-chi/chi/v5` 路由/中间件；质量工具 `gofumpt`、`golangci-lint`、`benchstat`  
**存储**： `MemoryStore` 抽象（内存默认，Bolt/Badger 可选）；无外部 DB 强依赖  
**测试**： `go test` 驱动，Make 目标：`fmt`、`lint`、`test`、`providers-test`、`coverage`、`bench`、`gen-fixtures`、`release`、`constitution-check`；契约/供应商/基准测试位于 `go/tests/{contract,providers,bench}/`  
**目标平台**： Linux/macOS（CLI/服务模式），CI 同步  
**项目类型**： Go CLI + 库 + 契约/集成测试  
**性能目标**： 相对 Python 基线 p95 延迟 -20%，峰值内存 -25%；供应商示例成功或正确跳过 ≥90%；E2E 五场景通过率 ≥95%  
**约束条件**： 纯 Go，不允许 cgo/FFI/子进程桥接 Python；认证仅 API Key Header，中间件仅日志/恢复/超时；RAG 不可用时需回退提示/错误；密钥脱敏，日志落在 artifacts 下  
**规模/范围**： docs.agno.com/basics 五场景 + 九家模型供应商（chat/stream/embedding），综合测试覆盖率 ≥85%

## 宪章检查

*Gate：Phase 0 调研前必须通过；Phase 1 设计后需再次检查。*

- [x] **纯 Go / 禁止桥接**：迁移 Python Basics 能力到 `go/internal/{agent,runtime,model,memory,tool}/` 与 `go/pkg/`，禁止 cgo/FFI/子进程调用 `./agno`；Python 仅用于离线治具生成，运行时与测试消费 Go 侧 fixtures。
- [x] **模型供应商矩阵（Ollama、Gemini、OpenAI、GLM4、OpenRouter、SiliconFlow、Cerebras、ModelScope、Groq）**：统一 `Chat`/`Embedding`/stream 接口；`.env.example` 填入 key/endpoint/model 变量；`make providers-test` 跳过缺 key/不可达并记录 `/Users/rex/cool.cnb/agno-Go/specs/001-agno-agents-refactor/artifacts/coverage/providers.log`。
- [x] **契约/治具与基准**：治具位于 `/Users/rex/cool.cnb/agno-Go/specs/001-agno-agents-refactor/contracts/fixtures`；偏差记录在 `contracts/deviations.md`；性能基准比对 `artifacts/baseline/python-bench.json`，未达标需标注 owner/补救并返回非零；`make constitution-check` 包含 fixtures 校验、差异报告与日志输出到 `/Users/rex/cool.cnb/agno-Go/specs/001-agno-agents-refactor/artifacts/coverage.txt`，运行时不依赖 Python。
- [x] **自动化与 Make**：入口为 Makefile，覆盖 `fmt`/`lint`/`test`/`providers-test`/`coverage`/`bench`/`gen-fixtures`/`release`/`constitution-check`；新增 VitePress 构建目标（T001/T027，Make 触发）纳入 `.github/workflows/ci.yml`；CI 调用 `make constitution-check` 复用本地流程。
- [x] **测试纪律 + 85% 覆盖率**：所有 Go 包含 `_test.go`；契约/供应商/基准测试纳入 `go/tests/`；综合覆盖率 ≥85%；五场景通过率聚合（扩展 T011 或新任务）写入 `artifacts/coverage.txt`，<95% 阻断并生成差异报告；PII/Prompt 注入测试漏/误拦截同样阻断并记录案例。
- [x] **密钥与安全**：`.env.example` 全量列出供应商变量；仅 API Key Header 认证，中间件限日志/恢复/超时；测试日志脱敏并落在 `specs/001-agno-agents-refactor/artifacts/`；未配置密钥自动跳过并记录原因。

## 项目结构

### 文档（当前功能）

```text
/Users/rex/cool.cnb/agno-Go/specs/001-agno-agents-refactor/
├── plan.md              # 本文件（/speckit.plan 输出）
├── research.md          # Phase 0 输出（已存在，需更新未知项时补充）
├── data-model.md        # Phase 1 输出
├── quickstart.md        # Phase 1 输出
├── contracts/           # Phase 1 输出（治具/契约）
├── artifacts/           # 覆盖率/基准/日志（含 coverage.txt、baseline/python-bench.json）
└── tasks.md             # Phase 2 输出（/speckit.tasks 创建）
```

### 源码（仓库根目录）

```text
/Users/rex/cool.cnb/agno-Go/agno/        # Python 参考实现（只读，不可被 Go 运行时调用）
/Users/rex/cool.cnb/agno-Go/go/
├── cmd/agno/                 # Go 入口（CLI/服务）
├── internal/
│   ├── agent/                # Agent/Workflow/Step Engine
│   ├── runtime/              # 服务编排、协议层
│   ├── model/                # 模型接口定义与路由
│   ├── memory/               # 状态/存储接口
│   └── tool/                 # 工具/MCP/拦截器
├── pkg/
│   ├── providers/<provider>/ # 模型供应商适配器（Ollama/Gemini/OpenAI/GLM4/OpenRouter/SiliconFlow/Cerebras/ModelScope/Groq）
│   ├── memory/               # 具体存储实现
│   └── tools/                # 额外可插拔组件
└── tests/
    ├── contract/             # 契约/golden
    ├── providers/            # 供应商集成
    └── bench/                # 基准

/Users/rex/cool.cnb/agno-Go/specs/001-agno-agents-refactor/
├── plan.md | research.md | data-model.md | quickstart.md
├── contracts/fixtures/      # Python 治具（脱敏）
├── contracts/deviations.md  # 与 Python 差异
└── artifacts/               # 覆盖率/基准/报告

/Users/rex/cool.cnb/agno-Go/scripts/      # Go/标准工具脚本（如治具生成）
/Users/rex/cool.cnb/agno-Go/.env.example  # 供应商 env 占位
/Users/rex/cool.cnb/agno-Go/Makefile      # 单一入口（fmt/lint/test/providers-test/coverage/bench/gen-fixtures/release/constitution-check/docs）
```

**结构决策**：沿用现有 `go/` 单仓布局与 `specs/001-agno-agents-refactor/` 文档/治具目录；供应商适配器集中在 `go/pkg/providers/*`，测试集中于 `go/tests/`，所有自动化通过 Make 入口与 `.github/workflows/ci.yml` 复用。

## 阶段计划

### Phase 0 - 大纲与调研（已完成，补充时更新 research.md）
- `research.md` 已覆盖当前 Clarifications，未留 NEEDS CLARIFICATION；若新增疑问，按模板追加 Decision/Rationale/Alternatives。
- 确认宪章 Gate 通过；无禁止项。

### Phase 1 - 设计与契约
- 更新 `data-model.md` 以细化 Session/Memory/KG/Workflow 实体与状态、知识库不可用回退（CG4）。
- 扩展 `quickstart.md`，写明五个 Basics 场景的 CLI/SDK 运行步骤、所需 env、治具关联（CG1），并指向 `contracts/fixtures` 与 `artifacts/coverage.txt` 聚合。
- 契约/治具：使用 `make gen-fixtures`（或 `go run ./go/scripts/gen_provider_baseline`）更新 fixtures，缺口写入 `contracts/deviations.md`；补充 PII/Prompt 注入拦截测试（CG3），结果计入 `artifacts/coverage.txt` 并阻断漏/误拦截。
- 运行 `.specify/scripts/bash/update-agent-context.sh codex` 追加本次计划涉及的新技术/依赖。

### Phase 2 - 任务规划（到此停止执行）
- 任务编排：在 `tasks.md`（/speckit.tasks）合并/分阶段模型路由任务 T009/T020（接口先行，再能力配置，记录依赖，DP1）与 metrics 任务 T008/T025（接口→实现/接线，DP2）。
- 基准与性能：在 T007/T023/T033 明确对比 `artifacts/baseline/python-bench.json`，校验 p95 -20%/峰值内存 -25%，未达标写入 `contracts/deviations.md` 并指派 owner/下一步（DG1）。
- 供应商与安全：扩展 T011 或新增聚合任务统计五场景通过率并输出至 `artifacts/coverage.txt`，<95% 阻断并生成差异报告（CG2）；`make providers-test` 跳过原因写入 `artifacts/coverage/providers.log`。
- 自动化：在 Makefile 增加/校验 `constitution-check`（含 fixtures 校验、日志到 `artifacts/coverage.txt`，CA1），缺口时补充 `make gen-fixtures`；新增 VitePress 构建目标并在 CI 触发（IC1）。
- 回退策略：新增知识库不可用回退的实现/测试任务，期望输出提示级回答或明确错误（CG4）。
- 认证/中间件：在任务中细化 FR-004 验收（仅 API Key Header；允许中间件：日志/恢复/超时；禁止 Basic/OAuth/JWT/自定义拦截）。

## 复杂度追踪

当前无宪章违规或必须说明的复杂度项，保持空表以供后续记录。

| 违规项 | 必要原因 | 更简单方案被拒绝的理由 |
|--------|----------|--------------------------|
| 无 | N/A | N/A |
