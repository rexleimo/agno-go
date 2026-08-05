# デプロイメントガイド

プロダクション環境でAgentOSをデプロイするための完全ガイド。

## クイックスタート

### Docker Composeを使用（推奨）

```bash
# 1. リポジトリをクローン
git clone https://github.com/rexleimo/agno-go.git
cd HNO

# 2. 環境を設定
cp .env.example .env
nano .env  # API キーを追加

# 3. サービスを開始
docker-compose up -d

# 4. 確認
curl http://localhost:8080/health
```

## Dockerデプロイメント

### 単一コンテナ

```bash
# イメージをビルド
docker build -t agentos:latest .

# コンテナを実行
docker run -d \
  -p 8080:8080 \
  -e OPENAI_API_KEY=sk-your-key \
  --name agentos \
  agentos:latest
```

### Docker Composeフルスタック

`docker-compose.yml`には以下が含まれます:

- **AgentOS** - HTTPサーバー (port 8080)
- **PostgreSQL** - データベース (port 5432)
- **Redis** - キャッシュ (port 6379)
- **ChromaDB** - ベクトルDB (port 8000, オプション)
- **Ollama** - ローカルモデル (port 11434, オプション)

```bash
# コアサービスを開始
docker-compose up -d

# オプションサービス付きで開始
docker-compose --profile with-ollama --profile with-vectordb up -d

# ログを表示
docker-compose logs -f agentos

# サービスを停止
docker-compose down
```

## Kubernetesデプロイメント

### 基本デプロイメント

```bash
# マニフェストを適用
kubectl apply -f k8s/

# ステータスを確認
kubectl get pods
kubectl get services

# ログを表示
kubectl logs -f deployment/agentos
```

### Kubernetesリソース

`k8s/`ディレクトリには以下が含まれます:

- **deployment.yaml** - ヘルスプローブ付きAgentOSデプロイメント
- **service.yaml** - LoadBalancerサービス
- **configmap.yaml** - 設定
- **secret.yaml** - APIキー
- **hpa.yaml** - Horizontal Pod Autoscaler

### デプロイメント例

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

## クラウドプラットフォームデプロイメント

### AWS ECS

```bash
# イメージをビルドしてプッシュ
docker build -t agno/agentos:latest .
docker tag agno/agentos:latest <aws_account_id>.dkr.ecr.us-east-1.amazonaws.com/agentos:latest
docker push <aws_account_id>.dkr.ecr.us-east-1.amazonaws.com/agentos:latest

# ECSタスク定義とサービスを作成
aws ecs create-service --cli-input-json file://ecs-service.json
```

### Google Cloud Run

```bash
# ビルドしてデプロイ
gcloud builds submit --tag gcr.io/<project-id>/agentos
gcloud run deploy agentos \
  --image gcr.io/<project-id>/agentos \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated
```

### Azure Container Instances

```bash
# コンテナをデプロイ
az container create \
  --resource-group myResourceGroup \
  --name agentos \
  --image <registry>/agentos:latest \
  --dns-name-label agentos-demo \
  --ports 8080 \
  --environment-variables OPENAI_API_KEY=sk-your-key
```

## 環境設定

### 必須変数

```bash
# LLM API キー
OPENAI_API_KEY=sk-your-openai-key

# サーバー設定
AGENTOS_ADDRESS=:8080
```

### オプション変数

```bash
# 追加のLLMプロバイダー
ANTHROPIC_API_KEY=sk-ant-your-key
OLLAMA_BASE_URL=http://localhost:11434

# ログ
LOG_LEVEL=info  # debug, info, warn, error
AGENTOS_DEBUG=false

# タイムアウト
REQUEST_TIMEOUT=30

# データベース
DATABASE_URL=postgresql://user:pass@localhost:5432/agentos

# Redisキャッシュ
REDIS_URL=redis://localhost:6379/0

# ChromaDB
CHROMA_URL=http://localhost:8000
```

