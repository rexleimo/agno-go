# 任务清单：Go 版 Agno Agents 重构（Basics 聚焦）

**输入**：`/Users/rex/cool.cnb/agno-Go/specs/001-agno-agents-refactor/` 设计文档（spec/plan/research/data-model/quickstart/contracts）

## Phase 1: Setup（共享基础设施）

- [X] T001 更新 `/Users/rex/cool.cnb/agno-Go/Makefile`：`constitution-check` 聚合 fmt/lint/test/providers-test/coverage/bench/gen-fixtures（含 fixtures 校验），将日志写入 `/Users/rex/cool.cnb/agno-Go/specs/001-agno-agents-refactor/artifacts/coverage.txt`；新增 `docs` 目标（VitePress build）；帮助说明补全。 
- [X] T002 [P] 调整 `/Users/rex/cool.cnb/agno-Go/.github/workflows/ci.yml`：调用 `make constitution-check`（含 `make gen-fixtures` 兜底），确保 artifacts/coverage.txt 与 fixtures 校验日志上传。 
- [X] T003 [P] 补全 `/Users/rex/cool.cnb/agno-Go/.env.example` 九家供应商变量（Ollama/Gemini/OpenAI/GLM4/OpenRouter/SiliconFlow/Cerebras/ModelScope/Groq），校正命名并标注必需/可选/默认模型。 
- [X] T004 [P] 更新 `/Users/rex/cool.cnb/agno-Go/scripts/gen-fixtures.sh` 与 `go/scripts/gen_provider_baseline`：默认输出到 `specs/001-agno-agents-refactor/contracts/fixtures/`，支持 VERIFY_ONLY，失败时写入 `contracts/deviations.md` 并退出非零。 
- [X] T005 [P] 确认 `/Users/rex/cool.cnb/agno-Go/specs/001-agno-agents-refactor/contracts/fixtures/`、`contracts/deviations.md`、`artifacts/coverage.txt`、`artifacts/bench.txt`、`artifacts/baseline/python-bench.json` 存在；若缺失则创建占位并记录来源。 

## Phase 2: Foundational（阻塞前置）

- [X] T006 建立契约测试骨架 `/Users/rex/cool.cnb/agno-Go/go/tests/contract/basics_contract_test.go`：加载 fixtures、比对输出，差异写入 `contracts/deviations.md`，将通过率/差异摘要写入 `artifacts/coverage.txt`。 
- [X] T007 [P] 建立供应商集成测试框架 `/Users/rex/cool.cnb/agno-Go/go/tests/providers/providers_integration_test.go`：读取 `.env`，缺 key/不可达自动跳过并记录到 `artifacts/coverage/providers.log`，汇总摘要写入 `artifacts/coverage.txt`。 
- [X] T008 [P] 建立基准模板 `/Users/rex/cool.cnb/agno-Go/go/tests/bench/basics_bench_test.go`：采集 p95/峰值内存，输出 `artifacts/bench.txt`，预留对比 `artifacts/baseline/python-bench.json` 的接口。 
- [X] T009 [P] 定义运行时观测接口 `/Users/rex/cool.cnb/agno-Go/go/internal/runtime/metrics/metrics.go`：暴露模型/工具延迟、token、内存、guardrail 事件，供 bench/coverage 使用。 
- [X] T010 定义模型路由接口 `/Users/rex/cool.cnb/agno-Go/go/internal/model/router.go`：声明能力（chat/stream/embedding）、缺密钥/不可达跳过策略、错误类型；不含具体 provider 配置（留待 US2）。 
- [X] T011 [P] 在 `/Users/rex/cool.cnb/agno-Go/go/internal/knowledge/fallback.go` 设计知识库不可用回退策略接口（提示级回答/明确错误），供 RAG 路径调用。 

## Phase 3: 用户故事 1 - Basics 行为对齐（优先级：P1）🎯 MVP

**目标**：复刻 docs.agno.com/basics 五个核心场景（基础 agent、记忆会话、RAG、工具+HITL、工作流分支）在 Go 侧的 CLI/SDK 行为，支持流式与 guardrails。  
**独立测试**：`go test ./go/tests/contract -run Basics`；五场景通过率 ≥95%，结果写入 `artifacts/coverage.txt`。 

