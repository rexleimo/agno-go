# Research: 官方文档站（用户文档）

**Feature**: `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/spec.md`  
**Branch**: `001-vitepress-docs`  
**Date**: 2025-11-22

本文件汇总与本功能相关的技术选型与约束决策，用于支撑 plan.md 中的技术背景与宪章检查。当前规格未包含形式化的 `[NEEDS CLARIFICATION]` 标记，因此研究聚焦于关键依赖与集成模式的 best practices。

---

## 1. VitePress 文档工程位置与结构

- **Decision**:  
  将官方文档站的 VitePress 工程放置在仓库根目录下的 `/Users/molei/codes/agno-Go/docs/` 目录中，使用 VitePress 默认的 `docs/.vitepress` 结构（配置、主题、导航），并通过 Makefile 封装 `docs` 相关构建与预览命令。

- **Rationale**:  
  - 与社区常见实践一致（文档与源码共仓库，根目录 `docs/` 作为默认入口），便于在 PR 中同步审查代码与文档。  
  - 与宪章中“文档源必须纳入版本控制并与 specs/quickstart 契约对齐”的要求匹配，便于从 `specs/001-vitepress-docs/quickstart.md` 等文件向 VitePress 导出内容。  
  - CI 集成更简单：现有仓库已经有统一的 Makefile 和 GitHub Actions 工作流，增加 `make docs-build` 等目标即可接入现有流水线。

- **Alternatives considered**:  
  - **独立文档仓库**（例如单独的 `agno-Go-docs` 仓库）：在版本管理和部署上更独立，但会引入跨仓库同步成本，违背“以 specs 为单一事实来源”的约束。  
  - **将文档放在 `go/` 或 `config/` 子目录**：会混淆运行时代码与文档的边界，不利于长期维护。  
  - **使用其他静态站点生成器（如 Docusaurus、MkDocs）**：与宪章“必须使用 VitePress 官方文档站”的约束不符，弃用。

---

## 2. 多语言文档结构与页面映射

- **Decision**:  
  文档站采用统一的页面树结构，并在 VitePress 中为 en/zh/ja/ko 四种语言分别配置 locale 根路径（例如 `/`, `/zh/`, `/ja/`, `/ko/`），确保以下页面在四种语言中一一对应：
  - 首页（项目定位与价值）  
  - 快速开始（与 `specs/001-vitepress-docs/quickstart.md` 对齐）  
  - 核心功能与 API 导航  
  - 模型供应商与能力矩阵  
  - 高级案例（至少三篇）  
  - 贡献与质量保障

- **Rationale**:  
  - 一致的页面树有利于社区讨论和外部链接分享，减少“某语言缺页”的风险。  
  - 通过 locale 配置，VitePress 原生支持多语言导航与标题翻译，无需自行实现路由逻辑。  
  - 将结构定义在 specs（data-model.md / quickstart.md）中，便于后续自动化或脚本生成导航配置。

- **Alternatives considered**:  
  - **不同语言使用不同的页面树**：可能更符合部分地区习惯，但会增加维护成本，且与宪章“结构与示例需等价”的要求冲突。  
  - **仅提供英文完整文档，其他语言只翻译部分章节**：短期内工作量较小，但不满足宪章对四种语言对齐的要求，可作为临时状态但需在 tasks 中列出补齐任务。  

---

## 3. 示例代码与路径表达约束

- **Decision**:  
  所有对外文档与示例代码中禁止使用维护者本机的绝对文件系统路径（如 `/Users/...`），统一使用以下形式：
  - 相对仓库根目录的路径，例如 `go/cmd/agno`、`config/default.yaml`；  
  - 相对运行目录的路径，例如 `./go/cmd/agno`、`./config/default.yaml`；  
  - 针对用户环境的占位符路径，例如 `<your-project-root>/go/cmd/agno`，并在文案中强调需要替换。  

- **Rationale**:  
  - 避免读者误以为必须使用与维护者完全相同的目录结构或用户名。  
  - 有利于在不同操作系统和 CI 环境中复制示例，减少路径相关错误。  
  - 与规格中“示例代码不得出现维护者绝对路径”的约束一致，同时仍能清晰表达文件位置。

- **Alternatives considered**:  
  - **保留少量绝对路径以示例真实环境**：会与规格和宪章直接冲突，且对读者价值有限。  
  - **完全不显示路径，只描述模块名称**：不利于新手找到配置文件和入口点，降低可操作性。

---

## 4. 文档与 Go 契约/fixtures 的对齐方式

- **Decision**:  
  文档中对 API、请求/响应结构和供应商行为的描述，以现有 Go 契约测试与 fixtures 为单一事实来源：  
  - 继续使用 `specs/001-go-agno-rewrite/contracts/fixtures/` 中的 Python 参考输出与 Go golden 测试作为行为基线；  
  - 在 `specs/001-vitepress-docs/contracts/` 中记录“文档契约”，例如哪些页面必须展示哪些能力、哪些环境变量/端点需要被说明；  
  - 在未来更新 API 或行为时，先更新契约与 fixtures，再同步调整 VitePress 文档与 quickstart。

- **Rationale**:  
  - 通过契约 → 文档 的单向流动，避免“文档先于契约”带来的不一致。  
  - 便于在 CI 中增加 `constitution-check` 或自定义检查，验证文档与契约之间的同步情况。  

- **Alternatives considered**:  
  - **直接从 Go 源码生成 API 文档**：能够降低维护成本，但难以涵盖高级案例和多供应商行为说明，仍需要手写文档。  
  - **完全手工维护 API 文档**：易于偏离真实行为，不利于长期维护。

---

## 5. 文档构建与 CI 集成

- **Decision**:  
  通过 Makefile 新增文档相关目标（名称在后续 tasks 中细化，例如 `docs-build`、`docs-serve`），并在 CI 的 `make constitution-check` 或单独 job 中调用这些目标，确保：
  - 文档在每次变更时至少完成一次构建；  
  - 可以在 CI 中检测丢失页面或明显的链接错误（如 404）。

- **Rationale**:  
  - 与宪章“Makefile 为唯一入口”的约束一致，避免直接在 CI 中调用 npm 命令。  
  - 有助于在文档变更与代码变更一起审查时尽早发现问题。  

- **Alternatives considered**:  
  - **本地构建但不在 CI 中验证**：降低短期复杂度，但容易导致文档站在发布后才暴露构建错误或链接断裂。  
  - **在 CI 中直接调用裸 `npm` 命令**：与现有“统一入口为 Makefile”的约束不符。