## データベースセットアップ

### PostgreSQL

```bash
# Dockerを使用
docker run -d \
  --name postgres \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=agentos \
  -p 5432:5432 \
  postgres:15

# 接続
export DATABASE_URL=postgresql://postgres:password@localhost:5432/agentos
```

### Redis

```bash
# Dockerを使用
docker run -d \
  --name redis \
  -p 6379:6379 \
  redis:7-alpine

# 接続
export REDIS_URL=redis://localhost:6379/0
```

## モニタリング & ログ

### ヘルスチェック

```bash
# ヘルスエンドポイント
curl http://localhost:8080/health

# レスポンス
{
  "status": "healthy",
  "service": "agentos",
  "time": 1704067200
}
```

### 構造化ログ

AgentOSは構造化ログにGoの`log/slog`を使用:

```json
{
  "time": "2025-10-02T10:00:00Z",
  "level": "INFO",
  "msg": "Server started",
  "address": ":8080"
}
```

### Prometheusメトリクス（将来）

```go
// 計画中のメトリクス
agno_agent_creations_total
agno_agent_run_duration_seconds
agno_http_requests_total
agno_http_request_duration_seconds
```

## セキュリティベストプラクティス

### 1. シークレット管理を使用

```bash
# Kubernetes Secrets
kubectl create secret generic agentos-secrets \
  --from-literal=openai-api-key=sk-your-key

# AWS Secrets Manager
aws secretsmanager create-secret \
  --name agentos/openai-api-key \
  --secret-string sk-your-key
```

### 2. HTTPSを有効化

```bash
# リバースプロキシ（nginx、Caddy）を使用
# またはロードバランサー（ALB、Cloud Load Balancer）
```

### 3. レート制限

リバースプロキシまたはAPIゲートウェイをレート制限に使用。

### 4. 入力検証

組み込みガードレールを使用:

```go
import "github.com/rexleimo/agno-go/pkg/hno/guardrails"

promptGuard := guardrails.NewPromptInjectionGuardrail()
agent.PreHooks = []hooks.Hook{promptGuard}
```

### 5. 非rootコンテナ

Dockerfileはすでに非rootユーザーを使用:

```dockerfile
USER nonroot:nonroot
```

## パフォーマンスチューニング

### 1. リソース制限

```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### 2. 水平スケーリング

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

### 3. コネクションプーリング

データベース接続にはコネクションプーリングを使用:

```go
db, err := sql.Open("postgres", databaseURL)
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

## トラブルシューティング

### よくある問題

**1. "Connection refused"**

サーバーが実行中か確認:
```bash
docker ps
kubectl get pods
```

**2. "API key not set"**

環境変数を確認:
```bash
docker exec agentos env | grep API_KEY
kubectl exec <pod> -- env | grep API_KEY
```

**3. "Out of memory"**

メモリ制限を増やす:
```yaml
resources:
  limits:
    memory: "1Gi"
```

**4. "Too many open files"**

ファイルディスクリプタ制限を増やす:
```bash
ulimit -n 65536
```

### デバッグログ

デバッグモードを有効化:

```bash
export AGENTOS_DEBUG=true
export LOG_LEVEL=debug
```

## プロダクションチェックリスト

- [ ] 安全なAPIキーを設定
- [ ] HTTPSを有効化
- [ ] ヘルスチェックを設定
- [ ] リソース制限を設定
- [ ] モニタリングを有効化
- [ ] ログを設定
- [ ] バックアップを設定
- [ ] 災害復旧をテスト
- [ ] ランブックを文書化
- [ ] アラートを設定

## 参照

- [アーキテクチャ](/advanced/architecture)
- [パフォーマンス](/advanced/performance)
- [APIリファレンス](/api/)
- [サンプル](/examples/)
- [GitHubリポジトリ](https://github.com/rexleimo/agno-go)
