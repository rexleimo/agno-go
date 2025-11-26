# 实现计划：官方文档站（用户文档）

**分支**：`001-vitepress-docs` | **日期**：2025-11-22 | **规格**：`/Users/molei/codes/agno-Go/specs/001-vitepress-docs/spec.md`  
**输入**：来自 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/spec.md` 的功能规格

**说明**：该计划由 `/speckit.plan` 流程填充，用于指导官方 VitePress 文档站与 Go AgentOS 行为/契约的对齐。

## 摘要

本功能旨在为 Go 版 AgentOS 提供一套结构清晰、易于上手且支持中/英/日/韩四种语言的官方文档站，提升项目对外影响力与采纳率。  
规格强调三类核心场景：新开发者 10 分钟内跑通第一个示例、现有用户系统性理解核心功能与“API 级”能力，以及高级用户可基于官方高级案例构建复杂 Agent 流程。  
本计划将：
- 在 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/` 下定义文档数据模型、页面与示例的“契约”，并产出 quickstart、大纲与导航约束；
- 规划 VitePress 文档工程在仓库中的位置与结构（`/Users/molei/codes/agno-Go/docs/`），确保其与 Go 实现（`/Users/molei/codes/agno-Go/go/...`）及契约/fixtures 一致；
- 约束所有对外示例代码仅使用相对路径或占位符，避免出现维护者本机的绝对路径，同时保持与 Python 参考实现/fixtures 的行为对齐；
- 将文档更新与多语言对齐纳入现有宪章 Gate（尤其是 `make constitution-check` 与 VitePress 文档更新要求），并在 tasks 中以可执行任务形式落地。

## 技术背景

<!--
  必须：将此处内容替换为该项目的技术细节。
  结构仅为建议，可根据迭代需要调整。
-->

**语言/版本**：Go 1.25.1（唯一运行时）+ Node 18+ / VitePress 1.x（官方文档站）；Python 3.11 仅用于离线治具生成，与本迭代无运行时耦合。  
**主要依赖**：标准库 `net/http` + `github.com/go-chi/chi/v5`（AgentOS HTTP 入口）、`golangci-lint`、`gofumpt`、`benchstat`（通过 Makefile 驱动），以及 VitePress（基于 Vite 的静态文档站，用于承载用户文档与代码示例）。  
**存储**：文档源文件与构建产物存储在 Git 仓库与静态站点托管（无新增线上数据库）；AgentOS 持久化依托现有 MemoryStore/Bolt/Badger 实现，本迭代不扩展或更改存储形态。  
**测试**：Go 侧沿用 `make fmt lint test providers-test coverage bench constitution-check`；文档侧需保证 VitePress 构建（如 `npm run docs:build`）纳入 CI，并引入基础的链接/结构检查和路径安全检查（如 `docs-check`），防止页面缺失、导航断链或示例路径泄露维护者本机信息。  
**目标平台**：AgentOS 运行在 Linux/macOS 服务器与开发机；文档站在 CI/CD 中构建为静态资源后部署到 `https://rexai.top/agno-Go/`，面向桌面与移动浏览器访问。  
**项目类型**：服务端 CLI + HTTP API（Go AgentOS）+ 静态 Web 文档站（VitePress）。  
**性能目标**：不改变现有 AgentOS 性能门槛（契约与基准仍由 `specs/001-go-agno-rewrite/` 约束）；文档站需在典型网络环境下保证常见页面首屏内容约 2 秒内可见，并在 CI 中保持构建时间在数分钟级别（通过构建报告采集观测数据，而非强制 Gate）。  
**约束条件**：运行时仅依赖 Go 1.25.1；构建链不得将 Python 引入为必需依赖；对外文档中禁止出现维护者本机的绝对文件系统路径，所有示例仅使用相对路径或通用占位符；文档与 Go 契约/fixtures 行为冲突时，以 Go 实现和契约为准。  
**规模/范围**：本迭代聚焦 1 套官方文档站的结构与内容规划，覆盖 4 种语言（en/zh/ja/ko）、至少 1 条完整快速开始路径、1 章核心功能与 API 导航以及 3+ 篇高级案例；不对 AgentOS 内核模块进行大规模重构。

## 宪章检查

*Gate：Phase 0 调研前必须通过；Phase 1 设计后需再次检查。*

