# agno-agentos Helm chart

This chart deploys the `cmd/agentos` HTTP server. It is separate from
`agno-session`: the session chart does not include the AgentOS Knowledge API.

## Prerequisites

- A built and pushed image from the root `Dockerfile`.
- A reachable Chroma server.
- A Kubernetes Secret containing the OpenAI embedding API key.
- Authentication at the ingress or gateway. AgentOS does not add authentication
  by default.

Create a Secret without placing a key in Helm values or shell history:

```sh
kubectl -n <namespace> create secret generic agentos-openai \
  --from-literal=OPENAI_API_KEY="$OPENAI_API_KEY"
```

Install a knowledge-enabled instance:

```sh
helm upgrade --install agentos deploy/helm/agno-agentos \
  --namespace <namespace> --create-namespace \
  --set image.repository=<registry>/agno-agentos \
  --set image.tag=<immutable-tag> \
  --set knowledge.enabled=true \
  --set knowledge.existingSecret=agentos-openai \
  --set knowledge.chromaURL=http://chromadb:8000 \
  --set knowledge.collection=hno_knowledge
```

The default knowledge readiness probe is `/api/v1/knowledge/health`. If an
`agentos.prefix` is configured, set `readinessProbe.knowledgePath` to the
corresponding mounted route. Liveness remains `/health`.

## Optional persistent sessions

For PostgreSQL-backed AgentOS sessions, create a second Secret with an
`AGENTOS_SESSION_DSN` key and enable the session section:

```sh
helm upgrade --install agentos deploy/helm/agno-agentos \
  --namespace <namespace> \
  --set session.enabled=true \
  --set session.existingSecret=agentos-session-postgres
```

The chart applies the `agno_agentos_sessions` schema with a pre-install and
pre-upgrade migration Job. Disable `session.migrations.enabled` only when that
migration is managed separately.
