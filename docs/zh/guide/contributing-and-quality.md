## 贡献与质量保障

本页说明如何为 Agno-Go 项目贡献代码或文档，以及在合并变更前需要通过的质量检查（quality gates）。

### 1. 从哪里开始

- **阅读仓库根目录的 `AGENTS.md`**，了解：
  - 整体目录结构（`go/`、`docs/`、`specs/`、`scripts/` 等）。  
  - 运行时约束（纯 Go，不允许运行时 Python/cgo 桥接）。  
  - 规格与治具如何约束行为。  
- **在实现前先查看对应规格**：
  - AgentOS 契约与治具：`specs/001-go-agno-rewrite/`。  
  - 官方文档站规划：`specs/001-vitepress-docs/`。  

对于非 trivial 的改动，建议先更新对应的 spec/plan/tasks，再修改运行时代码或文档。

### 2. 核心 `make` 目标

所有质量检查都通过仓库根目录的 `Makefile` 暴露。常用目标包括：

- `make fmt`：使用 `gofumpt` 格式化 Go 代码。  
- `make lint`：运行 `golangci-lint` 静态检查。  
- `make test`：运行 Go 单元/包测试（`go test ./...`）。  
- `make providers-test`：运行供应商集成测试（受环境变量控制，可跳过）。  
- `make coverage`：生成覆盖率报告与汇总。  
- `make bench`：运行基准测试并通过 `benchstat` 汇总结果。  
- `make constitution-check`：执行完整质量 Gate：fmt/lint/test/providers-test/coverage/bench 以及“禁止 cgo/Python 子进程”审计。  
- `make docs-build`：为 `docs/` 安装依赖并构建 VitePress 文档站。  
- `make docs-check`：对 `docs/` 运行路径安全检查（禁止任何维护者本机的绝对路径，如本地用户目录），然后执行一次完整文档构建。  

在提交 PR 之前，至少建议运行：

```bash
make fmt lint test docs-check
```

若本次改动涉及供应商行为、契约或性能敏感路径，建议额外运行：

```bash
make providers-test coverage bench constitution-check
```

### 3. Go 代码贡献要求

- **风格与布局**
  - 使用 `gofumpt` 统一格式（通过 `make fmt`）。  
  - 包结构尽量对齐现有的 `go/internal` 与 `go/pkg` 目录层次。  
- **测试**
  - 每个包都应有 `_test.go` 文件。  
  - 影响行为的改动应通过单元测试覆盖；若涉及对外接口或供应商行为，应在 `go/tests/contract` 或 `go/tests/providers` 中补充契约/集成测试。  
- **禁止运行时桥接**
  - Go 运行时不得通过子进程调用 Python，也不得依赖 cgo 桥接。  
  - `make constitution-check` 中的审计会检查这些约束。  

### 4. 文档与规格要求

- **规格是单一事实来源**
  - 新功能或行为变更应首先更新 `specs/` 下的规格，并在需要时重新生成任务清单。  
  - 以规格驱动 Go 实现与 VitePress 文档，而不是“先写代码再补文档”。  

- **文档对齐**
  - 保持 `docs/` 下的 VitePress 文档与以下内容一致：  
    - HTTP 契约：`specs/001-go-agno-rewrite/contracts/`。  
    - 供应商治具与行为：`specs/001-go-agno-rewrite/contracts/fixtures/`。  
  - 文档示例中避免使用维护者本机绝对路径，采用相对路径（如 `./config/default.yaml`）或通用占位符。  

- **多语言一致性**
  - 对于核心页面（概览、Quickstart/快速开始、Core Features & API、Provider Matrix、Advanced Guides、Configuration & Security、Contributing & Quality），需要确保：  
    - en/zh/ja/ko 均有对应页面；  
    - 代码示例在语义和行为上等价，仅文字说明与注释做本地化。  

### 5. PR 中建议包含的内容

在提交 PR 时，建议说明：

- 变更的简要描述以及对应的 spec/task 编号。  
- 本地执行过的 `make` 目标（可粘贴关键命令输出，方便 reviewer 理解）。  
- 若改动主要涉及文档：  
  - 可以附上截图或文字概述说明新/更新页面；  
  - 若有计划在后续补齐的部分（例如翻译），请在说明中标注。  

遵循以上实践，可以帮助团队保持 Go 运行时、契约与 VitePress 文档的一致性，同时确保新贡献符合项目约定的质量门槛。
