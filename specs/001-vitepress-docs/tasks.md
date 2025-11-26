---
description: "官方文档站（用户文档）实施任务清单"
---

# 任务清单：官方文档站（用户文档）

**输入**：`/Users/molei/codes/agno-Go/specs/001-vitepress-docs/` 中的设计文档  
**前置**：`plan.md`、`spec.md`、`research.md`、`data-model.md`、`contracts/`、`quickstart.md`

**测试**：本功能主要交付 VitePress 官方文档站与示例代码，不新增 Go 运行时代码；仍需确保现有 `make test`、`make providers-test`、`make coverage`、`make bench` 与 `make constitution-check` 全部通过。  

**覆盖率**：文档迭代不得降低现有 Go 代码的综合覆盖率（≥85%），相关命令和结果需在 artifacts 中留痕。  

**栈约束**：任务必须指向真实路径，覆盖 VitePress 文档工程（`/Users/molei/codes/agno-Go/docs/`）、Go 模块（`/Users/molei/codes/agno-Go/go/...`）、契约/治具（`/Users/molei/codes/agno-Go/specs/.../contracts/`）、脚本（`/Users/molei/codes/agno-Go/scripts/`）、自动化入口（`/Users/molei/codes/agno-Go/Makefile`）以及 CI 配置（`/Users/molei/codes/agno-Go/.github/workflows/ci.yml`）。禁止将 `/Users/molei/codes/agno-Go/agno` 下的 Python 代码作为运行时依赖。  

**文档与多语言**：涉及文档的任务必须指向 VitePress 工程中的具体路径（Markdown 文件或配置文件），说明影响的路由/侧边栏节点，并标明目标语言（en/zh/ja/ko）；如暂缺翻译，需在任务中显式标记，并规划补齐顺序。  

**组织方式**：任务按用户故事分组，确保每个故事都能独立实现与测试。

## 格式：`[ID] [P?] [Story] 描述`

- **[P]**：可并行执行（不同文件，且无依赖）
- **[Story]**：所属用户故事（如 US1、US2、US3）
- 描述中必须包含精确文件路径（绝对路径）

---

## Phase 1: Setup（共享基础设施）

**目的**：搭建 VitePress 文档工程骨架，并将文档构建接入现有自动化流程。

- [X] T001 在 `/Users/molei/codes/agno-Go/docs/` 初始化 VitePress 工程骨架（创建 `/Users/molei/codes/agno-Go/docs/package.json`、`/Users/molei/codes/agno-Go/docs/tsconfig.json`、`.gitignore` 与 `/Users/molei/codes/agno-Go/docs/.vitepress/` 目录）
- [X] T002 [P] 在 `/Users/molei/codes/agno-Go/docs/package.json` 中添加 VitePress 依赖与 scripts（如 `"docs:dev"`、`"docs:build"`、`"docs:preview"`），与仓库现有 Node 使用方式保持一致
- [X] T003 [P] 在 `/Users/molei/codes/agno-Go/docs/.vitepress/config.ts` 中创建基础站点配置（站点标题、base、默认语言），为后续 locales 与导航分组预留占位
- [X] T004 [P] 在 `/Users/molei/codes/agno-Go/Makefile` 中新增文档相关目标（例如 `docs-build`、`docs-serve`），内部调用 `/Users/molei/codes/agno-Go/docs/package.json` 中定义的 VitePress scripts
- [X] T005 在 `/Users/molei/codes/agno-Go/.github/workflows/ci.yml` 中插入文档构建步骤（调用 `make docs-build`/`make docs-check`），确保与现有 `fmt`、`lint`、`test`、`providers-test`、`coverage`、`bench`、`constitution-check` 一起执行

---

## Phase 2: Foundational（阻塞性前置）

**目的**：配置多语言站点结构、基础页面及路径约束检查，为所有用户故事解除公共阻塞。

