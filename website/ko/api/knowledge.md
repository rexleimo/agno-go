# 지식 API

지식 API는 **AgentOS** HTTP 서버의 일부입니다. 별도의 AIG 서비스가 아니며,
독립 실행형 `agno-session` 런타임에서는 제공되지 않습니다.

## 경로

아래 경로에는 이미 `/api/v1`이 포함되어 있습니다.

| `AGENTOS_PREFIX` | API 기본 경로 |
| --- | --- |
| 미설정 | `/api/v1/knowledge` |
| `/aig` | `/aig/api/v1/knowledge` |
| `/api/v1` | `/api/v1/knowledge` |

Gateway와 AgentOS가 같은 Prefix를 중복 추가하여
`/api/v1/api/v1/...` 경로가 되지 않도록 하세요.

## 필수 설정

```text
KNOWLEDGE_ENABLED=true
CHROMA_URL=http://chromadb:8000
CHROMA_TENANT=default_tenant
CHROMA_DATABASE=default_database
KNOWLEDGE_COLLECTION=hno_knowledge
OPENAI_API_KEY=[REDACTED]
```

Docker/Kubernetes에서는 `localhost` 대신 Chroma Service DNS 이름을 사용해야 합니다.
`CHROMADB_URL`은 `CHROMA_URL`의 호환 별칭입니다.

## 엔드포인트

```text
GET  /api/v1/knowledge/config
GET  /api/v1/knowledge/health
POST /api/v1/knowledge/content
POST /api/v1/knowledge/upload
POST /api/v1/knowledge/search
```

- `404`: 잘못된 Service, 경로, Prefix 또는 오래된 image입니다.
- `503`: AgentOS에는 도달했지만 Chroma 또는 embedding이 준비되지 않았습니다.
- `200` 및 `documents: 0`: 현재 tenant/database/collection namespace가 비어 있습니다.

Kubernetes에서는 `deploy/helm/agno-agentos`를 사용하세요.
`deploy/helm/agno-session`은 지식 API를 제공하지 않습니다.
