# インストール

このガイドでは、HNOをインストールおよびセットアップするさまざまな方法について説明します。

## 前提条件

- **Go 1.21以降** - [Goをダウンロード](https://golang.org/dl/)
- **APIキー** - OpenAI、Anthropic、またはOllama（ローカルモデル用）
- **Git** - リポジトリのクローン用

## 方法1: Go Get（推奨）

HNOをGoモジュールの依存関係としてインストール：

```bash
go get github.com/rexleimo/HNO
```

次にコードでインポート：

```go
import (
    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
)
```

## 方法2: リポジトリのクローン

例を探索し、貢献するためにリポジトリをクローン：

```bash
# Clone repository
git clone https://github.com/rexleimo/HNO.git
cd HNO

# Download dependencies
go mod download

# Verify installation
go test ./...
```

## 方法3: Docker

Goをインストールせずに、DockerでAgentOSサーバーを実行：

### Dockerを使用

```bash
# Build image
docker build -t agentos:latest .

# Run server
docker run -p 8080:8080 \
  -e OPENAI_API_KEY=sk-your-key \
  agentos:latest
```

### Docker Composeを使用（フルスタック）

```bash
# Copy environment template
cp .env.example .env

# Edit .env and add your API keys
nano .env

# Start all services
docker-compose up -d
```

これにより以下が起動します：
- **AgentOS**サーバー（ポート8080）
- **PostgreSQL**データベース
- **Redis**キャッシュ
- **ChromaDB**（オプション、RAG用）
- **Ollama**（オプション、ローカルモデル用）

## APIキーのセットアップ

### OpenAI

1. [OpenAI Platform](https://platform.openai.com/api-keys)からAPIキーを取得
2. 環境変数を設定：

```bash
export OPENAI_API_KEY=sk-your-key-here
```

### Anthropic Claude

1. [Anthropic Console](https://console.anthropic.com/)からAPIキーを取得
2. 環境変数を設定：

```bash
export ANTHROPIC_API_KEY=sk-ant-your-key
```

### Ollama（ローカルモデル）

1. Ollamaをインストール：[ollama.com](https://ollama.com)
2. モデルをプル：

```bash
ollama pull llama3
```

3. （オプション）ベースURLを設定：

```bash
export OLLAMA_BASE_URL=http://localhost:11434
```

## インストールの確認

### Goパッケージのテスト

テストファイル`test.go`を作成：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
)

func main() {
    model, err := openai.New("gpt-4o-mini", openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    ag, err := agent.New(agent.Config{
        Name:  "Test Agent",
        Model: model,
    })
    if err != nil {
        log.Fatal(err)
    }

    output, err := ag.Run(context.Background(), "Say hello!")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(output.Content)
}
```

実行：

```bash
export OPENAI_API_KEY=sk-your-key
go run test.go
```

### Dockerインストールのテスト

```bash
# Check health
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","service":"agentos","time":1704067200}
```

## 開発環境のセットアップ

貢献またはローカル開発用：

### 1. 開発ツールのインストール

```bash
# Install golangci-lint (linter)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install goimports (formatter)
go install golang.org/x/tools/cmd/goimports@latest
```

またはMakeを使用：

```bash
make install-tools
```

### 2. テストの実行

```bash
# Run all tests
make test

# Run specific package
go test -v ./pkg/hno/agent/...

# Generate coverage report
make coverage
```

### 3. フォーマットとLint

```bash
# Format code
make fmt

# Run linter
make lint

# Run go vet
make vet
```

### 4. 例のビルド

```bash
# Build all examples
make build

# Run specific example
./bin/simple_agent
```

## 環境変数

設定用の`.env`ファイルを作成：

```bash
# LLM API Keys
OPENAI_API_KEY=sk-your-openai-key
ANTHROPIC_API_KEY=sk-ant-your-anthropic-key
OLLAMA_BASE_URL=http://localhost:11434

# AgentOS Server
AGENTOS_ADDRESS=:8080
AGENTOS_DEBUG=true

# Logging
LOG_LEVEL=info

# Timeouts
REQUEST_TIMEOUT=30

# Database (if using PostgreSQL)
DATABASE_URL=postgresql://user:password@localhost:5432/agentos

# Redis (if using cache)
REDIS_URL=redis://localhost:6379/0

# ChromaDB (if using RAG)
CHROMA_URL=http://localhost:8000
```

## IDE設定

### VS Code

推奨拡張機能をインストール：

```json
{
  "recommendations": [
    "golang.go",
    "ms-azuretools.vscode-docker"
  ]
}
```

### GoLand

GoLandには組み込みのGoサポートがあります。プロジェクトディレクトリを開くだけです。

## トラブルシューティング

### よくある問題

**1. "Go version too old"**

Goを1.21以降に更新：
```bash
# Check version
go version

# Download latest: https://golang.org/dl/
```

**2. "Module not found"**

```bash
go mod download
go mod tidy
```

**3. "Permission denied"（Docker）**

ユーザーをdockerグループに追加：
```bash
sudo usermod -aG docker $USER
newgrp docker
```

**4. "Port already in use"**

`.env`でポートを変更：
```bash
AGENTOS_ADDRESS=:9090
```

### ヘルプの取得

問題が発生した場合：

1. [GitHub Issues](https://github.com/rexleimo/HNO/issues)を確認
2. [Discussions](https://github.com/rexleimo/HNO/discussions)で質問
3. [ドキュメント](/guide/)を確認

## 次のステップ

HNOがインストールされたので：

1. [クイックスタート](/guide/quick-start) - 最初のエージェントを構築
2. [コアコンセプト](/guide/agent) - Agent、Team、Workflowについて学ぶ
3. [Examples](/examples/) - 動作する例を探索
4. [APIリファレンス](/api/) - 詳細なAPIドキュメント

## プラットフォーム固有の注意事項

### macOS

特別な要件はありません。Homebrewでインストール：

```bash
brew install go
```

### Linux

パッケージマネージャーからGoをインストール：

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install golang-go

# Fedora
sudo dnf install golang

# Arch
sudo pacman -S go
```

### Windows

[golang.org](https://golang.org/dl/)からインストーラーをダウンロードするか、Chocolateyを使用：

```powershell
choco install golang
```

**注意**: 最良の体験のためにPowerShellまたはWSL2を使用してください。

## 本番環境デプロイ

本番環境のデプロイについては以下を参照：

- [デプロイガイド](/advanced/deployment) - Docker、Kubernetes、クラウドプラットフォーム
- [パフォーマンスガイド](/advanced/performance) - 最適化のヒント
- [セキュリティのベストプラクティス](/advanced/deployment#security) - 本番環境のセキュリティ