- [x] **纯 Go / 禁止桥接**：本迭代仅新增/更新 VitePress 文档内容，不引入新的运行时代码或跨语言桥接；所有示例明确声明运行时为 Go 1.25.1，禁止示例中通过子进程调用 `/Users/molei/codes/agno-Go/agno` 下的 Python 代码，Go 接口仍以 `/Users/molei/codes/agno-Go/go/internal/{agent,runtime,model,memory,tool}/` 与 `/Users/molei/codes/agno-Go/go/pkg/` 为唯一事实来源。  
- [x] **模型供应商矩阵（ollm、Gemini、OpenAI、GLM4、OpenRouter、SiliconFlow、Cerebras、ModelScope、Groq）**：文档将新增“模型供应商与能力矩阵”页面，基于现有 Go 适配器 `/Users/molei/codes/agno-Go/go/pkg/providers/` 与契约测试行为，列出九家供应商支持的 Chat/Embedding/流式能力、主要差异点及对应 env 变量名称（来自 `/Users/molei/codes/agno-Go/.env.example`），不在本迭代内新增或移除供应商。  
- [x] **契约/治具与基准**：本计划不会新增运行时代码契约，但会在 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/contracts/` 下记录文档结构与页面/示例的“文档契约”，并通过 `docs-site-openapi.yaml` 反映现有 HTTP surface；文档更新需遵循 FR-009：当 `/Users/molei/codes/agno-Go/specs/001-go-agno-rewrite/contracts/` 中的契约或治具发生变更时，必须在发布前检查并同步更新 VitePress 文档，检查结果记录在 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/artifacts/api-docs-checklist.md` 或等价文件。  
- [x] **自动化与 Make**：沿用根目录 `/Users/molei/codes/agno-Go/Makefile` 作为单一入口，Go 侧继续使用现有 `fmt/lint/test/providers-test/coverage/bench/gen-fixtures/release/constitution-check` 目标；文档侧将新增 `docs-build`、`docs-serve`、`docs-check` 等目标封装 VitePress 命令与路径安全检查，并在 CI（`/Users/molei/codes/agno-Go/.github/workflows/ci.yml`）中通过 `make docs-build` 与 `make docs-check` 执行。  
- [x] **测试纪律 + 85% 覆盖率**：本迭代不调整 Go 测试结构，但需要在文档中显式说明当前测试纪律（包括 `go/tests/{contract,providers,bench}/` 与覆盖率门槛），并通过 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/contracts/` 中的文档契约约定：文档示例不得鼓励绕过现有测试/契约路径；`make test`、`make providers-test`、`make coverage` 仍是发布 Gate 的一部分。  
- [x] **密钥与安全**：文档将统一使用占位符 env 变量（例如 `OPENAI_API_KEY`、`GEMINI_API_KEY` 等），并在 Quickstart、Provider Matrix 及专门的“配置与安全实践”章节中强调不得提交真实 key；在 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/quickstart.md` 与 VitePress 文档中同步说明 `/Users/molei/codes/agno-Go/.env.example` 的作用和 secret 注入方式，对任何演示基准数据说明脱敏策略。  
- [x] **VitePress 官方文档与多语言**：本迭代聚焦官方 VitePress 文档站的结构与内容规划，明确：文档源放置于 `/Users/molei/codes/agno-Go/docs/`（VitePress 工程），并以 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/quickstart.md` 为单一事实来源同步到 en/zh/ja/ko 四种语言页面；计划中通过 data-model 与 tasks 定义页面树与导航，对各语言间的对齐策略和翻译落地路径做出约定，并在发布 Gate 中要求至少上述四种语言的核心页面保持等价。

## 项目结构

### 文档（当前功能）

```text
/Users/molei/codes/agno-Go/specs/001-vitepress-docs/
├── plan.md              # 本文件（/speckit.plan 输出）
├── research.md          # Phase 0 输出（调研与决策）
├── data-model.md        # Phase 1 输出（文档域数据模型）
├── quickstart.md        # Phase 1 输出（对外 Quickstart 与示例）
├── contracts/           # Phase 1 输出（文档结构与示例契约）
└── tasks.md             # Phase 2 输出（/speckit.tasks 创建）
```

### 源码（仓库根目录）
<!--
  必须：将占位树替换为真实结构，删除未用选项，补上真实路径（如 apps/admin、packages/foo）。交付的计划中不得保留 Option 标签。
-->

```text
/Users/molei/codes/agno-Go/agno/                  # Python 参考实现（只读，不可被 Go 运行时调用）
/Users/molei/codes/agno-Go/go/
├── cmd/agno/                                 # Go 入口（CLI/服务）
├── internal/
│   ├── agent/                                # Agent/Workflow/Step Engine
│   ├── runtime/                              # 服务编排、协议层（含 HTTP Server 与中间件）
│   ├── model/                                # 模型接口定义与路由
│   ├── memory/                               # 状态/存储接口
│   └── tool/                                 # 工具/MCP/拦截器
├── pkg/
│   ├── providers/                            # 模型供应商适配器（九家供应商）
│   ├── memory/                               # 具体存储实现
│   └── tools/                                # 额外可插拔组件
└── tests/
    ├── contract/                             # 契约/golden 测试（消费 specs/001-go-agno-rewrite/contracts/fixtures）
    ├── providers/                            # 供应商集成测试
    └── bench/                                # 性能基准

/Users/molei/codes/agno-Go/specs/001-go-agno-rewrite/  # 核心 AgentOS 契约与治具（在其他迭代中维护）
├── contracts/fixtures/                        # Python 治具（脱敏）
├── contracts/deviations.md                    # 与 Python 行为差异
└── artifacts/                                 # 覆盖率/基准/报告

