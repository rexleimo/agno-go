# Knowledge API

The Knowledge API is part of the **AgentOS** HTTP server. It is not a separate
AIG service, and it is not provided by the standalone `agno-session` runtime.
Route knowledge requests to an AgentOS deployment built from `cmd/agentos`.

## Base path

All endpoint paths below already include `/api/v1`.

| `AGENTOS_PREFIX` | Knowledge API base path |
| --- | --- |
| unset | `/api/v1/knowledge` |
| `/aig` | `/aig/api/v1/knowledge` |
| `/api/v1` | `/api/v1/knowledge` |
| `/aig/api/v1` | `/aig/api/v1/knowledge` |

Do not configure a gateway and AgentOS to add the same prefix twice.

## Required configuration

A knowledge-enabled AgentOS process requires all of the following:

```text
KNOWLEDGE_ENABLED=true
CHROMA_URL=http://chromadb:8000
CHROMA_TENANT=default_tenant
CHROMA_DATABASE=default_database
KNOWLEDGE_COLLECTION=hno_knowledge
OPENAI_API_KEY=[REDACTED]
EMBEDDING_MODEL=text-embedding-3-small
```

`CHROMADB_URL` is accepted as a temporary compatibility alias for `CHROMA_URL`.
Inside Docker or Kubernetes, use the Chroma service DNS name instead of
`localhost`.

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/knowledge/config` | Read active Knowledge API configuration. |
| `GET` | `/api/v1/knowledge/health` | Verify the vector store is reachable. |
| `POST` | `/api/v1/knowledge/content` | Ingest JSON, plain text, or multipart content. |
| `POST` | `/api/v1/knowledge/upload` | Alias for content ingestion. |
| `POST` | `/api/v1/knowledge/search` | Run a semantic search. |

### Check configuration

```bash
curl -i http://localhost:8080/api/v1/knowledge/config
curl -i http://localhost:8080/api/v1/knowledge/health
```

### Ingest content

```bash
curl -X POST http://localhost:8080/api/v1/knowledge/content \
  -H 'Content-Type: application/json' \
  -d '{
    "content": "AgentOS knowledge content is stored in Chroma.",
    "metadata": {"source": "deployment-check"}
  }'
```

### Search content

```bash
curl -X POST http://localhost:8080/api/v1/knowledge/search \
  -H 'Content-Type: application/json' \
  -d '{"query":"Where is knowledge content stored?","limit":5}'
```

`collection_name` is an optional logical collection scope. When supplied, it
is also applied as a metadata filter so a request cannot silently query a
separate logical collection. The physical Chroma collection is configured by
`KNOWLEDGE_COLLECTION`.

## Diagnose failures

| Response | Meaning | Next check |
| --- | --- | --- |
| `404` | Wrong service, path, prefix, or old image. | Confirm the request reaches AgentOS, not `agno-session`. |
| `503` | AgentOS is reachable but Knowledge API initialization or the vector store is unavailable. | Check Chroma, embedding configuration, and AgentOS logs. |
| `200` with `documents: 0` | Vector store is reachable but the configured namespace is empty. | Compare URL, tenant, database, collection, and writer configuration. |
| `200` search with no results | Search completed without matching chunks. | Check ingestion, metadata filters, and collection scope. |

The same `CHROMA_URL`, `CHROMA_TENANT`, `CHROMA_DATABASE`,
`KNOWLEDGE_COLLECTION`, and embedding model must be used by both ingestion and
search.

## Deployment

- Docker Compose: use `docker-compose.knowledge.yml` together with the
  `with-vectordb` profile.
- Kubernetes: use `deploy/helm/agno-agentos`, not `deploy/helm/agno-session`.
- Put authentication and authorization at the gateway or ingress before
  exposing ingestion or search endpoints publicly.