- [X] T006 在 `/Users/molei/codes/agno-Go/docs/.vitepress/config.ts` 中配置 en/zh/ja/ko 四种语言的 `locales` 与顶层导航分组（如 Overview、Quickstart、Core Features & API、Provider Matrix、Advanced Guides、Contributing）
- [X] T007 [P] 在 `/Users/molei/codes/agno-Go/docs/index.md`、`/Users/molei/codes/agno-Go/docs/zh/index.md`、`/Users/molei/codes/agno-Go/docs/ja/index.md`、`/Users/molei/codes/agno-Go/docs/ko/index.md` 中创建首页占位内容，简要说明项目定位并提供指向各自 Quickstart 页面的链接
- [X] T008 [P] 在 `/Users/molei/codes/agno-Go/docs/guide/` 与 `/Users/molei/codes/agno-Go/docs/zh/guide/`、`/Users/molei/codes/agno-Go/docs/ja/guide/`、`/Users/molei/codes/agno-Go/docs/ko/guide/` 下创建 `quickstart.md`、`core-features-and-api.md`、`advanced/` 目录及空 Markdown 文件，保证 VitePress 初次构建不因缺少页面失败
- [X] T009 [P] 在 `/Users/molei/codes/agno-Go/scripts/check-docs-paths.sh` 编写 shell 脚本，扫描 `/Users/molei/codes/agno-Go/docs/` 下的 Markdown/配置文件中是否存在 `/Users/` 或 `C:\Users\` 等维护者本机绝对路径，发现即返回非零退出码
- [X] T010 在 `/Users/molei/codes/agno-Go/Makefile` 中新增 `docs-check` 目标（调用 `/Users/molei/codes/agno-Go/scripts/check-docs-paths.sh` 并执行 `docs-build`），并将 `docs-check` 串联到 `constitution-check`；同步更新 `/Users/molei/codes/agno-Go/.github/workflows/ci.yml` 在合适阶段运行 `make docs-check`

---

## Phase 3: 用户故事 1 - 新开发者 10 分钟完成第一个示例（优先级：P1）

**目标**：让第一次接触项目的开发者在仅依赖官方文档的前提下，在 10 分钟内完成一次从“启动服务”到“收到模型响应”的完整调用，并理解最小可用路径。  

**独立测试**：测试人员只访问文档站首页和 Quickstart 页面，不查看源码或历史文档，按文档指引完成环境准备、启动服务、创建 Agent 与会话并发送消息。

### 用户故事 1 的实现

- [X] T011 [P] [US1] 将 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/quickstart.md` 的结构与步骤整理为英文版 Quickstart 内容，填充到 `/Users/molei/codes/agno-Go/docs/guide/quickstart.md`，确保示例仅使用相对路径或占位符路径（如 `<your-project-root>/go/cmd/agno`、`./config/default.yaml`）
- [X] T012 [P] [US1] 基于 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/quickstart.md` 衍生中文 Quickstart 页面 `/Users/molei/codes/agno-Go/docs/zh/guide/quickstart.md`，保证步骤、请求/响应示例与英文版结构一致
- [X] T013 [P] [US1] 在 `/Users/molei/codes/agno-Go/docs/ja/guide/quickstart.md` 中编写日文 Quickstart 页面，保持与英文版在步骤数量、端点路径与代码示例上的等价
- [X] T014 [P] [US1] 在 `/Users/molei/codes/agno-Go/docs/ko/guide/quickstart.md` 中编写韩文 Quickstart 页面，保持与英文版在结构与示例上的等价
- [X] T015 [US1] 在 `/Users/molei/codes/agno-Go/docs/.vitepress/config.ts` 中为四种语言的导航与侧边栏加入 Quickstart 链接，确保从 `/Users/molei/codes/agno-Go/docs/index.md`、`/Users/molei/codes/agno-Go/docs/zh/index.md` 等首页至各自 quickstart 页面不超过一次点击
- [X] T016 [US1] 按 `/Users/molei/codes/agno-Go/docs/guide/quickstart.md` 中的步骤，在本地依次执行 `go run ./go/cmd/agno --config ./config/default.yaml` 与 `curl` 调用 `/health`、`/agents`、`/agents/{agentId}/sessions`、`/agents/{agentId}/sessions/{sessionId}/messages`，并在 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/artifacts/quickstart-notes.md` 中记录耗时与发现的问题

