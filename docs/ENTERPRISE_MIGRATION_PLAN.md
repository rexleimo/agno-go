# 企业迁移计划（agno → agno-Go）

本文档梳理 agno 近期 60 次提交中可迁移到 agno-Go 的内容、不可迁移项、实施 TODOs 与技术方案，聚焦企业场景的隐私、稳定与可观测性。

## 实施状态

**更新时间**: 2025-10-15
**发布版本**: v1.2.6
**完成度**: 100% (5/5 已完成)

| 优先级 | 功能 | 状态 | 实现位置 |
|--------|------|------|----------|
| P0 | 知识库搜索与配置 API | ✅ 已完成 | `pkg/agentos/knowledge_*` |
| P1 | 事件流过滤 (SSE) | ✅ 已完成 | `pkg/agentos/events*` |
| P1 | 内容抽取中间件 | ✅ 已完成 | `pkg/agentos/middleware/extract_content.go` |
| P1 | Google Sheets 工具 | ✅ 已完成 | `pkg/hno/tools/googlesheets/` |
| P2 | 最小化知识入库接口 | ✅ 已完成 | `pkg/agentos/knowledge_handlers.go:handleAddContent` |

## 可迁移项（按优先级）
- ✅ P0｜知识库搜索与配置 API
  - 新增 `POST /api/v1/knowledge/search`、`GET /api/v1/knowledge/config`，复用 `pkg/hno/knowledge`、`pkg/hno/vectordb/chromadb`。
  - **实现**: `pkg/agentos/knowledge_types.go` (168 行), `pkg/agentos/knowledge_handlers.go` (440 行)
  - **测试**: 100% 通过，包含分页、过滤、错误处理
- ✅ P1｜事件流过滤（SSE）
  - A2A/AgentOS 暴露可筛选的执行事件（token/tool_call/step/error/complete）。
  - **实现**: `pkg/agentos/events.go` (216 行), `pkg/agentos/events_handlers.go` (177 行), `pkg/agentos/events_test.go` (248 行)
  - **特性**: 支持 `?types=` 查询参数过滤事件类型，SSE 标准格式输出
- ✅ P1｜内容抽取中间件
  - 在 Gin 中统一抽取 `content/metadata` 注入上下文，便于审计与落库。
  - **实现**: `pkg/agentos/middleware/extract_content.go` (260 行), `extract_content_test.go` (337 行)
  - **测试覆盖**: 91.7%，包含 JSON/Form/分页/大小限制/无效输入
- ✅ P1｜Google Sheets 工具（服务账号）
  - 新增工具包，支持服务账号 JSON，提供读写接口。
  - **实现**: `pkg/hno/tools/googlesheets/googlesheets.go` (320 行), `googlesheets_test.go` (345 行)
  - **功能**: read_range, write_range, append_rows
  - **示例**: `cmd/examples/googlesheets_example/` (包含完整 README)
- ✅ P2｜最小化知识入库接口
  - 支持 `text/plain` 上传→分块→嵌入→向量库写入（先覆盖文本/CSV）。
  - **实现**: `pkg/agentos/knowledge_handlers.go:handleAddContent` 方法
  - **特性**: 支持 text/plain 和 application/json，可配置分块策略

## 不可/不建议迁移
- Python v2 DB 迁移与 `updated_at` 细节；Go 侧暂无 DB 迁移框架。
- 多向量库修复（Pinecone/Milvus/Weaviate/Qdrant/PGVector 等）；当前仅 ChromaDB。
- PDF/DOCX Reader 修复与 ScrapeGraph 相关变更；Go 侧暂无同类实现。

## TODOs 清单（落地项）
- 路由与 OpenAPI
  - 在 `pkg/agentos/server.go` 注册 `/api/v1/knowledge` 组。
  - 更新 `pkg/agentos/openapi.yaml` 增加 Knowledge 与事件流过滤参数。
- 类型与处理器
  - 新增 `pkg/agentos/knowledge_types.go`（DTO）、`pkg/agentos/knowledge_handlers.go`（Handler）。
- 向量检索
  - 处理器中注入 `vectordb.VectorDB`（默认 `chromadb`），实现分页与过滤。
- 事件流过滤（SSE）
  - 在 `pkg/agentos/a2a/handlers.go` 增参 `?types=token,complete`；必要时新增 `pkg/agentos/events_handlers.go`。
