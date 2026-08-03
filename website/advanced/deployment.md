# Deployment Guide

Complete guide for deploying AgentOS in production environments.

## Quick Start

### Using Docker Compose (Recommended)

```bash
# 1. Clone repository
git clone https://github.com/rexleimo/agno-Go.git
cd agno-Go

# 2. Configure environment
cp .env.example .env
nano .env  # Add your API keys

# 3. Start services
docker-compose up -d

# 4. Verify
curl http://localhost:8080/health
```

## Docker Deployment

### Single Container

```bash
# Build image
docker build -t agentos:latest .

# Run container
docker run -d \
  -p 8080:8080 \
  -e OPENAI_API_KEY=sk-your-key \
  --name agentos \
  agentos:latest
```

### Docker Compose Full Stack

The `docker-compose.yml` includes:

- **AgentOS** - HTTP server (port 8080)
- **PostgreSQL** - Database (port 5432)
- **Redis** - Cache (port 6379)
- **ChromaDB** - Vector DB (port 8000, optional)
- **Ollama** - Local models (port 11434, optional)

```bash
# Start core services
docker-compose up -d

# Start with optional services
docker-compose --profile with-ollama --profile with-vectordb up -d

# View logs
docker-compose logs -f agentos

# Stop services
docker-compose down
```

## Kubernetes Deployment

### Basic Deployment

```bash
# Apply manifests
kubectl apply -f k8s/

# Check status
kubectl get pods
kubectl get services

# View logs
kubectl logs -f deployment/agentos
```

### Kubernetes Resources

The `k8s/` directory includes:

- **deployment.yaml** - AgentOS deployment with health probes
- **service.yaml** - LoadBalancer service
- **configmap.yaml** - Configuration
- **secret.yaml** - API keys
- **hpa.yaml** - Horizontal Pod Autoscaler

### Example Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentos
spec:
  replicas: 3
  selector:
    matchLabels:
      app: agentos
  template:
    metadata:
      labels:
        app: agentos
    spec:
      containers:
      - name: agentos
        image: agentos:latest
        ports:
        - containerPort: 8080
        env:
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: agentos-secrets
              key: openai-api-key
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

## Cloud Platform Deployment

### AWS ECS

```bash
# Build and push image
docker build -t agno/agentos:latest .
docker tag agno/agentos:latest <aws_account_id>.dkr.ecr.us-east-1.amazonaws.com/agentos:latest
docker push <aws_account_id>.dkr.ecr.us-east-1.amazonaws.com/agentos:latest

# Create ECS task definition and service
aws ecs create-service --cli-input-json file://ecs-service.json
```

### Google Cloud Run

```bash
# Build and deploy
gcloud builds submit --tag gcr.io/<project-id>/agentos
gcloud run deploy agentos \
  --image gcr.io/<project-id>/agentos \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated
```

### Azure Container Instances

```bash
# Deploy container
az container create \
  --resource-group myResourceGroup \
  --name agentos \
  --image <registry>/agentos:latest \
  --dns-name-label agentos-demo \
  --ports 8080 \
  --environment-variables OPENAI_API_KEY=sk-your-key
```

## Environment Configuration

### Required Variables

```bash
# LLM API Keys
OPENAI_API_KEY=sk-your-openai-key

# Server Config
AGENTOS_ADDRESS=:8080
```

### Optional Variables

```bash
# Additional LLM Providers
ANTHROPIC_API_KEY=sk-ant-your-key
OLLAMA_BASE_URL=http://localhost:11434

# Logging
LOG_LEVEL=info  # debug, info, warn, error
AGENTOS_DEBUG=false

# Timeouts
REQUEST_TIMEOUT=30

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/agentos

# Redis Cache
REDIS_URL=redis://localhost:6379/0

# ChromaDB
CHROMA_URL=http://localhost:8000
```

## Database Setup

### PostgreSQL

```bash
# Using Docker
docker run -d \
  --name postgres \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=agentos \
  -p 5432:5432 \
  postgres:15

# Connect
export DATABASE_URL=postgresql://postgres:password@localhost:5432/agentos
```

### Redis

```bash
# Using Docker
docker run -d \
  --name redis \
  -p 6379:6379 \
  redis:7-alpine

# Connect
export REDIS_URL=redis://localhost:6379/0
```

## Monitoring & Logging

### Health Checks

```bash
# Health endpoint
curl http://localhost:8080/health

# Response
{
  "status": "healthy",
  "service": "agentos",
  "time": 1704067200
}
```

### Structured Logging

AgentOS uses Go's `log/slog` for structured logging:

```json
{
  "time": "2025-10-02T10:00:00Z",
  "level": "INFO",
  "msg": "Server started",
  "address": ":8080"
}
```

### Prometheus Metrics (Future)

```go
// Planned metrics
agno_agent_creations_total
agno_agent_run_duration_seconds
agno_http_requests_total
agno_http_request_duration_seconds
```

## Security Best Practices

### 1. Use Secrets Management

```bash
# Kubernetes Secrets
kubectl create secret generic agentos-secrets \
  --from-literal=openai-api-key=sk-your-key

# AWS Secrets Manager
aws secretsmanager create-secret \
  --name agentos/openai-api-key \
  --secret-string sk-your-key
```

### 2. Enable HTTPS

```bash
# Using reverse proxy (nginx, Caddy)
# Or load balancer (ALB, Cloud Load Balancer)
```

### 3. Rate Limiting

Use reverse proxy or API gateway for rate limiting.

### 4. Input Validation

Use built-in guardrails:

```go
import "github.com/rexleimo/agno-Go/pkg/hno/guardrails"

promptGuard := guardrails.NewPromptInjectionGuardrail()
agent.PreHooks = []hooks.Hook{promptGuard}
```

### 5. Non-Root Container

Dockerfile already uses non-root user:

```dockerfile
USER nonroot:nonroot
```

## Performance Tuning

### 1. Resource Limits

```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### 2. Horizontal Scaling

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: agentos-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: agentos
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### 3. Connection Pooling

For database connections, use connection pooling:

```go
db, err := sql.Open("postgres", databaseURL)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

## Troubleshooting

### Common Issues

**1. "Connection refused"**

Check if server is running:
```bash
docker ps
kubectl get pods
```

**2. "API key not set"**

Verify environment variables:
```bash
docker exec agentos env | grep API_KEY
kubectl exec <pod> -- env | grep API_KEY
```

**3. "Out of memory"**

Increase memory limits:
```yaml
resources:
  limits:
    memory: "1Gi"
```

**4. "Too many open files"**

Increase file descriptor limits:
```bash
ulimit -n 65536
```

### Debug Logging

Enable debug mode:

```bash
export AGENTOS_DEBUG=true
export LOG_LEVEL=debug
```

## Production Checklist

- [ ] Set secure API keys
- [ ] Enable HTTPS
- [ ] Configure health checks
- [ ] Set resource limits
- [ ] Enable monitoring
- [ ] Set up logging
- [ ] Configure backups
- [ ] Test disaster recovery
- [ ] Document runbooks
- [ ] Set up alerts

## References

- [Architecture](/advanced/architecture)
- [Performance](/advanced/performance)
- [API Reference](/api/)
- [Examples](/examples/)
- [GitHub Repository](https://github.com/rexleimo/agno-Go)
