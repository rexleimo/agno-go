# 部署指南 / Deployment Guide

在生产环境中部署 AgentOS 的完整指南。

## 快速开始 / Quick Start

### 使用 Docker Compose (推荐) / Using Docker Compose (Recommended)

```bash
# 1. 克隆仓库 / Clone repository
git clone https://github.com/rexleimo/agno-Go.git
cd agno-Go

# 2. 配置环境 / Configure environment
cp .env.example .env
nano .env  # Add your API keys

# 3. 启动服务 / Start services
docker-compose up -d

# 4. 验证 / Verify
curl http://localhost:8080/health
```

## Docker 部署 / Docker Deployment

### 单容器 / Single Container

```bash
# 构建镜像 / Build image
docker build -t agentos:latest .

# 运行容器 / Run container
docker run -d \
  -p 8080:8080 \
  -e OPENAI_API_KEY=sk-your-key \
  --name agentos \
  agentos:latest
```

### Docker Compose 完整栈 / Docker Compose Full Stack

`docker-compose.yml` 包含:

- **AgentOS** - HTTP 服务器(端口 8080)
- **PostgreSQL** - 数据库(端口 5432)
- **Redis** - 缓存(端口 6379)
- **ChromaDB** - 向量数据库(端口 8000,可选)
- **Ollama** - 本地模型(端口 11434,可选)

```bash
# 启动核心服务 / Start core services
docker-compose up -d

# 启动可选服务 / Start with optional services
docker-compose --profile with-ollama --profile with-vectordb up -d

# 查看日志 / View logs
docker-compose logs -f agentos

# 停止服务 / Stop services
docker-compose down
```

## Kubernetes 部署 / Kubernetes Deployment

### 基本部署 / Basic Deployment

```bash
# 应用清单 / Apply manifests
kubectl apply -f k8s/

# 检查状态 / Check status
kubectl get pods
kubectl get services

# 查看日志 / View logs
kubectl logs -f deployment/agentos
```

### Kubernetes 资源 / Kubernetes Resources

`k8s/` 目录包含:

- **deployment.yaml** - AgentOS 部署(带健康探针)
- **service.yaml** - LoadBalancer 服务
- **configmap.yaml** - 配置
- **secret.yaml** - API 密钥
- **hpa.yaml** - 水平 Pod 自动扩展器

### 部署示例 / Example Deployment

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

## 云平台部署 / Cloud Platform Deployment

### AWS ECS

```bash
# 构建并推送镜像 / Build and push image
docker build -t agno/agentos:latest .
docker tag agno/agentos:latest <aws_account_id>.dkr.ecr.us-east-1.amazonaws.com/agentos:latest
docker push <aws_account_id>.dkr.ecr.us-east-1.amazonaws.com/agentos:latest

# 创建 ECS 任务定义和服务 / Create ECS task definition and service
aws ecs create-service --cli-input-json file://ecs-service.json
```

### Google Cloud Run

```bash
# 构建并部署 / Build and deploy
gcloud builds submit --tag gcr.io/<project-id>/agentos
gcloud run deploy agentos \
  --image gcr.io/<project-id>/agentos \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated
```

### Azure Container Instances

```bash
# 部署容器 / Deploy container
az container create \
  --resource-group myResourceGroup \
  --name agentos \
  --image <registry>/agentos:latest \
  --dns-name-label agentos-demo \
  --ports 8080 \
  --environment-variables OPENAI_API_KEY=sk-your-key
```

## 环境配置 / Environment Configuration

### 必需变量 / Required Variables

```bash
# LLM API Keys
OPENAI_API_KEY=sk-your-openai-key

# Server Config
AGENTOS_ADDRESS=:8080
```

### 可选变量 / Optional Variables

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

## 数据库设置 / Database Setup

### PostgreSQL

```bash
# 使用 Docker / Using Docker
docker run -d \
  --name postgres \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=agentos \
  -p 5432:5432 \
  postgres:15

# 连接 / Connect
export DATABASE_URL=postgresql://postgres:password@localhost:5432/agentos
```