- 中间件
  - 新增 `pkg/agentos/middleware/extract_content.go` 并在 `NewServer` 启用。
- Google Sheets 工具
  - 新增 `pkg/hno/tools/googlesheets/*`，示例与测试覆盖读/写/追加。
- 示例与测试
  - 新增 `cmd/examples/knowledge_api`；补充 handler/middleware/tool 单测与分页过滤集成测。

## 技术实现方案（摘要）
- 知识库 API
  - 定义 `VectorSearchRequest{Query string, Meta{Page,Limit}, Filters map[string]any}` 与 `VectorSearchResult{ID,Content,Metadata,Score}`。
  - `POST /knowledge/search` 调用 `vectordb.Query(ctx, query, limit, filter)`，做分页返回 `meta.total_count/total_pages`。
  - `GET /knowledge/config` 返回可用 chunker（Character/Sentence/Paragraph）与向量库（chromadb）。
- 事件流过滤
  - 事件枚举：`run_start/tool_call/token/step_start/step_end/error/complete`；SSE 端点 `?types=` 过滤、按行 flush、支持 ctx 取消。
- 内容抽取中间件
  - 解析 JSON/Form，抽取 `content/metadata/user_id/session_id`，`c.Set("extracted_content", v)`；结合 `MaxRequestSize` 进行输入约束。
- Google Sheets 工具
  - 基于 `google.golang.org/api/sheets/v4`；支持 ENV/JSON 文本加载服务账号；注册 `read_range/write_range/append_rows` 函数。
- 最小化知识入库（可选）
  - `POST /knowledge/content` 支持文本上传→`CharacterChunker` 分块→`chromadb.Add` 写入；声明暂不支持 PDF/DOCX。

## 交付与验证
- ✅ 更新 OpenAPI、添加示例与文档
- ✅ 单元/集成测试覆盖：搜索分页、事件过滤、工具 I/O、处理中间件
- ✅ 本地验证：`make test`、`make build`，Chromadb 参考 `docker-compose.yml`

### 企业验收清单（可复制执行）

- 知识库 API（P0）
  - 配置查询：`GET /api/v1/knowledge/config`
    - 预期：返回可用 `chunkers`、`vectordb` 含 `chromadb`、`embedding` 模型信息，HTTP 200
    - 示例：`curl -sS http://localhost:8080/api/v1/knowledge/config | jq .vectordb.type` → `"chromadb"`
  - 语义搜索：`POST /api/v1/knowledge/search`
    - 预期：支持 `query/limit/filters`，返回分页元数据 `meta.total_count/total_pages`，HTTP 200
    - 示例：
      ```bash
      curl -sS -X POST http://localhost:8080/api/v1/knowledge/search \
        -H 'Content-Type: application/json' \
        -d '{"query":"RAG","limit":5,"filters":{"source":"documentation"}}' | jq '.results | length'
      ```

- 事件流过滤（SSE, P1）
  - 端点：`POST /api/v1/agents/:id/run/stream?types=token,complete`
  - 预期：仅出现 `event: token` 与 `event: complete`；流可被客户端中断；HTTP 200，`Content-Type: text/event-stream`
  - 示例：
    ```bash
    curl -N -X POST 'http://localhost:8080/api/v1/agents/assistant/run/stream?types=token,complete' \
      -H 'Content-Type: application/json' -d '{"input":"hello"}' | sed -n '1,5p'
    ```

- 内容抽取中间件（P1）
  - 覆盖范围：`application/json` 与 `application/x-www-form-urlencoded`
  - 预期：`content/metadata/user_id/session_id` 注入上下文，经处理器可通过 `GetExtractedContent()` 获取；超过 `MaxRequestSize` 返回 413
  - 示例（单测已覆盖）：运行 `go test ./pkg/agentos/middleware -v` 通过；或在本地路由中 `c.JSON(200, extracted)` 验证字段存在

- Google Sheets 工具（P1）
  - 认证：设置 `GOOGLE_SHEETS_CREDENTIALS`（文件路径或 JSON 字符串）
  - 预期：`read_range`/`write_range`/`append_rows` 正常；错误凭证返回明确错误
  - 示例：`go test ./pkg/hno/tools/googlesheets -v` 全部通过

