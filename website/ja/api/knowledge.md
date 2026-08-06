# ナレッジ API

ナレッジ API は **AgentOS** HTTP サーバーの一部です。独立した AIG サービスでは
なく、`agno-session` のスタンドアロン runtime からは提供されません。

## パス

API パスはすでに `/api/v1` を含みます。

| `AGENTOS_PREFIX` | API ベースパス |
| --- | --- |
| 未設定 | `/api/v1/knowledge` |
| `/aig` | `/aig/api/v1/knowledge` |
| `/api/v1` | `/api/v1/knowledge` |

`/api/v1/api/v1/...` にならないよう、Gateway と AgentOS で同じ Prefix を
二重に追加しないでください。

## 必須設定

```text
KNOWLEDGE_ENABLED=true
CHROMA_URL=http://chromadb:8000
CHROMA_TENANT=default_tenant
CHROMA_DATABASE=default_database
KNOWLEDGE_COLLECTION=hno_knowledge
OPENAI_API_KEY=[REDACTED]
```

Docker/Kubernetes では `localhost` ではなく Chroma Service の DNS 名を使用します。
`CHROMADB_URL` は `CHROMA_URL` の互換別名です。

## エンドポイント

```text
GET  /api/v1/knowledge/config
GET  /api/v1/knowledge/health
POST /api/v1/knowledge/content
POST /api/v1/knowledge/upload
POST /api/v1/knowledge/search
```

- `404`: 誤った Service、パス、Prefix、または古い image。
- `503`: AgentOS には到達したが、Chroma または embedding が利用不可。
- `200` と `documents: 0`: tenant/database/collection の現在の namespace が空。

Kubernetes では `deploy/helm/agno-agentos` を使用してください。
`deploy/helm/agno-session` はナレッジ API を提供しません。