### Redis

```bash
# 使用 Docker / Using Docker
docker run -d \
  --name redis \
  -p 6379:6379 \
  redis:7-alpine

# 连接 / Connect
export REDIS_URL=redis://localhost:6379/0
```

## 监控和日志 / Monitoring & Logging

### 健康检查 / Health Checks

```bash
# 健康端点 / Health endpoint
curl http://localhost:8080/health

# 响应 / Response
{
  "status": "healthy",
  "service": "agentos",
  "time": 1704067200
}
```

### 结构化日志 / Structured Logging

AgentOS 使用 Go 的 `log/slog` 进行结构化日志记录:

```json
{
  "time": "2025-10-02T10:00:00Z",
  "level": "INFO",
  "msg": "Server started",
  "address": ":8080"
}
```

### Prometheus 指标(未来功能) / Prometheus Metrics (Future)

```go
// Planned metrics
agno_agent_creations_total
agno_agent_run_duration_seconds
agno_http_requests_total
agno_http_request_duration_seconds
```

## 安全最佳实践 / Security Best Practices

### 1. 使用密钥管理 / Use Secrets Management

```bash
# Kubernetes Secrets
kubectl create secret generic agentos-secrets \
  --from-literal=openai-api-key=sk-your-key

# AWS Secrets Manager
aws secretsmanager create-secret \
  --name agentos/openai-api-key \
  --secret-string sk-your-key
```

### 2. 启用 HTTPS / Enable HTTPS

```bash
# 使用反向代理(nginx, Caddy)
# 或负载均衡器(ALB, Cloud Load Balancer)
```

### 3. 速率限制 / Rate Limiting

使用反向代理或 API 网关进行速率限制。

### 4. 输入验证 / Input Validation

使用内置防护:

```go
import "github.com/rexleimo/agno-Go/pkg/hno/guardrails"

promptGuard := guardrails.NewPromptInjectionGuardrail()
agent.PreHooks = []hooks.Hook{promptGuard}
```

### 5. 非 root 容器 / Non-Root Container

Dockerfile 已使用非 root 用户:

```dockerfile
USER nonroot:nonroot
```

## 性能调优 / Performance Tuning

### 1. 资源限制 / Resource Limits

```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### 2. 水平扩展 / Horizontal Scaling

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

### 3. 连接池 / Connection Pooling

对于数据库连接,使用连接池:

```go
db, err := sql.Open("postgres", databaseURL)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

## 故障排查 / Troubleshooting

### 常见问题 / Common Issues

**1. "Connection refused"**

检查服务器是否运行:
```bash
docker ps
kubectl get pods
```

**2. "API key not set"**

验证环境变量:
```bash
docker exec agentos env | grep API_KEY
kubectl exec <pod> -- env | grep API_KEY
```

**3. "Out of memory"**

增加内存限制:
```yaml
resources:
  limits:
    memory: "1Gi"
```

**4. "Too many open files"**

增加文件描述符限制:
```bash
ulimit -n 65536
```

### 调试日志 / Debug Logging

启用调试模式:

```bash
export AGENTOS_DEBUG=true
export LOG_LEVEL=debug
```

## 生产环境检查清单 / Production Checklist

- [ ] 设置安全的 API 密钥
- [ ] 启用 HTTPS
- [ ] 配置健康检查
- [ ] 设置资源限制
- [ ] 启用监控
- [ ] 设置日志记录
- [ ] 配置备份
- [ ] 测试灾难恢复
- [ ] 文档化运维手册
- [ ] 设置告警

## 参考资料 / References

- [架构 / Architecture](/advanced/architecture)
- [性能 / Performance](/advanced/performance)
- [API 参考 / API Reference](/api/)
- [示例 / Examples](/examples/)
- [GitHub 仓库 / GitHub Repository](https://github.com/rexleimo/agno-Go)