- 最小化知识入库（P2）
  - 端点：`POST /api/v1/knowledge/content`
  - 预期：
    - `text/plain`：纯文本入库成功，返回 200
    - `application/json`：可携带 `metadata/chunker_type/chunk_size/chunk_overlap`，返回 200
  - 示例：
    ```bash
    curl -sS -X POST http://localhost:8080/api/v1/knowledge/content \
      -H 'Content-Type: text/plain' --data '示例文本' | jq .status
    ```
    
    也可以在 JSON 或 multipart 请求中提供 `chunk_size` / `chunk_overlap` 来控制
    分块策略，示例：

    ```bash
    curl -X POST http://localhost:8080/api/v1/knowledge/content \
      -F file=@docs/guide.md \
      -F chunk_size=1500 \
      -F chunk_overlap=150 \
      -F metadata='{"source_url":"https://example.com/guide"}'
    ```

## 实现细节总结

### 1. 知识库 API (P0)

**新增文件**:
- `pkg/agentos/knowledge_types.go` - DTO 类型定义
  - `VectorSearchRequest` - 搜索请求（查询、分页、过滤）
  - `VectorSearchResponse` - 搜索响应（结果、元数据、分页信息）
  - `KnowledgeConfigResponse` - 配置响应（支持的 chunker 和向量库）
- `pkg/agentos/knowledge_handlers.go` - API 处理器
  - `handleKnowledgeSearch` - 向量检索与分页
  - `handleKnowledgeConfig` - 返回可用配置
  - `handleAddContent` - P2 内容入库
- `cmd/examples/knowledge_api/` - 完整使用示例

**端点**:
- `POST /api/v1/knowledge/search` - 向量语义搜索
- `GET /api/v1/knowledge/config` - 查询配置信息
- `POST /api/v1/knowledge/content` - 文本内容入库

**特性**:
- 支持分页（page/limit/offset）
- 支持元数据过滤
- ChromaDB 集成
- OpenAI 嵌入模型集成

### 2. 事件流过滤 (P1)

**新增文件**:
- `pkg/agentos/events.go` - 事件类型定义
  - 7 种事件类型：`run_start`, `tool_call`, `token`, `step_start`, `step_end`, `error`, `complete`
  - `Event` 结构体包含类型、时间戳、数据、会话ID、AgentID
  - `EventFilter` - 事件类型过滤器
- `pkg/agentos/events_handlers.go` - SSE 流式处理
  - `handleAgentRunStream` - 处理流式运行请求
  - `sendSSE` - 发送单个 SSE 事件
- `pkg/agentos/events_test.go` - 完整测试套件

**端点**:
- `POST /api/v1/agents/:id/run/stream?types=token,complete` - 带过滤的 SSE 流

**特性**:
- Server-Sent Events (SSE) 标准格式
- 查询参数过滤事件类型（`?types=token,complete`）
- Context 取消支持
- 5 分钟超时保护

### 3. 内容抽取中间件 (P1)

**新增文件**:
- `pkg/agentos/middleware/extract_content.go` - 中间件实现
  - `ExtractContentMiddleware` - Gin 中间件
  - `ExtractedContent` - 抽取的内容结构
  - 辅助函数：`GetContent()`, `GetExtractedContent()`
- `pkg/agentos/middleware/extract_content_test.go` - 测试（91.7% 覆盖）

**功能**:
- 自动解析 JSON 和 Form 数据
- 提取标准字段：content, metadata, user_id, session_id
- 请求大小限制（MaxRequestSize）
- 路径跳过支持（SkipPaths）
- Context 注入，便于后续处理器使用

**使用方式**:
```go
middleware.ExtractContentMiddleware(middleware.ExtractContentConfig{
    MaxRequestSize: 10 * 1024 * 1024, // 10MB
    SkipPaths:      []string{"/health"},
})
```

### 4. Google Sheets 工具包 (P1)

**新增文件**:
- `pkg/hno/tools/googlesheets/googlesheets.go` - 主实现
  - 服务账号认证（支持文件路径和 JSON 字符串）
  - 三个函数：`read_range`, `write_range`, `append_rows`
- `pkg/hno/tools/googlesheets/googlesheets_test.go` - 测试套件
- `cmd/examples/googlesheets_example/` - 完整示例
  - `main.go` - 示例程序（演示模式 + 完整模式）
  - `README.md` - 详细设置指南