### 测试
- [X] T012 [P] [US1] 在 `go/tests/contract/basics_contract_test.go` 添加五个场景的契约用例（含多模态校验），引用 `contracts/fixtures/`，将单场景结果写入 `artifacts/coverage.txt`。 
- [X] T013 [P] [US1] 编写 CLI/SDK 入口回放测试 `go/tests/contract/cli_sdk_scenarios_test.go`，覆盖流式输出与会话状态回放。 

### 实现
- [X] T014 [P] [US1] 实现 Agent/Team 执行引擎（指令/调试/hook/流式/HITL）于 `go/internal/runtime/engine.go`，与 Tool/MCP/Guardrail 接口对接。 
- [X] T015 [P] [US1] 实现 Session/Memory 管理（读写/搜索/回滚/TTL/共享策略）骨架：`runtime.Service` 通过 Engine + Store 写入历史，保留 TokenWindow，状态机保护。 
- [X] T016 [P] [US1] 实现知识接入与 Agentic RAG 流程于 `go/internal/knowledge/`，集成 `fallback.go` 回退（提示/错误）与多向量库路由（Engine + Strategy）。 
- [X] T017 [P] [US1] 实现工具/MCP/HITL/守护（含 PII/Prompt 注入检测）骨架于 `go/internal/tool/`（Registry/GuardrailConfig/Hook），供 runtime 接入。 
- [X] T018 [P] [US1] 实现工作流/分支/异步编排骨架于 `go/internal/workflow/engine.go`，留多模态校验钩子。 
- [X] T019 [US1] 在 `go/cmd/agno/main.go` 暴露五场景的 CLI/SDK 入口（可选择 fixtures），更新 `/Users/rex/cool.cnb/agno-Go/specs/001-agno-agents-refactor/quickstart.md` 运行步骤与治具关联。 
- [X] T020 [US1] 同步 `contracts/fixtures/` 与 `contracts/deviations.md`（记录与 Python 的差异），确保 PII/Prompt 拦截与流式输出覆盖。 
- [X] T035 [P] [US1] 在 `go/tests/contract/auth_middleware_negative_test.go` 添加认证/中间件负例：仅接受 API Key Header，拒绝 Basic/OAuth/JWT 与自定义拦截；验证 CLI/SDK 入口返回 4xx/错误提示并写入 `artifacts/coverage.txt`。 
- [X] T036 [US1] 在 `go/internal/runtime/`（路由/中间件）与 `go/cmd/agno/` 实施 FR-004 约束：只允许 API Key Header，中间件仅日志/恢复/超时；显式拒绝 Basic/OAuth/JWT/自定义拦截并记录脱敏错误。 

## Phase 4: 用户故事 2 - 供应商与接口演示（优先级：P2）

**目标**：九家模型/向量库适配（chat/stream/embedding）可切换/并行，缺 key/不可达自动跳过并记录；CLI 演示核心能力。  
**独立测试**：`make providers-test`，输出与跳过原因存于 `artifacts/coverage/providers.log` 并汇总至 `artifacts/coverage.txt`。 

### 测试
- [X] T021 [P] [US2] 在 `go/tests/providers/` 为九家供应商补充 chat/stream/embedding 集成用例，包含跳过与错误路径校验，写入 `artifacts/coverage/providers.log`。 

### 实现
- [X] T022 [P] [US2] 在 `go/internal/model/router.go` 配置供应商能力宣告与优先级/回退（接口已于 T010），覆盖 DP1“接口先行、能力后配”。 
- [X] T023 [P] [US2] 扩充 `go/internal/runtime/metrics/` 记录供应商延迟/错误以支持 DP2（接口→实现/接线）。 
- [X] T024 [P] [US2] 在 `go/pkg/providers/{ollama,gemini,openai,glm4,openrouter,siliconflow,cerebras,modelscope,groq}/` 完善客户端、错误映射、能力宣告与缺省模型配置，支持跳过/回退。 
- [X] T025 [US2] 在 `go/cmd/agno/main.go` 与 `go/internal/runtime/` 增加供应商选择/并行演示开关，日志写入 `artifacts/coverage/providers.log`，并更新 `.env.example` 注释（与 T003 对齐）。 
- [X] T026 [US2] 更新 `specs/001-agno-agents-refactor/contracts/deviations.md` 与 `quickstart.md`，记录各供应商差异/跳过条件与运行命令。 

