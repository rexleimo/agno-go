# Data Model: 官方文档站（用户文档）

**Feature**: `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/spec.md`  
**Branch**: `001-vitepress-docs`  
**Date**: 2025-11-22

本数据模型描述与官方文档站相关的核心概念与关系，用于指导 VitePress 文档工程的页面设计、导航配置与示例管理。模型聚焦“文档内容”本身，不涉及底层数据库或具体实现细节。

---

## 1. 实体：DocumentPage（文档页面）

**含义**：  
表示文档站中的单个页面，承载某一主题的说明、用户故事与代码示例。

**关键字段**：
- `id`：页面唯一标识（例如基于路径或 slug，如 `getting-started`、`providers/matrix`）。  
- `locale`：语言标识（`en` / `zh` / `ja` / `ko`）。  
- `title`：页面标题。  
- `section`：所属章节（例如 `Overview`、`Quickstart`、`Core Features`、`Advanced Guides`、`Contributing`）。  
- `path`：在文档站中的相对路由路径（如 `/quickstart`、`/zh/advanced/providers`）。  
- `summary`：对页面内容的简短概述，便于搜索与导航。  
- `lastUpdated`：最近一次内容更新的时间（用于“最后更新”标记）。  

**校验规则**：
- 相同 `id` 在不同 `locale` 下必须指向语义等价的内容（结构和示例一致，文字为本地化版本）。  
- 所有 `path` 必须为文档站内的相对路径，不包含维护者本机的绝对文件系统路径。  
- 首页和快速开始页面在所有 locale 中必须存在且可被导航到。  

---

## 2. 实体：CodeSample（代码示例）

**含义**：  
描述文档中的一个可复制代码片段或完整示例，用于演示如何使用 Go AgentOS 或相关工具。

**关键字段**：
- `id`：示例唯一标识（例如 `quickstart-basic-agent`、`advanced-multi-provider-routing`）。  
- `pageId`：所属 `DocumentPage.id`。  
- `language`：示例语言（如 `go`、`bash`、`yaml`）。  
- `title`：示例标题（例如“启动 AgentOS 服务”“创建会话并发送消息”）。  
- `description`：示例目的的简短文字说明。  
- `snippet`：示例代码内容（用于渲染到文档页面）。  
- `requiresEnv`：布尔值，是否依赖外部模型供应商密钥或特殊配置。  
- `relatedContracts`：与该示例相关的契约或测试标识（例如某个 OpenAPI 端点或契约测试用例名称）。  

**校验规则**：
- `snippet` 中不得包含维护者本机绝对路径（如以 `/Users/`、`C:\Users\` 开头的路径）；  
  只允许使用相对路径或通用占位符（如 `<project-root>/go/cmd/agno`、`./config/default.yaml`）。  
- 若 `requiresEnv = true`，文案必须明确列出所需 env 变量名称（例如 `OPENAI_API_KEY`），并说明不得提交真实 key。  
- 每个 `CodeSample` 至少应能在合理配置下独立运行或拷贝到用户项目中使用。  

---

## 3. 实体：AdvancedScenario（高级案例）

**含义**：  
代表一篇完整的高级案例文章，通常对应文档站中的一篇长文，包含多个步骤和一个或多个 `CodeSample`。

**关键字段**：
- `id`：高级案例唯一标识（如 `advanced-multi-provider-routing`、`memory-augmented-chat`）。  
- `pageId`：对应的 `DocumentPage.id`。  
- `title`：案例标题。  
- `summary`：用例的业务背景和目标。  
- `steps`：高层步骤列表（例如“准备配置”“启动服务”“调用 API 并观察行为”）。  
- `samples`：关联的 `CodeSample.id` 列表。  

**校验规则**：
- 每个 `AdvancedScenario` 须至少包含一个完整可运行链路（从准备环境到观测结果），并明确所需前置条件。  
- 不得引导用户使用与 Go 契约/fixtures 明显不一致的 API 或行为。  

---

## 4. 实体：ProviderCapabilityMatrix（供应商能力矩阵）

**含义**：  
描述九家模型供应商在不同能力上的支持情况，用于生成“模型供应商矩阵”页面。

**关键字段**：
- `provider`：供应商名称（`ollama`、`gemini`、`openai`、`glm4`、`openrouter`、`siliconflow`、`cerebras`、`modelscope`、`groq`）。  
- `supportsChat`：是否支持聊天能力。  
- `supportsEmbedding`：是否支持向量嵌入。  
- `supportsStreaming`：是否支持流式输出。  
- `envVars`：文档中需要说明的环境变量名称列表。  
- `notes`：关于限额、可用区域、已知偏差等的简短说明。  

**校验规则**：
- 所有字段内容必须与 Go 实现和现有契约/fixtures 一致；若存在偏差，需在 `contracts/deviations.md` 中说明。  
- 文档中展示的 `envVars` 名称必须与 `/Users/molei/codes/agno-Go/.env.example` 中保持一致。  

---

## 5. 实体：DocNavigation（文档导航）

**含义**：  
表示侧边栏/顶部导航的结构，用于确保不同语言版本之间的导航一致。

**关键字段**：
- `id`：导航项标识。  
- `locale`：语言标识。  
- `label`：显示文本。  
- `targetPageId`：指向的 `DocumentPage.id`。  
- `children`：子导航项列表（可为空）。  

**校验规则**：
- 对于给定的 `id`，各个 `locale` 下的结构（children 关系）必须一致，仅 `label` 文本可本地化。  
- 所有需要“首屏可达”的页面（如首页、快速开始、模型矩阵、高级案例入口）必须出现在导航结构中。  

---

## 6. 状态与流转（高层）

从“无文档”到“可对外发布”的基本状态流转：

1. **草稿 Draft**：  
   - 文档页面与示例在 `specs/001-vitepress-docs/` 中完成初稿设计（包括本数据模型与 quickstart）。  
   - VitePress 工程 `/Users/molei/codes/agno-Go/docs/` 中存在对应草稿页面。

2. **对齐 Aligning**：  
   - 页面内容与 Go 契约/fixtures、`.env.example` 中的配置说明完成对齐。  
   - 多语言结构在 `DocNavigation` 层面保持一致，但可能存在尚未翻译的内容。

3. **可发布 Ready**：  
   - 所有核心页面（首页、快速开始、核心功能、供应商矩阵、高级案例、贡献与质量保障）在四种语言中均已存在。  
   - 所有关联示例通过手动或自动验证，确保可运行且不包含维护者绝对路径。  
   - `make docs-build`（或等效目标）在 CI 中稳定通过。

4. **已发布 Published**：  
   - 对应版本的文档已部署至 `https://rexai.top/agno-Go/`，并在发布说明或变更日志中附上链接。  

该状态流转仅作为规划参考，不绑定具体实现方式。

