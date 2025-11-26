# 实现计划：[FEATURE]

**分支**：`[###-feature-name]` | **日期**： [DATE] | **规格**： [链接]
**输入**：来自 `/specs/[###-feature-name]/spec.md` 的功能规格

**说明**：该模板由 `/speckit.plan` 命令填充。

## 摘要

[摘自功能规格：关键需求 + 调研后的技术路径]

## 技术背景

<!--
  必须：将此处内容替换为该项目的技术细节。
  结构仅为建议，可根据迭代需要调整。
-->

**语言/版本**： [例如 Python 3.11、Swift 5.9、Rust 1.75 或 NEEDS CLARIFICATION]  
**主要依赖**： [例如 FastAPI、UIKit、LLVM 或 NEEDS CLARIFICATION]  
**存储**： [如适用，例如 PostgreSQL、CoreData、文件，或 N/A]  
**测试**： [例如 pytest、XCTest、cargo test 或 NEEDS CLARIFICATION]  
**目标平台**： [例如 Linux server、iOS 15+、WASM 或 NEEDS CLARIFICATION]  
**项目类型**： [single/web/mobile —— 决定源码结构]  
**性能目标**： [领域相关，例如 1000 req/s、10k lines/sec、60 fps 或 NEEDS CLARIFICATION]  
**约束条件**： [如 <200ms p95、<100MB 内存、离线可用，或 NEEDS CLARIFICATION]  
**规模/范围**： [例如 1 万用户、100 万 LOC、50 个页面，或 NEEDS CLARIFICATION]

## 宪章检查

*Gate：Phase 0 调研前必须通过；Phase 1 设计后需再次检查。*

- [ ] **纯 Go / 禁止桥接**：说明要迁移的 Python 能力、对应的 Go 模块路径（如 `go/internal/...`），确认不会通过 cgo/FFI/子进程调用 `./agno` 代码。
- [ ] **模型供应商矩阵（ollm、Gemini、OpenAI、GLM4、OpenRouter、SiliconFlow、Cerebras、ModelScope、Groq）**：列出本迭代涉及的供应商、能力（chat/embedding/流式）、所需 env 变量和差异点。
- [ ] **契约/治具与基准**：规划使用的 Python 参考输出、治具位置（`specs/.../contracts/fixtures`）、golden/契约测试与性能基准方案，确保运行时不依赖 Python。
- [ ] **自动化与 Make**：需要新增/调整的 make 目标（fmt/lint/test/providers-test/coverage/bench/gen-fixtures/release），以及 CI 复用方式。
- [ ] **测试纪律 + 85% 覆盖率**：列出需要的 Go 单元、契约、供应商集成测试与覆盖率策略，说明缺口与补救方案。
- [ ] **密钥与安全**：确认 `.env.example` 与 secret 注入方式，避免提交真实 key；若需共享基准数据，说明脱敏措施。
- [ ] **VitePress 官方文档与多语言**：说明本迭代影响的 VitePress 文档页面/导航结构、对应 `specs/[###-feature-name]/quickstart.md` 段落，以及 en/zh/ja/ko 四种语言的翻译与对齐计划。

## 项目结构

### 文档（当前功能）

```text
specs/[###-feature]/
├── plan.md              # 本文件（/speckit.plan 输出）
├── research.md          # Phase 0 输出（/speckit.plan）
├── data-model.md        # Phase 1 输出（/speckit.plan）
├── quickstart.md        # Phase 1 输出（/speckit.plan）
├── contracts/           # Phase 1 输出（/speckit.plan）
└── tasks.md             # Phase 2 输出（/speckit.tasks，非 /speckit.plan 创建）
```

### 源码（仓库根目录）
<!--
  必须：将占位树替换为真实结构，删除未用选项，补上真实路径（如 apps/admin、packages/foo）。交付的计划中不得保留 Option 标签。
-->

```text
agno/                         # Python 参考实现（只读，不可被 Go 运行时调用）
go/
├── cmd/agno/                 # Go 入口（CLI/服务）
├── internal/
│   ├── agent/                # Agent/Workflow/Step Engine
│   ├── runtime/              # 服务编排、协议层
│   ├── model/                # 模型接口定义与路由
│   ├── memory/               # 状态/存储接口
│   └── tool/                 # 工具/MCP/拦截器
├── pkg/
│   ├── providers/<provider>/ # 模型供应商适配器
│   ├── memory/               # 具体存储实现
│   └── tools/                # 额外可插拔组件
└── tests/
    ├── contract/             # 契约/golden
    ├── providers/            # 供应商集成
    └── bench/                # 基准

specs/[###-feature]/
├── plan.md | research.md | data-model.md | quickstart.md
├── contracts/fixtures/      # Python 治具（脱敏）
├── contracts/deviations.md  # 与 Python 差异
└── artifacts/               # 覆盖率/基准/报告

scripts/                     # Go/标准工具脚本（如治具生成）
.env.example                 # 供应商 env 占位
Makefile                     # 单一入口（fmt/lint/test/providers-test/coverage/bench/release）
```

**结构决策**： [记录所选结构，并引用上方列出的真实目录]

## 复杂度追踪

> **仅当宪章检查存在违规且必须说明时填写**

| 违规项 | 必要原因 | 更简单方案被拒绝的理由 |
|--------|----------|--------------------------|
| [例：第 4 个项目] | [当前需求] | [为何 3 个项目不够] |
| [例：Repository 模式] | [具体问题] | [为何直接 DB 访问不够] |