---

## Phase 4: 用户故事 2 - 现有用户系统性理解核心功能与 API（优先级：P2）

**目标**：帮助已经跑通 Quickstart 的用户系统性理解 Agent、Session、Message、Tool、Memory 与 Provider 的核心概念，以及与之对应的 HTTP API 和配置方式。  

**独立测试**：测试人员从一个具体需求出发（如“如何切换模型供应商”“如何使用会话与记忆”），仅通过文档中的导航与搜索，即可在 3 次点击内定位到相关页面并完成配置。

### 用户故事 2 的实现

- [X] T017 [P] [US2] 在 `/Users/molei/codes/agno-Go/docs/guide/core-features-and-api.md` 编写英文“Core Features & API Overview”，说明 Agent、Session、Message、Tool、Memory、Provider 概念，并引用 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/data-model.md` 与 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/contracts/docs-site-openapi.yaml` 中的主要端点和字段
- [X] T018 [P] [US2] 在 `/Users/molei/codes/agno-Go/docs/zh/guide/core-features-and-api.md` 编写对应中文页面，保持章节结构、示例与英文版完全一致，仅本地化文字说明与注释
- [X] T019 [P] [US2] 在 `/Users/molei/codes/agno-Go/docs/ja/guide/core-features-and-api.md` 编写日文页面，保持与英文版在内容结构与示例上的等价
- [X] T020 [P] [US2] 在 `/Users/molei/codes/agno-Go/docs/ko/guide/core-features-and-api.md` 编写韩文页面，保持与英文版在内容结构与示例上的等价
- [X] T021 [P] [US2] 在 `/Users/molei/codes/agno-Go/docs/guide/providers/matrix.md` 创建英文“Provider Capability Matrix”页面，从 `/Users/molei/codes/agno-Go/.env.example` 提取九家供应商的环境变量名称，并结合 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/data-model.md` 中的 ProviderCapabilityMatrix 描述支持的 chat/embedding/streaming 能力与备注
- [X] T022 [P] [US2] 在 `/Users/molei/codes/agno-Go/docs/zh/guide/providers/matrix.md`、`/Users/molei/codes/agno-Go/docs/ja/guide/providers/matrix.md`、`/Users/molei/codes/agno-Go/docs/ko/guide/providers/matrix.md` 创建对应的矩阵页面，保持表格结构与英文版相同，仅翻译标题与说明文字
- [X] T023 [US2] 在 `/Users/molei/codes/agno-Go/docs/.vitepress/config.ts` 中为各 locale 的导航与侧边栏添加 “Core Features & API” 与 “Provider Matrix” 入口，确保用户从首页开始不超过三次点击即可访问到这些页面
- [X] T024 [US2] 对比 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/contracts/docs-site-openapi.yaml` 与 `core-features-and-api` 页面中展示的端点与字段名称，在 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/artifacts/api-docs-checklist.md` 中记录检查结果与发现的差异（若有）

---

## Phase 5: 用户故事 3 - 高级用户基于高级案例构建复杂 Agent 流（优先级：P3）

**目标**：为有经验的用户提供多模型路由、知识库助手、持久记忆对话等高级案例，使其能够在 Agno-Go 之上构建生产级复杂工作流。  

**独立测试**：测试人员从高级案例文档中选择至少一个案例（如多模型路由），仅根据文档步骤完成配置和调用，并验证行为与文档描述相符。

### 用户故事 3 的实现

- [X] T025 [P] [US3] 在 `/Users/molei/codes/agno-Go/docs/guide/advanced/multi-provider-routing.md` 编写英文高级案例“多模型回退与路由”，展示如何基于 `/agents`、`/agents/{agentId}/sessions`、`/agents/{agentId}/sessions/{sessionId}/messages` 以及不同 provider 配置组合实现路由策略，所有代码示例仅使用相对路径和占位符 env 变量
- [X] T026 [P] [US3] 在 `/Users/molei/codes/agno-Go/docs/guide/advanced/knowledge-base-assistant.md` 编写英文高级案例“结合知识库的助手”，说明如何在 AgentOS 上集成外部向量检索或记忆存储，并引用 `/Users/molei/codes/agno-Go/go/tests/contract/` 与 `/Users/molei/codes/agno-Go/go/tests/memory/` 中相关测试的概念
- [X] T027 [P] [US3] 在 `/Users/molei/codes/agno-Go/docs/guide/advanced/memory-chat.md` 编写英文高级案例“带持久记忆的对话代理”，说明如何配置 MemoryStore 与 Session，使对话能够利用历史和用户元数据
- [X] T028 [P] [US3] 为上述三个高级案例在 `/Users/molei/codes/agno-Go/docs/zh/guide/advanced/`、`/Users/molei/codes/agno-Go/docs/ja/guide/advanced/`、`/Users/molei/codes/agno-Go/docs/ko/guide/advanced/` 下创建对应的 zh/ja/ko 版本 Markdown 文件（如 `multi-provider-routing.md` 等），保持步骤与代码示例完全等价，仅本地化文字与注释
- [X] T029 [US3] 在 `/Users/molei/codes/agno-Go/docs/.vitepress/config.ts` 中为各 locale 的导航添加 “Advanced Guides” 分组以及三个高级案例条目，确保 en/zh/ja/ko 中的结构和顺序保持一致
- [X] T030 [US3] 选择至少一个高级案例（推荐 `/Users/molei/codes/agno-Go/docs/guide/advanced/multi-provider-routing.md`），按文档步骤在本地运行示例，验证在正确配置 env 变量后工作流可正常运行，并在 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/artifacts/advanced-scenarios-notes.md` 记录验证过程

