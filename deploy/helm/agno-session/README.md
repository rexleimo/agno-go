# agno-session Helm chart

This chart deploys the standalone session service (`cmd/agentos-session`). It
does **not** deploy the AgentOS Knowledge API. Use `deploy/helm/agno-agentos`
for knowledge endpoints.

## Database secret

Create a Secret containing the PostgreSQL DSN before installation:

```bash
kubectl -n <namespace> create secret generic agno-session-postgres \
  --from-literal=AGNO_PG_DSN='[REDACTED]'
```

Install with the Secret reference instead of placing a DSN in Helm values:

```bash
helm upgrade --install agno-session deploy/helm/agno-session \
  --namespace <namespace> \
  --set config.existingSecret=agno-session-postgres
```

## Migration behavior

When a DSN or `config.existingSecret` is configured, the chart creates a
pre-install/pre-upgrade migration Job. It applies the `agno_sessions` schema
before the service Deployment starts. Set `migrations.enabled=false` only when
your platform runs the same migration independently.

For local Compose usage, `docker-compose.session.yml` mounts
`scripts/migrations/002_session_service.sql` into PostgreSQL initialization.
