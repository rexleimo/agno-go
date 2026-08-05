# 배포 가이드

프로덕션 환경에서 AgentOS를 배포하기 위한 완전한 가이드입니다.

## 빠른 시작

### Docker Compose 사용 (권장)

```bash
# 1. 저장소 클론
git clone https://github.com/rexleimo/agno-go.git
cd HNO

# 2. 환경 설정
cp .env.example .env
nano .env  # API 키 추가

# 3. 서비스 시작
docker-compose up -d

# 4. 확인
curl http://localhost:8080/health
```

## Docker 배포

### 단일 컨테이너

```bash
# 이미지 빌드
docker build -t agentos:latest .

# 컨테이너 실행
docker run -d \
  -p 8080:8080 \
  -e OPENAI_API_KEY=sk-your-key \
  --name agentos \
  agentos:latest
```

### Docker Compose 전체 스택

`docker-compose.yml` 포함 항목:

- **AgentOS** - HTTP 서버 (포트 8080)
- **PostgreSQL** - 데이터베이스 (포트 5432)
- **Redis** - 캐시 (포트 6379)
- **ChromaDB** - 벡터 DB (포트 8000, 선택사항)
- **Ollama** - 로컬 모델 (포트 11434, 선택사항)

```bash
# 핵심 서비스 시작
docker-compose up -d

# 선택 서비스와 함께 시작
docker-compose --profile with-ollama --profile with-vectordb up -d

# 로그 보기
docker-compose logs -f agentos

# 서비스 중지
docker-compose down
```

## Kubernetes 배포

### 기본 배포

```bash
# 매니페스트 적용
kubectl apply -f k8s/

# 상태 확인
kubectl get pods
kubectl get services

# 로그 보기
kubectl logs -f deployment/agentos
```

### Kubernetes 리소스

`k8s/` 디렉토리 포함 항목:

- **deployment.yaml** - 헬스 프로브가 있는 AgentOS 배포
- **service.yaml** - LoadBalancer 서비스
- **configmap.yaml** - 설정
- **secret.yaml** - API 키
- **hpa.yaml** - Horizontal Pod Autoscaler

### 배포 예제

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

## 클라우드 플랫폼 배포

### AWS ECS

```bash
# 이미지 빌드 및 푸시
docker build -t agno/agentos:latest .
docker tag agno/agentos:latest <aws_account_id>.dkr.ecr.us-east-1.amazonaws.com/agentos:latest
docker push <aws_account_id>.dkr.ecr.us-east-1.amazonaws.com/agentos:latest

# ECS 작업 정의 및 서비스 생성
aws ecs create-service --cli-input-json file://ecs-service.json
```

### Google Cloud Run

```bash
# 빌드 및 배포
gcloud builds submit --tag gcr.io/<project-id>/agentos
gcloud run deploy agentos \
  --image gcr.io/<project-id>/agentos \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated
```

### Azure Container Instances

```bash
# 컨테이너 배포
az container create \
  --resource-group myResourceGroup \
  --name agentos \
  --image <registry>/agentos:latest \
  --dns-name-label agentos-demo \
  --ports 8080 \
  --environment-variables OPENAI_API_KEY=sk-your-key
```

## 환경 설정

### 필수 변수

```bash
# LLM API Keys
OPENAI_API_KEY=sk-your-openai-key

# Server Config
AGENTOS_ADDRESS=:8080
```

### 선택 변수

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

## 데이터베이스 설정

### PostgreSQL

```bash
# Docker 사용
docker run -d \
  --name postgres \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=agentos \
  -p 5432:5432 \
  postgres:15

# 연결
export DATABASE_URL=postgresql://postgres:password@localhost:5432/agentos
```

### Redis

```bash
# Docker 사용
docker run -d \
  --name redis \
  -p 6379:6379 \
  redis:7-alpine

# 연결
export REDIS_URL=redis://localhost:6379/0
```

## 모니터링 및 로깅

### 헬스 체크

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

### 구조화된 로깅

AgentOS는 구조화된 로깅을 위해 Go의 `log/slog` 사용:

```json
{
  "time": "2025-10-02T10:00:00Z",
  "level": "INFO",
  "msg": "Server started",
  "address": ":8080"
}
```

### Prometheus 메트릭 (향후)

```go
// Planned metrics
agno_agent_creations_total
agno_agent_run_duration_seconds
agno_http_requests_total
agno_http_request_duration_seconds
```

## 보안 모범 사례

### 1. 시크릿 관리 사용

```bash
# Kubernetes Secrets
kubectl create secret generic agentos-secrets \
  --from-literal=openai-api-key=sk-your-key

# AWS Secrets Manager
aws secretsmanager create-secret \
  --name agentos/openai-api-key \
  --secret-string sk-your-key
```

### 2. HTTPS 활성화

```bash
# 역방향 프록시(nginx, Caddy) 사용
# 또는 로드 밸런서(ALB, Cloud Load Balancer) 사용
```

### 3. 속도 제한

역방향 프록시 또는 API 게이트웨이를 사용한 속도 제한.

### 4. 입력 검증

내장 가드레일 사용:

```go
import "github.com/rexleimo/agno-go/pkg/hno/guardrails"

promptGuard := guardrails.NewPromptInjectionGuardrail()
agent.PreHooks = []hooks.Hook{promptGuard}
```

### 5. Non-Root 컨테이너

Dockerfile은 이미 non-root 사용자 사용:

```dockerfile
USER nonroot:nonroot
```

## 성능 튜닝

### 1. 리소스 제한

```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### 2. 수평 확장

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

### 3. 연결 풀링

데이터베이스 연결에 연결 풀링 사용:

```go
db, err := sql.Open("postgres", databaseURL)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

## 문제 해결

### 일반적인 문제

**1. "Connection refused"**

서버 실행 여부 확인:
```bash
docker ps
kubectl get pods
```

**2. "API key not set"**

환경 변수 확인:
```bash
docker exec agentos env | grep API_KEY
kubectl exec <pod> -- env | grep API_KEY
```

**3. "Out of memory"**

메모리 제한 증가:
```yaml
resources:
  limits:
    memory: "1Gi"
```

**4. "Too many open files"**

파일 디스크립터 제한 증가:
```bash
ulimit -n 65536
```

### 디버그 로깅

디버그 모드 활성화:

```bash
export AGENTOS_DEBUG=true
export LOG_LEVEL=debug
```

## 프로덕션 체크리스트

- [ ] 안전한 API 키 설정
- [ ] HTTPS 활성화
- [ ] 헬스 체크 구성
- [ ] 리소스 제한 설정
- [ ] 모니터링 활성화
- [ ] 로깅 설정
- [ ] 백업 구성
- [ ] 재해 복구 테스트
- [ ] 런북 문서화
- [ ] 알림 설정

## 참고 자료

- [아키텍처](/advanced/architecture)
- [성능](/advanced/performance)
- [API 참조](/api/)
- [예제](/examples/)
- [GitHub 저장소](https://github.com/rexleimo/agno-go)
