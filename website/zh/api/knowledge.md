# 知识库 API

知识库 API 是 **AgentOS** HTTP 服务的一部分，不是独立的 AIG 服务，也不由独立的
`agno-session` 运行时提供。知识库请求必须转发到由 `cmd/agentos` 构建的 AgentOS
服务。

## 路由基础路径

下列路径已经包含 `/api/v1`。

| `AGENTOS_PREFIX` | 知识库 API 基础路径 |
| --- | --- |
| 未设置 | `/api/v1/knowledge` |
| `/aig` | `/aig/api/v1/knowledge` |
| `/api/v1` | `/api/v1/knowledge` |
| `/aig/api/v1` | `/aig/api/v1/knowledge` |

不要让网关和 AgentOS 同时追加同一段 Prefix，否则会得到
`/api/v1/api/v1/...` 这样的错误路径。

## 必需配置

启用知识库的 AgentOS 进程需要完整配置：

```text
KNOWLEDGE_ENABLED=true
CHROMA_URL=http://chromadb:8000
CHROMA_TENANT=default_tenant
CHROMA_DATABASE=default_database
KNOWLEDGE_COLLECTION=hno_knowledge
OPENAI_API_KEY=[REDACTED]
EMBEDDING_MODEL=text-embedding-3-small
```

`CHROMADB_URL` 仍可作为 `CHROMA_URL` 的兼容别名。Docker 或 Kubernetes
容器中必须使用 Chroma Service 的 DNS 名称，不能使用 `localhost`。

## 接口列表

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/knowledge/config` | 获取当前知识库 API 配置。 |
| `GET` | `/api/v1/knowledge/health` | 检查向量数据库是否可用。 |
| `POST` | `/api/v1/knowledge/content` | 写入 JSON、纯文本或 multipart 内容。 |
| `POST` | `/api/v1/knowledge/upload` | 内容写入接口的别名。 |
| `POST` | `/api/v1/knowledge/search` | 执行语义搜索。 |

### 查看配置和健康状态

```bash
curl -i http://localhost:8080/api/v1/knowledge/config
curl -i http://localhost:8080/api/v1/knowledge/health
```

### 写入内容

```bash
curl -X POST http://localhost:8080/api/v1/knowledge/content \
  -H 'Content-Type: application/json' \
  -d '{
    "content": "AgentOS 知识库内容存储在 Chroma 中。",
    "metadata": {"source": "deployment-check"}
  }'
```

### 搜索内容

```bash
curl -X POST http://localhost:8080/api/v1/knowledge/search \
  -H 'Content-Type: application/json' \
  -d '{"query":"知识库内容存在哪里？","limit":5}'
```

`collection_name` 是可选的逻辑集合范围。传入该字段时，服务会自动把它加入
metadata 过滤条件，避免请求表面上选择了某个集合、实际却查询到其他逻辑集合。
实际 Chroma 物理集合由 `KNOWLEDGE_COLLECTION` 决定。

## 排障结果对照

| 返回结果 | 含义 | 下一步 |
| --- | --- | --- |
| `404` | 请求到了错误服务、错误路径、错误 Prefix 或旧镜像。 | 确认流量到达 AgentOS，而不是 `agno-session`。 |
| `503` | 已到达 AgentOS，但知识库初始化或向量库不可用。 | 检查 Chroma、embedding 配置和 AgentOS 日志。 |
| `200` 且 `documents: 0` | 向量库可用，但当前 namespace 没有数据。 | 对比 URL、tenant、database、collection 和写入端配置。 |
| `200` 搜索为空 | 搜索成功但没有匹配 chunk。 | 检查是否实际入库、metadata 过滤条件和集合范围。 |

写入与查询必须使用完全相同的：

```text
CHROMA_URL
CHROMA_TENANT
CHROMA_DATABASE
KNOWLEDGE_COLLECTION
EMBEDDING_MODEL
```

## 部署方式

- Docker Compose：使用 `docker-compose.knowledge.yml` 和 `with-vectordb` profile。
- Kubernetes：使用 `deploy/helm/agno-agentos`，不要把知识库接口部署到
  `deploy/helm/agno-session`。
- 对外开放搜索或入库前，应在 Gateway/Ingress 层配置认证和授权。
