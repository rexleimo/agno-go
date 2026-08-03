# Installation

This guide covers different ways to install and set up HNO.

## Prerequisites

- **Go 1.21 or later** - [Download Go](https://golang.org/dl/)
- **API Key** - OpenAI, Anthropic, or Ollama (for local models)
- **Git** - For cloning the repository

## Method 1: Go Get (Recommended)

Install HNO as a Go module dependency:

```bash
go get github.com/rexleimo/HNO
```

Then import in your code:

```go
import (
    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
)
```

## Method 2: Clone Repository

Clone the repository to explore examples and contribute:

```bash
# Clone repository
git clone https://github.com/rexleimo/HNO.git
cd HNO

# Download dependencies
go mod download

# Verify installation
go test ./...
```

## Method 3: Docker

Use Docker to run AgentOS server without installing Go:

### Using Docker

```bash
# Build image
docker build -t agentos:latest .

# Run server
docker run -p 8080:8080 \
  -e OPENAI_API_KEY=sk-your-key \
  agentos:latest
```

### Using Docker Compose (Full Stack)

```bash
# Copy environment template
cp .env.example .env

# Edit .env and add your API keys
nano .env

# Start all services
docker-compose up -d
```

This starts:
- **AgentOS** server (port 8080)
- **PostgreSQL** database
- **Redis** cache
- **ChromaDB** (optional, for RAG)
- **Ollama** (optional, for local models)

## API Keys Setup

### OpenAI

1. Get API key from [OpenAI Platform](https://platform.openai.com/api-keys)
2. Set environment variable:

```bash
export OPENAI_API_KEY=sk-your-key-here
```

### Anthropic Claude

1. Get API key from [Anthropic Console](https://console.anthropic.com/)
2. Set environment variable:

```bash
export ANTHROPIC_API_KEY=sk-ant-your-key
```

### Ollama (Local Models)

1. Install Ollama: [ollama.com](https://ollama.com)
2. Pull a model:

```bash
ollama pull llama3
```

3. (Optional) Set base URL:

```bash
export OLLAMA_BASE_URL=http://localhost:11434
```

## Verify Installation

### Test Go Package

Create a test file `test.go`:

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

Run it:

```bash
export OPENAI_API_KEY=sk-your-key
go run test.go
```

### Test Docker Installation

```bash
# Check health
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","service":"agentos","time":1704067200}
```

## Development Setup

For contributing or local development:

### 1. Install Development Tools

```bash
# Install golangci-lint (linter)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install goimports (formatter)
go install golang.org/x/tools/cmd/goimports@latest
```

Or use Make:

```bash
make install-tools
```

### 2. Run Tests

```bash
# Run all tests
make test

# Run specific package
go test -v ./pkg/hno/agent/...

# Generate coverage report
make coverage
```

### 3. Format and Lint

```bash
# Format code
make fmt

# Run linter
make lint

# Run go vet
make vet
```

### 4. Build Examples

```bash
# Build all examples
make build

# Run specific example
./bin/simple_agent
```

## Environment Variables

Create a `.env` file for configuration:

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

## IDE Setup

### VS Code

Install recommended extensions:

```json
{
  "recommendations": [
    "golang.go",
    "ms-azuretools.vscode-docker"
  ]
}
```

### GoLand

GoLand has built-in Go support. Just open the project directory.

## Troubleshooting

### Common Issues

**1. "Go version too old"**

Update Go to 1.21+:
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

Add user to docker group:
```bash
sudo usermod -aG docker $USER
newgrp docker
```

**4. "Port already in use"**

Change port in `.env`:
```bash
AGENTOS_ADDRESS=:9090
```

### Getting Help

If you encounter issues:

1. Check [GitHub Issues](https://github.com/rexleimo/HNO/issues)
2. Ask in [Discussions](https://github.com/rexleimo/HNO/discussions)
3. Review [documentation](/guide/)

## Next Steps

Now that HNO is installed:

1. [Quick Start](/guide/quick-start) - Build your first agent
2. [Core Concepts](/guide/agent) - Learn about Agent, Team, Workflow
3. [Examples](/examples/) - Explore working examples
4. [API Reference](/api/) - Detailed API documentation

## Platform-Specific Notes

### macOS

No special requirements. Install via Homebrew:

```bash
brew install go
```

### Linux

Install Go from package manager:

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

Download installer from [golang.org](https://golang.org/dl/) or use Chocolatey:

```powershell
choco install golang
```

**Note**: Use PowerShell or WSL2 for best experience.

## Production Deployment

For production deployments, see:

- [Deployment Guide](/advanced/deployment) - Docker, Kubernetes, cloud platforms
- [Performance Guide](/advanced/performance) - Optimization tips
- [Security Best Practices](/advanced/deployment#security) - Production security