---

## Phase 6: Polish & Cross-Cutting Concerns

**目的**：收尾多语言一致性、路径安全检查、规格对齐与全局质量门。

- [X] T031 [P] 使用 `/Users/molei/codes/agno-Go/scripts/check-docs-paths.sh` 对 `/Users/molei/codes/agno-Go/docs/` 全量扫描，确认最终文档中不存在维护者本机绝对路径（如 `/Users/`、`C:\Users\` 等），如有则修正示例为相对路径或占位符路径
- [X] T032 [P] 在 `/Users/molei/codes/agno-Go/docs/.vitepress/config.ts` 中统一整理 en/zh/ja/ko 四种语言的导航顺序与命名，确保首页、Quickstart、Core Features & API、Provider Matrix、Advanced Guides、Contributing 等节点在各语言中一一对应
- [X] T033 在 `/Users/molei/codes/agno-Go/AGENTS.md` 中补充一段简短说明，指向官方文档站 `https://rexai.top/agno-Go/` 与 `/Users/molei/codes/agno-Go/docs/` 目录结构，帮助协作者快速找到文档源代码
- [X] T034 在 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/spec.md` 与 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/plan.md` 中回填实际实现结果（最终页面列表、已完成语言、本迭代交付范围），保证规格与实现保持同步
- [X] T035 [P] 对 `/Users/molei/codes/agno-Go/docs/` 下的 Markdown 文件执行统一格式化（标题层级、代码块语言标记），并通过 VitePress 本地预览或链接检查工具确认站内链接与外部链接可用
- [X] T038 [P] 在 `/Users/molei/codes/agno-Go/docs/guide/config-and-security.md` 编写英文 “Configuration & Security Practices” 页面，集中说明核心配置项、环境变量含义（参考 `/Users/molei/codes/agno-Go/.env.example` 与 `/Users/molei/codes/agno-Go/config/default.yaml`）、典型配置方式以及密钥管理与脱敏实践，确保仅使用相对路径或占位符路径示例
- [X] T039 [P] 在 `/Users/molei/codes/agno-Go/docs/zh/guide/config-and-security.md`、`/Users/molei/codes/agno-Go/docs/ja/guide/config-and-security.md`、`/Users/molei/codes/agno-Go/docs/ko/guide/config-and-security.md` 中创建对应的配置与安全实践页面，保持章节结构与示例与英文版等价，仅本地化文字说明与注释
- [X] T040 在 `/Users/molei/codes/agno-Go/docs/.vitepress/config.ts` 中为 en/zh/ja/ko 四种语言的导航与侧边栏添加 “Configuration & Security”/“配置与安全实践” 等入口，并从 Quickstart 与 Provider Matrix 页面添加链接指向该章节
- [X] T041 [P] 在 `/Users/molei/codes/agno-Go/docs/guide/contributing-and-quality.md` 编写英文 “Contributing & Quality Gates” 页面，说明如何为项目贡献代码、需要执行的 `make` 目标（fmt/lint/test/providers-test/coverage/bench/constitution-check/docs-build/docs-check）及其在 CI 中的作用
- [X] T042 [P] 在 `/Users/molei/codes/agno-Go/docs/zh/guide/contributing-and-quality.md`、`/Users/molei/codes/agno-Go/docs/ja/guide/contributing-and-quality.md`、`/Users/molei/codes/agno-Go/docs/ko/guide/contributing-and-quality.md` 中创建对应的贡献与质量保障页面，保持结构与关键内容与英文版等价，仅本地化文字与示例说明
- [X] T043 在 `/Users/molei/codes/agno-Go/docs/.vitepress/config.ts` 中为各 locale 的导航添加 “Contributing” 分组或入口，将其指向对应的贡献与质量保障页面，并在 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/artifacts/docs-build-report.md` 中新增一节简要描述发布前文档/契约同步与支持指标采集的人工检查步骤

---

## 覆盖率 Gate（所有故事完成后执行）

- [X] T036 运行 `make test`、`make providers-test`、`make coverage`、`make bench` 与 `make constitution-check`，确保本次文档改动未破坏现有 Go 代码质量门槛，并在 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/artifacts/coverage-and-bench.md` 中记录覆盖率与基准结果摘要
- [X] T037 在 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/artifacts/docs-build-report.md` 中记录最近一次 `make docs-build` 与 `make docs-check` 的输出摘要、构建时间、发现的问题，并预留或更新一节用于记录与 SC-001/SC-002 相关的可用性测试结果与支持咨询指标，为后续优化与目标对比提供基线
- [ ] T044 组织一次包含至少 10 名此前从未接触过本项目的新开发者的 Quickstart 可用性测试：根据 `/Users/molei/codes/agno-Go/docs/guide/quickstart.md` 及其多语言等价页面（`/Users/molei/codes/agno-Go/docs/zh/guide/quickstart.md`、`/Users/molei/codes/agno-Go/docs/ja/guide/quickstart.md`、`/Users/molei/codes/agno-Go/docs/ko/guide/quickstart.md`）指导参与者在仅访问文档站的前提下完成端到端流程，并在 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/artifacts/quickstart-notes.md` 与 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/artifacts/docs-build-report.md` 中记录每位参与者的起止时间、是否在 10 分钟内完成、遇到的问题及改进建议
- [ ] T045 在每次发布前根据文档中标记为“核心功能”与“高级案例”的示例清单，逐一重跑这些示例（参考 `/Users/molei/codes/agno-Go/docs/guide/core-features-and-api.md`、`/Users/molei/codes/agno-Go/docs/guide/advanced/multi-provider-routing.md`、`/Users/molei/codes/agno-Go/docs/guide/advanced/knowledge-base-assistant.md`、`/Users/molei/codes/agno-Go/docs/guide/advanced/memory-chat.md` 及其多语言版本），并在 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/artifacts/docs-build-report.md` 或 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/artifacts/example-verification-log.md` 中记录运行结果与发现的问题，用于验证 SC-003 要求的“核心功能与高级案例示例在每次发版前均通过内部验证流程”

---

## 依赖与执行顺序

### 阶段依赖

- **Setup（Phase 1）**：无依赖，可立即开始，用于搭建 VitePress 工程与 Make/CI 接入。
- **Foundational（Phase 2）**：依赖 Setup，提供多语言结构、基础页面与路径检查，是所有用户故事的公共前置。
- **用户故事（Phase 3–5）**：均依赖 Foundational 完成；若人手充足，可在 US1/US2/US3 之间并行推进，否则建议按优先级顺序（US1→US2→US3）串行。
- **Polish & Coverage Gate（Phase 6）**：依赖所有目标用户故事完成，用于最终一致性校验与质量门。

### 用户故事依赖

- **US1 (P1)**：Foundational 完成后即可开始，实现 Quickstart 与基础体验，是整个文档站的 MVP。  
- **US2 (P2)**：依赖 US1 已经提供最小闭环体验，但在内容上与 US1 相对独立，可在 Foundational 后并行推进。  
- **US3 (P3)**：依赖 US1 提供基础调用路径，依赖 US2 提供概念与 API 说明；在实践中可在 US2 接近完成时并行推进。  

### 单个故事内的顺序

- US1：先完成各语言 quickstart 页面（T011–T014），再接入导航（T015），最后用 T016 进行端到端验证。  
- US2：先完成 Core Features & API 页面（T017–T020），再完成 Provider Matrix 页面与导航（T021–T023），最后执行契约对齐检查（T024）。  
- US3：先完成英文高级案例（T025–T027），再补齐多语言版本与导航（T028–T029），最后执行至少一个案例的端到端验证（T030）。  

### 可并行机会

- Setup 与 Foundational 中标记为 [P] 的任务（例如 T002、T003、T007–T009、T031、T035）可由不同协作者并行完成，只要注意避免同一文件的冲突。  
- Foundational 完成后，US1/US2/US3 中所有 [P] 任务（如 T011–T014、T017–T022、T025–T028）可按语言或页面维度并行拆分。  
- Polish 阶段中的文档格式化（T035）与路径扫描（T031）可以与规格回填（T034）并行进行。  

---

## 并行示例：用户故事 1

```bash
# 并行创建多语言 quickstart 页面（不同文件，无直接依赖）：
Task: "T011 [P] [US1] 填充 /Users/molei/codes/agno-Go/docs/guide/quickstart.md 英文内容"
Task: "T012 [P] [US1] 填充 /Users/molei/codes/agno-Go/docs/zh/guide/quickstart.md 中文内容"
Task: "T013 [P] [US1] 填充 /Users/molei/codes/agno-Go/docs/ja/guide/quickstart.md 日文内容"
Task: "T014 [P] [US1] 填充 /Users/molei/codes/agno-Go/docs/ko/guide/quickstart.md 韩文内容"

