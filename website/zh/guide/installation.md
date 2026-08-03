# 安装

本指南涵盖了安装和设置 Agno-Go 的不同方法。

## 前置要求

- **Go 1.21 或更高版本** - [下载 Go](https://golang.org/dl/)
- **API 密钥** - OpenAI、Anthropic 或 Ollama (用于本地模型)
- **Git** - 用于克隆仓库

## 方式 1: Go Get (推荐)

将 Agno-Go 安装为 Go 模块依赖:

```bash
go get github.com/rexleimo/agno-Go
```

然后在代码中导入:

```go
import (
    "github.com/rexleimo/agno-Go/pkg/hno/agent"
    "github.com/rexleimo/agno-Go/pkg/hno/models/openai"
)
```

## 方式 2: 克隆仓库

克隆仓库以探索示例并贡献代码:

```bash
# Clone repository
git clone https://github.com/rexleimo/agno-Go.git
cd agno-Go

# Download dependencies
go mod download

# Verify installation
go test ./...
```

## 方式 3: Docker

使用 Docker 运行 AgentOS 服务器,无需安装 Go:

### 使用 Docker

```bash
# Build image
docker build -t agentos:latest .

# Run server
docker run -p 8080:8080 \
  -e OPENAI_API_KEY=sk-your-key \
  agentos:latest
```

### 使用 Docker Compose (完整堆栈)

```bash
# Copy environment template
cp .env.example .env

# Edit .env and add your API keys
nano .env

# Start all services
docker-compose up -d
```

这将启动:
- **AgentOS** 服务器 (端口 8080)
- **PostgreSQL** 数据库
- **Redis** 缓存
- **ChromaDB** (可选,用于 RAG)
- **Ollama** (可选,用于本地模型)

## API 密钥设置

### OpenAI

1. 从 [OpenAI Platform](https://platform.openai.com/api-keys) 获取 API 密钥
2. 设置环境变量:

```bash
export OPENAI_API_KEY=sk-your-key-here
```

### Anthropic Claude

1. 从 [Anthropic Console](https://console.anthropic.com/) 获取 API 密钥
2. 设置环境变量:

```bash
export ANTHROPIC_API_KEY=sk-ant-your-key
```

### Ollama (本地模型)

1. 安装 Ollama: [ollama.com](https://ollama.com)
2. 拉取模型:

```bash
ollama pull llama3
```

3. (可选) 设置基础 URL:

```bash
export OLLAMA_BASE_URL=http://localhost:11434
```

## 验证安装

### 测试 Go 包

创建测试文件 `test.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/rexleimo/agno-Go/pkg/hno/agent"
    "github.com/rexleimo/agno-Go/pkg/hno/models/openai"
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

运行:

```bash
export OPENAI_API_KEY=sk-your-key
go run test.go
```

### 测试 Docker 安装

```bash
# Check health
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","service":"agentos","time":1704067200}
```

## 开发环境设置

用于贡献或本地开发:

### 1. 安装开发工具

```bash
# Install golangci-lint (linter)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install goimports (formatter)
go install golang.org/x/tools/cmd/goimports@latest
```

或使用 Make:

```bash
make install-tools
```

### 2. 运行测试

```bash
# Run all tests
make test

# Run specific package
go test -v ./pkg/hno/agent/...

# Generate coverage report
make coverage
```

### 3. 格式化和检查

```bash
# Format code
make fmt

# Run linter
make lint

# Run go vet
make vet
```

### 4. 构建示例

```bash
# Build all examples
make build

# Run specific example
./bin/simple_agent
```

## 环境变量

创建 `.env` 文件进行配置:

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

## IDE 设置

### VS Code

安装推荐的扩展:

```json
{
  "recommendations": [
    "golang.go",
    "ms-azuretools.vscode-docker"
  ]
}
```

### GoLand

GoLand 内置 Go 支持。只需打开项目目录即可。

## 故障排除

### 常见问题

**1. "Go version too old"**

更新 Go 到 1.21+:
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

**3. "Permission denied" (Docker)**

将用户添加到 docker 组:
```bash
sudo usermod -aG docker $USER
newgrp docker
```

**4. "Port already in use"**

在 `.env` 中更改端口:
```bash
AGENTOS_ADDRESS=:9090
```

### 获取帮助

如果遇到问题:

1. 查看 [GitHub Issues](https://github.com/rexleimo/agno-Go/issues)
2. 在 [Discussions](https://github.com/rexleimo/agno-Go/discussions) 中提问
3. 查阅 [文档](/guide/)

## 下一步

现在 Agno-Go 已安装:

1. [Quick Start](/guide/quick-start) - 构建您的第一个 Agent
2. [Core Concepts](/guide/agent) - 了解 Agent、Team、Workflow
3. [Examples](/examples/) - 探索工作示例
4. [API Reference](/api/) - 详细的 API 文档

## 平台特定说明

### macOS

无特殊要求。通过 Homebrew 安装:

```bash
brew install go
```

### Linux

从包管理器安装 Go:

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

从 [golang.org](https://golang.org/dl/) 下载安装程序或使用 Chocolatey:

```powershell
choco install golang
```

**注意**: 使用 PowerShell 或 WSL2 以获得最佳体验。

## 生产部署

对于生产部署,请参阅:

- [Deployment Guide](/advanced/deployment) - Docker、Kubernetes、云平台
- [Performance Guide](/advanced/performance) - 优化技巧
- [Security Best Practices](/advanced/deployment#security) - 生产安全
