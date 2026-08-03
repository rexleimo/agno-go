# 설치

이 가이드는 HNO를 설치하고 설정하는 다양한 방법을 다룹니다.

## 전제 조건

- **Go 1.21 이상** - [Go 다운로드](https://golang.org/dl/)
- **API Key** - OpenAI, Anthropic, 또는 Ollama (로컬 모델용)
- **Git** - 리포지토리 복제용

## 방법 1: Go Get (권장)

HNO를 Go 모듈 의존성으로 설치:

```bash
go get github.com/rexleimo/HNO
```

그런 다음 코드에서 임포트:

```go
import (
    "github.com/rexleimo/HNO/pkg/hno/agent"
    "github.com/rexleimo/HNO/pkg/hno/models/openai"
)
```

## 방법 2: 리포지토리 복제

예제를 탐색하고 기여하려면 리포지토리를 복제하세요:

```bash
# 리포지토리 복제
git clone https://github.com/rexleimo/HNO.git
cd HNO

# 의존성 다운로드
go mod download

# 설치 확인
go test ./...
```

## 방법 3: Docker

Go 설치 없이 Docker를 사용하여 AgentOS 서버 실행:

### Docker 사용

```bash
# 이미지 빌드
docker build -t agentos:latest .

# 서버 실행
docker run -p 8080:8080 \
  -e OPENAI_API_KEY=sk-your-key \
  agentos:latest
```

### Docker Compose 사용 (전체 스택)

```bash
# 환경 템플릿 복사
cp .env.example .env

# .env를 편집하고 API 키 추가
nano .env

# 모든 서비스 시작
docker-compose up -d
```

다음이 시작됩니다:
- **AgentOS** 서버 (포트 8080)
- **PostgreSQL** 데이터베이스
- **Redis** 캐시
- **ChromaDB** (선택사항, RAG용)
- **Ollama** (선택사항, 로컬 모델용)

## API 키 설정

### OpenAI

1. [OpenAI Platform](https://platform.openai.com/api-keys)에서 API 키 받기
2. 환경 변수 설정:

```bash
export OPENAI_API_KEY=sk-your-key-here
```

### Anthropic Claude

1. [Anthropic Console](https://console.anthropic.com/)에서 API 키 받기
2. 환경 변수 설정:

```bash
export ANTHROPIC_API_KEY=sk-ant-your-key
```

### Ollama (로컬 모델)

1. Ollama 설치: [ollama.com](https://ollama.com)
2. 모델 다운로드:

```bash
ollama pull llama3
```

3. (선택사항) 베이스 URL 설정:

```bash
export OLLAMA_BASE_URL=http://localhost:11434
```

## 설치 확인

### Go 패키지 테스트

테스트 파일 `test.go` 생성:

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

실행:

```bash
export OPENAI_API_KEY=sk-your-key
go run test.go
```

### Docker 설치 테스트

```bash
# 헬스 체크
curl http://localhost:8080/health

# 예상 응답:
# {"status":"healthy","service":"agentos","time":1704067200}
```

## 개발 환경 설정

기여 또는 로컬 개발용:

### 1. 개발 도구 설치

```bash
# golangci-lint (린터) 설치
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# goimports (포매터) 설치
go install golang.org/x/tools/cmd/goimports@latest
```

또는 Make 사용:

```bash
make install-tools
```

### 2. 테스트 실행

```bash
# 모든 테스트 실행
make test

# 특정 패키지 실행
go test -v ./pkg/hno/agent/...

# 커버리지 보고서 생성
make coverage
```

### 3. 포맷 및 린트

```bash
# 코드 포맷
make fmt

# 린터 실행
make lint

# go vet 실행
make vet
```

### 4. 예제 빌드

```bash
# 모든 예제 빌드
make build

# 특정 예제 실행
./bin/simple_agent
```

## 환경 변수

구성을 위한 `.env` 파일 생성:

```bash
# LLM API 키
OPENAI_API_KEY=sk-your-openai-key
ANTHROPIC_API_KEY=sk-ant-your-anthropic-key
OLLAMA_BASE_URL=http://localhost:11434

# AgentOS 서버
AGENTOS_ADDRESS=:8080
AGENTOS_DEBUG=true

# 로깅
LOG_LEVEL=info

# 타임아웃
REQUEST_TIMEOUT=30

# 데이터베이스 (PostgreSQL 사용 시)
DATABASE_URL=postgresql://user:password@localhost:5432/agentos

# Redis (캐시 사용 시)
REDIS_URL=redis://localhost:6379/0

# ChromaDB (RAG 사용 시)
CHROMA_URL=http://localhost:8000
```

## IDE 설정

### VS Code

권장 확장 프로그램 설치:

```json
{
  "recommendations": [
    "golang.go",
    "ms-azuretools.vscode-docker"
  ]
}
```

### GoLand

GoLand에는 Go 지원이 내장되어 있습니다. 프로젝트 디렉토리를 열기만 하면 됩니다.

## 문제 해결

### 일반적인 문제

**1. "Go version too old"**

Go를 1.21 이상으로 업데이트:
```bash
# 버전 확인
go version

# 최신 버전 다운로드: https://golang.org/dl/
```

**2. "Module not found"**

```bash
go mod download
go mod tidy
```

**3. "Permission denied" (Docker)**

docker 그룹에 사용자 추가:
```bash
sudo usermod -aG docker $USER
newgrp docker
```

**4. "Port already in use"**

`.env`에서 포트 변경:
```bash
AGENTOS_ADDRESS=:9090
```

### 도움 받기

문제가 발생하면:

1. [GitHub Issues](https://github.com/rexleimo/HNO/issues) 확인
2. [Discussions](https://github.com/rexleimo/HNO/discussions)에서 질문
3. [문서](/guide/) 검토

## 다음 단계

이제 HNO가 설치되었습니다:

1. [Quick Start](/guide/quick-start) - 첫 번째 에이전트 구축
2. [Core Concepts](/guide/agent) - Agent, Team, Workflow에 대해 배우기
3. [Examples](/examples/) - 실제 예제 탐색
4. [API Reference](/api/) - 자세한 API 문서

## 플랫폼별 참고 사항

### macOS

특별한 요구 사항 없음. Homebrew로 설치:

```bash
brew install go
```

### Linux

패키지 매니저에서 Go 설치:

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

[golang.org](https://golang.org/dl/)에서 설치 프로그램 다운로드 또는 Chocolatey 사용:

```powershell
choco install golang
```

**참고**: 최상의 경험을 위해 PowerShell 또는 WSL2를 사용하세요.

## 프로덕션 배포

프로덕션 배포는 다음을 참조하세요:

- [Deployment Guide](/advanced/deployment) - Docker, Kubernetes, 클라우드 플랫폼
- [Performance Guide](/advanced/performance) - 최적화 팁
- [Security Best Practices](/advanced/deployment#security) - 프로덕션 보안