# 在 quickstart 内容稳定后，再由一人集中完成导航与端到端验证：
Task: "T015 [US1] 更新 /Users/molei/codes/agno-Go/docs/.vitepress/config.ts 中的 Quickstart 导航"
Task: "T016 [US1] 按 quickstart 文档执行端到端调用并记录验证结果"
```

---

## 实施策略

### MVP 优先（仅交付用户故事 1）

1. 完成 Phase 1（Setup）与 Phase 2（Foundational），具备可构建的多语言文档站骨架。  
2. 完成 Phase 3（US1），确保 Quickstart 在四种语言下可用，并通过一次端到端验证（T016）。  
3. 在此基础上即可对外宣传官方文档站的 MVP 版本。  

### 渐进式交付

1. Setup + Foundational → 完成文档基础设施与多语言结构。  
2. US1 → 提供完整的“10 分钟上手”路径。  
3. US2 → 补齐核心概念、API 概览与供应商矩阵。  
4. US3 → 补齐高级案例，提升生产级场景说服力。  
5. Polish & Coverage Gate → 统一风格、完成路径与质量门校验。  

### 并行团队策略

1. 由一名协作者主导 Phase 1–2（VitePress 工程与多语言骨架）。  
2. Foundational 完成后：
   - 开发者 A 专注 US1（Quickstart 与端到端验证）。  
   - 开发者 B 专注 US2（核心功能与供应商矩阵）。  
   - 开发者 C 专注 US3（高级案例与实践验证）。  
3. 所有故事完成后，由任意一名协作者执行 Phase 6（Polish & Coverage Gate），确保最终上线前的统一质量门。  