/Users/molei/codes/agno-Go/specs/001-vitepress-docs/   # 本迭代文档规划与契约
├── plan.md | research.md | data-model.md | quickstart.md | tasks.md
└── contracts/                                 # 文档结构与示例契约（本迭代新增）

/Users/molei/codes/agno-Go/scripts/           # Go/标准工具脚本（如治具生成、文档检查脚本）
/Users/molei/codes/agno-Go/config/default.yaml# AgentOS 默认配置（文档中会引用）
/Users/molei/codes/agno-Go/.env.example       # 供应商 env 占位与说明
/Users/molei/codes/agno-Go/Makefile           # 单一入口（fmt/lint/test/providers-test/coverage/bench/release/constitution-check/docs-build/docs-check）
/Users/molei/codes/agno-Go/docs/              # VitePress 文档工程（本迭代新增）
/Users/molei/codes/agno-Go/AGENTS.md          # Agent 协作者说明与文档站入口（由 update-agent-context 维护）
```

**结构决策**：  
- 文档与规划文件统一位于 `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/`，作为 VitePress 工程与 Go 行为文档化的单一事实来源。  
- 现有 Go 源码与测试结构保持不变，仅在文档中引用 `/Users/molei/codes/agno-Go/go/...` 与 `/Users/molei/codes/agno-Go/config/default.yaml` 作为示例来源，而不在对外文档中暴露这些绝对路径。  
- VitePress 文档工程放置于 `/Users/molei/codes/agno-Go/docs/`，通过 Makefile 封装构建/预览/检查命令，并从 `specs/001-vitepress-docs/quickstart.md`、`data-model.md` 与 `contracts/` 中同步导航与示例。  
- AGENTS 指南（`/Users/molei/codes/agno-Go/AGENTS.md`）在本迭代中扩展说明文档站入口与目录结构，便于 AI 代理与协作者理解整体布局。

## 实施结果摘要（2025-11-22）

截至当前环境，本计划已落实以下关键交付：

- **VitePress 工程与多语言骨架**  
  - `docs/` 下已创建 VitePress 工程，`docs/.vitepress/config.ts` 配置了 en/zh/ja/ko 四种语言的 `locales`、顶层导航与侧边栏。  
  - 首页与 Quickstart 页面在四种语言中均已存在，并与 `specs/001-vitepress-docs/quickstart.md` 中的对外示例保持一致。  

- **核心页面树与导航对齐**  
  - Core Features & API、Provider Matrix、三篇高级案例（多模型路由、知识库助手、持久记忆对话）、Configuration & Security、Contributing & Quality Gates 均已在四种语言下落地。  
  - 导航中统一提供 Overview / Quickstart / Core Features & API / Provider Matrix / Advanced Guides / Configuration & Security / Contributing & Quality 的入口，并确保各语言的结构与顺序对应。  

- **配置与安全 / 贡献与质量章节**  
  - 在 `docs/guide/config-and-security.md` 及其多语言版本中，系统梳理了 `.env.example` 与 `config/default.yaml` 中的关键配置项、环境变量和安全实践。  
  - 在 `docs/guide/contributing-and-quality.md` 及其多语言版本中，集中说明了项目的贡献流程与质量 Gate，涵盖 `make fmt/lint/test/providers-test/coverage/bench/constitution-check/docs-build/docs-check` 等命令的作用。  

- **构建与路径安全检查**  
  - `scripts/check-docs-paths.sh` 已完成并接入 `make docs-check`，可在 CI 中防止 `/Users/`、`C:\Users\` 等维护者绝对路径泄露到用户文档。  
  - 在当前环境中，`npm --prefix ./docs run docs:build` 能成功构建站点；相关输出与人工检查步骤记录在 `specs/001-vitepress-docs/artifacts/docs-build-report.md`。  

- **高级案例与验证笔记**  
  - 高级案例文档已按照数据模型与 contracts 完成。由于 `go/cmd/agno` 入口尚未实现，基于 Go 运行时的端到端高级案例验证（T030）目前记录为被运行时缺失阻塞，细节见 `artifacts/advanced-scenarios-notes.md`。  

后续迭代需要围绕 Phase 6 的剩余任务和“覆盖率 Gate”部分（T030、T036–T037、T044–T045 等）补齐端到端示例验证、覆盖率/基准报告与可用性测试，并在完成后更新本节内容。

## 复杂度追踪

> **仅当宪章检查存在违规且必须说明时填写**

当前计划不预期引入违反宪章的设计复杂度；如后续在实现阶段发现必须偏离宪章（例如引入额外语言运行时或打破 Makefile 单一入口），需要在此表中记录原因并提交单独的宪章修订提案。

| 违规项 | 必要原因 | 更简单方案被拒绝的理由 |
|--------|----------|--------------------------|
| （留空） | （留空） | （留空） |