## Phase 5: 用户故事 3 - 质量与基准（优先级：P3）

**目标**：覆盖率 ≥85%，五场景通过率 ≥95%，供应商成功/跳过 ≥90%，PII/Prompt 拦截无漏/误，性能相对 Python 基线 p95 -20% / 峰值内存 -25%，偏差有 owner/补救，未达标即阻断。  
**独立测试**：`make coverage`（含契约+供应商+安全）、`make bench`；`artifacts/coverage.txt`/`bench.txt` 输出达标或阻断。 

### 测试/基准
- [X] T027 [P] [US3] 在 `go/tests/contract/security_contract_test.go` 添加 PII/Prompt 注入拦截用例，漏/误拦截写入 `artifacts/coverage.txt` 并阻断。 
- [X] T028 [P] [US3] 在 `go/tests/contract/rag_fallback_test.go` 添加知识库不可用回退用例，验证提示/错误输出与会话不污染。 
- [X] T029 [P] [US3] 扩展 `go/tests/bench/basics_bench_test.go` 对比 `artifacts/baseline/python-bench.json`，计算 p95/峰值内存降幅；若未达到 -20%/-25% 阈值则写入 `contracts/deviations.md`（含 owner/下一步）并让基准任务返回非零；达标与差异写入 `artifacts/bench.txt`。 
- [X] T030 [P] [US3] 编写聚合脚本 `scripts/coverage/aggregate_basics.go`（或等效）汇总契约/供应商/安全测试与五场景通过率、供应商成功/跳过比例、覆盖率；若五场景 <95% 或供应商 <90% 或覆盖率 <85%，生成差异报告至 `artifacts/coverage.txt` 并返回非零（供 constitution-check/CI 阻断）。 

### 实现/文档
- [X] T031 [US3] 更新 `specs/001-agno-agents-refactor/artifacts/coverage.txt`、`bench.txt` 与 `contracts/deviations.md`，为未达标项标注 owner/下一步；同步 `quickstart.md` 中的验证命令。 
- [ ] T032 [US3] 在 `../rex-agno/agno-Go/docs` 更新 Basics 章节（运行步骤、env、差异、回退/安全说明），并在 CI 前运行 `npm run docs:build`（Make docs 目标）。 

## Phase 6: Polish & Cross-Cutting

- [ ] T033 [P] 复查脱敏与错误提示：检查 `go/internal/tool/`、`go/pkg/providers/` 日志与 `.env.example` 注释，确保密钥不泄露且错误可操作，必要时补充掩码逻辑。 
- [ ] T034 执行最终回归：`make fmt lint test providers-test coverage bench constitution-check`，将最新日志/报告归档至 `specs/001-agno-agents-refactor/artifacts/coverage.txt` 与 `artifacts/bench.txt`。 

## 依赖与执行顺序

- 阶段依赖：Phase 1 → Phase 2 → US1 (P1, MVP) → US2 (P2) → US3 (P3) → Polish。Foundational 完成后方可开始用户故事。 
- 用户故事独立性：US1/US2/US3 可在 Phase 2 之后并行，但建议先完成 US1 作为 MVP；US3 的聚合脚本/基准依赖 US1/US2 的实现输出。 
- 并行示例：Phase 1 的 T002/T003/T004/T005 可并行；Phase 2 的 T007/T008/T009/T011 可并行（T010 先定义接口）；US1 内 T012–T018 与 T035 可并行（不同目录，T036 依赖路由）；US2 内 T021–T024 可并行；US3 内 T027–T030 可并行。 

## Implementation Strategy（先 MVP，分步交付）

1) 完成 Setup+Foundational，确保 Make/CI/fixtures/bench 框架与回退/观测接口到位。  
2) 交付 MVP：完成 US1 契约+CLI/SDK 五场景与 PII/流式/回退路径，达到 ≥95% 通过率。  
3) 并行推进 US2（供应商适配/路由/metrics 接线）与 US3（覆盖率/安全/性能基准对比），及时在 `artifacts/coverage.txt`/`bench.txt` 写入差异与 owner。  
4) Polish：脱敏与错误提示审计，VitePress 文档同步，`make constitution-check` 全量回归并归档工件。 
