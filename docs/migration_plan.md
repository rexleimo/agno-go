# v2.1.5 功能迁移规划（Agno-Go）

## CI 升级巡检
- 复用：`Makefile` 中 `lint`、`test` 目标与既有脚本。
- 任务：新增统一 CI 工作流（测试 + 静态检查），对接缓存策略，保持与 Python 仓库一致的约束。

## 异步数据库接口
- 复用：`pkg/hno/session/storage.go` 定义的存储接口、`pkg/hno/session/db/batch` 现有批量写入结构。
- 任务：抽象统一的数据库接口层，补充上下文超时与批量节流配置，保证与内存/批量实现兼容。

## 异步 Postgres 驱动
- 复用：`pkg/hno/session/db/batch/postgres.go` 的 COPY + 临时表策略、参数化配置。
- 任务：拆分连接管理与批量调度，增加 goroutine 安全的连接池封装与重试逻辑，补充回归测试。

## SurrealDB 引擎
- 复用：`pkg/hno/session`、`pkg/hno/memory` 公共模型，`pkg/hno/session/db/batch` 的接口约束。
- 任务：引入 `surrealdb` Go 客户端，实现 CRUD、批量 upsert、指标统计，并保持 KISS 接口。

## Surreal Demo
- 复用：`cmd/examples/postgres_storage`、`cmd/examples/knowledge_api` 项目骨架。
- 任务：新增 `cmd/examples/surreal_demo`，覆盖环境配置、建表脚本、README 场景说明。

## 迁移负载防护
- 复用：`pkg/hno/session/db/batch.Config`、`PostgresBatchWriter` 超时与重试参数。
- 任务：增加批量节流（批次间 Sleep）、动态批量尺寸调节，以及清理 goroutine 的资源回收。

## 元数据保留
- 复用：`session.Session` 结构体（包含 agent/team/workflow 字段）、批量写入原逻辑。
- 任务：确保 upsert 时保留 `created_at`、`updated_at`、`team_id`、`agent_id`，补充单测覆盖。

## 内容中间件抽取
- 复用：`pkg/agentos/middleware/extract_content.go` 现有中间件。
- 任务：扩展为可配置链路，支持工具输出预处理、JSON schema 校验的可插拔策略。

## 知识 API 增强
- 复用：`pkg/agentos/knowledge_handlers.go`、`agentos.Config` 中的向量检索配置。
- 任务：新增搜索配置开关、分页元数据、MCP URL 放宽策略、运行时健康检查端点。

## 运行结果整洁
- 复用：`pkg/agentos/session_handlers.go` 响应模型及 `pkg/agentos/events.go` 事件流。
- 任务：保证媒体数据挂载在顶层 `GET /runs` 响应，修正结果变量初始化，补测 MySQL/Memory 场景。

## 推理适配器扩展
- 复用：`pkg/hno/reasoning` 现有适配层、`pkg/hno/models` 中多模型实现。
- 任务：补全 Gemini / Anthropic / VertexAI 推理管道的配置项、错误处理与单测矩阵。

## 可观测性刷新
- 复用：`pkg/agentos/events`、`cmd/examples/event_stream_demo` 示例。
- 任务：新增 Logfire 示例、事件过滤参数、Cookbook 清理与 README 更新。

## 文档梳理
- 复用：`docs/`、`website/`、`release_notes/` 现有结构。
- 任务：同步 README、AgentOS Cookbook、多语言导航与知识 API 文档，确保 lint 通过。