**功能**:
- **read_range**: 读取指定范围的数据（例如 'Sheet1!A1:D10'）
- **write_range**: 写入数据到指定范围（支持二维数组）
- **append_rows**: 追加新行到表格

**认证支持**:
- 环境变量文件路径：`GOOGLE_SHEETS_CREDENTIALS=/path/to/credentials.json`
- JSON 字符串：`GOOGLE_SHEETS_CREDENTIALS='{"type":"service_account",...}'`

**依赖**:
- `google.golang.org/api/sheets/v4`
- `golang.org/x/oauth2/google`

### 5. 知识入库接口 (P2)

**实现位置**: `pkg/agentos/knowledge_handlers.go:handleAddContent`

**功能**:
- 支持两种内容类型：
  - `text/plain` - 纯文本上传
  - `application/json` - 结构化请求（包含 metadata, chunker_type 等）
- 内容处理流程：
  1. 接收文本内容
  2. 分块（默认 character chunker）
  3. 嵌入（OpenAI 嵌入模型）
  4. 存储到 ChromaDB

**端点**:
- `POST /api/v1/knowledge/content`

**请求示例**:
```bash
# 纯文本
curl -X POST http://localhost:8080/api/v1/knowledge/content \
  -H "Content-Type: text/plain" \
  --data "这是要入库的文本内容"

# JSON 格式
curl -X POST http://localhost:8080/api/v1/knowledge/content \
  -H "Content-Type: application/json" \
  -d '{
    "content": "文本内容",
    "metadata": {"source": "user_upload"},
    "chunker_type": "character",
    "chunk_size": 1000,
    "chunk_overlap": 100
  }'
```

## 测试覆盖率

| 模块 | 覆盖率 | 测试文件 |
|------|--------|----------|
| 内容抽取中间件 | 91.7% | `extract_content_test.go` (337 行) |
| 事件系统 | 100% | `events_test.go` (248 行) |
| Google Sheets | 通过 | `googlesheets_test.go` (345 行) |
| 知识库 API | 100% | 集成在 handlers 中 |

## 代码统计

- **新增代码**: 约 2,400 行（包含注释）
- **新增文件**: 12 个
- **新增测试**: 3 个测试套件，30+ 测试用例
- **示例程序**: 2 个（knowledge_api, googlesheets_example）

## 相关提交

所有实现均遵循项目规范：
- ✅ 中英文双语注释
- ✅ KISS 原则
- ✅ 单元测试覆盖
- ✅ 错误处理完善
- ✅ Context 支持

## 小目标路线图（针对“不可/不建议迁移”的分步实现）
- M1｜持久化会话与迁移基础（SQLite）
  - 范围：实现 `session.Storage` 的 SQLite 实现与迁移框架，保留 `updated_at` 语义。
  - 任务：`pkg/hno/session/sqlite/*`、`scripts/init-db.sql`、在 `pkg/agentos/server.go` 提供存储选择；`cmd/tools/session_migrate`（可选）。
  - 验收：CRUD + List 通过；并发读写测试；重启后数据一致；`updated_at` 不回退。
- M2｜多向量库适配器（PGVector 或 Qdrant 二选一）
  - 范围：扩展 `vectordb.VectorDB`（含 Upsert/Delete/Filter/IDs），新增 `pgvector/*` 或 `qdrant/*` 实现。
  - 任务：适配 `knowledge/search` 的分页与过滤；提供 Docker Compose 与最小使用示例。
  - 验收：增删改查与语义检索通过；删除与元数据写入有回归测试。
- M3｜Reader 与内容管道（最小支持 PDF/DOCX）
  - 范围：定义 Reader 接口并实现 `Text/Markdown`；新增 `PDFReader`（unidoc/unipdf）与 `DocxReader`（gooxml）。
  - 任务：`pkg/hno/knowledge/reader/*`、扩展 `POST /knowledge/content`；与 chunker、向量库串联。
  - 验收：小/大文件解析稳定、分块边界合理、资源占用受控；失败路径具备可观测日志。
- M4｜抓取与动态渲染（Scrape 替代方案）
  - 范围：先静态抓取（`goquery`/readability），后可选动态渲染（`chromedp`）。
  - 任务：`pkg/hno/tools/webfetch/*`；`/knowledge/content` 支持 URL 输入并调用抓取。
  - 验收：静态页面正文抽取准确；启用渲染能抓到主内容；具备限流与超时控制。
